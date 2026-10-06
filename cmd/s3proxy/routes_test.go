package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"strconv"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

func TestRoutesCLI(t *testing.T) {
	binary := filepath.Join(t.TempDir(), "s3proxy")
	build := exec.Command("go", "build", "-o", binary, ".")
	if output, err := build.CombinedOutput(); err != nil {
		t.Fatalf("build CLI: %v\n%s", err, output)
	}
	var connections atomic.Int64
	backend := httptest.NewUnstartedServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusServiceUnavailable)
	}))
	backend.Config.ConnState = func(_ net.Conn, state http.ConnState) {
		if state == http.StateNew {
			connections.Add(1)
		}
	}
	backend.Start()
	defer backend.Close()
	occupied, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer occupied.Close()
	values := map[string]string{
		"ADDRESS": occupied.Addr().String(),
		"ENDPOINT": strings.Replace(backend.URL, "http://", "http://PRIVATE_user:PRIVATE_password@", 1) +
			"/PRIVATE_path?PRIVATE_query=PRIVATE_value#PRIVATE_fragment",
		"TEXT":       "PRIVATE_literal\"quoted\n\\雪",
		"REGION":     "PRIVATE_region",
		"HOST":       "PRIVATE_host.example.test",
		"PREFIX":     "/PRIVATE_path",
		"BUCKET":     "PRIVATE_bucket",
		"REGEX":      "^PRIVATE_(?P<tenant>[a-z]+)$",
		"TEMPLATE":   "PRIVATE_template/{{ .Key }}",
		"PARSER_REF": "PRIVATE_parser_prefix.path",
		"TARGET_REF": "PRIVATE_target_prefix.replica",
		"CRED_REF":   "PRIVATE_credential_prefix.primary",
		"ROUTE_REF":  "PRIVATE_route_prefix.z_write",
	}
	for name, value := range values {
		t.Setenv("ROUTES_PRIVATE_"+name, value)
	}
	run := func(t *testing.T, args ...string) (string, string, error) {
		t.Helper()
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		cmd := exec.CommandContext(ctx, binary, append([]string{"routes"}, args...)...)
		var stdout, stderr bytes.Buffer
		cmd.Stdout, cmd.Stderr = &stdout, &stderr
		err := cmd.Run()
		if ctx.Err() != nil {
			t.Fatal("offline routes command did not terminate")
		}
		assertRoutesConfidential(t, stdout.String()+stderr.String())
		return stdout.String(), stderr.String(), err
	}
	for _, literal := range []bool{false, true} {
		t.Run("literal="+strconv.FormatBool(literal), func(t *testing.T) {
			source := routesConfig
			if literal {
				for name, value := range values {
					source = strings.ReplaceAll(source, `env("ROUTES_PRIVATE_`+name+`")`, strconv.Quote(value))
				}
			}
			path := writeRoutesConfig(t, source)
			var previous string
			for i, flag := range []string{"--config", "-c", "--config"} {
				stdout, stderr, err := run(t, flag, path)
				if err != nil || stderr != "" {
					t.Fatal("valid routes command failed or emitted diagnostics")
				}
				assertTopologyJSON(t, stdout, expectedRoutesJSON)
				if i > 0 && stdout != previous {
					t.Fatal("topology output was not byte-identical")
				}
				previous = stdout
			}
		})
	}
	t.Run("non-bindable listener", func(t *testing.T) {
		t.Setenv("ROUTES_PRIVATE_ADDRESS", "192.0.2.1:8080")
		stdout, stderr, err := run(t, "-c", writeRoutesConfig(t, routesConfig))
		if err != nil || stderr != "" {
			t.Fatal("routes tried to use a listener address")
		}
		assertTopologyJSON(t, stdout, expectedRoutesJSON)
	})
	t.Run("validated finite environment fields", func(t *testing.T) {
		t.Setenv("ROUTES_PRIVATE_DISPATCH", "all")
		t.Setenv("ROUTES_PRIVATE_MATCH", "continue")
		t.Setenv("ROUTES_PRIVATE_OPERATION", "PutObject")
		t.Setenv("ROUTES_PRIVATE_DEFAULT", "")
		source := strings.NewReplacer(
			`dispatch = "all"`, `dispatch = env("ROUTES_PRIVATE_DISPATCH")`,
			`on_match = "continue"`, "on_match = env(\"ROUTES_PRIVATE_MATCH\")\n read_preference = env(\"ROUTES_PRIVATE_DEFAULT\")",
			`["DeleteObject", "PutObject"]`, `["DeleteObject", env("ROUTES_PRIVATE_OPERATION")]`,
		).Replace(routesConfig)
		stdout, stderr, err := run(t, "-c", writeRoutesConfig(t, source))
		if err != nil || stderr != "" {
			t.Fatal("validated finite fields or empty read preference default failed")
		}
		assertTopologyJSON(t, stdout, expectedRoutesJSON)
	})
	for _, tc := range []struct{ name, old, replacement, value, want string }{
		{"endpoint", `env("ROUTES_PRIVATE_ENDPOINT")`, `env("ROUTES_PRIVATE_INVALID")`, "http://PRIVATE_user:PRIVATE_password@example.test/%GG?PRIVATE_query", "invalid endpoint"}, // pragma: allowlist secret
		{"duration", `region = env("ROUTES_PRIVATE_REGION")`, "region = \"region\"\n timeout = env(\"ROUTES_PRIVATE_INVALID\")", "PRIVATE_duration", "invalid timeout"},
		{"listener duration", `address = env("ROUTES_PRIVATE_ADDRESS")`, "address = \":0\"\n timeouts { read = env(\"ROUTES_PRIVATE_INVALID\") }", "PRIVATE_duration", "invalid read timeout"},
		{"regex", `env("ROUTES_PRIVATE_REGEX")`, `env("ROUTES_PRIVATE_INVALID")`, "(?P<PRIVATE_\"雪>x)", "invalid pattern"},
		{"template", `env("ROUTES_PRIVATE_TEMPLATE")`, `env("ROUTES_PRIVATE_INVALID")`, "{{ PRIVATE_function }}", "invalid key_template"},
		{"parser ref", `env("ROUTES_PRIVATE_PARSER_REF")`, `env("ROUTES_PRIVATE_INVALID")`, "parser.PRIVATE_unknown", "unknown parser"},
		{"destination ref", `env("ROUTES_PRIVATE_TARGET_REF")`, `env("ROUTES_PRIVATE_INVALID")`, "target.s3.PRIVATE_unknown", "unknown destination"},
		{"credential ref", `env("ROUTES_PRIVATE_CRED_REF")`, `env("ROUTES_PRIVATE_INVALID")`, "credential.static.PRIVATE_unknown", "unknown credential"},
		{"policy ref", `allow_routes = [env("ROUTES_PRIVATE_ROUTE_REF")]`, `allow_routes = [env("ROUTES_PRIVATE_INVALID")]`, "route.PRIVATE_unknown", "allow_routes references unknown route"},
		{"bucket ref", `route = env("ROUTES_PRIVATE_ROUTE_REF")`, `route = env("ROUTES_PRIVATE_INVALID")`, "route.PRIVATE_unknown", "unknown route"},
		{"dispatch", `dispatch = "all"`, `dispatch = env("ROUTES_PRIVATE_INVALID")`, "PRIVATE_enum", "invalid dispatch"},
		{"match", `on_match = "continue"`, `on_match = env("ROUTES_PRIVATE_INVALID")`, "PRIVATE_enum", "invalid on_match"},
		{"preference", `read_preference = "hash"`, `read_preference = env("ROUTES_PRIVATE_INVALID")`, "PRIVATE_enum", "invalid read_preference"},
		{"operation", `["HeadObject", "GetObject"]`, `[env("ROUTES_PRIVATE_INVALID")]`, "PRIVATE_operation", "unsupported operation"},
		{"auth", `mode = "sigv4_static"`, `mode = env("ROUTES_PRIVATE_INVALID")`, "PRIVATE_mode", "invalid mode"},
		{"HCL evaluated key", `secret_key = env("ROUTES_PRIVATE_TEXT")`, `secret_key = {for v in [1, 2] : "derived-${env("ROUTES_PRIVATE_INVALID")}" => v}`, "PRIVATE_\"quoted\n\\雪", "Duplicate object key"},
		{"HCL type", `path_style = true`, `path_style = env("ROUTES_PRIVATE_INVALID")`, "PRIVATE_\"quoted\n\\雪", "Unsuitable value type"},
		{"empty operations", `["HeadObject", "GetObject"]`, `[]`, "", "at least one operation"},
		{"omitted operations", `operations = ["HeadObject", "GetObject"]`, ``, "", "Missing required argument"},
		{"wildcard operations", `["HeadObject", "GetObject"]`, `["*"]`, "", "unsupported operation"},
		{"duplicate operations", `["HeadObject", "GetObject"]`, `["GetObject", "GetObject"]`, "", "operations contain duplicate"},
		{"duplicate destinations", `[env("ROUTES_PRIVATE_TARGET_REF"), "target.s3.primary"]`, `[env("ROUTES_PRIVATE_TARGET_REF"), "target.s3.replica"]`, "", "destinations contain duplicate"},
		{"missing credential", `access_key = env("ROUTES_PRIVATE_TEXT")`, `access_key = env("ROUTES_PRIVATE_INVALID")`, "", "empty access_key"},
		{"parse", `path_style = true`, "path_style = @", "", "parse config"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Setenv("ROUTES_PRIVATE_INVALID", tc.value)
			source := strings.Replace(routesConfig, tc.old, tc.replacement, 1)
			if source == routesConfig {
				t.Fatal("fixture did not replace a field")
			}
			stdout, stderr, err := run(t, "--config", writeRoutesConfig(t, source))
			var exit *exec.ExitError
			if !errors.As(err, &exit) || exit.ExitCode() != 1 || stdout != "" || !strings.Contains(stderr, tc.want) {
				t.Fatal("invalid config must exit 1 with no report and a useful safe error")
			}
			if tc.value != "" {
				literal := strings.ReplaceAll(source, `env("ROUTES_PRIVATE_INVALID")`, strconv.Quote(tc.value))
				stdout, stderr, err = run(t, "-c", writeRoutesConfig(t, literal))
				if !errors.As(err, &exit) || exit.ExitCode() != 1 || stdout != "" || !strings.Contains(stderr, tc.want) {
					t.Fatal("invalid literal attribute must also fail without exposing its value")
				}
			}
		})
	}
	path := writeRoutesConfig(t, routesConfig)
	t.Run("real stdout failure", func(t *testing.T) {
		full, err := os.OpenFile("/dev/full", os.O_WRONLY, 0)
		if err != nil {
			t.Skip("/dev/full is unavailable on this platform")
		}
		defer full.Close()
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		cmd := exec.CommandContext(ctx, binary, "routes", "-c", path)
		var stderr bytes.Buffer
		cmd.Stdout, cmd.Stderr = full, &stderr
		err = cmd.Run()
		assertRoutesConfidential(t, stderr.String())
		var exit *exec.ExitError
		if !errors.As(err, &exit) || exit.ExitCode() != 1 || !strings.Contains(stderr.String(), "write route topology") {
			t.Fatal("real stdout failure must propagate to exit status and safe stderr")
		}
	})
	for _, args := range [][]string{nil, {"--config"}, {"--config", ""}, {"-c", path, "extra"}, {"-c", filepath.Join(t.TempDir(), "missing.hcl")}} {
		stdout, stderr, err := run(t, args...)
		var exit *exec.ExitError
		if !errors.As(err, &exit) || exit.ExitCode() != 1 || stdout != "" || stderr == "" {
			t.Fatal("argument/read failure must exit 1 with empty stdout and diagnostics")
		}
	}
	if connections.Load() != 0 {
		t.Fatal("routes opened backend connections")
	}
}

