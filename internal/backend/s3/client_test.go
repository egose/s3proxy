package s3

import (
	"context"
	"errors"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/egose/s3proxy/internal/config"
	"github.com/egose/s3proxy/internal/replaybody"
	"github.com/egose/s3proxy/internal/s3ops"
)

type generatedReadCloser struct {
	remaining int64
}

func (r *generatedReadCloser) Read(p []byte) (int, error) {
	if r.remaining == 0 {
		return 0, io.EOF
	}
	if int64(len(p)) > r.remaining {
		p = p[:r.remaining]
	}
	for i := range p {
		p[i] = 'x'
	}
	r.remaining -= int64(len(p))
	return len(p), nil
}

func (*generatedReadCloser) Close() error { return nil }

func mustURL(t *testing.T, raw string) *url.URL {
	t.Helper()
	u, err := url.Parse(raw)
	if err != nil {
		t.Fatal(err)
	}
	return u
}

func testTargets(t *testing.T, endpoint string, mutate ...func(*config.S3Target)) map[string]config.S3Target {
	t.Helper()
	target := config.S3Target{
		Name:           "primary",
		Endpoint:       endpoint,
		EndpointURL:    mustURL(t, endpoint),
		Region:         "us-east-1",
		ForcePathStyle: true,
		Credentials: config.StaticCredential{
			AccessKey: "ak",
			SecretKey: "sk",
		},
	}
	for _, fn := range mutate {
		fn(&target)
	}
	return map[string]config.S3Target{"primary": target}
}

func testBudget() *replaybody.Budget {
	return replaybody.NewBudget(replaybody.DefaultMaxBytes, replaybody.DefaultAggregateMaxBytes)
}

func newTestClient(t *testing.T, httpClient *http.Client, targets map[string]config.S3Target, budget *replaybody.Budget) Executor {
	t.Helper()
	client, err := NewClient(httpClient, targets, budget)
	if err != nil {
		t.Fatal(err)
	}
	return client
}

func TestNewClientRequiresDependencies(t *testing.T) {
	if _, err := NewClient(nil, testTargets(t, "https://s3.internal"), testBudget()); err == nil {
		t.Fatal("expected missing HTTP client error")
	}
	if _, err := NewClient(&http.Client{}, testTargets(t, "https://s3.internal"), nil); err == nil {
		t.Fatal("expected missing replay budget error")
	}
}

func TestBuildTargetURL_PreservesEndpointBasePath(t *testing.T) {
	src, err := http.NewRequest(http.MethodGet, "http://proxy.local/source/object?list-type=2", nil)
	if err != nil {
		t.Fatal(err)
	}

	targetURL, err := buildTargetURL(config.S3Target{
		Endpoint:       "https://minio.internal/base/path",
		EndpointURL:    mustURL(t, "https://minio.internal/base/path"),
		ForcePathStyle: true,
	}, "bucket", "key", src)
	if err != nil {
		t.Fatal(err)
	}

	if got, want := targetURL.Path, "/base/path/bucket/key"; got != want {
		t.Fatalf("Path = %q, want %q", got, want)
	}
	if got, want := targetURL.RawQuery, "list-type=2"; got != want {
		t.Fatalf("RawQuery = %q, want %q", got, want)
	}
}

func TestBuildTargetURL_PreservesEscapedKey(t *testing.T) {
	src, err := http.NewRequest(http.MethodGet, "http://proxy.local/source/object", nil)
	if err != nil {
		t.Fatal(err)
	}

	targetURL, err := buildTargetURL(config.S3Target{
		Endpoint:       "https://minio.internal",
		EndpointURL:    mustURL(t, "https://minio.internal"),
		ForcePathStyle: true,
	}, "bucket", "path%2Fwith%2Fslash.txt", src)
	if err != nil {
		t.Fatal(err)
	}
	if got, want := targetURL.RawPath, "/bucket/path%2Fwith%2Fslash.txt"; got != want {
		t.Fatalf("RawPath = %q, want %q", got, want)
	}
	if got, want := targetURL.Path, "/bucket/path/with/slash.txt"; got != want {
		t.Fatalf("Path = %q, want %q", got, want)
	}
}

