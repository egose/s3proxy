//go:build integration

package integration

import (
	"crypto/rand"
	"encoding/hex"
	"net/http"
	"net/url"
	"strings"
	"testing"
	"time"
)

// TestHealthReady exercises the unauthenticated health endpoints. The
// stack must be up before tests run; waitForReady blocks up to 60s.
func TestHealthReady(t *testing.T) {
	waitForReady(t, 60*time.Second)
	resp, body := signedRequest(t, http.MethodGet, "/healthz", nil, nil)
	_ = body
	assertStatus(t, resp, body, 200)
	resp, body = signedRequest(t, http.MethodGet, "/readyz", nil, nil)
	assertStatus(t, resp, body, 200)
}

// TestSingleDestRoundTrip proves the simplest end-to-end path: a signed
// PutObject against /primary/<key> hits MinIO, and a subsequent GetObject
// returns the same bytes. Exercises outbound SigV4 signing against MinIO's
// real S3 API.
func TestSingleDestRoundTrip(t *testing.T) {
	waitForReady(t, 60*time.Second)
	key := "roundtrip-" + randHex(8)
	body := []byte("hello from primary single-dest roundtrip " + key)

	resp, respBody := signedRequest(t, http.MethodPut, "/primary/"+key, body, nil)
	assertStatus(t, resp, respBody, 200)

	resp, respBody = signedRequest(t, http.MethodGet, "/primary/"+key, nil, nil)
	assertStatus(t, resp, respBody, 200)
	if string(respBody) != string(body) {
		t.Errorf("returned body %q does not match written %q", string(respBody), string(body))
	}

	// HeadObject should report content-length.
	resp, _ = signedRequest(t, http.MethodHead, "/primary/"+key, nil, nil)
	assertStatus(t, resp, respBody, 200)
	if resp.ContentLength != int64(len(body)) {
		t.Errorf("Content-Length = %d, want %d", resp.ContentLength, len(body))
	}
}

func TestHeadBucket(t *testing.T) {
	waitForReady(t, 60*time.Second)
	resp, respBody := signedRequest(t, http.MethodHead, "/primary", nil, nil)
	assertStatus(t, resp, respBody, 200)
}

func TestTamperedSignature(t *testing.T) {
	waitForReady(t, 60*time.Second)
	r := newProxyRequest(t, http.MethodGet, "/primary/does-not-exist-"+randHex(8), nil, nil)
	signRequest(t, r, nil)
	r.URL.Path = "/primary/tampered-" + randHex(8)
	resp, respBody := doRequest(t, r)
	assertStatus(t, resp, respBody, 403)
	assertBodyContains(t, respBody, "SignatureDoesNotMatch")
}

func TestVirtualHostedRoundTrip(t *testing.T) {
	waitForReady(t, 60*time.Second)
	key := "virtual-" + randHex(8)
	body := []byte("virtual hosted " + key)
	host := "virtual-bucket.s3proxy.test"

	resp, respBody := signedHostRequest(t, http.MethodPut, host, "/"+key, body, nil)
	assertStatus(t, resp, respBody, 200)

	resp, respBody = signedHostRequest(t, http.MethodGet, host, "/"+key, nil, nil)
	assertStatus(t, resp, respBody, 200)
	if string(respBody) != string(body) {
		t.Errorf("returned body %q does not match written %q", string(respBody), string(body))
	}
}

func TestNamedCaptureRewrite(t *testing.T) {
	waitForReady(t, 60*time.Second)
	tenant := "tenant" + randHex(3)
	key := "log-" + randHex(8)
	body := []byte("tenant log " + key)
	host := "tenant-" + tenant + "-logs.s3proxy.test"

	resp, respBody := signedHostRequest(t, http.MethodPut, host, "/"+key, body, nil)
	assertStatus(t, resp, respBody, 200)

	resp, respBody = signedRequest(t, http.MethodGet, "/primary/"+tenant+"/"+key, nil, nil)
	assertStatus(t, resp, respBody, 200)
	if string(respBody) != string(body) {
		t.Errorf("rewritten body %q, want %q", string(respBody), string(body))
	}
}

