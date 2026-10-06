package httpapi

import (
	"context"
	"encoding/xml"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"net/url"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"text/template"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	v4 "github.com/aws/aws-sdk-go-v2/aws/signer/v4"
	"github.com/egose/s3proxy/internal/auth"
	"github.com/egose/s3proxy/internal/backend/s3"
	"github.com/egose/s3proxy/internal/config"
	"github.com/egose/s3proxy/internal/dispatch"
	"github.com/egose/s3proxy/internal/namespace"
	"github.com/egose/s3proxy/internal/replaybody"
	"github.com/egose/s3proxy/internal/rewrite"
	"github.com/egose/s3proxy/internal/router"
	"github.com/egose/s3proxy/internal/testutil/s3signature"
)

type workflowList struct {
	XMLName               xml.Name `xml:"ListBucketResult"`
	Name                  string
	Prefix                string
	StartAfter            string `xml:",omitempty"`
	Delimiter             string `xml:",omitempty"`
	EncodingType          string `xml:",omitempty"`
	IsTruncated           bool
	ContinuationToken     string `xml:",omitempty"`
	NextContinuationToken string `xml:",omitempty"`
	Contents              []workflowObject
	CommonPrefixes        []workflowPrefix
}

type workflowObject struct {
	Key  string
	ETag string
	Size int
}
type workflowPrefix struct{ Prefix string }

func listingProxy(t *testing.T, upstream string, rule func(string) config.RewriteRule) *httptest.Server {
	t.Helper()
	u, _ := url.Parse(upstream)
	budget := replaybody.NewBudget(replaybody.DefaultMaxBytes, replaybody.DefaultAggregateMaxBytes)
	backend, err := s3.NewClient(&http.Client{}, map[string]config.S3Target{"store": {
		EndpointURL: u, Region: "us-east-1", ForcePathStyle: true, Timeout: 2 * time.Second,
		Credentials: config.StaticCredential{AccessKey: "backend-ak", SecretKey: "backend-sk"},
	}}, budget)
	if err != nil {
		t.Fatal(err)
	}
	dispatcher, err := dispatch.New(backend, budget)
	if err != nil {
		t.Fatal(err)
	}
	clients := map[string]config.Client{}
	parsers := map[string]config.Parser{}
	var routes []config.Route
	for _, tenant := range []string{"acme", "beta"} {
		clients[tenant] = config.Client{Name: tenant, AccessKey: tenant + "-ak", SecretKey: tenant + "-sk", AllowRoutes: []string{tenant}, AllowOps: []string{"ListObjectsV2", "HeadBucket", "PutObject", "GetObject"}}
		parsers[tenant] = config.Parser{Kind: config.ParserBucketRegex, Regex: regexp.MustCompile("^tenant-(?P<tenant>" + tenant + ")-logs$")}
		routes = append(routes, config.Route{Name: tenant, ParserRef: tenant, Operations: clients[tenant].AllowOps, DestinationRefs: []string{"store"}, Dispatch: config.DispatchFirst, ReadPreference: config.ReadFirst, OnMatch: config.MatchStop, Rewrite: rule(tenant)})
	}
	authenticator, err := auth.NewAuthenticator(config.Auth{Mode: config.AuthModeSigV4Static, Clients: clients}, budget)
	if err != nil {
		t.Fatal(err)
	}
	proxy := httptest.NewServer(NewHandler(Dependencies{
		Addressing: config.Addressing{PathStyle: true}, ReplayBudget: budget,
		Authenticator: authenticator, Authorizer: auth.NewAuthorizer(), Router: router.NewResolver(routes, parsers, []string{"store"}),
		Rewriter: rewrite.New(), Dispatcher: dispatcher, Logger: slog.New(slog.NewTextHandler(io.Discard, nil)),
	}))
	t.Cleanup(proxy.Close)
	return proxy
}

