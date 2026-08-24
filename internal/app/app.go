package app

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"os"
	"time"

	"github.com/egose/s3proxy/internal/auth"
	"github.com/egose/s3proxy/internal/backend/s3"
	"github.com/egose/s3proxy/internal/config"
	"github.com/egose/s3proxy/internal/dispatch"
	"github.com/egose/s3proxy/internal/httpapi"
	"github.com/egose/s3proxy/internal/listbuckets"
	"github.com/egose/s3proxy/internal/replaybody"
	"github.com/egose/s3proxy/internal/rewrite"
	"github.com/egose/s3proxy/internal/router"
)

const (
	defaultUpstreamDialTimeout           = 10 * time.Second
	defaultUpstreamKeepAlive             = 30 * time.Second
	defaultUpstreamTLSHandshakeTimeout   = 10 * time.Second
	defaultUpstreamResponseHeaderTimeout = 30 * time.Second
	defaultUpstreamExpectContinueTimeout = 1 * time.Second
	defaultUpstreamIdleConnTimeout       = 90 * time.Second
	defaultUpstreamMaxIdleConnsPerHost   = 32
	defaultReadTimeout                   = 30 * time.Second
	defaultMaxHeaderBytes                = 1 << 20
)

type BuildOptions struct {
	ConfigPath      string
	Logger          *slog.Logger
	ShutdownTimeout time.Duration
}

type App struct {
	server          *http.Server
	transport       *http.Transport
	logger          *slog.Logger
	shutdownTimeout time.Duration
	listen          func(network, address string) (net.Listener, error)
}

func Build(opts BuildOptions) (*App, error) {
	logger := opts.Logger
	if logger == nil {
		logger = slog.New(slog.NewJSONHandler(os.Stdout, nil))
	}

	rt, err := config.LoadFile(opts.ConfigPath)
	if err != nil {
		return nil, fmt.Errorf("config: %w", err)
	}
	logger.Info("config loaded", "routes", len(rt.Routes), "targets", len(rt.Targets), "auth_mode", rt.Auth.Mode)

	replayBudget := replaybody.NewBudget(rt.Listener.ReplayBodyMaxBytes, rt.Listener.ReplayBodyAggregateMaxBytes)

	authenticator, err := auth.NewAuthenticator(rt.Auth, replayBudget)
	if err != nil {
		return nil, fmt.Errorf("auth: %w", err)
	}
	authorizer := auth.NewAuthorizer()

	resolver := router.NewResolver(rt.Routes, rt.Parsers, targetNames(rt.Targets))
	rewriter := rewrite.New()

	transport := newUpstreamTransport()
	httpClient := &http.Client{
		Transport: transport,
	}
	backend, err := s3.NewClient(httpClient, rt.Targets, replayBudget)
	if err != nil {
		return nil, fmt.Errorf("backend: %w", err)
	}
	dispatcher, err := dispatch.New(backend, replayBudget)
	if err != nil {
		return nil, fmt.Errorf("dispatch: %w", err)
	}

	buckets := listbuckets.New(rt.Buckets, time.Now())

	handler := httpapi.NewHandler(httpapi.Dependencies{
		Addressing:    rt.Listener.Addressing,
		ReplayBudget:  replayBudget,
		Authenticator: authenticator,
		Authorizer:    authorizer,
		Router:        resolver,
		Rewriter:      rewriter,
		Dispatcher:    dispatcher,
		Buckets:       buckets,
		Logger:        logger,
	})

	server := &http.Server{
		Addr:           rt.Listener.Address,
		Handler:        handler,
		TLSConfig:      nil,
		ReadTimeout:    defaultReadTimeout,
		MaxHeaderBytes: defaultMaxHeaderBytes,
	}
	if rt.Listener.Timeouts.Read > 0 {
		server.ReadTimeout = rt.Listener.Timeouts.Read
	}
	if rt.Listener.Timeouts.ReadHeader > 0 {
		server.ReadHeaderTimeout = rt.Listener.Timeouts.ReadHeader
	}
	if rt.Listener.Timeouts.Idle > 0 {
		server.IdleTimeout = rt.Listener.Timeouts.Idle
	}
	if rt.Listener.Timeouts.Write > 0 {
		server.WriteTimeout = rt.Listener.Timeouts.Write
	}
	if rt.Listener.MaxHeaderBytes > 0 {
		server.MaxHeaderBytes = rt.Listener.MaxHeaderBytes
	}

	shutdownTimeout := opts.ShutdownTimeout
	if shutdownTimeout == 0 {
		shutdownTimeout = 10 * time.Second
	}
	return &App{server: server, transport: transport, logger: logger, shutdownTimeout: shutdownTimeout}, nil
}

