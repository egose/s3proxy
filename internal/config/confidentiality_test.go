package config

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
	"testing"
	"time"
)

func TestEnvDiagnosticConfidentiality(t *testing.T) {
	const expr = `env("S3PROXY_DIAGNOSTIC_VALUE")`
	const secret = "private\"quoted\n\\雪"
	for _, tc := range []struct {
		name, old, replacement, value, block, field, cause string
	}{
		{"endpoint userinfo", `"https://minio-a.internal"`, expr, "https://user:" + secret + "@example.test/%GG", `target.s3 "primary"`, "endpoint", "URL"},
		{"endpoint escape", `"https://minio-a.internal"`, expr, "https://user:private-password@example.test/%GG?private-query", `target.s3 "primary"`, "endpoint", "escape"},
		{"endpoint port", `"https://minio-a.internal"`, expr, "https://example.test:private-port/", `target.s3 "primary"`, "endpoint", "port"},
		{"endpoint host", `"https://minio-a.internal"`, expr, "https://private-host%00.example.test/", `target.s3 "primary"`, "endpoint", "escape"},
		{"endpoint query", `"https://minio-a.internal"`, expr, "https://example.test/?private-query\n", `target.s3 "primary"`, "endpoint", "URL"},
		{"target duration", `"5s"`, expr, secret, `target.s3 "primary"`, "timeout", "duration"},
		{"duration unit", `"5s"`, `"1${env("S3PROXY_DIAGNOSTIC_VALUE")}"`, "privateunit", `target.s3 "primary"`, "timeout", "duration"},
		{"read duration", `"30s"`, expr, secret, `listener.http "public"`, "read", "duration"},
		{"header duration", `"10s"`, expr, secret, `listener.http "public"`, "read_header", "duration"},
		{"idle duration", `"60s"`, expr, secret, `listener.http "public"`, "idle", "duration"},
		{"write duration", `"0s"`, expr, secret, `listener.http "public"`, "write", "duration"},
		{"regex", `"^tenant-(?P<tenant>[a-z0-9-]+)-logs$"`, expr, "(?P<" + secret + ">x)", `parser.bucket_regex "tenant_logs"`, "pattern", "capture"},
		{"template function", `prepend_key_prefix = "assets/"`, "key_template = " + expr, "{{ privatefunction }}", `route "images_rw"`, "key_template", "function"},
		{"template token", `prepend_key_prefix = "assets/"`, "key_template = " + expr, "{{ " + strconv.Quote(secret) + " @ }}", `route "images_rw"`, "key_template", "template"},
		{"credential ref", `credentials      = "primary"`, "credentials = " + expr, secret, `target.s3 "primary"`, "credential", "unknown"},
		{"parser ref", `parser       = "images"`, "parser = " + expr, secret, `route "images_rw"`, "parser", "unknown"},
		{"destination ref", `destinations = ["primary"]`, "destinations = [" + expr + "]", secret, `route "images_rw"`, "destination", "unknown"},
		{"bucket ref stripped", `route        = "images_rw"`, "route = " + expr, "route." + secret, `bucket "images"`, "route", "unknown"},
		{"policy ref stripped", `"route.images_rw"`, expr, "route." + secret, `client "ci"`, "allow_routes", "unknown"},
		{"policy bucket", `visible_buckets = ["images"]`, "visible_buckets = [" + expr + "]", secret, `client "ci"`, "visible_buckets", "unknown"},
		{"duplicate visible name", "visible_name = \"images\"\n  route        = \"images_rw\"", "visible_name = " + expr + "\n route = \"images_rw\"\n}\nbucket \"other\" {\n route = \"images_rw\"\n visible_name = " + expr, secret, "bucket", "visible_name", "duplicate"},
		{"policy operation", `visible_buckets = ["images"]`, "allow_ops = [" + expr + "]", secret, `client "ci"`, "allow_ops", "unsupported"},
		{"mode", `"sigv4_static"`, expr, secret, `auth "main"`, "mode", "invalid"},
		{"dispatch", `dispatch     = "first"`, "dispatch = " + expr, secret, `route "images_rw"`, "dispatch", "invalid"},
		{"match", `on_match     = "stop"`, "on_match = " + expr, secret, `route "images_rw"`, "on_match", "invalid"},
		{"read preference", `read_preference = "first"`, "read_preference = " + expr, secret, `route "images_rw"`, "read_preference", "invalid"},
		{"operation", `["GetObject", "PutObject"]`, "[" + expr + "]", secret, `route "images_rw"`, "operation", "unsupported"},
		{"HCL object key", "65536", "{" + expr + " = 1}", secret, "listener", "max_header_bytes", "Unsuitable value type"},
		{"HCL duplicate key", `"secretci"`, "{for v in [1, 2] : " + expr + " => v}", secret, "client", "secret_key", "Duplicate object key"},
		{"HCL index", `"secretci"`, "{public = 1}[" + expr + "]", secret, "client", "secret_key", "Invalid index"},
		{"HCL nested env", "65536", "env(" + expr + ")", "S3PROXY_DIAGNOSTIC_NESTED", "listener", "max_header_bytes", "Unsuitable value type"},
		{"HCL transformed duplicate key", `"secretci"`, "{for v in [1, 2] : \"derived-${" + expr + "}\" => v}", secret, "client", "secret_key", "Duplicate object key"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Setenv("S3PROXY_DIAGNOSTIC_VALUE", tc.value)
			t.Setenv("S3PROXY_DIAGNOSTIC_NESTED", secret)
			source := strings.Replace(exampleConfig, tc.old, tc.replacement, 1)
			if source == exampleConfig {
				t.Fatal("fixture did not replace a field")
			}
			for _, fromFile := range []bool{false, true} {
				var err error
				if fromFile {
					_, err = LoadFile(writeTmpConfig(t, source))
				} else {
					_, err = Load([]byte(source), "confidential.hcl")
				}
				assertConfidentialError(t, err, tc.value, secret)
				for _, want := range []string{tc.block, tc.field, tc.cause} {
					if !strings.Contains(err.Error(), want) {
						t.Errorf("diagnostic lacks expected context %q", want)
					}
				}
			}
		})
	}
}

