package replaybody

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"runtime"
	"testing"
	"time"
	"weak"
)

func TestReleasedReplayLifetimeWithLiveContext(t *testing.T) {
	const iterations = 32
	const otherReservation = 17
	data := replayLifecycleBytes()
	for _, length := range []struct {
		name  string
		value int64
	}{
		{name: "known", value: int64(len(data))},
		{name: "unknown", value: -1},
	} {
		for _, mode := range []string{"release", "owner-close", "reset-release"} {
			t.Run(length.name+"/"+mode, func(t *testing.T) {
				ctx, cancel := context.WithCancel(context.Background())
				defer cancel()
				budget := NewBudget(int64(len(data)), int64(len(data)+otherReservation))
				if err := budget.reserve(otherReservation); err != nil {
					t.Fatal(err)
				}
				defer budget.release(otherReservation)
				baseline := replayWatcherCount()
				refs := make([]replayLifetimeRefs, 0, iterations)
				for i := 0; i < iterations; i++ {
					refs = append(refs, releaseReplayWithLiveContext(t, ctx, budget, data, length.value, mode))
				}

				deadline := time.Now().Add(5 * time.Second)
				var payloads, storage, watchers int
				for {
					runtime.GC()
					payloads, storage = liveReplayReferences(refs)
					watchers = replayWatcherCount()
					if (payloads == 0 && storage == 0 && watchers <= baseline) || time.Now().After(deadline) {
						break
					}
					time.Sleep(10 * time.Millisecond)
				}
				if err := ctx.Err(); err != nil {
					t.Fatalf("release canceled the caller context: %v", err)
				}
				if got := budget.Used(); got != otherReservation {
					t.Fatalf("used = %d, want other reservation %d", got, otherReservation)
				}
				if payloads != 0 || storage != 0 || watchers > baseline {
					t.Fatalf("released replays with live context retain %d payloads, %d storage allocations, %d watchers (baseline %d)", payloads, storage, watchers, baseline)
				}
				t.Logf("released %d replays of %d bytes: reachable payloads/storage = %d/%d, watchers = %d (baseline %d); context remains live", iterations, len(data), payloads, storage, watchers, baseline)
			})
		}
	}
}

func releaseReplayWithLiveContext(t *testing.T, ctx context.Context, budget *Budget, data []byte, length int64, mode string) replayLifetimeRefs {
	t.Helper()
	otherReservation := budget.Used()
	source := &trackedFragmentBody{Reader: bytes.NewReader(data)}
	req, err := http.NewRequestWithContext(ctx, http.MethodPut, "http://proxy.local/bucket/key", source)
	if err != nil {
		t.Fatal(err)
	}
	req.ContentLength = length
	if err := budget.Ensure(req); err != nil {
		t.Fatal(err)
	}
	if req.ContentLength != int64(len(data)) || budget.Used() != otherReservation+int64(len(data)) {
		t.Fatalf("length/used = %d/%d, want %d/%d", req.ContentLength, budget.Used(), len(data), otherReservation+int64(len(data)))
	}
	payload := req.Body.(*replayReadCloser).payload
	refs := replayLifetimeRefs{payload: weak.Make(payload)}
	if len(payload.body) > 0 {
		refs.storage = append(refs.storage, weak.Make(&payload.body[0]))
	}
	for _, chunk := range payload.chunks {
		refs.storage = append(refs.storage, weak.Make(&chunk[0]))
	}
	getBody := req.GetBody
	acquired, err := getBody()
	if err != nil {
		t.Fatal(err)
	}
	defer acquired.Close()
	prefix := make([]byte, readChunkSize+7)
	if _, err := io.ReadFull(acquired, prefix); err != nil || !bytes.Equal(prefix, data[:len(prefix)]) {
		t.Fatalf("read before release differs: %v", err)
	}

	endOwnership := func() error { return Release(req) }
	if mode == "owner-close" {
		endOwnership = req.Body.Close
	}
	if mode == "reset-release" {
		if _, err := io.CopyN(io.Discard, req.Body, 31); err != nil {
			t.Fatal(err)
		}
		if err := Reset(req); err != nil {
			t.Fatal(err)
		}
		checkReplayBytes(t, req.Body, data)
		if err := req.Body.Close(); err != nil {
			t.Fatal(err)
		}
		if budget.Used() != otherReservation+int64(len(data)) {
			t.Fatal("closing a reset reader released caller ownership")
		}
	}
	for i := 0; i < 2; i++ {
		if err := endOwnership(); err != nil {
			t.Fatal(err)
		}
	}
	if mode != "owner-close" && (req.GetBody != nil || req.Body != http.NoBody) {
		t.Fatal("Release did not clear the replay hooks")
	}
	if got := budget.Used(); got != otherReservation {
		t.Fatalf("used after release = %d, want other reservation %d", got, otherReservation)
	}
	checkReplayBytes(t, acquired, data[len(prefix):])
	later, err := getBody()
	if err != nil {
		t.Fatal(err)
	}
	defer later.Close()
	checkReplayBytes(t, later, data)
	if source.closes.Load() != 1 {
		t.Fatalf("original body closes = %d, want 1", source.closes.Load())
	}
	return refs
}

