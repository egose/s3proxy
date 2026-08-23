package auth

import (
	"context"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	v4 "github.com/aws/aws-sdk-go-v2/aws/signer/v4"
	"github.com/egose/s3proxy/internal/config"
	"github.com/egose/s3proxy/internal/replaybody"
)

const (
	inboundService = "s3"
	inboundSkew    = 15 * time.Minute
	maxPresignTTL  = 7 * 24 * time.Hour
)

type sigV4Verifier struct {
	signer        *v4.Signer
	clientsByAK   map[string]config.Client
	defaultRegion string
	replayBudget  *replaybody.Budget
	now           func() time.Time
}

func newSigV4Verifier(clientsByAK map[string]config.Client, defaultRegion string, replayBudget *replaybody.Budget) *sigV4Verifier {
	return &sigV4Verifier{
		signer:        v4.NewSigner(),
		clientsByAK:   clientsByAK,
		defaultRegion: defaultRegion,
		replayBudget:  replayBudget,
		now:           func() time.Time { return time.Now().UTC() },
	}
}

func (v *sigV4Verifier) Verify(r *http.Request) (*Principal, error) {
	authHeader := r.Header.Get("Authorization")
	if authHeader != "" {
		return v.verifyHeader(r)
	}
	return v.verifyQuery(r)
}

func (v *sigV4Verifier) verifyHeader(r *http.Request) (*Principal, error) {
	accessKey, scope, signedHeaders, providedSignature, err := parseAuthHeader(r.Header.Get("Authorization"))
	if err != nil {
		return nil, err
	}

	client, ok := v.clientsByAK[accessKey]
	if !ok {
		return nil, errUnknownAccessKey
	}

	amzDate := r.Header.Get("X-Amz-Date")
	if !containsHeader(signedHeaders, "x-amz-date") {
		return nil, errUnsignedDate
	}
	if err := v.checkDateSkew(amzDate); err != nil {
		return nil, err
	}

	region, err := validateScope(scope, amzDate[:8])
	if err != nil {
		return nil, err
	}

	payloadHash, err := validatePayloadHashClaim(r, signedHeaders)
	if err != nil {
		return nil, err
	}

	clone := cloneForSigning(r)
	clone.Header.Del("Authorization")

	// Strip any headers from the clone that the original signature did not
	// include (e.g. Accept-Encoding injected by the Go http transport). The
	// original SignedHeaders list governs what the signature covers, so the
	// re-signed request must include exactly those headers.
	stripUnsignedHeaders(clone, signedHeaders)

	creds := aws.Credentials{
		AccessKeyID:     client.AccessKey,
		SecretAccessKey: client.SecretKey,
	}

	if err := v.signer.SignHTTP(context.Background(), creds, clone, payloadHash, inboundService, region, parseAmzDate(amzDate)); err != nil {
		return nil, err
	}

	_, _, _, generatedSignature, err := parseAuthHeader(clone.Header.Get("Authorization"))
	if err != nil {
		return nil, err
	}
	if subtle.ConstantTimeCompare([]byte(generatedSignature), []byte(providedSignature)) != 1 {
		return nil, errSignatureMismatch
	}
	if err := v.verifyPayloadBody(r, payloadHash); err != nil {
		return nil, err
	}
	filterAuthenticatedHeaders(r, signedHeaders)

	return principalFromClient(client), nil
}