func signedListingRequest(t *testing.T, proxy *httptest.Server, tenant, method, path, payload string, presigned bool) (*http.Response, []byte) {
	t.Helper()
	var body io.Reader
	if method == "PUT" {
		body = strings.NewReader(payload)
	}
	r, err := http.NewRequest(method, proxy.URL+path, body)
	if err != nil {
		t.Fatal(err)
	}
	signer := v4.NewSigner(func(o *v4.SignerOptions) { o.DisableURIPathEscaping = true })
	creds := aws.Credentials{AccessKeyID: tenant + "-ak", SecretAccessKey: tenant + "-sk"}
	if presigned {
		q := r.URL.Query()
		q.Set("X-Amz-Expires", "600")
		r.URL.RawQuery = q.Encode()
		uri, _, err := signer.PresignHTTP(context.Background(), creds, r, "UNSIGNED-PAYLOAD", "s3", "us-east-1", time.Now().UTC())
		if err != nil {
			t.Fatal(err)
		}
		r.URL, _ = url.Parse(uri)
	} else {
		r.Header.Set("X-Amz-Content-Sha256", "UNSIGNED-PAYLOAD")
		if method == "PUT" {
			r.Header.Set("Content-Length", strconv.Itoa(len(payload)))
		}
		if err := signer.SignHTTP(context.Background(), creds, r, "UNSIGNED-PAYLOAD", "s3", "us-east-1", time.Now().UTC()); err != nil {
			t.Fatal(err)
		}
	}
	resp, err := proxy.Client().Do(r)
	if err != nil {
		t.Fatal(err)
	}
	data, err := io.ReadAll(resp.Body)
	resp.Body.Close()
	if err != nil {
		t.Fatal(err)
	}
	return resp, data
}

