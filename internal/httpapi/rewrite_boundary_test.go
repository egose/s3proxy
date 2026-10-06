package httpapi

import (
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"text/template"

	"github.com/egose/s3proxy/internal/auth"
	"github.com/egose/s3proxy/internal/backend/s3"
	"github.com/egose/s3proxy/internal/config"
	"github.com/egose/s3proxy/internal/replaybody"
	"github.com/egose/s3proxy/internal/rewrite"
	"github.com/egose/s3proxy/internal/router"
	"github.com/egose/s3proxy/internal/s3ops"
)

func TestHandler_InvalidRewriteDoesNotDispatch(t *testing.T) {
	for _, tt := range []struct {
		method string
		op     s3ops.Operation
	}{
		{"GET", s3ops.OpGetObject}, {"HEAD", s3ops.OpHeadObject}, {"PUT", s3ops.OpPutObject}, {"DELETE", s3ops.OpDeleteObject},
	} {
		for name, rule := range map[string]config.RewriteRule{
			"strip key":      {StripKeyPrefix: "key"},
			"strip path":     {StripPathPrefix: "/bucket/key"},
			"empty template": {KeyTemplate: `{{ "" }}`, CompiledTemplate: template.Must(template.New("key").Parse(`{{ "" }}`))},
			"bucket path":    {Bucket: "store/other"},
		} {
			for _, continued := range []bool{false, true} {
				if continued && !s3ops.IsWrite(tt.op) {
					continue
				}
				t.Run(fmt.Sprintf("%s/%s/continue=%t", tt.method, name, continued), func(t *testing.T) {
					matches := []router.Match{{Route: config.Route{Name: "invalid", Rewrite: rule}}}
					if continued {
						matches = append([]router.Match{{Route: config.Route{Name: "valid"}}}, matches...)
					}
					dispatcher := &countingDispatcher{response: &s3.Response{StatusCode: http.StatusOK}}
					principal := &auth.Principal{AllowRoutes: []string{"valid", "invalid"}, AllowOps: []string{string(tt.op)}}
					h := NewHandler(Dependencies{
						Addressing:    config.Addressing{PathStyle: true},
						ReplayBudget:  replaybody.NewBudget(replaybody.DefaultMaxBytes, replaybody.DefaultAggregateMaxBytes),
						Authenticator: stubAuthenticator{principal: principal},
						Authorizer:    auth.NewAuthorizer(),
						Router:        stubResolver{matches: matches},
						Rewriter:      rewrite.New(),
						Dispatcher:    dispatcher,
					})
					rr := httptest.NewRecorder()
					body := &rewriteBoundaryBody{}
					h.ServeHTTP(rr, httptest.NewRequest(tt.method, "/bucket/key", body))
					if rr.Code != http.StatusBadRequest || !strings.Contains(rr.Body.String(), "<Code>InvalidRequest</Code>") {
						t.Errorf("response = %d %s", rr.Code, rr.Body.String())
					}
					if dispatcher.calls != 0 {
						t.Errorf("dispatch calls = %d, want 0", dispatcher.calls)
					}
					if body.reads != 0 {
						t.Errorf("body reads = %d, want 0", body.reads)
					}
				})
			}
		}
	}
}

func TestHandler_BucketOnlyRewriteDispatches(t *testing.T) {
	for _, tt := range []struct {
		method, query string
		op            s3ops.Operation
	}{
		{"HEAD", "", s3ops.OpHeadBucket}, {"GET", "?list-type=2", s3ops.OpListObjectsV2},
	} {
		dispatcher := &countingDispatcher{response: &s3.Response{StatusCode: http.StatusOK}}
		h := NewHandler(Dependencies{
			Addressing:    config.Addressing{PathStyle: true},
			Authenticator: stubAuthenticator{},
			Authorizer:    auth.NewAuthorizer(),
			Router:        stubResolver{matches: []router.Match{{Route: config.Route{Name: "bucket", Rewrite: config.RewriteRule{Bucket: "store"}}}}},
			Rewriter:      rewrite.New(),
			Dispatcher:    dispatcher,
		})
		rr := httptest.NewRecorder()
		h.ServeHTTP(rr, httptest.NewRequest(tt.method, "/bucket"+tt.query, nil))
		if rr.Code != http.StatusOK || dispatcher.calls != 1 || dispatcher.op != tt.op {
			t.Fatalf("%s: status = %d, calls = %d, operation = %s", tt.op, rr.Code, dispatcher.calls, dispatcher.op)
		}
	}
}

func TestHandler_PrefixBucketRewriteDispatches(t *testing.T) {
	for _, tt := range []struct{ method, query string }{{"HEAD", ""}, {"GET", "?list-type=2"}} {
		dispatcher := &countingDispatcher{response: &s3.Response{StatusCode: http.StatusOK}}
		h := NewHandler(Dependencies{
			Addressing:    config.Addressing{PathStyle: true},
			Authenticator: stubAuthenticator{},
			Authorizer:    auth.NewAuthorizer(),
			Router:        stubResolver{matches: []router.Match{{Route: config.Route{Name: "bucket", Rewrite: config.RewriteRule{PrependKeyPrefix: "assets/"}}}}},
			Rewriter:      rewrite.New(),
			Dispatcher:    dispatcher,
		})
		rr := httptest.NewRecorder()
		h.ServeHTTP(rr, httptest.NewRequest(tt.method, "/bucket"+tt.query, nil))
		if rr.Code != http.StatusOK || dispatcher.calls != 1 {
			t.Fatalf("%s: status = %d, calls = %d", tt.method, rr.Code, dispatcher.calls)
		}
	}
}

type rewriteBoundaryBody struct{ reads int }

func (b *rewriteBoundaryBody) Read([]byte) (int, error) {
	b.reads++
	return 0, io.EOF
}