func (v *sigV4Verifier) verifyQuery(r *http.Request) (*Principal, error) {
	query := r.URL.Query()
	if err := rejectDuplicatePresignParams(query); err != nil {
		return nil, err
	}
	algorithm, err := requiredPresignParam(query, "X-Amz-Algorithm", errUnsupportedAuthScheme)
	if err != nil {
		return nil, err
	}
	if algorithm != "AWS4-HMAC-SHA256" {
		return nil, errUnsupportedAuthScheme
	}
	cred, err := requiredPresignParam(query, "X-Amz-Credential", errMissingAuth)
	if err != nil {
		return nil, err
	}

	parts := strings.Split(cred, "/")
	if len(parts) != 5 {
		return nil, errInvalidCredential
	}
	accessKey := parts[0]
	scope := strings.Join(parts[1:], "/")

	client, ok := v.clientsByAK[accessKey]
	if !ok {
		return nil, errUnknownAccessKey
	}

	date, err := requiredPresignParam(query, "X-Amz-Date", errMissingDate)
	if err != nil {
		return nil, err
	}
	if _, err := requiredPresignParam(query, "X-Amz-Expires", errMissingExpires); err != nil {
		return nil, err
	}
	if err := v.checkPresignWindow(r, date); err != nil {
		return nil, err
	}

	region, err := validateScope(scope, date[:8])
	if err != nil {
		return nil, err
	}
	signedHeadersRaw, err := requiredPresignParam(query, "X-Amz-SignedHeaders", errMissingSignedHeaders)
	if err != nil {
		return nil, err
	}
	signedHeaders, err := parseSignedHeaders(signedHeadersRaw)
	if err != nil {
		return nil, err
	}
	payloadHash, err := validatePayloadHashClaim(r, signedHeaders)
	if err != nil {
		return nil, err
	}

	provided, err := requiredPresignParam(query, "X-Amz-Signature", errMissingSignature)
	if err != nil {
		return nil, err
	}
	clone := r.Clone(context.Background())
	clone.Body = http.NoBody
	cloneQuery := clone.URL.Query()
	cloneQuery.Del("X-Amz-Signature")
	clone.URL.RawQuery = cloneQuery.Encode()
	stripUnsignedHeaders(clone, signedHeaders)

	creds := aws.Credentials{
		AccessKeyID:     client.AccessKey,
		SecretAccessKey: client.SecretKey,
	}

	signedURI, _, err := v.signer.PresignHTTP(context.Background(), creds, clone, payloadHash, inboundService, region, parseAmzDate(date))
	if err != nil {
		return nil, err
	}

	generated, err := signatureFromPresignedURI(signedURI)
	if err != nil {
		return nil, err
	}
	if subtle.ConstantTimeCompare([]byte(generated), []byte(provided)) != 1 {
		return nil, errSignatureMismatch
	}
	if err := v.verifyPayloadBody(r, payloadHash); err != nil {
		return nil, err
	}
	filterAuthenticatedHeaders(r, signedHeaders)

	return principalFromClient(client), nil
}

func rejectDuplicatePresignParams(query url.Values) error {
	for _, name := range []string{
		"X-Amz-Algorithm",
		"X-Amz-Credential",
		"X-Amz-Date",
		"X-Amz-Expires",
		"X-Amz-Signature",
		"X-Amz-SignedHeaders",
		"X-Amz-Security-Token",
	} {
		if len(query[name]) > 1 {
			return fmt.Errorf("%w: %s", errDuplicatePresignParameter, name)
		}
	}
	return nil
}

func requiredPresignParam(query url.Values, name string, missing error) (string, error) {
	values, ok := query[name]
	if !ok || len(values) == 0 || values[0] == "" {
		return "", missing
	}
	return values[0], nil
}

func signatureFromPresignedURI(signedURI string) (string, error) {
	parsed, err := url.Parse(signedURI)
	if err != nil {
		return "", err
	}
	signatures := parsed.Query()["X-Amz-Signature"]
	if len(signatures) != 1 || signatures[0] == "" {
		return "", errMissingSignature
	}
	return signatures[0], nil
}

func parseAuthHeader(header string) (accessKey, scope string, signedHeaders []string, signature string, err error) {
	if !strings.HasPrefix(header, "AWS4-HMAC-SHA256 ") {
		return "", "", nil, "", errUnsupportedAuthScheme
	}
	rest := strings.TrimPrefix(header, "AWS4-HMAC-SHA256 ")
	for _, field := range strings.Split(rest, ",") {
		field = strings.TrimSpace(field)
		switch {
		case strings.HasPrefix(field, "Credential="):
			cred := strings.TrimPrefix(field, "Credential=")
			elements := strings.Split(cred, "/")
			if len(elements) != 5 {
				return "", "", nil, "", errInvalidCredential
			}
			accessKey = elements[0]
			scope = strings.Join(elements[1:], "/")
		case strings.HasPrefix(field, "SignedHeaders="):
			raw := strings.TrimPrefix(field, "SignedHeaders=")
			signedHeaders, err = parseSignedHeaders(raw)
			if err != nil {
				return "", "", nil, "", err
			}
		case strings.HasPrefix(field, "Signature="):
			signature = strings.TrimSpace(strings.TrimPrefix(field, "Signature="))
		}
	}
	if accessKey == "" {
		return "", "", nil, "", errMissingAccessKey
	}
	if signature == "" {
		return "", "", nil, "", errMissingAuthSignature
	}
	return accessKey, scope, signedHeaders, signature, nil
}

