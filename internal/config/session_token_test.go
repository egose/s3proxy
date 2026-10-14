package config

import (
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"
	"testing"
)

func TestLoad_BackendSessionTokenValues(t *testing.T) {
	const envExpr = `env("S3PROXY_CREDENTIAL_SESSION_TOKEN")`
	for _, tc := range []struct {
		name, expression, value string
		unset                   bool
	}{
		{name: "omitted"},
		{name: "empty literal", expression: `""`},
		{name: "empty environment", expression: envExpr},
		{name: "unset environment", expression: envExpr, unset: true},
		{name: "literal", expression: `"CRED_PRIVATE_token/+=value"`, value: "CRED_PRIVATE_token/+=value"},
		{name: "literal escaped", expression: `"CRED_PRIVATE_$${literal}%%{directive}\n\r\t\"quoted\"\\雪"`, value: "CRED_PRIVATE_${literal}%{directive}\n\r\t\"quoted\"\\雪"},
		{name: "environment", expression: envExpr, value: "CRED_PRIVATE_token/+=value"},
		{name: "environment escaped", expression: envExpr, value: "CRED_PRIVATE_${literal}%{directive}\n\r\t\"quoted\"\\雪"},
		{name: "environment whitespace", expression: envExpr, value: " \tCRED_PRIVATE_token\n "},
		{name: "environment expression text", expression: envExpr, value: `CRED_PRIVATE_${1 + 1}%{ if true }env("OTHER")%{ endif }`},
		{name: "environment interpolation", expression: `"${env("S3PROXY_CREDENTIAL_SESSION_TOKEN")}"`, value: `CRED_PRIVATE_${literal}%{directive}`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Setenv("S3PROXY_CREDENTIAL_SESSION_TOKEN", tc.value)
			if tc.unset {
				if err := os.Unsetenv("S3PROXY_CREDENTIAL_SESSION_TOKEN"); err != nil {
					t.Fatal("could not unset test environment variable")
				}
			}
			source := backendSessionTokenConfig(tc.expression)
			for _, fromFile := range []bool{false, true} {
				var rt *Runtime
				var err error
				if fromFile {
					rt, err = LoadFile(writeTmpConfig(t, source))
				} else {
					rt, err = Load([]byte(source), "session.hcl")
				}
				if err != nil {
					t.Fatal("valid backend session credential configuration failed to load")
				}
				cred := rt.Targets["primary"].Credentials
				if cred.SessionToken != tc.value || cred.AccessKey != "AKIAPRIMARY" || cred.SecretKey != "secretprimary" || cred.Name != "primary" {
					t.Error("backend credential values were not preserved literally")
				}
				if rt.Auth.Clients["ci"].SecretKey != "secretci" {
					t.Error("backend credential affected inbound client credentials")
				}
			}
		})
	}
}

func TestLoad_BackendSessionTokenSnapshot(t *testing.T) {
	const original = "CRED_PRIVATE_original-token"
	const replacement = "CRED_PRIVATE_replacement-token"
	t.Setenv("S3PROXY_CREDENTIAL_SESSION_TOKEN", original)
	source := backendSessionTokenConfig(`env("S3PROXY_CREDENTIAL_SESSION_TOKEN")`) + `
target "s3" "shared" {
  endpoint = "https://storage.example.test"
  region = "us-east-1"
  credentials = "credential.static.primary"
}
`
	rt, err := Load([]byte(source), "session.hcl")
	if err != nil {
		t.Fatal("session token snapshot fixture failed to load")
	}
	t.Setenv("S3PROXY_CREDENTIAL_SESSION_TOKEN", replacement)
	if err := Validate(rt); err != nil {
		t.Fatal("session token snapshot failed revalidation")
	}
	if rt.Targets["primary"].Credentials.SessionToken != original || rt.Targets["shared"].Credentials.SessionToken != original {
		t.Error("environment changes modified loaded target credentials")
	}
	target := rt.Targets["primary"]
	target.Credentials.SessionToken = replacement
	rt.Targets["primary"] = target
	if rt.Targets["shared"].Credentials.SessionToken != original {
		t.Error("targets sharing a credential reference share mutable credential state")
	}
	reloaded, err := Load([]byte(source), "session.hcl")
	if err != nil || reloaded.Targets["primary"].Credentials.SessionToken != replacement {
		t.Fatal("a fresh config load did not observe the replacement environment token")
	}
}

