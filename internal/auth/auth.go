package auth

import (
	"fmt"
	"net/http"

	"github.com/jahn/s3proxy/internal/config"
	"github.com/jahn/s3proxy/internal/s3ops"
)

type Principal struct {
	Name           string
	AccessKey      string
	AllowRoutes    []string
	AllowOps       []string
	VisibleBuckets []string
}

type Authenticator interface {
	Authenticate(r *http.Request) (*Principal, error)
}

type sigv4StaticAuthenticator struct {
	verifier *sigV4Verifier
}

func NewAuthenticator(cfg config.Auth) (Authenticator, error) {
	switch cfg.Mode {
	case config.AuthModeNone:
		return &noneAuthenticator{}, nil
	case config.AuthModeSigV4Static:
		clientsByAK := make(map[string]config.Client, len(cfg.Clients))
		for _, c := range cfg.Clients {
			clientsByAK[c.AccessKey] = c
		}
		return &sigv4StaticAuthenticator{
			verifier: newSigV4Verifier(clientsByAK, "us-east-1"),
		}, nil
	default:
		return nil, fmt.Errorf("unsupported auth mode %q", cfg.Mode)
	}
}

type noneAuthenticator struct{}

func (a *noneAuthenticator) Authenticate(r *http.Request) (*Principal, error) {
	return nil, nil
}

func (a *sigv4StaticAuthenticator) Authenticate(r *http.Request) (*Principal, error) {
	return a.verifier.Verify(r)
}

type Authorizer interface {
	AllowRoute(p *Principal, routeName string, op s3ops.Operation) bool
	AllowBucketVisibility(p *Principal, visibleBucket string) bool
}

type staticAuthorizer struct{}

func NewAuthorizer(cfg config.Auth) Authorizer {
	return &staticAuthorizer{}
}

func (a *staticAuthorizer) AllowRoute(p *Principal, routeName string, op s3ops.Operation) bool {
	if p == nil {
		return true
	}
	if matchStringList(p.AllowRoutes, routeName) && matchOpList(p.AllowOps, op) {
		return true
	}
	return false
}

func (a *staticAuthorizer) AllowBucketVisibility(p *Principal, visibleBucket string) bool {
	if p == nil {
		return true
	}
	return matchStringList(p.VisibleBuckets, visibleBucket)
}

func matchStringList(allow []string, value string) bool {
	for _, a := range allow {
		if a == "*" || a == value {
			return true
		}
	}
	return false
}

func matchOpList(allow []string, op s3ops.Operation) bool {
	if len(allow) == 0 {
		return true
	}
	for _, a := range allow {
		if a == "*" || a == string(op) {
			return true
		}
	}
	return false
}
