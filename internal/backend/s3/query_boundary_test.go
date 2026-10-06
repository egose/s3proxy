package s3

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"

	"github.com/egose/s3proxy/internal/replaybody"
	"github.com/egose/s3proxy/internal/s3ops"
)

func TestClientDo_QueryBoundary(t *testing.T) {
	var calls atomic.Int64
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { calls.Add(1); w.WriteHeader(400) }))
	defer server.Close()
	targets := testTargets(t, server.URL)
	client := newTestClient(t, server.Client(), targets, testBudget())
	for _, tc := range []struct {
		method, key, query string
		op                 s3ops.Operation
	}{
		{"DELETE", "key", "versionId=QUERY_SECRET%GG", s3ops.OpDeleteObject},
		{"DELETE", "key", "tagging=;QUERY_SECRET", s3ops.OpDeleteObject},
		{"PUT", "key", "QUERY_SECRET%=value", s3ops.OpPutObject},
		{"PUT", "key", "x-id=PutObject&x-id=QUERY_SECRET%2", s3ops.OpPutObject},
		{"GET", "", "list-type=2&prefix=QUERY_SECRET%GG", s3ops.OpListObjectsV2},
		{"GET", "", "list-type=2&prefix=keep&prefix=QUERY_SECRET%GG", s3ops.OpListObjectsV2},
	} {
		for _, replay := range []bool{false, true} {
			t.Run(tc.query+"/replay="+map[bool]string{false: "false", true: "true"}[replay], func(t *testing.T) {
				body := &observedBody{}
				src := httptest.NewRequest(tc.method, "/bucket/"+tc.key+"?"+tc.query, body)
				src.ContentLength = -1
				getBodies := 0
				if replay {
					src.GetBody = func() (io.ReadCloser, error) { getBodies++; return body, nil }
				}
				defer replaybody.Release(src)
				before := calls.Load()
				resp, err := client.Do(context.Background(), Request{Operation: tc.op, Target: "primary", Bucket: "bucket", Key: tc.key, Source: src})
				if resp != nil && resp.Body != nil {
					resp.Body.Close()
				}
				if resp != nil || err == nil || err.Error() != "malformed request query" {
					t.Errorf("Do = %v, %v; want safe malformed query error", resp, err)
				}
				if body.reads != 0 || getBodies != 0 || calls.Load() != before {
					t.Errorf("reads/replays/network = %d/%d/%d", body.reads, getBodies, calls.Load()-before)
				}
				u, err := buildTargetURL(targets["primary"], "bucket", tc.key, src)
				if u != nil || err == nil || err.Error() != "malformed request query" {
					t.Errorf("buildTargetURL = %v, %v", u, err)
				}
			})
		}
	}
}