func TestHandler_SignedNamespaceListThenGet(t *testing.T) {
	for _, mode := range []string{"prepend", "template"} {
		t.Run(mode, func(t *testing.T) {
			prefixFor := func(tenant string) string {
				if mode == "prepend" {
					return "assets %2F/" + tenant + "/"
				}
				return tenant + "/"
			}
			var mu sync.Mutex
			objects := map[string]string{}
			tokens := map[string]int{}
			var calls atomic.Int64
			upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				calls.Add(1)
				headers := "host;x-amz-content-sha256;x-amz-date"
				if r.Method == "PUT" {
					headers = "content-length;" + headers
				}
				if r.Header.Get("Authorization") != s3signature.Authorization(r, "backend-ak", "backend-sk", "us-east-1", headers) {
					t.Error("invalid independently verified outbound S3 signature")
					w.WriteHeader(403)
					return
				}
				q := r.URL.Query()
				for key := range q {
					if strings.HasPrefix(strings.ToLower(key), "x-amz-") {
						t.Errorf("inbound auth query leaked: %s", key)
					}
				}
				mu.Lock()
				defer mu.Unlock()
				if r.Method == "HEAD" {
					if r.URL.Path != "/store" {
						t.Errorf("HeadBucket path=%q", r.URL.Path)
					}
					return
				}
				if q.Get("list-type") == "2" {
					if r.URL.Path != "/store" {
						t.Errorf("list path=%q", r.URL.Path)
					}
					prefix := q.Get("prefix")
					if !strings.HasPrefix(prefix, prefixFor("acme")) && !strings.HasPrefix(prefix, prefixFor("beta")) {
						t.Errorf("unscoped prefix=%q", prefix)
					}
					encoded := q.Get("encoding-type") == "url"
					encode := func(s string) string {
						if encoded {
							return url.QueryEscape(s)
						}
						return s
					}
					result := workflowList{Name: "store", Prefix: encode(prefix), EncodingType: q.Get("encoding-type"), Delimiter: encode(q.Get("delimiter")), ContinuationToken: q.Get("continuation-token")}
					if q.Has("start-after") {
						result.StartAfter = encode(q.Get("start-after"))
					}
					var keys []string
					for key := range objects {
						if strings.HasPrefix(key, prefix) && key > q.Get("start-after") {
							keys = append(keys, key)
						}
					}
					sort.Strings(keys)
					start := 0
					if token := q.Get("continuation-token"); token != "" {
						var ok bool
						start, ok = tokens[prefix+"|"+token]
						if !ok {
							t.Error("opaque token changed or switched namespace")
							w.WriteHeader(400)
							return
						}
					}
					end := len(keys)
					if q.Get("delimiter") == "" && end > start+2 {
						end = start + 2
						result.IsTruncated = true
						result.NextContinuationToken = fmt.Sprintf("opaque/%%2F+&=%s:%d", prefix, end)
						tokens[prefix+"|"+result.NextContinuationToken] = end
					}
					seen := map[string]bool{}
					for _, key := range keys[start:end] {
						if delimiter := q.Get("delimiter"); delimiter != "" {
							if i := strings.Index(strings.TrimPrefix(key, prefix), delimiter); i >= 0 {
								common := key[:len(prefix)+i+len(delimiter)]
								if !seen[common] {
									result.CommonPrefixes = append(result.CommonPrefixes, workflowPrefix{encode(common)})
									seen[common] = true
								}
								continue
							}
						}
						result.Contents = append(result.Contents, workflowObject{Key: encode(key), ETag: "object-etag", Size: len(objects[key])})
					}
					data, _ := xml.Marshal(result)
					w.Header().Set("ETag", "stale-list-etag")
					w.Header().Set("Content-MD5", "stale")
					w.Header().Set("X-Amz-Checksum-Sha256", "stale")
					w.Header().Set("Content-Type", "text/xml")
					w.Header().Set("Content-Length", strconv.Itoa(len(data)))
					w.Write(data)
					return
				}
				key := strings.TrimPrefix(r.URL.Path, "/store/")
				if r.Method == "PUT" {
					data, _ := io.ReadAll(r.Body)
					objects[key] = string(data)
					return
				}
				data, ok := objects[key]
				if !ok {
					w.WriteHeader(404)
					return
				}
				io.WriteString(w, data)
			}))
			defer upstream.Close()
			proxy := listingProxy(t, upstream.URL, func(tenant string) config.RewriteRule {
				rw := config.RewriteRule{Bucket: "store"}
				if mode == "prepend" {
					rw.PrependKeyPrefix = "assets%20%252F/" + tenant + "/"
				} else {
					rw.KeyTemplate = "{{ .Captures.tenant }}/{{ .Key }}"
					rw.CompiledTemplate = template.Must(template.New("key").Parse(rw.KeyTemplate))
				}
				return rw
			})
			keys := []string{"space key", "percent%2F+雪", "dir/file", "/leading", "//double", "a/../b", "slash/escaped"}
			for _, tenant := range []string{"acme", "beta"} {
				base := "/tenant-" + tenant + "-logs"
				for _, key := range keys {
					raw := url.PathEscape(key)
					resp, data := signedListingRequest(t, proxy, tenant, "PUT", base+"/"+raw, tenant+":"+key, false)
					if resp.StatusCode != 200 {
						t.Fatalf("PUT status=%d body=%s", resp.StatusCode, data)
					}
				}
				resp, data := signedListingRequest(t, proxy, tenant, "HEAD", base, "", false)
				if resp.StatusCode != 200 {
					t.Fatalf("HEAD status=%d body=%s", resp.StatusCode, data)
				}
				for _, encoded := range []bool{false, true} {
					q := url.Values{"list-type": {"2"}, "prefix": {""}, "max-keys": {"2"}}
					if encoded {
						q.Set("encoding-type", "url")
					}
					seen := map[string]bool{}
					for page := 0; ; page++ {
						if page > 10 {
							t.Fatal("pagination did not terminate")
						}
						resp, data := signedListingRequest(t, proxy, tenant, "GET", base+"?"+q.Encode(), "", page%2 == 1)
						if resp.StatusCode != 200 {
							t.Fatalf("LIST status=%d body=%s", resp.StatusCode, data)
						}
						if resp.Header.Get("ETag") != "" || resp.Header.Get("Content-MD5") != "" || resp.Header.Get("X-Amz-Checksum-Sha256") != "" || resp.ContentLength != int64(len(data)) || resp.Header.Get("Content-Type") != "application/xml" {
							t.Fatalf("transformed headers=%v", resp.Header)
						}
						var result workflowList
						if err := xml.Unmarshal(data, &result); err != nil {
							t.Fatal(err)
						}
						if result.Name != "tenant-"+tenant+"-logs" || result.Prefix != "" || result.ContinuationToken != q.Get("continuation-token") {
							t.Fatalf("namespace/pagination=%+v", result)
						}
						for _, object := range result.Contents {
							key := object.Key
							if encoded {
								key, _ = url.PathUnescape(key)
							}
							if seen[key] {
								t.Fatalf("duplicate key=%q", key)
							}
							seen[key] = true
							resp, bytes := signedListingRequest(t, proxy, tenant, "GET", base+"/"+url.PathEscape(key), "", true)
							if resp.StatusCode != 200 || string(bytes) != tenant+":"+key {
								t.Fatalf("list-then-get %q status=%d bytes=%q", key, resp.StatusCode, bytes)
							}
						}
						if !result.IsTruncated {
							break
						}
						if result.NextContinuationToken == "" {
							t.Fatal("missing token")
						}
						q.Set("continuation-token", result.NextContinuationToken)
					}
					if len(seen) != len(keys) {
						t.Fatalf("listed %v, want %v", seen, keys)
					}
				}
				for _, q := range []url.Values{
					{"list-type": {"2"}, "prefix": {"dir/"}, "encoding-type": {"url"}},
					{"list-type": {"2"}, "prefix": {"percent%2F+"}, "encoding-type": {"url"}},
					{"list-type": {"2"}, "delimiter": {"/"}, "encoding-type": {"url"}},
					{"list-type": {"2"}, "start-after": {"percent%2F+雪"}, "encoding-type": {"url"}},
				} {
					resp, data := signedListingRequest(t, proxy, tenant, "GET", base+"?"+q.Encode(), "", true)
					if resp.StatusCode != 200 {
						t.Fatalf("filtered LIST status=%d body=%s", resp.StatusCode, data)
					}
					var result workflowList
					xml.Unmarshal(data, &result)
					prefix, _ := url.PathUnescape(result.Prefix)
					start, _ := url.PathUnescape(result.StartAfter)
					if prefix != q.Get("prefix") || start != q.Get("start-after") {
						t.Fatalf("echo=%+v query=%v", result, q)
					}
					if q.Get("delimiter") != "" && len(result.CommonPrefixes) == 0 {
						t.Fatal("missing common prefixes")
					}
					for _, p := range result.CommonPrefixes {
						value, _ := url.PathUnescape(p.Prefix)
						if strings.Contains(value, prefixFor(tenant)) {
							t.Fatalf("untranslated common prefix=%q", value)
						}
					}
				}
			}
			before := calls.Load()
			resp, _ := signedListingRequest(t, proxy, "acme", "GET", "/tenant-beta-logs?list-type=2", "", true)
			if resp.StatusCode != 403 || calls.Load() != before {
				t.Fatal("tenant route authorization failed")
			}
		})
	}
}

