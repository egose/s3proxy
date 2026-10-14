package httpapi

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"reflect"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	v4 "github.com/aws/aws-sdk-go-v2/aws/signer/v4"
)

func signOperationHeaderRequest(t *testing.T, r *http.Request, mode string) {
	t.Helper()
	if mode == "none" {
		return
	}
	r.Header.Set("X-Amz-Content-Sha256", "UNSIGNED-PAYLOAD")
	signer := v4.NewSigner(func(o *v4.SignerOptions) { o.DisableURIPathEscaping = true })
	creds := aws.Credentials{AccessKeyID: "allowed-ak", SecretAccessKey: "allowed-sk"}
	if mode == "header" {
		if err := signer.SignHTTP(context.Background(), creds, r, "UNSIGNED-PAYLOAD", "s3", "us-east-1", time.Now().UTC()); err != nil {
			t.Fatal(err)
		}
		return
	}
	q := r.URL.Query()
	q.Set("X-Amz-Expires", "600")
	r.URL.RawQuery = q.Encode()
	uri, _, err := signer.PresignHTTP(context.Background(), creds, r, "UNSIGNED-PAYLOAD", "s3", "us-east-1", time.Now().UTC())
	if err != nil {
		t.Fatal(err)
	}
	r.URL, err = url.Parse(uri)
	if err != nil {
		t.Fatal(err)
	}
}

func TestHandler_OperationHeadersHTTP(t *testing.T) {
	for _, authCase := range []struct {
		name, mode   string
		signedMarker bool
		invalid      bool
	}{
		{name: "auth none", mode: "none"},
		{name: "signed header", mode: "header", signedMarker: true},
		{name: "filtered header", mode: "header"},
		{name: "filtered presign", mode: "presign"},
		{name: "invalid header", mode: "header", signedMarker: true, invalid: true},
		{name: "invalid presign", mode: "presign", invalid: true},
	} {
		for _, marker := range []struct{ name, header, value string }{
			{"copy", "X-Amz-Copy-Source", "/visible/source"},
			{"multipart", "X-Amz-Multipart-Upload-Id", "upload-id"},
		} {
			for _, tt := range []struct {
				name   string
				values []string
				mixed  bool
			}{
				{name: "nonempty", values: []string{marker.value}},
				{name: "empty first", values: []string{"", marker.value}},
				{name: "empty only", values: []string{""}},
				{name: "whitespace only", values: []string{" \t "}},
				{name: "duplicate empty", values: []string{"", ""}},
				{name: "mixed case", values: []string{"", marker.value}, mixed: true},
			} {
				for _, topology := range []string{"single", "fanout", "continue"} {
					t.Run(authCase.name+"/"+marker.name+"/"+tt.name+"/"+topology, func(t *testing.T) {
						f, budget, _ := payloadFixture(t, authCase.mode, topology)
						original := map[string]string{"/visible/key": "original", "/continued/key": "original replica"}
						f.mu.Lock()
						for key, value := range original {
							f.objects[key] = value
						}
						f.mu.Unlock()
						var reads atomic.Int64
						headersAfter := make(chan http.Header, 1)
						h := f.proxy.Config.Handler
						f.proxy.Config.Handler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
							want := make([]string, len(tt.values))
							for i, value := range tt.values {
								want[i] = strings.TrimSpace(value)
							}
							if got := r.Header.Values(marker.header); !reflect.DeepEqual(got, want) {
								t.Errorf("wire selector values = %q, want %q", got, want)
							}
							r.Body = queryReadCounter{r.Body, &reads}
							h.ServeHTTP(w, r)
							headersAfter <- r.Header.Clone()
						})
						r, err := http.NewRequest(http.MethodPut, f.proxy.URL+"/visible/key", strings.NewReader("replacement"))
						if err != nil {
							t.Fatal(err)
						}
						name := marker.header
						if tt.mixed {
							name = strings.Replace(name, "Amz", "aMZ", 1)
						}
						if authCase.signedMarker {
							r.Header[name] = append([]string(nil), tt.values...)
						}
						signOperationHeaderRequest(t, r, authCase.mode)
						if !authCase.signedMarker {
							r.Header[name] = append([]string(nil), tt.values...)
						}
						if authCase.invalid {
							if authCase.mode == "header" {
								r.Header.Set("Authorization", r.Header.Get("Authorization")+"0")
							} else {
								q := r.URL.Query()
								q.Set("X-Amz-Signature", strings.Repeat("0", 64))
								r.URL.RawQuery = q.Encode()
							}
						}
						resp, err := f.proxy.Client().Do(r)
						if err != nil {
							t.Fatal(err)
						}
						data, err := io.ReadAll(resp.Body)
						resp.Body.Close()
						if err != nil {
							t.Fatal(err)
						}
						after := <-headersAfter
						status, code := http.StatusNotImplemented, "NotImplemented"
						if authCase.invalid {
							status, code = http.StatusForbidden, "SignatureDoesNotMatch"
						}
						if resp.StatusCode != status || !strings.Contains(string(data), "<Code>"+code+"</Code>") {
							t.Errorf("response = %d %s, want %d %s", resp.StatusCode, data, status, code)
						}
						wantAuthCalls := int64(1)
						if authCase.mode == "none" {
							wantAuthCalls = 0
						}
						if got := f.auth.calls.Load(); got != wantAuthCalls {
							t.Errorf("auth calls = %d, want %d", got, wantAuthCalls)
						}
						_, present := after[marker.header]
						wantPresent := authCase.mode == "none" || authCase.signedMarker || authCase.invalid
						if present != wantPresent {
							t.Errorf("selector present after auth = %t, want %t", present, wantPresent)
						}
						if reads.Load() != 0 || f.router.calls.Load() != 0 || f.calls.Load() != 0 || budget.Used() != 0 {
							t.Errorf("reads/router/backend/budget = %d/%d/%d/%d, want all zero", reads.Load(), f.router.calls.Load(), f.calls.Load(), budget.Used())
						}
						f.mu.Lock()
						unchanged := reflect.DeepEqual(f.objects, original)
						f.mu.Unlock()
						if !unchanged {
							t.Error("rejected request mutated local object state")
						}
					})
				}
			}
		}
	}
}

