package config

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

const exampleConfig = `
listener "http" "public" {
  address = ":8080"
  max_header_bytes = 65536
  replay_body_max_bytes = 1048576

  addressing {
    path_style     = true
    virtual_hosted = true
    host_suffixes  = ["s3proxy.example.com"]
  }

  timeouts {
    read        = "30s"
    read_header = "10s"
    idle        = "60s"
    write       = "0s"
  }
}

auth "main" {
  mode = "sigv4_static"

  client "ci" {
    access_key = "AKIACI123"
    secret_key = "secretci"

    allow_routes = [
      "route.images_rw",
    ]

    visible_buckets = ["images"]
  }
}

credential "static" "primary" {
  access_key = "AKIAPRIMARY"
  secret_key = "secretprimary"
}

target "s3" "primary" {
  endpoint         = "https://minio-a.internal"
  region           = "us-east-1"
  force_path_style = true
  timeout          = "5s"
  credentials      = "primary"
}

parser "path_prefix" "images" {
  prefix = "/images"
}

parser "bucket_regex" "tenant_logs" {
  pattern = "^tenant-(?P<tenant>[a-z0-9-]+)-logs$"
}

route "images_rw" {
  parser       = "images"
  operations   = ["GetObject", "PutObject"]
  destinations = ["primary"]
  dispatch     = "first"
  on_match     = "stop"
  read_preference = "first"

  rewrite {
    strip_path_prefix  = "/images"
    prepend_key_prefix = "assets/"
    bucket             = "images-store"
  }
}

bucket "images" {
  visible_name = "images"
  route        = "images_rw"
}
`

func writeTmpConfig(t *testing.T, content string) string {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, "config.hcl")
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestLoadFile_ValidMinimal(t *testing.T) {
	path := writeTmpConfig(t, exampleConfig)
	rt, err := LoadFile(path)
	if err != nil {
		t.Fatalf("LoadFile failed: %v", err)
	}
	if rt.Listener.Address != ":8080" {
		t.Errorf("expected address :8080, got %q", rt.Listener.Address)
	}
	if got, want := rt.Listener.MaxHeaderBytes, 65536; got != want {
		t.Fatalf("expected max_header_bytes %d, got %d", want, got)
	}
	if got, want := rt.Listener.ReplayBodyMaxBytes, int64(1048576); got != want {
		t.Fatalf("expected replay_body_max_bytes %d, got %d", want, got)
	}
	if got, want := rt.Listener.Timeouts.Read, 30*time.Second; got != want {
		t.Fatalf("expected read timeout %s, got %s", want, got)
	}
	if rt.Auth.Mode != AuthModeSigV4Static {
		t.Errorf("expected sigv4_static mode")
	}
	if got, want := rt.Auth.Clients["ci"].AllowRoutes[0], "images_rw"; got != want {
		t.Fatalf("expected allow_routes[0] %q, got %q", want, got)
	}
	if _, ok := rt.Targets["primary"]; !ok {
		t.Error("expected target 'primary'")
	}
	if got, want := rt.Targets["primary"].Timeout, 5*time.Second; got != want {
		t.Fatalf("expected target timeout %s, got %s", want, got)
	}
	if rt.Targets["primary"].EndpointURL == nil {
		t.Fatal("expected parsed endpoint URL")
	}
	if _, ok := rt.Parsers["images"]; !ok {
		t.Error("expected parser 'images'")
	}
	if _, ok := rt.Parsers["tenant_logs"]; !ok {
		t.Error("expected parser 'tenant_logs'")
	}
	if rt.Parsers["tenant_logs"].Regex == nil {
		t.Fatal("expected parser 'tenant_logs' to have compiled regex")
	}
	if rt.Routes[0].Rewrite.CompiledTemplate != nil {
		t.Fatal("expected first route to have no compiled key_template")
	}
	if len(rt.Routes) != 1 {
		t.Fatalf("expected 1 route, got %d", len(rt.Routes))
	}
	if rt.Routes[0].ParserRef != "images" {
		t.Errorf("expected parser ref 'images', got %q", rt.Routes[0].ParserRef)
	}
	if rt.Routes[0].Rewrite.StripPathPrefix != "/images" {
		t.Errorf("expected strip_path_prefix /images, got %q", rt.Routes[0].Rewrite.StripPathPrefix)
	}
}

