package app

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/json"
	"encoding/xml"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	v4 "github.com/aws/aws-sdk-go-v2/aws/signer/v4"
)

func TestBuild_RewriteExecutionDiagnosticsHTTP(t *testing.T) {
	for _, tc := range []struct{ name, source string }{
		{"missing capture", `REWRITELOG_PARTIAL_OUTPUT/{{ .Captures.REWRITELOG_MISSING_CAPTURE }}/{{ .Key }}`},
		{"failing literal action", `REWRITELOG_PARTIAL_OUTPUT/{{ index "REWRITELOG_ACTION_LITERAL\"雪\\value" 9999 }}`},
	} {
		for _, origin := range []string{"literal", "env"} {
			for _, mode := range []string{"none", "header", "presign"} {
				for _, continued := range []bool{false, true} {
					t.Run(fmt.Sprintf("%s/%s/%s/continued=%t", tc.name, origin, mode, continued), func(t *testing.T) {
						var calls, reads atomic.Int64
						var mu sync.Mutex
						objects := map[string]string{"/earlier/acme/key": "earlier-original", "/physical/acme/key": "original"}
						upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
							calls.Add(1)
							data, err := io.ReadAll(r.Body)
							if err != nil {
								t.Error("backend could not read fixture payload")
								w.WriteHeader(http.StatusBadRequest)
								return
							}
							mu.Lock()
							objects[r.URL.EscapedPath()] = string(data)
							mu.Unlock()
							w.WriteHeader(http.StatusOK)
						}))
						defer upstream.Close()
						expression := strconv.Quote(tc.source)
						if origin == "env" {
							t.Setenv("S3PROXY_REWRITELOG_TEST_TEMPLATE", tc.source)
							expression = `env("S3PROXY_REWRITELOG_TEST_TEMPLATE")`
						}
						path := filepath.Join(t.TempDir(), "rewrite.hcl")
						source := rewriteDiagnosticsConfig(upstream.URL, mode, expression, continued)
						if err := os.WriteFile(path, []byte(source), 0o600); err != nil {
							t.Fatal("could not write rewrite fixture config")
						}
						var logs bytes.Buffer
						a, err := Build(BuildOptions{ConfigPath: path, Logger: slog.New(slog.NewJSONHandler(&logs, nil))})
						if err != nil {
							t.Fatal("app rejected a valid runtime-failure template during configuration")
						}
						defer a.transport.CloseIdleConnections()
						proxy := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
							r.Body = rewriteDiagnosticBody{ReadCloser: r.Body, reads: &reads}
							a.server.Handler.ServeHTTP(w, r)
						}))
						defer proxy.Close()
						client := proxy.Client()
						client.Timeout = 5 * time.Second
						const requestID = "rewrite-diagnostics"
						const payload = "replacement"
						r, err := http.NewRequest(http.MethodPut, proxy.URL+"/tenant-acme-logs/key", strings.NewReader(payload))
						if err != nil {
							t.Fatal("could not construct rewrite fixture request")
						}
						r.Header.Set("X-Request-Id", requestID)
						signRewriteDiagnosticRequest(t, r, mode, payload)
						resp, err := client.Do(r)
						if err != nil {
							t.Fatal("rewrite fixture HTTP request failed")
						}
						data, err := io.ReadAll(resp.Body)
						resp.Body.Close()
						proxy.Close()
						if err != nil {
							t.Fatal("could not read rewrite fixture response")
						}
						var response struct {
							XMLName   xml.Name `xml:"Error"`
							Code      string   `xml:"Code"`
							Message   string   `xml:"Message"`
							RequestID string   `xml:"RequestId"`
						}
						if err := xml.Unmarshal(data, &response); err != nil {
							t.Error("rewrite response was not valid S3 error XML")
						}
						if resp.StatusCode != http.StatusInternalServerError || response.Code != "InternalError" || response.Message != "We encountered an internal error. Please try again." {
							t.Error("execution failure did not retain the InternalError HTTP/XML mapping")
						}
						if resp.Header.Get("X-Request-Id") != requestID || response.RequestID != requestID {
							t.Error("rewrite response lost the request ID")
						}
						if calls.Load() != 0 {
							t.Errorf("backend calls = %d, want zero across all matched routes", calls.Load())
						}
						mu.Lock()
						if len(objects) != 2 || objects["/earlier/acme/key"] != "earlier-original" || objects["/physical/acme/key"] != "original" {
							t.Error("failed rewrite changed backend objects")
						}
						mu.Unlock()
						if mode == "none" && reads.Load() != 0 {
							t.Error("rewrite failure read the body before dispatch-related replay")
						}
						if mode != "none" && reads.Load() == 0 {
							t.Error("rewrite failure bypassed authenticated payload verification")
						}
						if strings.Contains(logs.String()+string(data)+fmt.Sprint(resp.Header), "REWRITELOG_") {
							t.Error("HTTP diagnostics exposed private template or credential input")
						}
						assertRewriteHTTPLogs(t, logs.String(), requestID, len(data), continued)
					})
				}
			}
		}
	}
}

