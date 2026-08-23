package auth

import (
	"errors"
	"fmt"
	"net/http"

	"github.com/egose/s3proxy/internal/config"
	"github.com/egose/s3proxy/internal/replaybody"
	"github.com/egose/s3proxy/internal/s3ops"
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

func NewAuthenticator(cfg config.Auth, replayBudget *replaybody.Budget) (Authenticator, error) {
	switch cfg.Mode {
	case config.AuthModeNone:
		return &noneAuthenticator{}, nil
	case config.AuthModeSigV4Static:
		if replayBudget == nil {
			return nil, fmt.Errorf("replay budget is required for %s auth", cfg.Mode)
		}
		clientsByAK := make(map[string]config.Client, len(cfg.Clients))
		for _, c := range cfg.Clients {
			clientsByAK[c.AccessKey] = cloneClient(c)
		}
		return &sigv4StaticAuthenticator{
			verifier: newSigV4Verifier(clientsByAK, "us-east-1", replayBudget),
		}, nil
	default:
		return nil, fmt.Errorf("unsupported auth mode %q", cfg.Mode)
	}
}

func cloneClient(c config.Client) config.Client {
	c.AllowRoutes = append([]string(nil), c.AllowRoutes...)
	c.AllowOps = append([]string(nil), c.AllowOps...)
	c.VisibleBuckets = append([]string(nil), c.VisibleBuckets...)
	return c
}

func principalFromClient(c config.Client) *Principal {
	return &Principal{
		Name:           c.Name,
		AccessKey:      c.AccessKey,
		AllowRoutes:    append([]string(nil), c.AllowRoutes...),
		AllowOps:       append([]string(nil), c.AllowOps...),
		VisibleBuckets: append([]string(nil), c.VisibleBuckets...),
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
	AllowOperation(p *Principal, op s3ops.Operation) bool
	AllowRoute(p *Principal, routeName string, op s3ops.Operation) bool
}

type staticAuthorizer struct{}

func NewAuthorizer() Authorizer {
	return &staticAuthorizer{}
}

func (a *staticAuthorizer) AllowOperation(p *Principal, op s3ops.Operation) bool {
	if p == nil {
		return true
	}
	return matchOpList(p.AllowOps, op)
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

func IsSignatureMismatch(err error) bool {
	return errors.Is(err, errSignatureMismatch)
}
