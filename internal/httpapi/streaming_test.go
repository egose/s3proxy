package httpapi

import (
	"bytes"
	"context"
	"crypto/sha256"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/egose/s3proxy/internal/backend/s3"
	"github.com/egose/s3proxy/internal/config"
	"github.com/egose/s3proxy/internal/dispatch"
	"github.com/egose/s3proxy/internal/namespace"
	"github.com/egose/s3proxy/internal/router"
)

func TestHandler_ResponseTransferHTTP(t *testing.T) {
	for _, path := range []struct {
		name   string
		url    string
		status int
		err    error
	}{
		{"object", "/bucket/key", http.StatusOK, nil},
		{"listing", "/bucket?list-type=2", http.StatusOK, nil},
		{"upstream_error", "/bucket/key", http.StatusForbidden, nil},
		{"dispatch_error", "/bucket/key", http.StatusForbidden, errors.New("fan-out failed")},
	} {
		for _, knownLength := range []bool{false, true} {
			for _, flushed := range []bool{false, true} {
				for _, outcome := range []string{"copy_error", "copy_and_close_error", "complete", "close_error"} {
					name := fmt.Sprintf("%s/known_length=%t/flushed=%t/%s", path.name, knownLength, flushed, outcome)
					t.Run(name, func(t *testing.T) {
						payload := "object bytes"
						if path.status >= 400 {
							payload = "<Error><Code>AccessDenied</Code></Error>"
						} else if path.name == "listing" {
							payload = "<ListBucketResult><Name>bucket</Name><Prefix></Prefix><IsTruncated>false</IsTruncated></ListBucketResult>"
						}
						copyFailure := strings.HasPrefix(outcome, "copy_")
						body := &streamTestBody{reader: strings.NewReader(payload)}
						safeError := &url.Error{Op: "Get", URL: "http://user:password@backend.local/private-bucket/private-key?token=private-token", Err: errors.New("stream failed")}
						if copyFailure {
							body.readErr = safeError
						}
						if strings.Contains(outcome, "close_error") {
							body.closeErr = &url.Error{Op: "Close", URL: safeError.URL, Err: errors.New("close failed")}
						}
						header := http.Header{"Content-Type": {"application/octet-stream"}}
						if knownLength {
							length := len(payload)
							if copyFailure {
								length += 100
							}
							header.Set("Content-Length", fmt.Sprint(length))
						}
						var logs bytes.Buffer
						h := streamTestHandler(body, header, path.status, path.err, &logs)
						done := make(chan struct{})
						server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
							defer close(done)
							if flushed {
								w = flushEachWrite{w}
							}
							h.ServeHTTP(w, r)
						}))
						defer server.Close()
						client := server.Client()
						client.Timeout = 5 * time.Second
						req, err := http.NewRequest(http.MethodGet, server.URL+path.url, nil)
						if err != nil {
							t.Fatal(err)
						}
						req.Header.Set("X-Request-Id", "stream-test")
						resp, requestErr := client.Do(req)
						var data []byte
						var readErr error
						if requestErr == nil {
							data, readErr = io.ReadAll(resp.Body)
							resp.Body.Close()
							if resp.StatusCode != path.status {
								t.Errorf("status = %d, want %d", resp.StatusCode, path.status)
							}
						}
						select {
						case <-done:
						case <-time.After(5 * time.Second):
							t.Fatal("handler did not finish")
						}
						if copyFailure {
							if requestErr == nil && readErr == nil {
								t.Errorf("partial response accepted as complete: %q", data)
							}
							if flushed && (requestErr != nil || !errors.Is(readErr, io.ErrUnexpectedEOF) || string(data) != payload) {
								t.Errorf("flushed response = %q, request error = %v, read error = %v; want payload and unexpected EOF", data, requestErr, readErr)
							}
							if !flushed && requestErr == nil {
								t.Error("buffered response escaped before abort; want request failure before headers")
							}
							if len(data) > 0 && !strings.HasPrefix(payload, string(data)) {
								t.Errorf("unexpected bytes appended: %q", data)
							}
						} else if requestErr != nil || readErr != nil || string(data) != payload {
							t.Errorf("response = %q, request error = %v, read error = %v; want complete payload", data, requestErr, readErr)
						}
						if body.closes != 1 {
							t.Errorf("body closes = %d, want 1", body.closes)
						}
						text := logs.String()
						for _, want := range []string{"request_id=stream-test", "request complete", "method=GET", fmt.Sprintf("status=%d", path.status), fmt.Sprintf("bytes=%d", len(payload)), "duration_ms="} {
							if !strings.Contains(text, want) {
								t.Errorf("logs = %q, missing %q", text, want)
							}
						}
						if strings.Count(text, "request complete") != 1 {
							t.Errorf("want one completion record: %q", text)
						}
						if strings.Contains(text, "response copy failed") != copyFailure || (copyFailure && !strings.Contains(text, "stream failed")) {
							t.Errorf("copy failure telemetry = %q", text)
						}
						if body.closeErr != nil && (!strings.Contains(text, "response cleanup failed") || !strings.Contains(text, "close failed")) {
							t.Errorf("missing cleanup failure telemetry: %q", text)
						}
						for _, secret := range []string{"user:password", "private-bucket", "private-key", "private-token"} { // pragma: allowlist secret
							if strings.Contains(text, secret) {
								t.Errorf("logs expose %q: %q", secret, text)
							}
						}
					})
				}
			}
		}
	}
}

