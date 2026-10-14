package app

import (
	"bytes"
	"context"
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
	"sync/atomic"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	v4 "github.com/aws/aws-sdk-go-v2/aws/signer/v4"
	"github.com/egose/s3proxy/internal/testutil/s3signature"
)

func TestBuild_BackendSessionTokenWorkflowHTTP(t *testing.T) {
	for _, mode := range []string{"none", "header", "presign"} {
		for _, route := range []struct {
			name         string
			destinations []string
		}{
			{"temporary", []string{"primary"}},
			{"tokenless", []string{"longlived"}},
			{"fanout", []string{"primary", "replica", "longlived"}},
		} {
			t.Run(mode+"/"+route.name, func(t *testing.T) {
				credentials := map[string]aws.Credentials{
					"primary":   {AccessKeyID: "CRED_PRIVATE_primary-ak", SecretAccessKey: "CRED_PRIVATE_primary-sk", SessionToken: "CRED_PRIVATE_primary-token/+=value"},
					"replica":   {AccessKeyID: "CRED_PRIVATE_replica-ak", SecretAccessKey: "CRED_PRIVATE_replica-sk", SessionToken: "CRED_PRIVATE_replica-token/+=value"},
					"longlived": {AccessKeyID: "CRED_PRIVATE_longlived-ak", SecretAccessKey: "CRED_PRIVATE_longlived-sk"},
				}
				t.Setenv("S3PROXY_WORKFLOW_SESSION_TOKEN", credentials["primary"].SessionToken)
				type capture struct {
					target  string
					request *http.Request
					body    []byte
				}
				captures := make(chan capture, 16)
				endpoints := make(map[string]string)
				for _, name := range []string{"primary", "replica", "longlived"} {
					upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
						body, err := io.ReadAll(r.Body)
						if err != nil {
							t.Error("upstream could not read request body")
						}
						captures <- capture{target: name, request: r.Clone(context.Background()), body: body}
						if strings.HasSuffix(r.URL.Path, "/failure") {
							w.WriteHeader(http.StatusServiceUnavailable)
							_, _ = io.WriteString(w, "<Error><Code>ServiceUnavailable</Code></Error>")
							return
						}
						if r.Method == http.MethodGet {
							_, _ = io.WriteString(w, "payload")
						}
					}))
					defer upstream.Close()
					endpoints[name] = upstream.URL + "/base%2Fpath"
				}
				source := sessionTokenWorkflowConfig(mode, route.destinations, endpoints, credentials)
				path := filepath.Join(t.TempDir(), "session.hcl")
				if err := os.WriteFile(path, []byte(source), 0o600); err != nil {
					t.Fatal("could not write workflow config")
				}
				var logs bytes.Buffer
				a, err := Build(BuildOptions{ConfigPath: path, Logger: slog.New(slog.NewJSONHandler(&logs, nil))})
				if err != nil {
					t.Fatal("app rejected backend session credential configuration")
				}
				defer a.transport.CloseIdleConnections()
				t.Setenv("S3PROXY_WORKFLOW_SESSION_TOKEN", "CRED_PRIVATE_replacement-token")
				proxy := httptest.NewServer(a.server.Handler)
				defer proxy.Close()
				client := proxy.Client()
				client.Timeout = 5 * time.Second
				for _, tc := range []struct {
					name, method, key string
					status            int
				}{
					{"upload", http.MethodPut, "lower%2fseparator//a/../space%20key%25", http.StatusOK},
					{"read", http.MethodGet, "lower%2fseparator//a/../space%20key%25", http.StatusOK},
					{"upstream failure", http.MethodGet, "failure", http.StatusServiceUnavailable},
					{"invalid signature", http.MethodGet, "key", http.StatusForbidden},
				} {
					if mode == "none" && tc.name == "invalid signature" {
						continue
					}
					var body io.Reader
					if tc.method == http.MethodPut {
						body = strings.NewReader("payload")
					}
					r, err := http.NewRequest(tc.method, proxy.URL+"/visible/"+tc.key, body)
					if err != nil {
						t.Fatal("could not create workflow request")
					}
					r.URL.RawQuery = url.Values{"X-Amz-Security-Token": {"CRED_PRIVATE_inbound-query-token"}}.Encode()
					r.Header.Set("X-Amz-Security-Token", "CRED_PRIVATE_inbound-header-token")
					r.Header.Set("X-Amz-Content-Sha256", "UNSIGNED-PAYLOAD")
					signer := v4.NewSigner(func(o *v4.SignerOptions) { o.DisableURIPathEscaping = true })
					clientCredentials := aws.Credentials{AccessKeyID: "CRED_PRIVATE_client-ak", SecretAccessKey: "CRED_PRIVATE_client-sk"}
					switch mode {
					case "header":
						if err := signer.SignHTTP(context.Background(), clientCredentials, r, "UNSIGNED-PAYLOAD", "s3", "us-east-1", time.Now().UTC()); err != nil {
							t.Fatal("could not sign inbound request")
						}
						if tc.name == "invalid signature" {
							r.Header.Set("Authorization", r.Header.Get("Authorization")+"0")
						}
					case "presign":
						r.Header.Del("X-Amz-Security-Token")
						query := r.URL.Query()
						query.Set("X-Amz-Expires", "600")
						r.URL.RawQuery = query.Encode()
						uri, _, err := signer.PresignHTTP(context.Background(), clientCredentials, r, "UNSIGNED-PAYLOAD", "s3", "us-east-1", time.Now().UTC())
						if err != nil {
							t.Fatal("could not presign inbound request")
						}
						r.URL, err = url.Parse(uri)
						if err != nil {
							t.Fatal("could not parse presigned fixture")
						}
						r.Header.Set("X-Amz-Security-Token", "CRED_PRIVATE_inbound-header-token")
						if tc.name == "invalid signature" {
							query = r.URL.Query()
							query.Set("X-Amz-Signature", strings.Repeat("0", 64))
							r.URL.RawQuery = query.Encode()
						}
					}
					resp, err := client.Do(r)
					if err != nil {
						t.Fatal("workflow HTTP request failed")
					}
					data, err := io.ReadAll(resp.Body)
					resp.Body.Close()
					if err != nil || resp.StatusCode != tc.status {
						t.Fatalf("%s: unexpected HTTP response (status %d)", tc.name, resp.StatusCode)
					}
					if strings.Contains(string(data), "CRED_PRIVATE") || strings.Contains(fmt.Sprint(resp.Header), "CRED_PRIVATE") {
						t.Fatal("workflow response exposed credentials")
					}
					if tc.name == "read" && string(data) != "payload" {
						t.Fatal("workflow read lost upstream object bytes")
					}
					wantTargets := route.destinations[:1]
					if tc.method == http.MethodPut {
						wantTargets = route.destinations
					}
					if tc.name == "invalid signature" {
						wantTargets = nil
						if !strings.Contains(string(data), "<Code>SignatureDoesNotMatch</Code>") {
							t.Error("inbound authentication error mapping changed")
						}
					}
					if len(captures) != len(wantTargets) {
						t.Fatalf("%s: upstream calls = %d, want %d", tc.name, len(captures), len(wantTargets))
					}
					seen := make(map[string]bool)
					for range wantTargets {
						got := <-captures
						if seen[got.target] {
							t.Error("workflow repeated a destination")
						}
						seen[got.target] = true
						out := got.request
						cred := credentials[got.target]
						headers := "host;x-amz-content-sha256;x-amz-date"
						if tc.method == http.MethodPut {
							headers = "content-length;" + headers
							if out.Header.Get("Content-Length") != "7" || string(got.body) != "payload" {
								t.Error("workflow upload length or body changed")
							}
						}
						if cred.SessionToken != "" {
							headers += ";x-amz-security-token"
							if values := out.Header.Values("X-Amz-Security-Token"); len(values) != 1 || values[0] != cred.SessionToken {
								t.Error("workflow did not send exactly the target's startup token")
							}
						} else if len(out.Header.Values("X-Amz-Security-Token")) != 0 {
							t.Error("workflow leaked a token to a long-lived target")
						}
						if mode == "none" {
							headers = "accept-encoding;" + headers
						}
						if out.URL.RawQuery != "" || out.URL.EscapedPath() != "/base%2Fpath/physical/assets/"+tc.key {
							t.Error("workflow leaked inbound query authentication or changed the S3 path")
						}
						if out.Header.Get("X-Amz-Content-Sha256") != "UNSIGNED-PAYLOAD" || len(out.Header.Get("X-Amz-Date")) != 16 {
							t.Fatal("workflow emitted an invalid signing date or payload hash")
						}
						if out.Header.Get("Authorization") != s3signature.Authorization(out, cred.AccessKeyID, cred.SecretAccessKey, "us-east-1", headers) {
							t.Error("workflow signature differs from the independent S3 oracle")
						}
					}
					for _, target := range wantTargets {
						if !seen[target] {
							t.Error("workflow reached the wrong destination")
						}
					}
				}
				proxy.Close()
				if strings.Contains(logs.String(), "CRED_PRIVATE") {
					t.Error("runtime logs exposed a credential or inbound token")
				}
				for _, message := range []string{"config loaded", "request complete"} {
					if !strings.Contains(logs.String(), message) {
						t.Errorf("runtime logs lost %q telemetry", message)
					}
				}
				if len(route.destinations) > 1 && !strings.Contains(logs.String(), "destination attempt") {
					t.Error("runtime logs lost fan-out destination telemetry")
				}
				if mode != "none" && !strings.Contains(logs.String(), "auth failed") {
					t.Error("runtime logs lost authentication failure telemetry")
				}
			})
		}
	}
}

