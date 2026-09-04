package application

import (
	"context"

	"github.com/MaxCaribe/library-go/internal/domain"
)

// BookRepository is declared here, from the service's point of view, and
// implemented in infrastructure. Update returns the stored book so callers
// never need a second read to learn the fields the repository owns.
type BookRepository interface {
	Create(ctx context.Context, book domain.Book) error
	GetByID(ctx context.Context, id string) (domain.Book, error)
	List(ctx context.Context, limit, offset int) ([]domain.Book, int, error)
	Update(ctx context.Context, book domain.Book) (domain.Book, error)
}
