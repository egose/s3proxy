package s3

import (
	"context"
	"errors"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/egose/s3proxy/internal/config"
	"github.com/egose/s3proxy/internal/s3ops"
)

func mustURL(t *testing.T, raw string) *url.URL {
	t.Helper()
	u, err := url.Parse(raw)
	if err != nil {
		t.Fatal(err)
	}
	return u
}

func TestBuildTargetURL_PreservesEndpointBasePath(t *testing.T) {
	src, err := http.NewRequest(http.MethodGet, "http://proxy.local/source/object?list-type=2", nil)
	if err != nil {
		t.Fatal(err)
	}

	targetURL, err := buildTargetURL(config.S3Target{
		Endpoint:       "https://minio.internal/base/path",
		EndpointURL:    mustURL(t, "https://minio.internal/base/path"),
		ForcePathStyle: true,
	}, "bucket", "key", src)
	if err != nil {
		t.Fatal(err)
	}

	if got, want := targetURL.Path, "/base/path/bucket/key"; got != want {
		t.Fatalf("Path = %q, want %q", got, want)
	}
	if got, want := targetURL.RawQuery, "list-type=2"; got != want {
		t.Fatalf("RawQuery = %q, want %q", got, want)
	}
}

func TestBuildTargetURL_PreservesEscapedKey(t *testing.T) {
	src, err := http.NewRequest(http.MethodGet, "http://proxy.local/source/object", nil)
	if err != nil {
		t.Fatal(err)
	}

	targetURL, err := buildTargetURL(config.S3Target{
		Endpoint:       "https://minio.internal",
		EndpointURL:    mustURL(t, "https://minio.internal"),
		ForcePathStyle: true,
	}, "bucket", "path%2Fwith%2Fslash.txt", src)
	if err != nil {
		t.Fatal(err)
	}
	if got, want := targetURL.RawPath, "/bucket/path%2Fwith%2Fslash.txt"; got != want {
		t.Fatalf("RawPath = %q, want %q", got, want)
	}
	if got, want := targetURL.Path, "/bucket/path/with/slash.txt"; got != want {
		t.Fatalf("Path = %q, want %q", got, want)
	}
}

func TestBuildTargetURL_StripsInboundAuthQueryParams(t *testing.T) {
	src, err := http.NewRequest(http.MethodGet, "http://proxy.local/source/object?list-type=2&X-Amz-Algorithm=AWS4-HMAC-SHA256&X-Amz-Credential=test&X-Amz-Signature=deadbeef", nil)
	if err != nil {
		t.Fatal(err)
	}

	targetURL, err := buildTargetURL(config.S3Target{
		Endpoint:       "https://minio.internal/base/path",
		EndpointURL:    mustURL(t, "https://minio.internal/base/path"),
		ForcePathStyle: true,
	}, "bucket", "key", src)
	if err != nil {
		t.Fatal(err)
	}

	query, err := url.ParseQuery(targetURL.RawQuery)
	if err != nil {
		t.Fatal(err)
	}
	if got, want := query.Get("list-type"), "2"; got != want {
		t.Fatalf("list-type = %q, want %q", got, want)
	}
	if got := query.Get("X-Amz-Algorithm"); got != "" {
		t.Fatalf("expected X-Amz-Algorithm to be stripped, got %q", got)
	}
	if got := query.Get("X-Amz-Credential"); got != "" {
		t.Fatalf("expected X-Amz-Credential to be stripped, got %q", got)
	}
	if got := query.Get("X-Amz-Signature"); got != "" {
		t.Fatalf("expected X-Amz-Signature to be stripped, got %q", got)
	}
}

func TestClientDo_RespectsTargetTimeout(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(100 * time.Millisecond)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	src, err := http.NewRequest(http.MethodGet, "http://proxy.local/bucket/key", nil)
	if err != nil {
		t.Fatal(err)
	}

	client := NewClient(&http.Client{})
	_, err = client.Do(context.Background(), Request{
		Operation: s3ops.OpGetObject,
		Target: config.S3Target{
			Endpoint:       server.URL,
			EndpointURL:    mustURL(t, server.URL),
			Region:         "us-east-1",
			ForcePathStyle: true,
			Timeout:        20 * time.Millisecond,
			Credentials: config.StaticCredential{
				AccessKey: "ak",
				SecretKey: "sk",
			},
		},
		Bucket: "bucket",
		Key:    "key",
		Source: src,
	})
	if err == nil {
		t.Fatal("expected timeout error")
	}
	if !errors.Is(err, context.DeadlineExceeded) {
		var netErr net.Error
		if !errors.As(err, &netErr) || !netErr.Timeout() {
			if !strings.Contains(err.Error(), "timeout") && !strings.Contains(err.Error(), "deadline exceeded") {
				t.Fatalf("expected timeout error, got %v", err)
			}
		}
	}
}

