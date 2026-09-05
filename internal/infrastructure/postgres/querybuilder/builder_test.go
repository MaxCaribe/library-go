package querybuilder_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/MaxCaribe/library-go/internal/infrastructure/postgres/querybuilder"
)

func TestNoConditions(t *testing.T) {
	b := querybuilder.New()

	assert.Empty(t, b.WhereClause())
	assert.Empty(t, b.Args())
}

func TestPlaceholdersAreNumberedInOrder(t *testing.T) {
	b := querybuilder.New().
		And("book_id = $%d", "abc").
		And("occurred_at >= $%d", "2026-01-01").
		AndAny("field", []string{"title", "authors"})

	assert.Equal(t, "WHERE book_id = $1 AND occurred_at >= $2 AND field = ANY($3)", b.WhereClause())
	assert.Equal(t, []any{"abc", "2026-01-01", []string{"title", "authors"}}, b.Args())
}

func TestAndAnyIgnoresAnEmptySet(t *testing.T) {
	b := querybuilder.New().And("book_id = $%d", "abc").AndAny("field", nil)

	assert.Equal(t, "WHERE book_id = $1", b.WhereClause(), "an empty filter must not exclude everything")
	assert.Len(t, b.Args(), 1)
}

func TestPaginatedContinuesTheNumbering(t *testing.T) {
	b := querybuilder.New().And("book_id = $%d", "abc")

	clause, args := b.Paginated(10, 20)

	assert.Equal(t, "LIMIT $2 OFFSET $3", clause)
	assert.Equal(t, []any{"abc", 10, 20}, args)
	assert.Equal(t, []any{"abc"}, b.Args(), "the count query must still see only its own args")
}
