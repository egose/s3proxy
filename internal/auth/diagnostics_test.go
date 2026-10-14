package auth

import (
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/egose/s3proxy/internal/config"
	"github.com/egose/s3proxy/internal/replaybody"
)

func TestSigV4Static_ParserDiagnosticsValueFree(t *testing.T) {
	const ak = "AKIATEST"
	const sk = "testsecret123"
	budget := replaybody.NewBudget(1, 1)
	authenticator, err := NewAuthenticator(config.Auth{
		Mode: config.AuthModeSigV4Static,
		Clients: map[string]config.Client{
			"ci": {Name: "ci", AccessKey: ak, SecretKey: sk},
		},
	}, budget)
	if err != nil {
		t.Fatal("could not construct test authenticator")
	}

	for _, path := range []struct {
		name, field string
		presign     bool
		want        error
	}{
		{"header date", "X-Amz-Date", false, AuthError("invalid X-Amz-Date")},
		{"presign date", "X-Amz-Date", true, AuthError("invalid X-Amz-Date")},
		{"presign expires", "X-Amz-Expires", true, errInvalidExpires},
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
			for _, getBody := range []bool{false, true} {
				t.Run(fmt.Sprintf("%s/%s/getBody=%t", path.name, value.name, getBody), func(t *testing.T) {
					var r *http.Request
					if path.presign {
						r = mustPresignedHashedPayloadRequest(t, ak, sk, "payload", time.Now().UTC(), time.Minute)
						q := r.URL.Query()
						q.Set(path.field, value.value)
						r.URL.RawQuery = q.Encode()
					} else {
						r = signedHashedPayloadRequest(t, ak, sk, "payload")
						r.Header.Set(path.field, value.value)
					}
					body := &countingErrorReadCloser{err: errors.New("unexpected payload read")}
					r.Body, r.ContentLength, r.GetBody = body, -1, nil
					getCalls := 0
					if getBody {
						r.GetBody = func() (io.ReadCloser, error) {
							getCalls++
							return body, nil
						}
					}
					principal, err := authenticator.Authenticate(r)
					if principal != nil {
						t.Error("invalid authentication returned a principal")
					}
					if body.reads != 0 || getCalls != 0 || budget.Used() != 0 {
						t.Errorf("body reads/GetBody/budget = %d/%d/%d, want all zero", body.reads, getCalls, budget.Used())
					}
					assertValueFreeAuthDiagnostic(t, err, path.want, value.value)
				})
			}
		}
	}
}

func assertValueFreeAuthDiagnostic(t *testing.T, err, want error, value string) {
	t.Helper()
	if err == nil {
		t.Fatal("expected invalid authentication to fail")
	}
	wrapped := fmt.Errorf("authenticate request: %w", err)
	if err.Error() != want.Error() || !errors.Is(err, want) || !errors.Is(wrapped, want) {
		t.Error("authentication error lacks the fixed failure category or errors.Is classification")
	}
	if IsSignatureMismatch(wrapped) {
		t.Error("parser failure was classified as a signature mismatch")
	}
	var dateError *time.ParseError
	var expiresError *strconv.NumError
	if errors.As(wrapped, &dateError) || errors.As(wrapped, &expiresError) {
		t.Error("authentication error retained a value-bearing parser cause")
	}
	for current := wrapped; current != nil; current = errors.Unwrap(current) {
		text := current.Error() + fmt.Sprintf(" %s %v %+v %#v %q", current, current, current, current, current)
		for _, sentinel := range []string{value, "AUTHLOG_DATE", "AUTHLOG_EXPIRES", "雪"} {
			for _, spelling := range []string{sentinel, strconv.Quote(sentinel), strconv.QuoteToASCII(sentinel), url.QueryEscape(sentinel)} {
				if strings.Contains(text, spelling) {
					t.Error("authentication error formatting exposed invalid input")
					return
				}
			}
		}
	}
}

func TestSigV4Static_ZeroExpiryAndClockBoundaries(t *testing.T) {
	const ak = "AKIATEST"
	const sk = "testsecret123"
	base := time.Date(2026, 10, 12, 12, 0, 0, 0, time.UTC)
	authenticator, err := newTestAuthenticator(config.Auth{
		Mode: config.AuthModeSigV4Static,
		Clients: map[string]config.Client{
			"ci": {Name: "ci", AccessKey: ak, SecretKey: sk},
		},
	})
	if err != nil {
		t.Fatal("could not construct test authenticator")
	}
	for _, tt := range []struct {
		name          string
		header        bool
		signedAt, now time.Time
		expires       time.Duration
		want          error
	}{
		{name: "header exact past skew", header: true, signedAt: base.Add(-inboundSkew), now: base},
		{name: "header past skew plus nanosecond", header: true, signedAt: base.Add(-inboundSkew), now: base.Add(time.Nanosecond), want: errRequestExpired},
		{name: "header future skew plus nanosecond", header: true, signedAt: base.Add(inboundSkew), now: base.Add(-time.Nanosecond), want: errRequestExpired},
		{name: "presign exact future skew", signedAt: base.Add(inboundSkew), now: base, expires: time.Minute},
		{name: "presign future skew plus nanosecond", signedAt: base.Add(inboundSkew), now: base.Add(-time.Nanosecond), expires: time.Minute, want: errRequestExpired},
		{name: "zero expiry at signing time", signedAt: base, now: base},
		{name: "zero expiry before signing time", signedAt: base, now: base.Add(-time.Nanosecond)},
		{name: "zero expiry exact future skew", signedAt: base.Add(inboundSkew), now: base},
		{name: "zero expiry over future skew", signedAt: base.Add(inboundSkew), now: base.Add(-time.Nanosecond), want: errRequestExpired},
		{name: "zero expiry after signing time", signedAt: base, now: base.Add(time.Nanosecond), want: errPresignExpired},
		{name: "seven days exact expiry", signedAt: base.Add(-maxPresignTTL), now: base, expires: maxPresignTTL},
		{name: "seven days after expiry", signedAt: base.Add(-maxPresignTTL), now: base.Add(time.Nanosecond), expires: maxPresignTTL, want: errPresignExpired},
		{name: "above seven days", signedAt: base, now: base, expires: maxPresignTTL + time.Second, want: errInvalidExpires},
		{name: "negative expiry", signedAt: base, now: base, expires: -time.Second, want: errInvalidExpires},
	} {
		t.Run(tt.name, func(t *testing.T) {
			setAuthenticatorNow(t, authenticator, tt.now)
			var r *http.Request
			if tt.header {
				r = signedUnsignedPayloadRequestAt(t, ak, sk, tt.signedAt)
			} else {
				r = mustPresignedRequest(t, ak, sk, tt.signedAt, tt.expires)
			}
			principal, err := authenticator.Authenticate(r)
			if !errors.Is(err, tt.want) {
				t.Error("authentication result did not preserve the clock/expiry boundary")
			}
			if (principal != nil) != (tt.want == nil) {
				t.Error("principal did not match the expected authentication result")
			}
		})
	}
}
