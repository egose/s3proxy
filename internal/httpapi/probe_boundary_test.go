package httpapi

import (
	"context"
	"encoding/xml"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	v4 "github.com/aws/aws-sdk-go-v2/aws/signer/v4"
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

type probeAuthenticator struct {
	Authenticator
	calls atomic.Int64
}

func (a *probeAuthenticator) Authenticate(r *http.Request) (*auth.Principal, error) {
	a.calls.Add(1)
	return a.Authenticator.Authenticate(r)
}

type probeResolver struct {
	RouteResolver
	calls atomic.Int64
}

func (r *probeResolver) Resolve(ctx *requestctx.Context, op s3ops.Operation) ([]router.Match, error) {
	r.calls.Add(1)
	return r.RouteResolver.Resolve(ctx, op)
}

type probeFixture struct {
	proxy   *httptest.Server
	auth    *probeAuthenticator
	router  *probeResolver
	calls   atomic.Int64
	mu      sync.Mutex
	objects map[string]string
	logs    strings.Builder
}

func (f *probeFixture) Write(p []byte) (int, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.logs.Write(p)
}

func (f *probeFixture) object(key string) (string, bool) {
	f.mu.Lock()
	defer f.mu.Unlock()
	value, ok := f.objects[key]
	return value, ok
}

func newProbeFixture(t *testing.T, addressing config.Addressing) *probeFixture {
	t.Helper()
	f := &probeFixture{objects: map[string]string{}}
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		f.calls.Add(1)
		f.mu.Lock()
		defer f.mu.Unlock()
		key := r.URL.EscapedPath()
		w.Header().Set("X-Test-Backend", "called")
		if r.URL.Query().Get("list-type") == "2" {
			result := workflowList{Name: strings.TrimPrefix(r.URL.Path, "/")}
			for path, data := range f.objects {
				if strings.HasPrefix(path, key+"/") {
					result.Contents = append(result.Contents, workflowObject{Key: strings.TrimPrefix(path, key+"/"), Size: len(data)})
				}
			}
			if err := xml.NewEncoder(w).Encode(result); err != nil {
				t.Error(err)
			}
			return
		}
		if r.Method == http.MethodHead && !strings.Contains(strings.TrimPrefix(key, "/"), "/") {
			return
		}
		switch r.Method {
		case http.MethodPut:
			data, err := io.ReadAll(r.Body)
			if err != nil {
				t.Error(err)
				w.WriteHeader(400)
				return
			}
			f.objects[key] = string(data)
		case http.MethodGet, http.MethodHead:
			data, ok := f.objects[key]
			if !ok {
				w.WriteHeader(404)
				return
			}
			w.Header().Set("Content-Length", strconv.Itoa(len(data)))
			if r.Method == http.MethodGet {
				_, _ = io.WriteString(w, data)
			}
		case http.MethodDelete:
			delete(f.objects, key)
			w.WriteHeader(204)
		default:
			t.Errorf("unexpected backend method %s", r.Method)
			w.WriteHeader(500)
		}
	}))
	t.Cleanup(upstream.Close)
	endpoint, err := url.Parse(upstream.URL)
	if err != nil {
		t.Fatal(err)
	}
	budget := replaybody.NewBudget(replaybody.DefaultMaxBytes, replaybody.DefaultAggregateMaxBytes)
	backend, err := s3.NewClient(upstream.Client(), map[string]config.S3Target{"store": {
		EndpointURL: endpoint, Region: "us-east-1", ForcePathStyle: true,
		Credentials: config.StaticCredential{AccessKey: "backend-ak", SecretKey: "backend-sk"},
	}}, budget)
	if err != nil {
		t.Fatal(err)
	}
	dispatcher, err := dispatch.New(backend, budget)
	if err != nil {
		t.Fatal(err)
	}
	ops := []string{"PutObject", "GetObject", "HeadObject", "DeleteObject", "ListObjectsV2", "HeadBucket"}
	clients := map[string]config.Client{}
	for _, name := range []string{"allowed", "route-denied", "operation-denied"} {
		client := config.Client{Name: name, AccessKey: name + "-ak", SecretKey: name + "-sk", AllowRoutes: []string{"*"}, AllowOps: ops}
		if name == "route-denied" {
			client.AllowRoutes = []string{"other"}
		}
		if name == "operation-denied" {
			client.AllowOps = []string{"ListBuckets"}
		}
		clients[name] = client
	}
	authenticator, err := auth.NewAuthenticator(config.Auth{Mode: config.AuthModeSigV4Static, Clients: clients}, budget)
	if err != nil {
		t.Fatal(err)
	}
	f.auth = &probeAuthenticator{Authenticator: authenticator}
	parsers := map[string]config.Parser{}
	var routes []config.Route
	for _, bucket := range []string{"visible", "healthz", "readyz"} {
		parsers[bucket] = config.Parser{Kind: config.ParserBucketExact, Bucket: bucket}
		routes = append(routes, config.Route{Name: bucket, ParserRef: bucket, Operations: ops, DestinationRefs: []string{"store"}, Dispatch: config.DispatchFirst, OnMatch: config.MatchStop, Rewrite: config.RewriteRule{Bucket: bucket}})
	}
	f.router = &probeResolver{RouteResolver: router.NewResolver(routes, parsers, []string{"store"})}
	f.proxy = httptest.NewServer(NewHandler(Dependencies{
		Addressing: addressing, ReplayBudget: budget, Authenticator: f.auth,
		Authorizer: auth.NewAuthorizer(), Router: f.router, Rewriter: rewrite.New(), Dispatcher: dispatcher,
		Logger: slog.New(slog.NewTextHandler(f, nil)),
	}))
	t.Cleanup(f.proxy.Close)
	return f
}

