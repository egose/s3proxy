package replaybody

import (
	"bytes"
	"context"
	"errors"
	"io"
	"net/http"
	"sync"
	"sync/atomic"
	"testing"
)

type fragmentedReader struct {
	data        []byte
	sizes       []int
	reads       int
	eofWithData bool
	terminalErr error
}

func (r *fragmentedReader) Read(p []byte) (int, error) {
	r.reads++
	if len(r.data) == 0 {
		if r.terminalErr != nil {
			return 0, r.terminalErr
		}
		return 0, io.EOF
	}
	size := r.sizes[(r.reads-1)%len(r.sizes)]
	if len(p) > size {
		p = p[:size]
	}
	n := copy(p, r.data)
	r.data = r.data[n:]
	if len(r.data) == 0 && (r.eofWithData || r.terminalErr != nil) {
		if r.terminalErr != nil {
			return n, r.terminalErr
		}
		return n, io.EOF
	}
	return n, nil
}

type trackedFragmentBody struct {
	io.Reader
	closes   atomic.Int32
	closeErr error
}

func (b *trackedFragmentBody) Close() error {
	b.closes.Add(1)
	return b.closeErr
}

type replayReadFunc func([]byte) (int, error)

func (f replayReadFunc) Read(p []byte) (int, error) { return f(p) }

func TestUnknownShortReadsCoalesceAndReplay(t *testing.T) {
	for _, tc := range []struct {
		name        string
		size        int
		reads       []int
		eofWithData bool
	}{
		{name: "one-byte", size: 3*readChunkSize + 137, reads: []int{1}},
		{name: "one-byte-data-eof", size: 3*readChunkSize + 137, reads: []int{1}, eofWithData: true},
		{name: "irregular", size: 3*readChunkSize + 137, reads: []int{0, 1, 7, 1023, 3, 8191, 31}},
		{name: "irregular-data-eof", size: 3*readChunkSize + 137, reads: []int{0, 1, 7, 1023, 3, 8191, 31}, eofWithData: true},
		{name: "full-chunk", size: readChunkSize, reads: []int{1}, eofWithData: true},
		{name: "empty", reads: []int{1}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			data := make([]byte, tc.size)
			for i := range data {
				data[i] = byte(i*31 + i/251)
			}
			budget := NewBudget(int64(tc.size), int64(tc.size))
			source := &trackedFragmentBody{Reader: &fragmentedReader{data: data, sizes: tc.reads, eofWithData: tc.eofWithData}}
			req, err := http.NewRequest(http.MethodPut, "http://proxy.local/bucket/key", source)
			if err != nil {
				t.Fatal(err)
			}
			req.ContentLength = -1
			if err := budget.Ensure(req); err != nil {
				t.Fatal(err)
			}
			defer Release(req)
			if source.closes.Load() != 1 || budget.Used() != int64(tc.size) || req.ContentLength != int64(tc.size) {
				t.Fatalf("closes=%d used=%d length=%d", source.closes.Load(), budget.Used(), req.ContentLength)
			}
			if tc.size > 0 {
				payload := req.Body.(*replayReadCloser).payload
				wantChunks := (tc.size + readChunkSize - 1) / readChunkSize
				if len(payload.chunks) != wantChunks || cap(payload.chunks) > 3*wantChunks {
					t.Fatalf("chunk len/cap = %d/%d, want %d chunks with bounded metadata", len(payload.chunks), cap(payload.chunks), wantChunks)
				}
				capacity := 0
				for i, chunk := range payload.chunks {
					capacity += cap(chunk)
					if len(chunk) != cap(chunk) || (i < wantChunks-1 && len(chunk) != readChunkSize) {
						t.Fatalf("chunk %d len/cap = %d/%d", i, len(chunk), cap(chunk))
					}
				}
				if capacity != tc.size || payload.charged != int64(capacity) {
					t.Fatalf("retained capacity=%d charged=%d, want %d", capacity, payload.charged, tc.size)
				}
			}
			check := func(reader io.Reader) {
				got, err := io.ReadAll(reader)
				if err != nil || !bytes.Equal(got, data) {
					t.Errorf("replayed bytes differ: length=%d err=%v", len(got), err)
				}
			}
			var wg sync.WaitGroup
			for i := 0; i < 8; i++ {
				wg.Add(1)
				go func() {
					defer wg.Done()
					reader, err := req.GetBody()
					if err != nil {
						t.Error(err)
						return
					}
					defer reader.Close()
					check(reader)
				}()
			}
			wg.Wait()
			if budget.Used() != int64(tc.size) {
				t.Fatal("independent reader closure released caller reservation")
			}
			_, _ = req.Body.Read(make([]byte, 17))
			if err := Reset(req); err != nil {
				t.Fatal(err)
			}
			check(req.Body)
			acquired, err := req.GetBody()
			if err != nil {
				t.Fatal(err)
			}
			defer acquired.Close()
			if err := Release(req); err != nil {
				t.Fatal(err)
			}
			if budget.Used() != 0 || req.GetBody != nil || req.Body != http.NoBody {
				t.Fatal("release did not clear reservation and replay hooks")
			}
			check(acquired)
		})
	}
}

