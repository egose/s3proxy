package s3

import (
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"testing"

	"github.com/egose/s3proxy/internal/config"
)

func BenchmarkForwardHeaders(b *testing.B) {
	benchmarkForwardHeaders(b, false)
}

func BenchmarkForwardHeadersLegacyPerHeaderConnectionParse(b *testing.B) {
	benchmarkForwardHeaders(b, true)
}

func benchmarkForwardHeaders(b *testing.B, legacy bool) {
	headers := make(http.Header)
	headers.Set("Connection", "X-Trace, X-Drop")
	for i := 0; i < 32; i++ {
		headers.Add("X-Test-"+strconv.Itoa(i), strings.Repeat("v", 8))
	}
	headers.Set("X-Trace", "skip")
	headers.Set("Authorization", "skip")

	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		out := make(http.Header)
		var connectionTokens map[string]struct{}
		if !legacy {
			connectionTokens = connectionHeaderTokens(headers)
		}
		for key, vals := range headers {
			forward := shouldForwardHeader(key, connectionTokens)
			if legacy {
				forward = legacyShouldForwardHeader(key, headers)
			}
			if !forward {
				continue
			}
			for _, v := range vals {
				out.Add(key, v)
			}
		}
		if len(out) == 0 {
			b.Fatal("no headers forwarded")
		}
	}
}

func BenchmarkBuildTargetURLEscapedPath(b *testing.B) {
	src := &http.Request{URL: &url.URL{}}
	target := config.S3Target{
		EndpointURL:    &url.URL{Scheme: "https", Host: "s3.internal"},
		ForcePathStyle: true,
	}
	key := "photos/2026/raw file @ 100%/snowman.jpg"

	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		u, err := buildTargetURL(target, "bucket", key, src)
		if err != nil {
			b.Fatal(err)
		}
		if u.RawPath == "" {
			b.Fatal("empty raw path")
		}
	}
}

func BenchmarkCanonicalizeEscapedPath(b *testing.B) {
	benchmarkCanonicalizeEscapedPath(b, canonicalizeEscapedPath)
}

func BenchmarkCanonicalizeEscapedPathLegacySprintf(b *testing.B) {
	benchmarkCanonicalizeEscapedPath(b, legacyCanonicalizeEscapedPath)
}

func benchmarkCanonicalizeEscapedPath(b *testing.B, fn func(string) string) {
	key := "photos/2026/raw file @ 100%/snowman.jpg"
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		if got := fn(key); got == "" {
			b.Fatal("empty path")
		}
	}
}

func legacyShouldForwardHeader(name string, headers http.Header) bool {
	if legacyIsHopByHopHeader(name, headers) {
		return false
	}
	switch strings.ToLower(name) {
	case "authorization", "x-amz-security-token", "x-amz-decoded-content-length",
		"x-amz-content-sha256", "x-amz-date", "content-length":
		return false
	case "host":
		return false
	default:
		return true
	}
}

func legacyIsHopByHopHeader(name string, headers http.Header) bool {
	canonical := strings.ToLower(name)
	switch canonical {
	case "connection", "keep-alive", "proxy-authenticate", "proxy-authorization",
		"te", "trailer", "transfer-encoding", "upgrade":
		return true
	}
	_, ok := connectionHeaderTokens(headers)[canonical]
	return ok
}

func legacyCanonicalizeEscapedPath(value string) string {
	if value == "" {
		return ""
	}
	var b strings.Builder
	b.Grow(len(value))
	for i := 0; i < len(value); i++ {
		c := value[i]
		if c == '%' && i+2 < len(value) && isHex(value[i+1]) && isHex(value[i+2]) {
			b.WriteByte('%')
			b.WriteByte(value[i+1])
			b.WriteByte(value[i+2])
			i += 2
			continue
		}
		if isSafePathByte(c) {
			b.WriteByte(c)
			continue
		}
		b.WriteString(fmt.Sprintf("%%%02X", c))
	}
	return b.String()
}
