package requestpayload_test

import (
	"net/http"
	"reflect"
	"testing"

	"github.com/egose/s3proxy/internal/requestpayload"
)

func TestValidate(t *testing.T) {
	for _, tc := range []struct {
		name    string
		headers http.Header
		reject  bool
	}{
		{"nil", nil, false},
		{"encoding", http.Header{"Content-Encoding": {"aws-chunked"}}, true},
		{"encoding token list", http.Header{"content-ENCODING": {"gzip, \tAWS-CHUNKED \t, br"}}, true},
		{"encoding repeated", http.Header{"Content-Encoding": {"gzip", "aws-chunked"}}, true},
		{"encoding duplicate casing", http.Header{"Content-Encoding": {"gzip"}, "CONTENT-encoding": {" aws-chunked "}}, true},
		{"hash signed", http.Header{"X-Amz-Content-Sha256": {"STREAMING-AWS4-HMAC-SHA256-PAYLOAD"}}, true},
		{"hash unsigned trailer", http.Header{"X-Amz-Content-Sha256": {"STREAMING-UNSIGNED-PAYLOAD-TRAILER"}}, true},
		{"hash repeated", http.Header{"x-AMZ-content-SHA256": {"UNSIGNED-PAYLOAD", " \tstreaming-FUTURE_SECRET \t"}}, true},
		{"hash duplicate casing", http.Header{"X-Amz-Content-Sha256": {"UNSIGNED-PAYLOAD"}, "x-amz-content-sha256": {"STREAMING-"}}, true},
		{"trailer", http.Header{"X-Amz-Trailer": {"x-amz-checksum-crc32"}}, true},
		{"trailer repeated", http.Header{"x-AMZ-trailer": {" \t", " SECRET "}}, true},
		{"trailer duplicate casing", http.Header{"X-Amz-Trailer": {""}, "x-amz-trailer": {"x-amz-checksum-sha256"}}, true},
		{"combined", http.Header{"Content-Encoding": {"aws-chunked"}, "X-Amz-Content-Sha256": {"STREAMING-SECRET"}, "X-Amz-Trailer": {"SECRET"}}, true},
		{"ordinary encoding", http.Header{"Content-Encoding": {"gzip, br", "identity"}}, false},
		{"encoding substrings", http.Header{"Content-Encoding": {"not-aws-chunked, aws-chunked-extra, xaws-chunked"}}, false},
		{"encoding is token not parameter", http.Header{"Content-Encoding": {"aws-chunked;foo=bar", "\"aws-chunked\""}}, false},
		{"empty trailer", http.Header{"X-Amz-Trailer": {"", " \t "}}, false},
		{"ordinary hash", http.Header{"X-Amz-Content-Sha256": {"UNSIGNED-PAYLOAD", "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855"}}, false}, // pragma: allowlist secret
		{"hash substrings", http.Header{"X-Amz-Content-Sha256": {"NOT-STREAMING-PAYLOAD", "STREAMING"}}, false},
		{"ordinary chunking checksums", http.Header{"Transfer-Encoding": {"chunked"}, "X-Amz-Checksum-Crc32": {"AAAAAA=="}, "X-Amz-Decoded-Content-Length": {"5"}}, false},
		{"header name substring", http.Header{"X-Content-Encoding": {"aws-chunked"}, "X-Amz-Trailer-Other": {"SECRET"}}, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			before := tc.headers.Clone()
			err := requestpayload.Validate(tc.headers)
			if tc.reject && err != requestpayload.ErrUnsupported || !tc.reject && err != nil {
				t.Fatalf("error=%v reject=%t", err, tc.reject)
			}
			if !reflect.DeepEqual(before, tc.headers) {
				t.Fatal("headers changed")
			}
		})
	}
}
