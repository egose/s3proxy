package main

import (
	"bytes"
	"context"
	"errors"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/egose/s3proxy/internal/auth"
	"github.com/egose/s3proxy/internal/config"
	"github.com/egose/s3proxy/internal/listbuckets"
	"github.com/egose/s3proxy/internal/requestctx"
	"github.com/egose/s3proxy/internal/rewrite"
	"github.com/egose/s3proxy/internal/router"
	"github.com/egose/s3proxy/internal/s3ops"
)

func TestPrintExampleConfigCLI(t *testing.T) {
	binary := filepath.Join(t.TempDir(), "s3proxy")
	if output, err := exec.Command("go", "build", "-o", binary, ".").CombinedOutput(); err != nil {
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
	values := map[string]string{
		"S3PROXY_CLIENT_ACCESS_KEY":         "synthetic-client-access",
		"S3PROXY_CLIENT_SECRET_KEY":         "synthetic-client-secret",
		"S3PROXY_TARGET_PRIMARY_ACCESS_KEY": "synthetic-backend-access",
		"S3PROXY_TARGET_PRIMARY_SECRET_KEY": "synthetic-backend-secret",
		"S3PROXY_TARGET_PRIMARY_ENDPOINT":   backend.URL,
	}
	baseEnv := []string{}
	for _, entry := range os.Environ() {
		name, _, _ := strings.Cut(entry, "=")
		if _, required := values[name]; !required {
			baseEnv = append(baseEnv, entry)
		}
	}
	environment := func(vars map[string]string) []string {
		env := append([]string(nil), baseEnv...)
		for name, value := range vars {
			env = append(env, name+"="+value)
		}
		return env
	}
	workdir := t.TempDir()
	run := func(env []string, output io.Writer, args ...string) (string, string, error) {
		t.Helper()
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		cmd := exec.CommandContext(ctx, binary, args...)
		cmd.Env, cmd.Dir = env, workdir
		var stdout, stderr bytes.Buffer
		cmd.Stdout, cmd.Stderr = &stdout, &stderr
		if output != nil {
			cmd.Stdout = output
		}
		err := cmd.Run()
		if ctx.Err() != nil {
			t.Fatal("offline CLI did not terminate")
		}
		return stdout.String(), stderr.String(), err
	}
	var printed string
	for _, mode := range []string{"unset", "valid", "secret malformed"} {
		t.Run(mode, func(t *testing.T) {
			vars := map[string]string{}
			for name, value := range values {
				if mode == "valid" {
					vars[name] = value
				} else if mode == "secret malformed" {
					vars[name] = "PRIVATE_\"\n\r\t\\雪${env(\"INJECTED\")}%{bad}http://%GG"
				}
			}
			stdout, stderr, err := run(environment(vars), nil, "print-example-config")
			if err != nil || stderr != "" || stdout != exampleConfig || !strings.HasSuffix(stdout, "\n") {
				t.Fatal("printing must succeed with identical static HCL and no diagnostics")
			}
			if printed != "" && stdout != printed {
				t.Fatal("environment changed output")
			}
			printed = stdout
			for name := range values {
				if !strings.Contains(stdout, `env("`+name+`")`) {
					t.Fatalf("missing literal reference to %s", name)
				}
			}
		})
	}
	entries, err := os.ReadDir(workdir)
	if err != nil || len(entries) != 0 {
		t.Fatal("printing created files in the working directory")
	}
	for _, args := range [][]string{
		{"print-example-config", "extra"},
		{"print-example-config", "--unknown"},
		{"print-example-config", "--config", "missing.hcl"},
		{"print-example-config", "--output", "example.hcl"},
	} {
		stdout, stderr, err := run(baseEnv, nil, args...)
		assertExampleCLIFailure(t, stdout, stderr, err, "")
	}
	t.Run("real output failure", func(t *testing.T) {
		full, err := os.OpenFile("/dev/full", os.O_WRONLY, 0)
		if err != nil {
			t.Skip("/dev/full unavailable")
		}
		defer full.Close()
		stdout, stderr, err := run(baseEnv, full, "print-example-config")
		assertExampleCLIFailure(t, stdout, stderr, err, "write example config")
	})
	path := writeRoutesConfig(t, printed)
	checkOffline := func(path string) {
		t.Helper()
		stdout, stderr, err := run(environment(values), nil, "validate", "--config", path)
		if err != nil || stdout != "config is valid\n" || stderr != "" {
			t.Fatalf("validate printed HCL: %v, stdout=%q stderr=%q", err, stdout, stderr)
		}
		stdout, stderr, err = run(environment(values), nil, "routes", "--config", path)
		if err != nil || stderr != "" {
			t.Fatalf("routes printed HCL: %v, stderr=%q", err, stderr)
		}
		assertTopologyJSON(t, stdout, expectedExampleTopology)
	}
	checkOffline(path)
	occupied, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer occupied.Close()
	checkOffline(writeRoutesConfig(t, strings.Replace(printed, "127.0.0.1:8080", occupied.Addr().String(), 1)))
	for name := range values {
		t.Run("missing "+name, func(t *testing.T) {
			vars := map[string]string{}
			for other, value := range values {
				if other != name {
					vars[other] = value
				}
			}
			for _, command := range []string{"validate", "routes"} {
				stdout, stderr, err := run(environment(vars), nil, command, "-c", path)
				assertExampleCLIFailure(t, stdout, stderr, err, "validate config")
				for _, value := range values {
					if strings.Contains(stderr, value) {
						t.Fatal("missing-variable diagnostic exposed a configured value")
					}
				}
			}
		})
	}
	for name, value := range values {
		t.Setenv(name, value)
	}
	rt, err := config.LoadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	assertExampleContract(t, rt)
	if connections.Load() != 0 {
		t.Fatal("offline commands contacted the backend")
	}
}

func assertExampleCLIFailure(t *testing.T, stdout, stderr string, err error, want string) {
	t.Helper()
	var exit *exec.ExitError
	if !errors.As(err, &exit) || exit.ExitCode() != 1 || stdout != "" || stderr == "" || !strings.Contains(stderr, want) || strings.Contains(stderr, "config loaded") {
		t.Fatalf("expected controlled CLI failure: err=%v stdout=%q stderr=%q", err, stdout, stderr)
	}
}

func assertExampleContract(t *testing.T, rt *config.Runtime) {
	t.Helper()
	if rt.Listener.Address != "127.0.0.1:8080" || !rt.Listener.Addressing.PathStyle || rt.Listener.Addressing.VirtualHosted || rt.Auth.Mode != config.AuthModeSigV4Static {
		t.Fatal("starter must use native loopback, path-style, and SigV4 auth")
	}
	if rt.Listener.ReplayBodyMaxBytes != 32<<20 || rt.Listener.ReplayBodyAggregateMaxBytes != 256<<20 || rt.Listener.Timeouts != (config.Timeouts{ReadHeader: 10 * time.Second, Read: 2 * time.Minute, Write: 5 * time.Minute, Idle: time.Minute}) {
		t.Fatal("documented replay bounds/timeouts changed")
	}
	if len(rt.Auth.Clients) != 1 || len(rt.Targets) != 1 || len(rt.Parsers) != 1 || len(rt.Routes) != 1 || len(rt.Buckets) != 1 {
		t.Fatal("starter must expose exactly one client, target, parser, route, and bucket")
	}
	client := rt.Auth.Clients["local"]
	target := rt.Targets["primary"]
	if client.AccessKey != "synthetic-client-access" || client.SecretKey != "synthetic-client-secret" || target.Credentials.AccessKey != "synthetic-backend-access" || target.Credentials.SecretKey != "synthetic-backend-secret" || target.Region != "us-east-1" || !target.ForcePathStyle || target.Timeout != 2*time.Minute {
		t.Fatal("separate credentials or documented target settings changed")
	}
	ops := []string{"GetObject", "HeadObject", "PutObject", "DeleteObject", "HeadBucket", "ListObjectsV2", "ListBuckets"}
	if !reflect.DeepEqual(client.AllowOps, ops) || !reflect.DeepEqual(client.AllowRoutes, []string{"images_rw"}) || !reflect.DeepEqual(client.VisibleBuckets, []string{"images"}) || rt.Buckets[0].RouteRef != "images_rw" {
		t.Fatal("client permissions must be explicit and narrowly scoped")
	}
	principal := &auth.Principal{AllowRoutes: client.AllowRoutes, AllowOps: client.AllowOps, VisibleBuckets: client.VisibleBuckets}
	authorizer := auth.NewAuthorizer()
	resolver := router.NewResolver(rt.Routes, rt.Parsers, []string{"primary"})
	for _, tc := range []struct{ method, path, op, key string }{
		{"GET", "/images/a%20b//c", "GetObject", "a%20b//c"},
		{"HEAD", "/images/hello.txt", "HeadObject", "hello.txt"},
		{"PUT", "/images/hello.txt", "PutObject", "hello.txt"},
		{"DELETE", "/images/hello.txt", "DeleteObject", "hello.txt"},
		{"HEAD", "/images", "HeadBucket", ""},
		{"GET", "/images?list-type=2&prefix=hello", "ListObjectsV2", ""},
	} {
		op := s3ops.Operation(tc.op)
		ctx, err := requestctx.FromRequest(httptest.NewRequest(tc.method, "http://localhost"+tc.path, nil), rt.Listener.Addressing)
		if err != nil {
			t.Fatal(err)
		}
		if classified, err := s3ops.Classify(ctx); err != nil || classified != op {
			t.Fatalf("%s classified as %s: %v", tc.path, classified, err)
		}
		matches, err := resolver.Resolve(ctx, op)
		if err != nil || len(matches) != 1 || matches[0].Route.Name != "images_rw" || !reflect.DeepEqual(matches[0].Destinations, []string{"primary"}) || !authorizer.AllowRoute(principal, "images_rw", op) {
			t.Fatalf("%s not routed/authorized: %v", tc.op, err)
		}
		rw, err := rewrite.New().Apply(ctx, matches[0].Route.Rewrite, matches[0].Captures)
		if err != nil || rw.Bucket != "images-store" || rw.Key != tc.key {
			t.Fatalf("%s rewrite = %+v, err=%v", tc.op, rw, err)
		}
		if tc.op == "ListObjectsV2" && (rw.Namespace == nil || rw.Namespace.Prefix != "") {
			t.Fatal("listing must expose the same unprefixed object namespace")
		}
	}
	for _, bucket := range []string{"images-store", "images-other", "other"} {
		if _, err := resolver.Resolve(&requestctx.Context{Bucket: bucket}, s3ops.OpGetObject); err == nil {
			t.Fatalf("unexpected route for %s", bucket)
		}
	}
	if authorizer.AllowRoute(principal, "other", s3ops.OpGetObject) || authorizer.AllowOperation(principal, s3ops.Operation("CreateBucket")) {
		t.Fatal("starter authorized an undeclared route/operation")
	}
	if !authorizer.AllowOperation(principal, s3ops.OpListBuckets) {
		t.Fatal("virtual ListBuckets must be permitted")
	}
	root, err := requestctx.FromRequest(httptest.NewRequest("GET", "http://localhost/", nil), rt.Listener.Addressing)
	if err != nil {
		t.Fatal(err)
	}
	if op, err := s3ops.Classify(root); err != nil || op != s3ops.OpListBuckets {
		t.Fatal("root request must classify as virtual discovery")
	}
	if _, err := resolver.Resolve(root, s3ops.OpListBuckets); err == nil {
		t.Fatal("virtual discovery must not require a backend route")
	}
	views := listbuckets.New(rt.Buckets, time.Time{}).List(principal)
	if len(views) != 1 || views[0].Name != "images" {
		t.Fatal("virtual discovery must expose only images")
	}
}

func TestPrintExampleConfigOutputFailure(t *testing.T) {
	writeErr := errors.New("output unavailable")
	for _, tc := range []struct {
		name   string
		writer io.Writer
		want   error
	}{
		{"failed", routesWriter{err: writeErr}, writeErr},
		{"partial failed", routesWriter{n: 12, err: writeErr}, writeErr},
		{"short", routesWriter{n: 12}, io.ErrShortWrite},
	} {
		t.Run(tc.name, func(t *testing.T) {
			cmd := newPrintExampleConfigCommand()
			cmd.SetArgs([]string{})
			cmd.SetOut(tc.writer)
			cmd.SetErr(io.Discard)
			if err := cmd.Execute(); !errors.Is(err, tc.want) {
				t.Fatalf("output error = %v, want %v", err, tc.want)
			}
		})
	}
}

const expectedExampleTopology = `{"routes":[{"ordinal":1,"label":"images_rw","parser":{"label":"images","kind":"bucket_exact"},"operations":["GetObject","HeadObject","PutObject","DeleteObject","HeadBucket","ListObjectsV2"],"destinations":["primary"],"dispatch":"first","on_match":"stop","read_preference":"first"}]}`
