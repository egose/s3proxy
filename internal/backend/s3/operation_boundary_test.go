package s3

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/egose/s3proxy/internal/config"
	"github.com/egose/s3proxy/internal/requestctx"
	"github.com/egose/s3proxy/internal/rewrite"
	"github.com/egose/s3proxy/internal/s3ops"
)

func TestClientDo_RejectsInvalidOperationShapeBeforeIO(t *testing.T) {
	var calls atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()
	client := newTestClient(t, server.Client(), testTargets(t, server.URL), testBudget())
	tests := []struct {
		name, method, bucket, key string
		op                        s3ops.Operation
	}{
		{"empty get", "GET", "bucket", "", s3ops.OpGetObject},
		{"empty head", "HEAD", "bucket", "", s3ops.OpHeadObject},
		{"empty put", "PUT", "bucket", "", s3ops.OpPutObject},
		{"empty delete", "DELETE", "bucket", "", s3ops.OpDeleteObject},
		{"head bucket with key", "HEAD", "bucket", "key", s3ops.OpHeadBucket},
		{"list with key", "GET", "bucket", "key", s3ops.OpListObjectsV2},
		{"wrong method", "DELETE", "bucket", "key", s3ops.OpGetObject},
		{"empty bucket", "GET", "", "key", s3ops.OpGetObject},
		{"bucket contains path", "HEAD", "bucket/key", "", s3ops.OpHeadBucket},
		{"bucket escaped path", "HEAD", "bucket%2Fkey", "", s3ops.OpHeadBucket},
		{"unknown", "GET", "bucket", "key", s3ops.OpUnknown},
		{"undeclared", "DELETE", "bucket", "", "DeleteBucket"},
		{"list v1", "GET", "bucket", "", s3ops.OpListObjectsV1},
		{"list buckets is local", "GET", "", "", s3ops.OpListBuckets},
		{"copy unsupported", "PUT", "bucket", "key", s3ops.OpCopyObject},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			body := &observedBody{}
			src := httptest.NewRequest(tt.method, "http://proxy.local/bucket/key", body)
			src.ContentLength = -1
			before := calls.Load()
			resp, err := client.Do(context.Background(), Request{Operation: tt.op, Target: "primary", Bucket: tt.bucket, Key: tt.key, Source: src})
			if resp != nil && resp.Body != nil {
				resp.Body.Close()
			}
			if err == nil {
				t.Error("expected controlled rejection")
			}
			if body.reads != 0 {
				t.Errorf("body reads = %d, want 0", body.reads)
			}
			if calls.Load() != before {
				t.Error("rejected request reached upstream")
			}
		})
	}
}

type observedBody struct{ reads int }

func (b *observedBody) Read([]byte) (int, error) {
	b.reads++
	return 0, io.EOF
}
func (*observedBody) Close() error { return nil }

func TestClientDo_RejectsMalformedSourceBeforeIO(t *testing.T) {
	client := newTestClient(t, &http.Client{Transport: roundTripFunc(func(*http.Request) (*http.Response, error) {
		t.Error("malformed source reached transport")
		return &http.Response{StatusCode: http.StatusOK, Body: http.NoBody}, nil
	})}, testTargets(t, "https://s3.local"), testBudget())
	for name, mutate := range map[string]func(*Request){
		"nil source":        func(r *Request) { r.Source = nil },
		"nil URL":           func(r *Request) { r.Source.URL = nil },
		"missing list type": func(r *Request) { r.Operation, r.Key = s3ops.OpListObjectsV2, "" },
		"subresource":       func(r *Request) { r.Source.URL.RawQuery = "acl=" },
		"multipart query":   func(r *Request) { r.Source.URL.RawQuery = "uploadId=abc" },
		"multipart header":  func(r *Request) { r.Source.Header.Set("X-Amz-Multipart-Upload-Id", "abc") },
		"copy header": func(r *Request) {
			r.Operation = s3ops.OpPutObject
			r.Source.Method = "PUT"
			r.Source.Header.Set("X-Amz-Copy-Source", "/bucket/source")
		},
	} {
		t.Run(name, func(t *testing.T) {
			body := &observedBody{}
			src := httptest.NewRequest("GET", "/bucket/key", body)
			src.GetBody = func() (io.ReadCloser, error) { t.Error("malformed source replayed body"); return body, nil }
			req := Request{Operation: s3ops.OpGetObject, Target: "primary", Bucket: "bucket", Key: "key", Source: src}
			mutate(&req)
			if _, err := client.Do(context.Background(), req); err == nil {
				t.Fatal("expected controlled rejection")
			}
			if body.reads != 0 {
				t.Fatalf("body reads = %d, want 0", body.reads)
			}
		})
	}
}

