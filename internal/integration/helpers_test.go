//go:build integration

// Package integration hosts end-to-end tests that exercise the running s3proxy
// against the sandbox docker-compose stack (MinIO + SeaweedFS). The tests are
// gated by the `integration` build tag so they are not picked up by `go test`
// or `make test` unless explicitly requested via `-tags integration`.
//
// Expected environment (see .env.example and sandbox/README):
//
//	S3PROXY_URL                  e.g. http://localhost:8082
//	S3PROXY_CLIENT_CI_ACCESS_KEY
//	S3PROXY_CLIENT_CI_SECRET_KEY
//
// The sandbox stack is started by `make sandbox-up DAEMON=true` and torn down
// by `make sandbox-down` (or `make sandbox-destroy`).
package integration

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	v4 "github.com/aws/aws-sdk-go-v2/aws/signer/v4"
)

const defaultProxyURL = "http://localhost:8082"

func proxyURL() string {
	if v := os.Getenv("S3PROXY_URL"); v != "" {
		return v
	}
	return defaultProxyURL
}

func clientCreds() (string, string) {
	ak := os.Getenv("S3PROXY_CLIENT_CI_ACCESS_KEY")
	sk := os.Getenv("S3PROXY_CLIENT_CI_SECRET_KEY")
	if ak == "" || sk == "" {
		ak = "test-client-ak"
		sk = "test-client-sk"
	}
	return ak, sk
}

// waitForReady polls the proxy /healthz endpoint, failing the test if the
// service does not become reachable within wait.
func waitForReady(t *testing.T, wait time.Duration) {
	t.Helper()
	deadline := time.Now().Add(wait)
	for {
		resp, err := http.Get(proxyURL() + "/healthz")
		if err == nil && resp.StatusCode == 200 {
			resp.Body.Close()
			return
		}
		if resp != nil {
			resp.Body.Close()
		}
		if time.Now().After(deadline) {
			if err != nil {
				t.Fatalf("proxy never became ready at %s: %v", proxyURL(), err)
			}
			t.Fatalf("proxy never returned 200 from /healthz at %s", proxyURL())
		}
		time.Sleep(500 * time.Millisecond)
	}
}

// signRequest applies AWS SigV4 to r using the test client credentials and the
// proxy's host. The body is read once and restored so the same *http.Request
// can be sent by the http client.
func signRequest(t *testing.T, r *http.Request, body []byte) {
	t.Helper()
	ak, sk := clientCreds()
	creds := aws.Credentials{AccessKeyID: ak, SecretAccessKey: sk}
	if r.Header.Get("X-Amz-Content-Sha256") == "" {
		r.Header.Set("X-Amz-Content-Sha256", "UNSIGNED-PAYLOAD")
	}
	signer := v4.NewSigner()
	if err := signer.SignHTTP(context.Background(), creds, r, "UNSIGNED-PAYLOAD", "s3", "us-east-1", time.Now().UTC()); err != nil {
		t.Fatalf("SignHTTP failed: %v", err)
	}
}

// signedRequest builds a signed request and returns its response. Optionally
// includes a request body.
func signedRequest(t *testing.T, method, path string, body []byte, query url.Values) (*http.Response, []byte) {
	t.Helper()
	r := newProxyRequest(t, method, path, body, query)
	if body != nil {
		r.Header.Set("Content-Length", fmt.Sprintf("%d", len(body)))
	}
	signRequest(t, r, body)
	return doRequest(t, r)
}

func signedRequestWithHeaders(t *testing.T, method, path string, body []byte, query url.Values, headers http.Header) (*http.Response, []byte) {
	t.Helper()
	r := newProxyRequest(t, method, path, body, query)
	for k, vals := range headers {
		for _, v := range vals {
			r.Header.Add(k, v)
		}
	}
	if body != nil {
		r.Header.Set("Content-Length", fmt.Sprintf("%d", len(body)))
	}
	signRequest(t, r, body)
	return doRequest(t, r)
}

func signedHostRequest(t *testing.T, method, host, path string, body []byte, query url.Values) (*http.Response, []byte) {
	t.Helper()
	r := newProxyRequest(t, method, path, body, query)
	r.Host = host
	if body != nil {
		r.Header.Set("Content-Length", fmt.Sprintf("%d", len(body)))
	}
	signRequest(t, r, body)
	return doRequest(t, r)
}

