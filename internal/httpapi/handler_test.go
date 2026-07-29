package httpapi

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/egose/s3proxy/internal/auth"
	"github.com/egose/s3proxy/internal/backend/s3"
	"github.com/egose/s3proxy/internal/config"
	"github.com/egose/s3proxy/internal/dispatch"
	"github.com/egose/s3proxy/internal/replaybody"
	"github.com/egose/s3proxy/internal/requestctx"
	"github.com/egose/s3proxy/internal/rewrite"
	"github.com/egose/s3proxy/internal/router"
	"github.com/egose/s3proxy/internal/s3ops"
)

func TestHandler_ListObjectsV1ReturnsNotImplemented(t *testing.T) {
	h := NewHandler(Dependencies{
		Addressing: config.Addressing{PathStyle: true},
	})
	req := httptest.NewRequest(http.MethodGet, "/bucket?prefix=logs/", nil)
	rr := httptest.NewRecorder()

	h.ServeHTTP(rr, req)

	if rr.Code != http.StatusNotImplemented {
		t.Fatalf("status = %d, want %d", rr.Code, http.StatusNotImplemented)
	}
}

func TestHandler_CopyObjectReturnsNotImplemented(t *testing.T) {
	h := NewHandler(Dependencies{
		Addressing: config.Addressing{PathStyle: true},
	})
	req := httptest.NewRequest(http.MethodPut, "/bucket/dest", nil)
	req.Header.Set("x-amz-copy-source", "/bucket/src")
	rr := httptest.NewRecorder()

	h.ServeHTTP(rr, req)

	if rr.Code != http.StatusNotImplemented {
		t.Fatalf("status = %d, want %d", rr.Code, http.StatusNotImplemented)
	}
}

func TestHandler_ContinueDispatchesAllWriteMatches(t *testing.T) {
	dispatcher := &stubFanout{
		results: []*dispatch.Result{
			{Primary: &s3.Response{StatusCode: http.StatusOK, Header: http.Header{}, Body: io.NopCloser(strings.NewReader("ok"))}},
			{Primary: &s3.Response{StatusCode: http.StatusOK, Header: http.Header{}, Body: io.NopCloser(strings.NewReader("ignored"))}},
		},
	}
	h := NewHandler(Dependencies{
		Addressing:    config.Addressing{PathStyle: true},
		Authenticator: stubAuthenticator{},
		Authorizer:    stubAuthorizer{},
		Router:        stubResolver{matches: []router.Match{{Route: config.Route{Name: "one"}}, {Route: config.Route{Name: "two"}}}},
		Rewriter:      stubRewriter{},
		Dispatcher:    dispatcher,
	})
	req := httptest.NewRequest(http.MethodPut, "/bucket/key", strings.NewReader("payload"))
	req.ContentLength = int64(len("payload"))
	rr := httptest.NewRecorder()

	h.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rr.Code, http.StatusOK)
	}
	if got, want := len(dispatcher.matches), 2; got != want {
		t.Fatalf("dispatch calls = %d, want %d", got, want)
	}
}

func TestHandler_DispatchErrorWithSuccessfulPrimaryReturnsBadGateway(t *testing.T) {
	dispatcher := &stubFanout{
		results: []*dispatch.Result{{Primary: &s3.Response{StatusCode: http.StatusOK, Header: http.Header{}, Body: io.NopCloser(strings.NewReader("ok"))}}},
		errs:    []error{context.DeadlineExceeded},
	}
	h := NewHandler(Dependencies{
		Addressing:    config.Addressing{PathStyle: true},
		Authenticator: stubAuthenticator{},
		Authorizer:    stubAuthorizer{},
		Router:        stubResolver{matches: []router.Match{{Route: config.Route{Name: "one"}}}},
		Rewriter:      stubRewriter{},
		Dispatcher:    dispatcher,
	})
	req := httptest.NewRequest(http.MethodPut, "/bucket/key", strings.NewReader("payload"))
	req.ContentLength = int64(len("payload"))
	rr := httptest.NewRecorder()

	h.ServeHTTP(rr, req)

	if rr.Code != http.StatusBadGateway {
		t.Fatalf("status = %d, want %d", rr.Code, http.StatusBadGateway)
	}
}

func TestHandler_ReturnsInvalidRequestWhenAddressingModeDoesNotMatch(t *testing.T) {
	h := NewHandler(Dependencies{
		Addressing: config.Addressing{
			VirtualHosted: true,
			HostSuffixes:  []string{"s3proxy.example.com"},
		},
	})
	req := httptest.NewRequest(http.MethodGet, "/bucket/key", nil)
	req.Host = "localhost:8080"
	rr := httptest.NewRecorder()

	h.ServeHTTP(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", rr.Code, http.StatusBadRequest)
	}
	if !strings.Contains(rr.Body.String(), "InvalidRequest") {
		t.Fatalf("body = %q, want InvalidRequest", rr.Body.String())
	}
}