func assertRewriteHTTPLogs(t *testing.T, logs, requestID string, responseBytes int, continued bool) {
	t.Helper()
	decoder := json.NewDecoder(strings.NewReader(logs))
	var records []map[string]any
	for {
		var record map[string]any
		err := decoder.Decode(&record)
		if err == io.EOF {
			break
		}
		if err != nil {
			t.Fatal("captured rewrite log was not valid JSON")
		}
		records = append(records, record)
	}
	if len(records) != 3 {
		t.Fatalf("log records = %d, want configuration, rewrite failure, and request completion", len(records))
	}
	routes := float64(1)
	if continued {
		routes = 2
	}
	if records[0]["msg"] != "config loaded" || records[0]["routes"] != routes {
		t.Error("fixture did not load the expected route configuration")
	}
	failure, completion := records[1], records[2]
	if failure["msg"] != "rewrite failed" || failure["level"] != "ERROR" || failure["route"] != "objects" || failure["error"] != "key_template execution failed" {
		t.Error("rewrite log lost the route or fixed execution failure category")
	}
	if completion["msg"] != "request complete" || completion["level"] != "INFO" || completion["method"] != http.MethodPut || completion["status"] != float64(http.StatusInternalServerError) || completion["bytes"] != float64(responseBytes) {
		t.Error("rewrite log lost request completion metadata")
	}
	if duration, ok := completion["duration_ms"].(float64); !ok || duration < 0 {
		t.Error("rewrite log lost the request duration")
	}
	if failure["request_id"] != requestID || completion["request_id"] != requestID {
		t.Error("rewrite log lost the request ID")
	}
}

func signRewriteDiagnosticRequest(t *testing.T, r *http.Request, mode, payload string) {
	t.Helper()
	if mode == "none" {
		return
	}
	hash := fmt.Sprintf("%x", sha256.Sum256([]byte(payload)))
	r.Header.Set("X-Amz-Content-Sha256", hash)
	signer := v4.NewSigner(func(o *v4.SignerOptions) { o.DisableURIPathEscaping = true })
	credentials := aws.Credentials{AccessKeyID: "REWRITELOG_CLIENT_ACCESS", SecretAccessKey: "REWRITELOG_CLIENT_SECRET"}
	if mode == "header" {
		if err := signer.SignHTTP(context.Background(), credentials, r, hash, "s3", "us-east-1", time.Now().UTC()); err != nil {
			t.Fatal("could not sign rewrite fixture request")
		}
		return
	}
	r.URL.RawQuery = "X-Amz-Expires=600"
	uri, _, err := signer.PresignHTTP(context.Background(), credentials, r, hash, "s3", "us-east-1", time.Now().UTC())
	if err != nil {
		t.Fatal("could not presign rewrite fixture request")
	}
	r.URL, err = url.Parse(uri)
	if err != nil {
		t.Fatal("could not parse presigned rewrite fixture request")
	}
}

func rewriteDiagnosticsConfig(endpoint, mode, expression string, continued bool) string {
	client := ""
	if mode != "none" {
		mode = "sigv4_static"
		client = `client "caller" {
    access_key = "REWRITELOG_CLIENT_ACCESS"
    secret_key = "REWRITELOG_CLIENT_SECRET"
    allow_routes = ["*"]
    allow_ops = ["PutObject"]
  }`
	}
	earlier := ""
	if continued {
		earlier = `route "earlier" {
  parser = "visible"
  operations = ["PutObject"]
  destinations = ["store"]
  dispatch = "first"
  on_match = "continue"
  rewrite {
    bucket = "earlier"
    key_template = "{{ .Captures.tenant }}/{{ .Key }}"
  }
}`
	}
	return fmt.Sprintf(`
listener "http" "public" {
  address = "127.0.0.1:0"
  addressing { path_style = true }
}
auth "main" {
  mode = %q
  %s
}
credential "static" "backend" {
  access_key = "REWRITELOG_BACKEND_ACCESS"
  secret_key = "REWRITELOG_BACKEND_SECRET"
}
target "s3" "store" {
  endpoint = %q
  region = "us-east-1"
  force_path_style = true
  credentials = "backend"
}
parser "bucket_regex" "visible" {
  pattern = "^tenant-(?P<tenant>[a-z]+)-logs$"
}
%s
route "objects" {
  parser = "visible"
  operations = ["PutObject"]
  destinations = ["store"]
  dispatch = "first"
  on_match = "stop"
  rewrite {
    bucket = "physical"
    key_template = %s
  }
}
`, mode, client, endpoint, earlier, expression)
}

type rewriteDiagnosticBody struct {
	io.ReadCloser
	reads *atomic.Int64
}

func (b rewriteDiagnosticBody) Read(p []byte) (int, error) {
	b.reads.Add(1)
	return b.ReadCloser.Read(p)
}