func streamTestHandler(body io.ReadCloser, header http.Header, status int, dispatchErr error, logs io.Writer) http.Handler {
	return NewHandler(Dependencies{
		Addressing:    config.Addressing{PathStyle: true},
		Authenticator: stubAuthenticator{},
		Authorizer:    stubAuthorizer{},
		Router:        stubResolver{matches: []router.Match{{Route: config.Route{Name: "stream"}}}},
		Rewriter:      stubRewriter{},
		Dispatcher: &stubFanout{
			results: []*dispatch.Result{{Primary: &s3.Response{StatusCode: status, Header: header, Body: body}}},
			errs:    []error{dispatchErr},
		},
		Logger: slog.New(slog.NewTextHandler(logs, nil)),
	})
}

func TestHandler_LargeResponseStreamsBeforeUpstreamEOF(t *testing.T) {
	for _, knownLength := range []bool{false, true} {
		t.Run(fmt.Sprintf("known_length=%t", knownLength), func(t *testing.T) {
			chunk := bytes.Repeat([]byte("streaming bytes\x00"), 4096)
			const chunks = 128
			header := http.Header{}
			if knownLength {
				header.Set("Content-Length", fmt.Sprint(len(chunk)*chunks))
			}
			reader, writer := io.Pipe()
			defer reader.Close()
			release := make(chan struct{})
			var once sync.Once
			unblock := func() { once.Do(func() { close(release) }) }
			producerDone := make(chan error, 1)
			go func() {
				defer writer.Close()
				for i := 0; i < chunks; i++ {
					if i == 1 {
						<-release
					}
					if _, err := writer.Write(chunk); err != nil {
						producerDone <- err
						return
					}
				}
				producerDone <- nil
			}()
			body := &streamTestBody{reader: reader}
			var logs bytes.Buffer
			h := streamTestHandler(body, header, http.StatusOK, nil, &logs)
			done := make(chan struct{})
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				defer close(done)
				h.ServeHTTP(w, r)
			}))
			defer server.Close()
			defer unblock()
			client := server.Client()
			client.Timeout = 5 * time.Second
			resp, err := client.Get(server.URL + "/bucket/key")
			if err != nil {
				t.Fatal(err)
			}
			defer resp.Body.Close()
			prefix := make([]byte, 4096)
			if _, err := io.ReadFull(resp.Body, prefix); err != nil {
				t.Fatalf("cannot read while upstream awaits client progress: %v", err)
			}
			unblock()
			got := sha256.New()
			got.Write(prefix)
			n, err := io.Copy(got, resp.Body)
			if err != nil || n+int64(len(prefix)) != int64(len(chunk)*chunks) {
				t.Fatalf("body bytes after prefix = %d, error = %v", n, err)
			}
			want := sha256.New()
			for i := 0; i < chunks; i++ {
				want.Write(chunk)
			}
			if !bytes.Equal(got.Sum(nil), want.Sum(nil)) {
				t.Fatal("streamed bytes differ")
			}
			if err := <-producerDone; err != nil {
				t.Fatal(err)
			}
			select {
			case <-done:
			case <-time.After(5 * time.Second):
				t.Fatal("handler did not finish")
			}
			if body.closes != 1 || !strings.Contains(logs.String(), fmt.Sprintf("bytes=%d", len(chunk)*chunks)) || strings.Contains(logs.String(), "failed") {
				t.Fatalf("closes = %d, logs = %q", body.closes, logs.String())
			}
		})
	}
}

