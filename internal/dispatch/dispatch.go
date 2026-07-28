package dispatch

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"

	"github.com/egose/s3proxy/internal/backend/s3"
	"github.com/egose/s3proxy/internal/config"
	"github.com/egose/s3proxy/internal/rewrite"
	"github.com/egose/s3proxy/internal/router"
	"github.com/egose/s3proxy/internal/s3ops"
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

	if s3ops.IsRead(op) && match.Route.ReadPreference == config.ReadOrderedFailover {
		return d.dispatchOrderedFailover(ctx, result, match, req, op, rw)
	}

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

	if err := ensureReplayableBody(req); err != nil {
		return nil, err
	}

	for _, dest := range match.Destinations {
		if err := resetRequestBody(req); err != nil {
			return nil, err
		}

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

func (d *dispatcher) dispatchOrderedFailover(ctx context.Context, result *Result, match router.Match, req *http.Request, op s3ops.Operation, rw rewrite.Result) (*Result, error) {
	if len(match.Destinations) == 0 {
		return nil, fmt.Errorf("no destination available")
	}

	for _, dest := range match.Destinations {
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

		if resp.StatusCode >= http.StatusInternalServerError {
			result.Errors[dest.Name] = fmt.Errorf("upstream returned status %d", resp.StatusCode)
			if result.Primary != nil && result.Primary.Body != nil {
				result.Primary.Body.Close()
			}
			result.Primary = resp
			continue
		}

		result.Primary = resp
		return result, nil
	}

	if result.Primary != nil {
		return result, fmt.Errorf("ordered failover exhausted %d destinations", len(result.Errors))
	}
	return nil, fmt.Errorf("ordered failover exhausted %d destinations", len(result.Errors))
}

func ensureReplayableBody(req *http.Request) error {
	if req.Body == nil || req.Body == http.NoBody || req.GetBody != nil {
		return nil
	}
	body, err := io.ReadAll(req.Body)
	if err != nil {
		return fmt.Errorf("read request body: %w", err)
	}
	req.GetBody = func() (io.ReadCloser, error) {
		return io.NopCloser(bytes.NewReader(body)), nil
	}
	req.Body = io.NopCloser(bytes.NewReader(body))
	req.ContentLength = int64(len(body))
	return nil
}

func resetRequestBody(req *http.Request) error {
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
