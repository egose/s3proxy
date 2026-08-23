package router

import (
	"strings"
	"testing"

	"github.com/egose/s3proxy/internal/config"
	"github.com/egose/s3proxy/internal/requestctx"
	"github.com/egose/s3proxy/internal/s3ops"
)

func BenchmarkResolvePathPrefix(b *testing.B) {
	r := newTestResolver(buildTestRuntime())
	ctx := &requestctx.Context{RawPath: "/images/cat.jpg", Bucket: "images", Key: "cat.jpg"}

	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		matches, err := r.Resolve(ctx, s3ops.OpGetObject)
		if err != nil {
			b.Fatal(err)
		}
		if len(matches) != 1 {
			b.Fatalf("matches = %d, want 1", len(matches))
		}
	}
}

func BenchmarkMatchParserPathPrefixLegacyCaptureMap(b *testing.B) {
	p := config.Parser{Name: "images", Kind: config.ParserPathPrefix, Prefix: "/images"}
	ctx := &requestctx.Context{RawPath: "/images/cat.jpg", Bucket: "images", Key: "cat.jpg"}

	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		ok, captures, err := legacyMatchParser(p, ctx)
		if err != nil {
			b.Fatal(err)
		}
		if !ok || len(captures) != 0 {
			b.Fatal("unexpected match result")
		}
	}
}

func BenchmarkMatchParserPathPrefix(b *testing.B) {
	p := config.Parser{Name: "images", Kind: config.ParserPathPrefix, Prefix: "/images"}
	ctx := &requestctx.Context{RawPath: "/images/cat.jpg", Bucket: "images", Key: "cat.jpg"}

	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		ok, captures, err := matchParser(p, ctx)
		if err != nil {
			b.Fatal(err)
		}
		if !ok || len(captures) != 0 {
			b.Fatal("unexpected match result")
		}
	}
}

func BenchmarkResolveBucketRegex(b *testing.B) {
	r := newTestResolver(buildTestRuntime())
	ctx := &requestctx.Context{Bucket: "tenant-acme-logs", Key: "file.log"}

	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		matches, err := r.Resolve(ctx, s3ops.OpGetObject)
		if err != nil {
			b.Fatal(err)
		}
		if matches[0].Captures["tenant"] != "acme" {
			b.Fatal("missing tenant capture")
		}
	}
}

func legacyMatchParser(p config.Parser, ctx *requestctx.Context) (bool, map[string]string, error) {
	captures := make(map[string]string)
	switch p.Kind {
	case config.ParserPathPrefix:
		if ctx.RawPath == p.Prefix || strings.HasPrefix(ctx.RawPath, p.Prefix+"/") {
			return true, captures, nil
		}
		return false, nil, nil
	case config.ParserBucketExact:
		if ctx.Bucket == p.Bucket {
			return true, captures, nil
		}
		return false, nil, nil
	case config.ParserHostSuffix:
		if hostMatchesSuffix(ctx.Host, p.Suffix) {
			return true, captures, nil
		}
		return false, nil, nil
	}
	return matchParser(p, ctx)
}
