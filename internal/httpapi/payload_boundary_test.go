package httpapi

import (
	"bytes"
	"compress/gzip"
	"context"
	"crypto/sha256"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"reflect"
	"strings"
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
	"github.com/egose/s3proxy/internal/router"
)

const payloadEnvelope = "5;chunk-signature=PAYLOAD_SECRET\r\nhello\r\n0;chunk-signature=PAYLOAD_SECRET\r\n\r\n"

func signPayloadRequest(t *testing.T, r *http.Request, mode, payload string) {
	t.Helper()
	if mode == "none" {
		return
	}
	hash := fmt.Sprintf("%x", sha256.Sum256([]byte(payload)))
	r.Header.Set("X-Amz-Content-Sha256", hash)
	signer := v4.NewSigner(func(o *v4.SignerOptions) {
		o.DisableURIPathEscaping = true
	})
	creds := aws.Credentials{AccessKeyID: "allowed-ak", SecretAccessKey: "allowed-sk"}
	if mode == "header" {
		if err := signer.SignHTTP(context.Background(), creds, r, hash, "s3", "us-east-1", time.Now().UTC()); err != nil {
			t.Fatal(err)
		}
		return
	}
	r.URL.RawQuery = "X-Amz-Expires=600"
	uri, _, err := signer.PresignHTTP(context.Background(), creds, r, hash, "s3", "us-east-1", time.Now().UTC())
	if err != nil {
		t.Fatal(err)
	}
	r.URL, err = url.Parse(uri)
	if err != nil {
		t.Fatal(err)
	}
}

func payloadFixture(t *testing.T, mode, topology string) (*probeFixture, *replaybody.Budget, chan http.Header) {
	t.Helper()
	f := newProbeFixture(t, config.Addressing{PathStyle: true})
	captured := make(chan http.Header, 8)
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		f.calls.Add(1)
		data, err := io.ReadAll(r.Body)
		if err != nil {
			t.Error(err)
			w.WriteHeader(400)
			return
		}
		f.mu.Lock()
		f.objects[r.URL.Path] = string(data)
		f.mu.Unlock()
		captured <- r.Header.Clone()
	}))
	t.Cleanup(upstream.Close)
	endpoint, err := url.Parse(upstream.URL)
	if err != nil {
		t.Fatal(err)
	}
	budget := replaybody.NewBudget(1024, 4096)
	backend, err := s3.NewClient(upstream.Client(), map[string]config.S3Target{"store": {
		EndpointURL: endpoint, Region: "us-east-1", ForcePathStyle: true,
		Credentials: config.StaticCredential{AccessKey: "backend-ak", SecretKey: "backend-sk"},
	}}, budget)
	if err != nil {
		t.Fatal(err)
	}
	d, err := dispatch.New(backend, budget)
	if err != nil {
		t.Fatal(err)
	}
	h := f.proxy.Config.Handler.(*handler)
	h.deps.Dispatcher, h.deps.ReplayBudget = d, budget
	if mode == "none" {
		f.auth.Authenticator, err = auth.NewAuthenticator(config.Auth{Mode: config.AuthModeNone}, nil)
		if err != nil {
			t.Fatal(err)
		}
	}
	route := config.Route{Name: "visible", ParserRef: "visible", Operations: []string{"PutObject"}, DestinationRefs: []string{"store"}, Dispatch: config.DispatchFirst, OnMatch: config.MatchStop, Rewrite: config.RewriteRule{Bucket: "visible"}}
	if topology == "fanout" {
		route.Dispatch = config.DispatchAll
		route.DestinationRefs = []string{"store", "store"}
	}
	routes := []config.Route{route}
	if topology == "continue" {
		routes[0].OnMatch = config.MatchContinue
		route.Name = "continued"
		route.Rewrite.Bucket = "continued"
		routes = append(routes, route)
	}
	f.router.RouteResolver = router.NewResolver(routes, map[string]config.Parser{"visible": {Kind: config.ParserBucketExact, Bucket: "visible"}}, []string{"store"})
	return f, budget, captured
}