func presignedRequest(t *testing.T, method, path string, query url.Values, signedAt time.Time, expires time.Duration) (*http.Response, []byte) {
	t.Helper()
	r := newProxyRequest(t, method, path, nil, query)
	signPresignedRequest(t, r, signedAt, expires)
	return doRequest(t, r)
}

func newProxyRequest(t *testing.T, method, path string, body []byte, query url.Values) *http.Request {
	t.Helper()
	target, err := url.Parse(proxyURL())
	if err != nil {
		t.Fatalf("parse proxy url: %v", err)
	}

	u := &url.URL{Scheme: target.Scheme, Host: target.Host, Path: path}
	if query != nil {
		u.RawQuery = query.Encode()
	}

	var bodyReader io.Reader
	if body != nil {
		bodyReader = bytes.NewReader(body)
	}
	r, err := http.NewRequest(method, u.String(), bodyReader)
	if err != nil {
		t.Fatalf("NewRequest failed: %v", err)
	}
	r.Host = u.Host
	return r
}

func signPresignedRequest(t *testing.T, r *http.Request, signedAt time.Time, expires time.Duration) {
	t.Helper()
	ak, sk := clientCreds()
	query := r.URL.Query()
	query.Set("X-Amz-Expires", strconv.FormatInt(int64(expires/time.Second), 10))
	r.URL.RawQuery = query.Encode()

	signedURI, _, err := v4.NewSigner().PresignHTTP(
		context.Background(),
		aws.Credentials{AccessKeyID: ak, SecretAccessKey: sk},
		r,
		"UNSIGNED-PAYLOAD",
		"s3",
		"us-east-1",
		signedAt,
	)
	if err != nil {
		t.Fatalf("PresignHTTP failed: %v", err)
	}
	parsed, err := url.Parse(signedURI)
	if err != nil {
		t.Fatalf("parse presigned url: %v", err)
	}
	r.URL = parsed
	r.Host = parsed.Host
}

func doRequest(t *testing.T, r *http.Request) (*http.Response, []byte) {
	t.Helper()

	resp, err := http.DefaultClient.Do(r)
	if err != nil {
		t.Fatalf("HTTP %s %s failed: %v", r.Method, r.URL.String(), err)
	}
	defer resp.Body.Close()
	respBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("read body failed: %v", err)
	}
	return resp, respBytes
}

// dumpResponse returns a human-readable representation of the response,
// useful for debugging failed assertions. A nil resp is tolerated so callers
// that only have the body bytes (e.g. assertBodyContains) can still dump.
func dumpResponse(resp *http.Response, body []byte) string {
	var b strings.Builder
	if resp != nil {
		b.WriteString(fmt.Sprintf("HTTP %d %s\n", resp.StatusCode, resp.Status))
		for k, v := range resp.Header {
			for _, vv := range v {
				b.WriteString(fmt.Sprintf("%s: %s\n", k, vv))
			}
		}
		b.WriteString("\n")
	}
	if len(body) > 512 {
		b.Write(body[:512])
		b.WriteString("... (")
		b.WriteString(fmt.Sprintf("%d", len(body)))
		b.WriteString(" total bytes)")
	} else {
		b.Write(body)
	}
	_ = httputil.DumpRequest // keep import referenced for potential future use
	return b.String()
}

func assertStatus(t *testing.T, resp *http.Response, body []byte, want int) {
	t.Helper()
	if resp.StatusCode != want {
		t.Errorf("status = %d, want %d\n%s", resp.StatusCode, want, dumpResponse(resp, body))
	}
}

func assertBodyContains(t *testing.T, body []byte, want string) {
	t.Helper()
	if !strings.Contains(string(body), want) {
		t.Errorf("body does not contain %q\n%s", want, dumpResponse(nil, body))
	}
}

func assertBodyNotContains(t *testing.T, body []byte, want string) {
	t.Helper()
	if strings.Contains(string(body), want) {
		t.Errorf("body should not contain %q\n%s", want, dumpResponse(nil, body))
	}
}