func TestHandler_SignatureMismatchReturnsSignatureDoesNotMatch(t *testing.T) {
	h := NewHandler(Dependencies{
		Addressing:    config.Addressing{PathStyle: true},
		Authenticator: stubAuthenticator{err: auth.AuthError("signature does not match")},
	})
	req := httptest.NewRequest(http.MethodGet, "/bucket/key", nil)
	rr := httptest.NewRecorder()

	h.ServeHTTP(rr, req)

	if rr.Code != http.StatusForbidden {
		t.Fatalf("status = %d, want %d", rr.Code, http.StatusForbidden)
	}
	if !strings.Contains(rr.Body.String(), "SignatureDoesNotMatch") {
		t.Fatalf("body = %q, want SignatureDoesNotMatch", rr.Body.String())
	}
}

func TestHandler_MultiRouteOversizedBodyReturnsEntityTooLarge(t *testing.T) {
	h := NewHandler(Dependencies{
		Addressing:         config.Addressing{PathStyle: true},
		ReplayBodyMaxBytes: 1,
		Authenticator:      stubAuthenticator{},
		Authorizer:         stubAuthorizer{},
		Router:             stubResolver{matches: []router.Match{{Route: config.Route{Name: "one"}}, {Route: config.Route{Name: "two"}}}},
		Rewriter:           stubRewriter{},
		Dispatcher:         &stubFanout{},
	})
	req := httptest.NewRequest(http.MethodPut, "/bucket/key", strings.NewReader("x"))
	req.ContentLength = 2
	rr := httptest.NewRecorder()

	h.ServeHTTP(rr, req)

	if rr.Code != http.StatusRequestEntityTooLarge {
		t.Fatalf("status = %d, want %d", rr.Code, http.StatusRequestEntityTooLarge)
	}
	if !strings.Contains(rr.Body.String(), "EntityTooLarge") {
		t.Fatalf("body = %q, want EntityTooLarge", rr.Body.String())
	}
}

func TestHandler_DispatchOversizedBodyReturnsEntityTooLarge(t *testing.T) {
	h := NewHandler(Dependencies{
		Addressing:    config.Addressing{PathStyle: true},
		Authenticator: stubAuthenticator{},
		Authorizer:    stubAuthorizer{},
		Router:        stubResolver{matches: []router.Match{{Route: config.Route{Name: "one"}}}},
		Rewriter:      stubRewriter{},
		Dispatcher:    &stubFanout{errs: []error{replaybody.ErrBodyTooLarge}, results: []*dispatch.Result{nil}},
	})
	req := httptest.NewRequest(http.MethodPut, "/bucket/key", strings.NewReader("payload"))
	req.ContentLength = int64(len("payload"))
	rr := httptest.NewRecorder()

	h.ServeHTTP(rr, req)

	if rr.Code != http.StatusRequestEntityTooLarge {
		t.Fatalf("status = %d, want %d", rr.Code, http.StatusRequestEntityTooLarge)
	}
	if !strings.Contains(rr.Body.String(), "EntityTooLarge") {
		t.Fatalf("body = %q, want EntityTooLarge", rr.Body.String())
	}
}

func TestWriteS3Response_StripsHopByHopHeaders(t *testing.T) {
	rr := httptest.NewRecorder()
	writeS3Response(rr, &s3.Response{
		StatusCode: http.StatusOK,
		Header: http.Header{
			"Connection":   []string{"X-Internal"},
			"X-Internal":   []string{"secret"},
			"Trailer":      []string{"X-Trailer"},
			"X-End-To-End": []string{"ok"},
		},
		Body: io.NopCloser(strings.NewReader("ok")),
	})

	resp := rr.Result()
	if got := resp.Header.Get("Connection"); got != "" {
		t.Fatalf("expected Connection to be stripped, got %q", got)
	}
	if got := resp.Header.Get("X-Internal"); got != "" {
		t.Fatalf("expected X-Internal to be stripped, got %q", got)
	}
	if got := resp.Header.Get("Trailer"); got != "" {
		t.Fatalf("expected Trailer to be stripped, got %q", got)
	}
	if got := resp.Header.Get("X-End-To-End"); got != "ok" {
		t.Fatalf("expected X-End-To-End to survive, got %q", got)
	}
}

type stubAuthenticator struct{ err error }

func (s stubAuthenticator) Authenticate(*http.Request) (*auth.Principal, error) { return nil, s.err }

type stubAuthorizer struct{}

func (stubAuthorizer) AllowRoute(*auth.Principal, string, s3ops.Operation) bool { return true }

func (stubAuthorizer) AllowBucketVisibility(*auth.Principal, string) bool { return true }

type stubResolver struct{ matches []router.Match }

func (s stubResolver) Resolve(*requestctx.Context, s3ops.Operation) ([]router.Match, error) {
	return s.matches, nil
}

type stubRewriter struct{}

func (stubRewriter) Apply(*requestctx.Context, config.Route, map[string]string) (rewrite.Result, error) {
	return rewrite.Result{Bucket: "bucket", Key: "key"}, nil
}

type stubFanout struct {
	matches []router.Match
	results []*dispatch.Result
	errs    []error
	index   int
}

func (s *stubFanout) Dispatch(_ context.Context, match router.Match, _ *http.Request, _ s3ops.Operation, _ rewrite.Result) (*dispatch.Result, error) {
	s.matches = append(s.matches, match)
	result := s.results[s.index]
	var err error
	if s.index < len(s.errs) {
		err = s.errs[s.index]
	}
	s.index++
	return result, err
}
