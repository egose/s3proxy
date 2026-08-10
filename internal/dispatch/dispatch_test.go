package dispatch

import (
	"context"
	"errors"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/egose/s3proxy/internal/backend/s3"
	"github.com/egose/s3proxy/internal/config"
	"github.com/egose/s3proxy/internal/replaybody"
	"github.com/egose/s3proxy/internal/rewrite"
	"github.com/egose/s3proxy/internal/router"
	"github.com/egose/s3proxy/internal/s3ops"
)

type errorReader struct{}

func (errorReader) Read([]byte) (int, error) { return 0, errors.New("boom") }

func (errorReader) Close() error { return nil }

func TestDispatch_FanoutBodyReadError(t *testing.T) {
	d := &dispatcher{backend: &stubBackend{}}
	req, err := http.NewRequest(http.MethodPut, "http://proxy.local/bucket/key", nil)
	if err != nil {
		t.Fatal(err)
	}
	req.Body = errorReader{}

	_, err = d.Dispatch(context.Background(), router.Match{
		Route: config.Route{
			Dispatch:        config.DispatchAll,
			DestinationRefs: []string{"primary", "replica"},
		},
		Destinations: []config.S3Target{{Name: "primary"}, {Name: "replica"}},
	}, req, s3ops.OpPutObject, rewrite.Result{Bucket: "bucket", Key: "key"})
	if err == nil {
		t.Fatal("expected read request body error")
	}
	if !strings.Contains(err.Error(), "read request body") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestDispatch_FanoutRejectsOversizedBody(t *testing.T) {
	d := &dispatcher{backend: &stubBackend{}}
	req, err := http.NewRequest(http.MethodPut, "http://proxy.local/bucket/key", io.NopCloser(strings.NewReader("x")))
	if err != nil {
		t.Fatal(err)
	}
	req.ContentLength = replaybody.DefaultMaxBytes + 1

	_, err = d.Dispatch(context.Background(), router.Match{
		Route: config.Route{
			Dispatch:        config.DispatchAll,
			DestinationRefs: []string{"primary", "replica"},
		},
		Destinations: []config.S3Target{{Name: "primary"}, {Name: "replica"}},
	}, req, s3ops.OpPutObject, rewrite.Result{Bucket: "bucket", Key: "key"})
	if !replaybody.IsTooLarge(err) {
		t.Fatalf("expected oversized replay error, got %v", err)
	}
}

func TestDispatch_FanoutReplicaFailureClearsSuccessfulPrimary(t *testing.T) {
	primaryBody := &trackingReadCloser{Reader: strings.NewReader("ok")}
	backend := &stubBackend{
		responses: []stubCall{
			{resp: responseWithStatusAndBody(http.StatusOK, primaryBody)},
			{err: errors.New("replica write failed")},
		},
	}
	d := &dispatcher{backend: backend}
	req, err := http.NewRequest(http.MethodPut, "http://proxy.local/bucket/key", strings.NewReader("payload"))
	if err != nil {
		t.Fatal(err)
	}
	req.ContentLength = int64(len("payload"))

	result, err := d.Dispatch(context.Background(), router.Match{
		Route: config.Route{
			Dispatch:        config.DispatchAll,
			DestinationRefs: []string{"primary", "replica"},
		},
		Destinations: []config.S3Target{{Name: "primary"}, {Name: "replica"}},
	}, req, s3ops.OpPutObject, rewrite.Result{Bucket: "bucket", Key: "key"})
	if err == nil {
		t.Fatal("expected fan-out error")
	}
	if result == nil {
		t.Fatal("expected dispatch result")
	}
	if result.Primary != nil {
		t.Fatalf("expected primary response to be cleared on partial failure, got %#v", result.Primary)
	}
	if !primaryBody.closed {
		t.Fatal("expected successful primary body to be closed on partial failure")
	}
}

func TestDispatch_FanoutDrainsAndClosesExtraResponses(t *testing.T) {
	replicaBody := &trackingReadCloser{Reader: strings.NewReader("replica")}
	backend := &stubBackend{
		responses: []stubCall{
			{resp: responseWithStatus(http.StatusOK)},
			{resp: responseWithStatusAndBody(http.StatusOK, replicaBody)},
		},
	}
	d := &dispatcher{backend: backend}
	req, err := http.NewRequest(http.MethodPut, "http://proxy.local/bucket/key", strings.NewReader("payload"))
	if err != nil {
		t.Fatal(err)
	}
	req.ContentLength = int64(len("payload"))

	result, err := d.Dispatch(context.Background(), router.Match{
		Route:        config.Route{Dispatch: config.DispatchAll, DestinationRefs: []string{"primary", "replica"}},
		Destinations: []config.S3Target{{Name: "primary"}, {Name: "replica"}},
	}, req, s3ops.OpPutObject, rewrite.Result{Bucket: "bucket", Key: "key"})
	if err != nil {
		t.Fatalf("Dispatch error = %v", err)
	}
	if !replicaBody.closed {
		t.Fatal("expected replica response body to be closed")
	}
	if got := replicaBody.Reader.(*strings.Reader).Len(); got != 0 {
		t.Fatalf("replica body remaining = %d, want drained", got)
	}
	s3.DrainAndClose(result.Primary)
}

func TestDispatch_DiscardDrainIsBounded(t *testing.T) {
	body := &trackingReadCloser{Reader: strings.NewReader(strings.Repeat("x", 2<<20))}
	backend := &stubBackend{
		responses: []stubCall{
			{resp: responseWithStatus(http.StatusOK)},
			{resp: responseWithStatusAndBody(http.StatusOK, body)},
		},
	}
	d := &dispatcher{backend: backend}
	req, err := http.NewRequest(http.MethodPut, "http://proxy.local/bucket/key", strings.NewReader("payload"))
	if err != nil {
		t.Fatal(err)
	}
	req.ContentLength = int64(len("payload"))

	result, err := d.Dispatch(context.Background(), router.Match{
		Route:        config.Route{Dispatch: config.DispatchAll, DestinationRefs: []string{"primary", "replica"}},
		Destinations: []config.S3Target{{Name: "primary"}, {Name: "replica"}},
	}, req, s3ops.OpPutObject, rewrite.Result{Bucket: "bucket", Key: "key"})
	if err != nil {
		t.Fatalf("Dispatch error = %v", err)
	}
	if !body.closed {
		t.Fatal("expected oversized discarded body to be closed")
	}
	if got := body.Reader.(*strings.Reader).Len(); got == 0 {
		t.Fatal("expected discarded body drain to be bounded")
	}
	s3.DrainAndClose(result.Primary)
}

func TestDispatch_FanoutPrimaryFailurePreservesPrimaryError(t *testing.T) {
	backend := &stubBackend{
		responses: []stubCall{
			{resp: responseWithStatus(http.StatusForbidden)},
			{resp: responseWithStatus(http.StatusOK)},
		},
	}
	d := &dispatcher{backend: backend}
	req, err := http.NewRequest(http.MethodPut, "http://proxy.local/bucket/key", strings.NewReader("payload"))
	if err != nil {
		t.Fatal(err)
	}
	req.ContentLength = int64(len("payload"))

	result, err := d.Dispatch(context.Background(), router.Match{
		Route: config.Route{
			Dispatch:        config.DispatchAll,
			DestinationRefs: []string{"primary", "replica"},
		},
		Destinations: []config.S3Target{{Name: "primary"}, {Name: "replica"}},
	}, req, s3ops.OpPutObject, rewrite.Result{Bucket: "bucket", Key: "key"})
	if err == nil {
		t.Fatal("expected fan-out error")
	}
	if result == nil || result.Primary == nil {
		t.Fatalf("expected primary error response, got %#v", result)
	}
	if got, want := result.Primary.StatusCode, http.StatusForbidden; got != want {
		t.Fatalf("StatusCode = %d, want %d", got, want)
	}
	result.Primary.Body.Close()
}

func TestDispatch_OrderedFailoverRetriesTransportError(t *testing.T) {
	backend := &stubBackend{
		responses: []stubCall{
			{err: errors.New("dial tcp timeout")},
			{resp: responseWithStatus(http.StatusOK)},
		},
	}
	d := &dispatcher{backend: backend}
	req := mustRequest(t)

	result, err := d.Dispatch(context.Background(), orderedFailoverMatch(), req, s3ops.OpGetObject, rewrite.Result{Bucket: "bucket", Key: "key"})
	if err != nil {
		t.Fatalf("Dispatch error = %v", err)
	}
	if result.Primary == nil || result.Primary.StatusCode != http.StatusOK {
		t.Fatalf("Primary = %#v, want 200 response", result.Primary)
	}
	if got := backend.calls; got != 2 {
		t.Fatalf("calls = %d, want 2", got)
	}
	if _, ok := result.Errors["primary"]; !ok {
		t.Fatalf("expected error recorded for first destination, got %#v", result.Errors)
	}
	if got, want := len(result.Attempts), 2; got != want {
		t.Fatalf("attempts = %d, want %d", got, want)
	}
	if result.Attempts[0].Target != "primary" || result.Attempts[0].Error == nil {
		t.Fatalf("first attempt = %#v, want primary error", result.Attempts[0])
	}
	if result.Attempts[1].Target != "replica" || result.Attempts[1].StatusCode != http.StatusOK || result.Attempts[1].Error != nil {
		t.Fatalf("second attempt = %#v, want replica 200", result.Attempts[1])
	}
	result.Primary.Body.Close()
}

func TestDispatch_OrderedFailoverRetriesUpstream5xx(t *testing.T) {
	failedBody := &trackingReadCloser{Reader: strings.NewReader("bad gateway")}
	backend := &stubBackend{
		responses: []stubCall{
			{resp: responseWithStatusAndBody(http.StatusBadGateway, failedBody)},
			{resp: responseWithStatus(http.StatusOK)},
		},
	}
	d := &dispatcher{backend: backend}
	req := mustRequest(t)

	result, err := d.Dispatch(context.Background(), orderedFailoverMatch(), req, s3ops.OpGetObject, rewrite.Result{Bucket: "bucket", Key: "key"})
	if err != nil {
		t.Fatalf("Dispatch error = %v", err)
	}
	if result.Primary == nil || result.Primary.StatusCode != http.StatusOK {
		t.Fatalf("Primary = %#v, want 200 response", result.Primary)
	}
	if got := backend.calls; got != 2 {
		t.Fatalf("calls = %d, want 2", got)
	}
	if _, ok := result.Errors["primary"]; !ok {
		t.Fatalf("expected 5xx recorded for first destination, got %#v", result.Errors)
	}
	if got, want := len(result.Attempts), 2; got != want {
		t.Fatalf("attempts = %d, want %d", got, want)
	}
	if result.Attempts[0].Target != "primary" || result.Attempts[0].StatusCode != http.StatusBadGateway || result.Attempts[0].Error == nil {
		t.Fatalf("first attempt = %#v, want primary 502 error", result.Attempts[0])
	}
	if result.Attempts[1].Target != "replica" || result.Attempts[1].StatusCode != http.StatusOK || result.Attempts[1].Error != nil {
		t.Fatalf("second attempt = %#v, want replica 200", result.Attempts[1])
	}
	if !failedBody.closed {
		t.Fatal("expected failed failover response body to be closed before retry")
	}
	if got := failedBody.Reader.(*strings.Reader).Len(); got != 0 {
		t.Fatalf("failed body remaining = %d, want drained", got)
	}
	result.Primary.Body.Close()
}

func TestDispatch_DiscardedResponsesReuseConnections(t *testing.T) {
	var conns atomic.Int64
	server := httptest.NewUnstartedServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("small discarded body"))
	}))
	server.Config.ConnState = func(_ net.Conn, state http.ConnState) {
		if state == http.StateNew {
			conns.Add(1)
		}
	}
	server.Start()
	defer server.Close()

	backend := httpBackend{client: server.Client(), url: server.URL}
	d := &dispatcher{backend: backend}
	for i := 0; i < 3; i++ {
		req, err := http.NewRequest(http.MethodPut, "http://proxy.local/bucket/key", strings.NewReader("payload"))
		if err != nil {
			t.Fatal(err)
		}
		req.ContentLength = int64(len("payload"))
		result, err := d.Dispatch(context.Background(), router.Match{
			Route:        config.Route{Dispatch: config.DispatchAll, DestinationRefs: []string{"primary", "replica"}},
			Destinations: []config.S3Target{{Name: "primary"}, {Name: "replica"}},
		}, req, s3ops.OpPutObject, rewrite.Result{Bucket: "bucket", Key: "key"})
		if err != nil {
			t.Fatalf("Dispatch error = %v", err)
		}
		if _, err := io.Copy(io.Discard, result.Primary.Body); err != nil {
			t.Fatal(err)
		}
		result.Primary.Body.Close()
	}
	if got := conns.Load(); got > 2 {
		t.Fatalf("connections = %d, want discarded responses to reuse the initial connections", got)
	}
}

