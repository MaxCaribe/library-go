package handlers

import (
	"context"
	"log/slog"
	"net/http"

	"github.com/MaxCaribe/library-go/internal/application"
	"github.com/MaxCaribe/library-go/internal/domain"
	"github.com/MaxCaribe/library-go/internal/infrastructure/httpserver/dto"
	"github.com/MaxCaribe/library-go/internal/infrastructure/httpserver/response"
)

type HistoryService interface {
	ListForBook(ctx context.Context, filter application.ChangeFilter, limit, offset int) ([]domain.Change, int, error)
}

type HistoryHandler struct {
	service HistoryService
	logger  *slog.Logger
}

func NewHistoryHandler(service HistoryService, logger *slog.Logger) *HistoryHandler {
	return &HistoryHandler{service: service, logger: logger}
}

func (h *HistoryHandler) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("GET /books/{id}/history", h.List)
}

func (h *HistoryHandler) List(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	id := r.PathValue("id")
	if fields := dto.ValidateBookID(id); len(fields) > 0 {
		response.ValidationError(w, fields)
		return
	}

	filter, fields := dto.ParseChangeFilter(r, id)
	if len(fields) > 0 {
		response.ValidationError(w, fields)
		return
	}

	page, pageSize := dto.ParsePagination(r)
	limit, offset := dto.ComputePagination(page, pageSize)

	changes, total, err := h.service.ListForBook(ctx, filter, limit, offset)
	if err != nil {
		if response.DomainError(w, err) {
			return
		}
		h.logger.ErrorContext(ctx, "list book history", "error", err)
		response.Error(w, http.StatusInternalServerError, "failed to list history")
		return
	}

	response.JSON(w, http.StatusOK, dto.NewPaginatedResponse(dto.ToChangeResponses(changes), total, page, pageSize))
}
