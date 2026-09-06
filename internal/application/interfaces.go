package application

import (
	"context"

	"github.com/MaxCaribe/library-go/internal/domain"
)

type BookRepository interface {
	Create(ctx context.Context, book domain.Book) error
	GetByID(ctx context.Context, id string) (domain.Book, error)
	Exists(ctx context.Context, id string) (bool, error)
	List(ctx context.Context, limit, offset int) ([]domain.Book, int, error)
	Update(ctx context.Context, book domain.Book) (domain.Book, error)
}

type ChangeRepository interface {
	List(ctx context.Context, filter ChangeFilter, limit, offset int) ([]domain.Change, int, error)
}
