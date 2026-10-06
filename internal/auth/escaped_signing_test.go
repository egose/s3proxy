package auth

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/egose/s3proxy/internal/config"
	"github.com/egose/s3proxy/internal/replaybody"
)

func TestSigV4Static_S3EscapedKeys(t *testing.T) {
	const ak, sk = "escaped-client", "escaped-secret"
	now := time.Date(2026, 9, 26, 12, 0, 0, 0, time.UTC)
	a, err := newTestAuthenticator(config.Auth{
		Mode:    config.AuthModeSigV4Static,
		Clients: map[string]config.Client{"client": {Name: "client", AccessKey: ak, SecretKey: sk}},
	})
	if err != nil {
		t.Fatal(err)
	}
	setAuthenticatorNow(t, a, now)
	for _, path := range []string{
		"/bucket/space%20key", "/bucket/percent%25key", "/bucket/%E9%9B%AA.txt",
		"/bucket/escaped%2Fseparator", "/bucket/lower%2fseparator", "/bucket/literal%252Fseparator",
		"/bucket//a/../b//key%20name",
	} {
		for _, presigned := range []bool{false, true} {
			mode := "header"
			if presigned {
				mode = "presigned"
			}
			t.Run(mode+path, func(t *testing.T) {
				for _, mutation := range []string{"none", "path", "signature", "signed header", "body", "separator decoded"} {
					if mutation == "separator decoded" && !strings.Contains(strings.ToLower(path), "%2f") {
						continue
					}
					t.Run(mutation, func(t *testing.T) {
						const body = "payload"
						const hash = "239f59ed55e737c77147cf55ad0c1b030b6d7ee748a7426952f9b852d5a935e5" // pragma: allowlist secret
						r := httptest.NewRequest(http.MethodPut, "https://proxy.example.com"+path, strings.NewReader(body))
						defer replaybody.Release(r)
						r.Header.Set("Content-Length", "7")
						r.Header.Set("X-Amz-Content-Sha256", hash)
						r.Header.Set("X-Amz-Meta-Owner", "alice")
						creds := aws.Credentials{AccessKeyID: ak, SecretAccessKey: sk}
						if presigned {
							r.URL.RawQuery = "X-Amz-Expires=600"
							uri, _, err := v4signer().PresignHTTP(context.Background(), creds, r, hash, "s3", "us-east-1", now)
							if err != nil {
								t.Fatal(err)
							}
							r.URL, err = url.Parse(uri)
							if err != nil {
								t.Fatal(err)
							}
						} else if err := v4signer().SignHTTP(context.Background(), creds, r, hash, "s3", "us-east-1", now); err != nil {
							t.Fatal(err)
						}
						r.Header.Set("Accept-Encoding", "gzip")
						r.Header.Set("X-Amz-Acl", "public-read")
						wantErr := errSignatureMismatch
						switch mutation {
						case "separator decoded":
							r.URL.RawPath = strings.ReplaceAll(strings.ReplaceAll(path, "%2F", "/"), "%2f", "/")
						case "path":
							r.URL.Path += "changed"
							r.URL.RawPath = path + "changed"
						case "signature":
							if presigned {
								q := r.URL.Query()
								q.Set("X-Amz-Signature", strings.Repeat("0", 64))
								r.URL.RawQuery = q.Encode()
							} else {
								r.Header.Set("Authorization", r.Header.Get("Authorization")+"0")
							}
						case "signed header":
							r.Header.Set("X-Amz-Meta-Owner", "mallory")
						case "body":
							r.Body = io.NopCloser(strings.NewReader("changed"))
							wantErr = errPayloadHashMismatch
						}
						p, err := a.Authenticate(r)
						if mutation != "none" {
							if !errors.Is(err, wantErr) {
								t.Fatalf("Authenticate = %v, want %v", err, wantErr)
							}
							return
						}
						if err != nil || p == nil || p.Name != "client" {
							t.Fatalf("Authenticate = %v, %v", p, err)
						}
						if r.URL.EscapedPath() != path || r.Header.Get("X-Amz-Meta-Owner") != "alice" || r.Header.Get("X-Amz-Acl") != "" {
							t.Fatalf("path or authenticated headers changed: %s %v", r.URL.EscapedPath(), r.Header)
						}
					})
				}
			})
		}
	}
}
