package s3

import (
	"errors"
	"io"
)

const maxDiscardBodyBytes int64 = 512 << 10

func DrainAndClose(resp *Response) error {
	if resp == nil || resp.Body == nil {
		return nil
	}
	_, drainErr := io.CopyN(io.Discard, resp.Body, maxDiscardBodyBytes)
	closeErr := resp.Body.Close()
	if closeErr != nil {
		return closeErr
	}
	if drainErr != nil && !errors.Is(drainErr, io.EOF) {
		return drainErr
	}
	return nil
}