func TestHandler_ResponseTransferPanicAndWriteFailureCleanup(t *testing.T) {
	for _, failure := range []string{"read_panic", "header_panic", "write_panic", "write_error", "short_write", "listing_write_error"} {
		t.Run(failure, func(t *testing.T) {
			sentinel := errors.New("transfer panic")
			body := &streamTestBody{reader: strings.NewReader("payload")}
			if failure == "read_panic" {
				body.reader = streamPanicReader{sentinel}
			}
			var logs bytes.Buffer
			path := "/bucket/key"
			header := http.Header{}
			if failure == "listing_write_error" {
				mapping := namespace.Mapping{VisibleBucket: "bucket", Prefix: "tenant/"}
				upstream := &streamTestBody{reader: strings.NewReader("<ListBucketResult><Name>backend</Name><Prefix>tenant/</Prefix><IsTruncated>false</IsTruncated><Contents><Key>tenant/key</Key></Contents></ListBucketResult>")}
				data, err := mapping.Transform(context.Background(), upstream, "backend", url.Values{})
				if err != nil || upstream.closes != 1 {
					t.Fatalf("transform error = %v, upstream closes = %d", err, upstream.closes)
				}
				body.reader = bytes.NewReader(data)
				header.Set("Content-Length", fmt.Sprint(len(data)))
				path = "/bucket?list-type=2"
			}
			h := streamTestHandler(body, header, http.StatusOK, nil, &logs)
			w := &streamFailureWriter{ResponseRecorder: httptest.NewRecorder(), failure: failure, sentinel: sentinel}
			wantPanic := sentinel
			if failure == "write_error" || failure == "short_write" || failure == "listing_write_error" {
				wantPanic = http.ErrAbortHandler
			}
			func() {
				defer func() {
					if got := recover(); got != wantPanic {
						t.Errorf("panic = %v, want %v", got, wantPanic)
					}
				}()
				h.ServeHTTP(w, httptest.NewRequest(http.MethodGet, path, nil))
			}()
			if body.closes != 1 || strings.Count(logs.String(), "request complete") != 1 {
				t.Errorf("closes = %d, logs = %q", body.closes, logs.String())
			}
			if wantPanic == http.ErrAbortHandler && !strings.Contains(logs.String(), "response copy failed") {
				t.Errorf("missing write failure log: %q", logs.String())
			}
		})
	}
}

type streamPanicReader struct{ err error }

func (r streamPanicReader) Read([]byte) (int, error) { panic(r.err) }

type streamFailureWriter struct {
	*httptest.ResponseRecorder
	failure  string
	sentinel error
}

func (w *streamFailureWriter) WriteHeader(status int) {
	if w.failure == "header_panic" {
		panic(w.sentinel)
	}
	w.ResponseRecorder.WriteHeader(status)
}

func (w *streamFailureWriter) Write(p []byte) (int, error) {
	switch w.failure {
	case "write_panic":
		panic(w.sentinel)
	case "write_error", "listing_write_error":
		return 0, errors.New("downstream write failed")
	case "short_write":
		return len(p) - 1, nil
	default:
		return w.ResponseRecorder.Write(p)
	}
}

type streamTestBody struct {
	reader   io.Reader
	readErr  error
	closeErr error
	closes   int
}

func (b *streamTestBody) Read(p []byte) (int, error) {
	n, err := b.reader.Read(p)
	if err == io.EOF && b.readErr != nil {
		return n, b.readErr
	}
	return n, err
}

func (b *streamTestBody) Close() error {
	b.closes++
	if closer, ok := b.reader.(io.Closer); ok {
		return errors.Join(b.closeErr, closer.Close())
	}
	return b.closeErr
}

type flushEachWrite struct{ http.ResponseWriter }

func (w flushEachWrite) Write(p []byte) (int, error) {
	n, err := w.ResponseWriter.Write(p)
	w.ResponseWriter.(http.Flusher).Flush()
	return n, err
}