func TestLoadFile_NoListener(t *testing.T) {
	cfg := `
auth "main" {
  mode = "none"
}
`
	_, err := LoadFile(writeTmpConfig(t, cfg))
	if err == nil {
		t.Fatal("expected error for missing listener")
	}
}

func TestValidate_DuplicateVisibleBucket(t *testing.T) {
	cfg := exampleConfig + `
bucket "images_dup" {
  visible_name = "images"
  route        = "images_rw"
}
`
	_, err := LoadFile(writeTmpConfig(t, cfg))
	if err == nil {
		t.Fatal("expected error for duplicate visible bucket")
	}
}

func TestValidate_DuplicateClientAccessKey(t *testing.T) {
	cfg := `
listener "http" "public" {
  address = ":8080"
  addressing { path_style = true }
}

auth "main" {
  mode = "sigv4_static"
  client "a" { access_key = "SAME" secret_key = "s1" }
  client "b" { access_key = "SAME" secret_key = "s2" }
}

credential "static" "c" {
  access_key = "k"
  secret_key = "s"
}
target "s3" "t" {
  endpoint   = "https://e"
  region     = "r"
  credentials = "c"
}

parser "path_prefix" "p" { prefix = "/p" }
route "r" {
  parser       = "p"
  operations   = ["GetObject"]
  destinations = ["t"]
  dispatch     = "first"
  on_match     = "stop"
}
`
	_, err := LoadFile(writeTmpConfig(t, cfg))
	if err == nil {
		t.Fatal("expected error for duplicate access key")
	}
}

func TestLoadFile_AuthNone(t *testing.T) {
	cfg := `
listener "http" "public" {
  address = ":8080"
  addressing { path_style = true }
}

auth "main" {
  mode = "none"
}

credential "static" "c" {
  access_key = "k"
  secret_key = "s"
}
target "s3" "t" {
  endpoint    = "https://e"
  region      = "r"
  credentials = "c"
}

parser "path_prefix" "p" { prefix = "/p" }
route "r" {
  parser       = "p"
  operations   = ["GetObject"]
  destinations = ["t"]
  dispatch     = "first"
  on_match     = "stop"
}
`
	rt, err := LoadFile(writeTmpConfig(t, cfg))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if rt.Auth.Mode != AuthModeNone {
		t.Errorf("expected none mode")
	}
}

func TestLoadFile_EnvExpansion(t *testing.T) {
	os.Setenv("S3PROXY_TEST_KEY", "envkey")
	defer os.Unsetenv("S3PROXY_TEST_KEY")

	cfg := `
listener "http" "public" {
  address = ":8080"
  addressing { path_style = true }
}

auth "main" {
  mode = "none"
}

credential "static" "c" {
  access_key = env("S3PROXY_TEST_KEY")
  secret_key = "s"
}
target "s3" "t" {
  endpoint    = "https://e"
  region      = "r"
  credentials = "c"
}

parser "path_prefix" "p" { prefix = "/p" }
route "r" {
  parser       = "p"
  operations   = ["GetObject"]
  destinations = ["t"]
  dispatch     = "first"
  on_match     = "stop"
}
`
	rt, err := LoadFile(writeTmpConfig(t, cfg))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if rt.Targets["t"].Credentials.AccessKey != "envkey" {
		t.Errorf("expected env-expanded access key 'envkey', got %q", rt.Targets["t"].Credentials.AccessKey)
	}
}

