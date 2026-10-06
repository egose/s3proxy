package httpapi

import (
	"context"
	"crypto/sha256"
	"encoding/xml"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	v4 "github.com/aws/aws-sdk-go-v2/aws/signer/v4"
	"github.com/egose/s3proxy/internal/auth"
	"github.com/egose/s3proxy/internal/config"
	"github.com/egose/s3proxy/internal/testutil/s3signature"
)

type queryReadCounter struct {
	io.ReadCloser
	reads *atomic.Int64
}

func (b queryReadCounter) Read(p []byte) (int, error) {
	b.reads.Add(1)
	return b.ReadCloser.Read(p)
}

func signQueryRequest(t *testing.T, r *http.Request, mode, payload string) {
	t.Helper()
	if mode == "none" {
		return
	}
	hash := fmt.Sprintf("%x", sha256.Sum256([]byte(payload)))
	r.Header.Set("X-Amz-Content-Sha256", hash)
	signer := v4.NewSigner(func(o *v4.SignerOptions) { o.DisableURIPathEscaping = true })
	creds := aws.Credentials{AccessKeyID: "allowed-ak", SecretAccessKey: "allowed-sk"}
	if mode == "header" {
		if err := signer.SignHTTP(context.Background(), creds, r, hash, "s3", "us-east-1", time.Now().UTC()); err != nil {
			t.Fatal(err)
		}
		return
	}
	r.URL.RawQuery += "&X-Amz-Expires=600"
	uri, _, err := signer.PresignHTTP(context.Background(), creds, r, hash, "s3", "us-east-1", time.Now().UTC())
	if err != nil {
		t.Fatal(err)
	}
	r.URL, err = url.Parse(uri)
	if err != nil {
		t.Fatal(err)
	}
}

func TestHandler_QueryBoundary(t *testing.T) {
	for _, mode := range []string{"none", "header", "presign"} {
		for _, tc := range []struct{ name, method, path, valid, bad string }{
			{"delete version", "DELETE", "/visible/key", "", "versionId=QUERY_SECRET%GG"},
			{"delete tagging", "DELETE", "/visible/key", "", "tagging=;QUERY_SECRET"},
			{"name invalid", "PUT", "/visible/key", "", "QUERY_SECRET%GG=value"},
			{"name truncated", "PUT", "/visible/key", "", "QUERY_SECRET%=value"},
			{"value truncated", "PUT", "/visible/key", "", "versionId=QUERY_SECRET%2"},
			{"semicolon name", "DELETE", "/visible/key", "", "tagging;QUERY_SECRET=value"}, // pragma: allowlist secret
			{"duplicate operation", "DELETE", "/visible/key", "x-id=DeleteObject", "x-id=QUERY_SECRET%GG"},
			{"listing prefix", "GET", "/visible", "list-type=2", "prefix=QUERY_SECRET%GG"},
			{"duplicate prefix", "GET", "/visible", "list-type=2&prefix=keep", "prefix=QUERY_SECRET%GG"},
			{"duplicate list type", "GET", "/visible", "list-type=2", "list-type=QUERY_SECRET%GG"},
			{"duplicate signature", "PUT", "/visible/key", "", "X-Amz-Signature=QUERY_SECRET%GG"},
			{"health probe", "GET", "/healthz", "", "QUERY_SECRET=%GG"},
			{"ready probe", "GET", "/readyz", "", "QUERY_SECRET=;bad"},
		} {
			t.Run(mode+"/"+tc.name, func(t *testing.T) {
				f := newProbeFixture(t, config.Addressing{PathStyle: true})
				if mode == "none" {
					f.auth.Authenticator, _ = auth.NewAuthenticator(config.Auth{Mode: config.AuthModeNone}, nil)
				}
				f.mu.Lock()
				f.objects["/visible/key"] = "original"
				f.mu.Unlock()
				var reads atomic.Int64
				h := f.proxy.Config.Handler
				f.proxy.Config.Handler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					r.Body = queryReadCounter{r.Body, &reads}
					h.ServeHTTP(w, r)
				})
				const payload = "replacement"
				r, err := http.NewRequest(tc.method, f.proxy.URL+tc.path+"?"+tc.valid, strings.NewReader(payload))
				if err != nil {
					t.Fatal(err)
				}
				signQueryRequest(t, r, mode, payload)
				r.URL.RawQuery += "&" + tc.bad
				resp, err := f.proxy.Client().Do(r)
				if err != nil {
					t.Fatal(err)
				}
				data, err := io.ReadAll(resp.Body)
				resp.Body.Close()
				if err != nil {
					t.Fatal(err)
				}
				if resp.StatusCode != 400 || !strings.Contains(string(data), "<Code>InvalidRequest</Code>") {
					t.Errorf("response = %d %s, want InvalidRequest 400", resp.StatusCode, data)
				}
				if reads.Load() != 0 || f.calls.Load() != 0 || f.auth.calls.Load() != 0 || f.router.calls.Load() != 0 {
					t.Errorf("reads/backend/auth/router = %d/%d/%d/%d, want all zero", reads.Load(), f.calls.Load(), f.auth.calls.Load(), f.router.calls.Load())
				}
				if value, ok := f.object("/visible/key"); !ok || value != "original" {
					t.Errorf("stored object = %q, %v; want original", value, ok)
				}
				f.mu.Lock()
				logs := f.logs.String()
				f.mu.Unlock()
				for _, secret := range []string{"QUERY_SECRET", "%GG", "allowed-ak", "X-Amz-Signature="} { // pragma: allowlist secret
					if strings.Contains(logs+string(data), secret) {
						t.Errorf("diagnostics exposed %q", secret)
					}
				}
			})
		}
	}
}

