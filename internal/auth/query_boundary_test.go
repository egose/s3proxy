package auth

import (
	"context"
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

type queryBody struct{ reads int }

func (b *queryBody) Read(p []byte) (int, error) { b.reads++; return 0, io.EOF }
func (*queryBody) Close() error                 { return nil }

func TestSigV4_QueryBoundary(t *testing.T) {
	const hash = "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855" // pragma: allowlist secret
	v := newSigV4Verifier(map[string]config.Client{"ak": {AccessKey: "ak", SecretKey: "sk"}}, "us-east-1", replaybody.NewBudget(1024, 2048))
	for _, mode := range []string{"header", "presign"} {
		for _, bad := range []string{"versionId=QUERY_SECRET%GG", "QUERY_SECRET%=v", "tagging=;QUERY_SECRET", "x-id=QUERY_SECRET%2", "X-Amz-Signature=QUERY_SECRET%GG"} {
			t.Run(mode+"/"+bad, func(t *testing.T) {
				r := httptest.NewRequest(http.MethodPut, "https://proxy.local/bucket/key?x-id=PutObject", nil)
				r.Header.Set("X-Amz-Content-Sha256", hash)
				creds := aws.Credentials{AccessKeyID: "ak", SecretAccessKey: "sk"}
				if mode == "header" {
					if err := v4signer().SignHTTP(context.Background(), creds, r, hash, "s3", "us-east-1", time.Now().UTC()); err != nil {
						t.Fatal(err)
					}
				} else {
					r.URL.RawQuery += "&X-Amz-Expires=600"
					uri, _, err := v4signer().PresignHTTP(context.Background(), creds, r, hash, "s3", "us-east-1", time.Now().UTC())
					if err != nil {
						t.Fatal(err)
					}
					r.URL, err = url.Parse(uri)
					if err != nil {
						t.Fatal(err)
					}
				}
				r.URL.RawQuery += "&" + bad
				raw := r.URL.RawQuery
				body := &queryBody{}
				r.Body, r.ContentLength = body, -1
				defer replaybody.Release(r)
				p, err := v.Verify(r)
				if p != nil || err == nil || err.Error() != "malformed request query" {
					t.Errorf("Verify = %v, %v; want safe malformed query error", p, err)
				}
				if body.reads != 0 || r.GetBody != nil || r.URL.RawQuery != raw {
					t.Errorf("reads=%d replay=%v raw changed=%v", body.reads, r.GetBody != nil, r.URL.RawQuery != raw)
				}
				if err != nil && strings.Contains(err.Error(), "QUERY_SECRET") {
					t.Error("query leaked")
				}
			})
		}
	}
}

func TestSigV4_QueryBoundaryValidRawBytes(t *testing.T) {
	v := newSigV4Verifier(map[string]config.Client{"ak": {AccessKey: "ak", SecretKey: "sk"}}, "us-east-1", replaybody.NewBudget(1024, 2048))
	for _, mode := range []string{"header", "presign"} {
		r := httptest.NewRequest("GET", "https://proxy.local/bucket?list-type=2&prefix=a%3Bb%25%2B+space%20%E9%9B%AA&continuation-token=opaque%2F%252F%2B%26%3D%3B&custom=one&custom=two", nil)
		creds := aws.Credentials{AccessKeyID: "ak", SecretAccessKey: "sk"}
		if mode == "header" {
			if err := v4signer().SignHTTP(context.Background(), creds, r, "UNSIGNED-PAYLOAD", "s3", "us-east-1", time.Now().UTC()); err != nil {
				t.Fatal(err)
			}
		} else {
			r.URL.RawQuery += "&X-Amz-Expires=600"
			uri, _, err := v4signer().PresignHTTP(context.Background(), creds, r, "UNSIGNED-PAYLOAD", "s3", "us-east-1", time.Now().UTC())
			if err != nil {
				t.Fatal(err)
			}
			r.URL, err = url.Parse(uri)
			if err != nil {
				t.Fatal(err)
			}
		}
		r.URL.RawQuery = strings.ReplaceAll(strings.ReplaceAll(r.URL.RawQuery, "%3B", "%3b"), "%20", "+")
		r.URL.RawQuery = strings.ReplaceAll(r.URL.RawQuery, "prefix=", "%70refix=")
		raw := r.URL.RawQuery
		if p, err := v.Verify(r); p == nil || err != nil {
			t.Fatalf("%s Verify = %v, %v", mode, p, err)
		}
		if r.URL.RawQuery != raw {
			t.Fatal("valid raw query bytes changed")
		}
	}
}

func TestSigV4_QueryBoundaryBeforeAuthAndReplay(t *testing.T) {
	for _, header := range []string{"", "invalid Authorization"} {
		v := &sigV4Verifier{}
		r := httptest.NewRequest("PUT", "https://proxy.local/bucket/key?SECRET=%GG", nil)
		r.Header.Set("Authorization", header)
		body := &queryBody{}
		r.Body = body
		r.GetBody = func() (io.ReadCloser, error) { t.Error("replay called"); return body, nil }
		if p, err := v.Verify(r); p != nil || err == nil || err.Error() != "malformed request query" {
			t.Fatalf("Verify = %v, %v", p, err)
		}
		if body.reads != 0 {
			t.Fatal("body read")
		}
	}
}
