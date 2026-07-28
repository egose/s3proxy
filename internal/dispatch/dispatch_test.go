package dispatch

import (
	"context"
	"errors"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/egose/s3proxy/internal/backend/s3"
	"github.com/egose/s3proxy/internal/config"
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

func TestDispatch_FanoutReplicaFailureClearsSuccessfulPrimary(t *testing.T) {
	backend := &stubBackend{
		responses: []stubCall{
			{resp: responseWithStatus(http.StatusOK)},
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
	result.Primary.Body.Close()
}

func TestDispatch_OrderedFailoverRetriesUpstream5xx(t *testing.T) {
	backend := &stubBackend{
		responses: []stubCall{
			{resp: responseWithStatus(http.StatusBadGateway)},
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
	result.Primary.Body.Close()
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
	return &s3.Response{
		StatusCode: status,
		Header:     http.Header{},
		Body:       io.NopCloser(strings.NewReader("body")),
	}
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
