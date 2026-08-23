package s3

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"github.com/egose/s3proxy/internal/config"
	"github.com/egose/s3proxy/internal/replaybody"
	"github.com/egose/s3proxy/internal/s3ops"
)

type Request struct {
	Operation s3ops.Operation
	Target    string
	Bucket    string
	Key       string
	Source    *http.Request
}

type Response struct {
	StatusCode int
	Header     http.Header
	Body       io.ReadCloser
}

type Executor interface {
	Do(ctx context.Context, req Request) (*Response, error)
}

func NewClient(httpClient *http.Client, targets map[string]config.S3Target, replayBudget *replaybody.Budget) (Executor, error) {
	if httpClient == nil {
		return nil, fmt.Errorf("http client is required")
	}
	if replayBudget == nil {
		return nil, fmt.Errorf("replay budget is required")
	}
	return &client{httpClient: httpClient, targets: cloneTargets(targets), replayBudget: replayBudget}, nil
}

type client struct {
	httpClient   *http.Client
	targets      map[string]config.S3Target
	replayBudget *replaybody.Budget
}

func (c *client) Do(ctx context.Context, req Request) (*Response, error) {
	target, err := c.target(req.Target)
	if err != nil {
		return nil, err
	}
	var cancel context.CancelFunc
	if target.Timeout > 0 {
		ctx, cancel = context.WithTimeout(ctx, target.Timeout)
	}
	cancelOnError := func() {
		if cancel != nil {
			cancel()
		}
	}

	targetURL, err := buildTargetURL(target, req.Bucket, req.Key, req.Source)
	if err != nil {
		cancelOnError()
		return nil, fmt.Errorf("build target URL: %w", err)
	}

	body, getBody, contentLength, err := c.prepareSourceBody(req.Source)
	if err != nil {
		cancelOnError()
		return nil, err
	}

	outReq, err := http.NewRequestWithContext(ctx, req.Source.Method, targetURL.String(), nil)
	if err != nil {
		cancelOnError()
		return nil, fmt.Errorf("create outbound request: %w", err)
	}
	if body != nil {
		outReq.Body = body
		outReq.GetBody = getBody
		outReq.ContentLength = contentLength
		// Explicitly set the Content-Length header so the AWS SigV4 signer
		// includes it in SignedHeaders. Without this, net/http adds the
		// header at send time (after signing), causing MinIO/S3 backends to
		// reject the request with SignatureDoesNotMatch.
		outReq.Header.Set("Content-Length", strconv.FormatInt(contentLength, 10))
	}
	outReq.Host = targetURL.Host

	connectionTokens := connectionHeaderTokens(req.Source.Header)
	for key, vals := range req.Source.Header {
		if !shouldForwardHeader(key, connectionTokens) {
			continue
		}
		for _, v := range vals {
			outReq.Header.Add(key, v)
		}
	}
	outReq.Header.Set("Host", targetURL.Host)

	if err := signRequest(outReq, target); err != nil {
		cancelOnError()
		return nil, fmt.Errorf("sign outbound request: %w", err)
	}

	resp, err := c.httpClient.Do(outReq)
	if err != nil {
		cancelOnError()
		return nil, fmt.Errorf("upstream request failed: %w", sanitizeHTTPClientError(err))
	}
	respBody := resp.Body
	if cancel != nil {
		if respBody == nil {
			cancel()
		} else {
			respBody = cancelOnCloseReadCloser{body: respBody, cancel: cancel}
		}
	}

	return &Response{
		StatusCode: resp.StatusCode,
		Header:     resp.Header,
		Body:       respBody,
	}, nil
}

func sanitizeHTTPClientError(err error) error {
	var urlErr *url.Error
	if errors.As(err, &urlErr) {
		return fmt.Errorf("%s %s: %w", urlErr.Op, safeURLForError(urlErr.URL), urlErr.Err)
	}
	return err
}

func safeURLForError(raw string) string {
	u, err := url.Parse(raw)
	if err != nil || u.Scheme == "" || u.Host == "" {
		return "[redacted-url]"
	}
	host := u.Hostname()
	if port := u.Port(); port != "" {
		host += ":" + port
	}
	return u.Scheme + "://" + host
}