func (f *probeFixture) request(t *testing.T, method, host, path, payload, client, signing string) (*http.Response, string) {
	t.Helper()
	var body io.Reader
	if payload != "" {
		body = strings.NewReader(payload)
	}
	r, err := http.NewRequest(method, f.proxy.URL+path, body)
	if err != nil {
		t.Fatal(err)
	}
	r.Host = host
	r.Header.Set("X-Request-Id", "probe-boundary")
	if signing == "empty-header" {
		r.Header["Authorization"] = []string{""}
	} else if signing != "" {
		signer := v4.NewSigner(func(o *v4.SignerOptions) { o.DisableURIPathEscaping = true })
		creds := aws.Credentials{AccessKeyID: client + "-ak", SecretAccessKey: client + "-sk"}
		if strings.HasPrefix(signing, "presign") {
			q := r.URL.Query()
			q.Set("X-Amz-Expires", "600")
			r.URL.RawQuery = q.Encode()
			uri, _, err := signer.PresignHTTP(context.Background(), creds, r, "UNSIGNED-PAYLOAD", "s3", "us-east-1", time.Now().UTC())
			if err != nil {
				t.Fatal(err)
			}
			r.URL, err = url.Parse(uri)
			if err != nil {
				t.Fatal(err)
			}
			if signing == "presign-invalid" {
				q := r.URL.Query()
				q.Set("X-Amz-Signature", strings.Repeat("0", 64))
				r.URL.RawQuery = q.Encode()
			}
		} else {
			r.Header.Set("X-Amz-Content-Sha256", "UNSIGNED-PAYLOAD")
			if method == http.MethodPut {
				r.Header.Set("Content-Length", strconv.Itoa(len(payload)))
			}
			if err := signer.SignHTTP(context.Background(), creds, r, "UNSIGNED-PAYLOAD", "s3", "us-east-1", time.Now().UTC()); err != nil {
				t.Fatal(err)
			}
			if signing == "header-invalid" {
				r.Header.Set("Authorization", r.Header.Get("Authorization")+"0")
			}
		}
	}
	resp, err := f.proxy.Client().Do(r)
	if err != nil {
		t.Fatal(err)
	}
	data, err := io.ReadAll(resp.Body)
	resp.Body.Close()
	if err != nil {
		t.Fatal(err)
	}
	if resp.Header.Get("X-Request-Id") != "probe-boundary" {
		t.Fatal("request ID was not retained")
	}
	return resp, string(data)
}

func TestHandler_ProbeObjectLifecycle(t *testing.T) {
	for _, signing := range []string{"header", "presign"} {
		for _, key := range []string{"healthz", "readyz", "%68ealthz", "%72eadyz"} {
			t.Run(signing+"/"+key, func(t *testing.T) {
				f := newProbeFixture(t, config.Addressing{PathStyle: true, VirtualHosted: true, HostSuffixes: []string{"S3Proxy.Test.:8080"}})
				for _, step := range []struct {
					method string
					status int
					stored bool
				}{{"PUT", 200, true}, {"GET", 200, true}, {"HEAD", 200, true}, {"DELETE", 204, false}, {"GET", 404, false}, {"HEAD", 404, false}} {
					payload := ""
					if step.method == "PUT" {
						payload = "persisted " + key
					}
					before := f.calls.Load()
					resp, body := f.request(t, step.method, "Visible.S3Proxy.Test.:9000", "/"+key, payload, "allowed", signing)
					stored, exists := f.object("/visible/" + key)
					if resp.StatusCode != step.status || f.calls.Load() != before+1 || exists != step.stored || (exists && stored != "persisted "+key) {
						t.Fatalf("%s: status=%d body=%q backend calls=%d stored=%q exists=%v; want status=%d, one backend call, exists=%v", step.method, resp.StatusCode, body, f.calls.Load()-before, stored, exists, step.status, step.stored)
					}
					if step.method == "GET" && step.stored && body != stored {
						t.Fatalf("GET bytes=%q, stored=%q", body, stored)
					}
					if step.method == "HEAD" && step.stored && (body != "" || resp.ContentLength != int64(len(stored))) {
						t.Fatalf("HEAD body=%q length=%d", body, resp.ContentLength)
					}
				}
			})
		}
	}
}

