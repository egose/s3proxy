package dispatch

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/egose/s3proxy/internal/backend/s3"
	"github.com/egose/s3proxy/internal/config"
	"github.com/egose/s3proxy/internal/replaybody"
	"github.com/egose/s3proxy/internal/rewrite"
	"github.com/egose/s3proxy/internal/router"
	"github.com/egose/s3proxy/internal/s3ops"
)

type errorReader struct{}

func (errorReader) Read([]byte) (int, error) { return 0, errors.New("boom") }

func (errorReader) Close() error { return nil }

func TestNewRequiresDependencies(t *testing.T) {
	if _, err := New(nil, replaybody.NewBudget(10, 10)); err == nil {
		t.Fatal("expected missing backend error")
	}
	if _, err := New(&stubBackend{}, nil); err == nil {
		t.Fatal("expected missing replay budget error")
	}
}

func TestDispatchObservesReplayBudgetConsumedByEarlierStage(t *testing.T) {
	budget := replaybody.NewBudget(10, 3)
	earlier, err := http.NewRequest(http.MethodPut, "http://proxy.local/bucket/key", nil)
	if err != nil {
		t.Fatal(err)
	}
	earlier.Body = io.NopCloser(strings.NewReader("abc"))
	earlier.ContentLength = -1
	if err := budget.Ensure(earlier); err != nil {
		t.Fatal(err)
	}
	defer replaybody.Release(earlier)

	d, err := New(&stubBackend{}, budget)
	if err != nil {
		t.Fatal(err)
	}
	req, err := http.NewRequest(http.MethodPut, "http://proxy.local/bucket/key", nil)
	if err != nil {
		t.Fatal(err)
	}
	req.Body = io.NopCloser(strings.NewReader("x"))
	req.ContentLength = -1

	_, err = d.Dispatch(context.Background(), router.Match{
		Route:        config.Route{Dispatch: config.DispatchAll, DestinationRefs: []string{"primary", "replica"}},
		Destinations: []string{"primary", "replica"},
	}, req, s3ops.OpPutObject, rewrite.Result{Bucket: "bucket", Key: "key"})
	if !replaybody.IsBudgetExhausted(err) {
		t.Fatalf("err = %v, want budget exhausted", err)
	}
}