func assertRoutesConfidential(t *testing.T, output string) {
	t.Helper()
	for _, fragment := range []string{"PRIVATE", "quoted", "雪", "example.test", "%GG", "config loaded"} {
		if strings.Contains(output, fragment) {
			t.Fatal("routes exposed private data or runtime logs")
		}
	}
}

func assertTopologyJSON(t *testing.T, output, expected string) {
	t.Helper()
	var got, want any
	if json.Unmarshal([]byte(output), &got) != nil || json.Unmarshal([]byte(expected), &want) != nil {
		t.Fatal("invalid topology JSON")
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatal("topology differs from the exact allowlisted schema/order/defaults")
	}
	if !strings.HasSuffix(output, "\n") {
		t.Fatal("JSON report must end with a newline")
	}
}

func writeRoutesConfig(t *testing.T, source string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "routes.hcl")
	if err := os.WriteFile(path, []byte(source), 0o600); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestRoutesCommandEmptyTopology(t *testing.T) {
	source := strings.Split(validConfig, `route "all"`)[0]
	source = strings.ReplaceAll(source, "${ADDRESS}", "127.0.0.1:0")
	var stdout, stderr bytes.Buffer
	cmd := newRoutesCommand()
	cmd.SetArgs([]string{"-c", writeRoutesConfig(t, source)})
	cmd.SetOut(&stdout)
	cmd.SetErr(&stderr)
	if err := cmd.Execute(); err != nil || stderr.Len() != 0 {
		t.Fatal("empty topology should succeed without diagnostics")
	}
	assertTopologyJSON(t, stdout.String(), `{"routes":[]}`)
}

