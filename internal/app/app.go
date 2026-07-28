package app

import (
	"context"
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
	defaultReadTimeout                   = 30 * time.Second
	defaultMaxHeaderBytes                = 1 << 20
)

type BuildOptions struct {
	ConfigPath string
	Version    string
}

type App struct {
	Config *config.Runtime
	Server *http.Server
}

func Build(ctx context.Context, opts BuildOptions) (*App, error) {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	slog.SetDefault(logger)

	rt, err := config.LoadFile(opts.ConfigPath)
	if err != nil {
		return nil, fmt.Errorf("config: %w", err)
	}
	logger.Info("config loaded", "routes", len(rt.Routes), "targets", len(rt.Targets), "auth_mode", rt.Auth.Mode)

	authenticator, err := auth.NewAuthenticator(rt.Auth)
	if err != nil {
		return nil, fmt.Errorf("auth: %w", err)
	}
	authorizer := auth.NewAuthorizer(rt.Auth)

	resolver := router.NewResolver(rt)
	rewriter := rewrite.New()

	httpClient := &http.Client{
		Transport: &http.Transport{
			Proxy: http.ProxyFromEnvironment,
			DialContext: (&net.Dialer{
				Timeout:   defaultUpstreamDialTimeout,
				KeepAlive: defaultUpstreamKeepAlive,
			}).DialContext,
			ForceAttemptHTTP2:     true,
			MaxIdleConns:          100,
			IdleConnTimeout:       defaultUpstreamIdleConnTimeout,
			TLSHandshakeTimeout:   defaultUpstreamTLSHandshakeTimeout,
			ExpectContinueTimeout: defaultUpstreamExpectContinueTimeout,
			ResponseHeaderTimeout: defaultUpstreamResponseHeaderTimeout,
		},
	}
	backend := s3.NewClient(httpClient)
	dispatcher := dispatch.New(backend)

	buckets := listbuckets.New(rt.Buckets, time.Now())

	handler := httpapi.NewHandler(httpapi.Dependencies{
		Addressing:    rt.Listener.Addressing,
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

	return &App{Config: rt, Server: server}, nil
}

func (a *App) Run(ctx context.Context) error {
	logger := slog.Default()
	logger.Info("starting server", "address", a.Server.Addr)

	errCh := make(chan error, 1)
	go func() {
		if err := a.Server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			errCh <- err
		}
	}()

	select {
	case err := <-errCh:
		return err
	case <-ctx.Done():
		logger.Info("shutting down server")
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		if err := a.Server.Shutdown(shutdownCtx); err != nil {
			return fmt.Errorf("server shutdown: %w", err)
		}
		logger.Info("server stopped")
		return nil
	}
}
