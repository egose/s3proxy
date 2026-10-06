package httpapi

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	v4 "github.com/aws/aws-sdk-go-v2/aws/signer/v4"
	"github.com/egose/s3proxy/internal/auth"
	"github.com/egose/s3proxy/internal/backend/s3"
	"github.com/egose/s3proxy/internal/config"
	"github.com/egose/s3proxy/internal/dispatch"
	"github.com/egose/s3proxy/internal/replaybody"
	"github.com/egose/s3proxy/internal/rewrite"
	"github.com/egose/s3proxy/internal/router"
	"github.com/egose/s3proxy/internal/testutil/s3signature"
)

func TestHandler_AuthenticatedS3EscapedKeyRoundTrip(t *testing.T) {
	var calls atomic.Int64
	var mu sync.Mutex
	objects := map[string]string{}
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		headers := "host;x-amz-content-sha256;x-amz-date"
		if r.Method == http.MethodPut {
			headers = "content-length;" + headers + ";x-amz-meta-owner"
			if r.Header.Get("Content-Length") == "" || r.Header.Get("X-Amz-Meta-Owner") != "alice" {
				t.Error("missing signed upload length or metadata")
			}
		}
		if r.Header.Get("X-Amz-Acl") != "" || r.URL.RawQuery != "" {
			t.Error("unsigned control headers or client presign parameters reached upstream")
		}
		if r.Header.Get("X-Amz-Content-Sha256") != "UNSIGNED-PAYLOAD" {
			t.Error("outbound payload hash changed")
		}
		want := s3signature.Authorization(r, "backend-ak", "backend-sk", "us-east-1", headers)
		if r.Header.Get("Authorization") != want {
			t.Error("upstream signature differs from independent S3 HMAC calculation")
			http.Error(w, "SignatureDoesNotMatch", http.StatusForbidden)
			return
		}
		key := r.URL.EscapedPath()
		mu.Lock()
		defer mu.Unlock()
		switch r.Method {
		case http.MethodPut:
			data, err := io.ReadAll(r.Body)
			if err != nil {
				t.Error(err)
				w.WriteHeader(http.StatusBadRequest)
				return
			}
			objects[key] = string(data)
		case http.MethodGet:
			data, ok := objects[key]
			if !ok {
				w.WriteHeader(http.StatusNotFound)
				return
			}
			_, _ = io.WriteString(w, data)
		}
	}))
	defer upstream.Close()
	endpoint, err := url.Parse(upstream.URL + "/base%20path")
	if err != nil {
		t.Fatal(err)
	}
	budget := replaybody.NewBudget(replaybody.DefaultMaxBytes, replaybody.DefaultAggregateMaxBytes)
	backend, err := s3.NewClient(upstream.Client(), map[string]config.S3Target{"store": {
		EndpointURL: endpoint, Region: "us-east-1", ForcePathStyle: true,
		Credentials: config.StaticCredential{AccessKey: "backend-ak", SecretKey: "backend-sk"},
	}}, budget)
	if err != nil {
		t.Fatal(err)
	}
	dispatcher, err := dispatch.New(backend, budget)
	if err != nil {
		t.Fatal(err)
	}
	authenticator, err := auth.NewAuthenticator(config.Auth{
		Mode: config.AuthModeSigV4Static,
		Clients: map[string]config.Client{"client": {
			Name: "client", AccessKey: "client-ak", SecretKey: "client-sk",
			AllowRoutes: []string{"objects"}, AllowOps: []string{"PutObject", "GetObject"},
		}},
	}, budget)
	if err != nil {
		t.Fatal(err)
	}
	resolver := router.NewResolver([]config.Route{{
		Name: "objects", ParserRef: "visible", DestinationRefs: []string{"store"},
		Operations: []string{"PutObject", "GetObject"},
		Dispatch:   config.DispatchFirst, OnMatch: config.MatchStop,
		Rewrite: config.RewriteRule{Bucket: "physical", PrependKeyPrefix: "assets/"},
	}}, map[string]config.Parser{"visible": {Kind: config.ParserBucketExact, Bucket: "visible"}}, []string{"store"})
	proxy := httptest.NewServer(NewHandler(Dependencies{
		Addressing: config.Addressing{PathStyle: true}, ReplayBudget: budget,
		Authenticator: authenticator, Authorizer: auth.NewAuthorizer(), Router: resolver,
		Rewriter: rewrite.New(), Dispatcher: dispatcher, Logger: slog.New(slog.NewTextHandler(io.Discard, nil)),
	}))
	defer proxy.Close()
	signer := v4.NewSigner(func(o *v4.SignerOptions) { o.DisableURIPathEscaping = true })
	creds := aws.Credentials{AccessKeyID: "client-ak", SecretAccessKey: "client-sk"}
	for _, key := range []string{"space%20key", "percent%25key", "%E9%9B%AA.txt", "escaped%2Fseparator", "lower%2fseparator", "literal%252Fseparator", "/a/../b//key%20name"} {
		t.Run(key, func(t *testing.T) {
			payload := "object bytes: " + key
			for _, method := range []string{http.MethodPut, http.MethodGet} {
				for _, mutation := range []string{"none", "path", "signature"} {
					var body io.Reader
					hash := "UNSIGNED-PAYLOAD"
					if method == http.MethodPut {
						body = strings.NewReader(payload)
						sum := sha256.Sum256([]byte(payload))
						hash = hex.EncodeToString(sum[:])
					}
					r, err := http.NewRequest(method, proxy.URL+"/visible/"+key, body)
					if err != nil {
						t.Fatal(err)
					}
					if method == http.MethodPut {
						r.Header.Set("Content-Length", strconv.Itoa(len(payload)))
						r.Header.Set("X-Amz-Content-Sha256", hash)
						r.Header.Set("X-Amz-Meta-Owner", "alice")
						if err := signer.SignHTTP(context.Background(), creds, r, hash, "s3", "us-east-1", time.Now().UTC()); err != nil {
							t.Fatal(err)
						}
					} else {
						r.URL.RawQuery = "X-Amz-Expires=600"
						uri, _, err := signer.PresignHTTP(context.Background(), creds, r, hash, "s3", "us-east-1", time.Now().UTC())
						if err != nil {
							t.Fatal(err)
						}
						r.URL, err = url.Parse(uri)
						if err != nil {
							t.Fatal(err)
						}
					}
					r.Header.Set("X-Amz-Acl", "public-read")
					switch mutation {
					case "path":
						r.URL.Path += "changed"
						r.URL.RawPath = "/visible/" + key + "changed"
					case "signature":
						if method == http.MethodPut {
							r.Header.Set("Authorization", r.Header.Get("Authorization")+"0")
						} else {
							q := r.URL.Query()
							q.Set("X-Amz-Signature", strings.Repeat("0", 64))
							r.URL.RawQuery = q.Encode()
						}
					}
					before := calls.Load()
					resp, err := proxy.Client().Do(r)
					if err != nil {
						t.Fatal(err)
					}
					data, err := io.ReadAll(resp.Body)
					resp.Body.Close()
					if err != nil {
						t.Fatal(err)
					}
					if mutation != "none" {
						if resp.StatusCode != http.StatusForbidden || !strings.Contains(string(data), "<Code>SignatureDoesNotMatch</Code>") || calls.Load() != before {
							t.Fatalf("%s/%s: status = %d, body = %s, upstream calls = %d", method, mutation, resp.StatusCode, data, calls.Load()-before)
						}
						continue
					}
					if resp.StatusCode != http.StatusOK || calls.Load() != before+1 {
						t.Fatalf("%s: status = %d, body = %s, upstream calls = %d", method, resp.StatusCode, data, calls.Load()-before)
					}
					if method == http.MethodGet && string(data) != payload {
						t.Fatalf("object = %q, want %q", data, payload)
					}
				}
			}
			mu.Lock()
			got, ok := objects["/base%20path/physical/assets/"+key]
			mu.Unlock()
			if !ok || got != payload {
				t.Fatal("rewritten upstream key or object bytes changed")
			}
		})
	}
	if budget.Used() != 0 {
		t.Fatalf("replay budget leaked %d bytes", budget.Used())
	}
}