func (c *client) target(name string) (config.S3Target, error) {
	if c == nil || c.targets == nil {
		return config.S3Target{}, fmt.Errorf("target registry is not configured")
	}
	target, ok := c.targets[name]
	if !ok {
		return config.S3Target{}, fmt.Errorf("target %q is not configured", name)
	}
	return cloneTarget(target), nil
}

type cancelOnCloseReadCloser struct {
	body   io.ReadCloser
	cancel context.CancelFunc
}

func (r cancelOnCloseReadCloser) Read(p []byte) (int, error) {
	n, err := r.body.Read(p)
	if err != nil {
		r.cancel()
	}
	return n, err
}

func (r cancelOnCloseReadCloser) Close() error {
	err := r.body.Close()
	r.cancel()
	return err
}

func (c *client) prepareSourceBody(src *http.Request) (io.ReadCloser, func() (io.ReadCloser, error), int64, error) {
	if src == nil || src.Body == nil || src.Body == http.NoBody {
		return nil, nil, 0, nil
	}
	if src.GetBody == nil && src.ContentLength >= 0 {
		return src.Body, nil, src.ContentLength, nil
	}
	if src.GetBody == nil {
		if c.replayBudget == nil {
			return nil, nil, 0, fmt.Errorf("replay budget is not configured")
		}
		if err := c.replayBudget.Ensure(src); err != nil {
			return nil, nil, 0, err
		}
	}
	body, err := src.GetBody()
	if err != nil {
		return nil, nil, 0, fmt.Errorf("get request body: %w", err)
	}
	return body, src.GetBody, src.ContentLength, nil
}

func buildTargetURL(target config.S3Target, bucket, key string, src *http.Request) (*url.URL, error) {
	if target.EndpointURL == nil {
		return nil, fmt.Errorf("target endpoint is not parsed")
	}
	base := cloneURL(target.EndpointURL)
	var err error

	var path string
	var rawPath string
	if target.ForcePathStyle {
		path, rawPath, err = buildObjectPath("/"+bucket, key)
		if err != nil {
			return nil, err
		}
	} else {
		if base.Host != "" {
			hostParts := strings.Split(base.Host, ".")
			base.Host = bucket + "." + hostParts[0]
			if len(hostParts) > 1 {
				base.Host += "." + strings.Join(hostParts[1:], ".")
			}
		}
		path, rawPath, err = buildObjectPath("", key)
		if err != nil {
			return nil, err
		}
	}
	joinedPath := joinURLPath(base.Path, path)
	joinedRawPath := joinURLPath(base.EscapedPath(), rawPath)

	targetURL := &url.URL{
		Scheme:   base.Scheme,
		Host:     base.Host,
		Path:     joinedPath,
		RawPath:  joinedRawPath,
		RawQuery: filteredQuery(src.URL.Query()).Encode(),
	}

	if target.ForcePathStyle {
		targetURL.Host = base.Host
	}

	return targetURL, nil
}

func cloneURL(src *url.URL) *url.URL {
	if src == nil {
		return nil
	}
	copy := *src
	return &copy
}

func cloneTargets(targets map[string]config.S3Target) map[string]config.S3Target {
	out := make(map[string]config.S3Target, len(targets))
	for name, target := range targets {
		out[name] = cloneTarget(target)
	}
	return out
}

func cloneTarget(target config.S3Target) config.S3Target {
	if target.EndpointURL != nil {
		endpoint := *target.EndpointURL
		target.EndpointURL = &endpoint
	}
	return target
}

func buildObjectPath(prefix, key string) (string, string, error) {
	rawKey := canonicalizeEscapedPath(key)
	decodedKey, err := url.PathUnescape(rawKey)
	if err != nil {
		return "", "", fmt.Errorf("unescape key path: %w", err)
	}
	if prefix == "" {
		if key == "" {
			return "/", "/", nil
		}
		return "/" + decodedKey, "/" + rawKey, nil
	}
	decodedPrefix := prefix
	rawPrefix := prefix
	if key == "" {
		return decodedPrefix, rawPrefix, nil
	}
	return joinObjectPath(decodedPrefix, decodedKey), joinObjectPath(rawPrefix, rawKey), nil
}