func parseSignedHeaders(raw string) ([]string, error) {
	if raw == "" {
		return nil, errMissingSignedHeaders
	}
	var signedHeaders []string
	for _, h := range strings.Split(raw, ";") {
		h = strings.TrimSpace(h)
		if h != "" {
			signedHeaders = append(signedHeaders, strings.ToLower(h))
		}
	}
	if len(signedHeaders) == 0 {
		return nil, errMissingSignedHeaders
	}
	return signedHeaders, nil
}

func (v *sigV4Verifier) checkDateSkew(amzDate string) error {
	if amzDate == "" {
		return errMissingDate
	}
	t, err := parseAmzDateE(amzDate)
	if err != nil {
		return err
	}
	if delta := v.now().Sub(t); delta > inboundSkew || delta < -inboundSkew {
		return errRequestExpired
	}
	return nil
}

func (v *sigV4Verifier) checkPresignWindow(r *http.Request, amzDate string) error {
	expiresRaw := r.URL.Query().Get("X-Amz-Expires")
	expiresSeconds, err := strconv.ParseInt(expiresRaw, 10, 64)
	if err != nil {
		return fmt.Errorf("%w: %q", errInvalidExpires, expiresRaw)
	}
	if expiresSeconds < 0 {
		return fmt.Errorf("%w: %q", errInvalidExpires, expiresRaw)
	}
	if expiresSeconds > int64(maxPresignTTL/time.Second) {
		return fmt.Errorf("%w: %q", errInvalidExpires, expiresRaw)
	}
	signedAt, err := parseAmzDateE(amzDate)
	if err != nil {
		return err
	}
	now := v.now()
	if signedAt.Sub(now) > inboundSkew {
		return errRequestExpired
	}
	if now.After(signedAt.Add(time.Duration(expiresSeconds) * time.Second)) {
		return errPresignExpired
	}
	return nil
}

func parseAmzDate(s string) time.Time {
	t, _ := parseAmzDateE(s)
	return t
}

func parseAmzDateE(s string) (time.Time, error) {
	return time.Parse("20060102T150405Z", s)
}

func validateScope(scope, date string) (string, error) {
	elements := strings.Split(scope, "/")
	if len(elements) != 4 {
		return "", errInvalidCredential
	}
	if elements[0] != date {
		return "", errInvalidCredential
	}
	if elements[1] == "" {
		return "", errInvalidCredential
	}
	if elements[2] != inboundService {
		return "", errInvalidCredential
	}
	if elements[3] != "aws4_request" {
		return "", errInvalidCredential
	}
	return elements[1], nil
}

// stripUnsignedHeaders removes every header from clone.Header that is not in
// the signedHeaders allowlist. "host" and "x-amz-content-sha256" and
// "x-amz-date" are always retained because the AWS SDK signer relies on them.
func stripUnsignedHeaders(clone *http.Request, signedHeaders []string) {
	allowed := make(map[string]bool, len(signedHeaders)+3)
	for _, h := range signedHeaders {
		allowed[h] = true
	}
	allowed["host"] = true
	allowed["x-amz-content-sha256"] = true
	allowed["x-amz-date"] = true
	allowed["authorization"] = false

	for name := range clone.Header {
		if !allowed[strings.ToLower(name)] {
			clone.Header.Del(name)
		}
	}
}

