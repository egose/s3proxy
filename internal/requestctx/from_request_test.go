package requestctx

import (
	"net/http"
	"net/url"
	"testing"

	"github.com/egose/s3proxy/internal/config"
)

func TestFromRequest_PathStyle(t *testing.T) {
	cfg := config.Addressing{PathStyle: true}
	r := &http.Request{
		Host:   "localhost:8080",
		URL:    &url.URL{Path: "/mybucket/path/to/object.txt"},
		Method: "GET",
	}
	ctx, err := FromRequest(r, cfg)
	if err != nil {
		t.Fatal(err)
	}
	if ctx.Bucket != "mybucket" {
		t.Errorf("expected bucket 'mybucket', got %q", ctx.Bucket)
	}
	if ctx.Key != "path/to/object.txt" {
		t.Errorf("expected key 'path/to/object.txt', got %q", ctx.Key)
	}
	if ctx.AddressingMode != AddressingPathStyle {
		t.Errorf("expected path_style addressing")
	}
}

func TestFromRequest_PathStyle_RootPath(t *testing.T) {
	cfg := config.Addressing{PathStyle: true}
	r := &http.Request{
		Host:   "localhost:8080",
		URL:    &url.URL{Path: "/"},
		Method: "GET",
	}
	ctx, err := FromRequest(r, cfg)
	if err != nil {
		t.Fatal(err)
	}
	if ctx.Bucket != "" || ctx.Key != "" {
		t.Errorf("expected empty bucket and key, got %q/%q", ctx.Bucket, ctx.Key)
	}
}

func TestFromRequest_VirtualHosted(t *testing.T) {
	cfg := config.Addressing{
		PathStyle:     true,
		VirtualHosted: true,
		HostSuffixes:  []string{"s3proxy.example.com"},
	}
	r := &http.Request{
		Host:   "mybucket.s3proxy.example.com",
		URL:    &url.URL{Path: "/path/to/object.txt"},
		Method: "GET",
		Header: http.Header{},
	}
	ctx, err := FromRequest(r, cfg)
	if err != nil {
		t.Fatal(err)
	}
	if ctx.Bucket != "mybucket" {
		t.Errorf("expected bucket 'mybucket', got %q", ctx.Bucket)
	}
	if ctx.Key != "path/to/object.txt" {
		t.Errorf("expected key 'path/to/object.txt', got %q", ctx.Key)
	}
	if ctx.AddressingMode != AddressingVirtualHosted {
		t.Errorf("expected virtual_hosted addressing")
	}
}

func TestFromRequest_BucketOnlyNoKey(t *testing.T) {
	cfg := config.Addressing{PathStyle: true}
	r := &http.Request{
		Host:   "localhost:8080",
		URL:    &url.URL{Path: "/mybucket"},
		Method: "GET",
	}
	ctx, err := FromRequest(r, cfg)
	if err != nil {
		t.Fatal(err)
	}
	if ctx.Bucket != "mybucket" {
		t.Errorf("expected bucket 'mybucket', got %q", ctx.Bucket)
	}
	if ctx.Key != "" {
		t.Errorf("expected empty key, got %q", ctx.Key)
	}
}

func TestFromRequest_PreservesEscapedPathKey(t *testing.T) {
	cfg := config.Addressing{PathStyle: true}
	r := &http.Request{
		Host:   "localhost:8080",
		URL:    &url.URL{Path: "/mybucket/path/with/slash.txt", RawPath: "/mybucket/path%2Fwith%2Fslash.txt"},
		Method: "GET",
	}
	ctx, err := FromRequest(r, cfg)
	if err != nil {
		t.Fatal(err)
	}
	if got, want := ctx.RawPath, "/mybucket/path%2Fwith%2Fslash.txt"; got != want {
		t.Fatalf("RawPath = %q, want %q", got, want)
	}
	if got, want := ctx.Key, "path%2Fwith%2Fslash.txt"; got != want {
		t.Fatalf("Key = %q, want %q", got, want)
	}
}

func TestFromRequest_RejectsPathStyleWhenDisabled(t *testing.T) {
	cfg := config.Addressing{
		VirtualHosted: true,
		HostSuffixes:  []string{"s3proxy.example.com"},
	}
	r := &http.Request{
		Host:   "localhost:8080",
		URL:    &url.URL{Path: "/mybucket/path/to/object.txt"},
		Method: "GET",
	}

	_, err := FromRequest(r, cfg)
	if !IsNoAddressingMatch(err) {
		t.Fatalf("expected no-addressing-match error, got %v", err)
	}
}