func TestHandler_PayloadBoundaryHTTP(t *testing.T) {
	for _, mode := range []string{"none", "header", "presign"} {
		for _, marker := range []string{"encoding", "hash", "trailer", "combined"} {
			for _, topology := range []string{"single", "fanout", "continue"} {
				for _, unknown := range []bool{false, true} {
					t.Run(fmt.Sprintf("%s/%s/%s/unknown=%t", mode, marker, topology, unknown), func(t *testing.T) {
						f, budget, captured := payloadFixture(t, mode, topology)
						f.objects["/visible/key"] = "original"
						var reads atomic.Int64
						h := f.proxy.Config.Handler
						f.proxy.Config.Handler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
							r.Body = queryReadCounter{r.Body, &reads}
							h.ServeHTTP(w, r)
						})
						r, err := http.NewRequest("PUT", f.proxy.URL+"/visible/key", strings.NewReader(payloadEnvelope))
						if err != nil {
							t.Fatal(err)
						}
						if unknown {
							r.ContentLength, r.GetBody = -1, nil
						}
						if marker == "encoding" || marker == "combined" {
							r.Header["Content-Encoding"] = []string{"gzip", " \tAwS-ChUnKeD \t"}
						}
						if marker == "trailer" || marker == "combined" {
							r.Header["X-Amz-Trailer"] = []string{"", " x-amz-checksum-PAYLOAD_SECRET "}
						}
						if mode != "presign" {
							r.Header.Set("X-Amz-Decoded-Content-Length", "5")
						}
						trailer := r.Header.Values("X-Amz-Trailer")
						if mode == "presign" {
							r.Header.Del("X-Amz-Trailer")
						}
						signPayloadRequest(t, r, mode, payloadEnvelope)
						if len(trailer) > 0 {
							r.Header["X-Amz-Trailer"] = trailer
						}
						if marker == "hash" || marker == "combined" {
							r.Header.Add("X-Amz-Content-Sha256", " \tStReAmInG-PAYLOAD_SECRET \t")
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
						stored, _ := f.object("/visible/key")
						if resp.StatusCode != 501 || !strings.Contains(string(data), "<Code>NotImplemented</Code>") || stored != "original" {
							t.Errorf("status=%d response=%s stored=%q; want 501 and original", resp.StatusCode, data, stored)
							select {
							case header := <-captured:
								t.Logf("local fixture captured encoding=%q hash=%q decoded-length=%q trailer=%q backend-authorized=%t", header.Values("Content-Encoding"), header.Get("X-Amz-Content-Sha256"), header.Get("X-Amz-Decoded-Content-Length"), header.Values("X-Amz-Trailer"), strings.Contains(header.Get("Authorization"), "backend-ak/"))
							default:
							}
						}
						if reads.Load() != 0 || f.calls.Load() != 0 || f.auth.calls.Load() != 0 || f.router.calls.Load() != 0 || budget.Used() != 0 {
							t.Errorf("reads/backend/auth/router/budget=%d/%d/%d/%d/%d", reads.Load(), f.calls.Load(), f.auth.calls.Load(), f.router.calls.Load(), budget.Used())
						}
						f.mu.Lock()
						logs := f.logs.String()
						f.mu.Unlock()
						for _, secret := range []string{"PAYLOAD_SECRET", "allowed-ak", "allowed-sk", "X-Amz-Signature=", "chunk-signature="} { // pragma: allowlist secret
							if strings.Contains(logs+string(data), secret) {
								t.Errorf("diagnostics exposed %q", secret)
							}
						}
					})
				}
			}
		}
	}
}

