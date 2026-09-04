package handlers

import (
	"context"
	"log/slog"
	"net/http"

	"github.com/MaxCaribe/library-go/internal/domain"
	"github.com/MaxCaribe/library-go/internal/infrastructure/httpserver/dto"
	"github.com/MaxCaribe/library-go/internal/infrastructure/httpserver/request"
	"github.com/MaxCaribe/library-go/internal/infrastructure/httpserver/response"
)

type BookService interface {
	Create(ctx context.Context, book domain.Book) (domain.Book, error)
	Get(ctx context.Context, id string) (domain.Book, error)
	List(ctx context.Context, limit, offset int) ([]domain.Book, int, error)
	Update(ctx context.Context, id string, book domain.Book) (domain.Book, error)
}

type BookHandler struct {
	service BookService
	logger  *slog.Logger
}

func NewBookHandler(service BookService, logger *slog.Logger) *BookHandler {
	return &BookHandler{service: service, logger: logger}
}

func (h *BookHandler) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("POST /books", h.Create)
	mux.HandleFunc("GET /books", h.List)
	mux.HandleFunc("GET /books/{id}", h.Get)
	mux.HandleFunc("PUT /books/{id}", h.Update)
}

func (h *BookHandler) Create(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	var body dto.BookRequest
	if !request.DecodeJSON(w, r, &body) {
		return
	}

	book, fields := body.Parse()
	if len(fields) > 0 {
		response.ValidationError(w, fields)
		return
	}

	created, err := h.service.Create(ctx, book)
	if err != nil {
		if response.DomainError(w, err) {
			return
		}
		h.logger.ErrorContext(ctx, "create book", "error", err)
		response.Error(w, http.StatusInternalServerError, "failed to create book")
		return
	}

	w.Header().Set("Location", "/books/"+created.ID)
	response.WithData(w, http.StatusCreated, dto.ToBookResponse(created))
}

func (h *BookHandler) List(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	page, pageSize := dto.ParsePagination(r)
	limit, offset := dto.ComputePagination(page, pageSize)

	books, total, err := h.service.List(ctx, limit, offset)
	if err != nil {
		h.logger.ErrorContext(ctx, "list books", "error", err)
		response.Error(w, http.StatusInternalServerError, "failed to list books")
		return
	}

	response.JSON(w, http.StatusOK, dto.NewPaginatedResponse(dto.ToBookResponseList(books), total, page, pageSize))
}

func (h *BookHandler) Get(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	book, err := h.service.Get(ctx, r.PathValue("id"))
	if err != nil {
		if response.DomainError(w, err) {
			return
		}
		h.logger.ErrorContext(ctx, "get book", "error", err)
		response.Error(w, http.StatusInternalServerError, "failed to get book")
		return
	}

	response.WithData(w, http.StatusOK, dto.ToBookResponse(book))
}

func (h *BookHandler) Update(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	var body dto.BookRequest
	if !request.DecodeJSON(w, r, &body) {
		return
	}

	book, fields := body.Parse()
	if len(fields) > 0 {
		response.ValidationError(w, fields)
		return
	}

	updated, err := h.service.Update(ctx, r.PathValue("id"), book)
	if err != nil {
		if response.DomainError(w, err) {
			return
		}
		h.logger.ErrorContext(ctx, "update book", "error", err)
		response.Error(w, http.StatusInternalServerError, "failed to update book")
		return
	}

	response.WithData(w, http.StatusOK, dto.ToBookResponse(updated))
}