func TestLoadFile_EnvExpansionEscapesSpecialChars(t *testing.T) {
	want := "line1\n\"quoted\"\\tail"
	os.Setenv("S3PROXY_TEST_ESCAPED", want)
	defer os.Unsetenv("S3PROXY_TEST_ESCAPED")

	cfg := `
listener "http" "public" {
  address = ":8080"
  addressing { path_style = true }
}

auth "main" {
  mode = "none"
}

credential "static" "c" {
  access_key = env("S3PROXY_TEST_ESCAPED")
  secret_key = "s"
}
target "s3" "t" {
  endpoint    = "https://e"
  region      = "r"
  credentials = "c"
}

parser "path_prefix" "p" { prefix = "/p" }
route "r" {
  parser       = "p"
  operations   = ["GetObject"]
  destinations = ["t"]
  dispatch     = "first"
  on_match     = "stop"
}
`
	rt, err := LoadFile(writeTmpConfig(t, cfg))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got := rt.Targets["t"].Credentials.AccessKey; got != want {
		t.Fatalf("access key = %q, want %q", got, want)
	}
}

func TestLoadFile_CompilesKeyTemplate(t *testing.T) {
	cfg := `
listener "http" "public" {
  address = ":8080"
  addressing { path_style = true }
}

auth "main" { mode = "none" }

credential "static" "c" {
  access_key = "k"
  secret_key = "s"
}
target "s3" "t" {
  endpoint    = "https://e"
  region      = "r"
  credentials = "c"
}

parser "bucket_regex" "p" { pattern = "^tenant-(?P<tenant>[a-z0-9-]+)-logs$" }
route "r" {
  parser       = "p"
  operations   = ["GetObject"]
  destinations = ["t"]
  dispatch     = "first"
  on_match     = "stop"
  rewrite {
    bucket       = "shared-logs"
    key_template = "{{ .Captures.tenant }}/{{ .Key }}"
  }
}
`
	rt, err := LoadFile(writeTmpConfig(t, cfg))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if rt.Routes[0].Rewrite.CompiledTemplate == nil {
		t.Fatal("expected compiled key_template")
	}
}

func TestLoadFile_RejectsInvalidKeyTemplate(t *testing.T) {
	cfg := `
listener "http" "public" {
  address = ":8080"
  addressing { path_style = true }
}

auth "main" { mode = "none" }

credential "static" "c" {
  access_key = "k"
  secret_key = "s"
}
target "s3" "t" {
  endpoint    = "https://e"
  region      = "r"
  credentials = "c"
}

parser "path_prefix" "p" { prefix = "/p" }
route "r" {
  parser       = "p"
  operations   = ["GetObject"]
  destinations = ["t"]
  dispatch     = "first"
  on_match     = "stop"
  rewrite {
    key_template = "{{ .Captures.tenant }"
  }
}
`
	_, err := LoadFile(writeTmpConfig(t, cfg))
	if err == nil {
		t.Fatal("expected error for invalid key_template")
	}
}

func TestLoadFile_AllowsOnMatchContinueForWriteRoutes(t *testing.T) {
	cfg := `
listener "http" "public" {
  address = ":8080"
  addressing { path_style = true }
}

auth "main" { mode = "none" }

credential "static" "c" {
  access_key = "k"
  secret_key = "s"
}
target "s3" "t" {
  endpoint    = "https://e"
  region      = "r"
  credentials = "c"
}

parser "path_prefix" "p" { prefix = "/p" }
route "r" {
  parser       = "p"
  operations   = ["PutObject"]
  destinations = ["t"]
  dispatch     = "first"
  on_match     = "continue"
}
`
	rt, err := LoadFile(writeTmpConfig(t, cfg))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got, want := rt.Routes[0].OnMatch, MatchContinue; got != want {
		t.Fatalf("OnMatch = %q, want %q", got, want)
	}
	if got, want := rt.Routes[0].Operations[0], "PutObject"; got != want {
		t.Fatalf("operation = %q, want %q", got, want)
	}
}

