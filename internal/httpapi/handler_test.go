package httpapi

import (
	"bytes"
	"context"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"net/url"
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

func TestHandler_UnsupportedQueryOperationsDoNotDispatch(t *testing.T) {
	tests := []struct {
		name    string
		method  string
		path    string
		headers http.Header
	}{
		{name: "get acl", method: http.MethodGet, path: "/bucket/key?acl="},
		{name: "put tagging", method: http.MethodPut, path: "/bucket/key?tagging="},
		{name: "delete tagging", method: http.MethodDelete, path: "/bucket/key?tagging="},
		{name: "retention", method: http.MethodGet, path: "/bucket/key?retention="},
		{name: "legal hold", method: http.MethodPut, path: "/bucket/key?legal-hold="},
		{name: "torrent", method: http.MethodGet, path: "/bucket/key?torrent="},
		{name: "version id", method: http.MethodGet, path: "/bucket/key?versionId=1"},
		{name: "restore", method: http.MethodPost, path: "/bucket/key?restore="},
		{name: "select", method: http.MethodPost, path: "/bucket/key?select=&select-type=2"},
		{name: "response override", method: http.MethodGet, path: "/bucket/key?response-content-type=text%2Fplain"},
		{name: "multipart create", method: http.MethodPost, path: "/bucket/key?uploads="},
		{name: "multipart upload part", method: http.MethodPut, path: "/bucket/key?partNumber=1&uploadId=abc"},
		{name: "multipart complete", method: http.MethodPost, path: "/bucket/key?uploadId=abc"},
		{name: "copy object", method: http.MethodPut, path: "/bucket/key", headers: http.Header{"X-Amz-Copy-Source": []string{"/bucket/src"}}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dispatcher := &countingDispatcher{}
			h := NewHandler(Dependencies{
				Addressing:    config.Addressing{PathStyle: true},
				Authenticator: stubAuthenticator{},
				Authorizer:    stubAuthorizer{},
				Router:        stubResolver{matches: []router.Match{{Route: config.Route{Name: "one"}}}},
				Rewriter:      stubRewriter{},
				Dispatcher:    dispatcher,
			})
			req := httptest.NewRequest(tt.method, tt.path, strings.NewReader("payload"))
			for key, vals := range tt.headers {
				for _, val := range vals {
					req.Header.Add(key, val)
				}
			}
			rr := httptest.NewRecorder()

			h.ServeHTTP(rr, req)

			if rr.Code != http.StatusNotImplemented {
				t.Fatalf("status = %d, want %d", rr.Code, http.StatusNotImplemented)
			}
			if dispatcher.calls != 0 {
				t.Fatalf("dispatch calls = %d, want 0", dispatcher.calls)
			}
		})
	}
}

func TestHandler_SupportedQueryOperationsDispatch(t *testing.T) {
	tests := []struct {
		name   string
		method string
		path   string
		wantOp s3ops.Operation
	}{
		{name: "get object", method: http.MethodGet, path: "/bucket/key?x-id=GetObject", wantOp: s3ops.OpGetObject},
		{name: "head object", method: http.MethodHead, path: "/bucket/key?x-id=HeadObject", wantOp: s3ops.OpHeadObject},
		{name: "put object", method: http.MethodPut, path: "/bucket/key?x-id=PutObject", wantOp: s3ops.OpPutObject},
		{name: "delete object", method: http.MethodDelete, path: "/bucket/key?x-id=DeleteObject", wantOp: s3ops.OpDeleteObject},
		{name: "list objects v2", method: http.MethodGet, path: "/bucket?list-type=2&prefix=logs%2F&delimiter=%2F&max-keys=10&x-id=ListObjectsV2", wantOp: s3ops.OpListObjectsV2},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dispatcher := &countingDispatcher{response: &s3.Response{StatusCode: http.StatusOK, Header: http.Header{}, Body: io.NopCloser(strings.NewReader("ok"))}}
			h := NewHandler(Dependencies{
				Addressing:    config.Addressing{PathStyle: true},
				Authenticator: stubAuthenticator{},
				Authorizer:    stubAuthorizer{},
				Router:        stubResolver{matches: []router.Match{{Route: config.Route{Name: "one"}}}},
				Rewriter:      stubRewriter{},
				Dispatcher:    dispatcher,
			})
			req := httptest.NewRequest(tt.method, tt.path, strings.NewReader("payload"))
			rr := httptest.NewRecorder()

			h.ServeHTTP(rr, req)

			if rr.Code != http.StatusOK {
				t.Fatalf("status = %d, want %d", rr.Code, http.StatusOK)
			}
			if dispatcher.calls != 1 {
				t.Fatalf("dispatch calls = %d, want 1", dispatcher.calls)
			}
			if dispatcher.op != tt.wantOp {
				t.Fatalf("operation = %s, want %s", dispatcher.op, tt.wantOp)
			}
		})
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
		ReplayBudget:  replaybody.NewBudget(replaybody.DefaultMaxBytes, replaybody.DefaultAggregateMaxBytes),
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
		ReplayBudget:  replaybody.NewBudget(replaybody.DefaultMaxBytes, replaybody.DefaultAggregateMaxBytes),
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
		ReplayBudget:  replaybody.NewBudget(replaybody.DefaultMaxBytes, replaybody.DefaultAggregateMaxBytes),
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
		ReplayBudget:  replaybody.NewBudget(replaybody.DefaultMaxBytes, replaybody.DefaultAggregateMaxBytes),
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
		ReplayBudget:  replaybody.NewBudget(replaybody.DefaultMaxBytes, replaybody.DefaultAggregateMaxBytes),
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