func TestHandler_ProbeObjectDenials(t *testing.T) {
	f := newProbeFixture(t, config.Addressing{PathStyle: true, VirtualHosted: true, HostSuffixes: []string{"s3proxy.test"}})
	for _, key := range []string{"healthz", "readyz"} {
		resp, body := f.request(t, "PUT", "visible.s3proxy.test", "/"+key, "original", "allowed", "header")
		if resp.StatusCode != 200 {
			t.Fatalf("seed: %d %s", resp.StatusCode, body)
		}
		for _, method := range []string{"PUT", "GET", "HEAD", "DELETE"} {
			for _, denial := range []struct{ client, signing, code string }{
				{"allowed", "", "AccessDenied"}, {"allowed", "empty-header", "AccessDenied"},
				{"allowed", "header-invalid", "SignatureDoesNotMatch"}, {"allowed", "presign-invalid", "SignatureDoesNotMatch"},
				{"route-denied", "header", "AccessDenied"}, {"route-denied", "presign", "AccessDenied"},
				{"operation-denied", "header", "AccessDenied"}, {"operation-denied", "presign", "AccessDenied"},
			} {
				t.Run(key+"/"+method+"/"+denial.client+"/"+denial.signing, func(t *testing.T) {
					before := f.calls.Load()
					resp, body := f.request(t, method, "visible.s3proxy.test", "/"+key, "replacement", denial.client, denial.signing)
					stored, exists := f.object("/visible/" + key)
					if resp.StatusCode != 403 || f.calls.Load() != before || !exists || stored != "original" || (method != "HEAD" && !strings.Contains(body, "<Code>"+denial.code+"</Code>")) {
						t.Fatalf("status=%d body=%q backend calls=%d stored=%q exists=%v", resp.StatusCode, body, f.calls.Load()-before, stored, exists)
					}
				})
			}
		}
	}
}

func TestHandler_ProbeNamedBuckets(t *testing.T) {
	f := newProbeFixture(t, config.Addressing{PathStyle: true})
	for _, bucket := range []string{"healthz", "readyz"} {
		for _, signing := range []string{"header", "presign"} {
			t.Run(bucket+"/"+signing, func(t *testing.T) {
				resp, body := f.request(t, "PUT", "localhost", "/"+bucket+"/item", "listed bytes", "allowed", signing)
				if resp.StatusCode != 200 {
					t.Fatalf("seed: %d %s", resp.StatusCode, body)
				}
				for _, method := range []string{"GET", "HEAD"} {
					path := "/" + bucket
					if method == "GET" {
						path += "?list-type=2"
					}
					before := f.calls.Load()
					resp, body := f.request(t, method, "localhost", path, "", "allowed", signing)
					if resp.StatusCode != 200 || f.calls.Load() != before+1 || resp.Header.Get("X-Test-Backend") != "called" {
						t.Fatalf("%s: status=%d body=%q calls=%d", method, resp.StatusCode, body, f.calls.Load()-before)
					}
					if method == "GET" {
						var result workflowList
						if err := xml.Unmarshal([]byte(body), &result); err != nil || result.Name != bucket || len(result.Contents) != 1 || result.Contents[0].Key != "item" || result.Contents[0].Size != len("listed bytes") {
							t.Fatalf("listing=%+v err=%v", result, err)
						}
					}
				}
			})
		}
	}
}

