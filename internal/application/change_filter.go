package application

import (
	"time"

	"github.com/MaxCaribe/library-go/internal/domain"
)

type ChangeSortField string

const (
	SortByOccurredAt ChangeSortField = "occurred_at"
	SortByField      ChangeSortField = "field"
)

// ChangeFilter selects changes. Empty Fields or Kinds mean "all", not "none".
// From is inclusive and To exclusive, so adjacent windows neither overlap nor
// drop a row.
type ChangeFilter struct {
	BookID     string
	Fields     []domain.ChangeField
	Kinds      []domain.ChangeKind
	From       *time.Time
	To         *time.Time
	SortBy     ChangeSortField
	Descending bool
}