// TestSingleDestSeaweedFS forwards to the SeaweedFS backend via the /replica/*
// prefix. This validates that routing can reach an alternate backend whose
// endpoint, addressing, and credentials differ from MinIO.
func TestSingleDestSeaweedFS(t *testing.T) {
	waitForReady(t, 60*time.Second)
	key := "seaweed-" + randHex(8)
	body := []byte("hello from replica " + key)

	resp, respBody := signedRequest(t, http.MethodPut, "/replica/"+key, body, nil)
	assertStatus(t, resp, respBody, 200)

	resp, respBody = signedRequest(t, http.MethodGet, "/replica/"+key, nil, nil)
	assertStatus(t, resp, respBody, 200)
	if string(respBody) != string(body) {
		t.Errorf("returned body %q does not match written %q", string(respBody), string(body))
	}

	// The /primary/* prefix must NOT find the same key, proving the two
	// routes route to genuinely different backends.
	resp, _ = signedRequest(t, http.MethodGet, "/primary/"+key, nil, nil)
	assertStatus(t, resp, respBody, 404)
}

// TestFanoutWriteReplication proves two-destination fan-out: a PutObject
// against /replicate/* is written to BOTH backends. We then read it back
// from each backend separately via the /primary and /replica prefixes.
func TestFanoutWriteReplication(t *testing.T) {
	waitForReady(t, 60*time.Second)
	key := "fanout-" + randHex(8)
	body := []byte("fan-out write " + key)

	resp, respBody := signedRequest(t, http.MethodPut, "/replicate/"+key, body, nil)
	assertStatus(t, resp, respBody, 200)

	// Read back from primary (MinIO).
	resp, respBody = signedRequest(t, http.MethodGet, "/primary/"+key, nil, nil)
	assertStatus(t, resp, respBody, 200)
	if string(respBody) != string(body) {
		t.Errorf("primary returned %q, want %q", string(respBody), string(body))
	}

	// Read back from replica (SeaweedFS).
	resp, respBody = signedRequest(t, http.MethodGet, "/replica/"+key, nil, nil)
	assertStatus(t, resp, respBody, 200)
	if string(respBody) != string(body) {
		t.Errorf("replica returned %q, want %q", string(respBody), string(body))
	}
}

func TestFanoutPrimaryHTTPFailureReturnsPrimaryError(t *testing.T) {
	waitForReady(t, 60*time.Second)
	key := "fanout-primary-http-fail-" + randHex(8)
	body := []byte("fanout primary fail")

	resp, respBody := signedRequest(t, http.MethodPut, "/fanout-primary-http-fail/"+key, body, nil)
	assertStatus(t, resp, respBody, 403)
	assertBodyContains(t, respBody, "InvalidAccessKeyId")

	resp, respBody = signedRequest(t, http.MethodGet, "/primary/"+key, nil, nil)
	assertStatus(t, resp, respBody, 200)
	if string(respBody) != string(body) {
		t.Errorf("primary returned %q, want %q", string(respBody), string(body))
	}
}

func TestFanoutReplicaHTTPFailureReturnsFailure(t *testing.T) {
	waitForReady(t, 60*time.Second)
	key := "fanout-replica-http-fail-" + randHex(8)
	body := []byte("fanout replica fail")

	resp, respBody := signedRequest(t, http.MethodPut, "/fanout-replica-http-fail/"+key, body, nil)
	assertStatus(t, resp, respBody, 502)
	assertBodyContains(t, respBody, "InternalError")

	resp, respBody = signedRequest(t, http.MethodGet, "/primary/"+key, nil, nil)
	assertStatus(t, resp, respBody, 200)
	if string(respBody) != string(body) {
		t.Errorf("primary returned %q, want %q", string(respBody), string(body))
	}
}

func TestFanoutTransportFailureReturnsProxyError(t *testing.T) {
	waitForReady(t, 60*time.Second)
	key := "fanout-transport-fail-" + randHex(8)
	body := []byte("fanout transport fail")

	resp, respBody := signedRequest(t, http.MethodPut, "/fanout-transport-fail/"+key, body, nil)
	assertStatus(t, resp, respBody, 502)
	assertBodyContains(t, respBody, "InternalError")

	resp, respBody = signedRequest(t, http.MethodGet, "/primary/"+key, nil, nil)
	assertStatus(t, resp, respBody, 200)
	if string(respBody) != string(body) {
		t.Errorf("primary returned %q, want %q", string(respBody), string(body))
	}
}

func TestReplayLimitReturnsEntityTooLarge(t *testing.T) {
	waitForReady(t, 60*time.Second)
	key := "replay-too-large-" + randHex(8)
	body := []byte(strings.Repeat("x", 65))

	resp, respBody := signedRequest(t, http.MethodPut, "/replicate/"+key, body, nil)
	assertStatus(t, resp, respBody, 413)
	assertBodyContains(t, respBody, "EntityTooLarge")
}