func TestBuildTargetURL_PreservesOpaquePathStyleKey(t *testing.T) {
	src, err := http.NewRequest(http.MethodGet, "http://proxy.local/source/object", nil)
	if err != nil {
		t.Fatal(err)
	}

	targetURL, err := buildTargetURL(config.S3Target{
		Endpoint:       "https://minio.internal",
		EndpointURL:    mustURL(t, "https://minio.internal"),
		ForcePathStyle: true,
	}, "bucket", "foo//bar/../baz/./qux", src)
	if err != nil {
		t.Fatal(err)
	}
	if got, want := targetURL.RawPath, "/bucket/foo//bar/../baz/./qux"; got != want {
		t.Fatalf("RawPath = %q, want %q", got, want)
	}
	if got, want := targetURL.Path, "/bucket/foo//bar/../baz/./qux"; got != want {
		t.Fatalf("Path = %q, want %q", got, want)
	}
}

func TestBuildTargetURL_StripsInboundAuthQueryParams(t *testing.T) {
	src, err := http.NewRequest(http.MethodGet, "http://proxy.local/source/object?list-type=2&X-Amz-Algorithm=AWS4-HMAC-SHA256&X-Amz-Credential=test&X-Amz-Signature=deadbeef", nil)
	if err != nil {
		t.Fatal(err)
	}

	targetURL, err := buildTargetURL(config.S3Target{
		Endpoint:       "https://minio.internal/base/path",
		EndpointURL:    mustURL(t, "https://minio.internal/base/path"),
		ForcePathStyle: true,
	}, "bucket", "key", src)
	if err != nil {
		t.Fatal(err)
	}

	query, err := url.ParseQuery(targetURL.RawQuery)
	if err != nil {
		t.Fatal(err)
	}
	if got, want := query.Get("list-type"), "2"; got != want {
		t.Fatalf("list-type = %q, want %q", got, want)
	}
	if got := query.Get("X-Amz-Algorithm"); got != "" {
		t.Fatalf("expected X-Amz-Algorithm to be stripped, got %q", got)
	}
	if got := query.Get("X-Amz-Credential"); got != "" {
		t.Fatalf("expected X-Amz-Credential to be stripped, got %q", got)
	}
	if got := query.Get("X-Amz-Signature"); got != "" {
		t.Fatalf("expected X-Amz-Signature to be stripped, got %q", got)
	}
}

func TestClientDo_RespectsTargetTimeout(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(100 * time.Millisecond)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	src, err := http.NewRequest(http.MethodGet, "http://proxy.local/bucket/key", nil)
	if err != nil {
		t.Fatal(err)
	}

	client := newTestClient(t, &http.Client{}, testTargets(t, server.URL, func(target *config.S3Target) { target.Timeout = 20 * time.Millisecond }), testBudget())
	_, err = client.Do(context.Background(), Request{
		Operation: s3ops.OpGetObject,
		Target:    "primary",
		Bucket:    "bucket",
		Key:       "key",
		Source:    src,
	})
	if err == nil {
		t.Fatal("expected timeout error")
	}
	if !errors.Is(err, context.DeadlineExceeded) {
		var netErr net.Error
		if !errors.As(err, &netErr) || !netErr.Timeout() {
			if !strings.Contains(err.Error(), "timeout") && !strings.Contains(err.Error(), "deadline exceeded") {
				t.Fatalf("expected timeout error, got %v", err)
			}
		}
	}
}

