package middleware

import (
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestRecoveryMiddleware(t *testing.T) {
	panicking := http.HandlerFunc(func(_ http.ResponseWriter, _ *http.Request) {
		panic("boom")
	})

	h := NewRecoveryMiddleware(slog.New(slog.DiscardHandler)).Handle(panicking)

	w := httptest.NewRecorder()
	assert.NotPanics(t, func() {
		h.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/books", nil))
	})

	assert.Equal(t, http.StatusInternalServerError, w.Code)
	assert.JSONEq(t, `{"error":"internal server error"}`, w.Body.String())
}

func TestRequestIDMiddleware(t *testing.T) {
	var seen string
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		seen = w.Header().Get(RequestIDHeader)
	})
	h := NewRequestIDMiddleware().Handle(next)

	t.Run("generates an id when absent", func(t *testing.T) {
		w := httptest.NewRecorder()
		h.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/books", nil))

		assert.NotEmpty(t, w.Header().Get(RequestIDHeader))
		assert.Equal(t, w.Header().Get(RequestIDHeader), seen)
	})

	t.Run("preserves an incoming id", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/books", nil)
		req.Header.Set(RequestIDHeader, "abc-123")
		w := httptest.NewRecorder()
		h.ServeHTTP(w, req)

		assert.Equal(t, "abc-123", w.Header().Get(RequestIDHeader))
	})
}