func TestLoad_BackendSessionTokenDiagnostics(t *testing.T) {
	const value = "CRED_PRIVATE_\"quoted\n\\雪"
	const envExpr = `env("S3PROXY_CREDENTIAL_SESSION_TOKEN")`
	t.Setenv("S3PROXY_CREDENTIAL_SESSION_TOKEN", value)
	for _, tc := range []struct{ name, expression, category string }{
		{"tuple", "[" + envExpr + "]", "Unsuitable value type"},
		{"object value", "{token = " + envExpr + "}", "Unsuitable value type"},
		{"object key", "{(" + envExpr + ") = 1}", "Unsuitable value type"},
		{"duplicate key", "{for v in [1, 2] : " + envExpr + " => v}", "Duplicate object key"},
		{"invalid index", "{public = 1}[" + envExpr + "]", "Invalid index"},
		{"env argument", "env([" + envExpr + "])", "Invalid function argument"},
	} {
		for _, literal := range []bool{false, true} {
			t.Run(tc.name+"/literal="+strconv.FormatBool(literal), func(t *testing.T) {
				expression := tc.expression
				if literal {
					expression = strings.ReplaceAll(expression, envExpr, strconv.Quote(value))
				}
				source := backendSessionTokenConfig(expression)
				_, err := Load([]byte(source), "session.hcl")
				assertSessionTokenDiagnostic(t, err, value)
				for _, want := range []string{"decode config", "session.hcl:", tc.category} {
					if !strings.Contains(err.Error(), want) {
						t.Error("session token diagnostic lost its phase, location, or category")
					}
				}
				if !literal && !strings.Contains(err.Error(), `credential "static" "primary": session_token`) {
					t.Error("environment token diagnostic lost its block and field context")
				}
			})
		}
	}
}

func TestLoad_SessionTokenIsBackendOnly(t *testing.T) {
	const value = "CRED_PRIVATE_backend-token"
	t.Setenv("S3PROXY_CREDENTIAL_SESSION_TOKEN", value)
	for _, expression := range []string{strconv.Quote(value), `env("S3PROXY_CREDENTIAL_SESSION_TOKEN")`} {
		source := strings.Replace(exampleConfig, `secret_key = "secretci"`, "secret_key = \"secretci\"\n    session_token = "+expression, 1)
		_, err := Load([]byte(source), "session.hcl")
		assertSessionTokenDiagnostic(t, err, value)
		if !strings.Contains(err.Error(), "Unsupported argument") || !strings.Contains(err.Error(), "session_token") {
			t.Error("inbound client session_token was not rejected as an unsupported field")
		}
	}
}

func backendSessionTokenConfig(expression string) string {
	if expression == "" {
		return exampleConfig
	}
	return strings.Replace(exampleConfig, `secret_key = "secretprimary"`, "secret_key = \"secretprimary\"\n  session_token = "+expression, 1)
}

func assertSessionTokenDiagnostic(t *testing.T, err error, value string) {
	t.Helper()
	if err == nil {
		t.Fatal("invalid session token configuration was accepted")
	}
	for current := fmt.Errorf("caller: %w", err); current != nil; current = errors.Unwrap(current) {
		for _, format := range []string{"%s", "%v", "%+v", "%#v", "%q"} {
			output := current.Error() + fmt.Sprintf(format, current)
			for _, fragment := range []string{value, "CRED_PRIVATE", "quoted", "雪"} {
				if strings.Contains(output, fragment) {
					t.Fatal("session token diagnostic exposed a credential value")
				}
			}
		}
	}
}