func TestEnsureCancellationAtReplayInstall(t *testing.T) {
	const otherReservation = 17
	data := replayLifecycleBytes()
	for _, length := range []struct {
		name  string
		value int64
	}{
		{name: "known", value: int64(len(data))},
		{name: "unknown", value: -1},
	} {
		t.Run(length.name, func(t *testing.T) {
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			budget := NewBudget(int64(len(data)), int64(len(data)+otherReservation))
			if err := budget.reserve(otherReservation); err != nil {
				t.Fatal(err)
			}
			defer budget.release(otherReservation)
			source := &cancelOnCloseBody{
				trackedFragmentBody: &trackedFragmentBody{Reader: bytes.NewReader(data)},
				cancel:              cancel,
			}
			req, err := http.NewRequestWithContext(ctx, http.MethodPut, "http://proxy.local/bucket/key", source)
			if err != nil {
				t.Fatal(err)
			}
			defer Release(req)
			req.ContentLength = length.value
			if err := budget.Ensure(req); err != nil {
				t.Fatal(err)
			}
			if ctx.Err() != context.Canceled || source.closes.Load() != 1 {
				t.Fatalf("context error = %v, original body closes = %d", ctx.Err(), source.closes.Load())
			}
			deadline := time.Now().Add(5 * time.Second)
			for budget.Used() != otherReservation && time.Now().Before(deadline) {
				time.Sleep(time.Millisecond)
			}
			if got := budget.Used(); got != otherReservation {
				t.Fatalf("used after cancellation = %d, want other reservation %d", got, otherReservation)
			}
			checkReplayBytes(t, req.Body, data)
			reader, err := req.GetBody()
			if err != nil {
				t.Fatal(err)
			}
			defer reader.Close()
			checkReplayBytes(t, reader, data)
			if err := Release(req); err != nil {
				t.Fatal(err)
			}
			if got := budget.Used(); got != otherReservation {
				t.Fatalf("used after cancellation and release = %d, want %d", got, otherReservation)
			}
		})
	}
}

type cancelOnCloseBody struct {
	*trackedFragmentBody
	cancel context.CancelFunc
}

func (b *cancelOnCloseBody) Close() error {
	err := b.trackedFragmentBody.Close()
	b.cancel()
	return err
}

type replayLifetimeRefs struct {
	payload weak.Pointer[replayPayload]
	storage []weak.Pointer[byte]
}

func liveReplayReferences(refs []replayLifetimeRefs) (payloads, storage int) {
	for _, ref := range refs {
		if ref.payload.Value() != nil {
			payloads++
		}
		for _, allocation := range ref.storage {
			if allocation.Value() != nil {
				storage++
			}
		}
	}
	return payloads, storage
}

func replayWatcherCount() int {
	for size := 64 * 1024; ; size *= 2 {
		stack := make([]byte, size)
		n := runtime.Stack(stack, true)
		if n < len(stack) {
			return bytes.Count(stack[:n], []byte("\ngithub.com/egose/s3proxy/internal/replaybody.install.func"))
		}
	}
}

func replayLifecycleBytes() []byte {
	data := make([]byte, 2*readChunkSize+137)
	for i := range data {
		data[i] = byte(i*31 + i/251)
	}
	return data
}

func checkReplayBytes(t *testing.T, reader io.Reader, want []byte) {
	t.Helper()
	got, err := io.ReadAll(reader)
	if err != nil || !bytes.Equal(got, want) {
		t.Errorf("replayed bytes differ: length = %d, want %d, error = %v", len(got), len(want), err)
	}
}