func assertConfidentialError(t *testing.T, err error, values ...string) {
	t.Helper()
	if err == nil {
		t.Fatal("expected invalid configuration to fail")
	}
	for current := err; current != nil; current = errors.Unwrap(current) {
		for _, value := range append(values, "private", "quoted", "雪") {
			for _, spelling := range []string{value, strconv.Quote(value), strconv.QuoteToASCII(value)} {
				if strings.Contains(fmt.Sprintf("%v %+v %#v", current, current, current), spelling) {
					t.Fatal("diagnostic exposed environment-derived data")
				}
			}
		}
	}
}

func TestEnvCompileDiagnosticSourceLocation(t *testing.T) {
	for _, tc := range []struct{ name, old, value, position string }{
		{"timeout", `"5s"`, "private-duration", "47,22-"},
		{"endpoint", `"https://minio-a.internal"`, "https://user:private-password@example.test/%GG", "44,22-"},
		{"pattern", `"^tenant-(?P<tenant>[a-z0-9-]+)-logs$"`, "(?P<private\"name>x)", "56,13-"},
		{"read", `"30s"`, "private-duration", "16,19-"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Setenv("S3PROXY_DIAGNOSTIC_VALUE", tc.value)
			source := "\n\r\n" + strings.Replace(exampleConfig, tc.old, `env("S3PROXY_DIAGNOSTIC_VALUE")`, 1)
			_, err := Load([]byte(source), "original.hcl")
			assertConfidentialError(t, err, tc.value)
			if !strings.Contains(err.Error(), "original.hcl:"+tc.position) {
				t.Errorf("compile diagnostic lost original expression position: %s", err)
			}
		})
	}
}

func TestValidateDiagnosticsDoNotEchoAttributeValues(t *testing.T) {
	for _, tc := range []struct {
		name, field, cause string
		mutate             func(*Runtime, string)
	}{
		{"mode", "mode", "invalid", func(rt *Runtime, value string) { rt.Auth.Mode = AuthMode(value) }},
		{"parser ref", "parser", "unknown", func(rt *Runtime, value string) { rt.Routes[0].ParserRef = value }},
		{"destination ref", "destination", "unknown", func(rt *Runtime, value string) { rt.Routes[0].DestinationRefs = []string{value} }},
		{"dispatch", "dispatch", "invalid", func(rt *Runtime, value string) { rt.Routes[0].Dispatch = DispatchMode(value) }},
		{"match", "on_match", "invalid", func(rt *Runtime, value string) { rt.Routes[0].OnMatch = MatchMode(value) }},
		{"read preference", "read_preference", "invalid", func(rt *Runtime, value string) { rt.Routes[0].ReadPreference = ReadPreference(value) }},
		{"operation", "operation", "unsupported", func(rt *Runtime, value string) { rt.Routes[0].Operations = []string{value} }},
		{"bucket ref", "route", "unknown", func(rt *Runtime, value string) { rt.Buckets[0].RouteRef = value }},
		{"duplicate visible name", "visible_name", "duplicate", func(rt *Runtime, value string) {
			rt.Buckets[0].VisibleName = value
			rt.Buckets = append(rt.Buckets, VirtualBucket{Name: "other", VisibleName: value, RouteRef: "images_rw"})
		}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			rt, err := Load([]byte(exampleConfig), "public.hcl")
			if err != nil {
				t.Fatal(err)
			}
			const value = "private\"value\n\\雪"
			tc.mutate(rt, value)
			err = Validate(rt)
			assertConfidentialError(t, err, value)
			if !strings.Contains(err.Error(), tc.field) || !strings.Contains(err.Error(), tc.cause) {
				t.Fatal("validation diagnostic lost field/cause")
			}
		})
	}
}