func TestDispatch_FanoutBodyReadError(t *testing.T) {
	d := newTestDispatcher(&stubBackend{})
	req, err := http.NewRequest(http.MethodPut, "http://proxy.local/bucket/key", nil)
	if err != nil {
		t.Fatal(err)
	}
	req.Body = errorReader{}

	_, err = d.Dispatch(context.Background(), router.Match{
		Route: config.Route{
			Dispatch:        config.DispatchAll,
			DestinationRefs: []string{"primary", "replica"},
		},
		Destinations: []string{"primary", "replica"},
	}, req, s3ops.OpPutObject, rewrite.Result{Bucket: "bucket", Key: "key"})
	if err == nil {
		t.Fatal("expected read request body error")
	}
	if !strings.Contains(err.Error(), "read request body") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestDispatch_FanoutRejectsOversizedBody(t *testing.T) {
	d := newTestDispatcher(&stubBackend{})
	req, err := http.NewRequest(http.MethodPut, "http://proxy.local/bucket/key", io.NopCloser(strings.NewReader("x")))
	if err != nil {
		t.Fatal(err)
	}
	req.ContentLength = replaybody.DefaultMaxBytes + 1

	_, err = d.Dispatch(context.Background(), router.Match{
		Route: config.Route{
			Dispatch:        config.DispatchAll,
			DestinationRefs: []string{"primary", "replica"},
		},
		Destinations: []string{"primary", "replica"},
	}, req, s3ops.OpPutObject, rewrite.Result{Bucket: "bucket", Key: "key"})
	if !replaybody.IsTooLarge(err) {
		t.Fatalf("expected oversized replay error, got %v", err)
	}
}

func TestDispatch_FanoutReplicaFailureClearsSuccessfulPrimary(t *testing.T) {
	primaryBody := &trackingReadCloser{Reader: strings.NewReader("ok")}
	backend := &stubBackend{
		responsesByTarget: map[string]stubCall{
			"primary": {resp: responseWithStatusAndBody(http.StatusOK, primaryBody)},
			"replica": {err: errors.New("replica write failed")},
		},
	}
	d := newTestDispatcher(backend)
	req, err := http.NewRequest(http.MethodPut, "http://proxy.local/bucket/key", strings.NewReader("payload"))
	if err != nil {
		t.Fatal(err)
	}
	req.ContentLength = int64(len("payload"))

	result, err := d.Dispatch(context.Background(), router.Match{
		Route: config.Route{
			Dispatch:        config.DispatchAll,
			DestinationRefs: []string{"primary", "replica"},
		},
		Destinations: []string{"primary", "replica"},
	}, req, s3ops.OpPutObject, rewrite.Result{Bucket: "bucket", Key: "key"})
	if err == nil {
		t.Fatal("expected fan-out error")
	}
	if result == nil {
		t.Fatal("expected dispatch result")
	}
	if result.Primary != nil {
		t.Fatalf("expected primary response to be cleared on partial failure, got %#v", result.Primary)
	}
	if !primaryBody.closed {
		t.Fatal("expected successful primary body to be closed on partial failure")
	}
}

func TestDispatch_FanoutDrainsAndClosesExtraResponses(t *testing.T) {
	replicaBody := &trackingReadCloser{Reader: strings.NewReader("replica")}
	backend := &stubBackend{
		responsesByTarget: map[string]stubCall{
			"primary": {resp: responseWithStatus(http.StatusOK)},
			"replica": {resp: responseWithStatusAndBody(http.StatusOK, replicaBody)},
		},
	}
	d := newTestDispatcher(backend)
	req, err := http.NewRequest(http.MethodPut, "http://proxy.local/bucket/key", strings.NewReader("payload"))
	if err != nil {
		t.Fatal(err)
	}
	req.ContentLength = int64(len("payload"))

	result, err := d.Dispatch(context.Background(), router.Match{
		Route:        config.Route{Dispatch: config.DispatchAll, DestinationRefs: []string{"primary", "replica"}},
		Destinations: []string{"primary", "replica"},
	}, req, s3ops.OpPutObject, rewrite.Result{Bucket: "bucket", Key: "key"})
	if err != nil {
		t.Fatalf("Dispatch error = %v", err)
	}
	if !replicaBody.closed {
		t.Fatal("expected replica response body to be closed")
	}
	if got := replicaBody.Reader.(*strings.Reader).Len(); got != 0 {
		t.Fatalf("replica body remaining = %d, want drained", got)
	}
	s3.DrainAndClose(result.Primary)
}

func TestDispatch_DiscardDrainIsBounded(t *testing.T) {
	body := &trackingReadCloser{Reader: strings.NewReader(strings.Repeat("x", 2<<20))}
	backend := &stubBackend{
		responsesByTarget: map[string]stubCall{
			"primary": {resp: responseWithStatus(http.StatusOK)},
			"replica": {resp: responseWithStatusAndBody(http.StatusOK, body)},
		},
	}
	d := newTestDispatcher(backend)
	req, err := http.NewRequest(http.MethodPut, "http://proxy.local/bucket/key", strings.NewReader("payload"))
	if err != nil {
		t.Fatal(err)
	}
	req.ContentLength = int64(len("payload"))

	result, err := d.Dispatch(context.Background(), router.Match{
		Route:        config.Route{Dispatch: config.DispatchAll, DestinationRefs: []string{"primary", "replica"}},
		Destinations: []string{"primary", "replica"},
	}, req, s3ops.OpPutObject, rewrite.Result{Bucket: "bucket", Key: "key"})
	if err != nil {
		t.Fatalf("Dispatch error = %v", err)
	}
	if !body.closed {
		t.Fatal("expected oversized discarded body to be closed")
	}
	if got := body.Reader.(*strings.Reader).Len(); got == 0 {
		t.Fatal("expected discarded body drain to be bounded")
	}
	s3.DrainAndClose(result.Primary)
}

func TestDispatch_FanoutPrimaryFailurePreservesPrimaryError(t *testing.T) {
	backend := &stubBackend{
		responsesByTarget: map[string]stubCall{
			"primary": {resp: responseWithStatus(http.StatusForbidden)},
			"replica": {resp: responseWithStatus(http.StatusOK)},
		},
	}
	d := newTestDispatcher(backend)
	req, err := http.NewRequest(http.MethodPut, "http://proxy.local/bucket/key", strings.NewReader("payload"))
	if err != nil {
		t.Fatal(err)
	}
	req.ContentLength = int64(len("payload"))

	result, err := d.Dispatch(context.Background(), router.Match{
		Route: config.Route{
			Dispatch:        config.DispatchAll,
			DestinationRefs: []string{"primary", "replica"},
		},
		Destinations: []string{"primary", "replica"},
	}, req, s3ops.OpPutObject, rewrite.Result{Bucket: "bucket", Key: "key"})
	if err == nil {
		t.Fatal("expected fan-out error")
	}
	if result == nil || result.Primary == nil {
		t.Fatalf("expected primary error response, got %#v", result)
	}
	if got, want := result.Primary.StatusCode, http.StatusForbidden; got != want {
		t.Fatalf("StatusCode = %d, want %d", got, want)
	}
	result.Primary.Body.Close()
}

func TestDispatch_FanoutRetainsPrimaryBodyBeforeLaterDestinationDelay(t *testing.T) {
	bodyCtx, cancelBody := context.WithCancel(context.Background())
	primaryBody := &cancelSensitiveBody{ctx: bodyCtx, data: []byte("primary complete")}
	backend := &delayingFanoutBackend{
		first:  responseWithStatusAndBody(http.StatusOK, primaryBody),
		second: responseWithStatus(http.StatusOK),
		delay:  50 * time.Millisecond,
		cancel: cancelBody,
	}
	d := newTestDispatcher(backend)
	req, err := http.NewRequest(http.MethodPut, "http://proxy.local/bucket/key", strings.NewReader("payload"))
	if err != nil {
		t.Fatal(err)
	}
	req.ContentLength = int64(len("payload"))

	result, err := d.Dispatch(context.Background(), router.Match{
		Route:        config.Route{Dispatch: config.DispatchAll, DestinationRefs: []string{"primary", "replica"}},
		Destinations: []string{"primary", "replica"},
	}, req, s3ops.OpPutObject, rewrite.Result{Bucket: "bucket", Key: "key"})
	if err != nil {
		t.Fatalf("Dispatch error = %v", err)
	}
	if !primaryBody.closed {
		t.Fatal("expected original primary body to be closed after retention")
	}
	body, err := io.ReadAll(result.Primary.Body)
	if err != nil {
		t.Fatalf("read retained primary body: %v", err)
	}
	if got, want := string(body), "primary complete"; got != want {
		t.Fatalf("body = %q, want %q", got, want)
	}
}

func TestDispatch_FanoutRetainsPrimaryErrorBodyAfterLaterDestinationDelay(t *testing.T) {
	bodyCtx, cancelBody := context.WithCancel(context.Background())
	primaryBody := &cancelSensitiveBody{ctx: bodyCtx, data: []byte("<Error>denied</Error>")}
	backend := &delayingFanoutBackend{
		first:  responseWithStatusAndBody(http.StatusForbidden, primaryBody),
		second: responseWithStatus(http.StatusOK),
		delay:  50 * time.Millisecond,
		cancel: cancelBody,
	}
	d := newTestDispatcher(backend)
	req, err := http.NewRequest(http.MethodPut, "http://proxy.local/bucket/key", strings.NewReader("payload"))
	if err != nil {
		t.Fatal(err)
	}
	req.ContentLength = int64(len("payload"))

	result, err := d.Dispatch(context.Background(), router.Match{
		Route:        config.Route{Dispatch: config.DispatchAll, DestinationRefs: []string{"primary", "replica"}},
		Destinations: []string{"primary", "replica"},
	}, req, s3ops.OpPutObject, rewrite.Result{Bucket: "bucket", Key: "key"})
	if err == nil {
		t.Fatal("expected fan-out error")
	}
	body, readErr := io.ReadAll(result.Primary.Body)
	if readErr != nil {
		t.Fatalf("read retained primary error body: %v", readErr)
	}
	if got, want := string(body), "<Error>denied</Error>"; got != want {
		t.Fatalf("body = %q, want %q", got, want)
	}
}

func TestDispatch_FanoutPrimaryErrorBodyClosesAfterLaterFailure(t *testing.T) {
	primaryBody := &trackingReadCloser{Reader: strings.NewReader("<Error>denied</Error>")}
	backend := &stubBackend{
		responsesByTarget: map[string]stubCall{
			"primary": {resp: responseWithStatusAndBody(http.StatusForbidden, primaryBody)},
			"replica": {err: errors.New("replica failed")},
		},
	}
	d := newTestDispatcher(backend)
	req, err := http.NewRequest(http.MethodPut, "http://proxy.local/bucket/key", strings.NewReader("payload"))
	if err != nil {
		t.Fatal(err)
	}
	req.ContentLength = int64(len("payload"))

	result, err := d.Dispatch(context.Background(), router.Match{
		Route:        config.Route{Dispatch: config.DispatchAll, DestinationRefs: []string{"primary", "replica"}},
		Destinations: []string{"primary", "replica"},
	}, req, s3ops.OpPutObject, rewrite.Result{Bucket: "bucket", Key: "key"})
	if err == nil {
		t.Fatal("expected fan-out error")
	}
	if !primaryBody.closed {
		t.Fatal("expected original primary error body to be closed after retention")
	}
	result.Primary.Body.Close()
}

func TestDispatch_FanoutDelayedDestinationsRunInParallel(t *testing.T) {
	delay := 100 * time.Millisecond
	backend := &boundedFanoutBackend{delay: delay}
	d := newTestDispatcher(backend)
	req := fanoutRequest(t, "payload")
	defer replaybody.Release(req)

	started := time.Now()
	result, err := d.Dispatch(context.Background(), fanoutMatch("primary", "replica"), req, s3ops.OpPutObject, rewrite.Result{Bucket: "bucket", Key: "key"})
	elapsed := time.Since(started)
	if err != nil {
		t.Fatalf("Dispatch error = %v", err)
	}
	if elapsed >= delay+delay/2 {
		t.Fatalf("elapsed = %s, want near one %s backend interval", elapsed, delay)
	}
	if got := backend.maxActive.Load(); got != 2 {
		t.Fatalf("max active attempts = %d, want 2", got)
	}
	s3.DrainAndClose(result.Primary)
}

func TestDispatch_FanoutActiveAttemptsNeverExceedBound(t *testing.T) {
	backend := &boundedFanoutBackend{block: make(chan struct{})}
	d := newTestDispatcher(backend)
	req := fanoutRequest(t, "payload")
	defer replaybody.Release(req)
	destinations := []string{"target-0", "target-1", "target-2", "target-3", "target-4", "target-5", "target-6", "target-7"}
	done := make(chan error, 1)

	go func() {
		result, err := d.Dispatch(context.Background(), fanoutMatch(destinations...), req, s3ops.OpPutObject, rewrite.Result{Bucket: "bucket", Key: "key"})
		if result != nil && result.Primary != nil {
			s3.DrainAndClose(result.Primary)
		}
		done <- err
	}()

	waitForAtomic(t, &backend.active, maxParallelWriteFanout)
	time.Sleep(30 * time.Millisecond)
	if got := backend.maxActive.Load(); got > maxParallelWriteFanout {
		t.Fatalf("max active attempts = %d, want <= %d", got, maxParallelWriteFanout)
	}
	close(backend.block)
	if err := <-done; err != nil {
		t.Fatalf("Dispatch error = %v", err)
	}
	if got := backend.calls.Load(); got != int64(len(destinations)) {
		t.Fatalf("calls = %d, want %d", got, len(destinations))
	}
}

func TestDispatch_FanoutPreservesOrderAndPrimaryUnderReversedCompletion(t *testing.T) {
	primaryBody := &trackingReadCloser{Reader: strings.NewReader("<Error>denied</Error>")}
	replicaBody := &trackingReadCloser{Reader: strings.NewReader("ok")}
	backend := &targetDelayBackend{
		responses: map[string]*s3.Response{
			"primary": responseWithStatusAndBody(http.StatusForbidden, primaryBody),
			"replica": responseWithStatusAndBody(http.StatusOK, replicaBody),
		},
		delays: map[string]time.Duration{"primary": 75 * time.Millisecond},
	}
	d := newTestDispatcher(backend)
	req := fanoutRequest(t, "payload")
	defer replaybody.Release(req)

	result, err := d.Dispatch(context.Background(), fanoutMatch("primary", "replica"), req, s3ops.OpPutObject, rewrite.Result{Bucket: "bucket", Key: "key"})
	if err == nil {
		t.Fatal("expected fan-out error")
	}
	if got, want := len(result.Attempts), 2; got != want {
		t.Fatalf("attempts = %d, want %d", got, want)
	}
	if result.Attempts[0].Target != "primary" || result.Attempts[0].StatusCode != http.StatusForbidden || result.Attempts[0].Error == nil {
		t.Fatalf("first attempt = %#v, want primary 403 failure", result.Attempts[0])
	}
	if result.Attempts[1].Target != "replica" || result.Attempts[1].StatusCode != http.StatusOK || result.Attempts[1].Error != nil {
		t.Fatalf("second attempt = %#v, want replica 200 success", result.Attempts[1])
	}
	if result.Primary == nil || result.Primary.StatusCode != http.StatusForbidden {
		t.Fatalf("primary = %#v, want primary 403 response", result.Primary)
	}
	body, readErr := io.ReadAll(result.Primary.Body)
	if readErr != nil {
		t.Fatalf("read primary body: %v", readErr)
	}
	if got, want := string(body), "<Error>denied</Error>"; got != want {
		t.Fatalf("body = %q, want %q", got, want)
	}
	if !primaryBody.closed || !replicaBody.closed {
		t.Fatalf("closed primary=%v replica=%v, want both original responses closed", primaryBody.closed, replicaBody.closed)
	}
}

func TestDispatch_FanoutUsesIndependentReplayReaders(t *testing.T) {
	backend := &independentReaderBackend{reads: make(chan string, 2)}
	d := newTestDispatcher(backend)
	req := fanoutRequest(t, "payload")
	defer replaybody.Release(req)

	result, err := d.Dispatch(context.Background(), fanoutMatch("primary", "replica"), req, s3ops.OpPutObject, rewrite.Result{Bucket: "bucket", Key: "key"})
	if err != nil {
		t.Fatalf("Dispatch error = %v", err)
	}
	s3.DrainAndClose(result.Primary)
	seen := map[string]bool{}
	for i := 0; i < 2; i++ {
		seen[<-backend.reads] = true
	}
	if !seen["primary:payload"] || !seen["replica:payload"] {
		t.Fatalf("reads = %#v, want each target to read full payload", seen)
	}
	if backend.sharedBody.Load() {
		t.Fatal("attempts shared a request body")
	}
}

func TestDispatch_FanoutParentCancellationTerminatesActiveAttempts(t *testing.T) {
	backend := &cancelTrackingBackend{done: make(chan struct{}, maxParallelWriteFanout)}
	d := newTestDispatcher(backend)
	ctx, cancel := context.WithCancel(context.Background())
	req, err := http.NewRequestWithContext(ctx, http.MethodPut, "http://proxy.local/bucket/key", strings.NewReader("payload"))
	if err != nil {
		t.Fatal(err)
	}
	req.ContentLength = int64(len("payload"))
	budget := d.replayBudget
	destinations := []string{"target-0", "target-1", "target-2", "target-3", "target-4", "target-5"}
	done := make(chan error, 1)

	go func() {
		result, err := d.Dispatch(ctx, fanoutMatch(destinations...), req, s3ops.OpPutObject, rewrite.Result{Bucket: "bucket", Key: "key"})
		if result != nil && result.Primary != nil {
			s3.DrainAndClose(result.Primary)
		}
		done <- err
	}()

	waitForAtomic(t, &backend.active, maxParallelWriteFanout)
	cancel()
	for i := 0; i < maxParallelWriteFanout; i++ {
		select {
		case <-backend.done:
		case <-time.After(time.Second):
			t.Fatal("timed out waiting for active attempt to exit")
		}
	}
	if err := <-done; !errors.Is(err, context.Canceled) {
		t.Fatalf("error = %v, want context canceled", err)
	}
	if got := backend.active.Load(); got != 0 {
		t.Fatalf("active attempts = %d, want 0", got)
	}
	if got := backend.maxActive.Load(); got > maxParallelWriteFanout {
		t.Fatalf("max active attempts = %d, want <= %d", got, maxParallelWriteFanout)
	}
	if err := replaybody.Release(req); err != nil {
		t.Fatal(err)
	}
	if got := budget.Used(); got != 0 {
		t.Fatalf("replay budget used = %d, want 0", got)
	}
}

func TestDispatch_SingleTargetFailureReturnsAttemptMetadata(t *testing.T) {
	sentinel := errors.New("dial tcp timeout")
	backend := &stubBackend{responses: []stubCall{{err: sentinel}}}
	d := newTestDispatcher(backend)
	req := mustRequest(t)

	result, err := d.Dispatch(context.Background(), router.Match{
		Route:        config.Route{Dispatch: config.DispatchFirst, DestinationRefs: []string{"primary"}},
		Destinations: []string{"primary"},
	}, req, s3ops.OpGetObject, rewrite.Result{Bucket: "bucket", Key: "key"})
	if !errors.Is(err, sentinel) {
		t.Fatalf("error = %v, want sentinel", err)
	}
	if result == nil {
		t.Fatal("expected result with attempt metadata")
	}
	if got, want := len(result.Attempts), 1; got != want {
		t.Fatalf("attempts = %d, want %d", got, want)
	}
	if result.Attempts[0].Target != "primary" || !errors.Is(result.Attempts[0].Error, sentinel) {
		t.Fatalf("attempt = %#v, want primary sentinel error", result.Attempts[0])
	}
	if got, want := result.FailureCount(), 1; got != want {
		t.Fatalf("FailureCount = %d, want %d", got, want)
	}
}

func TestResultFailureCountDerivedFromAttempts(t *testing.T) {
	result := &Result{Attempts: []Attempt{
		{Target: "primary", Error: errors.New("first failed")},
		{Target: "primary", StatusCode: http.StatusOK},
		{Target: "replica", Error: errors.New("second failed")},
	}}

	if got, want := result.FailureCount(), 2; got != want {
		t.Fatalf("FailureCount = %d, want %d", got, want)
	}
}

func TestDispatch_OrderedFailoverRetriesTransportError(t *testing.T) {
	backend := &stubBackend{
		responses: []stubCall{
			{err: errors.New("dial tcp timeout")},
			{resp: responseWithStatus(http.StatusOK)},
		},
	}
	d := newTestDispatcher(backend)
	req := mustRequest(t)

	result, err := d.Dispatch(context.Background(), orderedFailoverMatch(), req, s3ops.OpGetObject, rewrite.Result{Bucket: "bucket", Key: "key"})
	if err != nil {
		t.Fatalf("Dispatch error = %v", err)
	}
	if result.Primary == nil || result.Primary.StatusCode != http.StatusOK {
		t.Fatalf("Primary = %#v, want 200 response", result.Primary)
	}
	if got := backend.calls; got != 2 {
		t.Fatalf("calls = %d, want 2", got)
	}
	if got, want := result.FailureCount(), 1; got != want {
		t.Fatalf("FailureCount = %d, want %d", got, want)
	}
	if got, want := len(result.Attempts), 2; got != want {
		t.Fatalf("attempts = %d, want %d", got, want)
	}
	if result.Attempts[0].Target != "primary" || result.Attempts[0].Error == nil {
		t.Fatalf("first attempt = %#v, want primary error", result.Attempts[0])
	}
	if result.Attempts[1].Target != "replica" || result.Attempts[1].StatusCode != http.StatusOK || result.Attempts[1].Error != nil {
		t.Fatalf("second attempt = %#v, want replica 200", result.Attempts[1])
	}
	result.Primary.Body.Close()
}

func TestDispatch_OrderedFailoverRetriesUpstream5xx(t *testing.T) {
	failedBody := &trackingReadCloser{Reader: strings.NewReader("bad gateway")}
	backend := &stubBackend{
		responses: []stubCall{
			{resp: responseWithStatusAndBody(http.StatusBadGateway, failedBody)},
			{resp: responseWithStatus(http.StatusOK)},
		},
	}
	d := newTestDispatcher(backend)
	req := mustRequest(t)

	result, err := d.Dispatch(context.Background(), orderedFailoverMatch(), req, s3ops.OpGetObject, rewrite.Result{Bucket: "bucket", Key: "key"})
	if err != nil {
		t.Fatalf("Dispatch error = %v", err)
	}
	if result.Primary == nil || result.Primary.StatusCode != http.StatusOK {
		t.Fatalf("Primary = %#v, want 200 response", result.Primary)
	}
	if got := backend.calls; got != 2 {
		t.Fatalf("calls = %d, want 2", got)
	}
	if got, want := result.FailureCount(), 1; got != want {
		t.Fatalf("FailureCount = %d, want %d", got, want)
	}
	if got, want := len(result.Attempts), 2; got != want {
		t.Fatalf("attempts = %d, want %d", got, want)
	}
	if result.Attempts[0].Target != "primary" || result.Attempts[0].StatusCode != http.StatusBadGateway || result.Attempts[0].Error == nil {
		t.Fatalf("first attempt = %#v, want primary 502 error", result.Attempts[0])
	}
	if result.Attempts[1].Target != "replica" || result.Attempts[1].StatusCode != http.StatusOK || result.Attempts[1].Error != nil {
		t.Fatalf("second attempt = %#v, want replica 200", result.Attempts[1])
	}
	if !failedBody.closed {
		t.Fatal("expected failed failover response body to be closed before retry")
	}
	if got := failedBody.Reader.(*strings.Reader).Len(); got != 0 {
		t.Fatalf("failed body remaining = %d, want drained", got)
	}
	result.Primary.Body.Close()
}

func TestDispatch_DiscardedResponsesReuseConnections(t *testing.T) {
	var conns atomic.Int64
	server := httptest.NewUnstartedServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("small discarded body"))
	}))
	server.Config.ConnState = func(_ net.Conn, state http.ConnState) {
		if state == http.StateNew {
			conns.Add(1)
		}
	}
	server.Start()
	defer server.Close()

	backend := httpBackend{client: server.Client(), url: server.URL}
	d := newTestDispatcher(backend)
	for i := 0; i < 3; i++ {
		req, err := http.NewRequest(http.MethodPut, "http://proxy.local/bucket/key", strings.NewReader("payload"))
		if err != nil {
			t.Fatal(err)
		}
		req.ContentLength = int64(len("payload"))
		result, err := d.Dispatch(context.Background(), router.Match{
			Route:        config.Route{Dispatch: config.DispatchAll, DestinationRefs: []string{"primary", "replica"}},
			Destinations: []string{"primary", "replica"},
		}, req, s3ops.OpPutObject, rewrite.Result{Bucket: "bucket", Key: "key"})
		if err != nil {
			t.Fatalf("Dispatch error = %v", err)
		}
		if _, err := io.Copy(io.Discard, result.Primary.Body); err != nil {
			t.Fatal(err)
		}
		result.Primary.Body.Close()
	}
	if got := conns.Load(); got > 2 {
		t.Fatalf("connections = %d, want discarded responses to reuse the initial connections", got)
	}
}

