package auth

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	v4 "github.com/aws/aws-sdk-go-v2/aws/signer/v4"
	"github.com/egose/s3proxy/internal/config"
)

const (
	inboundService = "s3"
	inboundSkew    = 15 * time.Minute
)

type sigV4Verifier struct {
	signer        *v4.Signer
	clientsByAK   map[string]config.Client
	defaultRegion string
}

func newSigV4Verifier(clientsByAK map[string]config.Client, defaultRegion string) *sigV4Verifier {
	return &sigV4Verifier{
		signer:        v4.NewSigner(),
		clientsByAK:   clientsByAK,
		defaultRegion: defaultRegion,
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
	accessKey, scope, signedHeaders, err := parseAuthHeader(r.Header.Get("Authorization"))
	if err != nil {
		return nil, err
	}

	client, ok := v.clientsByAK[accessKey]
	if !ok {
		return nil, errUnknownAccessKey
	}

	amzDate := r.Header.Get("X-Amz-Date")
	if err := checkDateSkew(amzDate); err != nil {
		return nil, err
	}

	region := regionFromScope(scope, v.defaultRegion)
	bodyBytes, err := readAndRestoreBody(r)
	if err != nil {
		return nil, err
	}

	clone := cloneForSigning(r, bodyBytes)
	clone.Header.Del("Authorization")
	providedAuth := r.Header.Get("Authorization")

	// Strip any headers from the clone that the original signature did not
	// include (e.g. Accept-Encoding injected by the Go http transport). The
	// original SignedHeaders list governs what the signature covers, so the
	// re-signed request must include exactly those headers.
	stripUnsignedHeaders(clone, signedHeaders)

	creds := aws.Credentials{
		AccessKeyID:     client.AccessKey,
		SecretAccessKey: client.SecretKey,
	}

	payloadHash := clone.Header.Get("X-Amz-Content-Sha256")
	if payloadHash == "" {
		payloadHash = unsignedPayloadSentinel
	}

	if err := v.signer.SignHTTP(context.Background(), creds, clone, payloadHash, inboundService, region, parseAmzDate(amzDate)); err != nil {
		return nil, err
	}

	generated := clone.Header.Get("Authorization")
	if generated != providedAuth {
		return nil, errSignatureMismatch
	}

	return &Principal{
		Name:           client.Name,
		AccessKey:      client.AccessKey,
		AllowRoutes:    client.AllowRoutes,
		AllowOps:       client.AllowOps,
		VisibleBuckets: client.VisibleBuckets,
	}, nil
}

func (v *sigV4Verifier) verifyQuery(r *http.Request) (*Principal, error) {
	cred := r.URL.Query().Get("X-Amz-Credential")
	if cred == "" {
		return nil, errMissingAuth
	}

	parts := strings.Split(cred, "/")
	if len(parts) < 4 {
		return nil, errInvalidCredential
	}
	accessKey := parts[0]
	scope := strings.Join(parts[1:], "/")

	client, ok := v.clientsByAK[accessKey]
	if !ok {
		return nil, errUnknownAccessKey
	}

	date := r.URL.Query().Get("X-Amz-Date")
	if err := checkDateSkew(date); err != nil {
		return nil, err
	}

	region := regionFromScope(scope, v.defaultRegion)

	clone := r.Clone(context.Background())
	provided := r.URL.Query().Get("X-Amz-Signature")
	if provided == "" {
		return nil, errMissingSignature
	}

	creds := aws.Credentials{
		AccessKeyID:     client.AccessKey,
		SecretAccessKey: client.SecretKey,
	}

	_, _, err := v.signer.PresignHTTP(context.Background(), creds, clone, unsignedPayloadSentinel, inboundService, region, parseAmzDate(date))
	if err != nil {
		return nil, err
	}

	generated := clone.URL.Query().Get("X-Amz-Signature")
	if generated != provided {
		return nil, errSignatureMismatch
	}

	return &Principal{
		Name:           client.Name,
		AccessKey:      client.AccessKey,
		AllowRoutes:    client.AllowRoutes,
		AllowOps:       client.AllowOps,
		VisibleBuckets: client.VisibleBuckets,
	}, nil
}

func parseAuthHeader(header string) (accessKey, scope string, signedHeaders []string, err error) {
	if !strings.HasPrefix(header, "AWS4-HMAC-SHA256 ") {
		return "", "", nil, errUnsupportedAuthScheme
	}
	rest := strings.TrimPrefix(header, "AWS4-HMAC-SHA256 ")
	for _, field := range strings.Split(rest, ",") {
		field = strings.TrimSpace(field)
		switch {
		case strings.HasPrefix(field, "Credential="):
			cred := strings.TrimPrefix(field, "Credential=")
			elements := strings.Split(cred, "/")
			if len(elements) < 4 {
				return "", "", nil, errInvalidCredential
			}
			accessKey = elements[0]
			scope = strings.Join(elements[1:], "/")
		case strings.HasPrefix(field, "SignedHeaders="):
			raw := strings.TrimPrefix(field, "SignedHeaders=")
			for _, h := range strings.Split(raw, ";") {
				h = strings.TrimSpace(h)
				if h != "" {
					signedHeaders = append(signedHeaders, strings.ToLower(h))
				}
			}
		}
	}
	if accessKey == "" {
		return "", "", nil, errMissingAccessKey
	}
	if len(signedHeaders) == 0 {
		return "", "", nil, errMissingSignedHeaders
	}
	return accessKey, scope, signedHeaders, nil
}

func checkDateSkew(amzDate string) error {
	if amzDate == "" {
		return nil
	}
	t, err := parseAmzDateE(amzDate)
	if err != nil {
		return err
	}
	if delta := time.Since(t); delta > inboundSkew || delta < -inboundSkew {
		return errRequestExpired
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

func regionFromScope(scope, fallback string) string {
	elements := strings.Split(scope, "/")
	if len(elements) >= 2 {
		return elements[1]
	}
	return fallback
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

func readAndRestoreBody(r *http.Request) ([]byte, error) {
	if r.Body == nil || r.Body == http.NoBody {
		return nil, nil
	}
	body, err := io.ReadAll(r.Body)
	if err != nil {
		return nil, err
	}
	r.Body = io.NopCloser(bytes.NewReader(body))
	return body, nil
}

func cloneForSigning(r *http.Request, bodyBytes []byte) *http.Request {
	clone := r.Clone(context.Background())
	if bodyBytes != nil {
		clone.Body = io.NopCloser(bytes.NewReader(bodyBytes))
	}
	return clone
}

const unsignedPayloadSentinel = "UNSIGNED-PAYLOAD"

var (
	errUnknownAccessKey      = AuthError("unknown access key")
	errSignatureMismatch     = AuthError("signature does not match")
	errMissingAuth           = AuthError("missing Authorization header")
	errMissingSignature      = AuthError("missing X-Amz-Signature")
	errMissingAccessKey      = AuthError("access key not found in credentials")
	errMissingSignedHeaders  = AuthError("SignedHeaders not found in Authorization")
	errInvalidCredential     = AuthError("invalid credential format")
	errUnsupportedAuthScheme = AuthError("unsupported authorization scheme")
	errRequestExpired        = AuthError("request timestamp outside allowed skew")
)

type AuthError string

func (e AuthError) Error() string { return string(e) }
