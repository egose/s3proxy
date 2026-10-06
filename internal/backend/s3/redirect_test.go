package s3

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/cookiejar"
	"net/http/httptest"
	"reflect"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/egose/s3proxy/internal/config"
	"github.com/egose/s3proxy/internal/replaybody"
	"github.com/egose/s3proxy/internal/s3ops"
	"github.com/egose/s3proxy/internal/testutil/s3signature"
)

func TestClientRejectsRedirectHTTP(t *testing.T) {
	for _, status := range []int{301, 302, 303, 307, 308} {
		for _, method := range []string{"GET", "PUT", "DELETE"} {
			for _, crossHost := range []bool{false, true} {
				for _, customPolicy := range []bool{false, true} {
					t.Run(fmt.Sprintf("%d/%s/cross=%t/custom=%t", status, method, crossHost, customPolicy), func(t *testing.T) {
						var calls, received, callbacks atomic.Int32
						receiver := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
							received.Add(1)
							io.Copy(io.Discard, r.Body)
						}))
						defer receiver.Close()
						upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
							calls.Add(1)
							if r.URL.Path != "/bucket/key" {
								return
							}
							headers := "host;x-amz-content-sha256;x-amz-date"
							if method == "PUT" {
								headers = "content-length;" + headers
								data, err := io.ReadAll(r.Body)
								if err != nil || string(data) != "payload" || r.Header.Get("Content-Length") != "7" {
									t.Error("upload bytes/framing changed")
								}
							}
							if r.Method != method || r.Header.Get("Authorization") != s3signature.Authorization(r, "ak", "sk", "us-east-1", headers) {
								t.Error("initial request method/signature changed")
							}
							location := "/bucket"
							if crossHost {
								location = strings.Replace(receiver.URL, "127.0.0.1", "localhost", 1) + "/bucket"
							}
							w.Header().Set("Location", location)
							w.WriteHeader(status)
						}))
						defer upstream.Close()
						hc := upstream.Client()
						if customPolicy {
							hc.Timeout = time.Minute
							hc.CheckRedirect = func(*http.Request, []*http.Request) error { callbacks.Add(1); return nil }
						} else {
							hc = &http.Client{}
						}
						before := *hc
						budget := testBudget()
						executor := newTestClient(t, hc, testTargets(t, upstream.URL), budget)
						var body io.Reader
						op := s3ops.OpGetObject
						if method == "PUT" {
							body = io.NopCloser(strings.NewReader("payload"))
							op = s3ops.OpPutObject
						} else if method == "DELETE" {
							op = s3ops.OpDeleteObject
						}
						source := httptest.NewRequest(method, "/visible/key", body)
						resp, err := executor.Do(context.Background(), Request{Operation: op, Target: "primary", Bucket: "bucket", Key: "key", Source: source})
						if resp != nil {
							resp.Body.Close()
						}
						if err == nil || resp != nil || !strings.Contains(err.Error(), "upstream redirect rejected") {
							t.Error("expected controlled redirect error and no response")
						}
						if calls.Load() != 1 || received.Load() != 0 || callbacks.Load() != 0 {
							t.Errorf("upstream=%d receiver=%d callbacks=%d", calls.Load(), received.Load(), callbacks.Load())
						}
						if hc.Transport != before.Transport || hc.Jar != before.Jar || hc.Timeout != before.Timeout || reflect.ValueOf(hc.CheckRedirect).Pointer() != reflect.ValueOf(before.CheckRedirect).Pointer() {
							t.Error("caller HTTP client mutated")
						}
						if executor.(*client).httpClient.Timeout != before.Timeout {
							t.Error("caller HTTP timeout not retained")
						}
						if method == "PUT" && budget.Used() != 7 {
							t.Error("request replay ownership changed before owner release")
						}
						if err := replaybody.Release(source); err != nil || budget.Used() != 0 {
							t.Error("request owner could not release replay budget")
						}
					})
				}
			}
		}
	}
}