func TestHandler_ListingFailuresAreAtomic(t *testing.T) {
	var calls atomic.Int64
	var response atomic.Value
	response.Store(`<ListBucketResult><Name>store</Name><Prefix>acme/</Prefix><IsTruncated>false</IsTruncated><Contents><Key>beta/secret</Key></Contents></ListBucketResult>`)
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		io.WriteString(w, response.Load().(string))
	}))
	defer upstream.Close()
	proxy := listingProxy(t, upstream.URL, func(string) config.RewriteRule { return config.RewriteRule{Bucket: "store", PrependKeyPrefix: "acme/"} })
	for _, input := range []string{response.Load().(string), "<ListBucketResult>", strings.Repeat("x", namespace.MaxResponseBytes+1)} {
		response.Store(input)
		resp, data := signedListingRequest(t, proxy, "acme", "GET", "/tenant-acme-logs?list-type=2", "", false)
		if resp.StatusCode != 502 || strings.Contains(string(data), "secret") || strings.Contains(string(data), "<ListBucketResult>") {
			t.Fatalf("status=%d body=%s", resp.StatusCode, data)
		}
	}
	for _, query := range []string{"list-type=2&encoding-type=invalid", "list-type=2&prefix=a&prefix=b"} {
		before := calls.Load()
		resp, _ := signedListingRequest(t, proxy, "acme", "GET", "/tenant-acme-logs?"+query, "", false)
		if resp.StatusCode != 400 || calls.Load() != before {
			t.Fatalf("query %s: status=%d calls=%d", query, resp.StatusCode, calls.Load()-before)
		}
	}
	badProxy := listingProxy(t, upstream.URL, func(string) config.RewriteRule {
		return config.RewriteRule{Bucket: "store", StripKeyPrefix: "private/"}
	})
	before := calls.Load()
	resp, _ := signedListingRequest(t, badProxy, "acme", "GET", "/tenant-acme-logs?list-type=2", "", false)
	if resp.StatusCode != 400 || calls.Load() != before {
		t.Fatal("unsupported mapping reached backend")
	}
}