func TestRoutesCommandOutputFailure(t *testing.T) {
	writeErr := errors.New("output unavailable")
	for _, tc := range []struct {
		name   string
		writer io.Writer
		want   error
	}{
		{"failed write", routesWriter{err: writeErr}, writeErr},
		{"partial failed write", routesWriter{n: 12, err: writeErr}, writeErr},
		{"short write", routesWriter{n: 12}, io.ErrShortWrite},
	} {
		t.Run(tc.name, func(t *testing.T) {
			cmd := newRoutesCommand()
			cmd.SetArgs([]string{"-c", tempConfig(t)})
			cmd.SetOut(tc.writer)
			cmd.SetErr(&bytes.Buffer{})
			if err := cmd.Execute(); !errors.Is(err, tc.want) {
				t.Fatalf("output error = %v, want %v", err, tc.want)
			}
		})
	}
}

type routesWriter struct {
	n   int
	err error
}

func (w routesWriter) Write(p []byte) (int, error) { return w.n, w.err }

const expectedRoutesJSON = `{"routes":[
  {"ordinal":1,"label":"z_write","parser":{"label":"path","kind":"path_prefix"},"operations":["DeleteObject","PutObject"],"destinations":["replica","primary"],"dispatch":"all","on_match":"continue","read_preference":"first"},
  {"ordinal":2,"label":"a_read","parser":{"label":"exact","kind":"bucket_exact"},"operations":["HeadObject","GetObject"],"destinations":["primary","replica"],"dispatch":"first","on_match":"stop","read_preference":"hash"},
  {"ordinal":3,"label":"regex_read","parser":{"label":"regex","kind":"bucket_regex"},"operations":["GetObject"],"destinations":["replica","primary"],"dispatch":"first","on_match":"stop","read_preference":"random"},
  {"ordinal":4,"label":"host_list","parser":{"label":"host","kind":"host_suffix"},"operations":["ListObjectsV2","HeadBucket","ListBuckets"],"destinations":["primary","replica"],"dispatch":"first","on_match":"stop","read_preference":"ordered_failover"}
]}`

