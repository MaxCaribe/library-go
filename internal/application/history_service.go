package application

import (
	"context"
	"log/slog"

	"github.com/MaxCaribe/library-go/internal/domain"
)

type HistoryService struct {
	changes ChangeRepository
	books   BookRepository
	logger  *slog.Logger
}

func NewHistoryService(changes ChangeRepository, books BookRepository, logger *slog.Logger) *HistoryService {
	return &HistoryService{changes: changes, books: books, logger: logger}
}

// ListForBook checks the book exists first: an unknown id returning an empty
// page would look like "this book has never changed" rather than "no such book".
func (s *HistoryService) ListForBook(ctx context.Context, query ChangeQuery) ([]domain.Change, int, error) {
	if _, err := s.books.GetByID(ctx, query.BookID); err != nil {
		return nil, 0, err
	}
	return s.changes.List(ctx, query)
}