// TestFanoutDelete also deletes across both backends. After delete, both
// backends report 404 via their respective routes.
func TestFanoutDelete(t *testing.T) {
	waitForReady(t, 60*time.Second)
	key := "del-" + randHex(8)
	body := []byte("to be deleted " + key)

	_, respBody := signedRequest(t, http.MethodPut, "/replicate/"+key, body, nil)
	_ = respBody

	resp, respBody := signedRequest(t, http.MethodDelete, "/replicate/"+key, nil, nil)
	assertStatus(t, resp, respBody, 204)

	resp, _ = signedRequest(t, http.MethodGet, "/primary/"+key, nil, nil)
	assertStatus(t, resp, respBody, 404)
	resp, _ = signedRequest(t, http.MethodGet, "/replica/"+key, nil, nil)
	assertStatus(t, resp, respBody, 404)
}

// TestOrderedFailoverRead proves read-side failover: the /failover/* route
// tries an intentionally missing first destination, then retries the read
// against SeaweedFS and returns that response.
func TestOrderedFailoverRead(t *testing.T) {
	waitForReady(t, 60*time.Second)
	key := "failover-" + randHex(8)
	body := []byte("ordered failover " + key)

	resp, respBody := signedRequest(t, http.MethodPut, "/replica/"+key, body, nil)
	assertStatus(t, resp, respBody, 200)

	resp, respBody = signedRequest(t, http.MethodGet, "/failover/"+key, nil, nil)
	assertStatus(t, resp, respBody, 200)
	if string(respBody) != string(body) {
		t.Errorf("returned body %q does not match written %q", string(respBody), string(body))
	}
}

func TestOrderedFailoverOn5xx(t *testing.T) {
	waitForReady(t, 60*time.Second)
	key := "failover5xx-" + randHex(8)
	body := []byte("ordered failover after 5xx " + key)

	resp, respBody := signedRequest(t, http.MethodPut, "/replica/"+key, body, nil)
	assertStatus(t, resp, respBody, 200)

	resp, respBody = signedRequest(t, http.MethodGet, "/failover5xx/"+key, nil, nil)
	assertStatus(t, resp, respBody, 200)
	if string(respBody) != string(body) {
		t.Errorf("returned body %q does not match written %q", string(respBody), string(body))
	}
}

func TestOrderedFailoverDoesNotFailoverOn404(t *testing.T) {
	waitForReady(t, 60*time.Second)
	key := "failover404-" + randHex(8)
	body := []byte("ordered failover should not reach this " + key)

	resp, respBody := signedRequest(t, http.MethodPut, "/replica/"+key, body, nil)
	assertStatus(t, resp, respBody, 200)

	resp, respBody = signedRequest(t, http.MethodGet, "/failover404/"+key, nil, nil)
	assertStatus(t, resp, respBody, 404)
	assertBodyContains(t, respBody, "NoSuchKey")
}

// TestPresignedGet_LongLivedStillValid proves query-signed auth honors
// X-Amz-Expires rather than the tighter 15-minute skew window alone.
func TestPresignedGet_LongLivedStillValid(t *testing.T) {
	waitForReady(t, 60*time.Second)
	key := "presigned-valid-" + randHex(8)
	body := []byte("presigned valid " + key)

	resp, respBody := signedRequest(t, http.MethodPut, "/primary/"+key, body, nil)
	assertStatus(t, resp, respBody, 200)

	resp, respBody = presignedRequest(t, http.MethodGet, "/primary/"+key, nil, time.Now().UTC().Add(-20*time.Minute), time.Hour)
	assertStatus(t, resp, respBody, 200)
	if string(respBody) != string(body) {
		t.Errorf("returned body %q does not match written %q", string(respBody), string(body))
	}
}

// TestPresignedGet_Expired proves query-signed auth rejects expired presigned
// URLs with AccessDenied instead of forwarding them upstream.
func TestPresignedGet_Expired(t *testing.T) {
	waitForReady(t, 60*time.Second)
	resp, respBody := presignedRequest(t, http.MethodGet, "/primary/does-not-exist-"+randHex(8), nil, time.Now().UTC().Add(-2*time.Minute), time.Minute)
	assertStatus(t, resp, respBody, 403)
	assertBodyContains(t, respBody, "AccessDenied")
}

