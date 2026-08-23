package dispatch

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"sync"

	"github.com/egose/s3proxy/internal/backend/s3"
	"github.com/egose/s3proxy/internal/config"
	"github.com/egose/s3proxy/internal/replaybody"
	"github.com/egose/s3proxy/internal/rewrite"
	"github.com/egose/s3proxy/internal/router"
	"github.com/egose/s3proxy/internal/s3ops"
)

const maxParallelWriteFanout = 4

type Result struct {
	Primary       *s3.Response
	CleanupErrors map[string]error
	Attempts      []Attempt
}

func (r *Result) FailureCount() int {
	if r == nil {
		return 0
	}
	count := 0
	for _, attempt := range r.Attempts {
		if attempt.Error != nil {
			count++
		}
	}
	return count
}

type Attempt struct {
	Target     string
	StatusCode int
	Error      error
}

type Fanout interface {
	Dispatch(ctx context.Context, match router.Match, req *http.Request, op s3ops.Operation, rw rewrite.Result) (*Result, error)
}

func New(backend s3.Executor, replayBudget *replaybody.Budget) (Fanout, error) {
	if backend == nil {
		return nil, fmt.Errorf("backend is required")
	}
	if replayBudget == nil {
		return nil, fmt.Errorf("replay budget is required")
	}
	return &dispatcher{backend: backend, replayBudget: replayBudget}, nil
}

type dispatcher struct {
	backend      s3.Executor
	replayBudget *replaybody.Budget
}

func (d *dispatcher) Dispatch(ctx context.Context, match router.Match, req *http.Request, op s3ops.Operation, rw rewrite.Result) (*Result, error) {
	result := &Result{}

	if s3ops.IsRead(op) && match.Route.ReadPreference == config.ReadOrderedFailover {
		return d.dispatchOrderedFailover(ctx, result, match, req, op, rw)
	}

	if s3ops.IsRead(op) || match.Route.Dispatch == config.DispatchFirst || !s3ops.SupportsFanout(op) {
		target := match.EffectiveRead
		if target == "" && len(match.Destinations) > 0 {
			target = match.Destinations[0]
		}
		if target == "" {
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
			result.Attempts = append(result.Attempts, Attempt{Target: target, Error: err})
			return result, err
		}
		result.Attempts = append(result.Attempts, Attempt{Target: target, StatusCode: resp.StatusCode})
		result.Primary = resp
		return result, nil
	}

	if err := d.replayBudget.Ensure(req); err != nil {
		return nil, err
	}

	d.dispatchAll(ctx, result, match, req, op, rw)

	if err := ctx.Err(); err != nil {
		if result.Primary != nil {
			recordCleanupError(result, "primary", s3.DrainAndCloseContext(ctx, result.Primary))
			result.Primary = nil
		}
		return result, err
	}

	failureCount := result.FailureCount()
	if failureCount > 0 {
		if result.Primary != nil && result.Primary.StatusCode < 400 {
			recordCleanupError(result, "primary", s3.DrainAndCloseContext(ctx, result.Primary))
			result.Primary = nil
		}
		return result, fmt.Errorf("fan-out had %d failures", failureCount)
	}

	return result, nil
}

type fanoutJob struct {
	index  int
	target string
}

type fanoutAttemptResult struct {
	index      int
	attempt    Attempt
	primary    *s3.Response
	cleanupErr error
	seen       bool
}

func (d *dispatcher) dispatchAll(ctx context.Context, result *Result, match router.Match, req *http.Request, op s3ops.Operation, rw rewrite.Result) {
	destinations := append([]string(nil), match.Destinations...)
	jobs := make(chan fanoutJob)
	results := make(chan fanoutAttemptResult, len(destinations))
	workers := len(destinations)
	if workers > maxParallelWriteFanout {
		workers = maxParallelWriteFanout
	}

	var wg sync.WaitGroup
	for i := 0; i < workers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for job := range jobs {
				if ctx.Err() != nil {
					continue
				}
				results <- d.dispatchAllAttempt(ctx, job.index, job.target, req, op, rw)
			}
		}()
	}

	go func() {
		defer close(jobs)
		for i, target := range destinations {
			select {
			case <-ctx.Done():
				return
			case jobs <- fanoutJob{index: i, target: target}:
			}
		}
	}()

	go func() {
		wg.Wait()
		close(results)
	}()

	ordered := make([]fanoutAttemptResult, len(destinations))
	for attemptResult := range results {
		ordered[attemptResult.index] = attemptResult
	}
	for _, attemptResult := range ordered {
		if !attemptResult.seen {
			continue
		}
		result.Attempts = append(result.Attempts, attemptResult.attempt)
		if attemptResult.primary != nil {
			result.Primary = attemptResult.primary
		}
		if attemptResult.cleanupErr != nil {
			recordCleanupError(result, attemptResult.attempt.Target, attemptResult.cleanupErr)
		}
	}
}

