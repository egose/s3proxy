package config

import (
	"errors"
	"fmt"
	"net/url"
	"reflect"
	"strconv"
	"strings"
	"testing"
)

func TestTargetEndpointInvalid(t *testing.T) {
	for _, tc := range []struct {
		name, endpoint, cause string
		pathStyle             bool
	}{
		{"userinfo", "https://private-user@private-host.invalid/private-base", "endpoint userinfo is unsupported; use a credential.static block", true},
		{"userinfo password", "https://private-user:private%22quoted%5C%E9%9B%AA@private-host.invalid/private-base", "endpoint userinfo is unsupported; use a credential.static block", true},
		{"empty userinfo", "https://@private-host.invalid/private-base", "endpoint userinfo is unsupported; use a credential.static block", true},
		{"empty user and password", "https://:@private-host.invalid/private-base", "endpoint userinfo is unsupported; use a credential.static block", true},
		{"query", "https://private-host.invalid/private-base?private-query=private%22quoted%E9%9B%AA", "endpoint query is unsupported", true},
		{"invalid query escape", "https://private-host.invalid/private-base?private-query=%GG", "endpoint query is unsupported", true},
		{"bare query", "https://private-host.invalid/private-base?", "endpoint query is unsupported", true},
		{"fragment", "https://private-host.invalid/private-base#private%22quoted%E9%9B%AA", "endpoint fragment is unsupported", true},
		{"empty fragment", "https://private-host.invalid/private-base#", "endpoint fragment is unsupported", true},
		{"opaque", "https:private-opaque", "endpoint opaque URLs are unsupported", true},
		{"missing hostname", "https://:9000/private-base", "endpoint hostname is required", true},
		{"empty host and port", "https://:/private-base", "endpoint hostname is required", true},
		{"missing authority", "https:///private-base", "endpoint hostname is required", true},
		{"high port", "https://private-host.invalid:65536/private-base", "endpoint port must be a decimal number between 0 and 65535", true},
		{"padded high port", "https://private-host.invalid:00065536/private-base", "endpoint port must be a decimal number between 0 and 65535", true},
		{"uint32 overflow", "https://private-host.invalid:4294967296/private-base", "endpoint port must be a decimal number between 0 and 65535", true},
		{"numeric overflow", "https://private-host.invalid:999999999999999999999999999999/private-base", "endpoint port must be a decimal number between 0 and 65535", true},
		{"IPv6 high port", "https://[2001:db8::1]:65536/private-base", "endpoint port must be a decimal number between 0 and 65535", true},
		{"virtual IPv6", "https://[2001:db8::1]/private-base", "endpoint bracketed hosts require force_path_style = true", false},
		{"virtual IPv6 port", "http://[::1]:9000/private-base", "endpoint bracketed hosts require force_path_style = true", false},
		{"virtual IPv6 zone", "http://[fe80::1%25private-zone]:9000/private-base", "endpoint bracketed hosts require force_path_style = true", false},
		{"virtual mapped IPv6", "http://[::ffff:192.0.2.1]:9000/private-base", "endpoint bracketed hosts require force_path_style = true", false},
		{"relative", "/private-base", "endpoint must be an absolute URL", true},
		{"unsupported scheme", "ftp://private-host.invalid/private-base", "endpoint scheme must be http or https", true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			u, err := url.Parse(tc.endpoint)
			if err != nil {
				t.Fatal("fixture must reach shared endpoint validation")
			}
			t.Run("direct Validate", func(t *testing.T) {
				rt := endpointTestRuntime(t)
				target := rt.Targets["primary"]
				target.Endpoint, target.EndpointURL, target.ForcePathStyle = tc.endpoint, u, tc.pathStyle
				rt.Targets["primary"] = target
				assertEndpointError(t, Validate(rt), tc.cause, tc.endpoint)
			})
			for _, literal := range []bool{false, true} {
				t.Run("Load literal="+strconv.FormatBool(literal), func(t *testing.T) {
					t.Setenv("S3PROXY_TEST_ENDPOINT", tc.endpoint)
					expr := `env("S3PROXY_TEST_ENDPOINT")`
					if literal {
						expr = strconv.Quote(tc.endpoint)
					}
					rt, err := Load([]byte(endpointTestConfig(expr, tc.pathStyle)), "endpoint.hcl")
					assertEndpointError(t, err, tc.cause, tc.endpoint)
					if rt != nil {
						t.Fatal("invalid endpoint returned a runtime")
					}
				})
			}
		})
	}
}

