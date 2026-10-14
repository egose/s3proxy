package s3

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"reflect"
	"strings"
	"testing"

	"github.com/egose/s3proxy/internal/config"
	"github.com/egose/s3proxy/internal/s3ops"
	"github.com/egose/s3proxy/internal/testutil/s3signature"
)

func TestClientDo_BackendSessionTokenIsolationAndSnapshot(t *testing.T) {
	credentials := map[string]config.StaticCredential{
		"primary": {AccessKey: "CRED_PRIVATE_primary-ak", SecretKey: "CRED_PRIVATE_primary-sk", SessionToken: "CRED_PRIVATE_primary-token/+=value"},
		"replica": {AccessKey: "CRED_PRIVATE_replica-ak", SecretKey: "CRED_PRIVATE_replica-sk", SessionToken: "CRED_PRIVATE_replica-token/+=value"},
		"omitted": {AccessKey: "CRED_PRIVATE_omitted-ak", SecretKey: "CRED_PRIVATE_omitted-sk"},
		"empty":   {AccessKey: "CRED_PRIVATE_empty-ak", SecretKey: "CRED_PRIVATE_empty-sk", SessionToken: ""},
	}
	type capture struct {
		request *http.Request
		body    []byte
	}
	captures := make(chan capture, 1)
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, err := io.ReadAll(r.Body)
		if err != nil {
			t.Error("could not read outbound body")
		}
		captures <- capture{request: r.Clone(context.Background()), body: body}
	}))
	defer upstream.Close()
	targets := make(map[string]config.S3Target)
	for name, credential := range credentials {
		targets[name] = config.S3Target{
			Name: name, Endpoint: upstream.URL + "/base%2Fpath", EndpointURL: mustURL(t, upstream.URL+"/base%2Fpath"),
			Region: "us-east-1", ForcePathStyle: true, Credentials: credential,
		}
	}
	budget := testBudget()
	executor := newTestClient(t, upstream.Client(), targets, budget)
	for name, target := range targets {
		target.Credentials = config.StaticCredential{AccessKey: "CRED_PRIVATE_replaced-ak", SecretKey: "CRED_PRIVATE_replaced-sk", SessionToken: "CRED_PRIVATE_replaced-token"}
		target.EndpointURL.Path = "/changed"
		target.EndpointURL.RawPath = ""
		targets[name] = target
	}
	clear(targets)
	for _, tc := range []struct {
		name    string
		headers http.Header
		query   url.Values
	}{
		{name: "absent"},
		{name: "header", headers: http.Header{"X-Amz-Security-Token": {"CRED_PRIVATE_inbound-header"}}},
		{name: "noncanonical duplicate headers", headers: http.Header{
			"x-amz-security-token": {"", "CRED_PRIVATE_inbound-header"},
			"X-AmZ-SeCuRiTy-ToKeN": {"CRED_PRIVATE_inbound-other-header"},
		}},
		{name: "presign query", query: url.Values{
			"X-Amz-Algorithm": {"AWS4-HMAC-SHA256"}, "X-Amz-Credential": {"CRED_PRIVATE_inbound-credential"},
			"X-Amz-Date": {"20261012T120000Z"}, "X-Amz-Expires": {"600"},
			"X-Amz-Security-Token": {"CRED_PRIVATE_inbound-query"}, "X-Amz-SignedHeaders": {"host"}, "X-Amz-Signature": {"CRED_PRIVATE_inbound-signature"},
		}},
		{name: "noncanonical duplicate query", query: url.Values{
			"x-amz-security-token": {"", "CRED_PRIVATE_inbound-query"}, "X-AmZ-SeCuRiTy-ToKeN": {"CRED_PRIVATE_inbound-other-query"},
		}},
		{name: "header and query", headers: http.Header{"X-Amz-Security-Token": {"", "CRED_PRIVATE_inbound-header"}}, query: url.Values{
			"X-Amz-Security-Token": {"CRED_PRIVATE_inbound-query"},
		}},
	} {
		for _, method := range []string{http.MethodGet, http.MethodPut} {
			t.Run(tc.name+"/"+method, func(t *testing.T) {
				const key = "lower%2fseparator//a/../space%20key%25"
				var body io.Reader
				op := s3ops.OpGetObject
				if method == http.MethodPut {
					body = strings.NewReader("payload")
					op = s3ops.OpPutObject
				}
				source, err := http.NewRequest(method, "http://proxy.example.test/visible/"+key, body)
				if err != nil {
					t.Fatal("could not build source request")
				}
				if body != nil {
					defer source.Body.Close()
				}
				if tc.headers != nil {
					source.Header = tc.headers.Clone()
				}
				source.URL.RawQuery = tc.query.Encode()
				originalHeaders := source.Header.Clone()
				originalQuery := source.URL.RawQuery
				for _, name := range []string{"primary", "omitted", "replica", "empty", "primary"} {
					resp, err := executor.Do(context.Background(), Request{Operation: op, Target: name, Bucket: "physical", Key: key, Source: source})
					if err != nil {
						t.Fatal("outbound request failed after caller-owned target mutation")
					}
					resp.Body.Close()
					if resp.StatusCode != http.StatusOK || len(captures) != 1 {
						t.Fatal("expected one successful captured upstream request")
					}
					got := <-captures
					out := got.request
					cred := credentials[name]
					if out.URL.RawQuery != "" || out.URL.EscapedPath() != "/base%2Fpath/physical/"+key {
						t.Error("inbound authentication query leaked or the snapshotted S3 path changed")
					}
					headers := "host;x-amz-content-sha256;x-amz-date"
					if method == http.MethodPut {
						headers = "content-length;" + headers
						if out.ContentLength != 7 || out.Header.Get("Content-Length") != "7" || string(got.body) != "payload" {
							t.Error("outbound upload lost its signed Content-Length or replayed bytes")
						}
					}
					if cred.SessionToken != "" {
						headers += ";x-amz-security-token"
						if values := out.Header.Values("X-Amz-Security-Token"); len(values) != 1 || values[0] != cred.SessionToken {
							t.Error("outbound request did not use exactly the selected target's token snapshot")
						}
					} else if len(out.Header.Values("X-Amz-Security-Token")) != 0 {
						t.Error("a token reached a tokenless target")
					}
					if len(out.Header.Get("X-Amz-Date")) != 16 || out.Header.Get("X-Amz-Content-Sha256") != unsignedPayload {
						t.Fatal("outbound signing date or unsigned payload behavior changed")
					}
					authorization := out.Header.Get("Authorization")
					if authorization != s3signature.Authorization(out, cred.AccessKey, cred.SecretKey, "us-east-1", headers) {
						t.Error("outbound signature differs from the independent oracle with the required signed headers")
					}
					if cred.SessionToken != "" {
						for _, token := range []string{"CRED_PRIVATE_tampered-token", ""} {
							tampered := out.Clone(context.Background())
							tampered.Header.Del("X-Amz-Security-Token")
							if token != "" {
								tampered.Header.Set("X-Amz-Security-Token", token)
							}
							if authorization == s3signature.Authorization(tampered, cred.AccessKey, cred.SecretKey, "us-east-1", headers) {
								t.Error("independent oracle accepted a mutated or removed session token")
							}
						}
					}
					if !reflect.DeepEqual(source.Header, originalHeaders) || source.URL.RawQuery != originalQuery {
						t.Error("outbound signing mutated caller-owned authentication data")
					}
				}
			})
		}
	}
	if budget.Used() != 0 {
		t.Error("session credential requests leaked replay budget")
	}
}
