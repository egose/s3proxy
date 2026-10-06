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

func TestListenerAddressCLI(t *testing.T) {
	fileT := t
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
	source := strings.Replace(validConfig, `"http://127.0.0.1:9000"`, strconv.Quote(backend.URL), 1)
	run := func(t *testing.T, command, expr string) (string, string, error) {
		t.Helper()
		cfg := strings.Replace(source, `"${ADDRESS}"`, expr, 1)
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		cmd := exec.CommandContext(ctx, binary, command, "-c", writeRoutesConfig(fileT, cfg))
		var stdout, stderr bytes.Buffer
		cmd.Stdout, cmd.Stderr = &stdout, &stderr
		err := cmd.Run()
		if ctx.Err() != nil {
			t.Fatal("listener command did not terminate")
		}
		return stdout.String(), stderr.String(), err
	}
	t.Run("invalid", func(t *testing.T) {
		for _, tc := range []struct{ name, address, cause string }{
			{"missing port", "127.0.0.1", "must use TCP host:port syntax"},
			{"URL", "http://127.0.0.1:8080", "must use TCP host:port syntax"},
			{"high port", ":65536", "numeric port must be between 0 and 65535"},
			{"private missing port", "private\"quoted\n\\雪", "must use TCP host:port syntax"},
			{"private URL", "http://private-listener.invalid:8080", "must use TCP host:port syntax"},
			{"private high port", "private-listener.invalid:65536", "numeric port must be between 0 and 65535"},
			{"private negative port", "private-listener.invalid:-1", "numeric port must be between 0 and 65535"},
			{"private overflow before service suffix", "private-listener.invalid:10737418240service", "numeric port must be between 0 and 65535"},
			{"private uint32 overflow before service suffix", "private-listener.invalid:4294967296service", "numeric port must be between 0 and 65535"},
			{"private IPv6 bracket", "[fe80::1%private-zone:8080", "must use TCP host:port syntax"},
		} {
			for _, literal := range []bool{false, true} {
				for _, command := range []string{"validate", "routes", "serve"} {
					t.Run(tc.name+"/literal="+strconv.FormatBool(literal)+"/"+command, func(t *testing.T) {
						t.Setenv("S3PROXY_LISTENER_ADDRESS", tc.address)
						expr := `env("S3PROXY_LISTENER_ADDRESS")`
						if literal {
							expr = strconv.Quote(tc.address)
						}
						stdout, stderr, err := run(t, command, expr)
						for _, fragment := range []string{tc.address, "private", "quoted", "雪", "config loaded", "starting server"} {
							if strings.Contains(stdout+stderr, fragment) {
								t.Error("invalid listener exposed an address value or runtime logging")
							}
						}
						var exit *exec.ExitError
						if !errors.As(err, &exit) || exit.ExitCode() != 1 || stdout != "" ||
							!strings.Contains(stderr, `listener.http "public": address `+tc.cause) {
							t.Fatal("invalid listener must exit 1 with empty stdout and safe block/field/cause diagnostics")
						}
					})
				}
			}
		}
	})
	t.Run("offline valid", func(t *testing.T) {
		occupied, err := net.Listen("tcp", "127.0.0.1:0")
		if err != nil {
			t.Fatal(err)
		}
		defer occupied.Close()
		for _, address := range []string{
			occupied.Addr().String(), "192.0.2.1:8080", ":0", ":1", ":65535", "0.0.0.0:8080",
			"[::1]:8080", "[fe80::1%offline-zone]:8080", "listener.invalid:8080",
			":http", "listener.invalid:offline-service-does-not-exist", ":", "127.0.0.1:", "[::1]:",
			":+8080", ":-0", ":+", ":-", ":00065535",
			"listener.invalid:1073741824service", "listener.invalid:4294967295service",
			"listener.invalid:4294967300service",
		} {
			t.Run(address, func(t *testing.T) {
				t.Setenv("S3PROXY_LISTENER_ADDRESS", address)
				for _, command := range []string{"validate", "routes"} {
					stdout, stderr, err := run(t, command, `env("S3PROXY_LISTENER_ADDRESS")`)
					if err != nil || stderr != "" {
						t.Fatal("syntactically valid listener must pass offline commands")
					}
					if command == "validate" {
						if stdout != "config is valid\n" {
							t.Fatal("unexpected validation output")
						}
					} else {
						assertTopologyJSON(t, stdout, `{"routes":[{"ordinal":1,"label":"all","parser":{"label":"all","kind":"path_prefix"},"operations":["GetObject"],"destinations":["primary"],"dispatch":"first","on_match":"stop","read_preference":"first"}]}`)
					}
				}
			})
		}
	})
	if connections.Load() != 0 {
		t.Fatal("listener validation or offline commands opened backend connections")
	}
}