func TestTargetEndpointDirectURLFields(t *testing.T) {
	for _, tc := range []struct {
		name, cause string
		mutate      func(*url.URL)
	}{
		{"userinfo", "endpoint userinfo is unsupported; use a credential.static block", func(u *url.URL) { u.User = url.User("private-user") }},
		{"query", "endpoint query is unsupported", func(u *url.URL) { u.RawQuery = "private-query" }},
		{"forced query", "endpoint query is unsupported", func(u *url.URL) { u.ForceQuery = true }},
		{"fragment", "endpoint fragment is unsupported", func(u *url.URL) { u.Fragment = "private-fragment" }},
		{"raw fragment only", "endpoint fragment is unsupported", func(u *url.URL) { u.RawFragment = "private-fragment" }},
		{"opaque with host", "endpoint opaque URLs are unsupported", func(u *url.URL) { u.Opaque = "private-opaque" }},
		{"missing hostname", "endpoint hostname is required", func(u *url.URL) { u.Host = ":9000" }},
		{"empty bracketed host", "endpoint hostname is required", func(u *url.URL) { u.Host = "[]:9000" }},
		{"nonnumeric port", "invalid endpoint: invalid URL port", func(u *url.URL) { u.Host = "private-host.invalid:private-port" }},
		{"negative port", "invalid endpoint: invalid URL port", func(u *url.URL) { u.Host = "private-host.invalid:-1" }},
		{"unbracketed IPv6", "invalid endpoint: invalid URL port", func(u *url.URL) { u.Host = "2001:db8::1" }},
		{"host escape", "invalid endpoint: invalid URL escape", func(u *url.URL) { u.Host = "private host.invalid" }},
		{"missing bracket", "invalid endpoint: invalid URL syntax", func(u *url.URL) { u.Host = "[fe80::1%private-zone" }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			rt := endpointTestRuntime(t)
			tc.mutate(rt.Targets["primary"].EndpointURL)
			assertEndpointError(t, Validate(rt), tc.cause)
		})
	}
}

func TestTargetEndpointSyntaxErrorsValueFree(t *testing.T) {
	for _, tc := range []struct{ name, endpoint, cause string }{
		{"port name", "https://private-host.invalid:private-service/private-base", "invalid URL port"},
		{"negative port", "https://private-host.invalid:-1/private-base", "invalid URL port"},
		{"signed port", "https://private-host.invalid:+9000/private-base", "invalid URL port"},
		{"port suffix", "https://private-host.invalid:999999999999999999999private/private-base", "invalid URL port"},
		{"host escape", "https://private-host%00.invalid/private-base", "invalid URL escape"},
		{"missing bracket", "https://[fe80::1%25private-zone/private-base", "invalid URL syntax"},
		{"empty bracketed host", "https://[]:9000/private-base", "invalid URL syntax"},
		{"unbracketed IPv6", "https://2001:db8::1/private-base", "invalid URL port"},
		{"path escape", "https://private-host.invalid/private%GG", "invalid URL escape"},
		{"fragment escape", "https://private-host.invalid/private-base#private%GG", "invalid URL escape"},
		{"control character", "https://private-host.invalid/private\"quoted\n\\雪", "invalid URL syntax"},
	} {
		for _, literal := range []bool{false, true} {
			t.Run(tc.name+"/literal="+strconv.FormatBool(literal), func(t *testing.T) {
				t.Setenv("S3PROXY_TEST_ENDPOINT", tc.endpoint)
				expr := `env("S3PROXY_TEST_ENDPOINT")`
				if literal {
					expr = strconv.Quote(tc.endpoint)
				}
				_, err := Load([]byte(endpointTestConfig(expr, true)), "endpoint.hcl")
				assertEndpointError(t, err, "invalid endpoint: "+tc.cause, tc.endpoint)
			})
		}
	}
}

