package router

import (
	"regexp"
	"testing"

	"github.com/egose/s3proxy/internal/config"
	"github.com/egose/s3proxy/internal/requestctx"
	"github.com/egose/s3proxy/internal/s3ops"
)

func buildTestRuntime() *config.Runtime {
	return &config.Runtime{
		Listener: config.Listener{
			Name:    "public",
			Address: ":8080",
			Addressing: config.Addressing{
				PathStyle: true,
			},
		},
		Auth: config.Auth{
			Name: "main",
			Mode: config.AuthModeNone,
		},
		Targets: map[string]config.S3Target{
			"primary": {Name: "primary", Endpoint: "https://a.internal", Region: "us-east-1"},
			"replica": {Name: "replica", Endpoint: "https://b.internal", Region: "us-east-1"},
		},
		Parsers: map[string]config.Parser{
			"images":      {Name: "images", Kind: config.ParserPathPrefix, Prefix: "/images"},
			"tenant_logs": {Name: "tenant_logs", Kind: config.ParserBucketRegex, Pattern: "^tenant-(?P<tenant>[a-z0-9-]+)-logs$", Regex: regexp.MustCompile("^tenant-(?P<tenant>[a-z0-9-]+)-logs$")},
			"hosted":      {Name: "hosted", Kind: config.ParserHostSuffix, Suffix: "example.com"},
		},
		Routes: []config.Route{
			{
				Name:            "images_rw",
				ParserRef:       "images",
				Operations:      []string{"GetObject", "PutObject"},
				DestinationRefs: []string{"primary"},
				Dispatch:        config.DispatchFirst,
				OnMatch:         config.MatchStop,
				ReadPreference:  config.ReadFirst,
			},
			{
				Name:            "logs_read",
				ParserRef:       "tenant_logs",
				Operations:      []string{"GetObject", "ListObjectsV2"},
				DestinationRefs: []string{"primary", "replica"},
				Dispatch:        config.DispatchFirst,
				OnMatch:         config.MatchStop,
				ReadPreference:  config.ReadFirst,
			},
			{
				Name:            "hosted_read",
				ParserRef:       "hosted",
				Operations:      []string{"GetObject"},
				DestinationRefs: []string{"primary"},
				Dispatch:        config.DispatchFirst,
				OnMatch:         config.MatchStop,
				ReadPreference:  config.ReadFirst,
			},
		},
	}
}

func TestResolve_PathPrefix(t *testing.T) {
	rt := buildTestRuntime()
	r := NewResolver(rt)
	ctx := &requestctx.Context{
		RawPath: "/images/cat.jpg",
		Bucket:  "images",
		Key:     "cat.jpg",
	}
	matches, err := r.Resolve(ctx, s3ops.OpGetObject)
	if err != nil {
		t.Fatal(err)
	}
	if len(matches) != 1 {
		t.Fatalf("expected 1 match, got %d", len(matches))
	}
	if matches[0].Route.Name != "images_rw" {
		t.Errorf("expected route images_rw, got %s", matches[0].Route.Name)
	}
}

func TestResolve_BucketRegex(t *testing.T) {
	rt := buildTestRuntime()
	r := NewResolver(rt)
	ctx := &requestctx.Context{
		Bucket: "tenant-acme-logs",
		Key:    "file.log",
	}
	matches, err := r.Resolve(ctx, s3ops.OpGetObject)
	if err != nil {
		t.Fatal(err)
	}
	if len(matches) != 1 {
		t.Fatalf("expected 1 match, got %d", len(matches))
	}
	if matches[0].Route.Name != "logs_read" {
		t.Errorf("expected route logs_read, got %s", matches[0].Route.Name)
	}
	if matches[0].Captures["tenant"] != "acme" {
		t.Errorf("expected capture tenant=acme, got %q", matches[0].Captures["tenant"])
	}
}

func TestResolve_NoMatch(t *testing.T) {
	rt := buildTestRuntime()
	r := NewResolver(rt)
	ctx := &requestctx.Context{
		RawPath: "/unknown/path",
		Bucket:  "unknown",
	}
	_, err := r.Resolve(ctx, s3ops.OpGetObject)
	if err == nil {
		t.Fatal("expected error for no match")
	}
}

