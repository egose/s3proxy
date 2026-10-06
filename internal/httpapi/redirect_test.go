package httpapi

import (
	"bytes"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/egose/s3proxy/internal/auth"
	"github.com/egose/s3proxy/internal/backend/s3"
	"github.com/egose/s3proxy/internal/config"
	"github.com/egose/s3proxy/internal/dispatch"
	"github.com/egose/s3proxy/internal/replaybody"
	"github.com/egose/s3proxy/internal/rewrite"
	"github.com/egose/s3proxy/internal/router"
)

func TestHandlerRedirectReleasesReplayBudget(t *testing.T) {
	for _, status := range []int{301, 302, 303, 307, 308} {
		for _, fanout := range []bool{false, true} {
			t.Run(fmt.Sprintf("%d/fanout=%t", status, fanout), func(t *testing.T) {
				budget := replaybody.NewBudget(64, 128)
				var calls atomic.Int32
				upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					calls.Add(1)
					if r.URL.Path != "/store/key" {
						return
					}
					data, err := io.ReadAll(r.Body)
					if err != nil || string(data) != "payload" || budget.Used() != 7 {
						t.Error("missing upload/replay reservation")
					}
					w.Header().Set("Location", "/store")
					w.WriteHeader(status)
				}))
				defer upstream.Close()
				endpoint, err := url.Parse(upstream.URL)
				if err != nil {
					t.Fatal(err)
				}
				target := config.S3Target{EndpointURL: endpoint, Region: "us-east-1", ForcePathStyle: true, Credentials: config.StaticCredential{AccessKey: "ak", SecretKey: "sk"}}
				backend, err := s3.NewClient(upstream.Client(), map[string]config.S3Target{"primary": target, "replica": target}, budget)
				if err != nil {
					t.Fatal(err)
				}
				dispatcher, err := dispatch.New(backend, budget)
				if err != nil {
					t.Fatal(err)
				}
				mode := config.DispatchFirst
				wantCalls := int32(1)
				if fanout {
					mode = config.DispatchAll
					wantCalls = 2
				}
				resolver := router.NewResolver([]config.Route{{Name: "objects", ParserRef: "visible", DestinationRefs: []string{"primary", "replica"}, Operations: []string{"PutObject"}, Dispatch: mode, OnMatch: config.MatchStop, Rewrite: config.RewriteRule{Bucket: "store"}}}, map[string]config.Parser{"visible": {Kind: config.ParserBucketExact, Bucket: "visible"}}, []string{"primary", "replica"})
				var logs bytes.Buffer
				h := NewHandler(Dependencies{
					Addressing: config.Addressing{PathStyle: true}, ReplayBudget: budget,
					Authenticator: stubAuthenticator{principal: &auth.Principal{AllowRoutes: []string{"objects"}, AllowOps: []string{"PutObject"}}},
					Authorizer:    auth.NewAuthorizer(), Router: resolver, Rewriter: rewrite.New(), Dispatcher: dispatcher,
					Logger: slog.New(slog.NewTextHandler(&logs, nil)),
				})
				rr := httptest.NewRecorder()
				h.ServeHTTP(rr, httptest.NewRequest("PUT", "/visible/key", io.NopCloser(strings.NewReader("payload"))))
				if rr.Code != http.StatusBadGateway || !strings.Contains(rr.Body.String(), "<Code>InternalError</Code>") || rr.Header().Get("Location") != "" {
					t.Error("expected controlled redirect failure")
				}
				if calls.Load() != wantCalls {
					t.Errorf("upstream calls=%d want=%d", calls.Load(), wantCalls)
				}
				if budget.Used() != 0 {
					t.Errorf("replay reservation leaked: %d", budget.Used())
				}
			})
		}
	}
}
