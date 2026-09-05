package application

import (
	"context"
	"time"

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

type ChangeSortField string

const (
	SortByOccurredAt ChangeSortField = "occurred_at"
	SortByField      ChangeSortField = "field"
)

// ChangeQuery is the whole history query. Empty Fields or Kinds mean "all",
// not "none". From is inclusive and To exclusive, so adjacent windows neither
// overlap nor drop a row.
type ChangeQuery struct {
	BookID     string
	Fields     []domain.ChangeField
	Kinds      []domain.ChangeKind
	From       *time.Time
	To         *time.Time
	SortBy     ChangeSortField
	Descending bool
	Limit      int
	Offset     int
}

type ChangeRepository interface {
	List(ctx context.Context, query ChangeQuery) ([]domain.Change, int, error)
}