func TestClientDo_SanitizesTransportURLInErrors(t *testing.T) {
	httpClient := &http.Client{Transport: roundTripFunc(func(*http.Request) (*http.Response, error) {
		return nil, errors.New("dial tcp connection refused")
	})}
	src, err := http.NewRequest(http.MethodGet, "http://proxy.local/bucket/key?token=sentinel-query-value", nil)
	if err != nil {
		t.Fatal(err)
	}

	client := newTestClient(t, httpClient, testTargets(t, "http://user:pass@upstream.local"), testBudget())
	_, err = client.Do(context.Background(), Request{
		Operation: s3ops.OpGetObject,
		Target:    "primary",
		Bucket:    "private-bucket",
		Key:       "secret-object",
		Source:    src,
	})
	if err == nil {
		t.Fatal("expected transport error")
	}
	text := err.Error()
	if !strings.Contains(text, "upstream request failed") || !strings.Contains(text, "http://upstream.local") {
		t.Fatalf("error = %q, want sanitized upstream host", text)
	}
	for _, forbidden := range []string{"sentinel-query-value", "user:pass", "private-bucket", "secret-object", "token="} {
		if strings.Contains(text, forbidden) {
			t.Fatalf("error = %q, must not contain %q", text, forbidden)
		}
	}
}

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

func TestClientDo_TargetTimeoutSurvivesUntilResponseBodyEOF(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		flusher, ok := w.(http.Flusher)
		if !ok {
			http.Error(w, "cannot flush", http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusOK)
		flusher.Flush()
		time.Sleep(20 * time.Millisecond)
		w.Write([]byte("ok"))
	}))
	defer server.Close()

	src, err := http.NewRequest(http.MethodGet, "http://proxy.local/bucket/key", nil)
	if err != nil {
		t.Fatal(err)
	}

	client := newTestClient(t, &http.Client{}, testTargets(t, server.URL, func(target *config.S3Target) { target.Timeout = 200 * time.Millisecond }), testBudget())
	resp, err := client.Do(context.Background(), Request{
		Operation: s3ops.OpGetObject,
		Target:    "primary",
		Bucket:    "bucket",
		Key:       "key",
		Source:    src,
	})
	if err != nil {
		t.Fatalf("Do failed: %v", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("read body: %v", err)
	}
	if got, want := string(body), "ok"; got != want {
		t.Fatalf("body = %q, want %q", got, want)
	}
}

func TestClientDo_TargetTimeoutCancelsSlowResponseBody(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		flusher, ok := w.(http.Flusher)
		if !ok {
			http.Error(w, "cannot flush", http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusOK)
		flusher.Flush()
		time.Sleep(80 * time.Millisecond)
		w.Write([]byte("late"))
	}))
	defer server.Close()

	src, err := http.NewRequest(http.MethodGet, "http://proxy.local/bucket/key", nil)
	if err != nil {
		t.Fatal(err)
	}

	client := newTestClient(t, &http.Client{}, testTargets(t, server.URL, func(target *config.S3Target) { target.Timeout = 20 * time.Millisecond }), testBudget())
	resp, err := client.Do(context.Background(), Request{
		Operation: s3ops.OpGetObject,
		Target:    "primary",
		Bucket:    "bucket",
		Key:       "key",
		Source:    src,
	})
	if err != nil {
		t.Fatalf("Do failed: %v", err)
	}
	defer resp.Body.Close()

	_, err = io.ReadAll(resp.Body)
	if err == nil {
		t.Fatal("expected timeout while reading body")
	}
	if !errors.Is(err, context.DeadlineExceeded) {
		var netErr net.Error
		if !errors.As(err, &netErr) || !netErr.Timeout() {
			if !strings.Contains(err.Error(), "timeout") && !strings.Contains(err.Error(), "deadline exceeded") {
				t.Fatalf("expected timeout error, got %v", err)
			}
		}
	}
}