func TestTargetEndpointOfflineContract(t *testing.T) {
	for _, tc := range []struct {
		name, endpoint string
		pathStyle      bool
	}{
		{"HTTP DNS", "http://private-host.invalid", false},
		{"HTTPS DNS", "https://private-host.invalid", false},
		{"single label", "https://private-storage", false},
		{"trailing dot and underscore", "https://private_storage.invalid.", false},
		{"mixed case", "HTTPS://Private-Storage.INVALID", false},
		{"Unicode hostname", "https://private-雪.invalid", false},
		{"IPv4 path", "http://192.0.2.1:9000", true},
		{"IPv4 virtual", "http://192.0.2.1:9000", false},
		{"zero port", "http://private-host.invalid:0", true},
		{"maximum port", "https://private-host.invalid:65535", false},
		{"empty port", "https://private-host.invalid:", false},
		{"leading zeros", "http://private-host.invalid:00065535", true},
		{"long zero padding", "http://private-host.invalid:00000000000000000000000000000000000001", true},
		{"escaped base path", "https://private-host.invalid/private%2fbase/%3Fkey%23value/%25/%E9%9B%AA/", true},
		{"virtual escaped base path", "https://private-host.invalid:9000/private%2Fbase/%3Fkey%23value/", false},
		{"IPv6", "https://[2001:db8::1]/private%2Fbase", true},
		{"IPv6 port", "http://[::1]:9000/private%20base", true},
		{"IPv6 empty port", "http://[::1]:/private-base", true},
		{"IPv6 maximum port", "https://[2001:db8::1]:65535/private-base", true},
		{"IPv6 zone", "http://[fe80::1%25private-zone]:9000/private-base", true},
		{"mapped IPv6", "http://[::ffff:192.0.2.1]:9000/private-base", true},
	} {
		for _, literal := range []bool{false, true} {
			t.Run(tc.name+"/literal="+strconv.FormatBool(literal), func(t *testing.T) {
				t.Setenv("S3PROXY_TEST_ENDPOINT", tc.endpoint)
				expr := `env("S3PROXY_TEST_ENDPOINT")`
				if literal {
					expr = strconv.Quote(tc.endpoint)
				}
				rt, err := Load([]byte(endpointTestConfig(expr, tc.pathStyle)), "endpoint.hcl")
				if err != nil {
					t.Fatal("supported endpoint failed offline loading")
				}
				wantURL, err := url.Parse(tc.endpoint)
				if err != nil {
					t.Fatal("supported endpoint fixture failed parsing")
				}
				before := rt.Targets["primary"]
				if before.Endpoint != tc.endpoint || !reflect.DeepEqual(before.EndpointURL, wantURL) || before.ForcePathStyle != tc.pathStyle {
					t.Fatal("loading changed the endpoint or addressing mode")
				}
				if err := Validate(rt); err != nil {
					t.Fatal("supported endpoint failed direct validation")
				}
				if !reflect.DeepEqual(rt.Targets["primary"], before) || !reflect.DeepEqual(rt.Targets["primary"].EndpointURL, wantURL) {
					t.Fatal("validation mutated the target")
				}
			})
		}
	}
}

func endpointTestRuntime(t *testing.T) *Runtime {
	t.Helper()
	rt, err := Load([]byte(exampleConfig), "endpoint.hcl")
	if err != nil {
		t.Fatal("valid endpoint fixture failed loading")
	}
	return rt
}

func endpointTestConfig(expr string, pathStyle bool) string {
	source := strings.Replace(exampleConfig, `"https://minio-a.internal"`, expr, 1)
	return strings.Replace(source, "force_path_style = true", "force_path_style = "+strconv.FormatBool(pathStyle), 1)
}

func assertEndpointError(t *testing.T, err error, cause string, values ...string) {
	t.Helper()
	if err == nil {
		t.Fatal("unsupported endpoint passed configuration validation")
	}
	var urlErr *url.Error
	var numErr *strconv.NumError
	if errors.As(err, &urlErr) || errors.As(err, &numErr) {
		t.Error("endpoint diagnostic retained a value-bearing parser error")
	}
	values = append(values, "private", "quoted", "雪", "999999999999999999999999999999")
	var leaf error
	for current := fmt.Errorf("caller: %w", err); current != nil; current = errors.Unwrap(current) {
		leaf = current
		for _, format := range []string{"%s", "%v", "%+v", "%#v", "%q"} {
			output := current.Error() + fmt.Sprintf(format, current)
			for _, value := range values {
				for _, spelling := range []string{value, strings.Trim(strconv.Quote(value), `"`), strings.Trim(strconv.QuoteToASCII(value), `"`)} {
					if strings.Contains(output, spelling) {
						t.Fatal("endpoint diagnostic formatting exposed a private value")
					}
				}
			}
		}
	}
	if !strings.HasSuffix(leaf.Error(), `target.s3 "primary": `+cause) {
		t.Fatal("endpoint diagnostic lost the safe target label, field, or failure category")
	}
}