func TestHandler_PayloadOrdinaryStorageHTTP(t *testing.T) {
	var compressed bytes.Buffer
	zw := gzip.NewWriter(&compressed)
	if _, err := zw.Write([]byte(payloadEnvelope)); err != nil {
		t.Fatal(err)
	}
	if err := zw.Close(); err != nil {
		t.Fatal(err)
	}
	for _, mode := range []string{"none", "header", "presign"} {
		for _, encoding := range []string{"", "gzip", "not-aws-chunked"} {
			for _, unknown := range []bool{false, true} {
				t.Run(fmt.Sprintf("%s/%s/unknown=%t", mode, encoding, unknown), func(t *testing.T) {
					f, budget, captured := payloadFixture(t, mode, "single")
					var wireChunked atomic.Bool
					h := f.proxy.Config.Handler
					f.proxy.Config.Handler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
						wireChunked.Store(len(r.TransferEncoding) == 1 && r.TransferEncoding[0] == "chunked")
						h.ServeHTTP(w, r)
					})
					payload := payloadEnvelope
					if encoding == "gzip" {
						payload = compressed.String()
					}
					r, err := http.NewRequest("PUT", f.proxy.URL+"/visible/key", strings.NewReader(payload))
					if err != nil {
						t.Fatal(err)
					}
					if unknown {
						r.ContentLength, r.GetBody = -1, nil
					}
					r.Header.Set("Content-Encoding", encoding)
					checksum := ""
					if mode != "presign" {
						checksum = "AAAAAA=="
						r.Header.Set("X-Amz-Checksum-Crc32", checksum)
					}
					signPayloadRequest(t, r, mode, payload)
					resp, err := f.proxy.Client().Do(r)
					if err != nil {
						t.Fatal(err)
					}
					data, err := io.ReadAll(resp.Body)
					resp.Body.Close()
					if err != nil {
						t.Fatal(err)
					}
					stored, ok := f.object("/visible/key")
					if resp.StatusCode != 200 || !ok || stored != payload || f.calls.Load() != 1 || wireChunked.Load() != unknown {
						t.Fatalf("status=%d response=%s stored=%q present=%t calls=%d", resp.StatusCode, data, stored, ok, f.calls.Load())
					}
					header := <-captured
					if header.Get("Content-Encoding") != encoding || header.Get("X-Amz-Checksum-Crc32") != checksum || budget.Used() != 0 {
						t.Fatalf("encoding=%q checksum=%q budget=%d", header.Get("Content-Encoding"), header.Get("X-Amz-Checksum-Crc32"), budget.Used())
					}
				})
			}
		}
	}
}

func TestHandler_PayloadMalformedQueryHTTP(t *testing.T) {
	for _, mode := range []string{"none", "header", "presign"} {
		t.Run(mode, func(t *testing.T) {
			f, budget, _ := payloadFixture(t, mode, "fanout")
			f.objects["/visible/key"] = "original"
			var reads atomic.Int64
			h := f.proxy.Config.Handler
			f.proxy.Config.Handler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				r.Body = queryReadCounter{r.Body, &reads}
				h.ServeHTTP(w, r)
			})
			r, err := http.NewRequest("PUT", f.proxy.URL+"/visible/key", strings.NewReader(payloadEnvelope))
			if err != nil {
				t.Fatal(err)
			}
			r.Header.Set("Content-Encoding", "aws-chunked")
			signPayloadRequest(t, r, mode, payloadEnvelope)
			r.URL.RawQuery += "&SECRET=%GG"
			resp, err := f.proxy.Client().Do(r)
			if err != nil {
				t.Fatal(err)
			}
			data, err := io.ReadAll(resp.Body)
			resp.Body.Close()
			if err != nil {
				t.Fatal(err)
			}
			stored, _ := f.object("/visible/key")
			if resp.StatusCode != 400 || !strings.Contains(string(data), "<Code>InvalidRequest</Code>") || stored != "original" || reads.Load() != 0 || f.auth.calls.Load() != 0 || f.router.calls.Load() != 0 || f.calls.Load() != 0 || budget.Used() != 0 {
				t.Fatalf("status=%d response=%s stored=%q reads/auth/router/backend=%d/%d/%d/%d", resp.StatusCode, data, stored, reads.Load(), f.auth.calls.Load(), f.router.calls.Load(), f.calls.Load())
			}
			f.mu.Lock()
			defer f.mu.Unlock()
			for _, secret := range []string{"SECRET", "%GG", "allowed-ak", "X-Amz-Signature="} { // pragma: allowlist secret
				if strings.Contains(f.logs.String()+string(data), secret) {
					t.Errorf("diagnostics exposed %q", secret)
				}
			}
		})
	}
}