func TestHandler_DispatchErrorWithPrimaryErrorWritesAndClosesBody(t *testing.T) {
	body := &trackingReadCloser{Reader: strings.NewReader("<Error>denied</Error>")}
	dispatcher := &stubFanout{
		results: []*dispatch.Result{{Primary: &s3.Response{StatusCode: http.StatusForbidden, Header: http.Header{}, Body: body}}},
		errs:    []error{errors.New("fan-out had 1 failures")},
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

	if rr.Code != http.StatusForbidden {
		t.Fatalf("status = %d, want %d", rr.Code, http.StatusForbidden)
	}
	if got, want := rr.Body.String(), "<Error>denied</Error>"; got != want {
		t.Fatalf("body = %q, want %q", got, want)
	}
	if !body.closed {
		t.Fatal("expected primary error body to be closed")
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

func TestHandler_PresignedUnsupportedQueryAuthenticatesBeforeRejectingOperation(t *testing.T) {
	dispatcher := &countingDispatcher{}
	h := NewHandler(Dependencies{
		Addressing:    config.Addressing{PathStyle: true},
		Authenticator: stubAuthenticator{err: auth.AuthError("signature does not match")},
		Authorizer:    stubAuthorizer{},
		Router:        stubResolver{matches: []router.Match{{Route: config.Route{Name: "one"}}}},
		Rewriter:      stubRewriter{},
		Dispatcher:    dispatcher,
	})
	req := httptest.NewRequest(http.MethodGet, "/bucket/key?response-content-type=text%2Fplain&X-Amz-Algorithm=AWS4-HMAC-SHA256&X-Amz-Credential=ak%2F20260823%2Fus-east-1%2Fs3%2Faws4_request&X-Amz-Date=20260823T000000Z&X-Amz-Expires=300&X-Amz-SignedHeaders=host&X-Amz-Signature=abcdef", nil)
	rr := httptest.NewRecorder()

	h.ServeHTTP(rr, req)

	if rr.Code != http.StatusForbidden {
		t.Fatalf("status = %d, want %d", rr.Code, http.StatusForbidden)
	}
	if !strings.Contains(rr.Body.String(), "SignatureDoesNotMatch") {
		t.Fatalf("body = %q, want SignatureDoesNotMatch", rr.Body.String())
	}
	if dispatcher.calls != 0 {
		t.Fatalf("dispatch calls = %d, want 0", dispatcher.calls)
	}
}

func TestHandler_AuthReplayErrorsUseReplayStatusMapping(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		status   int
		codeText string
	}{
		{name: "per request limit", err: replaybody.ErrBodyTooLarge, status: http.StatusRequestEntityTooLarge, codeText: "EntityTooLarge"},
		{name: "aggregate budget", err: replaybody.ErrBudgetExhausted, status: http.StatusServiceUnavailable, codeText: "SlowDown"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := NewHandler(Dependencies{
				Addressing:    config.Addressing{PathStyle: true},
				Authenticator: stubAuthenticator{err: tt.err},
			})
			req := httptest.NewRequest(http.MethodPut, "/bucket/key", strings.NewReader("payload"))
			rr := httptest.NewRecorder()

			h.ServeHTTP(rr, req)

			if rr.Code != tt.status {
				t.Fatalf("status = %d, want %d", rr.Code, tt.status)
			}
			if !strings.Contains(rr.Body.String(), tt.codeText) {
				t.Fatalf("body = %q, want %s", rr.Body.String(), tt.codeText)
			}
		})
	}
}

func TestHandler_MultiRouteOversizedBodyReturnsEntityTooLarge(t *testing.T) {
	h := NewHandler(Dependencies{
		Addressing:    config.Addressing{PathStyle: true},
		ReplayBudget:  replaybody.NewBudget(1, replaybody.DefaultAggregateMaxBytes),
		Authenticator: stubAuthenticator{},
		Authorizer:    stubAuthorizer{},
		Router:        stubResolver{matches: []router.Match{{Route: config.Route{Name: "one"}}, {Route: config.Route{Name: "two"}}}},
		Rewriter:      stubRewriter{},
		Dispatcher:    &stubFanout{},
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

func TestHandler_DispatchBudgetExhaustionReturnsSlowDown(t *testing.T) {
	h := NewHandler(Dependencies{
		Addressing:    config.Addressing{PathStyle: true},
		Authenticator: stubAuthenticator{},
		Authorizer:    stubAuthorizer{},
		Router:        stubResolver{matches: []router.Match{{Route: config.Route{Name: "one"}}}},
		Rewriter:      stubRewriter{},
		Dispatcher:    &stubFanout{errs: []error{replaybody.ErrBudgetExhausted}, results: []*dispatch.Result{nil}},
	})
	req := httptest.NewRequest(http.MethodPut, "/bucket/key", strings.NewReader("payload"))
	req.ContentLength = int64(len("payload"))
	rr := httptest.NewRecorder()

	h.ServeHTTP(rr, req)

	if rr.Code != http.StatusServiceUnavailable {
		t.Fatalf("status = %d, want %d", rr.Code, http.StatusServiceUnavailable)
	}
	if !strings.Contains(rr.Body.String(), "SlowDown") {
		t.Fatalf("body = %q, want SlowDown", rr.Body.String())
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

func TestHandler_ForwardsEncodedObjectBytesAndHeaders(t *testing.T) {
	encoded := []byte("\x1f\x8bencoded-object-bytes")
	dispatcher := &stubFanout{results: []*dispatch.Result{{Primary: &s3.Response{
		StatusCode: http.StatusOK,
		Header: http.Header{
			"Content-Encoding": []string{"gzip"},
			"Content-Length":   []string{"22"},
			"ETag":             []string{`"encoded-etag"`},
		},
		Body: io.NopCloser(bytes.NewReader(encoded)),
	}}}}
	h := NewHandler(Dependencies{
		Addressing:    config.Addressing{PathStyle: true},
		Authenticator: stubAuthenticator{},
		Authorizer:    stubAuthorizer{},
		Router:        stubResolver{matches: []router.Match{{Route: config.Route{Name: "one"}}}},
		Rewriter:      stubRewriter{},
		Dispatcher:    dispatcher,
	})
	req := httptest.NewRequest(http.MethodGet, "/bucket/key", nil)
	rr := httptest.NewRecorder()

	h.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rr.Code, http.StatusOK)
	}
	if !bytes.Equal(rr.Body.Bytes(), encoded) {
		t.Fatalf("body = %q, want encoded bytes %q", rr.Body.Bytes(), encoded)
	}
	resp := rr.Result()
	if got := resp.Header.Get("Content-Encoding"); got != "gzip" {
		t.Fatalf("Content-Encoding = %q, want gzip", got)
	}
	if got := resp.Header.Get("Content-Length"); got != "22" {
		t.Fatalf("Content-Length = %q, want 22", got)
	}
	if got := resp.Header.Get("ETag"); got != `"encoded-etag"` {
		t.Fatalf("ETag = %q, want encoded-etag", got)
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

func TestHandler_LogsCleanupErrorsWithoutReplacingResult(t *testing.T) {
	var logs bytes.Buffer
	logger := slog.New(slog.NewTextHandler(&logs, nil))
	dispatcher := &stubFanout{results: []*dispatch.Result{{
		Primary:       &s3.Response{StatusCode: http.StatusOK, Header: http.Header{}, Body: io.NopCloser(strings.NewReader("ok"))},
		CleanupErrors: map[string]error{"replica": errors.New("close failed")},
	}}}
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
	if got, want := rr.Body.String(), "ok"; got != want {
		t.Fatalf("body = %q, want %q", got, want)
	}
	for _, want := range []string{"response cleanup failed", "route=one", "target=replica", "close failed"} {
		if !strings.Contains(logs.String(), want) {
			t.Fatalf("logs = %q, want %q", logs.String(), want)
		}
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

func TestHandler_SingleTargetTransportFailureLogsOneSafeDestinationFailure(t *testing.T) {
	var logs bytes.Buffer
	logger := slog.New(slog.NewTextHandler(&logs, nil))
	transportErr := &url.Error{
		Op:  "Get",
		URL: "http://user:pass@upstream.local/private-bucket/secret-object?token=sentinel-query-value", // pragma: allowlist secret
		Err: errors.New("dial tcp connection refused"),
	}
	h := NewHandler(Dependencies{
		Addressing:    config.Addressing{PathStyle: true},
		Authenticator: stubAuthenticator{},
		Authorizer:    stubAuthorizer{},
		Router:        stubResolver{matches: []router.Match{{Route: config.Route{Name: "objects"}}}},
		Rewriter:      stubRewriter{},
		Dispatcher: &stubFanout{
			results: []*dispatch.Result{{Attempts: []dispatch.Attempt{{Target: "primary", Error: transportErr}}}},
			errs:    []error{transportErr},
		},
		Logger: logger,
	})
	req := httptest.NewRequest(http.MethodGet, "/private-bucket/secret-object?x-id=GetObject", nil)
	req.Header.Set("X-Request-Id", "req-safe")
	rr := httptest.NewRecorder()

	h.ServeHTTP(rr, req)

	logText := logs.String()
	if rr.Code != http.StatusBadGateway {
		t.Fatalf("status = %d, want %d", rr.Code, http.StatusBadGateway)
	}
	if got, want := strings.Count(logText, "destination attempt failed"), 1; got != want {
		t.Fatalf("destination failure logs = %d, want %d: %q", got, want, logText)
	}
	for _, want := range []string{"request_id=req-safe", "route=objects", "operation=GetObject", "target=primary", "dispatch failed"} {
		if !strings.Contains(logText, want) {
			t.Fatalf("logs = %q, want %q", logText, want)
		}
	}
	for _, forbidden := range []string{"sentinel-query-value", "user:pass", "private-bucket", "secret-object"} {
		if strings.Contains(logText, forbidden) {
			t.Fatalf("logs = %q, must not contain %q", logText, forbidden)
		}
	}
}

func TestHandler_SingleTargetSuccessDoesNotLogDestinationAttempt(t *testing.T) {
	var logs bytes.Buffer
	logger := slog.New(slog.NewTextHandler(&logs, nil))
	h := NewHandler(Dependencies{
		Addressing:    config.Addressing{PathStyle: true},
		Authenticator: stubAuthenticator{},
		Authorizer:    stubAuthorizer{},
		Router:        stubResolver{matches: []router.Match{{Route: config.Route{Name: "objects"}}}},
		Rewriter:      stubRewriter{},
		Dispatcher: &stubFanout{results: []*dispatch.Result{{
			Primary:  &s3.Response{StatusCode: http.StatusOK, Header: http.Header{}, Body: io.NopCloser(strings.NewReader("ok"))},
			Attempts: []dispatch.Attempt{{Target: "primary", StatusCode: http.StatusOK}},
		}}},
		Logger: logger,
	})
	req := httptest.NewRequest(http.MethodGet, "/bucket/key", nil)
	rr := httptest.NewRecorder()

	h.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rr.Code, http.StatusOK)
	}
	if strings.Contains(logs.String(), "destination attempt") {
		t.Fatalf("logs = %q, want no destination attempt log", logs.String())
	}
}

func TestHandler_RouteMissLogsOmitObjectIdentifiers(t *testing.T) {
	var logs bytes.Buffer
	logger := slog.New(slog.NewTextHandler(&logs, nil))
	h := NewHandler(Dependencies{
		Addressing:    config.Addressing{PathStyle: true},
		Authenticator: stubAuthenticator{},
		Router:        stubResolver{err: errors.New("no route")},
		Logger:        logger,
	})
	req := httptest.NewRequest(http.MethodGet, "/private-bucket/secret-object?x-id=GetObject", nil)
	rr := httptest.NewRecorder()

	h.ServeHTTP(rr, req)

	logText := logs.String()
	if rr.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want %d", rr.Code, http.StatusNotFound)
	}
	for _, want := range []string{"no route", "operation=GetObject", "request complete", "status=404"} {
		if !strings.Contains(logText, want) {
			t.Fatalf("logs = %q, want %q", logText, want)
		}
	}
	for _, forbidden := range []string{"private-bucket", "secret-object"} {
		if strings.Contains(logText, forbidden) {
			t.Fatalf("logs = %q, must not contain %q", logText, forbidden)
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

type countingDispatcher struct {
	calls    int
	op       s3ops.Operation
	response *s3.Response
}

func (d *countingDispatcher) Dispatch(_ context.Context, _ router.Match, _ *http.Request, op s3ops.Operation, _ rewrite.Result) (*dispatch.Result, error) {
	d.calls++
	d.op = op
	return &dispatch.Result{Primary: d.response}, nil
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
