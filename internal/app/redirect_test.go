package app

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	v4 "github.com/aws/aws-sdk-go-v2/aws/signer/v4"
)

func TestBuildRejectsUpstreamRedirects(t *testing.T) {
	for _, status := range []int{301, 302, 303, 307, 308} {
		for _, method := range []string{"GET", "PUT", "DELETE"} {
			for _, destination := range []string{"bucket", "cross-host", "malformed"} {
				t.Run(fmt.Sprintf("%d/%s/%s", status, method, destination), func(t *testing.T) {
					var calls, received atomic.Int32
					receiver := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { received.Add(1) }))
					defer receiver.Close()
					upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
						calls.Add(1)
						io.Copy(io.Discard, r.Body)
						if r.URL.Path != "/bucket/key" {
							return
						}
						location := "/bucket"
						if destination != "bucket" {
							location = strings.Replace(receiver.URL, "http://", "http://user:REDIRECT_PASSWORD@", 1) + "/REDIRECT_KEY?token=REDIRECT_QUERY"
							if destination == "malformed" {
								location = strings.Replace(location, "/REDIRECT_KEY", "/%GG/REDIRECT_KEY", 1)
							}
						}
						w.Header().Set("Location", location)
						w.WriteHeader(status)
						io.WriteString(w, "REDIRECT_BODY")
					}))
					defer upstream.Close()
					op := map[string]string{"GET": "GetObject", "PUT": "PutObject", "DELETE": "DeleteObject"}[method]
					src := strings.NewReplacer("${ADDRESS}", "127.0.0.1:0", "http://127.0.0.1:9000", upstream.URL, `prefix = "/"`, `prefix = "/bucket"`, `operations   = ["GetObject"]`, `operations   = ["`+op+`"]`, `mode = "none"`, `mode = "sigv4_static"
  client "restricted" {
    access_key = "client-ak"
    secret_key = "client-sk" # pragma: allowlist secret
    allow_routes = ["all"]
    allow_ops = ["`+op+`"]
  }`).Replace(validConfig)
					path := filepath.Join(t.TempDir(), "config.hcl")
					if err := os.WriteFile(path, []byte(src), 0o600); err != nil {
						t.Fatal(err)
					}
					var logs bytes.Buffer
					a, err := Build(BuildOptions{ConfigPath: path, Logger: slog.New(slog.NewTextHandler(&logs, nil))})
					if err != nil {
						t.Fatal(err)
					}
					defer a.transport.CloseIdleConnections()
					proxy := httptest.NewServer(a.server.Handler)
					defer proxy.Close()
					var body io.Reader
					if method == "PUT" {
						body = strings.NewReader("payload")
					}
					req, err := http.NewRequest(method, proxy.URL+"/bucket/key", body)
					if err != nil {
						t.Fatal(err)
					}
					req.Header.Set("X-Amz-Content-Sha256", "UNSIGNED-PAYLOAD")
					signer := v4.NewSigner(func(o *v4.SignerOptions) { o.DisableURIPathEscaping = true })
					if err := signer.SignHTTP(context.Background(), aws.Credentials{AccessKeyID: "client-ak", SecretAccessKey: "client-sk"}, req, "UNSIGNED-PAYLOAD", "s3", "us-east-1", time.Now()); err != nil {
						t.Fatal(err)
					}
					client := proxy.Client()
					client.CheckRedirect = func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }
					resp, err := client.Do(req)
					if err != nil {
						t.Fatal(err)
					}
					data, err := io.ReadAll(resp.Body)
					resp.Body.Close()
					proxy.Close()
					if err != nil {
						t.Fatal(err)
					}
					if resp.StatusCode != http.StatusBadGateway || !strings.Contains(string(data), "<Code>InternalError</Code>") || resp.Header.Get("Location") != "" {
						t.Error("expected 502 InternalError without Location")
					}
					if calls.Load() != 1 || received.Load() != 0 {
						t.Errorf("upstream=%d receiver=%d", calls.Load(), received.Load())
					}
					if strings.Contains(string(data)+logs.String(), "REDIRECT_") {
						t.Error("redirect data leaked to XML/logs")
					}
					if !strings.Contains(logs.String(), "dispatch failed") || !strings.Contains(logs.String(), "request complete") {
						t.Error("missing failure/completion telemetry")
					}
				})
			}
		}
	}
}