func TestLoadFile_RejectsOnMatchContinueForReadRoutes(t *testing.T) {
	cfg := `
listener "http" "public" {
  address = ":8080"
  addressing { path_style = true }
}

auth "main" { mode = "none" }

credential "static" "c" {
  access_key = "k"
  secret_key = "s"
}
target "s3" "t" {
  endpoint    = "https://e"
  region      = "r"
  credentials = "c"
}

parser "path_prefix" "p" { prefix = "/p" }
route "r" {
  parser       = "p"
  operations   = ["GetObject"]
  destinations = ["t"]
  dispatch     = "first"
  on_match     = "continue"
}
`
	_, err := LoadFile(writeTmpConfig(t, cfg))
	if err == nil {
		t.Fatal("expected error for on_match=continue on read routes")
	}
}

func TestLoadFile_AllowsOrderedFailover(t *testing.T) {
	cfg := `
listener "http" "public" {
  address = ":8080"
  addressing { path_style = true }
}

auth "main" { mode = "none" }

credential "static" "c" {
  access_key = "k"
  secret_key = "s"
}
target "s3" "t" {
  endpoint    = "https://e"
  region      = "r"
  credentials = "c"
}

parser "path_prefix" "p" { prefix = "/p" }
route "r" {
  parser          = "p"
  operations      = ["GetObject"]
  destinations    = ["t"]
  dispatch        = "first"
  on_match        = "stop"
  read_preference = "ordered_failover"
}
`
	rt, err := LoadFile(writeTmpConfig(t, cfg))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got, want := rt.Routes[0].ReadPreference, ReadOrderedFailover; got != want {
		t.Fatalf("ReadPreference = %q, want %q", got, want)
	}
}

func TestLoadFile_RejectsDispatchAllForReadRoutes(t *testing.T) {
	cfg := `
listener "http" "public" {
  address = ":8080"
  addressing { path_style = true }
}

auth "main" { mode = "none" }

credential "static" "c" {
  access_key = "k"
  secret_key = "s"
}
target "s3" "t" {
  endpoint    = "https://e"
  region      = "r"
  credentials = "c"
}

parser "path_prefix" "p" { prefix = "/p" }
route "r" {
  parser       = "p"
  operations   = ["GetObject"]
  destinations = ["t"]
  dispatch     = "all"
  on_match     = "stop"
}
`
	_, err := LoadFile(writeTmpConfig(t, cfg))
	if err == nil {
		t.Fatal("expected error for dispatch=all on read route")
	}
}

func TestLoadFile_AllowsDispatchAllForWriteRoutes(t *testing.T) {
	cfg := `
listener "http" "public" {
  address = ":8080"
  addressing { path_style = true }
}

auth "main" { mode = "none" }

credential "static" "c" {
  access_key = "k"
  secret_key = "s"
}
target "s3" "t" {
  endpoint    = "https://e"
  region      = "r"
  credentials = "c"
}

parser "path_prefix" "p" { prefix = "/p" }
route "r" {
  parser       = "p"
  operations   = ["PutObject", "DeleteObject"]
  destinations = ["t"]
  dispatch     = "all"
  on_match     = "stop"
}
`
	rt, err := LoadFile(writeTmpConfig(t, cfg))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got, want := rt.Routes[0].Dispatch, DispatchAll; got != want {
		t.Fatalf("Dispatch = %q, want %q", got, want)
	}
}