func TestDispatch_OrderedFailoverDoesNotRetry404(t *testing.T) {
	backend := &stubBackend{
		responses: []stubCall{{resp: responseWithStatus(http.StatusNotFound)}},
	}
	d := newTestDispatcher(backend)
	req := mustRequest(t)

	result, err := d.Dispatch(context.Background(), orderedFailoverMatch(), req, s3ops.OpGetObject, rewrite.Result{Bucket: "bucket", Key: "key"})
	if err != nil {
		t.Fatalf("Dispatch error = %v", err)
	}
	if result.Primary == nil || result.Primary.StatusCode != http.StatusNotFound {
		t.Fatalf("Primary = %#v, want 404 response", result.Primary)
	}
	if got := backend.calls; got != 1 {
		t.Fatalf("calls = %d, want 1", got)
	}
	if got := result.FailureCount(); got != 0 {
		t.Fatalf("FailureCount = %d, want 0", got)
	}
	result.Primary.Body.Close()
}

func TestDispatch_OrderedFailoverReturnsLast5xxWhenExhausted(t *testing.T) {
	backend := &stubBackend{
		responses: []stubCall{
			{resp: responseWithStatus(http.StatusBadGateway)},
			{resp: responseWithStatus(http.StatusServiceUnavailable)},
		},
	}
	d := newTestDispatcher(backend)
	req := mustRequest(t)

	result, err := d.Dispatch(context.Background(), orderedFailoverMatch(), req, s3ops.OpGetObject, rewrite.Result{Bucket: "bucket", Key: "key"})
	if err == nil {
		t.Fatal("expected ordered failover to report exhaustion")
	}
	if result == nil || result.Primary == nil {
		t.Fatalf("expected last upstream response, got %#v", result)
	}
	if got, want := result.Primary.StatusCode, http.StatusServiceUnavailable; got != want {
		t.Fatalf("StatusCode = %d, want %d", got, want)
	}
	if got := backend.calls; got != 2 {
		t.Fatalf("calls = %d, want 2", got)
	}
	result.Primary.Body.Close()
}

