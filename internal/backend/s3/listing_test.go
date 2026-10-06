package s3

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/egose/s3proxy/internal/config"
	"github.com/egose/s3proxy/internal/namespace"
	"github.com/egose/s3proxy/internal/s3ops"
)

type listingBody struct {
	io.Reader
	closes atomic.Int32
	reads  atomic.Int32
}

func (b *listingBody) Read(p []byte) (int, error) { b.reads.Add(1); return b.Reader.Read(p) }
func (b *listingBody) Close() error               { b.closes.Add(1); return nil }

func TestClientListingTransformationOwnsBody(t *testing.T) {
	valid := `<ListBucketResult><Name>store</Name><Prefix>assets/</Prefix><IsTruncated>false</IsTruncated><Contents><Key>assets/a</Key></Contents></ListBucketResult>`
	for _, tt := range []struct {
		name, input string
		status      int
		op          s3ops.Operation
		wantErr     bool
	}{
		{"success", valid, 200, s3ops.OpListObjectsV2, false},
		{"malformed", "<ListBucketResult>", 200, s3ops.OpListObjectsV2, true},
		{"oversized", strings.Repeat("x", namespace.MaxResponseBytes+1), 200, s3ops.OpListObjectsV2, true},
		{"upstream error", "<Error>upstream denial</Error>", 403, s3ops.OpListObjectsV2, false},
		{"object streams", "object bytes", 200, s3ops.OpGetObject, false},
	} {
		t.Run(tt.name, func(t *testing.T) {
			body := &listingBody{Reader: strings.NewReader(tt.input)}
			client := newTestClient(t, &http.Client{Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
				if tt.op == s3ops.OpListObjectsV2 && (r.URL.Path != "/store" || r.URL.Query().Get("prefix") != "assets/") {
					t.Errorf("URL=%s", r.URL)
				}
				return &http.Response{StatusCode: tt.status, Body: body, Header: http.Header{"Etag": {"stale"}, "Content-Length": {"123"}, "Content-Encoding": {"identity"}, "Digest": {"stale"}, "Content-Digest": {"stale"}, "Repr-Digest": {"stale"}, "X-Amz-Checksum-Crc32": {"stale"}, "X-Amz-Content-Sha256": {"stale"}, "Content-Range": {"bytes 0-10/123"}, "Last-Modified": {"stale"}, "X-Amz-Request-Id": {"keep"}}}, nil
			})}, testTargets(t, "http://upstream.test"), testBudget())
			source := httptest.NewRequest("GET", "/visible?list-type=2", nil)
			key := ""
			if tt.op == s3ops.OpGetObject {
				source = httptest.NewRequest("GET", "/visible/a", nil)
				key = "assets/a"
			}
			resp, err := client.Do(context.Background(), Request{Operation: tt.op, Target: "primary", Bucket: "store", Key: key, Source: source, Namespace: &namespace.Mapping{VisibleBucket: "visible", Prefix: "assets/"}})
			if (err != nil) != tt.wantErr {
				t.Fatalf("err=%v wantErr=%v", err, tt.wantErr)
			}
			if tt.op == s3ops.OpListObjectsV2 && tt.status == 200 {
				if body.closes.Load() != 1 {
					t.Fatalf("upstream closes=%d", body.closes.Load())
				}
				if err != nil {
					if resp != nil {
						t.Fatal("partial response on error")
					}
					return
				}
				for _, key := range []string{"ETag", "Content-Encoding", "Digest", "Content-Digest", "Repr-Digest", "X-Amz-Checksum-Crc32", "X-Amz-Content-Sha256", "Content-Range", "Last-Modified"} {
					if resp.Header.Get(key) != "" {
						t.Errorf("stale %s", key)
					}
				}
				if resp.Header.Get("X-Amz-Request-Id") != "keep" {
					t.Fatal("lost upstream request id")
				}
				data, _ := io.ReadAll(resp.Body)
				resp.Body.Close()
				if !strings.Contains(string(data), "<Name>visible</Name>") || !strings.Contains(string(data), "<Key>a</Key>") {
					t.Fatalf("body=%s", data)
				}
			} else {
				if body.reads.Load() != 0 || body.closes.Load() != 0 {
					t.Fatal("stream read/closed before returning")
				}
				data, _ := io.ReadAll(resp.Body)
				resp.Body.Close()
				if string(data) != tt.input || resp.Header.Get("Etag") != "stale" || body.closes.Load() != 1 {
					t.Fatal("streaming/error response changed")
				}
			}
		})
	}
}

func TestClientListingTimeoutAndCancellationCloseHTTPBody(t *testing.T) {
	for _, timeout := range []bool{true, false} {
		t.Run(map[bool]string{true: "target timeout", false: "request cancellation"}[timeout], func(t *testing.T) {
			started := make(chan struct{})
			closed := make(chan struct{})
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				io.WriteString(w, "<ListBucketResult>")
				w.(http.Flusher).Flush()
				close(started)
				<-r.Context().Done()
				close(closed)
			}))
			defer server.Close()
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			if !timeout {
				go func() { <-started; cancel() }()
			}
			targets := testTargets(t, server.URL, func(target *config.S3Target) {
				if timeout {
					target.Timeout = 40 * time.Millisecond
				}
			})
			client := newTestClient(t, server.Client(), targets, testBudget())
			resp, err := client.Do(ctx, Request{Operation: s3ops.OpListObjectsV2, Target: "primary", Bucket: "store", Source: httptest.NewRequest("GET", "/visible?list-type=2", nil)})
			want := context.Canceled
			if timeout {
				want = context.DeadlineExceeded
			}
			if !errors.Is(err, want) || resp != nil {
				t.Fatalf("response=%v err=%v", resp, err)
			}
			select {
			case <-closed:
			case <-time.After(time.Second):
				t.Fatal("upstream request body not closed")
			}
		})
	}
}