func filterAuthenticatedHeaders(r *http.Request, signedHeaders []string) {
	allowed := make(map[string]bool, len(signedHeaders)+1)
	for _, h := range signedHeaders {
		allowed[h] = true
	}
	allowed["range"] = true

	for name := range r.Header {
		canonical := strings.ToLower(name)
		if allowed[canonical] || isRequiredProxyHeader(canonical) {
			continue
		}
		r.Header.Del(name)
	}
}

func isRequiredProxyHeader(name string) bool {
	switch name {
	case "authorization", "x-amz-content-sha256", "x-amz-date", "content-length":
		return true
	default:
		return false
	}
}

func containsHeader(headers []string, name string) bool {
	for _, h := range headers {
		if h == name {
			return true
		}
	}
	return false
}

func validatePayloadHashClaim(r *http.Request, signedHeaders []string) (string, error) {
	payloadHash := r.Header.Get("X-Amz-Content-Sha256")
	if payloadHash == "" {
		return unsignedPayloadSentinel, nil
	}
	if payloadHash == unsignedPayloadSentinel {
		return payloadHash, nil
	}
	if !containsHeader(signedHeaders, "x-amz-content-sha256") {
		return "", errUnsignedPayloadHash
	}
	decoded, err := hex.DecodeString(payloadHash)
	if err != nil || len(decoded) != sha256.Size {
		return "", errInvalidPayloadHash
	}
	return payloadHash, nil
}

func (v *sigV4Verifier) verifyPayloadBody(r *http.Request, payloadHash string) error {
	if payloadHash == "" || payloadHash == unsignedPayloadSentinel {
		return nil
	}
	decoded, err := hex.DecodeString(payloadHash)
	if err != nil || len(decoded) != sha256.Size {
		return errInvalidPayloadHash
	}
	err = v.replayBudget.Ensure(r)
	if err != nil {
		return err
	}
	verified := false
	defer func() {
		if !verified {
			_ = replaybody.Release(r)
		}
	}()
	body := r.Body
	if r.GetBody != nil {
		body, err = r.GetBody()
		if err != nil {
			return err
		}
	}
	if body == nil {
		body = http.NoBody
	}
	defer body.Close()
	h := sha256.New()
	if _, err := io.Copy(h, body); err != nil {
		return err
	}
	if subtle.ConstantTimeCompare(h.Sum(nil), decoded) != 1 {
		return errPayloadHashMismatch
	}
	verified = true
	return nil
}

func cloneForSigning(r *http.Request) *http.Request {
	clone := r.Clone(context.Background())
	clone.Body = http.NoBody
	return clone
}

const unsignedPayloadSentinel = "UNSIGNED-PAYLOAD"

var (
	errUnknownAccessKey          = AuthError("unknown access key")
	errSignatureMismatch         = AuthError("signature does not match")
	errMissingAuth               = AuthError("missing Authorization header")
	errMissingAuthSignature      = AuthError("Signature not found in Authorization")
	errMissingSignature          = AuthError("missing X-Amz-Signature")
	errDuplicatePresignParameter = AuthError("duplicate SigV4 presign parameter")
	errMissingAccessKey          = AuthError("access key not found in credentials")
	errMissingSignedHeaders      = AuthError("SignedHeaders not found in Authorization")
	errInvalidCredential         = AuthError("invalid credential format")
	errMissingExpires            = AuthError("missing X-Amz-Expires")
	errInvalidExpires            = AuthError("invalid X-Amz-Expires")
	errUnsupportedAuthScheme     = AuthError("unsupported authorization scheme")
	errRequestExpired            = AuthError("request timestamp outside allowed skew")
	errPresignExpired            = AuthError("presigned request has expired")
	errMissingDate               = AuthError("missing X-Amz-Date")
	errUnsignedDate              = AuthError("X-Amz-Date must be signed")
	errInvalidPayloadHash        = AuthError("invalid X-Amz-Content-Sha256")
	errUnsignedPayloadHash       = AuthError("X-Amz-Content-Sha256 must be signed")
	errPayloadHashMismatch       = AuthError("payload hash does not match")
)

type AuthError string

func (e AuthError) Error() string { return string(e) }
