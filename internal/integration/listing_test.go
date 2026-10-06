//go:build integration

package integration

import (
	"encoding/xml"
	"net/http"
	"net/url"
	"strings"
	"testing"
	"time"
)

func TestNamespaceListingWorkflows(t *testing.T) {
	waitForReady(t, 60*time.Second)
	for _, bucket := range []string{"listed-primary", "listed-replica", "tenant-listaccept-logs"} {
		t.Run(bucket, func(t *testing.T) {
			prefix := "list-" + randHex(4) + "/"
			keys := []string{prefix + "space key", prefix + "percent%2F+雪", prefix + "dir/file"}
			for _, key := range keys {
				resp, data := signedRequest(t, "PUT", "/"+bucket+"/"+key, []byte(key), nil)
				assertStatus(t, resp, data, 200)
				defer func(key string) { signedRequest(t, "DELETE", "/"+bucket+"/"+key, nil, nil) }(key)
			}
			resp, data := signedRequest(t, "HEAD", "/"+bucket, nil, nil)
			assertStatus(t, resp, data, 200)
			q := url.Values{"list-type": {"2"}, "prefix": {prefix}, "max-keys": {"1"}, "encoding-type": {"url"}}
			seen := map[string]bool{}
			for page := 0; ; page++ {
				if page > 5 {
					t.Fatal("pagination did not terminate")
				}
				resp, data := presignedRequest(t, "GET", "/"+bucket, q, time.Now().UTC(), time.Minute)
				assertStatus(t, resp, data, 200)
				var result struct {
					Name, Prefix, NextContinuationToken string
					IsTruncated                         bool
					Contents                            []struct{ Key string }
				}
				if err := xml.Unmarshal(data, &result); err != nil {
					t.Fatal(err)
				}
				if result.Name != bucket {
					t.Fatalf("Name=%q", result.Name)
				}
				visiblePrefix, err := url.PathUnescape(result.Prefix)
				if err != nil || visiblePrefix != prefix {
					t.Fatalf("Prefix=%q err=%v", result.Prefix, err)
				}
				for _, object := range result.Contents {
					key, err := url.PathUnescape(object.Key)
					if err != nil {
						t.Fatal(err)
					}
					if seen[key] || !strings.HasPrefix(key, prefix) {
						t.Fatalf("unexpected key=%q", key)
					}
					seen[key] = true
					resp, data := signedRequest(t, "GET", "/"+bucket+"/"+key, nil, nil)
					assertStatus(t, resp, data, 200)
					if string(data) != key {
						t.Fatalf("GetObject %q = %q", key, data)
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
				t.Fatalf("listed keys=%v", seen)
			}
			q.Del("continuation-token")
			q.Set("max-keys", "100")
			q.Set("delimiter", "/")
			resp, data = signedRequest(t, "GET", "/"+bucket, nil, q)
			assertStatus(t, resp, data, http.StatusOK)
			var result struct{ CommonPrefixes []struct{ Prefix string } }
			if err := xml.Unmarshal(data, &result); err != nil {
				t.Fatal(err)
			}
			if len(result.CommonPrefixes) != 1 {
				t.Fatalf("common prefixes=%+v", result)
			}
			common, err := url.PathUnescape(result.CommonPrefixes[0].Prefix)
			if err != nil || common != prefix+"dir/" {
				t.Fatalf("common prefix=%q err=%v", common, err)
			}
		})
	}
}