func targetNames(targets map[string]config.S3Target) []string {
	out := make([]string, 0, len(targets))
	for name := range targets {
		out = append(out, name)
	}
	return out
}

func newUpstreamTransport() *http.Transport {
	return &http.Transport{
		Proxy: http.ProxyFromEnvironment,
		DialContext: (&net.Dialer{
			Timeout:   defaultUpstreamDialTimeout,
			KeepAlive: defaultUpstreamKeepAlive,
		}).DialContext,
		ForceAttemptHTTP2:     true,
		DisableCompression:    true,
		MaxIdleConns:          100,
		MaxIdleConnsPerHost:   defaultUpstreamMaxIdleConnsPerHost,
		IdleConnTimeout:       defaultUpstreamIdleConnTimeout,
		TLSHandshakeTimeout:   defaultUpstreamTLSHandshakeTimeout,
		ExpectContinueTimeout: defaultUpstreamExpectContinueTimeout,
		ResponseHeaderTimeout: defaultUpstreamResponseHeaderTimeout,
	}
}

func ValidateConfig(path string) error {
	_, err := config.LoadFile(path)
	if err != nil {
		return fmt.Errorf("config: %w", err)
	}
	return nil
}

func (a *App) Run(ctx context.Context) error {
	addr := a.server.Addr
	if addr == "" {
		addr = ":http"
	}
	listen := net.Listen
	if a.listen != nil {
		listen = a.listen
	}
	ln, err := listen("tcp", addr)
	if err != nil {
		return err
	}
	defer a.transport.CloseIdleConnections()

	requestsCtx, cancelRequests := context.WithCancel(context.Background())
	defer cancelRequests()
	previousBaseContext := a.server.BaseContext
	a.server.BaseContext = func(ln net.Listener) context.Context {
		if previousBaseContext == nil {
			return requestsCtx
		}
		base := previousBaseContext(ln)
		ctx, cancel := context.WithCancel(base)
		go func() {
			<-requestsCtx.Done()
			cancel()
		}()
		return ctx
	}
	defer func() { a.server.BaseContext = previousBaseContext }()
	a.logger.Info("starting server", "address", a.server.Addr)

	errCh := make(chan error, 1)
	go func() {
		errCh <- a.server.Serve(ln)
	}()

	select {
	case err := <-errCh:
		return normalizeServerError(err)
	case <-ctx.Done():
		a.logger.Info("shutting down server")
		shutdownCtx, cancel := context.WithTimeout(context.Background(), a.shutdownTimeout)
		shutdownErr := a.server.Shutdown(shutdownCtx)
		cancel()
		if shutdownErr != nil {
			cancelRequests()
			closeErr := normalizeServerError(a.server.Close())
			serveErr := normalizeServerError(<-errCh)
			return serverShutdownError(shutdownErr, closeErr, serveErr)
		}
		if err := normalizeServerError(<-errCh); err != nil {
			return err
		}
		a.logger.Info("server stopped")
		return nil
	}
}

func serverShutdownError(shutdownErr, closeErr, serveErr error) error {
	errs := []error{shutdownErr}
	if closeErr != nil {
		errs = append(errs, fmt.Errorf("server close: %w", closeErr))
	}
	if serveErr != nil {
		errs = append(errs, fmt.Errorf("server serve: %w", serveErr))
	}
	return fmt.Errorf("server shutdown: %w", errors.Join(errs...))
}

func normalizeServerError(err error) error {
	if err == nil || err == http.ErrServerClosed {
		return nil
	}
	return err
}
