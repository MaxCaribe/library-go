package middleware

import (
	"log/slog"
	"net/http"
	"time"

	"github.com/MaxCaribe/library-go/pkg/logging"
)

type StatusRecorder struct {
	http.ResponseWriter
	Status  int
	Written bool
}

func (r *StatusRecorder) WriteHeader(status int) {
	r.Status = status
	r.Written = true
	r.ResponseWriter.WriteHeader(status)
}

func (r *StatusRecorder) Write(b []byte) (int, error) {
	if !r.Written {
		r.Status = http.StatusOK
		r.Written = true
	}
	return r.ResponseWriter.Write(b)
}

type LoggingMiddleware struct {
	logger *slog.Logger
}

func NewLoggingMiddleware(logger *slog.Logger) *LoggingMiddleware {
	return &LoggingMiddleware{logger: logger}
}

func (m *LoggingMiddleware) Handle(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		recorder := &StatusRecorder{ResponseWriter: w, Status: http.StatusOK}
		next.ServeHTTP(recorder, r)
		elapsed := time.Since(start)

		requestID, _ := logging.GetRequestID(r.Context())

		m.logger.InfoContext(r.Context(), "http request",
			"request_id", requestID,
			"method", r.Method,
			"path", r.URL.Path,
			"status", recorder.Status,
			"elapsed_ms", elapsed.Milliseconds(),
		)
	})
}
