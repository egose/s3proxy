package main

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestValidateCLIEnvDiagnosticConfidentiality(t *testing.T) {
	binary := filepath.Join(t.TempDir(), "s3proxy")
	build := exec.Command("go", "build", "-o", binary, ".")
	if output, err := build.CombinedOutput(); err != nil {
		t.Fatalf("build CLI: %v\n%s", err, output)
	}
	const expr = `env("S3PROXY_CLI_DIAGNOSTIC")`
	for _, tc := range []struct {
		name, old, replacement, value, want string
	}{
		{"endpoint", `"http://127.0.0.1:9000"`, expr, "https://user:private-password@example.test/%GG?private-query", "invalid endpoint"}, // pragma: allowlist secret
		{"duration", `region           = "us-east-1"`, "region = \"us-east-1\"\n timeout = " + expr, "private\"quoted\n\\雪", "invalid timeout"},
		{"template", `on_match     = "stop"`, "on_match = \"stop\"\n rewrite { key_template = " + expr + " }", "{{ privatefunction }}", "invalid key_template"},
		{"reference", `parser       = "all"`, "parser = " + expr, "parser.private\"quoted\n\\雪", "unknown parser"},
		{"enum", `dispatch     = "first"`, "dispatch = " + expr, "private\"quoted\n\\雪", "invalid dispatch"},
		{"HCL", `secret_key = "secret"`, "secret_key = {for v in [1, 2] : " + expr + " => v}", "private\"quoted\n\\雪", "Duplicate object key"}, // pragma: allowlist secret
		{"regex", "parser \"path_prefix\" \"all\" {\n  prefix = \"/\"", "parser \"bucket_regex\" \"all\" {\n pattern = " + expr, "(?P<private\"name>x)", "invalid pattern"},
		{"listener duration", `address = "127.0.0.1:0"`, "address = \"127.0.0.1:0\"\n timeouts { read = " + expr + " }", "1privateunit", "invalid read timeout"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Setenv("S3PROXY_CLI_DIAGNOSTIC", tc.value)
			source := strings.ReplaceAll(validConfig, "${ADDRESS}", "127.0.0.1:0")
			source = strings.Replace(source, tc.old, tc.replacement, 1)
			path := filepath.Join(t.TempDir(), "config.hcl")
			if err := os.WriteFile(path, []byte(source), 0o600); err != nil {
				t.Fatal(err)
			}
			var stdout, stderr bytes.Buffer
			cmd := exec.Command(binary, "validate", "--config", path)
			cmd.Stdout, cmd.Stderr = &stdout, &stderr
			err := cmd.Run()
			if exit, ok := err.(*exec.ExitError); !ok || exit.ExitCode() != 1 {
				t.Fatal("invalid config must exit with status 1")
			}
			output := stdout.String() + stderr.String()
			for _, fragment := range []string{"private", "quoted", "雪", "example.test", "%GG"} {
				if strings.Contains(output, fragment) {
					t.Fatal("CLI exposed environment-derived data")
				}
			}
			if stdout.Len() != 0 || !strings.Contains(stderr.String(), tc.want) {
				t.Fatal("CLI lost the useful error or emitted success output")
			}
		})
	}
	t.Run("literal success", func(t *testing.T) {
		t.Setenv("S3PROXY_CLI_DIAGNOSTIC", "${literal}%{ if true }\"quoted\"\n\\雪%{ endif }")
		source := strings.ReplaceAll(validConfig, "${ADDRESS}", "127.0.0.1:0")
		source = strings.Replace(source, `secret_key = "secret"`, "secret_key = "+expr, 1)
		path := filepath.Join(t.TempDir(), "config.hcl")
		if err := os.WriteFile(path, []byte(source), 0o600); err != nil {
			t.Fatal(err)
		}
		var stdout, stderr bytes.Buffer
		cmd := exec.Command(binary, "validate", "--config", path)
		cmd.Stdout, cmd.Stderr = &stdout, &stderr
		if err := cmd.Run(); err != nil || stdout.String() != "config is valid\n" || stderr.Len() != 0 {
			t.Fatal("real validate CLI failed literal-value success contract")
		}
	})
}