func TestDispatch_OrderedFailoverReportsTransportAttemptsWhenExhausted(t *testing.T) {
	backend := &stubBackend{
		responses: []stubCall{
			{err: errors.New("primary timeout")},
			{err: errors.New("replica timeout")},
		},
	}
	d := newTestDispatcher(backend)
	req := mustRequest(t)

	result, err := d.Dispatch(context.Background(), orderedFailoverMatch(), req, s3ops.OpGetObject, rewrite.Result{Bucket: "bucket", Key: "key"})
	if err == nil {
		t.Fatal("expected ordered failover to report exhaustion")
	}
	if result == nil {
		t.Fatal("expected result with attempt metadata")
	}
	if got, want := len(result.Attempts), 2; got != want {
		t.Fatalf("attempts = %d, want %d", got, want)
	}
	for i, target := range []string{"primary", "replica"} {
		if result.Attempts[i].Target != target || result.Attempts[i].Error == nil {
			t.Fatalf("attempt %d = %#v, want %s error", i, result.Attempts[i], target)
		}
	}
}

func mustRequest(t *testing.T) *http.Request {
	t.Helper()
	req, err := http.NewRequest(http.MethodGet, "http://proxy.local/bucket/key", nil)
	if err != nil {
		t.Fatal(err)
	}
	return req
}