func TestResolve_OperationFilter(t *testing.T) {
	rt := buildTestRuntime()
	r := NewResolver(rt)
	ctx := &requestctx.Context{
		RawPath: "/images/cat.jpg",
		Bucket:  "images",
	}
	_, err := r.Resolve(ctx, s3ops.OpDeleteObject)
	if err == nil {
		t.Fatal("expected error for unallowed operation")
	}
}

func TestResolve_EffectiveRead(t *testing.T) {
	rt := buildTestRuntime()
	r := NewResolver(rt)
	ctx := &requestctx.Context{
		Bucket: "tenant-acme-logs",
	}
	matches, err := r.Resolve(ctx, s3ops.OpListObjectsV2)
	if err != nil {
		t.Fatal(err)
	}
	if matches[0].EffectiveRead == nil {
		t.Fatal("expected effective read target")
	}
	if matches[0].EffectiveRead.Name != "primary" {
		t.Errorf("expected effective read 'primary', got %s", matches[0].EffectiveRead.Name)
	}
}

func TestHashIndex(t *testing.T) {
	idx1 := hashIndex("bucket", "key1", 3)
	idx2 := hashIndex("bucket", "key1", 3)
	if idx1 != idx2 {
		t.Errorf("hash should be deterministic, got %d and %d", idx1, idx2)
	}
	if idx1 < 0 || idx1 >= 3 {
		t.Errorf("hash index out of range: %d", idx1)
	}
}

func TestResolve_ContinueReturnsMultipleMatches(t *testing.T) {
	rt := &config.Runtime{
		Targets: map[string]config.S3Target{
			"primary": {Name: "primary", Endpoint: "https://a.internal", Region: "us-east-1"},
			"replica": {Name: "replica", Endpoint: "https://b.internal", Region: "us-east-1"},
		},
		Parsers: map[string]config.Parser{
			"images": {Name: "images", Kind: config.ParserPathPrefix, Prefix: "/images"},
		},
		Routes: []config.Route{
			{
				Name:            "first",
				ParserRef:       "images",
				Operations:      []string{"PutObject"},
				DestinationRefs: []string{"primary"},
				Dispatch:        config.DispatchFirst,
				OnMatch:         config.MatchContinue,
				ReadPreference:  config.ReadFirst,
			},
			{
				Name:            "second",
				ParserRef:       "images",
				Operations:      []string{"PutObject"},
				DestinationRefs: []string{"replica"},
				Dispatch:        config.DispatchFirst,
				OnMatch:         config.MatchStop,
				ReadPreference:  config.ReadFirst,
			},
		},
	}
	r := NewResolver(rt)
	ctx := &requestctx.Context{RawPath: "/images/cat.jpg", Bucket: "images", Key: "cat.jpg"}

	matches, err := r.Resolve(ctx, s3ops.OpPutObject)
	if err != nil {
		t.Fatal(err)
	}
	if got, want := len(matches), 2; got != want {
		t.Fatalf("len(matches) = %d, want %d", got, want)
	}
	if got, want := matches[0].Route.Name, "first"; got != want {
		t.Fatalf("first route = %q, want %q", got, want)
	}
	if got, want := matches[1].Route.Name, "second"; got != want {
		t.Fatalf("second route = %q, want %q", got, want)
	}
}

func TestResolve_HostSuffixRequiresLabelBoundary(t *testing.T) {
	rt := buildTestRuntime()
	r := NewResolver(rt)

	_, err := r.Resolve(&requestctx.Context{Host: "badexample.com"}, s3ops.OpGetObject)
	if err == nil {
		t.Fatal("expected no host_suffix match for badexample.com")
	}

	matches, err := r.Resolve(&requestctx.Context{Host: "files.example.com"}, s3ops.OpGetObject)
	if err != nil {
		t.Fatal(err)
	}
	if got, want := matches[0].Route.Name, "hosted_read"; got != want {
		t.Fatalf("route = %q, want %q", got, want)
	}
}