func TestPresignedGet_TamperingRejected(t *testing.T) {
	waitForReady(t, 60*time.Second)
	tests := []struct {
		name    string
		request func() *http.Request
		mutate  func(*http.Request)
	}{
		{
			name: "signature",
			request: func() *http.Request {
				return newProxyRequest(t, http.MethodGet, "/primary/does-not-exist-"+randHex(8), nil, nil)
			},
			mutate: func(r *http.Request) {
				q := r.URL.Query()
				q.Set("X-Amz-Signature", strings.Repeat("a", 64))
				r.URL.RawQuery = q.Encode()
			},
		},
		{
			name: "path",
			request: func() *http.Request {
				return newProxyRequest(t, http.MethodGet, "/primary/does-not-exist-"+randHex(8), nil, nil)
			},
			mutate: func(r *http.Request) {
				r.URL.Path = "/primary/tampered-" + randHex(8)
			},
		},
		{
			name: "query",
			request: func() *http.Request {
				return newProxyRequest(t, http.MethodGet, "/primary/does-not-exist-"+randHex(8), nil, nil)
			},
			mutate: func(r *http.Request) {
				q := r.URL.Query()
				q.Set("response-content-type", "text/plain")
				r.URL.RawQuery = q.Encode()
			},
		},
		{
			name: "signed header",
			request: func() *http.Request {
				r := newProxyRequest(t, http.MethodGet, "/primary/does-not-exist-"+randHex(8), nil, nil)
				r.Header.Set("Range", "bytes=0-1")
				return r
			},
			mutate: func(r *http.Request) {
				r.Header.Set("Range", "bytes=1-2")
			},
		},
		{
			name: "algorithm",
			request: func() *http.Request {
				return newProxyRequest(t, http.MethodGet, "/primary/does-not-exist-"+randHex(8), nil, nil)
			},
			mutate: func(r *http.Request) {
				q := r.URL.Query()
				q.Set("X-Amz-Algorithm", "AWS4-HMAC-SHA1")
				r.URL.RawQuery = q.Encode()
			},
		},
		{
			name: "duplicate signature",
			request: func() *http.Request {
				return newProxyRequest(t, http.MethodGet, "/primary/does-not-exist-"+randHex(8), nil, nil)
			},
			mutate: func(r *http.Request) {
				q := r.URL.Query()
				q.Add("X-Amz-Signature", q.Get("X-Amz-Signature"))
				r.URL.RawQuery = q.Encode()
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := tt.request()
			signPresignedRequest(t, r, time.Now().UTC(), 10*time.Minute)
			tt.mutate(r)
			resp, respBody := doRequest(t, r)
			assertStatus(t, resp, respBody, 403)
		})
	}
}

func TestMultipartAndCopyObjectRejected(t *testing.T) {
	waitForReady(t, 60*time.Second)
	key := "unsupported-" + randHex(8)
	query := url.Values{}
	query.Set("uploads", "")
	resp, respBody := signedRequest(t, http.MethodPut, "/primary/"+key, nil, query)
	assertStatus(t, resp, respBody, 501)
	assertBodyContains(t, respBody, "NotImplemented")

	headers := http.Header{"X-Amz-Copy-Source": []string{"/testbucket/source"}}
	resp, respBody = signedRequestWithHeaders(t, http.MethodPut, "/primary/"+key, nil, nil, headers)
	assertStatus(t, resp, respBody, 501)
	assertBodyContains(t, respBody, "NotImplemented")
}

// TestContinueWriteComposition proves on_match=continue for write-only routes:
// a single PutObject against /compose/* is applied to both route matches, one
// targeting MinIO and one targeting SeaweedFS.
func TestContinueWriteComposition(t *testing.T) {
	waitForReady(t, 60*time.Second)
	key := "compose-" + randHex(8)
	body := []byte("composed write " + key)

	resp, respBody := signedRequest(t, http.MethodPut, "/compose/"+key, body, nil)
	assertStatus(t, resp, respBody, 200)

	resp, respBody = signedRequest(t, http.MethodGet, "/primary/"+key, nil, nil)
	assertStatus(t, resp, respBody, 200)
	if string(respBody) != string(body) {
		t.Errorf("primary returned %q, want %q", string(respBody), string(body))
	}

	resp, respBody = signedRequest(t, http.MethodGet, "/replica/"+key, nil, nil)
	assertStatus(t, resp, respBody, 200)
	if string(respBody) != string(body) {
		t.Errorf("replica returned %q, want %q", string(respBody), string(body))
	}
}

