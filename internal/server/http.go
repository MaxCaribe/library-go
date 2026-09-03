package server

import (
	"log/slog"
	"net/http"

	"github.com/MaxCaribe/library-go/internal/infrastructure/httpserver/handlers"
	"github.com/MaxCaribe/library-go/internal/infrastructure/httpserver/middleware"
)

const maxRequestBodyBytes int64 = 1_000_000 // 1 MB

func newHTTPHandler(_ *appServices, logger *slog.Logger) http.Handler {
	healthHandler := handlers.NewHeartHandler()

	recoveryMiddleware := middleware.NewRecoveryMiddleware(logger)
	bodyLimitMiddleware := middleware.NewBodyLimitMiddleware(maxRequestBodyBytes)
	requestIDMiddleware := middleware.NewRequestIDMiddleware()
	loggingMiddleware := middleware.NewLoggingMiddleware(logger)

	mux := http.NewServeMux()
	healthHandler.RegisterRoutes(mux)

	// Chain order (outermost -> innermost):
	// 1. Recovery: catch panics from any layer
	// 2. Body limit: reject oversized payloads before decoding
	// 3. Request ID: assign a unique ID for tracing
	// 4. Logging: log the request with its status and duration
	return middleware.Chain(mux,
		recoveryMiddleware.Handle,
		bodyLimitMiddleware.Handle,
		requestIDMiddleware.Handle,
		loggingMiddleware.Handle,
	)
}
