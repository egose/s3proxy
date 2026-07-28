package auth

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strconv"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	v4 "github.com/aws/aws-sdk-go-v2/aws/signer/v4"
	"github.com/egose/s3proxy/internal/config"
	"github.com/egose/s3proxy/internal/s3ops"
)

func v4signer() *v4.Signer {
	return v4.NewSigner()
}

func TestNoneAuthenticator(t *testing.T) {
	authenticator, err := NewAuthenticator(config.Auth{Mode: config.AuthModeNone})
	if err != nil {
		t.Fatal(err)
	}
	p, err := authenticator.Authenticate(&http.Request{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if p != nil {
		t.Error("expected nil principal for none auth")
	}
}

func TestSigV4Static_UnknownAccessKey(t *testing.T) {
	cfg := config.Auth{
		Mode: config.AuthModeSigV4Static,
		Clients: map[string]config.Client{
			"ci": {Name: "ci", AccessKey: "AKIACI", SecretKey: "s"},
		},
	}
	authenticator, err := NewAuthenticator(cfg)
	if err != nil {
		t.Fatal(err)
	}
	r := httptest.NewRequest("GET", "/bucket/key", nil)
	r.Header.Set("Authorization", "AWS4-HMAC-SHA256 Credential=UNKNOWNAK/20240101/us-east-1/s3/aws4_request, ...")
	_, err = authenticator.Authenticate(r)
	if err == nil {
		t.Fatal("expected error for unknown access key")
	}
}

func TestSigV4Static_ValidHeader(t *testing.T) {
	const ak = "AKIATEST"
	const sk = "testsecret123"
	cfg := config.Auth{
		Mode: config.AuthModeSigV4Static,
		Clients: map[string]config.Client{
			"ci": {Name: "ci", AccessKey: ak, SecretKey: sk},
		},
	}
	authenticator, err := NewAuthenticator(cfg)
	if err != nil {
		t.Fatal(err)
	}

	r := httptest.NewRequest("GET", "/bucket/key", nil)
	r.Host = "s3proxy.example.com"
	r.Header.Set("X-Amz-Content-Sha256", "UNSIGNED-PAYLOAD")

	signer := v4signer()
	creds := aws.Credentials{AccessKeyID: ak, SecretAccessKey: sk}
	if err := signer.SignHTTP(context.Background(), creds, r, "UNSIGNED-PAYLOAD", "s3", "us-east-1", time.Now().UTC()); err != nil {
		t.Fatalf("signing failed: %v", err)
	}

	p, err := authenticator.Authenticate(r)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if p.Name != "ci" {
		t.Errorf("expected principal name 'ci', got %q", p.Name)
	}
}

func TestSigV4Static_BadSignature(t *testing.T) {
	const ak = "AKIATEST"
	const sk = "testsecret123"
	cfg := config.Auth{
		Mode: config.AuthModeSigV4Static,
		Clients: map[string]config.Client{
			"ci": {Name: "ci", AccessKey: ak, SecretKey: sk},
		},
	}
	authenticator, err := NewAuthenticator(cfg)
	if err != nil {
		t.Fatal(err)
	}

	r := httptest.NewRequest("GET", "/bucket/key", nil)
	r.Host = "s3proxy.example.com"
	r.Header.Set("X-Amz-Content-Sha256", "UNSIGNED-PAYLOAD")

	signer := v4signer()
	goodCreds := aws.Credentials{AccessKeyID: ak, SecretAccessKey: sk}
	if err := signer.SignHTTP(context.Background(), goodCreds, r, "UNSIGNED-PAYLOAD", "s3", "us-east-1", time.Now().UTC()); err != nil {
		t.Fatalf("signing failed: %v", err)
	}

	originalAuth := r.Header.Get("Authorization")
	tampered := originalAuth[:len(originalAuth)-2] + "XX"
	r.Header.Set("Authorization", tampered)

	_, err = authenticator.Authenticate(r)
	if err == nil {
		t.Fatal("expected error for tampered signature")
	}
}

func TestSigV4Static_ValidHeaderDoesNotReadBody(t *testing.T) {
	const ak = "AKIATEST"
	const sk = "testsecret123"
	cfg := config.Auth{
		Mode: config.AuthModeSigV4Static,
		Clients: map[string]config.Client{
			"ci": {Name: "ci", AccessKey: ak, SecretKey: sk},
		},
	}
	authenticator, err := NewAuthenticator(cfg)
	if err != nil {
		t.Fatal(err)
	}

	r := httptest.NewRequest(http.MethodPut, "/bucket/key", nil)
	r.Host = "s3proxy.example.com"
	r.Header.Set("X-Amz-Content-Sha256", "UNSIGNED-PAYLOAD")
	r.Header.Set("Content-Length", "7")
	r.ContentLength = 7

	signer := v4signer()
	creds := aws.Credentials{AccessKeyID: ak, SecretAccessKey: sk}
	if err := signer.SignHTTP(context.Background(), creds, r, "UNSIGNED-PAYLOAD", "s3", "us-east-1", time.Now().UTC()); err != nil {
		t.Fatalf("signing failed: %v", err)
	}

	r.Body = errorReadCloser{err: errors.New("body should not be read")}
	r.GetBody = nil

	p, err := authenticator.Authenticate(r)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if p == nil || p.Name != "ci" {
		t.Fatalf("unexpected principal: %#v", p)
	}
}

func TestSigV4Static_PresignedQueryExpired(t *testing.T) {
	const ak = "AKIATEST"
	const sk = "testsecret123"
	cfg := config.Auth{
		Mode: config.AuthModeSigV4Static,
		Clients: map[string]config.Client{
			"ci": {Name: "ci", AccessKey: ak, SecretKey: sk},
		},
	}
	authenticator, err := NewAuthenticator(cfg)
	if err != nil {
		t.Fatal(err)
	}

	signedAt := time.Now().UTC().Add(-2 * time.Minute)
	req := mustPresignedRequest(t, ak, sk, signedAt, time.Minute)

	_, err = authenticator.Authenticate(req)
	if err == nil {
		t.Fatal("expected expired presigned request to fail")
	}
}

func TestSigV4Static_PresignedQueryValid(t *testing.T) {
	const ak = "AKIATEST"
	const sk = "testsecret123"
	cfg := config.Auth{
		Mode: config.AuthModeSigV4Static,
		Clients: map[string]config.Client{
			"ci": {Name: "ci", AccessKey: ak, SecretKey: sk},
		},
	}
	authenticator, err := NewAuthenticator(cfg)
	if err != nil {
		t.Fatal(err)
	}

	req := mustPresignedRequest(t, ak, sk, time.Now().UTC(), 10*time.Minute)

	p, err := authenticator.Authenticate(req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if p == nil || p.Name != "ci" {
		t.Fatalf("unexpected principal: %#v", p)
	}
}

func TestSigV4Static_PresignedQueryLongLivedValid(t *testing.T) {
	const ak = "AKIATEST"
	const sk = "testsecret123"
	cfg := config.Auth{
		Mode: config.AuthModeSigV4Static,
		Clients: map[string]config.Client{
			"ci": {Name: "ci", AccessKey: ak, SecretKey: sk},
		},
	}
	authenticator, err := NewAuthenticator(cfg)
	if err != nil {
		t.Fatal(err)
	}

	req := mustPresignedRequest(t, ak, sk, time.Now().UTC().Add(-20*time.Minute), time.Hour)

	p, err := authenticator.Authenticate(req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if p == nil || p.Name != "ci" {
		t.Fatalf("unexpected principal: %#v", p)
	}
}

func mustPresignedRequest(t *testing.T, ak, sk string, signedAt time.Time, expires time.Duration) *http.Request {
	t.Helper()
	req := httptest.NewRequest(http.MethodGet, "https://proxy.example.com/bucket/key", nil)
	query := req.URL.Query()
	query.Set("X-Amz-Expires", strconv.FormatInt(int64(expires/time.Second), 10))
	req.URL.RawQuery = query.Encode()

	signedURI, _, err := v4signer().PresignHTTP(
		context.Background(),
		aws.Credentials{AccessKeyID: ak, SecretAccessKey: sk},
		req,
		"UNSIGNED-PAYLOAD",
		"s3",
		"us-east-1",
		signedAt,
	)
	if err != nil {
		t.Fatalf("presigning failed: %v", err)
	}
	parsed, err := url.Parse(signedURI)
	if err != nil {
		t.Fatalf("parse signed URI: %v", err)
	}
	req.URL = parsed
	req.Host = parsed.Host
	return req
}

func TestAuthorizer_AllowRoute(t *testing.T) {
	az := NewAuthorizer(config.Auth{})
	p := &Principal{
		Name:        "ci",
		AllowRoutes: []string{"route.images"},
	}
	if !az.AllowRoute(p, "route.images", s3ops.OpGetObject) {
		t.Error("expected route to be allowed")
	}
	if az.AllowRoute(p, "route.logs", s3ops.OpGetObject) {
		t.Error("expected route to be denied")
	}
}

type errorReadCloser struct{ err error }

func (e errorReadCloser) Read([]byte) (int, error) { return 0, e.err }

func (errorReadCloser) Close() error { return nil }

func TestAuthorizer_WildcardRoute(t *testing.T) {
	az := NewAuthorizer(config.Auth{})
	p := &Principal{
		Name:        "admin",
		AllowRoutes: []string{"*"},
	}
	if !az.AllowRoute(p, "route.anything", s3ops.OpGetObject) {
		t.Error("expected wildcard route to be allowed")
	}
}

func TestAuthorizer_NilPrincipal(t *testing.T) {
	az := NewAuthorizer(config.Auth{})
	if !az.AllowRoute(nil, "route.anything", s3ops.OpGetObject) {
		t.Error("expected nil principal to be allowed (none mode)")
	}
}

func TestAuthorizer_BucketVisibility(t *testing.T) {
	az := NewAuthorizer(config.Auth{})
	p := &Principal{
		Name:           "ci",
		VisibleBuckets: []string{"images", "logs"},
	}
	if !az.AllowBucketVisibility(p, "images") {
		t.Error("expected bucket 'images' to be visible")
	}
	if az.AllowBucketVisibility(p, "secret") {
		t.Error("expected bucket 'secret' to be denied")
	}
}

func TestAuthorizer_WildcardBucket(t *testing.T) {
	az := NewAuthorizer(config.Auth{})
	p := &Principal{
		Name:           "admin",
		VisibleBuckets: []string{"*"},
	}
	if !az.AllowBucketVisibility(p, "anything") {
		t.Error("expected wildcard bucket visibility")
	}
}