func TestDispatch_OrderedFailoverDoesNotRetry404(t *testing.T) {
	backend := &stubBackend{
		responses: []stubCall{{resp: responseWithStatus(http.StatusNotFound)}},
	}
	d := &dispatcher{backend: backend}
	req := mustRequest(t)

	result, err := d.Dispatch(context.Background(), orderedFailoverMatch(), req, s3ops.OpGetObject, rewrite.Result{Bucket: "bucket", Key: "key"})
	if err != nil {
		t.Fatalf("Dispatch error = %v", err)
	}
	if result.Primary == nil || result.Primary.StatusCode != http.StatusNotFound {
		t.Fatalf("Primary = %#v, want 404 response", result.Primary)
	}
	if got := backend.calls; got != 1 {
		t.Fatalf("calls = %d, want 1", got)
	}
	if len(result.Errors) != 0 {
		t.Fatalf("unexpected errors: %#v", result.Errors)
	}
	result.Primary.Body.Close()
}

func TestDispatch_OrderedFailoverReturnsLast5xxWhenExhausted(t *testing.T) {
	backend := &stubBackend{
		responses: []stubCall{
			{resp: responseWithStatus(http.StatusBadGateway)},
			{resp: responseWithStatus(http.StatusServiceUnavailable)},
		},
	}
	d := &dispatcher{backend: backend}
	req := mustRequest(t)

	result, err := d.Dispatch(context.Background(), orderedFailoverMatch(), req, s3ops.OpGetObject, rewrite.Result{Bucket: "bucket", Key: "key"})
	if err == nil {
		t.Fatal("expected ordered failover to report exhaustion")
	}
	if result == nil || result.Primary == nil {
		t.Fatalf("expected last upstream response, got %#v", result)
	}
	if got, want := result.Primary.StatusCode, http.StatusServiceUnavailable; got != want {
		t.Fatalf("StatusCode = %d, want %d", got, want)
	}
	if got := backend.calls; got != 2 {
		t.Fatalf("calls = %d, want 2", got)
	}
	result.Primary.Body.Close()
}

