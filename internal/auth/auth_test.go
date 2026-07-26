package auth

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	v4 "github.com/aws/aws-sdk-go-v2/aws/signer/v4"
	"github.com/jahn/s3proxy/internal/config"
	"github.com/jahn/s3proxy/internal/s3ops"
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