func TestHandler_OperationHeadersPayloadHashPrecedenceHTTP(t *testing.T) {
	for _, mode := range []string{"header", "presign"} {
		for _, marker := range []string{"X-Amz-Copy-Source", "X-Amz-Multipart-Upload-Id"} {
			for _, matchingHash := range []bool{false, true} {
				t.Run(fmt.Sprintf("%s/%s/matchingHash=%t", mode, marker, matchingHash), func(t *testing.T) {
					f, budget, _ := payloadFixture(t, mode, "single")
					f.mu.Lock()
					f.objects["/visible/key"] = "original"
					f.mu.Unlock()
					var reads atomic.Int64
					h := f.proxy.Config.Handler
					f.proxy.Config.Handler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
						r.Body = queryReadCounter{r.Body, &reads}
						h.ServeHTTP(w, r)
					})
					const payload = "replacement"
					r, err := http.NewRequest(http.MethodPut, f.proxy.URL+"/visible/key", strings.NewReader(payload))
					if err != nil {
						t.Fatal(err)
					}
					signedPayload := payload
					if !matchingHash {
						signedPayload += "tampered"
					}
					signPayloadRequest(t, r, mode, signedPayload)
					r.Header[marker] = []string{"", "selector"}
					resp, err := f.proxy.Client().Do(r)
					if err != nil {
						t.Fatal(err)
					}
					data, err := io.ReadAll(resp.Body)
					resp.Body.Close()
					if err != nil {
						t.Fatal(err)
					}
					status, code := http.StatusNotImplemented, "NotImplemented"
					if !matchingHash {
						status, code = http.StatusForbidden, "AccessDenied"
					}
					if resp.StatusCode != status || !strings.Contains(string(data), "<Code>"+code+"</Code>") {
						t.Errorf("response = %d %s, want %d %s", resp.StatusCode, data, status, code)
					}
					if reads.Load() == 0 || f.auth.calls.Load() != 1 || f.router.calls.Load() != 0 || f.calls.Load() != 0 || budget.Used() != 0 {
						t.Errorf("reads/auth/router/backend/budget = %d/%d/%d/%d/%d", reads.Load(), f.auth.calls.Load(), f.router.calls.Load(), f.calls.Load(), budget.Used())
					}
					if stored, ok := f.object("/visible/key"); !ok || stored != "original" {
						t.Error("rejected request mutated local object state")
					}
				})
			}
		}
	}
}
