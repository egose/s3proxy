package s3

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/egose/s3proxy/internal/config"
	"github.com/egose/s3proxy/internal/s3ops"
	"github.com/egose/s3proxy/internal/testutil/s3signature"
)

func TestClient_S3EscapedKeySignatures(t *testing.T) {
	for _, key := range []string{"space%20key", "percent%25key", "%E9%9B%AA.txt", "escaped%2Fseparator", "lower%2fseparator", "literal%252Fseparator", "/a/../b//key%20name"} {
		for _, method := range []string{http.MethodGet, http.MethodPut} {
			t.Run(method+"/"+key, func(t *testing.T) {
				calls := 0
				server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					calls++
					if want := "/base%20path/bucket/" + key; r.URL.EscapedPath() != want {
						t.Errorf("escaped path = %q, want %q", r.URL.EscapedPath(), want)
					}
					headers := "host;x-amz-content-sha256;x-amz-date"
					if method == http.MethodPut {
						headers = "content-length;" + headers
						if r.Header.Get("Content-Length") != "7" {
							t.Errorf("Content-Length = %q", r.Header.Get("Content-Length"))
						}
					}
					if r.Header.Get("X-Amz-Content-Sha256") != "UNSIGNED-PAYLOAD" {
						t.Error("outbound payload hash changed")
					}
					want := s3signature.Authorization(r, "ak", "sk", "us-east-1", headers)
					if got := r.Header.Get("Authorization"); got != want {
						t.Errorf("Authorization = %s, want independent S3 signature %s", got, want)
						w.WriteHeader(http.StatusForbidden)
						return
					}
					_, _ = io.Copy(io.Discard, r.Body)
				}))
				defer server.Close()
				client := newTestClient(t, server.Client(), testTargets(t, server.URL+"/base%20path"), testBudget())
				var body io.Reader
				op := s3ops.OpGetObject
				if method == http.MethodPut {
					body = strings.NewReader("payload")
					op = s3ops.OpPutObject
				}
				source := httptest.NewRequest(method, "http://proxy.local/bucket/"+key, body)
				resp, err := client.Do(context.Background(), Request{Operation: op, Target: "primary", Bucket: "bucket", Key: key, Source: source})
				if err != nil {
					t.Fatal(err)
				}
				resp.Body.Close()
				if resp.StatusCode != http.StatusOK || calls != 1 {
					t.Fatalf("status = %d, calls = %d", resp.StatusCode, calls)
				}
			})
		}
	}
}

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
