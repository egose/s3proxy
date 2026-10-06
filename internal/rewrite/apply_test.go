package rewrite

import (
	"net/http"
	"testing"
	"text/template"

	"github.com/egose/s3proxy/internal/config"
	"github.com/egose/s3proxy/internal/requestctx"
)

func TestApply_PreservesPrependedKeyBytes(t *testing.T) {
	for _, prefix := range []string{"assets", "assets/"} {
		for _, key := range []string{"foo", "/foo", "//foo", "/", "foo//bar/../", "%2Ffoo", "a%20b/雪"} {
			t.Run(prefix+":"+key, func(t *testing.T) {
				result, err := New().Apply(&requestctx.Context{Bucket: "bucket", Key: key}, config.RewriteRule{PrependKeyPrefix: prefix}, nil)
				if err != nil {
					t.Fatal(err)
				}
				if want := "assets/" + key; result.Key != want {
					t.Fatalf("key = %q, want %q", result.Key, want)
				}
			})
		}
	}
}

func TestApply_RejectsEmptyObjectKey(t *testing.T) {
	for _, method := range []string{http.MethodGet, http.MethodHead, http.MethodPut, http.MethodDelete} {
		for name, rule := range map[string]config.RewriteRule{
			"strip key":      {StripKeyPrefix: "key"},
			"strip path":     {StripPathPrefix: "/bucket/key"},
			"empty template": {KeyTemplate: `{{ "" }}`, CompiledTemplate: template.Must(template.New("key").Parse(`{{ "" }}`))},
		} {
			t.Run(method+"/"+name, func(t *testing.T) {
				_, err := New().Apply(&requestctx.Context{Method: method, Bucket: "bucket", Key: "key", RawPath: "/bucket/key"}, rule, nil)
				if err == nil {
					t.Fatal("expected invalid rewrite error")
				}
			})
		}
	}
}

func TestApply_BucketOnlyRewritePreservesEmptyKey(t *testing.T) {
	for _, method := range []string{http.MethodGet, http.MethodHead} {
		result, err := New().Apply(&requestctx.Context{Method: method, Bucket: "bucket"}, config.RewriteRule{Bucket: "store"}, nil)
		if err != nil || result.Bucket != "store" || result.Key != "" {
			t.Fatalf("%s: result = %+v, error = %v", method, result, err)
		}
	}
}

func TestApply_StripPathPrefix(t *testing.T) {
	e := New()
	ctx := &requestctx.Context{
		RawPath: "/images/path/to/file.jpg",
		Bucket:  "images",
		Key:     "path/to/file.jpg",
	}
	rule := config.RewriteRule{
		StripPathPrefix: "/images",
	}
	result, err := e.Apply(ctx, rule, nil)
	if err != nil {
		t.Fatal(err)
	}
	if result.Key != "path/to/file.jpg" {
		t.Errorf("expected key 'path/to/file.jpg', got %q", result.Key)
	}
}

func TestApply_StripPathPrefixRequiresBoundary(t *testing.T) {
	e := New()
	ctx := &requestctx.Context{
		RawPath: "/images-old/path/to/file.jpg",
		Bucket:  "images-old",
		Key:     "path/to/file.jpg",
	}
	rule := config.RewriteRule{
		StripPathPrefix: "/images",
	}
	result, err := e.Apply(ctx, rule, nil)
	if err != nil {
		t.Fatal(err)
	}
	if got, want := result.Key, "path/to/file.jpg"; got != want {
		t.Fatalf("Key = %q, want %q", got, want)
	}
}

func TestApply_PrependKeyPrefix(t *testing.T) {
	e := New()
	ctx := &requestctx.Context{
		Bucket: "mybucket",
		Key:    "file.jpg",
	}
	rule := config.RewriteRule{
		PrependKeyPrefix: "assets/",
	}
	result, err := e.Apply(ctx, rule, nil)
	if err != nil {
		t.Fatal(err)
	}
	if result.Key != "assets/file.jpg" {
		t.Errorf("expected key 'assets/file.jpg', got %q", result.Key)
	}
}