func TestClientDo_StreamsKnownLengthSourceBody(t *testing.T) {
	var bodies []string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, err := io.ReadAll(r.Body)
		if err != nil {
			t.Fatalf("read body: %v", err)
		}
		bodies = append(bodies, string(body))
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	src, err := http.NewRequest(http.MethodPut, "http://proxy.local/bucket/key", nil)
	if err != nil {
		t.Fatal(err)
	}
	src.Body = io.NopCloser(strings.NewReader("payload"))
	src.ContentLength = int64(len("payload"))

	client := newTestClient(t, &http.Client{}, testTargets(t, server.URL), testBudget())
	request := Request{
		Operation: s3ops.OpPutObject,
		Target:    "primary",
		Bucket:    "bucket",
		Key:       "key",
		Source:    src,
	}

	resp, err := client.Do(context.Background(), request)
	if err != nil {
		t.Fatalf("Do failed: %v", err)
	}
	resp.Body.Close()

	if got, want := len(bodies), 1; got != want {
		t.Fatalf("calls = %d, want %d", got, want)
	}
	for i, body := range bodies {
		if body != "payload" {
			t.Fatalf("body[%d] = %q, want payload", i, body)
		}
	}
	if src.GetBody != nil {
		t.Fatal("expected known-length single-destination source body to stay streaming")
	}
}

func TestClientDo_ReusesReplayableSourceBody(t *testing.T) {
	var bodies []string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, err := io.ReadAll(r.Body)
		if err != nil {
			t.Fatalf("read body: %v", err)
		}
		bodies = append(bodies, string(body))
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	src, err := http.NewRequest(http.MethodPut, "http://proxy.local/bucket/key", nil)
	if err != nil {
		t.Fatal(err)
	}
	src.GetBody = func() (io.ReadCloser, error) {
		return io.NopCloser(strings.NewReader("payload")), nil
	}
	src.Body, err = src.GetBody()
	if err != nil {
		t.Fatal(err)
	}
	src.ContentLength = int64(len("payload"))

	client := newTestClient(t, &http.Client{}, testTargets(t, server.URL), testBudget())
	request := Request{
		Operation: s3ops.OpPutObject,
		Target:    "primary",
		Bucket:    "bucket",
		Key:       "key",
		Source:    src,
	}

	resp, err := client.Do(context.Background(), request)
	if err != nil {
		t.Fatalf("first Do failed: %v", err)
	}
	resp.Body.Close()
	resp, err = client.Do(context.Background(), request)
	if err != nil {
		t.Fatalf("second Do failed: %v", err)
	}
	resp.Body.Close()

	if got, want := len(bodies), 2; got != want {
		t.Fatalf("calls = %d, want %d", got, want)
	}
	for i, body := range bodies {
		if body != "payload" {
			t.Fatalf("body[%d] = %q, want payload", i, body)
		}
	}
}

