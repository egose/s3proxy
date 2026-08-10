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
