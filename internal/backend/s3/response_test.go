package s3

import (
	"context"
	"errors"
	"io"
	"net/http"
	"strings"
	"sync"
	"testing"
	"time"
)

func TestDrainAndClose_BoundsNeverProducingBody(t *testing.T) {
	body := newBlockingBody()
	start := time.Now()
	err := DrainAndClose(&Response{StatusCode: http.StatusOK, Header: http.Header{}, Body: body})
	elapsed := time.Since(start)

	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("error = %v, want deadline exceeded", err)
	}
	if elapsed > responseDiscardTimeout+200*time.Millisecond {
		t.Fatalf("cleanup took %s, want bounded by discard timeout", elapsed)
	}
	if !body.closed() {
		t.Fatal("expected body to be closed")
	}
}

func TestDrainAndClose_ParentCancellationEndsCleanup(t *testing.T) {
	body := newBlockingBody()
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	start := time.Now()
	err := DrainAndCloseContext(ctx, &Response{StatusCode: http.StatusOK, Header: http.Header{}, Body: body})
	elapsed := time.Since(start)

	if !errors.Is(err, context.Canceled) {
		t.Fatalf("error = %v, want context canceled", err)
	}
	if elapsed > 100*time.Millisecond {
		t.Fatalf("cleanup took %s, want prompt cancellation", elapsed)
	}
	if !body.closed() {
		t.Fatal("expected body to be closed")
	}
}

func TestBufferWriteResponse_DetachesSmallResponseFromParentCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	original := &trackingBody{Reader: strings.NewReader("<Error>denied</Error>")}
	resp, err := BufferWriteResponse(ctx, &Response{
		StatusCode: http.StatusForbidden,
		Header:     http.Header{"X-Test": []string{"ok"}},
		Body:       original,
	})
	if err != nil {
		t.Fatalf("BufferWriteResponse error = %v", err)
	}
	if !original.closed {
		t.Fatal("expected original body to be closed")
	}
	cancel()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("read buffered body: %v", err)
	}
	if got, want := string(body), "<Error>denied</Error>"; got != want {
		t.Fatalf("body = %q, want %q", got, want)
	}
	if got := resp.Header.Get("X-Test"); got != "ok" {
		t.Fatalf("header = %q, want ok", got)
	}
}

type blockingBody struct {
	closeOnce sync.Once
	done      chan struct{}
}

func newBlockingBody() *blockingBody {
	return &blockingBody{done: make(chan struct{})}
}

func (b *blockingBody) Read([]byte) (int, error) {
	<-b.done
	return 0, io.EOF
}

func (b *blockingBody) Close() error {
	b.closeOnce.Do(func() { close(b.done) })
	return nil
}

func (b *blockingBody) closed() bool {
	select {
	case <-b.done:
		return true
	default:
		return false
	}
}

type trackingBody struct {
	io.Reader
	closed bool
}

func (b *trackingBody) Close() error {
	b.closed = true
	return nil
}