func TestDispatch_OrderedFailoverReportsTransportAttemptsWhenExhausted(t *testing.T) {
	backend := &stubBackend{
		responses: []stubCall{
			{err: errors.New("primary timeout")},
			{err: errors.New("replica timeout")},
		},
	}
	d := &dispatcher{backend: backend}
	req := mustRequest(t)

	result, err := d.Dispatch(context.Background(), orderedFailoverMatch(), req, s3ops.OpGetObject, rewrite.Result{Bucket: "bucket", Key: "key"})
	if err == nil {
		t.Fatal("expected ordered failover to report exhaustion")
	}
	if result == nil {
		t.Fatal("expected result with attempt metadata")
	}
	if got, want := len(result.Attempts), 2; got != want {
		t.Fatalf("attempts = %d, want %d", got, want)
	}
	for i, target := range []string{"primary", "replica"} {
		if result.Attempts[i].Target != target || result.Attempts[i].Error == nil {
			t.Fatalf("attempt %d = %#v, want %s error", i, result.Attempts[i], target)
		}
	}
}

func mustRequest(t *testing.T) *http.Request {
	t.Helper()
	req, err := http.NewRequest(http.MethodGet, "http://proxy.local/bucket/key", nil)
	if err != nil {
		t.Fatal(err)
	}
	return req
}

