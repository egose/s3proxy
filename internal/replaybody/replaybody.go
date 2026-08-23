package replaybody

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"sync"
)

const DefaultMaxBytes int64 = 32 << 20
const DefaultAggregateMaxBytes int64 = DefaultMaxBytes * 8
const readChunkSize = 32 * 1024

var ErrBodyTooLarge = errors.New("request body too large to replay")
var ErrBudgetExhausted = errors.New("aggregate replay body budget exhausted")

func IsTooLarge(err error) bool {
	return errors.Is(err, ErrBodyTooLarge)
}

func IsBudgetExhausted(err error) bool {
	return errors.Is(err, ErrBudgetExhausted)
}

type Budget struct {
	maxRequestBytes   int64
	maxAggregateBytes int64
	mu                sync.Mutex
	used              int64
}

func NewBudget(maxRequestBytes, maxAggregateBytes int64) *Budget {
	if maxRequestBytes <= 0 {
		maxRequestBytes = DefaultMaxBytes
	}
	if maxAggregateBytes <= 0 {
		maxAggregateBytes = DefaultAggregateMaxBytes
	}
	return &Budget{maxRequestBytes: maxRequestBytes, maxAggregateBytes: maxAggregateBytes}
}

func (b *Budget) Used() int64 {
	if b == nil {
		return 0
	}
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.used
}

func (b *Budget) Ensure(req *http.Request) error {
	if req.Body == nil || req.Body == http.NoBody || req.GetBody != nil {
		return nil
	}
	if b == nil {
		return fmt.Errorf("replay budget is not configured")
	}
	maxBytes := b.maxRequestBytes
	if req.ContentLength > maxBytes {
		return ErrBodyTooLarge
	}
	original := req.Body
	var closeOnce sync.Once
	var closeErr error
	closeOriginal := func() error {
		closeOnce.Do(func() {
			closeErr = original.Close()
		})
		return closeErr
	}
	done := make(chan struct{})
	if ctxDone := req.Context().Done(); ctxDone != nil {
		go func() {
			select {
			case <-ctxDone:
				_ = closeOriginal()
			case <-done:
			}
		}()
	}
	body, err := b.read(req.Context(), original, maxBytes, req.ContentLength)
	close(done)
	errClose := closeOriginal()
	if err != nil {
		return err
	}
	if errClose != nil {
		body.release(b)
		return fmt.Errorf("close request body: %w", errClose)
	}
	install(req, body, b)
	return nil
}

func Reset(req *http.Request) error {
	if req.GetBody == nil {
		if req.Body == nil {
			req.Body = http.NoBody
		}
		return nil
	}
	body, err := req.GetBody()
	if err != nil {
		return fmt.Errorf("get request body: %w", err)
	}
	req.Body = body
	return nil
}

func Release(req *http.Request) error {
	if req == nil {
		return nil
	}
	var err error
	if req.Body != nil && req.Body != http.NoBody {
		err = req.Body.Close()
		if replay, ok := req.Body.(*replayReadCloser); ok {
			replay.payload.release()
		}
	}
	req.Body = http.NoBody
	req.GetBody = nil
	return err
}

func (b *Budget) read(ctx context.Context, src io.Reader, maxBytes, contentLength int64) (*bufferedBody, error) {
	if contentLength > 0 {
		return b.readKnown(ctx, src, contentLength)
	}
	return b.readUnknown(ctx, src, maxBytes)
}

func (b *Budget) readKnown(ctx context.Context, src io.Reader, length int64) (*bufferedBody, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if err := b.reserve(length); err != nil {
		return nil, err
	}
	body := make([]byte, length)
	if _, err := io.ReadFull(src, body); err != nil {
		b.release(length)
		if ctxErr := ctx.Err(); ctxErr != nil {
			return nil, ctxErr
		}
		return nil, fmt.Errorf("read request body: %w", err)
	}
	if err := ctx.Err(); err != nil {
		b.release(length)
		return nil, err
	}
	return &bufferedBody{body: body, length: length, charged: length}, nil
}

