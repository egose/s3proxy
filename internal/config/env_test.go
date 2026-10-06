package config

import (
	"fmt"
	"os"
	"strings"
	"testing"
)

func TestLoadFile_EnvLiteralValues(t *testing.T) {
	for i, value := range []string{
		`${S3PROXY_SENTINEL_SECRET}`,
		`${1 + 1}`,
		`%{ if true }S3PROXY_SENTINEL_SECRET%{ endif }`,
		"S3PROXY_SENTINEL_SECRET\n\r\t\"quoted\"\\path/雪/é",
		`$${escaped} %%{escaped} env("OTHER_VARIABLE")`,
	} {
		t.Run(fmt.Sprintf("literal-%d", i), func(t *testing.T) {
			t.Setenv("S3PROXY_TEST_LITERAL", value)
			for _, expr := range []string{
				`env("S3PROXY_TEST_LITERAL")`,
				`env ( "S3PROXY_TEST_LITERAL" )`,
				"env(\n  \"S3PROXY_TEST_LITERAL\"\n)",
				`env(/* variable */ "S3PROXY_TEST_LITERAL")`,
				`"${env("S3PROXY_TEST_LITERAL")}"`,
			} {
				cfg := strings.ReplaceAll(exampleConfig, `"secretci"`, expr)
				cfg = strings.ReplaceAll(cfg, `"secretprimary"`, expr)
				rt, err := LoadFile(writeTmpConfig(t, cfg))
				if err != nil {
					t.Fatalf("LoadFile with %s: %v", expr, err)
				}
				if rt.Auth.Clients["ci"].SecretKey != value || rt.Targets["primary"].Credentials.SecretKey != value { // pragma: allowlist secret
					t.Fatal("environment value was not preserved literally in both credentials")
				}
			}
		})
	}
}

func TestLoadFile_EnvTextIsNotEvaluated(t *testing.T) {
	t.Setenv("S3PROXY_TEST_LITERAL", "*/\nthis is not HCL\n/* ${S3PROXY_SENTINEL_SECRET}")
	for _, expr := range []string{
		`"env(\"S3PROXY_TEST_LITERAL\")"`,
		"<<EOT\nenv(\"S3PROXY_TEST_LITERAL\")\nEOT",
	} {
		cfg := "# env(\"S3PROXY_TEST_LITERAL\")\n// env(\"S3PROXY_TEST_LITERAL\")\n/* env(\"S3PROXY_TEST_LITERAL\") */\n" + exampleConfig
		cfg = strings.Replace(cfg, `"secretprimary"`, expr, 1)
		rt, err := LoadFile(writeTmpConfig(t, cfg))
		if err != nil {
			t.Fatalf("LoadFile: %v", err)
		}
		if got := rt.Targets["primary"].Credentials.SecretKey; strings.TrimSuffix(got, "\n") != `env("S3PROXY_TEST_LITERAL")` {
			t.Fatalf("literal env text changed: %q", got)
		}
	}
}

func TestLoadFile_EnvMissingAndEmpty(t *testing.T) {
	const name = "S3PROXY_TEST_MISSING"
	for _, unset := range []bool{true, false} {
		t.Setenv(name, "")
		if unset {
			if err := os.Unsetenv(name); err != nil {
				t.Fatal(err)
			}
		}
		for _, tc := range []struct {
			old, replacement, wantErr string
		}{
			{`"secretci"`, `env("S3PROXY_TEST_MISSING")`, `client "ci" has empty secret_key`},
			{`"AKIACI123"`, `env("S3PROXY_TEST_MISSING")`, `client "ci" has empty access_key`},
			{`"secretprimary"`, `env("S3PROXY_TEST_MISSING")`, `credentials secret_key is empty`},
			{`"AKIAPRIMARY"`, `env("S3PROXY_TEST_MISSING")`, `credentials access_key is empty`},
			{`"https://minio-a.internal"`, `env("S3PROXY_TEST_MISSING")`, `endpoint is required`},
			{`prepend_key_prefix = "assets/"`, `prepend_key_prefix = env("S3PROXY_TEST_MISSING")`, ""},
		} {
			rt, err := LoadFile(writeTmpConfig(t, strings.Replace(exampleConfig, tc.old, tc.replacement, 1)))
			if tc.wantErr == "" {
				if err != nil || rt.Routes[0].Rewrite.PrependKeyPrefix != "" {
					t.Fatalf("optional missing value: runtime=%v, err=%v", rt != nil, err)
				}
			} else if err == nil || !strings.Contains(err.Error(), tc.wantErr) {
				t.Fatalf("error = %v, want %q", err, tc.wantErr)
			}
		}
	}
}

func TestLoadFile_EnvDiagnosticsKeepSourceLocation(t *testing.T) {
	t.Setenv("S3PROXY_TEST_ADDRESS", ":8080")
	for _, tc := range []struct {
		name, value, phase, location string
	}{
		{"parse", "@", "parse config", ":5,24-"},
		{"decode", `"not-a-number"`, "decode config", ":5,25-"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			cfg := "\n\r\n  listener \"http\" \"public\" {\n    address = env(\"S3PROXY_TEST_ADDRESS\")\n    max_header_bytes = " + tc.value + "\n    addressing { path_style = true }\n}\n"
			path := writeTmpConfig(t, cfg)
			_, err := LoadFile(path)
			if err == nil || !strings.Contains(err.Error(), tc.phase) || !strings.Contains(err.Error(), path+tc.location) {
				t.Fatalf("error = %v, want %s at %s%s", err, tc.phase, path, tc.location)
			}
		})
	}
}

func TestLoadFile_EnvErrorsDoNotExposeSecrets(t *testing.T) {
	const secret = "S3PROXY_SENTINEL_SECRET" // pragma: allowlist secret
	t.Setenv("S3PROXY_TEST_SECRET", "${"+secret+"}")
	base := strings.ReplaceAll(exampleConfig, `"secretprimary"`, `env("S3PROXY_TEST_SECRET")`)
	for _, tc := range []struct {
		name, config, wantErr string
	}{
		{"parse", base + "\n@", "parse config"},
		{"decode", strings.Replace(base, "65536", `env("S3PROXY_TEST_SECRET")`, 1), "decode config"},
		{"validation", strings.Replace(base, `"secretci"`, `""`, 1), "empty secret_key"},
		{"null argument", strings.Replace(base, `env("S3PROXY_TEST_SECRET")`, `env(null)`, 1), "Invalid function argument"},
		{"collection argument", strings.Replace(base, `env("S3PROXY_TEST_SECRET")`, `env([env("S3PROXY_TEST_SECRET")])`, 1), "Invalid function argument"},
		{"missing argument", strings.Replace(base, `env("S3PROXY_TEST_SECRET")`, `env()`, 1), "Not enough function arguments"},
		{"extra argument", strings.Replace(base, `env("S3PROXY_TEST_SECRET")`, `env("S3PROXY_TEST_SECRET", "extra")`, 1), "Too many function arguments"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			_, err := LoadFile(writeTmpConfig(t, tc.config))
			if err == nil || !strings.Contains(err.Error(), tc.wantErr) {
				t.Fatalf("error = %v, want %q", err, tc.wantErr)
			}
			if strings.Contains(err.Error(), secret) {
				t.Fatal("diagnostic exposed the environment secret")
			}
		})
	}
}