func orderedFailoverMatch() router.Match {
	return router.Match{
		Route:        config.Route{ReadPreference: config.ReadOrderedFailover},
		Destinations: []string{"primary", "replica"},
	}
}

func fanoutMatch(destinations ...string) router.Match {
	refs := append([]string(nil), destinations...)
	return router.Match{
		Route:        config.Route{Dispatch: config.DispatchAll, DestinationRefs: refs},
		Destinations: append([]string(nil), destinations...),
	}
}

func fanoutRequest(t *testing.T, body string) *http.Request {
	t.Helper()
	req, err := http.NewRequest(http.MethodPut, "http://proxy.local/bucket/key", strings.NewReader(body))
	if err != nil {
		t.Fatal(err)
	}
	req.ContentLength = int64(len(body))
	req.GetBody = nil
	return req
}

func waitForAtomic(t *testing.T, value *atomic.Int64, want int) {
	t.Helper()
	deadline := time.Now().Add(time.Second)
	for time.Now().Before(deadline) {
		if value.Load() == int64(want) {
			return
		}
		time.Sleep(time.Millisecond)
	}
	t.Fatalf("value = %d, want %d", value.Load(), want)
}

func newTestDispatcher(backend s3.Executor) *dispatcher {
	return &dispatcher{backend: backend, replayBudget: replaybody.NewBudget(replaybody.DefaultMaxBytes, replaybody.DefaultAggregateMaxBytes)}
}

