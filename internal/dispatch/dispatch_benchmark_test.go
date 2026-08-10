package dispatch

import (
	"context"
	"net/http"
	"strings"
	"testing"

	"github.com/egose/s3proxy/internal/backend/s3"
	"github.com/egose/s3proxy/internal/config"
	"github.com/egose/s3proxy/internal/rewrite"
	"github.com/egose/s3proxy/internal/router"
	"github.com/egose/s3proxy/internal/s3ops"
)

func BenchmarkDispatchFanoutSerial(b *testing.B) {
	match := router.Match{
		Route:        config.Route{Dispatch: config.DispatchAll, DestinationRefs: []string{"primary", "replica"}},
		Destinations: []config.S3Target{{Name: "primary"}, {Name: "replica"}},
	}
	rw := rewrite.Result{Bucket: "bucket", Key: "key"}

	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		backend := &benchmarkBackend{}
		d := &dispatcher{backend: backend}
		req, err := http.NewRequest(http.MethodPut, "http://proxy.local/bucket/key", strings.NewReader("payload"))
		if err != nil {
			b.Fatal(err)
		}
		req.ContentLength = int64(len("payload"))
		result, err := d.Dispatch(context.Background(), match, req, s3ops.OpPutObject, rw)
		if err != nil {
			b.Fatal(err)
		}
		s3.DrainAndClose(result.Primary)
	}
}

type benchmarkBackend struct{}

func (b *benchmarkBackend) Do(context.Context, s3.Request) (*s3.Response, error) {
	return responseWithStatus(http.StatusOK), nil
}
