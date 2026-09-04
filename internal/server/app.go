package server

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/MaxCaribe/library-go/internal/config"
	httpServer "github.com/MaxCaribe/library-go/internal/infrastructure/httpserver/server"
	"github.com/MaxCaribe/library-go/pkg/logging"
)

// shutdownTimeout bounds how long in-flight requests may finish after SIGTERM.
// Keep it below the orchestrator's grace period (Kubernetes terminationGracePeriodSeconds).
const shutdownTimeout = 15 * time.Second

type App struct {
	cancelCtx context.CancelFunc
	infra     io.Closer
	server    *httpServer.Server
	logger    *slog.Logger
}

func New(cfg config.Config) (*App, error) {
	ctx, cancelCtx := context.WithCancel(context.Background())

	logger := newLogger(cfg)

	infra, err := newInfra(ctx, cfg, logger)
	if err != nil {
		cancelCtx()
		return nil, err
	}

	services, err := newServices(infra, cfg, logger)
	if err != nil {
		cancelCtx()
		// Already failing, so the construction error is the one worth returning.
		_ = infra.Close()
		return nil, err
	}

	handler := newHTTPHandler(services, logger)
	addr := fmt.Sprintf(":%v", cfg.Http.Port)

	return &App{
		cancelCtx: cancelCtx,
		infra:     infra,
		server:    httpServer.New(handler, addr, logger),
		logger:    logger,
	}, nil
}

func newLogger(cfg config.Config) *slog.Logger {
	return logging.CreateRootSlogger(!cfg.Debug).With("package", "server")
}

func (a *App) Start() error {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	serverErrors := make(chan error, 1)

	go func() {
		a.logger.InfoContext(ctx, "Starting REST API server")
		if err := a.server.Start(); err != nil {
			serverErrors <- err
		}
	}()

	select {
	case err := <-serverErrors:
		a.logger.Error("Server failed to start", "error", err)
		return fmt.Errorf("server error: %w", err)
	case <-ctx.Done():
		a.logger.InfoContext(ctx, "Shutdown signal received")
	}

	a.cancelCtx()

	shutdownCtx, cancel := context.WithTimeout(context.Background(), shutdownTimeout)
	defer cancel()

	a.logger.InfoContext(shutdownCtx, "Shutting down server...")
	if err := a.server.Stop(shutdownCtx); err != nil {
		a.logger.Error("Server shutdown error", "error", err)
		return fmt.Errorf("server shutdown error: %w", err)
	}

	a.logger.InfoContext(shutdownCtx, "Closing infrastructure...")
	if err := a.infra.Close(); err != nil {
		a.logger.Error("Infrastructure close error", "error", err)
		return fmt.Errorf("infrastructure close error: %w", err)
	}

	a.logger.InfoContext(shutdownCtx, "Graceful shutdown completed")
	return nil
}