func orderedFailoverMatch() router.Match {
	return router.Match{
		Route: config.Route{ReadPreference: config.ReadOrderedFailover},
		Destinations: []config.S3Target{
			{Name: "primary"},
			{Name: "replica"},
		},
	}
}

func responseWithStatus(status int) *s3.Response {
	return responseWithStatusAndBody(status, io.NopCloser(strings.NewReader("body")))
}

func responseWithStatusAndBody(status int, body io.ReadCloser) *s3.Response {
	return &s3.Response{
		StatusCode: status,
		Header:     http.Header{},
		Body:       body,
	}
}

type trackingReadCloser struct {
	io.Reader
	closed bool
}

func (r *trackingReadCloser) Close() error {
	r.closed = true
	return nil
}

type httpBackend struct {
	client *http.Client
	url    string
}

func (b httpBackend) Do(ctx context.Context, req s3.Request) (*s3.Response, error) {
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodGet, b.url, nil)
	if err != nil {
		return nil, err
	}
	resp, err := b.client.Do(httpReq)
	if err != nil {
		return nil, err
	}
	return &s3.Response{StatusCode: resp.StatusCode, Header: resp.Header, Body: resp.Body}, nil
}

type stubBackend struct {
	responses []stubCall
	calls     int
}

type stubCall struct {
	resp *s3.Response
	err  error
}

func (s *stubBackend) Do(context.Context, s3.Request) (*s3.Response, error) {
	if s.calls >= len(s.responses) {
		return nil, errors.New("unexpected backend call")
	}
	call := s.responses[s.calls]
	s.calls++
	return call.resp, call.err
}