func TestApply_BucketOverride(t *testing.T) {
	e := New()
	ctx := &requestctx.Context{
		Bucket: "original-bucket",
		Key:    "file.jpg",
	}
	rule := config.RewriteRule{
		Bucket: "new-bucket",
	}
	result, err := e.Apply(ctx, rule, nil)
	if err != nil {
		t.Fatal(err)
	}
	if result.Bucket != "new-bucket" {
		t.Errorf("expected bucket 'new-bucket', got %q", result.Bucket)
	}
}

func TestApply_KeyTemplate(t *testing.T) {
	e := New()
	ctx := &requestctx.Context{
		Bucket: "tenant-acme-logs",
		Key:    "app.log",
	}
	rule := config.RewriteRule{
		Bucket:           "shared-logs",
		KeyTemplate:      "{{ .Captures.tenant }}/{{ .Key }}",
		CompiledTemplate: template.Must(template.New("key").Option("missingkey=error").Parse("{{ .Captures.tenant }}/{{ .Key }}")),
	}
	captures := map[string]string{"tenant": "acme"}
	result, err := e.Apply(ctx, rule, captures)
	if err != nil {
		t.Fatal(err)
	}
	if result.Bucket != "shared-logs" {
		t.Errorf("expected bucket 'shared-logs', got %q", result.Bucket)
	}
	if result.Key != "acme/app.log" {
		t.Errorf("expected key 'acme/app.log', got %q", result.Key)
	}
}

func TestApply_KeyTemplateMissingCaptureFails(t *testing.T) {
	e := New()
	ctx := &requestctx.Context{
		Bucket: "tenant-acme-logs",
		Key:    "app.log",
	}
	rule := config.RewriteRule{
		Bucket:           "shared-logs",
		KeyTemplate:      "{{ .Captures.tenant }}/{{ .Key }}",
		CompiledTemplate: template.Must(template.New("key").Option("missingkey=error").Parse("{{ .Captures.tenant }}/{{ .Key }}")),
	}
	_, err := e.Apply(ctx, rule, map[string]string{})
	if err == nil {
		t.Fatal("expected key_template execution error")
	}
}

func TestApply_StripAndPrepend(t *testing.T) {
	e := New()
	ctx := &requestctx.Context{
		RawPath: "/images/cat.jpg",
		Bucket:  "images",
		Key:     "cat.jpg",
	}
	rule := config.RewriteRule{
		StripPathPrefix:  "/images",
		PrependKeyPrefix: "assets/",
		Bucket:           "images-store",
	}
	result, err := e.Apply(ctx, rule, nil)
	if err != nil {
		t.Fatal(err)
	}
	if result.Bucket != "images-store" {
		t.Errorf("expected bucket 'images-store', got %q", result.Bucket)
	}
	if result.Key != "assets/cat.jpg" {
		t.Errorf("expected key 'assets/cat.jpg', got %q", result.Key)
	}
}

func TestApply_EmptyBucketFails(t *testing.T) {
	e := New()
	ctx := &requestctx.Context{
		Bucket: "",
		Key:    "file.jpg",
	}
	_, err := e.Apply(ctx, config.RewriteRule{}, nil)
	if err == nil {
		t.Fatal("expected error for empty bucket")
	}
}

func TestApply_NoRewrite(t *testing.T) {
	e := New()
	ctx := &requestctx.Context{
		Bucket: "mybucket",
		Key:    "path/to/file.txt",
	}
	result, err := e.Apply(ctx, config.RewriteRule{}, nil)
	if err != nil {
		t.Fatal(err)
	}
	if result.Bucket != "mybucket" || result.Key != "path/to/file.txt" {
		t.Errorf("expected unchanged bucket/key, got %q/%q", result.Bucket, result.Key)
	}
}