func TestBuild_BackendSessionTokenTransportDiagnosticsHTTP(t *testing.T) {
	var calls atomic.Int64
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
	}))
	defer upstream.Close()
	t.Setenv("S3PROXY_WORKFLOW_SESSION_TOKEN", "CRED_PRIVATE_\"quoted\r\n\\雪")
	endpoints := make(map[string]string)
	credentials := make(map[string]aws.Credentials)
	for _, name := range []string{"primary", "replica", "longlived"} {
		endpoints[name] = upstream.URL
		credentials[name] = aws.Credentials{AccessKeyID: "CRED_PRIVATE_backend-ak", SecretAccessKey: "CRED_PRIVATE_backend-sk"}
	}
	source := sessionTokenWorkflowConfig("none", []string{"primary"}, endpoints, credentials)
	path := filepath.Join(t.TempDir(), "session.hcl")
	if err := os.WriteFile(path, []byte(source), 0o600); err != nil {
		t.Fatal("could not write session credential fixture")
	}
	var logs bytes.Buffer
	a, err := Build(BuildOptions{ConfigPath: path, Logger: slog.New(slog.NewJSONHandler(&logs, nil))})
	if err != nil {
		t.Fatal("app rejected a literal environment token during config load")
	}
	defer a.transport.CloseIdleConnections()
	proxy := httptest.NewServer(a.server.Handler)
	defer proxy.Close()
	r, err := http.NewRequest(http.MethodGet, proxy.URL+"/visible/key", nil)
	if err != nil {
		t.Fatal("could not create diagnostic request")
	}
	r.Header.Set("X-Amz-Security-Token", "CRED_PRIVATE_inbound-header")
	r.URL.RawQuery = url.Values{"X-Amz-Security-Token": {"CRED_PRIVATE_inbound-query"}}.Encode()
	resp, err := proxy.Client().Do(r)
	if err != nil {
		t.Fatal("diagnostic HTTP request failed")
	}
	data, err := io.ReadAll(resp.Body)
	resp.Body.Close()
	proxy.Close()
	if err != nil || resp.StatusCode != http.StatusBadGateway || !strings.Contains(string(data), "<Code>InternalError</Code>") || calls.Load() != 0 {
		t.Error("invalid token header must fail at transport before reaching the backend")
	}
	for _, fragment := range []string{"CRED_PRIVATE", "quoted", "雪"} {
		if strings.Contains(string(data)+logs.String(), fragment) {
			t.Error("transport failure exposed a configured credential or inbound token")
		}
	}
	for _, message := range []string{"config loaded", "dispatch failed", "request complete", "X-Amz-Security-Token"} {
		if !strings.Contains(logs.String(), message) {
			t.Errorf("transport failure logs lost %q context", message)
		}
	}
}

