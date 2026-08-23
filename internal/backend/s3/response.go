package s3

import (
	"bytes"
	"context"
	"errors"
	"io"
	"net/http"
	"time"
)

const maxDiscardBodyBytes int64 = 512 << 10
const maxBufferedWriteResponseBytes int64 = 1 << 20
const responseDiscardTimeout = 250 * time.Millisecond
const responseBufferTimeout = 2 * time.Second

var ErrResponseBodyTooLarge = errors.New("response body too large to retain")

func DrainAndClose(resp *Response) error {
	return DrainAndCloseContext(context.Background(), resp)
}

func DrainAndCloseContext(ctx context.Context, resp *Response) error {
	if resp == nil || resp.Body == nil {
		return nil
	}
	return readAndClose(ctx, resp.Body, responseDiscardTimeout, func() error {
		_, err := io.CopyN(io.Discard, resp.Body, maxDiscardBodyBytes)
		if errors.Is(err, io.EOF) {
			return nil
		}
		return err
	})
}

func BufferWriteResponse(ctx context.Context, resp *Response) (*Response, error) {
	if resp == nil || resp.Body == nil {
		return resp, nil
	}
	var buf bytes.Buffer
	err := readAndClose(ctx, resp.Body, responseBufferTimeout, func() error {
		_, err := io.Copy(&buf, io.LimitReader(resp.Body, maxBufferedWriteResponseBytes+1))
		if err != nil {
			return err
		}
		if int64(buf.Len()) > maxBufferedWriteResponseBytes {
			return ErrResponseBodyTooLarge
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return &Response{
		StatusCode: resp.StatusCode,
		Header:     cloneHeader(resp.Header),
		Body:       io.NopCloser(bytes.NewReader(buf.Bytes())),
	}, nil
}

func readAndClose(ctx context.Context, body io.ReadCloser, timeout time.Duration, read func() error) error {
	if ctx == nil {
		ctx = context.Background()
	}
	readCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	done := make(chan error, 1)
	go func() {
		done <- read()
	}()
	select {
	case err := <-done:
		return errors.Join(err, body.Close())
	case <-readCtx.Done():
		closeErr := body.Close()
		select {
		case err := <-done:
			return errors.Join(err, closeErr, readCtx.Err())
		default:
			return errors.Join(readCtx.Err(), closeErr)
		}
	}
}

func cloneHeader(header http.Header) http.Header {
	if header == nil {
		return http.Header{}
	}
	return header.Clone()
}