func TestHandler_PayloadBoundaryPrecedenceAndOwnership(t *testing.T) {
	for _, mode := range []string{"none", "header", "presign", "invalid auth"} {
		for _, tc := range []struct {
			name, method, path string
			status             int
		}{
			{"upload", "PUT", "/visible/key", 501},
			{"malformed query", "PUT", "/visible/key?secret=%GG", 400},
			{"malformed health query", "GET", "/healthz?secret=%GG", 400},
		} {
			for _, getBody := range []bool{false, true} {
				t.Run(fmt.Sprintf("%s/%s/getBody=%t", mode, tc.name, getBody), func(t *testing.T) {
					f, budget, _ := payloadFixture(t, mode, "continue")
					h := f.proxy.Config.Handler.(*handler)
					h.deps.ReplayBudget = replaybody.NewBudget(1, 1)
					r := httptest.NewRequest(tc.method, "http://proxy/visible/key", nil)
					if mode == "invalid auth" {
						r.Header.Set("Authorization", "SECRET invalid credentials")
					} else {
						signPayloadRequest(t, r, mode, payloadEnvelope)
					}
					u, err := url.Parse(tc.path)
					if err != nil {
						t.Fatal(err)
					}
					r.URL.Path = u.Path
					if u.RawQuery != "" {
						r.URL.RawQuery += "&" + u.RawQuery
					}
					r.Header["content-ENCODING"] = []string{"gzip", " \tAWS-CHUNKED \t"}
					r.Header["x-amz-trailer"] = []string{"", "SECRET"}
					var reads atomic.Int64
					body := queryReadCounter{io.NopCloser(strings.NewReader(payloadEnvelope)), &reads}
					r.Body, r.ContentLength = body, -1
					getCalls := 0
					if getBody {
						r.GetBody = func() (io.ReadCloser, error) {
							getCalls++
							return io.NopCloser(strings.NewReader(payloadEnvelope)), nil
						}
					}
					before := r.Header.Clone()
					w := httptest.NewRecorder()
					h.ServeHTTP(w, r)
					if w.Code != tc.status || reads.Load() != 0 || getCalls != 0 || f.auth.calls.Load() != 0 || f.router.calls.Load() != 0 || f.calls.Load() != 0 || budget.Used() != 0 || h.deps.ReplayBudget.Used() != 0 {
						t.Fatalf("status=%d reads=%d getBody=%d auth/router/backend=%d/%d/%d", w.Code, reads.Load(), getCalls, f.auth.calls.Load(), f.router.calls.Load(), f.calls.Load())
					}
					if !reflect.DeepEqual(before, r.Header) || r.Body != body || (r.GetBody != nil) != getBody || r.ContentLength != -1 {
						t.Fatal("caller request changed")
					}
					if strings.Contains(f.logs.String()+w.Body.String(), "SECRET") || strings.Contains(f.logs.String()+w.Body.String(), "%GG") {
						t.Fatal("unsafe diagnostics")
					}
				})
			}
		}
	}
	for _, path := range []string{"/healthz", "/readyz"} {
		t.Run(path, func(t *testing.T) {
			h := NewHandler(Dependencies{Addressing: config.Addressing{PathStyle: true}})
			r := httptest.NewRequest("GET", "http://proxy"+path, nil)
			r.Header.Set("Content-Encoding", "aws-chunked")
			w := httptest.NewRecorder()
			h.ServeHTTP(w, r)
			if w.Code != 200 || w.Body.String() != "ok" {
				t.Fatalf("probe=%d %s", w.Code, w.Body.String())
			}
		})
	}
}