const routesConfig = `
listener "http" "public" {
  address = env("ROUTES_PRIVATE_ADDRESS")
  addressing {
    path_style = true
    virtual_hosted = true
    host_suffixes = [env("ROUTES_PRIVATE_HOST")]
  }
}
auth "main" {
  mode = "sigv4_static"
  client "operator" {
    access_key = env("ROUTES_PRIVATE_TEXT")
    secret_key = env("ROUTES_PRIVATE_TEXT")
    allow_routes = [env("ROUTES_PRIVATE_ROUTE_REF")]
    allow_ops = ["PutObject"]
    visible_buckets = [env("ROUTES_PRIVATE_BUCKET")]
  }
}
credential "static" "primary" {
  access_key = env("ROUTES_PRIVATE_TEXT")
  secret_key = env("ROUTES_PRIVATE_TEXT")
}
target "s3" "primary" {
  endpoint = env("ROUTES_PRIVATE_ENDPOINT")
  region = env("ROUTES_PRIVATE_REGION")
  credentials = env("ROUTES_PRIVATE_CRED_REF")
}
target "s3" "replica" {
  endpoint = env("ROUTES_PRIVATE_ENDPOINT")
  region = env("ROUTES_PRIVATE_REGION")
  credentials = "credential.static.primary"
}
parser "path_prefix" "path" {
  prefix = env("ROUTES_PRIVATE_PREFIX")
}
parser "bucket_exact" "exact" {
  bucket = env("ROUTES_PRIVATE_BUCKET")
}
parser "bucket_regex" "regex" {
  pattern = env("ROUTES_PRIVATE_REGEX")
}
parser "host_suffix" "host" {
  suffix = env("ROUTES_PRIVATE_HOST")
}
route "z_write" {
  parser = env("ROUTES_PRIVATE_PARSER_REF")
  operations = ["DeleteObject", "PutObject"]
  destinations = [env("ROUTES_PRIVATE_TARGET_REF"), "target.s3.primary"]
  dispatch = "all"
  on_match = "continue"
  rewrite {
    strip_path_prefix = env("ROUTES_PRIVATE_PREFIX")
    strip_key_prefix = env("ROUTES_PRIVATE_TEXT")
    prepend_key_prefix = env("ROUTES_PRIVATE_TEXT")
    bucket = env("ROUTES_PRIVATE_BUCKET")
    key_template = env("ROUTES_PRIVATE_TEMPLATE")
  }
}
route "a_read" {
  parser = "parser.bucket_exact.exact"
  operations = ["HeadObject", "GetObject"]
  destinations = ["primary", "replica"]
  dispatch = "first"
  on_match = "stop"
  read_preference = "hash"
}
route "regex_read" {
  parser = "regex"
  operations = ["GetObject"]
  destinations = ["replica", "primary"]
  dispatch = "first"
  on_match = "stop"
  read_preference = "random"
}
route "host_list" {
  parser = "parser.host_suffix.host"
  operations = ["ListObjectsV2", "HeadBucket", "ListBuckets"]
  destinations = ["primary", "replica"]
  dispatch = "first"
  on_match = "stop"
  read_preference = "ordered_failover"
}
bucket "visible" {
  visible_name = env("ROUTES_PRIVATE_BUCKET")
  route = env("ROUTES_PRIVATE_ROUTE_REF")
}
`
