// Package querybuilder assembles WHERE clauses with correct $N placeholders.
// It is deliberately small: only the shapes this service actually needs, so
// there is nothing unused to mislead a reader.
package querybuilder

import (
	"fmt"
	"strings"
)

type Builder struct {
	conditions []string
	args       []any
}

func New() *Builder {
	return &Builder{}
}

// And appends a condition containing a single %d, which is replaced by the
// placeholder index for arg. Example: And("book_id = $%d", id).
func (b *Builder) And(condition string, arg any) *Builder {
	b.args = append(b.args, arg)
	b.conditions = append(b.conditions, fmt.Sprintf(condition, len(b.args)))
	return b
}

// AndAny matches a column against a set. Postgres expands `= ANY($1)` from a
// single array parameter, so the placeholder count does not depend on the
// number of values.
func (b *Builder) AndAny(column string, values []string) *Builder {
	if len(values) == 0 {
		return b
	}
	return b.And(column+" = ANY($%d)", values)
}

func (b *Builder) WhereClause() string {
	if len(b.conditions) == 0 {
		return ""
	}
	return "WHERE " + strings.Join(b.conditions, " AND ")
}

func (b *Builder) Args() []any {
	return b.args
}

// Paginated returns the LIMIT/OFFSET fragment and the full argument list, so
// the same builder serves both the count query and the page query.
func (b *Builder) Paginated(limit, offset int) (string, []any) {
	args := append(append([]any{}, b.args...), limit, offset)
	return fmt.Sprintf("LIMIT $%d OFFSET $%d", len(args)-1, len(args)), args
}
