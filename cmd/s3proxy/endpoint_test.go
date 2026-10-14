package main

import (
	"bytes"
	"context"
	"errors"
	"net"
	"net/http"
	"net/http/httptest"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

func TestTargetEndpointCLI(t *testing.T) {
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
	t.Cleanup(func() {
		if connections.Load() != 0 {
			t.Error("endpoint validation opened backend connections")
		}
	})
	t.Setenv("S3PROXY_ENDPOINT_TEST_CREDENTIAL", "private\"credential\n\\雪")
	base := strings.ReplaceAll(validConfig, "${ADDRESS}", backend.Listener.Addr().String())
	base = strings.Replace(base, `secret_key = "secret"`, `secret_key = env("S3PROXY_ENDPOINT_TEST_CREDENTIAL")`, 1)
	run := func(t *testing.T, command, endpoint string, literal, pathStyle bool) (string, string, error) {
		t.Helper()
		t.Setenv("S3PROXY_ENDPOINT_TEST_VALUE", endpoint)
		expr := `env("S3PROXY_ENDPOINT_TEST_VALUE")`
		if literal {
			expr = strconv.Quote(endpoint)
		}
		source := strings.Replace(base, `"http://127.0.0.1:9000"`, expr, 1)
		source = strings.Replace(source, "force_path_style = true", "force_path_style = "+strconv.FormatBool(pathStyle), 1)
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		cmd := exec.CommandContext(ctx, binary, command, "-c", writeRoutesConfig(t, source))
		var stdout, stderr bytes.Buffer
		cmd.Stdout, cmd.Stderr = &stdout, &stderr
		err := cmd.Run()
		if ctx.Err() != nil {
			t.Fatal("offline endpoint command did not terminate")
		}
		for _, fragment := range []string{endpoint, "private", "quoted", "雪", "S3PROXY_ENDPOINT_TEST", "config loaded", "starting server"} {
			if strings.Contains(stdout.String()+stderr.String(), fragment) {
				t.Error("endpoint command exposed a private value or runtime logging")
			}
		}
		return stdout.String(), stderr.String(), err
	}
	privateBase := backend.URL + "/private%2Fbase/%3Fquery%23fragment"
	for _, tc := range []struct {
		name, endpoint, cause string
		pathStyle             bool
	}{
		{"userinfo", strings.Replace(privateBase, "http://", "http://private-user:private%22quoted%E9%9B%AA@", 1), "endpoint userinfo is unsupported; use a credential.static block", true},
		{"empty userinfo", strings.Replace(privateBase, "http://", "http://@", 1), "endpoint userinfo is unsupported; use a credential.static block", true},
		{"query", privateBase + "?private-query=private%22quoted%E9%9B%AA", "endpoint query is unsupported", true},
		{"bare query", privateBase + "?", "endpoint query is unsupported", true},
		{"fragment", privateBase + "#private%22quoted%E9%9B%AA", "endpoint fragment is unsupported", true},
		{"empty fragment", privateBase + "#", "endpoint fragment is unsupported", true},
		{"opaque", "https:private-opaque", "endpoint opaque URLs are unsupported", true},
		{"missing hostname", "https://:9000/private-base", "endpoint hostname is required", true},
		{"empty bracketed host", "https://[]:9000/private-base", "invalid endpoint: invalid URL syntax", true},
		{"missing authority", "https:///private-base", "endpoint hostname is required", true},
		{"high port", "https://private-host.invalid:65536/private-base", "endpoint port must be a decimal number between 0 and 65535", true},
		{"overflow", "https://private-host.invalid:999999999999999999999999999999/private-base", "endpoint port must be a decimal number between 0 and 65535", true},
		{"nonnumeric port", "https://private-host.invalid:private-port/private-base", "invalid endpoint: invalid URL port", true},
		{"negative port", "https://private-host.invalid:-1/private-base", "invalid endpoint: invalid URL port", true},
		{"missing bracket", "https://[fe80::1%25private-zone/private-base", "invalid endpoint: invalid URL syntax", true},
		{"unbracketed IPv6", "https://2001:db8::1/private-base", "invalid endpoint: invalid URL port", true},
		{"virtual IPv6", "https://[2001:db8::1]/private-base", "endpoint bracketed hosts require force_path_style = true", false},
		{"virtual IPv6 port", "http://[::1]:9000/private-base", "endpoint bracketed hosts require force_path_style = true", false},
		{"virtual IPv6 zone", "http://[fe80::1%25private-zone]:9000/private-base", "endpoint bracketed hosts require force_path_style = true", false},
		{"virtual mapped IPv6", "http://[::ffff:192.0.2.1]:9000/private-base", "endpoint bracketed hosts require force_path_style = true", false},
	} {
		for _, literal := range []bool{false, true} {
			for _, command := range []string{"validate", "routes", "serve"} {
				t.Run(tc.name+"/literal="+strconv.FormatBool(literal)+"/"+command, func(t *testing.T) {
					stdout, stderr, err := run(t, command, tc.endpoint, literal, tc.pathStyle)
					var exit *exec.ExitError
					if !errors.As(err, &exit) || exit.ExitCode() != 1 || stdout != "" ||
						!strings.Contains(stderr, `target.s3 "primary": `+tc.cause) {
						t.Fatal("invalid endpoint must exit 1 before success/topology/runtime output, with a safe target/endpoint cause")
					}
				})
			}
		}
	}
	t.Run("offline valid", func(t *testing.T) {
		for _, tc := range []struct {
			name, endpoint string
			pathStyle      bool
		}{
			{"occupied and unhealthy", privateBase, true},
			{"unavailable DNS", "https://private-host.invalid:65535/private%2Fbase", false},
			{"single label", "http://private-storage/private-base", false},
			{"underscore and trailing dot", "https://private_storage.invalid./private-base", true},
			{"IPv4", "http://192.0.2.1:9000/private-base", true},
			{"empty port", "https://private-host.invalid:/private-base", false},
			{"zero port", "http://private-host.invalid:0/private-base", true},
			{"IPv6", "https://[2001:db8::1]/private%2Fbase", true},
			{"IPv6 port", "http://[::1]:9000/private%20base", true},
			{"IPv6 zone", "http://[fe80::1%25private-zone]:9000/private-base", true},
			{"mapped IPv6", "http://[::ffff:192.0.2.1]:9000/private-base", true},
		} {
			for _, literal := range []bool{false, true} {
				for _, command := range []string{"validate", "routes"} {
					t.Run(tc.name+"/literal="+strconv.FormatBool(literal)+"/"+command, func(t *testing.T) {
						stdout, stderr, err := run(t, command, tc.endpoint, literal, tc.pathStyle)
						if err != nil || stderr != "" {
							t.Fatal("supported endpoint must pass offline commands with an occupied listener")
						}
						if command == "validate" {
							if stdout != "config is valid\n" {
								t.Fatal("unexpected endpoint validation success output")
							}
						} else {
							assertTopologyJSON(t, stdout, `{"routes":[{"ordinal":1,"label":"all","parser":{"label":"all","kind":"path_prefix"},"operations":["GetObject"],"destinations":["primary"],"dispatch":"first","on_match":"stop","read_preference":"first"}]}`)
						}
					})
				}
			}
		}
	})
}
