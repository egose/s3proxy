package requestpayload_test

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"reflect"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	v4 "github.com/aws/aws-sdk-go-v2/aws/signer/v4"
	"github.com/egose/s3proxy/internal/auth"
	"github.com/egose/s3proxy/internal/backend/s3"
	"github.com/egose/s3proxy/internal/config"
	"github.com/egose/s3proxy/internal/dispatch"
	"github.com/egose/s3proxy/internal/replaybody"
	"github.com/egose/s3proxy/internal/requestpayload"
	"github.com/egose/s3proxy/internal/rewrite"
	"github.com/egose/s3proxy/internal/router"
	"github.com/egose/s3proxy/internal/s3ops"
)

type trackedBody struct {
	*strings.Reader
	reads, closes int
}

func (b *trackedBody) Read(p []byte) (int, error) {
	b.reads++
	return b.Reader.Read(p)
}

func (b *trackedBody) Close() error {
	b.closes++
	return nil
}

func TestEntryPointsRejectBeforeIO(t *testing.T) {
	var mu sync.Mutex
	stored, calls := "original", 0
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		data, err := io.ReadAll(r.Body)
		if err != nil {
			t.Error(err)
		}
		mu.Lock()
		stored, calls = string(data), calls+1
		mu.Unlock()
	}))
	defer upstream.Close()
	endpoint, err := url.Parse(upstream.URL)
	if err != nil {
		t.Fatal(err)
	}
	for _, entry := range []string{"header verifier", "presign verifier", "dispatcher first", "dispatcher fanout", "executor"} {
		for _, marker := range []http.Header{
			{"content-ENCODING": {"gzip", " AWS-CHUNKED "}},
			{"x-AMZ-content-sha256": {"UNSIGNED-PAYLOAD", " streaming-SECRET "}},
			{"x-AMZ-trailer": {"", " x-amz-checksum-SECRET "}},
			{"Content-Encoding": {"aws-chunked"}, "X-Amz-Content-Sha256": {"STREAMING-SECRET"}, "X-Amz-Trailer": {"SECRET"}},
		} {
			for _, length := range []int64{-1, 7} {
				for _, getBody := range []bool{false, true} {
					t.Run(fmt.Sprintf("%s/%v/length=%d/getBody=%t", entry, marker, length, getBody), func(t *testing.T) {
						budget := replaybody.NewBudget(1, 1)
						backend, err := s3.NewClient(upstream.Client(), map[string]config.S3Target{"store": {EndpointURL: endpoint, Region: "us-east-1", ForcePathStyle: true, Credentials: config.StaticCredential{AccessKey: "backend-ak", SecretKey: "backend-sk"}}}, budget)
						if err != nil {
							t.Fatal(err)
						}
						d, err := dispatch.New(backend, budget)
						if err != nil {
							t.Fatal(err)
						}
						a, err := auth.NewAuthenticator(config.Auth{Mode: config.AuthModeSigV4Static, Clients: map[string]config.Client{"client": {AccessKey: "client-ak", SecretKey: "client-sk"}}}, budget)
						if err != nil {
							t.Fatal(err)
						}
						r := httptest.NewRequest("PUT", "http://proxy/visible/key", nil)
						body := &trackedBody{Reader: strings.NewReader("payload")}
						r.Body, r.ContentLength = body, length
						getCalls := 0
						if getBody {
							r.GetBody = func() (io.ReadCloser, error) {
								getCalls++
								return io.NopCloser(strings.NewReader("payload")), nil
							}
						}
						signer := v4.NewSigner(func(o *v4.SignerOptions) { o.DisableURIPathEscaping = true })
						creds := aws.Credentials{AccessKeyID: "client-ak", SecretAccessKey: "client-sk"}
						if entry == "presign verifier" {
							r.URL.RawQuery = "X-Amz-Expires=600"
							uri, _, err := signer.PresignHTTP(context.Background(), creds, r, "UNSIGNED-PAYLOAD", "s3", "us-east-1", time.Now().UTC())
							if err != nil {
								t.Fatal(err)
							}
							r.URL, err = url.Parse(uri)
							if err != nil {
								t.Fatal(err)
							}
						} else if err := signer.SignHTTP(context.Background(), creds, r, "UNSIGNED-PAYLOAD", "s3", "us-east-1", time.Now().UTC()); err != nil {
							t.Fatal(err)
						}
						for name, values := range marker {
							r.Header[name] = append([]string(nil), values...)
						}
						before := r.Header.Clone()
						beforeURL := r.URL.String()
						switch entry {
						case "header verifier", "presign verifier":
							principal, verifyErr := a.Authenticate(r)
							err = verifyErr
							if principal != nil {
								t.Fatal("partial principal returned")
							}
						case "dispatcher first", "dispatcher fanout":
							mode := config.DispatchFirst
							if entry == "dispatcher fanout" {
								mode = config.DispatchAll
							}
							result, dispatchErr := d.Dispatch(r.Context(), router.Match{Route: config.Route{Dispatch: mode}, Destinations: []string{"store", "store"}}, r, s3ops.OpPutObject, rewrite.Result{Bucket: "visible", Key: "key"})
							err = dispatchErr
							if result != nil {
								t.Fatal("partial dispatch result returned")
							}
						case "executor":
							response, backendErr := backend.Do(r.Context(), s3.Request{Source: r, Target: "store", Bucket: "visible", Key: "key", Operation: s3ops.OpPutObject})
							err = backendErr
							if response != nil {
								t.Fatal("partial backend response returned")
							}
						}
						if err != requestpayload.ErrUnsupported || strings.Contains(err.Error(), "SECRET") {
							t.Fatalf("error=%v", err)
						}
						if body.reads != 0 || body.closes != 0 || getCalls != 0 || budget.Used() != 0 || r.Body != body || (r.GetBody != nil) != getBody || r.ContentLength != length || !reflect.DeepEqual(before, r.Header) || r.URL.String() != beforeURL {
							t.Fatalf("source changed or I/O occurred: reads=%d closes=%d getBody=%d budget=%d", body.reads, body.closes, getCalls, budget.Used())
						}
						data, err := io.ReadAll(body)
						if err != nil || string(data) != "payload" {
							t.Fatal("caller can no longer read original bytes")
						}
						body.Close()
						mu.Lock()
						defer mu.Unlock()
						if calls != 0 || stored != "original" {
							t.Fatalf("backend calls=%d stored=%q", calls, stored)
						}
					})
				}
			}
		}
	}
}