func TestHandler_QueryBoundaryValidListing(t *testing.T) {
	const prefix = "a;b%+ space 雪/"
	const token = "opaque/%2F+&=;雪"
	var calls atomic.Int64
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		q := r.URL.Query()
		if q.Get("prefix") != "assets/"+prefix || q.Get("continuation-token") != token || len(q["max-keys"]) != 2 {
			t.Errorf("outbound query = %v", q)
		}
		for k := range q {
			if strings.HasPrefix(strings.ToLower(k), "x-amz-") {
				t.Errorf("auth query forwarded: %s", k)
			}
		}
		if r.Header.Get("Authorization") != s3signature.Authorization(r, "backend-ak", "backend-sk", "us-east-1", "host;x-amz-content-sha256;x-amz-date") {
			t.Error("outbound signature invalid")
		}
		if err := xml.NewEncoder(w).Encode(workflowList{Name: "store", Prefix: "assets/" + prefix, ContinuationToken: token, Contents: []workflowObject{{Key: "assets/" + prefix + "key", Size: 7}}}); err != nil {
			t.Error(err)
		}
	}))
	defer upstream.Close()
	proxy := listingProxy(t, upstream.URL, func(string) config.RewriteRule {
		return config.RewriteRule{Bucket: "store", PrependKeyPrefix: "assets/"}
	})
	for _, presign := range []bool{false, true} {
		path := "/tenant-acme-logs?list-type=2&%70refix=a%3bb%25%2b+space%20%E9%9B%AA%2f&continuation-token=" + url.QueryEscape(token) + "&max-keys=2&max-keys=2"
		resp, data := signedListingRequest(t, proxy, "acme", "GET", path, "", presign)
		var listing workflowList
		if err := xml.Unmarshal(data, &listing); err != nil {
			t.Fatal(err)
		}
		if resp.StatusCode != 200 || listing.Prefix != prefix || listing.ContinuationToken != token || len(listing.Contents) != 1 || listing.Contents[0].Key != prefix+"key" {
			t.Fatalf("listing status=%d body=%s", resp.StatusCode, data)
		}
	}
	if calls.Load() != 2 {
		t.Fatalf("backend calls=%d", calls.Load())
	}
}

func TestHandler_QueryBoundaryWellFormedUnsupported(t *testing.T) {
	for _, mode := range []string{"none", "header", "presign"} {
		for _, query := range []string{"versionId=value", "tagging=%3Bvalue", "versionId=%25", "x-id=DeleteObject&x-id=DeleteObject"} {
			t.Run(mode+"/"+query, func(t *testing.T) {
				f := newProbeFixture(t, config.Addressing{PathStyle: true})
				if mode == "none" {
					f.auth.Authenticator, _ = auth.NewAuthenticator(config.Auth{Mode: config.AuthModeNone}, nil)
				}
				r, err := http.NewRequest("DELETE", f.proxy.URL+"/visible/key?"+query, nil)
				if err != nil {
					t.Fatal(err)
				}
				signQueryRequest(t, r, mode, "")
				resp, err := f.proxy.Client().Do(r)
				if err != nil {
					t.Fatal(err)
				}
				data, err := io.ReadAll(resp.Body)
				resp.Body.Close()
				if err != nil {
					t.Fatal(err)
				}
				if resp.StatusCode != 501 || !strings.Contains(string(data), "<Code>NotImplemented</Code>") || f.calls.Load() != 0 {
					t.Fatalf("status=%d body=%s backend=%d", resp.StatusCode, data, f.calls.Load())
				}
			})
		}
	}
}
