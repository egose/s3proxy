package s3

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"github.com/egose/s3proxy/internal/config"
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
	return &client{httpClient: httpClient}
}

type client struct {
	httpClient *http.Client
}

func (c *client) Do(ctx context.Context, req Request) (*Response, error) {
	targetURL, err := buildTargetURL(req.Target, req.Bucket, req.Key, req.Source)
	if err != nil {
		return nil, fmt.Errorf("build target URL: %w", err)
	}

	// Buffer the body so we can set Content-Length explicitly. Without an
	// explicit Content-Length, net/http sends `Content-Length: 0` and a
	// chunked body when the source Body is an io.NopCloser wrapping a
	// bytes.Reader — and MinIO rejects that with 411 MissingContentLength.
	// Buffering also lets the per-destination body be replayed for fan-out.
	var bodyReader io.Reader
	if req.Source.Body != nil && req.Source.Body != http.NoBody {
		bodyBytes, err := io.ReadAll(req.Source.Body)
		if err != nil {
			return nil, fmt.Errorf("read request body: %w", err)
		}
		bodyReader = bytes.NewReader(bodyBytes)
	}

	outReq, err := http.NewRequestWithContext(ctx, req.Source.Method, targetURL.String(), bodyReader)
	if err != nil {
		return nil, fmt.Errorf("create outbound request: %w", err)
	}
	if br, ok := bodyReader.(*bytes.Reader); ok {
		outReq.Body = io.NopCloser(br)
		outReq.ContentLength = int64(br.Len())
		// Explicitly set the Content-Length header so the AWS SigV4 signer
		// includes it in SignedHeaders. Without this, net/http adds the
		// header at send time (after signing), causing MinIO/S3 backends to
		// reject the request with SignatureDoesNotMatch.
		outReq.Header.Set("Content-Length", strconv.Itoa(br.Len()))
	}
	outReq.Host = targetURL.Host

	for key, vals := range req.Source.Header {
		if !shouldForwardHeader(key) {
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

func buildTargetURL(target config.S3Target, bucket, key string, src *http.Request) (*url.URL, error) {
	base, err := url.Parse(target.Endpoint)
	if err != nil {
		return nil, err
	}

	var path string
	if target.ForcePathStyle {
		path = "/" + bucket
		if key != "" {
			path += "/" + key
		}
	} else {
		if base.Host != "" {
			hostParts := strings.Split(base.Host, ".")
			base.Host = bucket + "." + hostParts[0]
			if len(hostParts) > 1 {
				base.Host += "." + strings.Join(hostParts[1:], ".")
			}
		}
		path = "/" + key
	}

	targetURL := &url.URL{
		Scheme:   base.Scheme,
		Host:     base.Host,
		Path:     path,
		RawQuery: src.URL.RawQuery,
	}

	if target.ForcePathStyle {
		targetURL.Host = base.Host
	}

	return targetURL, nil
}

func shouldForwardHeader(name string) bool {
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