// TestContinueDeleteComposition proves the same multi-route composition path
// for deletes: a single DeleteObject against /compose/* removes the key from
// both backends.
func TestContinueDeleteComposition(t *testing.T) {
	waitForReady(t, 60*time.Second)
	key := "compose-del-" + randHex(8)
	body := []byte("composed delete " + key)

	resp, respBody := signedRequest(t, http.MethodPut, "/compose/"+key, body, nil)
	assertStatus(t, resp, respBody, 200)

	resp, respBody = signedRequest(t, http.MethodDelete, "/compose/"+key, nil, nil)
	assertStatus(t, resp, respBody, 204)

	resp, respBody = signedRequest(t, http.MethodGet, "/primary/"+key, nil, nil)
	assertStatus(t, resp, respBody, 404)
	resp, respBody = signedRequest(t, http.MethodGet, "/replica/"+key, nil, nil)
	assertStatus(t, resp, respBody, 404)
}

// TestListObjectsV2 forwards to MinIO and expects a ListBucketResult XML
// response. We first PutObject a known key and then list it.
func TestListObjectsV2(t *testing.T) {
	waitForReady(t, 60*time.Second)
	keyPrefix := "listv2-" + randHex(4)
	key := keyPrefix + "aaa"
	body := []byte("listed object")

	_, _ = signedRequest(t, http.MethodPut, "/primary/"+key, body, nil)
	q := url.Values{}
	q.Set("list-type", "2")
	q.Set("prefix", keyPrefix)
	resp, respBody := signedRequest(t, http.MethodGet, "/primary/", nil, q)
	assertStatus(t, resp, respBody, 200)
	if !strings.HasPrefix(string(respBody), "<?xml") && !strings.HasPrefix(string(respBody), "<ListBucketResult") {
		t.Errorf("expected ListBucketResult XML\n%s", dumpResponse(resp, respBody))
	}
	assertBodyContains(t, respBody, key)
}

// TestNotForKey returns 404 from GetObject on MinIO when the key does not
// exist. The error XML is the upstream's verbatim response, so it should
// contain MinIO's NoSuchKey Code element. This also exercises the
// ordered_failover invariant: the single-destination route should NOT retry
// on a 404.
func TestNotForKey(t *testing.T) {
	waitForReady(t, 60*time.Second)
	resp, respBody := signedRequest(t, http.MethodGet, "/primary/does-not-exist-"+randHex(8), nil, nil)
	assertStatus(t, resp, respBody, 404)
	assertBodyContains(t, respBody, "NoSuchKey")
}

// TestNotForKeySeaweedFS mirrors TestNotForKey against the /replica/* route.
// SeaweedFS returns a slightly different XML shape from MinIO but the S3
// error Code is the same; we assert that the proxied error body is the
// upstream's verbatim XML (not a generic s3proxy error).
func TestNotForKeySeaweedFS(t *testing.T) {
	waitForReady(t, 60*time.Second)
	resp, respBody := signedRequest(t, http.MethodGet, "/replica/does-not-exist-"+randHex(8), nil, nil)
	assertStatus(t, resp, respBody, 404)
	assertBodyContains(t, respBody, "NoSuchKey")
}

// TestListBuckets returns the virtual bucket list configured in
// integration-config.hcl. Each configured bucket should appear in the XML
// response regardless of which backends expose them.
func TestListBuckets(t *testing.T) {
	waitForReady(t, 60*time.Second)
	resp, respBody := signedRequest(t, http.MethodGet, "/", nil, nil)
	assertStatus(t, resp, respBody, 200)
	assertBodyContains(t, respBody, "primary-bucket")
	assertBodyContains(t, respBody, "replica-bucket")
	assertBodyContains(t, respBody, "replicate-bucket")
	assertBodyContains(t, respBody, "failover-bucket")
	assertBodyContains(t, respBody, "compose-bucket")
	assertBodyContains(t, respBody, "virtual-bucket")
}

// TestAuthFailure sends an unsigned request to a routing prefix. The proxy
// should reject it with an AccessDenied error rather than forwarding.
func TestAuthFailure(t *testing.T) {
	waitForReady(t, 60*time.Second)
	// Build an unsigned request manually so we don't go through signRequest.
	key := "noauth-" + randHex(8)
	u := proxyURL() + "/primary/" + key
	resp, err := http.DefaultClient.Get(u)
	if err != nil {
		t.Fatalf("get failed: %v", err)
	}
	defer resp.Body.Close()
	assertStatus(t, resp, nil, 403)
}

// randHex returns n hex-encoded random bytes.
func randHex(n int) string {
	b := make([]byte, n)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}
