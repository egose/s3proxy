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
	"sync"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	v4 "github.com/aws/aws-sdk-go-v2/aws/signer/v4"
	"github.com/egose/s3proxy/internal/config"
	"github.com/egose/s3proxy/internal/replaybody"
	"github.com/egose/s3proxy/internal/s3ops"
)

func v4signer() *v4.Signer {
	return v4.NewSigner()
}

func newTestAuthenticator(cfg config.Auth) (Authenticator, error) {
	return NewAuthenticator(cfg, replaybody.NewBudget(replaybody.DefaultMaxBytes, replaybody.DefaultAggregateMaxBytes))
}

func TestNoneAuthenticator(t *testing.T) {
	authenticator, err := newTestAuthenticator(config.Auth{Mode: config.AuthModeNone})
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

func TestSigV4StaticRequiresReplayBudget(t *testing.T) {
	_, err := NewAuthenticator(config.Auth{
		Mode: config.AuthModeSigV4Static,
		Clients: map[string]config.Client{
			"ci": {Name: "ci", AccessKey: "AKIACI", SecretKey: "s"},
		},
	}, nil)
	if err == nil {
		t.Fatal("expected missing replay budget error")
	}
}

func TestSigV4Static_UnknownAccessKey(t *testing.T) {
	cfg := config.Auth{
		Mode: config.AuthModeSigV4Static,
		Clients: map[string]config.Client{
			"ci": {Name: "ci", AccessKey: "AKIACI", SecretKey: "s"},
		},
	}
	authenticator, err := newTestAuthenticator(cfg)
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
	authenticator, err := newTestAuthenticator(cfg)
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

func TestSigV4Static_PrincipalSnapshotsPolicy(t *testing.T) {
	const ak = "AKIATEST"
	const sk = "testsecret123"
	allowRoutes := []string{"route.images"}
	allowOps := []string{string(s3ops.OpGetObject)}
	visibleBuckets := []string{"images"}
	client := config.Client{
		Name:           "ci",
		AccessKey:      ak,
		SecretKey:      sk,
		AllowRoutes:    allowRoutes,
		AllowOps:       allowOps,
		VisibleBuckets: visibleBuckets,
	}
	authenticator, err := newTestAuthenticator(config.Auth{
		Mode: config.AuthModeSigV4Static,
		Clients: map[string]config.Client{
			"ci": client,
		},
	})
	if err != nil {
		t.Fatal(err)
	}

	allowRoutes[0] = "route.mutated"
	allowOps[0] = string(s3ops.OpDeleteObject)
	visibleBuckets[0] = "mutated"
	client.SecretKey = "mutated-secret"

	p, err := authenticator.Authenticate(signedUnsignedPayloadRequest(t, ak, sk))
	if err != nil {
		t.Fatalf("Authenticate failed after source mutation: %v", err)
	}
	if got, want := p.AllowRoutes[0], "route.images"; got != want {
		t.Fatalf("AllowRoutes[0] = %q, want %q", got, want)
	}
	if got, want := p.AllowOps[0], string(s3ops.OpGetObject); got != want {
		t.Fatalf("AllowOps[0] = %q, want %q", got, want)
	}
	if got, want := p.VisibleBuckets[0], "images"; got != want {
		t.Fatalf("VisibleBuckets[0] = %q, want %q", got, want)
	}

	p.AllowRoutes[0] = "route.returned"
	p.AllowOps[0] = string(s3ops.OpDeleteObject)
	p.VisibleBuckets[0] = "returned"
	p2, err := authenticator.Authenticate(signedUnsignedPayloadRequest(t, ak, sk))
	if err != nil {
		t.Fatalf("Authenticate failed after returned principal mutation: %v", err)
	}
	if got, want := p2.AllowRoutes[0], "route.images"; got != want {
		t.Fatalf("next AllowRoutes[0] = %q, want %q", got, want)
	}
	if got, want := p2.AllowOps[0], string(s3ops.OpGetObject); got != want {
		t.Fatalf("next AllowOps[0] = %q, want %q", got, want)
	}
	if got, want := p2.VisibleBuckets[0], "images"; got != want {
		t.Fatalf("next VisibleBuckets[0] = %q, want %q", got, want)
	}
}

func TestSigV4Static_ConcurrentAuthenticateWithPriorPrincipalMutation(t *testing.T) {
	const ak = "AKIATEST"
	const sk = "testsecret123"
	authenticator, err := newTestAuthenticator(config.Auth{
		Mode: config.AuthModeSigV4Static,
		Clients: map[string]config.Client{
			"ci": {
				Name:           "ci",
				AccessKey:      ak,
				SecretKey:      sk,
				AllowRoutes:    []string{"route.images"},
				AllowOps:       []string{string(s3ops.OpGetObject)},
				VisibleBuckets: []string{"images"},
			},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	prior, err := authenticator.Authenticate(signedUnsignedPayloadRequest(t, ak, sk))
	if err != nil {
		t.Fatal(err)
	}

	var wg sync.WaitGroup
	wg.Add(2)
	go func() {
		defer wg.Done()
		for i := 0; i < 100; i++ {
			p, err := authenticator.Authenticate(signedUnsignedPayloadRequest(t, ak, sk))
			if err != nil {
				t.Errorf("Authenticate failed: %v", err)
				return
			}
			if p.AllowRoutes[0] != "route.images" || p.AllowOps[0] != string(s3ops.OpGetObject) || p.VisibleBuckets[0] != "images" {
				t.Errorf("unexpected principal policy: %#v", p)
				return
			}
		}
	}()
	go func() {
		defer wg.Done()
		for i := 0; i < 100; i++ {
			prior.AllowRoutes[0] = "route.mutated"
			prior.AllowOps[0] = string(s3ops.OpDeleteObject)
			prior.VisibleBuckets[0] = "mutated"
			prior.AllowRoutes[0] = "route.images"
			prior.AllowOps[0] = string(s3ops.OpGetObject)
			prior.VisibleBuckets[0] = "images"
		}
	}()
	wg.Wait()
}

func TestSigV4Static_StripsUnsignedControlHeaders(t *testing.T) {
	const ak = "AKIATEST"
	const sk = "testsecret123"
	authenticator, err := newTestAuthenticator(config.Auth{
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
	authenticator, err := newTestAuthenticator(config.Auth{
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
	authenticator, err := newTestAuthenticator(config.Auth{
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
	authenticator, err := newTestAuthenticator(config.Auth{
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
	authenticator, err := newTestAuthenticator(config.Auth{
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
	authenticator, err := newTestAuthenticator(cfg)
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
	authenticator, err := newTestAuthenticator(cfg)
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

func TestSigV4Static_InvalidExplicitPayloadHashSignatureDoesNotReadOrReserve(t *testing.T) {
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
		t.Fatal(err)
	}

	t.Run("header", func(t *testing.T) {
		body := &countingErrorReadCloser{err: errors.New("body should not be read")}
		sum := sha256.Sum256([]byte("payload"))
		payloadHash := hex.EncodeToString(sum[:])
		r := httptest.NewRequest(http.MethodPut, "/bucket/key", nil)
		r.Host = "s3proxy.example.com"
		r.Body = body
		r.ContentLength = -1
		r.Header.Set("X-Amz-Content-Sha256", payloadHash)
		if err := v4signer().SignHTTP(context.Background(), aws.Credentials{AccessKeyID: ak, SecretAccessKey: sk}, r, payloadHash, "s3", "us-east-1", time.Now().UTC()); err != nil {
			t.Fatalf("signing failed: %v", err)
		}
		authHeader := r.Header.Get("Authorization")
		replacement := "0"
		if authHeader[len(authHeader)-1:] == replacement {
			replacement = "1"
		}
		r.Header.Set("Authorization", authHeader[:len(authHeader)-1]+replacement)

		if _, err := authenticator.Authenticate(r); !errors.Is(err, errSignatureMismatch) {
			t.Fatalf("err = %v, want %v", err, errSignatureMismatch)
		}
		if body.reads != 0 {
			t.Fatalf("body reads = %d, want 0", body.reads)
		}
		if budget.Used() != 0 {
			t.Fatalf("budget used = %d, want 0", budget.Used())
		}
	})

	t.Run("presigned", func(t *testing.T) {
		body := &countingErrorReadCloser{err: errors.New("body should not be read")}
		sum := sha256.Sum256([]byte("payload"))
		payloadHash := hex.EncodeToString(sum[:])
		r := httptest.NewRequest(http.MethodPut, "https://proxy.example.com/bucket/key", nil)
		r.Body = body
		r.ContentLength = -1
		r.Header.Set("X-Amz-Content-Sha256", payloadHash)
		q := r.URL.Query()
		q.Set("X-Amz-Expires", "600")
		r.URL.RawQuery = q.Encode()
		signedURI, _, err := v4signer().PresignHTTP(context.Background(), aws.Credentials{AccessKeyID: ak, SecretAccessKey: sk}, r, payloadHash, "s3", "us-east-1", time.Now().UTC())
		if err != nil {
			t.Fatalf("presigning failed: %v", err)
		}
		parsed, err := url.Parse(signedURI)
		if err != nil {
			t.Fatalf("parse signed URI: %v", err)
		}
		r.URL = parsed
		r.Host = parsed.Host
		q = r.URL.Query()
		q.Set("X-Amz-Signature", strings.Repeat("0", 64))
		r.URL.RawQuery = q.Encode()

		if _, err := authenticator.Authenticate(r); !errors.Is(err, errSignatureMismatch) {
			t.Fatalf("err = %v, want %v", err, errSignatureMismatch)
		}
		if body.reads != 0 {
			t.Fatalf("body reads = %d, want 0", body.reads)
		}
		if budget.Used() != 0 {
			t.Fatalf("budget used = %d, want 0", budget.Used())
		}
	})
}

func TestSigV4Static_ExplicitPayloadHashMismatchReleasesReservation(t *testing.T) {
	const ak = "AKIATEST"
	const sk = "testsecret123"
	budget := replaybody.NewBudget(100, 100)
	authenticator, err := NewAuthenticator(config.Auth{
		Mode: config.AuthModeSigV4Static,
		Clients: map[string]config.Client{
			"ci": {Name: "ci", AccessKey: ak, SecretKey: sk},
		},
	}, budget)
	if err != nil {
		t.Fatal(err)
	}

	r := signedHashedPayloadRequest(t, ak, sk, "payload")
	r.Body = io.NopCloser(strings.NewReader("changed"))
	r.GetBody = nil
	r.ContentLength = int64(len("changed"))
	if _, err := authenticator.Authenticate(r); !errors.Is(err, errPayloadHashMismatch) {
		t.Fatalf("err = %v, want %v", err, errPayloadHashMismatch)
	}
	if budget.Used() != 0 {
		t.Fatalf("budget used = %d, want 0", budget.Used())
	}
}

func TestSigV4Static_ExplicitPayloadHashPreservesReplayAfterVerification(t *testing.T) {
	const ak = "AKIATEST"
	const sk = "testsecret123"
	budget := replaybody.NewBudget(100, 100)
	authenticator, err := NewAuthenticator(config.Auth{
		Mode: config.AuthModeSigV4Static,
		Clients: map[string]config.Client{
			"ci": {Name: "ci", AccessKey: ak, SecretKey: sk},
		},
	}, budget)
	if err != nil {
		t.Fatal(err)
	}

	r := signedHashedPayloadRequest(t, ak, sk, "payload")
	r.GetBody = nil
	r.Body = io.NopCloser(strings.NewReader("payload"))
	r.ContentLength = int64(len("payload"))
	if _, err := authenticator.Authenticate(r); err != nil {
		t.Fatalf("Authenticate failed: %v", err)
	}
	data, err := io.ReadAll(r.Body)
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != "payload" {
		t.Fatalf("body = %q, want payload", string(data))
	}
	if err := replaybody.Reset(r); err != nil {
		t.Fatal(err)
	}
	data, err = io.ReadAll(r.Body)
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != "payload" {
		t.Fatalf("reset body = %q, want payload", string(data))
	}
	if err := replaybody.Release(r); err != nil {
		t.Fatal(err)
	}
	if budget.Used() != 0 {
		t.Fatalf("budget used = %d, want 0", budget.Used())
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
	authenticator, err := newTestAuthenticator(cfg)
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
	authenticator, err := newTestAuthenticator(cfg)
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
	authenticator, err := newTestAuthenticator(cfg)
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

func TestSigV4Static_PresignedQueryRejectsTampering(t *testing.T) {
	const ak = "AKIATEST"
	const sk = "testsecret123"
	authenticator, err := newTestAuthenticator(config.Auth{
		Mode: config.AuthModeSigV4Static,
		Clients: map[string]config.Client{
			"ci": {Name: "ci", AccessKey: ak, SecretKey: sk},
		},
	})
	if err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		name    string
		mutate  func(*http.Request)
		wantErr error
	}{
		{
			name: "one character signature change",
			mutate: func(r *http.Request) {
				q := r.URL.Query()
				sig := q.Get("X-Amz-Signature")
				replacement := "0"
				if sig[len(sig)-1:] == replacement {
					replacement = "1"
				}
				q.Set("X-Amz-Signature", sig[:len(sig)-1]+replacement)
				r.URL.RawQuery = q.Encode()
			},
			wantErr: errSignatureMismatch,
		},
		{
			name: "random signature with known access key",
			mutate: func(r *http.Request) {
				q := r.URL.Query()
				q.Set("X-Amz-Signature", strings.Repeat("a", 64))
				r.URL.RawQuery = q.Encode()
			},
			wantErr: errSignatureMismatch,
		},
		{
			name: "empty signature",
			mutate: func(r *http.Request) {
				q := r.URL.Query()
				q.Set("X-Amz-Signature", "")
				r.URL.RawQuery = q.Encode()
			},
			wantErr: errMissingSignature,
		},
		{
			name: "path changed",
			mutate: func(r *http.Request) {
				r.URL.Path = "/bucket/other-key"
			},
			wantErr: errSignatureMismatch,
		},
		{
			name: "query changed",
			mutate: func(r *http.Request) {
				q := r.URL.Query()
				q.Set("response-content-type", "text/plain")
				r.URL.RawQuery = q.Encode()
			},
			wantErr: errSignatureMismatch,
		},
		{
			name: "missing algorithm",
			mutate: func(r *http.Request) {
				q := r.URL.Query()
				q.Del("X-Amz-Algorithm")
				r.URL.RawQuery = q.Encode()
			},
			wantErr: errUnsupportedAuthScheme,
		},
		{
			name: "algorithm changed",
			mutate: func(r *http.Request) {
				q := r.URL.Query()
				q.Set("X-Amz-Algorithm", "AWS4-HMAC-SHA1")
				r.URL.RawQuery = q.Encode()
			},
			wantErr: errUnsupportedAuthScheme,
		},
		{
			name: "duplicate signature",
			mutate: func(r *http.Request) {
				q := r.URL.Query()
				q.Add("X-Amz-Signature", q.Get("X-Amz-Signature"))
				r.URL.RawQuery = q.Encode()
			},
			wantErr: errDuplicatePresignParameter,
		},
		{
			name: "duplicate algorithm",
			mutate: func(r *http.Request) {
				q := r.URL.Query()
				q.Add("X-Amz-Algorithm", "AWS4-HMAC-SHA256")
				r.URL.RawQuery = q.Encode()
			},
			wantErr: errDuplicatePresignParameter,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := mustPresignedRequest(t, ak, sk, time.Now().UTC(), 10*time.Minute)
			tt.mutate(req)
			if _, err := authenticator.Authenticate(req); !errors.Is(err, tt.wantErr) {
				t.Fatalf("err = %v, want %v", err, tt.wantErr)
			}
		})
	}

	t.Run("signed header changed", func(t *testing.T) {
		req := mustPresignedRequestWithHeaders(t, ak, sk, time.Now().UTC(), 10*time.Minute, http.Header{"Range": []string{"bytes=0-1"}})
		req.Header.Set("Range", "bytes=1-2")
		if _, err := authenticator.Authenticate(req); !errors.Is(err, errSignatureMismatch) {
			t.Fatalf("err = %v, want %v", err, errSignatureMismatch)
		}
	})
}

func TestSigV4Static_PresignedQueryRejectsDuplicateRequiredParameters(t *testing.T) {
	const ak = "AKIATEST"
	const sk = "testsecret123"
	authenticator, err := newTestAuthenticator(config.Auth{
		Mode: config.AuthModeSigV4Static,
		Clients: map[string]config.Client{
			"ci": {Name: "ci", AccessKey: ak, SecretKey: sk},
		},
	})
	if err != nil {
		t.Fatal(err)
	}

	for _, param := range []string{
		"X-Amz-Algorithm",
		"X-Amz-Credential",
		"X-Amz-Date",
		"X-Amz-Expires",
		"X-Amz-Signature",
		"X-Amz-Security-Token",
		"X-Amz-SignedHeaders",
	} {
		t.Run(param, func(t *testing.T) {
			req := mustPresignedRequest(t, ak, sk, time.Now().UTC(), 10*time.Minute)
			q := req.URL.Query()
			if value := q.Get(param); value != "" {
				q.Add(param, value)
			} else {
				q.Add(param, "one")
				q.Add(param, "two")
			}
			req.URL.RawQuery = q.Encode()
			if _, err := authenticator.Authenticate(req); !errors.Is(err, errDuplicatePresignParameter) {
				t.Fatalf("err = %v, want %v", err, errDuplicatePresignParameter)
			}
		})
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
	authenticator, err := newTestAuthenticator(cfg)
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
	authenticator, err := newTestAuthenticator(cfg)
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
	authenticator, err := newTestAuthenticator(cfg)
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
	authenticator, err := newTestAuthenticator(config.Auth{
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
	return mustPresignedRequestWithHeaders(t, ak, sk, signedAt, expires, nil)
}

func mustPresignedRequestWithHeaders(t *testing.T, ak, sk string, signedAt time.Time, expires time.Duration, headers http.Header) *http.Request {
	t.Helper()
	req := httptest.NewRequest(http.MethodGet, "https://proxy.example.com/bucket/key", nil)
	for name, values := range headers {
		for _, value := range values {
			req.Header.Add(name, value)
		}
	}
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
	az := NewAuthorizer()
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
	az := NewAuthorizer()
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

type countingErrorReadCloser struct {
	err    error
	reads  int
	closed bool
}

func (c *countingErrorReadCloser) Read([]byte) (int, error) {
	c.reads++
	return 0, c.err
}

func (c *countingErrorReadCloser) Close() error {
	c.closed = true
	return nil
}

func TestAuthorizer_WildcardRoute(t *testing.T) {
	az := NewAuthorizer()
	p := &Principal{
		Name:        "admin",
		AllowRoutes: []string{"*"},
	}
	if !az.AllowRoute(p, "route.anything", s3ops.OpGetObject) {
		t.Error("expected wildcard route to be allowed")
	}
}

func TestAuthorizer_NilPrincipal(t *testing.T) {
	az := NewAuthorizer()
	if !az.AllowRoute(nil, "route.anything", s3ops.OpGetObject) {
		t.Error("expected nil principal to be allowed (none mode)")
	}
}
