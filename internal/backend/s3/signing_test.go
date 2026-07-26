package s3

import (
	"net/http"
	"net/url"
	"strings"
	"testing"

	"github.com/jahn/s3proxy/internal/config"
)

func TestSignRequest_AddsAuthorization(t *testing.T) {
	req := &http.Request{
		Method: "GET",
		URL:    &url.URL{Scheme: "https", Host: "minio.internal", Path: "/bucket/key"},
		Header: http.Header{},
		Body:   http.NoBody,
	}
	target := config.S3Target{
		Endpoint: "https://minio.internal",
		Region:   "us-east-1",
		Credentials: config.StaticCredential{
			AccessKey: "AKIATEST",
			SecretKey: "secrettest",
		},
	}

	if err := signRequest(req, target); err != nil {
		t.Fatalf("signRequest failed: %v", err)
	}

	auth := req.Header.Get("Authorization")
	if auth == "" {
		t.Fatal("expected Authorization header to be set")
	}
	if !strings.HasPrefix(auth, "AWS4-HMAC-SHA256 Credential=AKIATEST/") {
		t.Errorf("unexpected Authorization header: %s", auth)
	}
	if req.Header.Get("X-Amz-Date") == "" {
		t.Error("expected X-Amz-Date header to be set")
	}
	if req.Header.Get("X-Amz-Content-Sha256") == "" {
		t.Error("expected X-Amz-Content-Sha256 header to be set")
	}
}

func TestSignRequest_DifferentCredentials(t *testing.T) {
	req1 := &http.Request{
		Method: "GET",
		URL:    &url.URL{Scheme: "https", Host: "minio.internal", Path: "/b/k"},
		Header: http.Header{},
		Body:   http.NoBody,
	}
	targetA := config.S3Target{
		Endpoint: "https://minio.internal",
		Region:   "us-east-1",
		Credentials: config.StaticCredential{
			AccessKey: "AKIATEST",
			SecretKey: "secrettest",
		},
	}
	if err := signRequest(req1, targetA); err != nil {
		t.Fatal(err)
	}
	authA := req1.Header.Get("Authorization")

	req2 := &http.Request{
		Method: "GET",
		URL:    &url.URL{Scheme: "https", Host: "minio.internal", Path: "/b/k"},
		Header: http.Header{},
		Body:   http.NoBody,
	}
	targetB := config.S3Target{
		Endpoint: "https://minio.internal",
		Region:   "us-east-1",
		Credentials: config.StaticCredential{
			AccessKey: "AKIADIFFERENT",
			SecretKey: "differentsecret",
		},
	}
	if err := signRequest(req2, targetB); err != nil {
		t.Fatal(err)
	}
	authB := req2.Header.Get("Authorization")

	if authA == authB {
		t.Error("expected different signatures for different credentials")
	}
}