func responseWithStatus(status int) *s3.Response {
	return responseWithStatusAndBody(status, io.NopCloser(strings.NewReader("body")))
}

func responseWithStatusAndBody(status int, body io.ReadCloser) *s3.Response {
	return &s3.Response{
		StatusCode: status,
		Header:     http.Header{},
		Body:       body,
	}
}

type trackingReadCloser struct {
	io.Reader
	closed bool
}

func (r *trackingReadCloser) Close() error {
	r.closed = true
	return nil
}

type httpBackend struct {
	client *http.Client
	url    string
}

func (b httpBackend) Do(ctx context.Context, req s3.Request) (*s3.Response, error) {
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodGet, b.url, nil)
	if err != nil {
		return nil, err
	}
	resp, err := b.client.Do(httpReq)
	if err != nil {
		return nil, err
	}
	return &s3.Response{StatusCode: resp.StatusCode, Header: resp.Header, Body: resp.Body}, nil
}

type stubBackend struct {
	responses         []stubCall
	responsesByTarget map[string]stubCall
	calls             int
	mu                sync.Mutex
}

type cancelSensitiveBody struct {
	ctx    context.Context
	data   []byte
	closed bool
	mu     sync.Mutex
}

func (b *cancelSensitiveBody) Read(p []byte) (int, error) {
	if err := b.ctx.Err(); err != nil {
		return 0, err
	}
	b.mu.Lock()
	defer b.mu.Unlock()
	if len(b.data) == 0 {
		return 0, io.EOF
	}
	n := copy(p, b.data)
	b.data = b.data[n:]
	return n, nil
}

