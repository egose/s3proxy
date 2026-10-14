package main

import (
	"bytes"
	"context"
	"errors"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

func TestBackendSessionTokenCLI(t *testing.T) {
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
	const token = "CRED_PRIVATE_token/+=\"quoted\n\\雪"
	const envExpr = `env("S3PROXY_SESSION_CLI_TOKEN")`
	t.Setenv("S3PROXY_SESSION_CLI_KEY", "CRED_PRIVATE_key")
	base := strings.NewReplacer(
		"${ADDRESS}", backend.Listener.Addr().String(),
		"http://127.0.0.1:9000", backend.URL+"/private%2Fbase",
		`access_key = "access"`, `access_key = env("S3PROXY_SESSION_CLI_KEY")`,
		`secret_key = "secret"`, `secret_key = env("S3PROXY_SESSION_CLI_KEY")`,
	).Replace(validConfig)
	for _, tc := range []struct {
		name, expression, category string
		empty, unset               bool
	}{
		{name: "omitted"},
		{name: "literal", expression: strconv.Quote(token)},
		{name: "environment", expression: envExpr},
		{name: "empty literal", expression: `""`},
		{name: "empty environment", expression: envExpr, empty: true},
		{name: "unset environment", expression: envExpr, unset: true},
		{name: "tuple", expression: "[" + envExpr + "]", category: "Unsuitable value type"},
		{name: "object key", expression: "{(" + envExpr + ") = 1}", category: "Unsuitable value type"},
		{name: "duplicate key", expression: "{for v in [1, 2] : " + envExpr + " => v}", category: "Duplicate object key"},
		{name: "invalid index", expression: "{public = 1}[" + envExpr + "]", category: "Invalid index"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Setenv("S3PROXY_SESSION_CLI_TOKEN", token)
			if tc.empty || tc.unset {
				t.Setenv("S3PROXY_SESSION_CLI_TOKEN", "")
			}
			if tc.unset {
				if err := os.Unsetenv("S3PROXY_SESSION_CLI_TOKEN"); err != nil {
					t.Fatal("could not unset test token variable")
				}
			}
			source := base
			if tc.expression != "" {
				source = strings.Replace(source, `credential "static" "primary" {`, "credential \"static\" \"primary\" {\n  session_token = "+tc.expression, 1)
			}
			commands := []string{"validate", "routes"}
			if tc.category != "" {
				commands = append(commands, "serve")
			}
			path := writeRoutesConfig(t, source)
			for _, command := range commands {
				ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
				cmd := exec.CommandContext(ctx, binary, command, "-c", path)
				var stdout, stderr bytes.Buffer
				cmd.Stdout, cmd.Stderr = &stdout, &stderr
				err := cmd.Run()
				timedOut := ctx.Err() != nil
				cancel()
				if timedOut {
					t.Fatal("offline session credential command did not terminate")
				}
				assertRoutesConfidential(t, stdout.String()+stderr.String())
				if tc.category != "" {
					var exit *exec.ExitError
					if !errors.As(err, &exit) || exit.ExitCode() != 1 || stdout.Len() != 0 ||
						!strings.Contains(stderr.String(), tc.category) || !strings.Contains(stderr.String(), `credential "static" "primary": session_token`) {
						t.Error("invalid token must fail before topology/runtime output with a safe field/category diagnostic")
					}
					continue
				}
				if err != nil || stderr.Len() != 0 {
					t.Fatal("valid session credentials failed offline validation")
				}
				if command == "validate" {
					if stdout.String() != "config is valid\n" {
						t.Error("session credential validation emitted unexpected output")
					}
				} else {
					assertTopologyJSON(t, stdout.String(), `{"routes":[{"ordinal":1,"label":"all","parser":{"label":"all","kind":"path_prefix"},"operations":["GetObject"],"destinations":["primary"],"dispatch":"first","on_match":"stop","read_preference":"first"}]}`)
				}
			}
		})
	}
	if connections.Load() != 0 {
		t.Error("offline session credential commands contacted a backend")
	}
}
