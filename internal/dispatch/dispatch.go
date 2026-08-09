package dispatch

import (
	"context"
	"fmt"
	"net/http"

	"github.com/egose/s3proxy/internal/backend/s3"
	"github.com/egose/s3proxy/internal/config"
	"github.com/egose/s3proxy/internal/replaybody"
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
	return NewWithReplayLimit(backend, replaybody.DefaultMaxBytes)
}

func NewWithReplayLimit(backend s3.Executor, replayBodyMaxBytes int64) Fanout {
	return NewWithReplayBudget(backend, replaybody.NewBudget(replayBodyMaxBytes, replaybody.DefaultAggregateMaxBytes))
}

func NewWithReplayBudget(backend s3.Executor, replayBudget *replaybody.Budget) Fanout {
	if replayBudget == nil {
		replayBudget = replaybody.NewBudget(replaybody.DefaultMaxBytes, replaybody.DefaultAggregateMaxBytes)
	}
	return &dispatcher{backend: backend, replayBudget: replayBudget}
}

type dispatcher struct {
	backend      s3.Executor
	replayBudget *replaybody.Budget
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

	if err := d.budget().Ensure(req); err != nil {
		return nil, err
	}

	for i, dest := range match.Destinations {
		if err := replaybody.Reset(req); err != nil {
			s3.DrainAndClose(result.Primary)
			result.Primary = nil
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

		if i == 0 {
			result.Primary = resp
		} else {
			s3.DrainAndClose(resp)
		}
	}

	if len(result.Errors) > 0 {
		if result.Primary != nil && result.Primary.StatusCode < 400 {
			s3.DrainAndClose(result.Primary)
			result.Primary = nil
		}
		return result, fmt.Errorf("fan-out had %d failures", len(result.Errors))
	}

	return result, nil
}

func (d *dispatcher) budget() *replaybody.Budget {
	if d.replayBudget != nil {
		return d.replayBudget
	}
	return replaybody.NewBudget(replaybody.DefaultMaxBytes, replaybody.DefaultAggregateMaxBytes)
}

func (d *dispatcher) dispatchOrderedFailover(ctx context.Context, result *Result, match router.Match, req *http.Request, op s3ops.Operation, rw rewrite.Result) (*Result, error) {
	if len(match.Destinations) == 0 {
		return nil, fmt.Errorf("no destination available")
	}

	for i, dest := range match.Destinations {
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
			if i < len(match.Destinations)-1 {
				s3.DrainAndClose(resp)
				continue
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