func (b *cancelSensitiveBody) Close() error {
	b.mu.Lock()
	b.closed = true
	b.mu.Unlock()
	return nil
}

type delayingFanoutBackend struct {
	first  *s3.Response
	second *s3.Response
	delay  time.Duration
	cancel context.CancelFunc
	calls  atomic.Int64
}

func (b *delayingFanoutBackend) Do(_ context.Context, req s3.Request) (*s3.Response, error) {
	b.calls.Add(1)
	if req.Target == "primary" {
		return b.first, nil
	}
	time.Sleep(b.delay)
	b.cancel()
	return b.second, nil
}

type stubCall struct {
	resp *s3.Response
	err  error
}

func (s *stubBackend) Do(_ context.Context, req s3.Request) (*s3.Response, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.responsesByTarget != nil {
		call, ok := s.responsesByTarget[req.Target]
		if !ok {
			return nil, fmt.Errorf("unexpected backend target %q", req.Target)
		}
		s.calls++
		return call.resp, call.err
	}
	if s.calls >= len(s.responses) {
		return nil, errors.New("unexpected backend call")
	}
	call := s.responses[s.calls]
	s.calls++
	return call.resp, call.err
}

type boundedFanoutBackend struct {
	delay     time.Duration
	block     chan struct{}
	calls     atomic.Int64
	active    atomic.Int64
	maxActive atomic.Int64
}