func (b *Budget) readUnknown(ctx context.Context, src io.Reader, maxBytes int64) (*bufferedBody, error) {
	var chunks [][]byte
	buf := make([]byte, readChunkSize)
	var length int64
	var charged int64
	for {
		if err := ctx.Err(); err != nil {
			b.release(charged)
			return nil, err
		}
		n, err := src.Read(buf)
		if n > 0 {
			if length+int64(n) > maxBytes {
				b.release(charged)
				return nil, ErrBodyTooLarge
			}
			if err := b.reserve(int64(n)); err != nil {
				b.release(charged)
				return nil, err
			}
			charged += int64(n)
			chunk := make([]byte, n)
			copy(chunk, buf[:n])
			chunks = append(chunks, chunk)
			length += int64(n)
		}
		if err == io.EOF {
			return &bufferedBody{chunks: chunks, length: length, charged: charged}, nil
		}
		if err != nil {
			b.release(charged)
			if ctxErr := ctx.Err(); ctxErr != nil {
				return nil, ctxErr
			}
			return nil, fmt.Errorf("read request body: %w", err)
		}
	}
}

func (b *Budget) reserve(n int64) error {
	if n <= 0 {
		return nil
	}
	b.mu.Lock()
	defer b.mu.Unlock()
	if b.used+n > b.maxAggregateBytes {
		return ErrBudgetExhausted
	}
	b.used += n
	return nil
}

func (b *Budget) release(n int64) {
	if b == nil || n <= 0 {
		return
	}
	b.mu.Lock()
	b.used -= n
	if b.used < 0 {
		b.used = 0
	}
	b.mu.Unlock()
}

func install(req *http.Request, body *bufferedBody, budget *Budget) {
	if body == nil || body.length == 0 {
		req.Body = http.NoBody
		req.GetBody = func() (io.ReadCloser, error) { return http.NoBody, nil }
		req.ContentLength = 0
		return
	}
	payload := &replayPayload{body: body.body, chunks: body.chunks, length: body.length, charged: body.charged, budget: budget}
	if done := req.Context().Done(); done != nil {
		go func() {
			<-done
			payload.release()
		}()
	}
	req.GetBody = func() (io.ReadCloser, error) {
		return &replayReadCloser{Reader: payload.reader(), payload: payload}, nil
	}
	req.Body = &replayReadCloser{Reader: payload.reader(), payload: payload, releaseOnClose: true}
	req.ContentLength = payload.length
}

type bufferedBody struct {
	body    []byte
	chunks  [][]byte
	length  int64
	charged int64
}

func (b *bufferedBody) release(budget *Budget) {
	if b == nil {
		return
	}
	budget.release(b.charged)
	b.charged = 0
}

type replayPayload struct {
	body    []byte
	chunks  [][]byte
	length  int64
	charged int64
	budget  *Budget
	once    sync.Once
}

func (p *replayPayload) reader() io.Reader {
	if p.body != nil {
		return bytes.NewReader(p.body)
	}
	return &chunkReader{chunks: p.chunks, remaining: p.length}
}

func (p *replayPayload) release() {
	p.once.Do(func() {
		p.budget.release(p.charged)
	})
}

type replayReadCloser struct {
	io.Reader
	payload        *replayPayload
	releaseOnClose bool
}

func (r *replayReadCloser) Close() error {
	if r.releaseOnClose {
		r.payload.release()
	}
	return nil
}

type chunkReader struct {
	chunks    [][]byte
	chunk     int
	offset    int
	remaining int64
}

func (r *chunkReader) Read(p []byte) (int, error) {
	if len(p) == 0 {
		return 0, nil
	}
	if r.remaining == 0 {
		return 0, io.EOF
	}
	written := 0
	for written < len(p) && r.remaining > 0 {
		chunk := r.chunks[r.chunk]
		n := copy(p[written:], chunk[r.offset:])
		written += n
		r.offset += n
		r.remaining -= int64(n)
		if r.offset == len(chunk) {
			r.chunk++
			r.offset = 0
		}
	}
	return written, nil
}