func TestClientDo_StreamsKnownLengthSourceBody(t *testing.T) {
	var bodies []string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, err := io.ReadAll(r.Body)
		if err != nil {
			t.Fatalf("read body: %v", err)
		}
		bodies = append(bodies, string(body))
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	src, err := http.NewRequest(http.MethodPut, "http://proxy.local/bucket/key", nil)
	if err != nil {
		t.Fatal(err)
	}
	src.Body = io.NopCloser(strings.NewReader("payload"))
	src.ContentLength = int64(len("payload"))

	client := NewClient(&http.Client{})
	request := Request{
		Operation: s3ops.OpPutObject,
		Target: config.S3Target{
			Endpoint:       server.URL,
			EndpointURL:    mustURL(t, server.URL),
			Region:         "us-east-1",
			ForcePathStyle: true,
			Credentials: config.StaticCredential{
				AccessKey: "ak",
				SecretKey: "sk",
			},
		},
		Bucket: "bucket",
		Key:    "key",
		Source: src,
	}

	resp, err := client.Do(context.Background(), request)
	if err != nil {
		t.Fatalf("Do failed: %v", err)
	}
	resp.Body.Close()

	if got, want := len(bodies), 1; got != want {
		t.Fatalf("calls = %d, want %d", got, want)
	}
	for i, body := range bodies {
		if body != "payload" {
			t.Fatalf("body[%d] = %q, want payload", i, body)
		}
	}
	if src.GetBody != nil {
		t.Fatal("expected known-length single-destination source body to stay streaming")
	}
}

func TestClientDo_ReusesReplayableSourceBody(t *testing.T) {
	var bodies []string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, err := io.ReadAll(r.Body)
		if err != nil {
			t.Fatalf("read body: %v", err)
		}
		bodies = append(bodies, string(body))
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	src, err := http.NewRequest(http.MethodPut, "http://proxy.local/bucket/key", nil)
	if err != nil {
		t.Fatal(err)
	}
	src.GetBody = func() (io.ReadCloser, error) {
		return io.NopCloser(strings.NewReader("payload")), nil
	}
	src.Body, err = src.GetBody()
	if err != nil {
		t.Fatal(err)
	}
	src.ContentLength = int64(len("payload"))

	client := NewClient(&http.Client{})
	request := Request{
		Operation: s3ops.OpPutObject,
		Target: config.S3Target{
			Endpoint:       server.URL,
			EndpointURL:    mustURL(t, server.URL),
			Region:         "us-east-1",
			ForcePathStyle: true,
			Credentials: config.StaticCredential{
				AccessKey: "ak",
				SecretKey: "sk",
			},
		},
		Bucket: "bucket",
		Key:    "key",
		Source: src,
	}

	resp, err := client.Do(context.Background(), request)
	if err != nil {
		t.Fatalf("first Do failed: %v", err)
	}
	resp.Body.Close()
	resp, err = client.Do(context.Background(), request)
	if err != nil {
		t.Fatalf("second Do failed: %v", err)
	}
	resp.Body.Close()

	if got, want := len(bodies), 2; got != want {
		t.Fatalf("calls = %d, want %d", got, want)
	}
	for i, body := range bodies {
		if body != "payload" {
			t.Fatalf("body[%d] = %q, want payload", i, body)
		}
	}
}

func TestClientDo_DoesNotForwardHopByHopHeaders(t *testing.T) {
	var gotConnection string
	var gotCustom string
	var gotTE string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotConnection = r.Header.Get("Connection")
		gotCustom = r.Header.Get("X-Remove-Me")
		gotTE = r.Header.Get("TE")
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	src, err := http.NewRequest(http.MethodGet, "http://proxy.local/bucket/key", nil)
	if err != nil {
		t.Fatal(err)
	}
	src.Header.Set("Connection", "X-Remove-Me")
	src.Header.Set("X-Remove-Me", "secret")
	src.Header.Set("TE", "trailers")

	client := NewClient(&http.Client{})
	resp, err := client.Do(context.Background(), Request{
		Operation: s3ops.OpGetObject,
		Target: config.S3Target{
			Endpoint:       server.URL,
			EndpointURL:    mustURL(t, server.URL),
			Region:         "us-east-1",
			ForcePathStyle: true,
			Credentials: config.StaticCredential{
				AccessKey: "ak",
				SecretKey: "sk",
			},
		},
		Bucket: "bucket",
		Key:    "key",
		Source: src,
	})
	if err != nil {
		t.Fatalf("Do failed: %v", err)
	}
	resp.Body.Close()

	if gotConnection != "" {
		t.Fatalf("expected Connection to be stripped, got %q", gotConnection)
	}
	if gotCustom != "" {
		t.Fatalf("expected X-Remove-Me to be stripped, got %q", gotCustom)
	}
	if gotTE != "" {
		t.Fatalf("expected TE to be stripped, got %q", gotTE)
	}
}
