package replaybody

import (
	"errors"
	"io"
	"net/http"
	"strings"
	"sync"
	"testing"
)

type closeTrackingBody struct {
	*strings.Reader
	closed bool
	err    error
}

func (b *closeTrackingBody) Close() error {
	b.closed = true
	return b.err
}

type failingReader struct{}

func (failingReader) Read([]byte) (int, error) { return 0, errors.New("boom") }
func (failingReader) Close() error             { return nil }

func TestEnsureNilNoBodyAndExistingGetBodyAreNoops(t *testing.T) {
	budget := NewBudget(10, 10)
	req, err := http.NewRequest(http.MethodPut, "http://proxy.local/bucket/key", nil)
	if err != nil {
		t.Fatal(err)
	}
	if err := budget.Ensure(req); err != nil {
		t.Fatal(err)
	}
	if budget.Used() != 0 {
		t.Fatalf("used = %d, want 0", budget.Used())
	}

	req.Body = io.NopCloser(strings.NewReader("payload"))
	req.GetBody = func() (io.ReadCloser, error) { return io.NopCloser(strings.NewReader("payload")), nil }
	if err := budget.Ensure(req); err != nil {
		t.Fatal(err)
	}
	if budget.Used() != 0 {
		t.Fatalf("used = %d, want 0", budget.Used())
	}
}

func TestEnsureRejectsKnownOversizeWithoutReading(t *testing.T) {
	budget := NewBudget(3, 10)
	body := &closeTrackingBody{Reader: strings.NewReader("payload")}
	req, err := http.NewRequest(http.MethodPut, "http://proxy.local/bucket/key", body)
	if err != nil {
		t.Fatal(err)
	}
	req.ContentLength = 4

	err = budget.Ensure(req)
	if !IsTooLarge(err) {
		t.Fatalf("err = %v, want too large", err)
	}
	if body.closed {
		t.Fatal("body should remain caller-owned when known length is rejected before reading")
	}
	if budget.Used() != 0 {
		t.Fatalf("used = %d, want 0", budget.Used())
	}
}

func TestEnsureRejectsUnknownLengthOverflowAndReleases(t *testing.T) {
	budget := NewBudget(3, 10)
	body := &closeTrackingBody{Reader: strings.NewReader("payload")}
	req, err := http.NewRequest(http.MethodPut, "http://proxy.local/bucket/key", body)
	if err != nil {
		t.Fatal(err)
	}
	req.ContentLength = -1

	err = budget.Ensure(req)
	if !IsTooLarge(err) {
		t.Fatalf("err = %v, want too large", err)
	}
	if budget.Used() != 0 {
		t.Fatalf("used = %d, want 0", budget.Used())
	}
}

func TestEnsureReadFailureReleases(t *testing.T) {
	budget := NewBudget(10, 10)
	req, err := http.NewRequest(http.MethodPut, "http://proxy.local/bucket/key", failingReader{})
	if err != nil {
		t.Fatal(err)
	}

	err = budget.Ensure(req)
	if err == nil || !strings.Contains(err.Error(), "read request body") {
		t.Fatalf("err = %v, want read failure", err)
	}
	if budget.Used() != 0 {
		t.Fatalf("used = %d, want 0", budget.Used())
	}
}

func TestEnsureBudgetExhaustionReleasesPartialReservation(t *testing.T) {
	budget := NewBudget(10, 3)
	req, err := http.NewRequest(http.MethodPut, "http://proxy.local/bucket/key", &closeTrackingBody{Reader: strings.NewReader("payload")})
	if err != nil {
		t.Fatal(err)
	}

	err = budget.Ensure(req)
	if !IsBudgetExhausted(err) {
		t.Fatalf("err = %v, want budget exhausted", err)
	}
	if budget.Used() != 0 {
		t.Fatalf("used = %d, want 0", budget.Used())
	}
}

func TestEnsureInstallsReplayBodyClosesOriginalAndIsIdempotent(t *testing.T) {
	budget := NewBudget(10, 10)
	original := &closeTrackingBody{Reader: strings.NewReader("payload")}
	req, err := http.NewRequest(http.MethodPut, "http://proxy.local/bucket/key", original)
	if err != nil {
		t.Fatal(err)
	}

	if err := budget.Ensure(req); err != nil {
		t.Fatal(err)
	}
	if !original.closed {
		t.Fatal("original body was not closed after replay ownership transfer")
	}
	if got, want := req.ContentLength, int64(len("payload")); got != want {
		t.Fatalf("ContentLength = %d, want %d", got, want)
	}
	if budget.Used() != int64(len("payload")) {
		t.Fatalf("used = %d, want %d", budget.Used(), len("payload"))
	}
	if err := budget.Ensure(req); err != nil {
		t.Fatal(err)
	}
	if budget.Used() != int64(len("payload")) {
		t.Fatalf("used after idempotent Ensure = %d, want %d", budget.Used(), len("payload"))
	}
}

func TestResetAndGetBodyError(t *testing.T) {
	req, err := http.NewRequest(http.MethodPut, "http://proxy.local/bucket/key", nil)
	if err != nil {
		t.Fatal(err)
	}
	req.GetBody = func() (io.ReadCloser, error) { return nil, errors.New("boom") }

	err = Reset(req)
	if err == nil || !strings.Contains(err.Error(), "get request body") {
		t.Fatalf("err = %v, want get body failure", err)
	}

	req.GetBody = func() (io.ReadCloser, error) { return io.NopCloser(strings.NewReader("payload")), nil }
	if err := Reset(req); err != nil {
		t.Fatal(err)
	}
	data, err := io.ReadAll(req.Body)
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != "payload" {
		t.Fatalf("body = %q, want payload", string(data))
	}
}

func TestReleaseReturnsBudgetAndClearsReplay(t *testing.T) {
	budget := NewBudget(10, 10)
	req, err := http.NewRequest(http.MethodPut, "http://proxy.local/bucket/key", strings.NewReader("payload"))
	if err != nil {
		t.Fatal(err)
	}
	if err := budget.Ensure(req); err != nil {
		t.Fatal(err)
	}
	if err := Release(req); err != nil {
		t.Fatal(err)
	}
	if budget.Used() != 0 {
		t.Fatalf("used = %d, want 0", budget.Used())
	}
	if req.GetBody != nil {
		t.Fatal("GetBody was not cleared")
	}
}

func TestConcurrentReservationsCannotExceedAggregateBudget(t *testing.T) {
	budget := NewBudget(4, 8)
	var wg sync.WaitGroup
	results := make(chan error, 3)
	for i := 0; i < 3; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			req, err := http.NewRequest(http.MethodPut, "http://proxy.local/bucket/key", &closeTrackingBody{Reader: strings.NewReader("data")})
			if err != nil {
				results <- err
				return
			}
			results <- budget.Ensure(req)
		}()
	}
	wg.Wait()
	close(results)

	successes := 0
	exhausted := 0
	for err := range results {
		if err == nil {
			successes++
			continue
		}
		if IsBudgetExhausted(err) {
			exhausted++
			continue
		}
		t.Fatalf("unexpected err: %v", err)
	}
	if successes != 2 || exhausted != 1 {
		t.Fatalf("successes=%d exhausted=%d, want 2 and 1", successes, exhausted)
	}
	if budget.Used() != 8 {
		t.Fatalf("used = %d, want 8", budget.Used())
	}
}

func TestDefaultLimit(t *testing.T) {
	budget := NewBudget(0, 0)
	if budget.maxRequestBytes != DefaultMaxBytes {
		t.Fatalf("maxRequestBytes = %d, want %d", budget.maxRequestBytes, DefaultMaxBytes)
	}
}
