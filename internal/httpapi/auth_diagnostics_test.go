package httpapi

import (
	"encoding/json"
	"encoding/xml"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/egose/s3proxy/internal/config"
)

func TestHandler_AuthParserDiagnosticsHTTP(t *testing.T) {
	for _, path := range []struct{ name, mode, field, cause string }{
		{"header date", "header", "X-Amz-Date", "invalid X-Amz-Date"},
		{"presign date", "presign", "X-Amz-Date", "invalid X-Amz-Date"},
		{"presign expires", "presign", "X-Amz-Expires", "invalid X-Amz-Expires"},
	} {
		values := []struct{ name, value string }{
			{"text", "AUTHLOG_DATE_SENTINEL"},
			{"quoted unicode", `AUTHLOG_DATE_SENTINEL"雪\value`},
			{"trailing data", "20261012T120000Z-AUTHLOG_DATE_SENTINEL"},
			{"invalid calendar date", "20260230T120000Z"},
			{"numeric overflow", "9223372036854775808123456789"},
		}
		if path.field == "X-Amz-Expires" {
			values = []struct{ name, value string }{
				{"text", "AUTHLOG_EXPIRES_SENTINEL"},
				{"quoted unicode", `AUTHLOG_EXPIRES_SENTINEL"雪\value`},
				{"numeric overflow", "9223372036854775808123456789"},
				{"numeric underflow", "-9223372036854775809123456789"},
				{"negative", "-123456789012345"},
				{"above seven days", "604801"},
				{"fractional", "12345.6789"},
			}
		}
		for _, value := range values {
			t.Run(path.name+"/"+value.name, func(t *testing.T) {
				f := newProbeFixture(t, config.Addressing{PathStyle: true})
				h := f.proxy.Config.Handler.(*handler)
				h.deps.Logger = slog.New(slog.NewJSONHandler(f, nil))
				f.mu.Lock()
				f.objects["/visible/key"] = "original"
				f.mu.Unlock()
				var reads, getCalls atomic.Int64
				finished := make(chan struct{})
				f.proxy.Config.Handler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					defer close(finished)
					got := r.Header.Get(path.field)
					if path.mode == "presign" {
						got = r.URL.Query().Get(path.field)
					}
					if got != value.value {
						t.Error("wire authentication value did not match the fixture")
					}
					body := queryReadCounter{r.Body, &reads}
					r.Body = body
					r.GetBody = func() (io.ReadCloser, error) {
						getCalls.Add(1)
						return body, nil
					}
					h.ServeHTTP(w, r)
				})
				const requestID = "auth-diagnostics"
				const payload = "replacement"
				r, err := http.NewRequest(http.MethodPut, f.proxy.URL+"/visible/key", strings.NewReader(payload))
				if err != nil {
					t.Fatal("could not construct HTTP request")
				}
				r.Header.Set("X-Request-Id", requestID)
				signPayloadRequest(t, r, path.mode, payload)
				if path.mode == "presign" {
					q := r.URL.Query()
					q.Set(path.field, value.value)
					r.URL.RawQuery = q.Encode()
				} else {
					r.Header.Set(path.field, value.value)
				}
				resp, err := f.proxy.Client().Do(r)
				if err != nil {
					t.Fatal("HTTP authentication request failed")
				}
				data, err := io.ReadAll(resp.Body)
				resp.Body.Close()
				if err != nil {
					t.Fatal("could not read HTTP authentication response")
				}
				<-finished
				var response struct {
					XMLName   xml.Name `xml:"Error"`
					Code      string   `xml:"Code"`
					Message   string   `xml:"Message"`
					RequestID string   `xml:"RequestId"`
				}
				if err := xml.Unmarshal(data, &response); err != nil {
					t.Error("authentication response was not valid S3 error XML")
				}
				if resp.StatusCode != http.StatusForbidden || response.Code != "AccessDenied" || response.Message != "Access Denied." {
					t.Error("invalid authentication did not retain the AccessDenied HTTP/XML mapping")
				}
				if resp.Header.Get("X-Request-Id") != requestID || response.RequestID != requestID {
					t.Error("authentication response lost the request ID")
				}
				if reads.Load() != 0 || getCalls.Load() != 0 || f.router.calls.Load() != 0 || f.calls.Load() != 0 || h.deps.ReplayBudget.Used() != 0 {
					t.Errorf("reads/GetBody/router/backend/budget = %d/%d/%d/%d/%d, want all zero", reads.Load(), getCalls.Load(), f.router.calls.Load(), f.calls.Load(), h.deps.ReplayBudget.Used())
				}
				if f.auth.calls.Load() != 1 {
					t.Errorf("authentication calls = %d, want 1", f.auth.calls.Load())
				}
				if stored, ok := f.object("/visible/key"); !ok || stored != "original" {
					t.Error("invalid authentication changed the fixture object")
				}
				f.mu.Lock()
				logs := f.logs.String()
				f.mu.Unlock()
				assertAuthHTTPDiagnostics(t, logs, data, requestID, path.cause, value.value)
			})
		}
	}
}

func assertAuthHTTPDiagnostics(t *testing.T, logs string, data []byte, requestID, cause, value string) {
	t.Helper()
	for _, sentinel := range []string{value, "AUTHLOG_DATE", "AUTHLOG_EXPIRES", "雪", "allowed-ak", "allowed-sk", "X-Amz-Signature="} {
		for _, spelling := range []string{sentinel, strconv.Quote(sentinel), strconv.QuoteToASCII(sentinel), url.QueryEscape(sentinel)} {
			if strings.Contains(logs+string(data), spelling) {
				t.Error("HTTP diagnostics exposed authentication input")
				break
			}
		}
	}
	decoder := json.NewDecoder(strings.NewReader(logs))
	var records []map[string]any
	for {
		var record map[string]any
		err := decoder.Decode(&record)
		if err == io.EOF {
			break
		}
		if err != nil {
			t.Fatal("captured authentication log was not valid JSON")
		}
		records = append(records, record)
	}
	if len(records) != 2 {
		t.Fatalf("log records = %d, want auth failure and request completion", len(records))
	}
	failure, completion := records[0], records[1]
	if failure["msg"] != "auth failed" || failure["level"] != "WARN" || failure["error"] != cause {
		t.Error("authentication log did not retain the fixed failure category")
	}
	if completion["msg"] != "request complete" || completion["level"] != "INFO" || completion["method"] != http.MethodPut || completion["status"] != float64(http.StatusForbidden) || completion["bytes"] != float64(len(data)) {
		t.Error("authentication log lost request completion metadata")
	}
	if duration, ok := completion["duration_ms"].(float64); !ok || duration < 0 {
		t.Error("authentication log lost the request duration")
	}
	for _, record := range records {
		if record["request_id"] != requestID {
			t.Error("authentication log lost the request ID")
		}
	}
}
