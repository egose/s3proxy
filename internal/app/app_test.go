package app

import (
	"bytes"
	"context"
	"io"
	"log/slog"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"
)

func TestBuildDoesNotReplaceDefaultLogger(t *testing.T) {
	original := slog.Default()
	t.Cleanup(func() { slog.SetDefault(original) })

	var defaultLogs bytes.Buffer
	slog.SetDefault(slog.New(slog.NewTextHandler(&defaultLogs, nil)))

	var buildLogs bytes.Buffer
	_, err := Build(context.Background(), BuildOptions{
		ConfigPath: tempConfig(t, "127.0.0.1:0"),
		Logger:     slog.New(slog.NewTextHandler(&buildLogs, nil)),
	})
	if err != nil {
		t.Fatalf("Build() error = %v", err)
	}

	slog.Info("default logger still active")
	if !strings.Contains(defaultLogs.String(), "default logger still active") {
		t.Fatalf("default logger was not used after Build")
	}
	if strings.Contains(buildLogs.String(), "default logger still active") {
		t.Fatalf("Build logger received global default log")
	}
}

func TestBuildsUseIndependentLoggers(t *testing.T) {
	var firstLogs bytes.Buffer
	var secondLogs bytes.Buffer

	first, err := Build(context.Background(), BuildOptions{
		ConfigPath: tempConfig(t, "127.0.0.1:0"),
		Logger:     slog.New(slog.NewTextHandler(&firstLogs, nil)),
	})
	if err != nil {
		t.Fatalf("first Build() error = %v", err)
	}
	second, err := Build(context.Background(), BuildOptions{
		ConfigPath: tempConfig(t, "127.0.0.1:0"),
		Logger:     slog.New(slog.NewTextHandler(&secondLogs, nil)),
	})
	if err != nil {
		t.Fatalf("second Build() error = %v", err)
	}

	first.logger.Info("first app")
	second.logger.Info("second app")

	if !strings.Contains(firstLogs.String(), "first app") || strings.Contains(firstLogs.String(), "second app") {
		t.Fatalf("first logs = %q", firstLogs.String())
	}
	if !strings.Contains(secondLogs.String(), "second app") || strings.Contains(secondLogs.String(), "first app") {
		t.Fatalf("second logs = %q", secondLogs.String())
	}
}

func TestValidateConfigDoesNotReplaceDefaultLogger(t *testing.T) {
	original := slog.Default()
	t.Cleanup(func() { slog.SetDefault(original) })

	var defaultLogs bytes.Buffer
	slog.SetDefault(slog.New(slog.NewTextHandler(&defaultLogs, nil)))

	if err := ValidateConfig(tempConfig(t, "127.0.0.1:0")); err != nil {
		t.Fatalf("ValidateConfig() error = %v", err)
	}
	slog.Info("validate default logger still active")
	if !strings.Contains(defaultLogs.String(), "validate default logger still active") {
		t.Fatalf("default logger was not used after ValidateConfig")
	}
}

func TestBuildConfiguresUpstreamIdlePool(t *testing.T) {
	a, err := Build(context.Background(), BuildOptions{
		ConfigPath: tempConfig(t, "127.0.0.1:0"),
		Logger:     testLogger(),
	})
	if err != nil {
		t.Fatalf("Build() error = %v", err)
	}
	if got, want := a.transport.MaxIdleConnsPerHost, defaultUpstreamMaxIdleConnsPerHost; got != want {
		t.Fatalf("MaxIdleConnsPerHost = %d, want %d", got, want)
	}
}

func TestUpstreamTransportReusesConcurrentSameHostConnections(t *testing.T) {
	var conns int
	var mu sync.Mutex
	server := httptest.NewUnstartedServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("ok"))
	}))
	server.Config.ConnState = func(_ net.Conn, state http.ConnState) {
		if state == http.StateNew {
			mu.Lock()
			conns++
			mu.Unlock()
		}
	}
	server.Start()
	defer server.Close()

	transport := newUpstreamTransport()
	defer transport.CloseIdleConnections()
	client := &http.Client{Transport: transport}

	for wave := 0; wave < 2; wave++ {
		var wg sync.WaitGroup
		for i := 0; i < defaultUpstreamMaxIdleConnsPerHost; i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				resp, err := client.Get(server.URL)
				if err != nil {
					t.Errorf("Get() error = %v", err)
					return
				}
				io.Copy(io.Discard, resp.Body)
				resp.Body.Close()
			}()
		}
		wg.Wait()
	}

	mu.Lock()
	got := conns
	mu.Unlock()
	if got > defaultUpstreamMaxIdleConnsPerHost {
		t.Fatalf("connections = %d, want no more than idle pool size %d", got, defaultUpstreamMaxIdleConnsPerHost)
	}
}

func TestRunReturnsBindFailure(t *testing.T) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer ln.Close()

	a, err := Build(context.Background(), BuildOptions{
		ConfigPath: tempConfig(t, ln.Addr().String()),
		Logger:     testLogger(),
	})
	if err != nil {
		t.Fatalf("Build() error = %v", err)
	}

	if err := a.Run(context.Background()); err == nil {
		t.Fatalf("Run() error = nil, want bind error")
	}
}