func TestClientDo_UsesGetBodyOncePerRequest(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, err := io.ReadAll(r.Body)
		if err != nil {
			t.Fatalf("read body: %v", err)
		}
		if got, want := string(body), "payload"; got != want {
			t.Fatalf("body = %q, want %q", got, want)
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	src, err := http.NewRequest(http.MethodPut, "http://proxy.local/bucket/key", nil)
	if err != nil {
		t.Fatal(err)
	}
	getBodyCalls := 0
	src.GetBody = func() (io.ReadCloser, error) {
		getBodyCalls++
		if getBodyCalls > 1 {
			return nil, errors.New("GetBody called too many times")
		}
		return io.NopCloser(strings.NewReader("payload")), nil
	}
	src.Body, err = src.GetBody()
	if err != nil {
		t.Fatal(err)
	}
	getBodyCalls = 0
	src.ContentLength = int64(len("payload"))

	client := newTestClient(t, &http.Client{}, testTargets(t, server.URL), testBudget())
	resp, err := client.Do(context.Background(), Request{
		Operation: s3ops.OpPutObject,
		Target:    "primary",
		Bucket:    "bucket",
		Key:       "key",
		Source:    src,
	})
	if err != nil {
		t.Fatalf("Do failed: %v", err)
	}
	resp.Body.Close()
	if got, want := getBodyCalls, 1; got != want {
		t.Fatalf("GetBody calls = %d, want %d", got, want)
	}
}

func TestClientDo_BuffersUnknownLengthSourceBodyWithinLimit(t *testing.T) {
	var bodies []string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, err := io.ReadAll(r.Body)
		if err != nil {
			t.Fatalf("read body: %v", err)
		}
		bodies = append(bodies, string(body))
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	src, err := http.NewRequest(http.MethodPut, "http://proxy.local/bucket/key", nil)
	if err != nil {
		t.Fatal(err)
	}
	src.Body = io.NopCloser(strings.NewReader("payload"))
	src.ContentLength = -1

	client := newTestClient(t, &http.Client{}, testTargets(t, server.URL), replaybody.NewBudget(16, replaybody.DefaultAggregateMaxBytes))
	request := Request{
		Operation: s3ops.OpPutObject,
		Target:    "primary",
		Bucket:    "bucket",
		Key:       "key",
		Source:    src,
	}

	resp, err := client.Do(context.Background(), request)
	if err != nil {
		t.Fatalf("first Do failed: %v", err)
	}
	resp.Body.Close()
	resp, err = client.Do(context.Background(), request)
	if err != nil {
		t.Fatalf("second Do failed: %v", err)
	}
	resp.Body.Close()

	if got, want := len(bodies), 2; got != want {
		t.Fatalf("calls = %d, want %d", got, want)
	}
	if src.GetBody == nil {
		t.Fatal("expected unknown-length source body to become replayable")
	}
	for i, body := range bodies {
		if body != "payload" {
			t.Fatalf("body[%d] = %q, want payload", i, body)
		}
	}
}

func TestClientDo_RejectsOversizedUnknownLengthSourceBody(t *testing.T) {
	calls := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	src, err := http.NewRequest(http.MethodPut, "http://proxy.local/bucket/key", nil)
	if err != nil {
		t.Fatal(err)
	}
	src.Body = &generatedReadCloser{remaining: replaybody.DefaultMaxBytes + 1}
	src.ContentLength = -1

	client := newTestClient(t, &http.Client{}, testTargets(t, server.URL), replaybody.NewBudget(3, replaybody.DefaultAggregateMaxBytes))
	_, err = client.Do(context.Background(), Request{
		Operation: s3ops.OpPutObject,
		Target:    "primary",
		Bucket:    "bucket",
		Key:       "key",
		Source:    src,
	})
	if !replaybody.IsTooLarge(err) {
		t.Fatalf("expected oversized replay error, got %v", err)
	}
	if calls != 0 {
		t.Fatalf("unexpected upstream call count = %d, want 0", calls)
	}
}

func TestClientDo_ObservesReplayBudgetConsumedByEarlierStage(t *testing.T) {
	calls := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	budget := replaybody.NewBudget(10, 3)
	earlier, err := http.NewRequest(http.MethodPut, "http://proxy.local/bucket/key", nil)
	if err != nil {
		t.Fatal(err)
	}
	earlier.Body = io.NopCloser(strings.NewReader("abc"))
	earlier.ContentLength = -1
	if err := budget.Ensure(earlier); err != nil {
		t.Fatal(err)
	}
	defer replaybody.Release(earlier)

	src, err := http.NewRequest(http.MethodPut, "http://proxy.local/bucket/key", nil)
	if err != nil {
		t.Fatal(err)
	}
	src.Body = io.NopCloser(strings.NewReader("x"))
	src.ContentLength = -1

	client := newTestClient(t, &http.Client{}, testTargets(t, server.URL), budget)
	_, err = client.Do(context.Background(), Request{
		Operation: s3ops.OpPutObject,
		Target:    "primary",
		Bucket:    "bucket",
		Key:       "key",
		Source:    src,
	})
	if !replaybody.IsBudgetExhausted(err) {
		t.Fatalf("err = %v, want budget exhausted", err)
	}
	if calls != 0 {
		t.Fatalf("upstream calls = %d, want 0", calls)
	}
}

func TestClientDo_DoesNotForwardHopByHopHeaders(t *testing.T) {
	var gotConnection string
	var gotCustom string
	var gotTE string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotConnection = r.Header.Get("Connection")
		gotCustom = r.Header.Get("X-Remove-Me")
		gotTE = r.Header.Get("TE")
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	src, err := http.NewRequest(http.MethodGet, "http://proxy.local/bucket/key", nil)
	if err != nil {
		t.Fatal(err)
	}
	src.Header.Set("Connection", "X-Remove-Me")
	src.Header.Set("X-Remove-Me", "secret")
	src.Header.Set("TE", "trailers")

	client := newTestClient(t, &http.Client{}, testTargets(t, server.URL), testBudget())
	resp, err := client.Do(context.Background(), Request{
		Operation: s3ops.OpGetObject,
		Target:    "primary",
		Bucket:    "bucket",
		Key:       "key",
		Source:    src,
	})
	if err != nil {
		t.Fatalf("Do failed: %v", err)
	}
	resp.Body.Close()

	if gotConnection != "" {
		t.Fatalf("expected Connection to be stripped, got %q", gotConnection)
	}
	if gotCustom != "" {
		t.Fatalf("expected X-Remove-Me to be stripped, got %q", gotCustom)
	}
	if gotTE != "" {
		t.Fatalf("expected TE to be stripped, got %q", gotTE)
	}
}

func TestClientDo_ForwardsAuthenticatedControlHeadersOnly(t *testing.T) {
	var gotACL string
	var gotDate string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotACL = r.Header.Get("X-Amz-Acl")
		gotDate = r.Header.Get("X-Amz-Date")
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	src, err := http.NewRequest(http.MethodPut, "http://proxy.local/bucket/key", nil)
	if err != nil {
		t.Fatal(err)
	}
	src.Header.Set("X-Amz-Acl", "private")
	src.Header.Set("X-Amz-Date", "20240101T000000Z")

	client := newTestClient(t, &http.Client{}, testTargets(t, server.URL), testBudget())
	resp, err := client.Do(context.Background(), Request{
		Operation: s3ops.OpPutObject,
		Target:    "primary",
		Bucket:    "bucket",
		Key:       "key",
		Source:    src,
	})
	if err != nil {
		t.Fatalf("Do failed: %v", err)
	}
	resp.Body.Close()

	if gotACL != "private" {
		t.Fatalf("X-Amz-Acl = %q, want private", gotACL)
	}
	if gotDate == "20240101T000000Z" {
		t.Fatalf("inbound X-Amz-Date was forwarded")
	}
}

func TestClientDo_ForwardsAcceptEncodingHeader(t *testing.T) {
	var gotAcceptEncoding string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotAcceptEncoding = r.Header.Get("Accept-Encoding")
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	src, err := http.NewRequest(http.MethodGet, "http://proxy.local/bucket/key", nil)
	if err != nil {
		t.Fatal(err)
	}
	src.Header.Set("Accept-Encoding", "br")

	client := newTestClient(t, &http.Client{}, testTargets(t, server.URL), testBudget())
	resp, err := client.Do(context.Background(), Request{
		Operation: s3ops.OpGetObject,
		Target:    "primary",
		Bucket:    "bucket",
		Key:       "key",
		Source:    src,
	})
	if err != nil {
		t.Fatalf("Do failed: %v", err)
	}
	resp.Body.Close()

	if gotAcceptEncoding != "br" {
		t.Fatalf("Accept-Encoding = %q, want br", gotAcceptEncoding)
	}
}
