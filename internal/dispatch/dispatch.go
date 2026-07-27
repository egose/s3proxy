package dispatch

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"

	"github.com/jahn/s3proxy/internal/backend/s3"
	"github.com/jahn/s3proxy/internal/config"
	"github.com/jahn/s3proxy/internal/rewrite"
	"github.com/jahn/s3proxy/internal/router"
	"github.com/jahn/s3proxy/internal/s3ops"
)

type Result struct {
	Primary *s3.Response
	Errors  map[string]error
}

type Fanout interface {
	Dispatch(ctx context.Context, match router.Match, req *http.Request, op s3ops.Operation, rw rewrite.Result) (*Result, error)
}

func New(backend s3.Executor) Fanout {
	return &dispatcher{backend: backend}
}

type dispatcher struct {
	backend s3.Executor
}

func (d *dispatcher) Dispatch(ctx context.Context, match router.Match, req *http.Request, op s3ops.Operation, rw rewrite.Result) (*Result, error) {
	result := &Result{Errors: make(map[string]error)}

	if s3ops.IsRead(op) || match.Route.Dispatch == config.DispatchFirst || !s3ops.SupportsFanout(op) {
		var target config.S3Target
		if match.EffectiveRead != nil {
			target = *match.EffectiveRead
		} else if len(match.Destinations) > 0 {
			target = match.Destinations[0]
		} else {
			return nil, fmt.Errorf("no destination available")
		}

		resp, err := d.backend.Do(ctx, s3.Request{
			Operation: op,
			Target:    target,
			Bucket:    rw.Bucket,
			Key:       rw.Key,
			Source:    req,
		})
		if err != nil {
			return nil, err
		}
		result.Primary = resp
		return result, nil
	}

	var bodyBytes []byte
	if req.Body != nil {
		bodyBytes, _ = io.ReadAll(req.Body)
	}

	for _, dest := range match.Destinations {
		req.Body = io.NopCloser(bytes.NewReader(bodyBytes))

		resp, err := d.backend.Do(ctx, s3.Request{
			Operation: op,
			Target:    dest,
			Bucket:    rw.Bucket,
			Key:       rw.Key,
			Source:    req,
		})
		if err != nil {
			result.Errors[dest.Name] = err
			continue
		}

		if resp.StatusCode >= 400 {
			result.Errors[dest.Name] = fmt.Errorf("upstream returned status %d", resp.StatusCode)
		}

		if result.Primary == nil {
			// The first response becomes the primary and is streamed back
			// to the client by the httpapi; do NOT close its body here.
			result.Primary = resp
		} else {
			// Non-primary responses are not surfaced to the client; close
			// their bodies immediately to release the upstream connection.
			resp.Body.Close()
		}
	}

	if len(result.Errors) > 0 {
		return result, fmt.Errorf("fan-out had %d failures", len(result.Errors))
	}

	return result, nil
}