func TestLoadFile_AllowsDispatchAllForMixedReadWriteRoutes(t *testing.T) {
	cfg := `
listener "http" "public" {
  address = ":8080"
  addressing { path_style = true }
}

auth "main" { mode = "none" }

credential "static" "c" {
  access_key = "k"
  secret_key = "s"
}
target "s3" "t1" {
  endpoint    = "https://e1"
  region      = "r"
  credentials = "c"
}
target "s3" "t2" {
  endpoint    = "https://e2"
  region      = "r"
  credentials = "c"
}

parser "path_prefix" "p" { prefix = "/p" }
route "r" {
  parser          = "p"
  operations      = ["GetObject", "PutObject", "DeleteObject"]
  destinations    = ["t1", "t2"]
  dispatch        = "all"
  on_match        = "stop"
  read_preference = "random"
}
`
	rt, err := LoadFile(writeTmpConfig(t, cfg))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got, want := rt.Routes[0].Dispatch, DispatchAll; got != want {
		t.Fatalf("Dispatch = %q, want %q", got, want)
	}
	if got, want := rt.Routes[0].ReadPreference, ReadRandom; got != want {
		t.Fatalf("ReadPreference = %q, want %q", got, want)
	}
}

func TestLoadFile_RejectsCopyObject(t *testing.T) {
	cfg := `
listener "http" "public" {
  address = ":8080"
  addressing { path_style = true }
}

auth "main" { mode = "none" }

credential "static" "c" {
  access_key = "k"
  secret_key = "s"
}
target "s3" "t" {
  endpoint    = "https://e"
  region      = "r"
  credentials = "c"
}

parser "path_prefix" "p" { prefix = "/p" }
route "r" {
  parser       = "p"
  operations   = ["CopyObject"]
  destinations = ["t"]
  dispatch     = "first"
  on_match     = "stop"
}
`
	_, err := LoadFile(writeTmpConfig(t, cfg))
	if err == nil {
		t.Fatal("expected error for CopyObject")
	}
}

func TestLoadFile_RejectsRelativeTargetEndpoint(t *testing.T) {
	cfg := `
listener "http" "public" {
  address = ":8080"
  addressing { path_style = true }
}

auth "main" { mode = "none" }

credential "static" "c" {
  access_key = "k"
  secret_key = "s"
}
target "s3" "t" {
  endpoint    = "minio.internal"
  region      = "r"
  credentials = "c"
}

parser "path_prefix" "p" { prefix = "/p" }
route "r" {
  parser       = "p"
  operations   = ["GetObject"]
  destinations = ["t"]
  dispatch     = "first"
  on_match     = "stop"
}
`
	_, err := LoadFile(writeTmpConfig(t, cfg))
	if err == nil {
		t.Fatal("expected error for relative endpoint")
	}
}

func TestLoadFile_RejectsUnsupportedTargetEndpointScheme(t *testing.T) {
	cfg := `
listener "http" "public" {
  address = ":8080"
  addressing { path_style = true }
}

auth "main" { mode = "none" }

credential "static" "c" {
  access_key = "k"
  secret_key = "s"
}
target "s3" "t" {
  endpoint    = "ftp://minio.internal"
  region      = "r"
  credentials = "c"
}

parser "path_prefix" "p" { prefix = "/p" }
route "r" {
  parser       = "p"
  operations   = ["GetObject"]
  destinations = ["t"]
  dispatch     = "first"
  on_match     = "stop"
}
`
	_, err := LoadFile(writeTmpConfig(t, cfg))
	if err == nil {
		t.Fatal("expected error for unsupported endpoint scheme")
	}
}

func TestLoadFile_RejectsNegativeReplayBodyMaxBytes(t *testing.T) {
	cfg := `
listener "http" "public" {
  address = ":8080"
  replay_body_max_bytes = -1
  addressing { path_style = true }
}

auth "main" { mode = "none" }

credential "static" "c" {
  access_key = "k"
  secret_key = "s"
}
target "s3" "t" {
  endpoint    = "https://e"
  region      = "r"
  credentials = "c"
}

parser "path_prefix" "p" { prefix = "/p" }
route "r" {
  parser       = "p"
  operations   = ["GetObject"]
  destinations = ["t"]
  dispatch     = "first"
  on_match     = "stop"
}
`
	_, err := LoadFile(writeTmpConfig(t, cfg))
	if err == nil {
		t.Fatal("expected error for negative replay_body_max_bytes")
	}
}
