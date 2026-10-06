package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestValidateCommandRejectsPositionalArgs(t *testing.T) {
	cmd := newValidateCommand()
	cmd.SetArgs([]string{"--config", tempConfig(t), "extra"})
	cmd.SetOut(&bytes.Buffer{})
	cmd.SetErr(&bytes.Buffer{})

	if err := cmd.Execute(); err == nil {
		t.Fatalf("Execute() error = nil, want positional arg error")
	}
}

func TestServeCommandRejectsPositionalArgs(t *testing.T) {
	cmd := newServeCommand()
	cmd.SetArgs([]string{"--config", tempConfig(t), "extra"})
	cmd.SetOut(&bytes.Buffer{})
	cmd.SetErr(&bytes.Buffer{})

	if err := cmd.Execute(); err == nil {
		t.Fatalf("Execute() error = nil, want positional arg error")
	}
}

func TestVersionCommandRejectsPositionalArgs(t *testing.T) {
	cmd := newVersionCommand()
	cmd.SetArgs([]string{"extra"})
	cmd.SetOut(&bytes.Buffer{})
	cmd.SetErr(&bytes.Buffer{})

	if err := cmd.Execute(); err == nil {
		t.Fatalf("Execute() error = nil, want positional arg error")
	}
}

func TestValidateCommandWritesToInjectedOutput(t *testing.T) {
	var out bytes.Buffer
	cmd := newValidateCommand()
	cmd.SetArgs([]string{"--config", tempConfig(t)})
	cmd.SetOut(&out)
	cmd.SetErr(&bytes.Buffer{})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if got := out.String(); got != "config is valid\n" {
		t.Fatalf("output = %q", got)
	}
}

func TestVersionCommandWritesToInjectedOutput(t *testing.T) {
	var out bytes.Buffer
	cmd := newVersionCommand()
	cmd.SetArgs(nil)
	cmd.SetOut(&out)
	cmd.SetErr(&bytes.Buffer{})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if got := out.String(); got != version+"\n" {
		t.Fatalf("output = %q", got)
	}
}

func TestValidateCommandEnvironmentValues(t *testing.T) {
	const sentinel = "S3PROXY_CLI_SENTINEL_SECRET"
	t.Setenv("S3PROXY_CLI_TEST_SECRET", "${"+sentinel+"}%{ if true }\n\"quoted\"\\雪%{ endif }")
	t.Setenv("S3PROXY_CLI_TEST_MISSING", "")
	if err := os.Unsetenv("S3PROXY_CLI_TEST_MISSING"); err != nil {
		t.Fatal(err)
	}
	base := strings.ReplaceAll(validConfig, "${ADDRESS}", "127.0.0.1:0")
	base = strings.Replace(base, `secret_key = "secret"`, `secret_key = env ( "S3PROXY_CLI_TEST_SECRET" )`, 1)
	for _, tc := range []struct {
		name, source, wantErr string
	}{
		{"valid", base, ""},
		{"parse location", "\n\n\n@\n" + base, ":4,1-"},
		{"decode", strings.Replace(base, `path_style = true`, `path_style = env("S3PROXY_CLI_TEST_SECRET")`, 1), "decode config"},
		{"missing credential", strings.Replace(base, `access_key = "access"`, `access_key = env("S3PROXY_CLI_TEST_MISSING")`, 1), "credentials access_key is empty"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "config.hcl")
			if err := os.WriteFile(path, []byte(tc.source), 0o600); err != nil {
				t.Fatal(err)
			}
			var out, stderr bytes.Buffer
			cmd := newValidateCommand()
			cmd.SetArgs([]string{"--config", path})
			cmd.SetOut(&out)
			cmd.SetErr(&stderr)
			err := cmd.Execute()
			if tc.wantErr == "" {
				if err != nil || out.String() != "config is valid\n" || stderr.Len() != 0 {
					t.Fatalf("validation failed: error=%v stdout=%q stderr=%q", err, out.String(), stderr.String())
				}
			} else if err == nil || !strings.Contains(err.Error(), tc.wantErr) || strings.Contains(out.String(), "config is valid") {
				t.Fatalf("error=%v stdout=%q, want %q", err, out.String(), tc.wantErr)
			}
			if strings.Contains(out.String()+stderr.String(), sentinel) || (err != nil && strings.Contains(err.Error(), sentinel)) {
				t.Fatal("CLI exposed the environment secret")
			}
		})
	}
}

func tempConfig(t *testing.T) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "config.hcl")
	src := strings.ReplaceAll(validConfig, "${ADDRESS}", "127.0.0.1:0")
	if err := os.WriteFile(path, []byte(src), 0o600); err != nil {
		t.Fatal(err)
	}
	return path
}

var validConfig = `
listener "http" "public" {
  address = "${ADDRESS}"

  addressing {
    path_style = true
  }
}

auth "main" {
  mode = "none"
}

credential "static" "primary" {
  access_key = "access"
  secret_key = "secret" // pragma: allowlist secret
}

target "s3" "primary" {
  endpoint         = "http://127.0.0.1:9000"
  region           = "us-east-1"
  force_path_style = true
  credentials      = "primary"
}

parser "path_prefix" "all" {
  prefix = "/"
}

route "all" {
  parser       = "all"
  operations   = ["GetObject"]
  destinations = ["primary"]
  dispatch     = "first"
  on_match     = "stop"
}
`