func (d *dispatcher) dispatchAllAttempt(ctx context.Context, index int, target string, req *http.Request, op s3ops.Operation, rw rewrite.Result) fanoutAttemptResult {
	attemptReq, err := cloneAttemptRequest(ctx, req)
	if err != nil {
		return fanoutAttemptResult{index: index, seen: true, attempt: Attempt{Target: target, Error: err}}
	}
	defer closeRequestBody(attemptReq.Body)

	resp, err := d.backend.Do(ctx, s3.Request{
		Operation: op,
		Target:    target,
		Bucket:    rw.Bucket,
		Key:       rw.Key,
		Source:    attemptReq,
	})
	if err != nil {
		return fanoutAttemptResult{index: index, seen: true, attempt: Attempt{Target: target, Error: err}}
	}

	attempt := Attempt{Target: target, StatusCode: resp.StatusCode}
	if resp.StatusCode >= 400 {
		attempt.Error = fmt.Errorf("upstream returned status %d", resp.StatusCode)
	}

	if index == 0 {
		buffered, err := s3.BufferWriteResponse(ctx, resp)
		if err != nil {
			attempt.Error = fmt.Errorf("retain primary response: %w", err)
			return fanoutAttemptResult{index: index, seen: true, attempt: attempt}
		}
		return fanoutAttemptResult{index: index, seen: true, attempt: attempt, primary: buffered}
	}

	return fanoutAttemptResult{index: index, seen: true, attempt: attempt, cleanupErr: s3.DrainAndCloseContext(ctx, resp)}
}

func cloneAttemptRequest(ctx context.Context, req *http.Request) (*http.Request, error) {
	attemptReq := req.Clone(ctx)
	if req.Body == nil || req.Body == http.NoBody {
		attemptReq.Body = http.NoBody
		attemptReq.GetBody = func() (io.ReadCloser, error) { return http.NoBody, nil }
		return attemptReq, nil
	}
	if req.GetBody == nil {
		return nil, fmt.Errorf("request body is not replayable")
	}
	body, err := req.GetBody()
	if err != nil {
		return nil, fmt.Errorf("get request body: %w", err)
	}
	attemptReq.Body = body
	attemptReq.GetBody = req.GetBody
	return attemptReq, nil
}

func closeRequestBody(body io.ReadCloser) {
	if body != nil && body != http.NoBody {
		body.Close()
	}
}

func recordCleanupError(result *Result, target string, err error) {
	if err == nil || result == nil {
		return
	}
	if result.CleanupErrors == nil {
		result.CleanupErrors = make(map[string]error)
	}
	result.CleanupErrors[target] = err
}

func (d *dispatcher) dispatchOrderedFailover(ctx context.Context, result *Result, match router.Match, req *http.Request, op s3ops.Operation, rw rewrite.Result) (*Result, error) {
	if len(match.Destinations) == 0 {
		return nil, fmt.Errorf("no destination available")
	}

	for i, target := range match.Destinations {
		resp, err := d.backend.Do(ctx, s3.Request{
			Operation: op,
			Target:    target,
			Bucket:    rw.Bucket,
			Key:       rw.Key,
			Source:    req,
		})
		if err != nil {
			result.Attempts = append(result.Attempts, Attempt{Target: target, Error: err})
			continue
		}
		attempt := Attempt{Target: target, StatusCode: resp.StatusCode}

		if resp.StatusCode >= http.StatusInternalServerError {
			attempt.Error = fmt.Errorf("upstream returned status %d", resp.StatusCode)
			result.Attempts = append(result.Attempts, attempt)
			if i < len(match.Destinations)-1 {
				recordCleanupError(result, target, s3.DrainAndCloseContext(ctx, resp))
				continue
			}
			result.Primary = resp
			continue
		}

		result.Attempts = append(result.Attempts, attempt)
		result.Primary = resp
		return result, nil
	}

	failureCount := result.FailureCount()
	if result.Primary != nil {
		return result, fmt.Errorf("ordered failover exhausted %d destinations", failureCount)
	}
	return result, fmt.Errorf("ordered failover exhausted %d destinations", failureCount)
}
