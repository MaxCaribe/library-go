package application

import (
	"context"

	"github.com/MaxCaribe/library-go/internal/domain"
)

type HistoryService struct {
	changes ChangeRepository
	books   BookRepository
}

func NewHistoryService(changes ChangeRepository, books BookRepository) *HistoryService {
	return &HistoryService{changes: changes, books: books}
}

func (s *HistoryService) ListForBook(ctx context.Context, filter ChangeFilter, limit, offset int) ([]domain.Change, int, error) {
	exists, err := s.books.Exists(ctx, filter.BookID)
	if err != nil {
		return nil, 0, err
	}
	if !exists {
		return nil, 0, domain.ErrNotFound
	}
	return s.changes.List(ctx, filter, limit, offset)
}
