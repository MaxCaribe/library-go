package application

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/google/uuid"

	"github.com/MaxCaribe/library-go/internal/domain"
)

type BookService struct {
	repo   BookRepository
	logger *slog.Logger
}

func NewBookService(repo BookRepository, logger *slog.Logger) *BookService {
	return &BookService{repo: repo, logger: logger}
}

func (s *BookService) Create(ctx context.Context, book domain.Book) (domain.Book, error) {
	// v7 is time-ordered, so the primary key index also reflects insertion order.
	id, err := uuid.NewV7()
	if err != nil {
		return domain.Book{}, fmt.Errorf("generate book id: %w", err)
	}

	now := time.Now().UTC()
	book.ID = id.String()
	book.CreatedAt = now
	book.UpdatedAt = now

	if err := s.repo.Create(ctx, book); err != nil {
		return domain.Book{}, err
	}

	s.logger.InfoContext(ctx, "book created", "book_id", book.ID, "title", book.Title)
	return book, nil
}

func (s *BookService) Get(ctx context.Context, id string) (domain.Book, error) {
	return s.repo.GetByID(ctx, id)
}

func (s *BookService) List(ctx context.Context, limit, offset int) ([]domain.Book, int, error) {
	return s.repo.List(ctx, limit, offset)
}

func (s *BookService) Update(ctx context.Context, id string, book domain.Book) (domain.Book, error) {
	book.ID = id

	updated, err := s.repo.Update(ctx, book)
	if err != nil {
		return domain.Book{}, err
	}

	s.logger.InfoContext(ctx, "book updated", "book_id", updated.ID)
	return updated, nil
}