func sessionTokenWorkflowConfig(mode string, destinations []string, endpoints map[string]string, credentials map[string]aws.Credentials) string {
	client := ""
	if mode != "none" {
		mode = "sigv4_static"
		client = `client "caller" {
    access_key = "CRED_PRIVATE_client-ak"
    secret_key = "CRED_PRIVATE_client-sk"
    allow_routes = ["objects"]
    allow_ops = ["GetObject", "PutObject"]
  }`
	}
	var source strings.Builder
	fmt.Fprintf(&source, `
listener "http" "public" {
  address = "127.0.0.1:0"
  addressing { path_style = true }
}
auth "main" {
  mode = %q
  %s
}
`, mode, client)
	for _, name := range []string{"primary", "replica", "longlived"} {
		cred := credentials[name]
		token := ""
		if name == "primary" {
			token = `session_token = env("S3PROXY_WORKFLOW_SESSION_TOKEN")`
		} else if cred.SessionToken != "" {
			token = "session_token = " + strconv.Quote(cred.SessionToken)
		}
		fmt.Fprintf(&source, `
credential "static" %q {
  access_key = %q
  secret_key = %q
  %s
}
target "s3" %q {
  endpoint = %q
  region = "us-east-1"
  force_path_style = true
  credentials = %q
}
`, name, cred.AccessKeyID, cred.SecretAccessKey, token, name, endpoints[name], name)
	}
	quoted := make([]string, len(destinations))
	for i, name := range destinations {
		quoted[i] = strconv.Quote(name)
	}
	fmt.Fprintf(&source, `
parser "bucket_exact" "visible" { bucket = "visible" }
route "objects" {
  parser = "visible"
  operations = ["GetObject", "PutObject"]
  destinations = [%s]
  dispatch = "all"
  on_match = "stop"
  rewrite {
    bucket = "physical"
    prepend_key_prefix = "assets/"
  }
}
`, strings.Join(quoted, ", "))
	return source.String()
}