func TestEnvCompiledValuesRemainLiteral(t *testing.T) {
	const literal = "${literal}%{ marker }\"quoted\"\n\\雪"
	const keyTemplate = "prefix/${literal}/%{ marker }/{{ .Key }}"
	source := exampleConfig
	for _, tc := range []struct{ name, old, value string }{
		{"ENDPOINT", `"https://minio-a.internal"`, "https://example.test/base%2Fpath?key=value"},
		{"TIMEOUT", `"5s"`, "1m2s"},
		{"PATTERN", `"^tenant-(?P<tenant>[a-z0-9-]+)-logs$"`, `^tenant-(?P<tenant>[a-z0-9-]+)-logs$`},
		{"PREFIX", `"assets/"`, literal},
		{"TEMPLATE", `bucket             = "images-store"`, keyTemplate},
		{"CREDENTIAL", `credentials      = "primary"`, "credential.static.primary"},
		{"MODE", `"sigv4_static"`, "sigv4_static"},
	} {
		t.Setenv("S3PROXY_LITERAL_"+tc.name, tc.value)
		replacement := `env("S3PROXY_LITERAL_` + tc.name + `")`
		switch tc.name {
		case "TEMPLATE":
			replacement = tc.old + "\n    key_template = " + replacement
		case "CREDENTIAL":
			replacement = "credentials = " + replacement
		}
		source = strings.Replace(source, tc.old, replacement, 1)
	}
	rt, err := LoadFile(writeTmpConfig(t, source))
	if err != nil {
		t.Fatal("valid environment configuration failed")
	}
	if rt.Targets["primary"].Timeout != 62*time.Second || rt.Targets["primary"].Endpoint != "https://example.test/base%2Fpath?key=value" ||
		rt.Routes[0].Rewrite.PrependKeyPrefix != literal || rt.Routes[0].Rewrite.KeyTemplate != keyTemplate ||
		rt.Routes[0].Rewrite.CompiledTemplate == nil || !rt.Parsers["tenant_logs"].Regex.MatchString("tenant-blue-logs") ||
		rt.Targets["primary"].Credentials.Name != "primary" || rt.Auth.Mode != AuthModeSigV4Static {
		t.Fatal("successful environment values or compilation changed")
	}
	if err := Validate(rt); err != nil {
		t.Fatal("revalidation of valid environment configuration failed")
	}
}

func TestEnvDecodeDiagnosticLocationAndPublicDetail(t *testing.T) {
	t.Setenv("S3PROXY_DIAGNOSTIC_VALUE", "private\"value\n\\雪")
	source := "\n\r\n  listener \"http\" \"public\" {\n    address = \":8080\"\n    max_header_bytes = env(\"S3PROXY_DIAGNOSTIC_VALUE\")\n    addressing { path_style = true }\n}\n"
	_, err := Load([]byte(source), "original.hcl")
	assertConfidentialError(t, err, "private\"value\n\\雪")
	for _, want := range []string{"original.hcl:5,24-", "max_header_bytes", `listener "http" "public"`, "Unsuitable value type"} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("missing context %q", want)
		}
	}
	source = strings.Replace(source, `env("S3PROXY_DIAGNOSTIC_VALUE")`, `"not-a-number"`, 1)
	_, err = Load([]byte(source), "public.hcl")
	if err == nil || !strings.Contains(err.Error(), "public.hcl:5,25-") || !strings.Contains(err.Error(), "a number is required") {
		t.Fatal("ordinary public HCL error lost its source location or detailed cause")
	}
}
