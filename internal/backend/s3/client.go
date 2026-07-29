package s3

import (
	"context"
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
	Target    config.S3Target
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

func NewClient(httpClient *http.Client) Executor {
	return NewClientWithReplayLimit(httpClient, replaybody.DefaultMaxBytes)
}

func NewClientWithReplayLimit(httpClient *http.Client, replayBodyMaxBytes int64) Executor {
	return &client{httpClient: httpClient, replayBodyMaxBytes: replayBodyMaxBytes}
}

type client struct {
	httpClient         *http.Client
	replayBodyMaxBytes int64
}

func (c *client) Do(ctx context.Context, req Request) (*Response, error) {
	if req.Target.Timeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, req.Target.Timeout)
		defer cancel()
	}

	targetURL, err := buildTargetURL(req.Target, req.Bucket, req.Key, req.Source)
	if err != nil {
		return nil, fmt.Errorf("build target URL: %w", err)
	}

	body, getBody, contentLength, err := c.prepareSourceBody(req.Source)
	if err != nil {
		return nil, err
	}

	outReq, err := http.NewRequestWithContext(ctx, req.Source.Method, targetURL.String(), nil)
	if err != nil {
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

	for key, vals := range req.Source.Header {
		if !shouldForwardHeader(key, req.Source.Header) {
			continue
		}
		for _, v := range vals {
			outReq.Header.Add(key, v)
		}
	}
	outReq.Header.Set("Host", targetURL.Host)

	if err := signRequest(outReq, req.Target); err != nil {
		return nil, fmt.Errorf("sign outbound request: %w", err)
	}

	resp, err := c.httpClient.Do(outReq)
	if err != nil {
		return nil, fmt.Errorf("upstream request failed: %w", err)
	}

	return &Response{
		StatusCode: resp.StatusCode,
		Header:     resp.Header,
		Body:       resp.Body,
	}, nil
}

func (c *client) prepareSourceBody(src *http.Request) (io.ReadCloser, func() (io.ReadCloser, error), int64, error) {
	if src == nil || src.Body == nil || src.Body == http.NoBody {
		return nil, nil, 0, nil
	}
	if src.GetBody == nil && src.ContentLength >= 0 {
		return src.Body, nil, src.ContentLength, nil
	}
	if src.GetBody == nil {
		if err := replaybody.EnsureWithLimit(src, c.replayBodyMaxBytes); err != nil {
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
		b.WriteString(fmt.Sprintf("%%%02X", c))
	}
	return b.String()
}

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

func shouldForwardHeader(name string, headers http.Header) bool {
	if isHopByHopHeader(name, headers) {
		return false
	}
	switch strings.ToLower(name) {
	case "authorization", "x-amz-security-token", "x-amz-decoded-content-length",
		"x-amz-content-sha256", "content-length":
		return false
	case "host":
		return false
	default:
		return true
	}
}

func isHopByHopHeader(name string, headers http.Header) bool {
	canonical := strings.ToLower(name)
	switch canonical {
	case "connection", "keep-alive", "proxy-authenticate", "proxy-authorization",
		"te", "trailer", "transfer-encoding", "upgrade":
		return true
	}
	_, ok := connectionHeaderTokens(headers)[canonical]
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