func TestHandler_ProbeBoundary(t *testing.T) {
	for _, mode := range []string{"both", "virtual-only", "path-only", "no-suffixes"} {
		addressing := config.Addressing{PathStyle: mode != "virtual-only", VirtualHosted: mode != "path-only", HostSuffixes: []string{"S3Proxy.Test.:8080"}}
		if mode == "no-suffixes" {
			addressing.HostSuffixes = nil
		}
		f := newProbeFixture(t, addressing)
		for _, name := range []string{"healthz", "readyz"} {
			for _, host := range []string{"localhost:8080", "127.0.0.1:8080", "[::1]:8080", "S3Proxy.Test.:9000", "s3proxy.test", ".s3proxy.test", "nots3proxy.test", "visible.s3proxy.test.evil", "alias.example"} {
				t.Run(mode+"/"+name+"/probe/"+host, func(t *testing.T) {
					beforeAuth, beforeRouter, beforeBackend := f.auth.calls.Load(), f.router.calls.Load(), f.calls.Load()
					resp, body := f.request(t, "GET", host, "/"+name, "", "", "")
					if resp.StatusCode != 200 || body != "ok" || f.auth.calls.Load() != beforeAuth || f.router.calls.Load() != beforeRouter || f.calls.Load() != beforeBackend {
						t.Fatalf("status=%d body=%q auth/router/backend=%d/%d/%d", resp.StatusCode, body, f.auth.calls.Load()-beforeAuth, f.router.calls.Load()-beforeRouter, f.calls.Load()-beforeBackend)
					}
					f.mu.Lock()
					logs := f.logs.String()
					f.mu.Unlock()
					for _, want := range []string{"request complete", "request_id=probe-boundary", "method=GET", "status=200", "bytes=2", "duration_ms="} {
						if !strings.Contains(logs, want) {
							t.Fatalf("missing %q in logs %q", want, logs)
						}
					}
				})
			}
			for _, host := range []string{"visible.s3proxy.test", "Visible.S3Proxy.Test.:9000", "nested.visible.s3proxy.test"} {
				t.Run(mode+"/"+name+"/bucket-host/"+host, func(t *testing.T) {
					status, wantBody := 403, "<Code>AccessDenied</Code>"
					if mode == "path-only" || mode == "no-suffixes" {
						status, wantBody = 200, "ok"
					}
					before := f.calls.Load()
					resp, body := f.request(t, "GET", host, "/"+name, "", "", "")
					if resp.StatusCode != status || !strings.Contains(body, wantBody) || f.calls.Load() != before {
						t.Fatalf("status=%d body=%q calls=%d", resp.StatusCode, body, f.calls.Load()-before)
					}
				})
			}
			for _, tc := range []struct {
				label, method, path, signing string
				status                       int
			}{
				{"head", "HEAD", "/" + name, "", 403},
				{"put", "PUT", "/" + name, "", 501},
				{"delete", "DELETE", "/" + name, "", 501},
				{"post", "POST", "/" + name, "", 501},
				{"options", "OPTIONS", "/" + name, "", 501},
				{"patch", "PATCH", "/" + name, "", 501},
				{"lowercase-method", "get", "/" + name, "", 501},
				{"signed", "GET", "/" + name, "header", 501},
				{"presigned", "GET", "/" + name, "presign", 501},
				{"invalid-signed", "GET", "/" + name, "header-invalid", 403},
				{"invalid-presigned", "GET", "/" + name, "presign-invalid", 403},
				{"empty-header", "GET", "/" + name, "empty-header", 501},
				{"bare-query", "GET", "/" + name + "?", "", 501},
				{"empty-pairs", "GET", "/" + name + "?&", "", 501},
				{"query", "GET", "/" + name + "?x-id=ListObjectsV2", "", 501},
				{"listing", "GET", "/" + name + "?list-type=2", "", 403},
				{"unsupported-query", "GET", "/" + name + "?acl", "", 501},
				{"malformed-query", "GET", "/" + name + "?prefix=%GG", "", 400},
				{"partial-presign", "GET", "/" + name + "?X-Amz-Signature=bad", "", 403},
				{"encoded-unreserved", "GET", "/" + name[:len(name)-1] + "%7a", "", 501},
				{"encoded-unreserved-upper", "GET", "/" + name[:len(name)-1] + "%7A", "", 501},
				{"encoded-slash", "GET", "/" + name + "%2F", "", 501},
				{"trailing-slash", "GET", "/" + name + "/", "", 501},
				{"similar", "GET", "/" + name + "extra", "", 501},
				{"child", "GET", "/" + name + "/key", "", 403},
				{"double-slash", "GET", "//" + name, "", 403},
			} {
				t.Run(mode+"/"+name+"/"+tc.label, func(t *testing.T) {
					want := tc.status
					if mode == "virtual-only" {
						want = 400
					}
					before := f.calls.Load()
					resp, body := f.request(t, tc.method, "localhost", tc.path, "", "allowed", tc.signing)
					if resp.StatusCode != want || body == "ok" || f.calls.Load() != before {
						t.Fatalf("status=%d want=%d body=%q backend calls=%d", resp.StatusCode, want, body, f.calls.Load()-before)
					}
				})
			}
		}
	}
}
