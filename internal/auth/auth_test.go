package auth

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	v4 "github.com/aws/aws-sdk-go-v2/aws/signer/v4"
	"github.com/egose/s3proxy/internal/config"
	"github.com/egose/s3proxy/internal/s3ops"
)

func v4signer() *v4.Signer {
	return v4.NewSigner()
}

func TestNoneAuthenticator(t *testing.T) {
	authenticator, err := NewAuthenticator(config.Auth{Mode: config.AuthModeNone})
	if err != nil {
		t.Fatal(err)
	}
	p, err := authenticator.Authenticate(&http.Request{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if p != nil {
		t.Error("expected nil principal for none auth")
	}
}

func TestSigV4Static_UnknownAccessKey(t *testing.T) {
	cfg := config.Auth{
		Mode: config.AuthModeSigV4Static,
		Clients: map[string]config.Client{
			"ci": {Name: "ci", AccessKey: "AKIACI", SecretKey: "s"},
		},
	}
	authenticator, err := NewAuthenticator(cfg)
	if err != nil {
		t.Fatal(err)
	}
	r := httptest.NewRequest("GET", "/bucket/key", nil)
	r.Header.Set("Authorization", "AWS4-HMAC-SHA256 Credential=UNKNOWNAK/20240101/us-east-1/s3/aws4_request, ...")
	_, err = authenticator.Authenticate(r)
	if err == nil {
		t.Fatal("expected error for unknown access key")
	}
}

func TestSigV4Static_ValidHeader(t *testing.T) {
	const ak = "AKIATEST"
	const sk = "testsecret123"
	cfg := config.Auth{
		Mode: config.AuthModeSigV4Static,
		Clients: map[string]config.Client{
			"ci": {Name: "ci", AccessKey: ak, SecretKey: sk},
		},
	}
	authenticator, err := NewAuthenticator(cfg)
	if err != nil {
		t.Fatal(err)
	}

	r := httptest.NewRequest("GET", "/bucket/key", nil)
	r.Host = "s3proxy.example.com"
	r.Header.Set("X-Amz-Content-Sha256", "UNSIGNED-PAYLOAD")

	signer := v4signer()
	creds := aws.Credentials{AccessKeyID: ak, SecretAccessKey: sk}
	if err := signer.SignHTTP(context.Background(), creds, r, "UNSIGNED-PAYLOAD", "s3", "us-east-1", time.Now().UTC()); err != nil {
		t.Fatalf("signing failed: %v", err)
	}

	p, err := authenticator.Authenticate(r)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if p.Name != "ci" {
		t.Errorf("expected principal name 'ci', got %q", p.Name)
	}
}

func TestSigV4Static_StripsUnsignedControlHeaders(t *testing.T) {
	const ak = "AKIATEST"
	const sk = "testsecret123"
	authenticator, err := NewAuthenticator(config.Auth{
		Mode: config.AuthModeSigV4Static,
		Clients: map[string]config.Client{
			"ci": {Name: "ci", AccessKey: ak, SecretKey: sk},
		},
	})
	if err != nil {
		t.Fatal(err)
	}

	r := httptest.NewRequest("GET", "/bucket/key", nil)
	r.Host = "s3proxy.example.com"
	r.Header.Set("X-Amz-Content-Sha256", "UNSIGNED-PAYLOAD")
	if err := v4signer().SignHTTP(context.Background(), aws.Credentials{AccessKeyID: ak, SecretAccessKey: sk}, r, "UNSIGNED-PAYLOAD", "s3", "us-east-1", time.Now().UTC()); err != nil {
		t.Fatalf("signing failed: %v", err)
	}
	r.Header.Set("X-Amz-Acl", "public-read")
	r.Header.Set("X-Amz-Tagging", "a=b")
	r.Header.Set("X-Amz-Meta-Owner", "mallory")
	r.Header.Set("X-Amz-Storage-Class", "GLACIER")
	r.Header.Set("If-Match", "etag")
	r.Header.Set("Range", "bytes=0-1")

	if _, err := authenticator.Authenticate(r); err != nil {
		t.Fatalf("Authenticate failed: %v", err)
	}
	for _, name := range []string{"X-Amz-Acl", "X-Amz-Tagging", "X-Amz-Meta-Owner", "X-Amz-Storage-Class", "If-Match"} {
		if got := r.Header.Get(name); got != "" {
			t.Fatalf("expected %s to be stripped, got %q", name, got)
		}
	}
	if got := r.Header.Get("Range"); got != "bytes=0-1" {
		t.Fatalf("Range = %q, want bytes=0-1", got)
	}
}

func TestSigV4Static_PreservesSignedControlHeaders(t *testing.T) {
	const ak = "AKIATEST"
	const sk = "testsecret123"
	authenticator, err := NewAuthenticator(config.Auth{
		Mode: config.AuthModeSigV4Static,
		Clients: map[string]config.Client{
			"ci": {Name: "ci", AccessKey: ak, SecretKey: sk},
		},
	})
	if err != nil {
		t.Fatal(err)
	}

	r := httptest.NewRequest("PUT", "/bucket/key", nil)
	r.Host = "s3proxy.example.com"
	r.Header.Set("X-Amz-Content-Sha256", "UNSIGNED-PAYLOAD")
	r.Header.Set("X-Amz-Acl", "private")
	r.Header.Set("X-Amz-Tagging", "a=b")
	r.Header.Set("X-Amz-Meta-Owner", "alice")
	r.Header.Set("X-Amz-Storage-Class", "STANDARD_IA")
	if err := v4signer().SignHTTP(context.Background(), aws.Credentials{AccessKeyID: ak, SecretAccessKey: sk}, r, "UNSIGNED-PAYLOAD", "s3", "us-east-1", time.Now().UTC()); err != nil {
		t.Fatalf("signing failed: %v", err)
	}

	if _, err := authenticator.Authenticate(r); err != nil {
		t.Fatalf("Authenticate failed: %v", err)
	}
	for name, want := range map[string]string{
		"X-Amz-Acl":           "private",
		"X-Amz-Tagging":       "a=b",
		"X-Amz-Meta-Owner":    "alice",
		"X-Amz-Storage-Class": "STANDARD_IA",
	} {
		if got := r.Header.Get(name); got != want {
			t.Fatalf("%s = %q, want %q", name, got, want)
		}
	}
}

func TestSigV4Static_RejectsMissingUnsignedAndInvalidScopeDates(t *testing.T) {
	const ak = "AKIATEST"
	const sk = "testsecret123"
	authenticator, err := NewAuthenticator(config.Auth{
		Mode: config.AuthModeSigV4Static,
		Clients: map[string]config.Client{
			"ci": {Name: "ci", AccessKey: ak, SecretKey: sk},
		},
	})
	if err != nil {
		t.Fatal(err)
	}

	t.Run("missing date", func(t *testing.T) {
		r := signedUnsignedPayloadRequest(t, ak, sk)
		r.Header.Del("X-Amz-Date")
		if _, err := authenticator.Authenticate(r); !errors.Is(err, errMissingDate) {
			t.Fatalf("err = %v, want %v", err, errMissingDate)
		}
	})

	t.Run("unsigned date", func(t *testing.T) {
		r := signedUnsignedPayloadRequest(t, ak, sk)
		auth := r.Header.Get("Authorization")
		auth = strings.Replace(auth, ";x-amz-date", "", 1)
		r.Header.Set("Authorization", auth)
		if _, err := authenticator.Authenticate(r); !errors.Is(err, errUnsignedDate) {
			t.Fatalf("err = %v, want %v", err, errUnsignedDate)
		}
	})

	t.Run("scope date mismatch", func(t *testing.T) {
		r := signedUnsignedPayloadRequest(t, ak, sk)
		auth := r.Header.Get("Authorization")
		auth = strings.Replace(auth, "/"+r.Header.Get("X-Amz-Date")[:8]+"/", "/19990101/", 1)
		r.Header.Set("Authorization", auth)
		if _, err := authenticator.Authenticate(r); !errors.Is(err, errInvalidCredential) {
			t.Fatalf("err = %v, want %v", err, errInvalidCredential)
		}
	})

	t.Run("wrong service", func(t *testing.T) {
		r := signedUnsignedPayloadRequest(t, ak, sk)
		r.Header.Set("Authorization", strings.Replace(r.Header.Get("Authorization"), "/s3/aws4_request", "/ec2/aws4_request", 1))
		if _, err := authenticator.Authenticate(r); !errors.Is(err, errInvalidCredential) {
			t.Fatalf("err = %v, want %v", err, errInvalidCredential)
		}
	})

	t.Run("wrong terminator", func(t *testing.T) {
		r := signedUnsignedPayloadRequest(t, ak, sk)
		r.Header.Set("Authorization", strings.Replace(r.Header.Get("Authorization"), "/s3/aws4_request", "/s3/not_request", 1))
		if _, err := authenticator.Authenticate(r); !errors.Is(err, errInvalidCredential) {
			t.Fatalf("err = %v, want %v", err, errInvalidCredential)
		}
	})
}

func TestSigV4Static_VerifiesExplicitPayloadHash(t *testing.T) {
	const ak = "AKIATEST"
	const sk = "testsecret123"
	authenticator, err := NewAuthenticator(config.Auth{
		Mode: config.AuthModeSigV4Static,
		Clients: map[string]config.Client{
			"ci": {Name: "ci", AccessKey: ak, SecretKey: sk},
		},
	})
	if err != nil {
		t.Fatal(err)
	}

	r := signedHashedPayloadRequest(t, ak, sk, "payload")
	if _, err := authenticator.Authenticate(r); err != nil {
		t.Fatalf("Authenticate matching body failed: %v", err)
	}

	r = signedHashedPayloadRequest(t, ak, sk, "payload")
	r.Body = io.NopCloser(strings.NewReader("changed"))
	r.GetBody = func() (io.ReadCloser, error) {
		return io.NopCloser(strings.NewReader("changed")), nil
	}
	if _, err := authenticator.Authenticate(r); !errors.Is(err, errPayloadHashMismatch) {
		t.Fatalf("err = %v, want %v", err, errPayloadHashMismatch)
	}
}

func TestSigV4Static_PresignedVerifiesExplicitPayloadHash(t *testing.T) {
	const ak = "AKIATEST"
	const sk = "testsecret123"
	authenticator, err := NewAuthenticator(config.Auth{
		Mode: config.AuthModeSigV4Static,
		Clients: map[string]config.Client{
			"ci": {Name: "ci", AccessKey: ak, SecretKey: sk},
		},
	})
	if err != nil {
		t.Fatal(err)
	}

	r := mustPresignedHashedPayloadRequest(t, ak, sk, "payload", time.Now().UTC(), 10*time.Minute)
	if _, err := authenticator.Authenticate(r); err != nil {
		t.Fatalf("Authenticate matching body failed: %v", err)
	}

	r = mustPresignedHashedPayloadRequest(t, ak, sk, "payload", time.Now().UTC(), 10*time.Minute)
	r.Body = io.NopCloser(strings.NewReader("changed"))
	r.GetBody = func() (io.ReadCloser, error) {
		return io.NopCloser(strings.NewReader("changed")), nil
	}
	if _, err := authenticator.Authenticate(r); !errors.Is(err, errPayloadHashMismatch) {
		t.Fatalf("err = %v, want %v", err, errPayloadHashMismatch)
	}

	r = mustPresignedRequest(t, ak, sk, time.Now().UTC(), 10*time.Minute)
	sum := sha256.Sum256([]byte("payload"))
	r.Header.Set("X-Amz-Content-Sha256", hex.EncodeToString(sum[:]))
	if _, err := authenticator.Authenticate(r); !errors.Is(err, errUnsignedPayloadHash) {
		t.Fatalf("err = %v, want %v", err, errUnsignedPayloadHash)
	}
}

func TestSigV4Static_ValidHeaderWithReorderedFields(t *testing.T) {
	const ak = "AKIATEST"
	const sk = "testsecret123"
	cfg := config.Auth{
		Mode: config.AuthModeSigV4Static,
		Clients: map[string]config.Client{
			"ci": {Name: "ci", AccessKey: ak, SecretKey: sk},
		},
	}
	authenticator, err := NewAuthenticator(cfg)
	if err != nil {
		t.Fatal(err)
	}

	r := httptest.NewRequest("GET", "/bucket/key", nil)
	r.Host = "s3proxy.example.com"
	r.Header.Set("X-Amz-Content-Sha256", "UNSIGNED-PAYLOAD")

	signer := v4signer()
	creds := aws.Credentials{AccessKeyID: ak, SecretAccessKey: sk}
	if err := signer.SignHTTP(context.Background(), creds, r, "UNSIGNED-PAYLOAD", "s3", "us-east-1", time.Now().UTC()); err != nil {
		t.Fatalf("signing failed: %v", err)
	}

	accessKey, scope, signedHeaders, signature, err := parseAuthHeader(r.Header.Get("Authorization"))
	if err != nil {
		t.Fatalf("parseAuthHeader failed: %v", err)
	}
	r.Header.Set("Authorization", "AWS4-HMAC-SHA256 Signature="+signature+", SignedHeaders="+strings.Join(signedHeaders, ";")+", Credential="+accessKey+"/"+scope)

	p, err := authenticator.Authenticate(r)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if p == nil || p.Name != "ci" {
		t.Fatalf("unexpected principal: %#v", p)
	}
}

func TestSigV4Static_BadSignature(t *testing.T) {
	const ak = "AKIATEST"
	const sk = "testsecret123"
	cfg := config.Auth{
		Mode: config.AuthModeSigV4Static,
		Clients: map[string]config.Client{
			"ci": {Name: "ci", AccessKey: ak, SecretKey: sk},
		},
	}
	authenticator, err := NewAuthenticator(cfg)
	if err != nil {
		t.Fatal(err)
	}

	r := httptest.NewRequest("GET", "/bucket/key", nil)
	r.Host = "s3proxy.example.com"
	r.Header.Set("X-Amz-Content-Sha256", "UNSIGNED-PAYLOAD")

	signer := v4signer()
	goodCreds := aws.Credentials{AccessKeyID: ak, SecretAccessKey: sk}
	if err := signer.SignHTTP(context.Background(), goodCreds, r, "UNSIGNED-PAYLOAD", "s3", "us-east-1", time.Now().UTC()); err != nil {
		t.Fatalf("signing failed: %v", err)
	}

	originalAuth := r.Header.Get("Authorization")
	tampered := originalAuth[:len(originalAuth)-2] + "XX"
	r.Header.Set("Authorization", tampered)

	_, err = authenticator.Authenticate(r)
	if err == nil {
		t.Fatal("expected error for tampered signature")
	}
}

func TestSigV4Static_ValidHeaderDoesNotReadBody(t *testing.T) {
	const ak = "AKIATEST"
	const sk = "testsecret123"
	cfg := config.Auth{
		Mode: config.AuthModeSigV4Static,
		Clients: map[string]config.Client{
			"ci": {Name: "ci", AccessKey: ak, SecretKey: sk},
		},
	}
	authenticator, err := NewAuthenticator(cfg)
	if err != nil {
		t.Fatal(err)
	}

	r := httptest.NewRequest(http.MethodPut, "/bucket/key", nil)
	r.Host = "s3proxy.example.com"
	r.Header.Set("X-Amz-Content-Sha256", "UNSIGNED-PAYLOAD")
	r.Header.Set("Content-Length", "7")
	r.ContentLength = 7

	signer := v4signer()
	creds := aws.Credentials{AccessKeyID: ak, SecretAccessKey: sk}
	if err := signer.SignHTTP(context.Background(), creds, r, "UNSIGNED-PAYLOAD", "s3", "us-east-1", time.Now().UTC()); err != nil {
		t.Fatalf("signing failed: %v", err)
	}

	r.Body = errorReadCloser{err: errors.New("body should not be read")}
	r.GetBody = nil

	p, err := authenticator.Authenticate(r)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if p == nil || p.Name != "ci" {
		t.Fatalf("unexpected principal: %#v", p)
	}
}

func TestSigV4Static_PresignedQueryExpired(t *testing.T) {
	const ak = "AKIATEST"
	const sk = "testsecret123"
	cfg := config.Auth{
		Mode: config.AuthModeSigV4Static,
		Clients: map[string]config.Client{
			"ci": {Name: "ci", AccessKey: ak, SecretKey: sk},
		},
	}
	authenticator, err := NewAuthenticator(cfg)
	if err != nil {
		t.Fatal(err)
	}

	signedAt := time.Now().UTC().Add(-2 * time.Minute)
	req := mustPresignedRequest(t, ak, sk, signedAt, time.Minute)

	_, err = authenticator.Authenticate(req)
	if err == nil {
		t.Fatal("expected expired presigned request to fail")
	}
}

func TestSigV4Static_PresignedQueryValid(t *testing.T) {
	const ak = "AKIATEST"
	const sk = "testsecret123"
	cfg := config.Auth{
		Mode: config.AuthModeSigV4Static,
		Clients: map[string]config.Client{
			"ci": {Name: "ci", AccessKey: ak, SecretKey: sk},
		},
	}
	authenticator, err := NewAuthenticator(cfg)
	if err != nil {
		t.Fatal(err)
	}

	req := mustPresignedRequest(t, ak, sk, time.Now().UTC(), 10*time.Minute)

	p, err := authenticator.Authenticate(req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if p == nil || p.Name != "ci" {
		t.Fatalf("unexpected principal: %#v", p)
	}
}

func TestSigV4Static_PresignedQueryIgnoresUnsignedHeaders(t *testing.T) {
	const ak = "AKIATEST"
	const sk = "testsecret123"
	cfg := config.Auth{
		Mode: config.AuthModeSigV4Static,
		Clients: map[string]config.Client{
			"ci": {Name: "ci", AccessKey: ak, SecretKey: sk},
		},
	}
	authenticator, err := NewAuthenticator(cfg)
	if err != nil {
		t.Fatal(err)
	}

	req := mustPresignedRequest(t, ak, sk, time.Now().UTC(), 10*time.Minute)
	req.Header.Set("Range", "bytes=0-1")

	p, err := authenticator.Authenticate(req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if p == nil || p.Name != "ci" {
		t.Fatalf("unexpected principal: %#v", p)
	}
}

func TestSigV4Static_PresignedQueryLongLivedValid(t *testing.T) {
	const ak = "AKIATEST"
	const sk = "testsecret123"
	cfg := config.Auth{
		Mode: config.AuthModeSigV4Static,
		Clients: map[string]config.Client{
			"ci": {Name: "ci", AccessKey: ak, SecretKey: sk},
		},
	}
	authenticator, err := NewAuthenticator(cfg)
	if err != nil {
		t.Fatal(err)
	}

	req := mustPresignedRequest(t, ak, sk, time.Now().UTC().Add(-20*time.Minute), time.Hour)

	p, err := authenticator.Authenticate(req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if p == nil || p.Name != "ci" {
		t.Fatalf("unexpected principal: %#v", p)
	}
}

func TestSigV4Static_PresignedQueryRejectsTooLongExpiry(t *testing.T) {
	const ak = "AKIATEST"
	const sk = "testsecret123"
	cfg := config.Auth{
		Mode: config.AuthModeSigV4Static,
		Clients: map[string]config.Client{
			"ci": {Name: "ci", AccessKey: ak, SecretKey: sk},
		},
	}
	authenticator, err := NewAuthenticator(cfg)
	if err != nil {
		t.Fatal(err)
	}

	req := mustPresignedRequest(t, ak, sk, time.Now().UTC(), 8*24*time.Hour)

	_, err = authenticator.Authenticate(req)
	if err == nil {
		t.Fatal("expected presigned request with too-long expiry to fail")
	}
}

func TestSigV4Static_DeterministicSkewAndExpiryBoundaries(t *testing.T) {
	const ak = "AKIATEST"
	const sk = "testsecret123"
	base := time.Date(2026, 8, 9, 12, 0, 0, 0, time.UTC)
	authenticator, err := NewAuthenticator(config.Auth{
		Mode: config.AuthModeSigV4Static,
		Clients: map[string]config.Client{
			"ci": {Name: "ci", AccessKey: ak, SecretKey: sk},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	setAuthenticatorNow(t, authenticator, base)

	t.Run("header exact future skew valid", func(t *testing.T) {
		r := signedUnsignedPayloadRequestAt(t, ak, sk, base.Add(inboundSkew))
		if _, err := authenticator.Authenticate(r); err != nil {
			t.Fatalf("Authenticate failed: %v", err)
		}
	})

	t.Run("header over future skew expired", func(t *testing.T) {
		r := signedUnsignedPayloadRequestAt(t, ak, sk, base.Add(inboundSkew+time.Second))
		if _, err := authenticator.Authenticate(r); !errors.Is(err, errRequestExpired) {
			t.Fatalf("err = %v, want %v", err, errRequestExpired)
		}
	})

	t.Run("presign exact expiry valid", func(t *testing.T) {
		req := mustPresignedRequest(t, ak, sk, base.Add(-time.Minute), time.Minute)
		if _, err := authenticator.Authenticate(req); err != nil {
			t.Fatalf("Authenticate failed: %v", err)
		}
	})

	t.Run("presign over expiry expired", func(t *testing.T) {
		setAuthenticatorNow(t, authenticator, base.Add(time.Nanosecond))
		req := mustPresignedRequest(t, ak, sk, base.Add(-time.Minute), time.Minute)
		if _, err := authenticator.Authenticate(req); !errors.Is(err, errPresignExpired) {
			t.Fatalf("err = %v, want %v", err, errPresignExpired)
		}
	})
}

func mustPresignedRequest(t *testing.T, ak, sk string, signedAt time.Time, expires time.Duration) *http.Request {
	t.Helper()
	req := httptest.NewRequest(http.MethodGet, "https://proxy.example.com/bucket/key", nil)
	query := req.URL.Query()
	query.Set("X-Amz-Expires", strconv.FormatInt(int64(expires/time.Second), 10))
	req.URL.RawQuery = query.Encode()

	signedURI, _, err := v4signer().PresignHTTP(
		context.Background(),
		aws.Credentials{AccessKeyID: ak, SecretAccessKey: sk},
		req,
		"UNSIGNED-PAYLOAD",
		"s3",
		"us-east-1",
		signedAt,
	)
	if err != nil {
		t.Fatalf("presigning failed: %v", err)
	}
	parsed, err := url.Parse(signedURI)
	if err != nil {
		t.Fatalf("parse signed URI: %v", err)
	}
	req.URL = parsed
	req.Host = parsed.Host
	return req
}

func mustPresignedHashedPayloadRequest(t *testing.T, ak, sk, body string, signedAt time.Time, expires time.Duration) *http.Request {
	t.Helper()
	sum := sha256.Sum256([]byte(body))
	payloadHash := hex.EncodeToString(sum[:])
	req := httptest.NewRequest(http.MethodPut, "https://proxy.example.com/bucket/key", strings.NewReader(body))
	req.GetBody = func() (io.ReadCloser, error) {
		return io.NopCloser(strings.NewReader(body)), nil
	}
	req.Header.Set("X-Amz-Content-Sha256", payloadHash)
	query := req.URL.Query()
	query.Set("X-Amz-Expires", strconv.FormatInt(int64(expires/time.Second), 10))
	req.URL.RawQuery = query.Encode()

	signedURI, _, err := v4signer().PresignHTTP(
		context.Background(),
		aws.Credentials{AccessKeyID: ak, SecretAccessKey: sk},
		req,
		payloadHash,
		"s3",
		"us-east-1",
		signedAt,
	)
	if err != nil {
		t.Fatalf("presigning failed: %v", err)
	}
	parsed, err := url.Parse(signedURI)
	if err != nil {
		t.Fatalf("parse signed URI: %v", err)
	}
	req.URL = parsed
	req.Host = parsed.Host
	return req
}

func signedUnsignedPayloadRequest(t *testing.T, ak, sk string) *http.Request {
	t.Helper()
	return signedUnsignedPayloadRequestAt(t, ak, sk, time.Now().UTC())
}

func signedUnsignedPayloadRequestAt(t *testing.T, ak, sk string, signedAt time.Time) *http.Request {
	t.Helper()
	r := httptest.NewRequest("GET", "/bucket/key", nil)
	r.Host = "s3proxy.example.com"
	r.Header.Set("X-Amz-Content-Sha256", "UNSIGNED-PAYLOAD")
	if err := v4signer().SignHTTP(context.Background(), aws.Credentials{AccessKeyID: ak, SecretAccessKey: sk}, r, "UNSIGNED-PAYLOAD", "s3", "us-east-1", signedAt); err != nil {
		t.Fatalf("signing failed: %v", err)
	}
	return r
}

func setAuthenticatorNow(t *testing.T, authenticator Authenticator, now time.Time) {
	t.Helper()
	a, ok := authenticator.(*sigv4StaticAuthenticator)
	if !ok {
		t.Fatalf("unexpected authenticator type %T", authenticator)
	}
	a.verifier.now = func() time.Time { return now }
}

func signedHashedPayloadRequest(t *testing.T, ak, sk, body string) *http.Request {
	t.Helper()
	sum := sha256.Sum256([]byte(body))
	payloadHash := hex.EncodeToString(sum[:])
	r := httptest.NewRequest(http.MethodPut, "/bucket/key", bytes.NewReader([]byte(body)))
	r.Host = "s3proxy.example.com"
	r.Header.Set("X-Amz-Content-Sha256", payloadHash)
	r.GetBody = func() (io.ReadCloser, error) {
		return io.NopCloser(strings.NewReader(body)), nil
	}
	if err := v4signer().SignHTTP(context.Background(), aws.Credentials{AccessKeyID: ak, SecretAccessKey: sk}, r, payloadHash, "s3", "us-east-1", time.Now().UTC()); err != nil {
		t.Fatalf("signing failed: %v", err)
	}
	return r
}

func TestAuthorizer_AllowRoute(t *testing.T) {
	az := NewAuthorizer(config.Auth{})
	p := &Principal{
		Name:        "ci",
		AllowRoutes: []string{"route.images"},
	}
	if !az.AllowRoute(p, "route.images", s3ops.OpGetObject) {
		t.Error("expected route to be allowed")
	}
	if az.AllowRoute(p, "route.logs", s3ops.OpGetObject) {
		t.Error("expected route to be denied")
	}
}

func TestAuthorizer_AllowOperation(t *testing.T) {
	az := NewAuthorizer(config.Auth{})
	if !az.AllowOperation(nil, s3ops.OpListBuckets) {
		t.Error("expected nil principal to be allowed")
	}
	if !az.AllowOperation(&Principal{}, s3ops.OpListBuckets) {
		t.Error("expected empty allow_ops to allow default operation set")
	}
	if !az.AllowOperation(&Principal{AllowOps: []string{"*"}}, s3ops.OpListBuckets) {
		t.Error("expected wildcard operation to be allowed")
	}
	if az.AllowOperation(&Principal{AllowOps: []string{string(s3ops.OpGetObject)}}, s3ops.OpListBuckets) {
		t.Error("expected ListBuckets to be denied")
	}
}

type errorReadCloser struct{ err error }

func (e errorReadCloser) Read([]byte) (int, error) { return 0, e.err }

func (errorReadCloser) Close() error { return nil }

func TestAuthorizer_WildcardRoute(t *testing.T) {
	az := NewAuthorizer(config.Auth{})
	p := &Principal{
		Name:        "admin",
		AllowRoutes: []string{"*"},
	}
	if !az.AllowRoute(p, "route.anything", s3ops.OpGetObject) {
		t.Error("expected wildcard route to be allowed")
	}
}

func TestAuthorizer_NilPrincipal(t *testing.T) {
	az := NewAuthorizer(config.Auth{})
	if !az.AllowRoute(nil, "route.anything", s3ops.OpGetObject) {
		t.Error("expected nil principal to be allowed (none mode)")
	}
}
