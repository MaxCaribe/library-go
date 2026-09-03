package server

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"time"
)

const (
	readHeaderTimeout = 5 * time.Second
	readTimeout       = 15 * time.Second
	writeTimeout      = 30 * time.Second
	idleTimeout       = 60 * time.Second
)

type Server struct {
	server *http.Server
	logger *slog.Logger
}

func New(handler http.Handler, addr string, logger *slog.Logger) *Server {
	return &Server{
		server: &http.Server{
			Addr:              addr,
			Handler:           handler,
			ReadHeaderTimeout: readHeaderTimeout,
			ReadTimeout:       readTimeout,
			WriteTimeout:      writeTimeout,
			IdleTimeout:       idleTimeout,
		},
		logger: logger,
	}
}

func (s *Server) Start() error {
	s.logger.InfoContext(context.Background(), "Starting server", "addr", s.server.Addr)
	if err := s.server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		s.logger.Error("Server failed to start", "error", err)
		return err
	}
	return nil
}

func (s *Server) Stop(ctx context.Context) error {
	s.logger.InfoContext(ctx, "Stopping server")
	return s.server.Shutdown(ctx)
}
