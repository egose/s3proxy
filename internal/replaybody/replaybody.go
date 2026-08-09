package replaybody

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"net/http"
	"sync"
)

const DefaultMaxBytes int64 = 32 << 20
const DefaultAggregateMaxBytes int64 = DefaultMaxBytes * 8

var ErrBodyTooLarge = errors.New("request body too large to replay")
var ErrBudgetExhausted = errors.New("aggregate replay body budget exhausted")

var defaultBudget = NewBudget(DefaultMaxBytes, DefaultAggregateMaxBytes)

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

func Ensure(req *http.Request) error {
	return defaultBudget.Ensure(req)
}

func EnsureWithLimit(req *http.Request, maxBytes int64) error {
	return NewBudget(maxBytes, DefaultAggregateMaxBytes).Ensure(req)
}

func (b *Budget) Ensure(req *http.Request) error {
	if req.Body == nil || req.Body == http.NoBody || req.GetBody != nil {
		return nil
	}
	if b == nil {
		b = defaultBudget
	}
	maxBytes := b.maxRequestBytes
	if req.ContentLength > maxBytes {
		return ErrBodyTooLarge
	}
	body, err := b.read(req.Body, maxBytes)
	if err != nil {
		return err
	}
	if err := req.Body.Close(); err != nil {
		b.release(int64(len(body)))
		return fmt.Errorf("close request body: %w", err)
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

func (b *Budget) read(src io.Reader, maxBytes int64) ([]byte, error) {
	var body []byte
	buf := make([]byte, 32*1024)
	reserved := int64(0)
	defer func() {
		if reserved > int64(len(body)) {
			b.release(reserved - int64(len(body)))
		}
	}()
	for {
		n, err := src.Read(buf)
		if n > 0 {
			if int64(len(body)+n) > maxBytes {
				b.release(reserved)
				reserved = 0
				return nil, ErrBodyTooLarge
			}
			if err := b.reserve(int64(n)); err != nil {
				b.release(reserved)
				reserved = 0
				return nil, err
			}
			reserved += int64(n)
			body = append(body, buf[:n]...)
		}
		if err == io.EOF {
			return body, nil
		}
		if err != nil {
			b.release(reserved)
			reserved = 0
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

func install(req *http.Request, body []byte, budget *Budget) {
	if body == nil {
		req.Body = http.NoBody
		req.GetBody = func() (io.ReadCloser, error) { return http.NoBody, nil }
		req.ContentLength = 0
		return
	}
	payload := &replayPayload{body: body, budget: budget}
	if done := req.Context().Done(); done != nil {
		go func() {
			<-done
			payload.release()
		}()
	}
	req.GetBody = func() (io.ReadCloser, error) {
		return &replayReadCloser{Reader: bytes.NewReader(payload.body), payload: payload}, nil
	}
	req.Body = &replayReadCloser{Reader: bytes.NewReader(payload.body), payload: payload, releaseOnClose: true}
	req.ContentLength = int64(len(body))
}

type replayPayload struct {
	body   []byte
	budget *Budget
	once   sync.Once
}

func (p *replayPayload) release() {
	p.once.Do(func() {
		p.budget.release(int64(len(p.body)))
		p.body = nil
	})
}

type replayReadCloser struct {
	*bytes.Reader
	payload        *replayPayload
	releaseOnClose bool
}

func (r *replayReadCloser) Close() error {
	if r.releaseOnClose {
		r.payload.release()
	}
	return nil
}