func TestClientNonRedirectResponsesRemainStreaming(t *testing.T) {
	for _, status := range []int{200, 201, 204, 304, 403, 404, 500} {
		t.Run(fmt.Sprint(status), func(t *testing.T) {
			body := &listingBody{Reader: strings.NewReader("original body")}
			executor := newTestClient(t, &http.Client{Transport: roundTripFunc(func(*http.Request) (*http.Response, error) {
				return &http.Response{StatusCode: status, Body: body, Header: http.Header{"Location": {"/object"}, "Etag": {"original"}}}, nil
			})}, testTargets(t, "http://upstream.test"), testBudget())
			resp, err := executor.Do(context.Background(), Request{Operation: s3ops.OpGetObject, Target: "primary", Bucket: "bucket", Key: "key", Source: httptest.NewRequest("GET", "/visible/key", nil)})
			if err != nil {
				t.Fatal(err)
			}
			if body.reads.Load() != 0 || body.closes.Load() != 0 {
				t.Error("response consumed before returning")
			}
			data, err := io.ReadAll(resp.Body)
			resp.Body.Close()
			if err != nil || string(data) != "original body" || resp.StatusCode != status || resp.Header.Get("ETag") != "original" || resp.Header.Get("Location") != "/object" || body.closes.Load() != 1 {
				t.Error("ordinary response contract changed")
			}
		})
	}
}

type redirectBody struct {
	reads, closes int
}

func (b *redirectBody) Read([]byte) (int, error) { b.reads++; return 0, io.EOF }
func (b *redirectBody) Close() error             { b.closes++; return errors.New("REDIRECT_CLOSE_SECRET") }

func TestClientRejectsRedirectCleanup(t *testing.T) {
	for _, status := range []int{301, 302, 303, 307, 308} {
		for name, location := range map[string]string{
			"valid":     "http://user:REDIRECT_PASSWORD@elsewhere.test/REDIRECT_KEY?token=REDIRECT_QUERY",     // pragma: allowlist secret
			"malformed": "http://user:REDIRECT_PASSWORD@elsewhere.test/REDIRECT_KEY/%GG?token=REDIRECT_QUERY", // pragma: allowlist secret
			"missing":   "",
		} {
			t.Run(fmt.Sprintf("%d/%s", status, name), func(t *testing.T) {
				responseBody := &redirectBody{}
				var requestContext context.Context
				calls, replays := 0, 0
				transport := roundTripFunc(func(r *http.Request) (*http.Response, error) {
					calls++
					if calls > 1 {
						if r.Body != nil {
							r.Body.Close()
						}
						return &http.Response{StatusCode: 200, Body: http.NoBody, Header: make(http.Header)}, nil
					}
					requestContext = r.Context()
					if _, ok := requestContext.Deadline(); !ok {
						t.Error("target/client deadline lost")
					}
					if r.Header.Get("Cookie") != "session=retained" {
						t.Error("caller jar not retained")
					}
					io.Copy(io.Discard, r.Body)
					r.Body.Close()
					return &http.Response{StatusCode: status, Header: http.Header{"Location": {location}}, Body: responseBody}, nil
				})
				jar, err := cookiejar.New(nil)
				if err != nil {
					t.Fatal(err)
				}
				jar.SetCookies(mustURL(t, "http://upstream.test"), []*http.Cookie{{Name: "session", Value: "retained"}})
				hc := &http.Client{Transport: transport, Jar: jar}
				executor := newTestClient(t, hc, testTargets(t, "http://upstream.test", func(target *config.S3Target) { target.Timeout = time.Hour }), testBudget())
				source, err := http.NewRequest("PUT", "http://proxy.test/visible/key", strings.NewReader("payload"))
				if err != nil {
					t.Fatal(err)
				}
				getBody := source.GetBody
				source.GetBody = func() (io.ReadCloser, error) { replays++; return getBody() }
				resp, err := executor.Do(context.Background(), Request{Operation: s3ops.OpPutObject, Target: "primary", Bucket: "bucket", Key: "key", Source: source})
				if resp != nil {
					resp.Body.Close()
				}
				if err == nil || resp != nil || !strings.Contains(err.Error(), "upstream redirect rejected") {
					t.Error("expected controlled redirect rejection")
				}
				for cause := err; cause != nil; cause = errors.Unwrap(cause) {
					if strings.Contains(cause.Error(), "REDIRECT_") {
						t.Error("redirect data leaked through error chain")
					}
				}
				if calls != 1 || replays != 1 || responseBody.closes != 1 || responseBody.reads != 0 {
					t.Errorf("calls=%d replays=%d closes=%d reads=%d", calls, replays, responseBody.closes, responseBody.reads)
				}
				if requestContext == nil || !errors.Is(requestContext.Err(), context.Canceled) {
					t.Error("target cancellation not released")
				}
				owned := executor.(*client).httpClient
				if owned == hc || owned.Jar != jar || owned.Timeout != hc.Timeout || hc.CheckRedirect != nil || reflect.ValueOf(hc.Transport).Pointer() != reflect.ValueOf(transport).Pointer() {
					t.Error("caller client ownership/settings changed")
				}
			})
		}
	}
}
