package handlers

import (
	"net/http"
	"time"

	"github.com/MaxCaribe/library-go/internal/infrastructure/httpserver/response"
)

type HeartHandler struct{}

func NewHeartHandler() *HeartHandler {
	return &HeartHandler{}
}

func (h *HeartHandler) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("GET /heartbeat", h.Heartbeat)
}

func (h *HeartHandler) Heartbeat(w http.ResponseWriter, _ *http.Request) {
	response.JSON(w, http.StatusOK, map[string]any{
		"status":    "ok",
		"timestamp": time.Now().UTC().Format(time.RFC3339),
	})
}