func TestUnknownShortReadLimitsAndFailures(t *testing.T) {
	const size = 2*readChunkSize + 13
	failure := errors.New("fragment failure")
	for _, tc := range []struct {
		name      string
		limit     int64
		aggregate int64
		readErr   error
		closeErr  error
		wantErr   error
	}{
		{name: "exact", limit: size, aggregate: size},
		{name: "overflow", limit: size - 1, aggregate: size, wantErr: ErrBodyTooLarge},
		{name: "aggregate", limit: size, aggregate: size - 1, wantErr: ErrBudgetExhausted},
		{name: "data-error", limit: size, aggregate: size, readErr: failure, wantErr: failure},
		{name: "close-error", limit: size, aggregate: size, closeErr: failure, wantErr: failure},
	} {
		t.Run(tc.name, func(t *testing.T) {
			budget := NewBudget(tc.limit, tc.aggregate+17)
			if err := budget.reserve(17); err != nil {
				t.Fatal(err)
			}
			defer budget.release(17)
			source := &trackedFragmentBody{
				Reader:   &fragmentedReader{data: bytes.Repeat([]byte("x"), size), sizes: []int{1}, eofWithData: true, terminalErr: tc.readErr},
				closeErr: tc.closeErr,
			}
			req, err := http.NewRequest(http.MethodPut, "http://proxy.local/bucket/key", source)
			if err != nil {
				t.Fatal(err)
			}
			req.ContentLength = -1
			err = budget.Ensure(req)
			if !errors.Is(err, tc.wantErr) {
				t.Fatalf("err=%v, want %v", err, tc.wantErr)
			}
			if source.closes.Load() != 1 {
				t.Fatalf("source closes=%d, want 1", source.closes.Load())
			}
			if tc.wantErr == nil {
				if budget.Used() != size+17 {
					t.Fatalf("used=%d, want %d", budget.Used(), size+17)
				}
				if err := req.Body.Close(); err != nil {
					t.Fatal(err)
				}
				if err := Release(req); err != nil {
					t.Fatal(err)
				}
			} else if req.GetBody != nil {
				t.Fatal("failed read installed replay")
			}
			if budget.Used() != 17 {
				t.Fatalf("used=%d, want other reservation 17", budget.Used())
			}
		})
	}
}

func TestUnknownShortReadTermination(t *testing.T) {
	failure := errors.New("read failure after progress")
	for _, tc := range []struct {
		name   string
		result error
		cancel bool
		stalls bool
		want   error
	}{
		{name: "error-after-progress", result: failure, want: failure},
		{name: "cancel-data-eof", result: io.EOF, cancel: true, want: context.Canceled},
		{name: "cancel-data", cancel: true, want: context.Canceled},
		{name: "cancel-error", result: failure, cancel: true, want: context.Canceled},
		{name: "no-progress", stalls: true, want: io.ErrNoProgress},
	} {
		t.Run(tc.name, func(t *testing.T) {
			budget := NewBudget(100, 100)
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			calls := 0
			source := &trackedFragmentBody{Reader: replayReadFunc(func(p []byte) (int, error) {
				calls++
				if calls == 1 {
					return copy(p, "abc"), nil
				}
				if budget.Used() != 3 {
					t.Errorf("used=%d before next read, want immediate partial charge 3", budget.Used())
				}
				if tc.cancel {
					cancel()
					return copy(p, "d"), tc.result
				}
				if tc.stalls && calls <= 101 {
					return 0, nil
				}
				if tc.stalls {
					return 0, failure
				}
				return 0, tc.result
			})}
			req, err := http.NewRequestWithContext(ctx, http.MethodPut, "http://proxy.local/bucket/key", source)
			if err != nil {
				t.Fatal(err)
			}
			req.ContentLength = -1
			if err := budget.Ensure(req); !errors.Is(err, tc.want) {
				t.Fatalf("err=%v, want %v", err, tc.want)
			}
			if source.closes.Load() != 1 || budget.Used() != 0 || req.GetBody != nil {
				t.Fatalf("closes=%d used=%d replay=%v", source.closes.Load(), budget.Used(), req.GetBody != nil)
			}
			if tc.stalls && calls != 101 {
				t.Fatalf("read calls=%d, want initial progress plus 100 empty reads", calls)
			}
		})
	}
}

func TestUnknownShortReadProgressResetsEmptyReadCount(t *testing.T) {
	budget := NewBudget(3, 3)
	calls := 0
	source := &trackedFragmentBody{Reader: replayReadFunc(func(p []byte) (int, error) {
		calls++
		if calls > 300 {
			return 0, io.EOF
		}
		if calls%100 == 0 {
			return copy(p, "x"), nil
		}
		return 0, nil
	})}
	req, err := http.NewRequest(http.MethodPut, "http://proxy.local/bucket/key", source)
	if err != nil {
		t.Fatal(err)
	}
	req.ContentLength = -1
	if err := budget.Ensure(req); err != nil {
		t.Fatal(err)
	}
	defer Release(req)
	got, err := io.ReadAll(req.Body)
	if err != nil || string(got) != "xxx" || calls != 301 {
		t.Fatalf("body=%q err=%v calls=%d", got, err, calls)
	}
}