func TestClientDo_PreservesRewrittenSlashKeys(t *testing.T) {
	paths := make(chan string, 1)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		paths <- r.URL.EscapedPath()
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()
	client := newTestClient(t, server.Client(), testTargets(t, server.URL+"/base"), testBudget())
	for _, key := range []string{"foo", "/foo", "//foo", "foo//bar", "%2Ffoo"} {
		t.Run(key, func(t *testing.T) {
			rw, err := rewrite.New().Apply(&requestctx.Context{Bucket: "bucket", Key: key}, config.RewriteRule{PrependKeyPrefix: "assets/"}, nil)
			if err != nil {
				t.Fatal(err)
			}
			resp, err := client.Do(context.Background(), Request{Operation: s3ops.OpGetObject, Target: "primary", Bucket: rw.Bucket, Key: rw.Key, Source: httptest.NewRequest("GET", "/bucket/"+key, nil)})
			if err != nil {
				t.Fatal(err)
			}
			resp.Body.Close()
			if got, want := <-paths, "/base/bucket/assets/"+key; got != want {
				t.Fatalf("path = %q, want %q", got, want)
			}
		})
	}
}

func TestClientDo_ValidBucketOperations(t *testing.T) {
	requests := make(chan string, 1)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests <- r.Method + " " + r.URL.RequestURI()
		w.WriteHeader(http.StatusOK)
		if r.Method == http.MethodGet {
			io.WriteString(w, `<ListBucketResult><Name>store</Name><Prefix>assets/</Prefix><IsTruncated>false</IsTruncated></ListBucketResult>`)
		}
	}))
	defer server.Close()
	client := newTestClient(t, server.Client(), testTargets(t, server.URL), testBudget())
	for _, tt := range []struct {
		op            s3ops.Operation
		method, query string
	}{
		{s3ops.OpHeadBucket, "HEAD", ""},
		{s3ops.OpListObjectsV2, "GET", "?list-type=2&prefix=assets%2F"},
	} {
		src := httptest.NewRequest(tt.method, "/visible"+tt.query, nil)
		resp, err := client.Do(context.Background(), Request{Operation: tt.op, Target: "primary", Bucket: "store", Source: src})
		if err != nil {
			t.Fatal(err)
		}
		resp.Body.Close()
		if got, want := <-requests, tt.method+" /store"+tt.query; got != want {
			t.Fatalf("request = %q, want %q", got, want)
		}
	}
}

func TestBuildTargetURL_PreservesLeadingSlashWithVirtualHostBasePath(t *testing.T) {
	for _, key := range []string{"foo", "/foo", "//foo"} {
		target := testTargets(t, "https://s3.local/base")["primary"]
		target.ForcePathStyle = false
		u, err := buildTargetURL(target, "bucket", key, httptest.NewRequest("GET", "/bucket/key", nil))
		if err != nil {
			t.Fatal(err)
		}
		if got, want := u.EscapedPath(), "/base/"+key; got != want {
			t.Errorf("path = %q, want %q", got, want)
		}
		if !strings.HasPrefix(u.Host, "bucket.") {
			t.Fatalf("host = %q", u.Host)
		}
	}
}
