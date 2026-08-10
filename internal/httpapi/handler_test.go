package httpapi

import (
	"bytes"
	"context"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/egose/s3proxy/internal/auth"
	"github.com/egose/s3proxy/internal/backend/s3"
	"github.com/egose/s3proxy/internal/config"
	"github.com/egose/s3proxy/internal/dispatch"
	"github.com/egose/s3proxy/internal/listbuckets"
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

func TestHandler_RootUnsupportedMethodsReturnNotImplemented(t *testing.T) {
	for _, method := range []string{http.MethodPost, http.MethodPut, http.MethodDelete} {
		t.Run(method, func(t *testing.T) {
			h := NewHandler(Dependencies{Addressing: config.Addressing{PathStyle: true}})
			req := httptest.NewRequest(method, "/", nil)
			rr := httptest.NewRecorder()

			h.ServeHTTP(rr, req)

			if rr.Code != http.StatusNotImplemented {
				t.Fatalf("status = %d, want %d", rr.Code, http.StatusNotImplemented)
			}
			if !strings.Contains(rr.Body.String(), "NotImplemented") {
				t.Fatalf("body = %q, want NotImplemented", rr.Body.String())
			}
		})
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

func TestHandler_ContinueRouteHTTPFailureDoesNotReturnEarlierSuccess(t *testing.T) {
	firstBody := &trackingReadCloser{Reader: strings.NewReader("ok")}
	dispatcher := &stubFanout{
		results: []*dispatch.Result{
			{Primary: &s3.Response{StatusCode: http.StatusOK, Header: http.Header{}, Body: firstBody}},
			{Primary: &s3.Response{StatusCode: http.StatusForbidden, Header: http.Header{}, Body: io.NopCloser(strings.NewReader("denied"))}},
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

	if rr.Code != http.StatusForbidden {
		t.Fatalf("status = %d, want %d", rr.Code, http.StatusForbidden)
	}
	if got, want := rr.Body.String(), "denied"; got != want {
		t.Fatalf("body = %q, want %q", got, want)
	}
	if !firstBody.closed {
		t.Fatal("expected earlier retained response body to be closed")
	}
}

func TestHandler_LaterDispatchFailureClosesEarlierSuccess(t *testing.T) {
	firstBody := &trackingReadCloser{Reader: strings.NewReader("ok")}
	dispatcher := &stubFanout{
		results: []*dispatch.Result{
			{Primary: &s3.Response{StatusCode: http.StatusOK, Header: http.Header{}, Body: firstBody}},
			nil,
		},
		errs: []error{nil, context.DeadlineExceeded},
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

	if rr.Code != http.StatusBadGateway {
		t.Fatalf("status = %d, want %d", rr.Code, http.StatusBadGateway)
	}
	if !firstBody.closed {
		t.Fatal("expected earlier retained response body to be closed")
	}
}

func TestHandler_LaterResetFailureClosesEarlierSuccess(t *testing.T) {
	firstBody := &trackingReadCloser{Reader: strings.NewReader("ok")}
	getBodyCalls := 0
	dispatcher := &stubFanout{
		results: []*dispatch.Result{{Primary: &s3.Response{StatusCode: http.StatusOK, Header: http.Header{}, Body: firstBody}}},
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
	req.GetBody = func() (io.ReadCloser, error) {
		getBodyCalls++
		if getBodyCalls == 2 {
			return nil, errors.New("reset failed")
		}
		return io.NopCloser(strings.NewReader("payload")), nil
	}
	rr := httptest.NewRecorder()

	h.ServeHTTP(rr, req)

	if rr.Code != http.StatusBadGateway {
		t.Fatalf("status = %d, want %d", rr.Code, http.StatusBadGateway)
	}
	if !firstBody.closed {
		t.Fatal("expected earlier retained response body to be closed")
	}
}

func TestHandler_LaterRewriteFailureClosesEarlierSuccess(t *testing.T) {
	firstBody := &trackingReadCloser{Reader: strings.NewReader("ok")}
	dispatcher := &stubFanout{
		results: []*dispatch.Result{{Primary: &s3.Response{StatusCode: http.StatusOK, Header: http.Header{}, Body: firstBody}}},
	}
	h := NewHandler(Dependencies{
		Addressing:    config.Addressing{PathStyle: true},
		Authenticator: stubAuthenticator{},
		Authorizer:    stubAuthorizer{},
		Router:        stubResolver{matches: []router.Match{{Route: config.Route{Name: "one"}}, {Route: config.Route{Name: "two"}}}},
		Rewriter:      &sequenceRewriter{errAt: 1, err: errors.New("rewrite failed")},
		Dispatcher:    dispatcher,
	})
	req := httptest.NewRequest(http.MethodPut, "/bucket/key", strings.NewReader("payload"))
	req.ContentLength = int64(len("payload"))
	rr := httptest.NewRecorder()

	h.ServeHTTP(rr, req)

	if rr.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, want %d", rr.Code, http.StatusInternalServerError)
	}
	if !firstBody.closed {
		t.Fatal("expected earlier retained response body to be closed")
	}
}

func TestHandler_ListBucketsRequiresAllowedOperation(t *testing.T) {
	h := NewHandler(Dependencies{
		Addressing:    config.Addressing{PathStyle: true},
		Authenticator: stubAuthenticator{principal: &auth.Principal{AllowOps: []string{string(s3ops.OpGetObject)}}},
		Authorizer:    stubAuthorizer{},
	})
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rr := httptest.NewRecorder()

	h.ServeHTTP(rr, req)

	if rr.Code != http.StatusForbidden {
		t.Fatalf("status = %d, want %d", rr.Code, http.StatusForbidden)
	}
	if !strings.Contains(rr.Body.String(), "AccessDenied") {
		t.Fatalf("body = %q, want AccessDenied", rr.Body.String())
	}
}

func TestHandler_ListBucketsAllowsWildcardAndDefaultOperations(t *testing.T) {
	tests := []struct {
		name      string
		principal *auth.Principal
	}{
		{name: "wildcard", principal: &auth.Principal{AllowOps: []string{"*"}}},
		{name: "default", principal: &auth.Principal{}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := NewHandler(Dependencies{
				Addressing:    config.Addressing{PathStyle: true},
				Authenticator: stubAuthenticator{principal: tt.principal},
				Authorizer:    stubAuthorizer{},
				Buckets:       stubBuckets{},
			})
			req := httptest.NewRequest(http.MethodGet, "/", nil)
			rr := httptest.NewRecorder()

			h.ServeHTTP(rr, req)

			if rr.Code != http.StatusOK {
				t.Fatalf("status = %d, want %d", rr.Code, http.StatusOK)
			}
		})
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

func TestHandler_MultiRouteBudgetExhaustionReturnsSlowDown(t *testing.T) {
	h := NewHandler(Dependencies{
		Addressing:    config.Addressing{PathStyle: true},
		ReplayBudget:  replaybody.NewBudget(10, 3),
		Authenticator: stubAuthenticator{},
		Authorizer:    stubAuthorizer{},
		Router:        stubResolver{matches: []router.Match{{Route: config.Route{Name: "one"}}, {Route: config.Route{Name: "two"}}}},
		Rewriter:      stubRewriter{},
		Dispatcher:    &stubFanout{},
	})
	req := httptest.NewRequest(http.MethodPut, "/bucket/key", strings.NewReader("payload"))
	rr := httptest.NewRecorder()

	h.ServeHTTP(rr, req)

	if rr.Code != http.StatusServiceUnavailable {
		t.Fatalf("status = %d, want %d", rr.Code, http.StatusServiceUnavailable)
	}
	if !strings.Contains(rr.Body.String(), "SlowDown") {
		t.Fatalf("body = %q, want SlowDown", rr.Body.String())
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

func TestHandler_LogsAndClosesResponseBodyCopyError(t *testing.T) {
	sentinel := errors.New("stream failed")
	body := &errAfterDataReadCloser{data: []byte("partial"), err: sentinel}
	var logs bytes.Buffer
	logger := slog.New(slog.NewTextHandler(&logs, nil))
	dispatcher := &stubFanout{
		results: []*dispatch.Result{{Primary: &s3.Response{StatusCode: http.StatusOK, Header: http.Header{}, Body: body}}},
	}
	h := NewHandler(Dependencies{
		Addressing:    config.Addressing{PathStyle: true},
		Authenticator: stubAuthenticator{},
		Authorizer:    stubAuthorizer{},
		Router:        stubResolver{matches: []router.Match{{Route: config.Route{Name: "one"}}}},
		Rewriter:      stubRewriter{},
		Dispatcher:    dispatcher,
		Logger:        logger,
	})
	req := httptest.NewRequest(http.MethodGet, "/bucket/key", nil)
	rr := httptest.NewRecorder()

	h.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rr.Code, http.StatusOK)
	}
	if got, want := rr.Body.String(), "partial"; got != want {
		t.Fatalf("body = %q, want %q", got, want)
	}
	if !body.closed {
		t.Fatal("expected response body to be closed")
	}
	if !strings.Contains(logs.String(), "response copy failed") || !strings.Contains(logs.String(), sentinel.Error()) {
		t.Fatalf("logs = %q, want copy error", logs.String())
	}
}

func TestHandler_CompletionLogs(t *testing.T) {
	tests := []struct {
		name       string
		req        *http.Request
		deps       Dependencies
		wantStatus int
		wantLog    []string
	}{
		{
			name:       "health",
			req:        httptest.NewRequest(http.MethodGet, "/healthz", nil),
			deps:       Dependencies{},
			wantStatus: http.StatusOK,
			wantLog:    []string{"request complete", "status=200", "bytes=2"},
		},
		{
			name: "auth denial",
			req:  httptest.NewRequest(http.MethodGet, "/bucket/key", nil),
			deps: Dependencies{
				Addressing:    config.Addressing{PathStyle: true},
				Authenticator: stubAuthenticator{err: errors.New("no credentials")},
			},
			wantStatus: http.StatusForbidden,
			wantLog:    []string{"request complete", "status=403", "auth failed"},
		},
		{
			name: "route miss",
			req:  httptest.NewRequest(http.MethodGet, "/bucket/key", nil),
			deps: Dependencies{
				Addressing:    config.Addressing{PathStyle: true},
				Authenticator: stubAuthenticator{},
				Router:        stubResolver{err: errors.New("no route")},
			},
			wantStatus: http.StatusNotFound,
			wantLog:    []string{"request complete", "status=404", "no route"},
		},
		{
			name: "upstream success",
			req:  httptest.NewRequest(http.MethodGet, "/bucket/key", nil),
			deps: Dependencies{
				Addressing:    config.Addressing{PathStyle: true},
				Authenticator: stubAuthenticator{},
				Authorizer:    stubAuthorizer{},
				Router:        stubResolver{matches: []router.Match{{Route: config.Route{Name: "one"}}}},
				Rewriter:      stubRewriter{},
				Dispatcher:    &stubFanout{results: []*dispatch.Result{{Primary: &s3.Response{StatusCode: http.StatusOK, Header: http.Header{}, Body: io.NopCloser(strings.NewReader("ok"))}}}},
			},
			wantStatus: http.StatusOK,
			wantLog:    []string{"request complete", "status=200", "bytes=2"},
		},
		{
			name: "upstream failure",
			req:  httptest.NewRequest(http.MethodGet, "/bucket/key", nil),
			deps: Dependencies{
				Addressing:    config.Addressing{PathStyle: true},
				Authenticator: stubAuthenticator{},
				Authorizer:    stubAuthorizer{},
				Router:        stubResolver{matches: []router.Match{{Route: config.Route{Name: "one"}}}},
				Rewriter:      stubRewriter{},
				Dispatcher:    &stubFanout{errs: []error{errors.New("upstream unavailable")}, results: []*dispatch.Result{nil}},
			},
			wantStatus: http.StatusBadGateway,
			wantLog:    []string{"request complete", "status=502", "dispatch failed"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var logs bytes.Buffer
			tt.deps.Logger = slog.New(slog.NewTextHandler(&logs, nil))
			h := NewHandler(tt.deps)
			rr := httptest.NewRecorder()

			h.ServeHTTP(rr, tt.req)

			if rr.Code != tt.wantStatus {
				t.Fatalf("status = %d, want %d", rr.Code, tt.wantStatus)
			}
			for _, want := range tt.wantLog {
				if !strings.Contains(logs.String(), want) {
					t.Fatalf("logs = %q, want %q", logs.String(), want)
				}
			}
		})
	}
}

func TestHandler_LogsDestinationAttempts(t *testing.T) {
	var logs bytes.Buffer
	logger := slog.New(slog.NewTextHandler(&logs, nil))
	h := NewHandler(Dependencies{
		Addressing:    config.Addressing{PathStyle: true},
		Authenticator: stubAuthenticator{},
		Authorizer:    stubAuthorizer{},
		Router:        stubResolver{matches: []router.Match{{Route: config.Route{Name: "objects"}}}},
		Rewriter:      stubRewriter{},
		Dispatcher: &stubFanout{results: []*dispatch.Result{{
			Primary: &s3.Response{StatusCode: http.StatusOK, Header: http.Header{}, Body: io.NopCloser(strings.NewReader("ok"))},
			Attempts: []dispatch.Attempt{
				{Target: "primary", Error: errors.New("dial tcp timeout")},
				{Target: "replica", StatusCode: http.StatusOK},
			},
		}}},
		Logger: logger,
	})
	req := httptest.NewRequest(http.MethodGet, "/bucket/key", nil)
	req.Header.Set("X-Request-Id", "req-1")
	rr := httptest.NewRecorder()

	h.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rr.Code, http.StatusOK)
	}
	for _, want := range []string{
		"destination attempt failed",
		"destination attempt succeeded",
		"request_id=req-1",
		"route=objects",
		"operation=GetObject",
		"target=primary",
		"target=replica",
		"status=200",
		"dial tcp timeout",
	} {
		if !strings.Contains(logs.String(), want) {
			t.Fatalf("logs = %q, want %q", logs.String(), want)
		}
	}
}

type stubAuthenticator struct {
	principal *auth.Principal
	err       error
}

func (s stubAuthenticator) Authenticate(*http.Request) (*auth.Principal, error) {
	return s.principal, s.err
}

type stubAuthorizer struct{}

func (stubAuthorizer) AllowOperation(p *auth.Principal, op s3ops.Operation) bool {
	if p == nil || len(p.AllowOps) == 0 {
		return true
	}
	for _, allow := range p.AllowOps {
		if allow == "*" || allow == string(op) {
			return true
		}
	}
	return false
}

func (stubAuthorizer) AllowRoute(*auth.Principal, string, s3ops.Operation) bool { return true }

type stubResolver struct {
	matches []router.Match
	err     error
}

func (s stubResolver) Resolve(*requestctx.Context, s3ops.Operation) ([]router.Match, error) {
	if s.err != nil {
		return nil, s.err
	}
	return s.matches, nil
}

type stubRewriter struct{}

func (stubRewriter) Apply(*requestctx.Context, config.RewriteRule, map[string]string) (rewrite.Result, error) {
	return rewrite.Result{Bucket: "bucket", Key: "key"}, nil
}

type sequenceRewriter struct {
	calls int
	errAt int
	err   error
}

func (s *sequenceRewriter) Apply(*requestctx.Context, config.RewriteRule, map[string]string) (rewrite.Result, error) {
	if s.calls == s.errAt {
		s.calls++
		return rewrite.Result{}, s.err
	}
	s.calls++
	return rewrite.Result{Bucket: "bucket", Key: "key"}, nil
}

type stubBuckets struct{}

func (stubBuckets) List(*auth.Principal) []listbuckets.BucketView { return nil }

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

type errAfterDataReadCloser struct {
	data   []byte
	err    error
	closed bool
}

type trackingReadCloser struct {
	io.Reader
	closed bool
}

func (r *trackingReadCloser) Close() error {
	r.closed = true
	return nil
}

func (r *errAfterDataReadCloser) Read(p []byte) (int, error) {
	if len(r.data) == 0 {
		return 0, r.err
	}
	n := copy(p, r.data)
	r.data = r.data[n:]
	if len(r.data) == 0 {
		return n, r.err
	}
	return n, nil
}

func (r *errAfterDataReadCloser) Close() error {
	r.closed = true
	return nil
}
