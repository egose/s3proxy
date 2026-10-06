package config

import (
	"errors"
	"strconv"
	"strings"
	"testing"
)

func TestListenerAddressInvalid(t *testing.T) {
	fileT := t
	for _, tc := range []struct{ name, address, cause string }{
		{"empty", "", "address is required"},
		{"missing port", "127.0.0.1", "address must use TCP host:port syntax"},
		{"URL", "http://127.0.0.1:8080", "address must use TCP host:port syntax"},
		{"URL without port", "http://private-listener.invalid", "address must use TCP host:port syntax"},
		{"high port", ":65536", "address numeric port must be between 0 and 65535"},
		{"negative port", "private-listener.invalid:-1", "address numeric port must be between 0 and 65535"},
		{"signed high port", "private-listener.invalid:+65536", "address numeric port must be between 0 and 65535"},
		{"overflow", "private-listener.invalid:999999999999999999999999999999", "address numeric port must be between 0 and 65535"},
		{"overflow before service suffix", "private-listener.invalid:10737418240service", "address numeric port must be between 0 and 65535"},
		{"uint32 overflow before service suffix", "private-listener.invalid:4294967296service", "address numeric port must be between 0 and 65535"},
		{"negative overflow before service suffix", "private-listener.invalid:-10737418240service", "address numeric port must be between 0 and 65535"},
		{"numeric uint32 wrap", "private-listener.invalid:4294967300", "address numeric port must be between 0 and 65535"},
		{"private missing port", "private\"quoted\n\\雪", "address must use TCP host:port syntax"},
		{"private URL", "http://private-listener.invalid:8080", "address must use TCP host:port syntax"},
		{"private high port", "private-listener.invalid:65536", "address numeric port must be between 0 and 65535"},
		{"unbracketed IPv6", "2001:db8::1:8080", "address must use TCP host:port syntax"},
		{"missing bracket", "[fe80::1%private-zone:8080", "address must use TCP host:port syntax"},
		{"missing IPv6 port", "[fe80::1%private-zone]", "address must use TCP host:port syntax"},
		{"extra bracket", "[::1]]:8080", "address must use TCP host:port syntax"},
		{"bracket in port", "private-listener.invalid:[8080]", "address must use TCP host:port syntax"},
		{"extra port", "private-listener.invalid:8080:90", "address must use TCP host:port syntax"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			check := func(t *testing.T, err error) {
				t.Helper()
				assertConfidentialError(t, err)
				for errors.Unwrap(err) != nil {
					err = errors.Unwrap(err)
				}
				if err.Error() != `listener.http "public": `+tc.cause {
					t.Fatal("listener error must contain only the public block label and safe field/cause")
				}
			}
			t.Run("direct Validate", func(t *testing.T) {
				rt, err := Load([]byte(exampleConfig), "listener.hcl")
				if err != nil {
					t.Fatal(err)
				}
				rt.Listener.Address = tc.address
				check(t, Validate(rt))
			})
			for _, literal := range []bool{false, true} {
				t.Run("Load literal="+strconv.FormatBool(literal), func(t *testing.T) {
					t.Setenv("S3PROXY_LISTENER_ADDRESS", tc.address)
					expr := `env("S3PROXY_LISTENER_ADDRESS")`
					if literal {
						expr = strconv.Quote(tc.address)
					}
					source := strings.Replace(exampleConfig, `":8080"`, expr, 1)
					_, err := Load([]byte(source), "listener.hcl")
					check(t, err)
					_, err = LoadFile(writeTmpConfig(fileT, source))
					check(t, err)
				})
			}
		})
	}
}

func TestListenerAddressOfflineContract(t *testing.T) {
	for _, address := range []string{
		":0", ":1", ":65535", "0.0.0.0:8080", "127.0.0.1:8080",
		"[::]:0", "[::1]:8080", "[fe80::1%offline-zone]:8080", "[fe80::1%123]:8080",
		"[::ffff:192.0.2.1]:8080", "192.0.2.1:8080", "listener.invalid:8080",
		"localhost:http", ":https", "listener.invalid:offline-service-does-not-exist",
		":", "127.0.0.1:", "[::1]:", "listener.invalid:",
		":+8080", ":-0", ":+", ":-", ":00065535", ":00000000000000000000000000000000001",
		"[localhost]:8080", "host_with_underscore:8080", "listener.invalid:123service",
		"listener.invalid:65536service", "listener.invalid:1073741824service",
		"listener.invalid:4294967295service", "listener.invalid:-1073741824service",
		"listener.invalid:4294967300service",
	} {
		t.Run(address, func(t *testing.T) {
			source := strings.Replace(exampleConfig, `":8080"`, strconv.Quote(address), 1)
			rt, err := Load([]byte(source), "listener.hcl")
			if err != nil {
				t.Fatalf("offline address rejected: %v", err)
			}
			if rt.Listener.Address != address {
				t.Fatal("validation changed listener address")
			}
			if err := Validate(rt); err != nil {
				t.Fatalf("direct validation rejected offline address: %v", err)
			}
		})
	}
}