func (b *boundedFanoutBackend) Do(ctx context.Context, req s3.Request) (*s3.Response, error) {
	b.calls.Add(1)
	active := b.active.Add(1)
	for {
		maxActive := b.maxActive.Load()
		if active <= maxActive || b.maxActive.CompareAndSwap(maxActive, active) {
			break
		}
	}
	defer b.active.Add(-1)
	if b.block != nil {
		select {
		case <-b.block:
		case <-ctx.Done():
			return nil, ctx.Err()
		}
	} else if b.delay > 0 {
		select {
		case <-time.After(b.delay):
		case <-ctx.Done():
			return nil, ctx.Err()
		}
	}
	return responseWithStatus(http.StatusOK), nil
}

type targetDelayBackend struct {
	responses map[string]*s3.Response
	delays    map[string]time.Duration
}

func (b *targetDelayBackend) Do(ctx context.Context, req s3.Request) (*s3.Response, error) {
	if delay := b.delays[req.Target]; delay > 0 {
		select {
		case <-time.After(delay):
		case <-ctx.Done():
			return nil, ctx.Err()
		}
	}
	resp := b.responses[req.Target]
	if resp == nil {
		return nil, fmt.Errorf("unexpected backend target %q", req.Target)
	}
	return resp, nil
}

type independentReaderBackend struct {
	reads      chan string
	firstBody  atomic.Pointer[io.ReadCloser]
	sharedBody atomic.Bool
}

func (b *independentReaderBackend) Do(_ context.Context, req s3.Request) (*s3.Response, error) {
	body := req.Source.Body
	if body == nil {
		return nil, errors.New("missing request body")
	}
	if b.firstBody.CompareAndSwap(nil, &body) {
	} else if *b.firstBody.Load() == body {
		b.sharedBody.Store(true)
	}
	payload, err := io.ReadAll(body)
	if err != nil {
		return nil, err
	}
	b.reads <- req.Target + ":" + string(payload)
	return responseWithStatus(http.StatusOK), nil
}

type cancelTrackingBackend struct {
	done      chan struct{}
	active    atomic.Int64
	maxActive atomic.Int64
}

func (b *cancelTrackingBackend) Do(ctx context.Context, _ s3.Request) (*s3.Response, error) {
	active := b.active.Add(1)
	for {
		maxActive := b.maxActive.Load()
		if active <= maxActive || b.maxActive.CompareAndSwap(maxActive, active) {
			break
		}
	}
	defer func() {
		b.active.Add(-1)
		b.done <- struct{}{}
	}()
	<-ctx.Done()
	return nil, ctx.Err()
}