func TestRunReturnsAfterExternalServerClose(t *testing.T) {
	a := newTestApp(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}), 10*time.Second)

	errCh := make(chan error, 1)
	go func() { errCh <- a.Run(context.Background()) }()
	waitForTCP(t, a.server.Addr)

	if err := a.server.Close(); err != nil {
		t.Fatalf("Close() error = %v", err)
	}
	if err := waitRun(t, errCh); err != nil {
		t.Fatalf("Run() error = %v", err)
	}
}

func TestRunContextCancellationShutsDown(t *testing.T) {
	a := newTestApp(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}), 10*time.Second)
	ctx, cancel := context.WithCancel(context.Background())
	errCh := make(chan error, 1)
	go func() { errCh <- a.Run(ctx) }()
	waitForTCP(t, a.server.Addr)

	cancel()
	if err := waitRun(t, errCh); err != nil {
		t.Fatalf("Run() error = %v", err)
	}
}

func TestRunAlreadyCanceledContextReturns(t *testing.T) {
	a := newTestApp(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}), 10*time.Second)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	errCh := make(chan error, 1)
	go func() { errCh <- a.Run(ctx) }()
	if err := waitRun(t, errCh); err != nil {
		t.Fatalf("Run() error = %v", err)
	}
}

func TestRunReportsShutdownTimeout(t *testing.T) {
	entered := make(chan struct{})
	release := make(chan struct{})
	a := newTestApp(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		close(entered)
		<-release
	}), 10*time.Millisecond)
	defer close(release)

	ctx, cancel := context.WithCancel(context.Background())
	errCh := make(chan error, 1)
	go func() { errCh <- a.Run(ctx) }()
	waitForTCP(t, a.server.Addr)

	requestDone := make(chan struct{})
	go func() {
		client := &http.Client{Transport: &http.Transport{Proxy: nil}}
		_, _ = client.Get("http://" + a.server.Addr + "/")
		close(requestDone)
	}()
	waitForHandler(t, entered, requestDone)

	cancel()
	err := waitRun(t, errCh)
	if err == nil || !strings.Contains(err.Error(), "server shutdown") {
		t.Fatalf("Run() error = %v, want shutdown timeout", err)
	}
}

func TestRepeatedBuildAndShutdown(t *testing.T) {
	for i := 0; i < 2; i++ {
		a, err := Build(context.Background(), BuildOptions{
			ConfigPath: tempConfig(t, freeAddr(t)),
			Logger:     testLogger(),
		})
		if err != nil {
			t.Fatalf("Build() %d error = %v", i, err)
		}

		ctx, cancel := context.WithCancel(context.Background())
		errCh := make(chan error, 1)
		go func() { errCh <- a.Run(ctx) }()
		waitForTCP(t, a.server.Addr)
		cancel()
		if err := waitRun(t, errCh); err != nil {
			t.Fatalf("Run() %d error = %v", i, err)
		}
	}
}

func newTestApp(t *testing.T, handler http.Handler, shutdownTimeout time.Duration) *App {
	t.Helper()
	addr := freeAddr(t)
	return &App{
		server: &http.Server{
			Addr:    addr,
			Handler: handler,
		},
		transport:       &http.Transport{},
		logger:          testLogger(),
		shutdownTimeout: shutdownTimeout,
	}
}

func tempConfig(t *testing.T, addr string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "config.hcl")
	src := strings.ReplaceAll(validConfig, "${ADDRESS}", addr)
	if err := os.WriteFile(path, []byte(src), 0o600); err != nil {
		t.Fatal(err)
	}
	return path
}

func testLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(&bytes.Buffer{}, nil))
}

func freeAddr(t *testing.T) string {
	t.Helper()
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	addr := ln.Addr().String()
	if err := ln.Close(); err != nil {
		t.Fatal(err)
	}
	return addr
}

func waitForTCP(t *testing.T, addr string) {
	t.Helper()
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		conn, err := net.DialTimeout("tcp", addr, 10*time.Millisecond)
		if err == nil {
			conn.Close()
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatalf("server did not listen on %s", addr)
}

func waitRun(t *testing.T, errCh <-chan error) error {
	t.Helper()
	select {
	case err := <-errCh:
		return err
	case <-time.After(2 * time.Second):
		t.Fatal("Run() did not return")
	}
	return nil
}

func waitForHandler(t *testing.T, entered <-chan struct{}, requestDone <-chan struct{}) {
	t.Helper()
	select {
	case <-entered:
		return
	case <-requestDone:
		t.Fatal("request finished before shutdown test could cancel")
	case <-time.After(2 * time.Second):
		t.Fatal("handler did not start")
	}
}

var validConfig = `
listener "http" "public" {
  address = "${ADDRESS}"

  addressing {
    path_style = true
  }
}

auth "main" {
  mode = "none"
}

credential "static" "primary" {
  access_key = "access"
  secret_key = "secret" // pragma: allowlist secret
}

target "s3" "primary" {
  endpoint         = "http://127.0.0.1:9000"
  region           = "us-east-1"
  force_path_style = true
  credentials      = "primary"
}

parser "path_prefix" "all" {
  prefix = "/"
}

route "all" {
  parser       = "all"
  operations   = ["GetObject"]
  destinations = ["primary"]
  dispatch     = "first"
  on_match     = "stop"
}
`
