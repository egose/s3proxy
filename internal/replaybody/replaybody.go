package replaybody

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"net/http"
)

const DefaultMaxBytes int64 = 32 << 20

var ErrBodyTooLarge = errors.New("request body too large to replay")

func IsTooLarge(err error) bool {
	return errors.Is(err, ErrBodyTooLarge)
}

func Ensure(req *http.Request) error {
	return EnsureWithLimit(req, DefaultMaxBytes)
}

func EnsureWithLimit(req *http.Request, maxBytes int64) error {
	if req.Body == nil || req.Body == http.NoBody || req.GetBody != nil {
		return nil
	}
	if maxBytes <= 0 {
		maxBytes = DefaultMaxBytes
	}
	if req.ContentLength > maxBytes {
		return ErrBodyTooLarge
	}
	body, err := io.ReadAll(io.LimitReader(req.Body, maxBytes+1))
	if err != nil {
		return fmt.Errorf("read request body: %w", err)
	}
	if int64(len(body)) > maxBytes {
		return ErrBodyTooLarge
	}
	install(req, body)
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

func install(req *http.Request, body []byte) {
	if body == nil {
		req.Body = http.NoBody
		req.GetBody = func() (io.ReadCloser, error) { return http.NoBody, nil }
		req.ContentLength = 0
		return
	}
	req.GetBody = func() (io.ReadCloser, error) {
		return io.NopCloser(bytes.NewReader(body)), nil
	}
	req.Body = io.NopCloser(bytes.NewReader(body))
	req.ContentLength = int64(len(body))
}
