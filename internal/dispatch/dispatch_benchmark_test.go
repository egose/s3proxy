package dispatch

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"testing"
	"time"

	"github.com/egose/s3proxy/internal/backend/s3"
	"github.com/egose/s3proxy/internal/config"
	"github.com/egose/s3proxy/internal/replaybody"
	"github.com/egose/s3proxy/internal/rewrite"
	"github.com/egose/s3proxy/internal/router"
	"github.com/egose/s3proxy/internal/s3ops"
)

func BenchmarkDispatchFanoutDelayedSerial(b *testing.B) {
	benchmarkDelayedFanout(b, false)
}

func BenchmarkDispatchFanoutDelayedParallel(b *testing.B) {
	benchmarkDelayedFanout(b, true)
}

func benchmarkDelayedFanout(b *testing.B, parallel bool) {
	for _, size := range []struct {
		name string
		body int
	}{
		{name: "Zero", body: 0},
		{name: "Small", body: 1024},
		{name: "Max", body: int(replaybody.DefaultMaxBytes)},
	} {
		b.Run(size.name, func(b *testing.B) {
			payload := bytes.Repeat([]byte("x"), size.body)
			match := router.Match{
				Route:        config.Route{Dispatch: config.DispatchAll, DestinationRefs: []string{"primary", "replica"}},
				Destinations: []string{"primary", "replica"},
			}
			rw := rewrite.Result{Bucket: "bucket", Key: "key"}
			b.ReportAllocs()
			b.SetBytes(int64(size.body))
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				backend := &benchmarkDelayedBackend{delay: time.Millisecond}
				d := &dispatcher{backend: backend, replayBudget: replaybody.NewBudget(replaybody.DefaultMaxBytes, replaybody.DefaultAggregateMaxBytes)}
				req, err := http.NewRequest(http.MethodPut, "http://proxy.local/bucket/key", bytes.NewReader(payload))
				if err != nil {
					b.Fatal(err)
				}
				req.ContentLength = int64(len(payload))
				req.GetBody = nil
				var result *Result
				if parallel {
					result, err = d.Dispatch(context.Background(), match, req, s3ops.OpPutObject, rw)
				} else {
					result, err = dispatchAllSerialBenchmark(context.Background(), d, match, req, s3ops.OpPutObject, rw)
				}
				if err != nil {
					b.Fatal(err)
				}
				s3.DrainAndClose(result.Primary)
				replaybody.Release(req)
			}
		})
	}
}

func dispatchAllSerialBenchmark(ctx context.Context, d *dispatcher, match router.Match, req *http.Request, op s3ops.Operation, rw rewrite.Result) (*Result, error) {
	result := &Result{}
	if err := d.replayBudget.Ensure(req); err != nil {
		return nil, err
	}
	for i, target := range match.Destinations {
		attemptResult := d.dispatchAllAttempt(ctx, i, target, req, op, rw)
		result.Attempts = append(result.Attempts, attemptResult.attempt)
		if attemptResult.primary != nil {
			result.Primary = attemptResult.primary
		}
		if attemptResult.cleanupErr != nil {
			recordCleanupError(result, attemptResult.attempt.Target, attemptResult.cleanupErr)
		}
	}
	if failureCount := result.FailureCount(); failureCount > 0 {
		if result.Primary != nil && result.Primary.StatusCode < 400 {
			recordCleanupError(result, "primary", s3.DrainAndCloseContext(ctx, result.Primary))
			result.Primary = nil
		}
		return result, fmt.Errorf("fan-out had %d failures", failureCount)
	}
	return result, nil
}

type benchmarkDelayedBackend struct {
	delay time.Duration
}

func (b *benchmarkDelayedBackend) Do(ctx context.Context, req s3.Request) (*s3.Response, error) {
	if req.Source.Body != nil {
		if _, err := io.Copy(io.Discard, req.Source.Body); err != nil {
			return nil, err
		}
	}
	select {
	case <-time.After(b.delay):
	case <-ctx.Done():
		return nil, ctx.Err()
	}
	return responseWithStatus(http.StatusOK), nil
}
