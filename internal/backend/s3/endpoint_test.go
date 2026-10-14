package s3

import (
	"net/http/httptest"
	"testing"

	"github.com/egose/s3proxy/internal/config"
)

func TestBuildTargetURL_ValidatedEndpoints(t *testing.T) {
	for _, tc := range []struct {
		name, endpoint, host string
		pathStyle            bool
	}{
		{"DNS path", "https://storage.invalid:9000", "storage.invalid:9000", true},
		{"DNS virtual", "https://storage.invalid:9000", "bucket.storage.invalid:9000", false},
		{"IPv4 path", "http://192.0.2.1:9000", "192.0.2.1:9000", true},
		{"IPv4 virtual", "http://192.0.2.1:9000", "bucket.192.0.2.1:9000", false},
		{"IPv6", "https://[2001:db8::1]", "[2001:db8::1]", true},
		{"IPv6 port", "http://[::1]:9000", "[::1]:9000", true},
		{"IPv6 zone", "http://[fe80::1%25offline-zone]:9000", "[fe80::1%offline-zone]:9000", true},
		{"mapped IPv6", "http://[::ffff:192.0.2.1]:9000", "[::ffff:192.0.2.1]:9000", true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			endpoint := tc.endpoint + "/base%2fpath/%3Fquery%23fragment/%E9%9B%AA"
			targets := testTargets(t, endpoint, func(target *config.S3Target) { target.ForcePathStyle = tc.pathStyle })
			rt := &config.Runtime{
				Listener: config.Listener{Name: "public", Address: ":0", Addressing: config.Addressing{PathStyle: true}},
				Auth:     config.Auth{Mode: config.AuthModeNone},
				Targets:  targets,
			}
			if err := config.Validate(rt); err != nil {
				t.Fatal("supported outbound endpoint failed configuration validation")
			}
			target := targets["primary"]
			before := *target.EndpointURL
			src := httptest.NewRequest("GET", "/source/object", nil)
			u, err := buildTargetURL(target, "bucket", "key%2Fpart", src)
			if err != nil {
				t.Fatal("validated endpoint failed outbound URL construction")
			}
			path, rawPath := "/base/path/?query#fragment/雪/", "/base%2fpath/%3Fquery%23fragment/%E9%9B%AA/"
			if tc.pathStyle {
				path, rawPath = path+"bucket/", rawPath+"bucket/"
			}
			if u.Scheme != before.Scheme || u.Host != tc.host || u.Path != path+"key/part" || u.EscapedPath() != rawPath+"key%2Fpart" {
				t.Fatal("outbound URL changed supported scheme/host/port/addressing or escaped base/key bytes")
			}
			if *target.EndpointURL != before || target.Endpoint != endpoint {
				t.Fatal("outbound URL construction mutated the validated endpoint")
			}
		})
	}
}