func joinObjectPath(prefix, key string) string {
	if prefix == "" {
		if key == "" {
			return "/"
		}
		return "/" + key
	}
	if key == "" {
		return prefix
	}
	if strings.HasSuffix(prefix, "/") {
		return prefix + key
	}
	return prefix + "/" + key
}

func canonicalizeEscapedPath(value string) string {
	if value == "" {
		return ""
	}
	var b strings.Builder
	b.Grow(len(value))
	for i := 0; i < len(value); i++ {
		c := value[i]
		if c == '%' && i+2 < len(value) && isHex(value[i+1]) && isHex(value[i+2]) {
			b.WriteByte('%')
			b.WriteByte(value[i+1])
			b.WriteByte(value[i+2])
			i += 2
			continue
		}
		if isSafePathByte(c) {
			b.WriteByte(c)
			continue
		}
		b.WriteByte('%')
		b.WriteByte(upperHex[c>>4])
		b.WriteByte(upperHex[c&0x0f])
	}
	return b.String()
}

const upperHex = "0123456789ABCDEF"

func isSafePathByte(c byte) bool {
	if c == '/' || c == '-' || c == '_' || c == '.' || c == '~' {
		return true
	}
	if c >= 'a' && c <= 'z' {
		return true
	}
	if c >= 'A' && c <= 'Z' {
		return true
	}
	if c >= '0' && c <= '9' {
		return true
	}
	return false
}

func isHex(c byte) bool {
	return (c >= '0' && c <= '9') || (c >= 'a' && c <= 'f') || (c >= 'A' && c <= 'F')
}

func shouldForwardHeader(name string, connectionTokens map[string]struct{}) bool {
	if isHopByHopHeader(name, connectionTokens) {
		return false
	}
	switch strings.ToLower(name) {
	case "authorization", "x-amz-security-token", "x-amz-decoded-content-length",
		"x-amz-content-sha256", "x-amz-date", "content-length":
		return false
	case "host":
		return false
	default:
		return true
	}
}

func isHopByHopHeader(name string, connectionTokens map[string]struct{}) bool {
	canonical := strings.ToLower(name)
	switch canonical {
	case "connection", "keep-alive", "proxy-authenticate", "proxy-authorization",
		"te", "trailer", "transfer-encoding", "upgrade":
		return true
	}
	_, ok := connectionTokens[canonical]
	return ok
}

func connectionHeaderTokens(headers http.Header) map[string]struct{} {
	tokens := make(map[string]struct{})
	for _, raw := range headers.Values("Connection") {
		for _, token := range strings.Split(raw, ",") {
			token = strings.ToLower(strings.TrimSpace(token))
			if token != "" {
				tokens[token] = struct{}{}
			}
		}
	}
	return tokens
}

func joinURLPath(prefix, suffix string) string {
	if prefix == "" || prefix == "/" {
		if suffix == "" {
			return "/"
		}
		return suffix
	}
	if suffix == "" || suffix == "/" {
		return strings.TrimRight(prefix, "/")
	}
	return strings.TrimRight(prefix, "/") + "/" + strings.TrimLeft(suffix, "/")
}

func filteredQuery(values url.Values) url.Values {
	if len(values) == 0 {
		return url.Values{}
	}
	out := make(url.Values, len(values))
	for key, vals := range values {
		if isInboundAuthQueryParam(key) {
			continue
		}
		copied := make([]string, len(vals))
		copy(copied, vals)
		out[key] = copied
	}
	return out
}

func isInboundAuthQueryParam(name string) bool {
	switch strings.ToLower(name) {
	case "x-amz-algorithm", "x-amz-credential", "x-amz-date", "x-amz-expires",
		"x-amz-security-token", "x-amz-signature", "x-amz-signedheaders":
		return true
	default:
		return false
	}
}
