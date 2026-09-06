package repositories

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/MaxCaribe/library-go/internal/application"
	"github.com/MaxCaribe/library-go/internal/domain"
	"github.com/MaxCaribe/library-go/internal/infrastructure/postgres/querybuilder"
	"github.com/MaxCaribe/library-go/pkg/utils"
)

const changeColumns = `id, book_id::text, change_set_id::text, occurred_at, field, kind, old_value, new_value`

type ChangeRepository struct {
	pool *pgxpool.Pool
}

func NewChangeRepository(pool *pgxpool.Pool) *ChangeRepository {
	return &ChangeRepository{pool: pool}
}

func (r *ChangeRepository) List(ctx context.Context, f application.ChangeFilter, limit, offset int) ([]domain.Change, int, error) {
	where := conditions(f)

	var total int
	if err := r.pool.QueryRow(ctx, `SELECT count(*) FROM changes `+where.WhereClause(), where.Args()...).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count changes: %w", err)
	}
	if total == 0 {
		return nil, 0, nil
	}

	pagination, args := where.Paginated(limit, offset)
	rows, err := r.pool.Query(ctx, fmt.Sprintf(
		`SELECT %s FROM changes %s %s %s`,
		changeColumns, where.WhereClause(), orderBy(f), pagination,
	), args...)
	if err != nil {
		return nil, 0, fmt.Errorf("select page: %w", err)
	}

	changes, err := pgx.CollectRows(rows, scanChange)
	if err != nil {
		return nil, 0, fmt.Errorf("select page: %w", err)
	}
	return changes, total, nil
}

func conditions(f application.ChangeFilter) *querybuilder.Builder {
	where := querybuilder.New().And("book_id = $%d", f.BookID)

	where.AndAny("field", utils.ToStrings(f.Fields))
	where.AndAny("kind", utils.ToStrings(f.Kinds))

	if f.From != nil {
		where.And("occurred_at >= $%d", *f.From)
	}
	if f.To != nil {
		where.And("occurred_at < $%d", *f.To)
	}
	return where
}

func orderBy(f application.ChangeFilter) string {
	switch {
	case f.SortBy == application.SortByField && f.Descending:
		return "ORDER BY field DESC, id DESC"
	case f.SortBy == application.SortByField:
		return "ORDER BY field ASC, id ASC"
	case f.Descending:
		return "ORDER BY occurred_at DESC, id DESC"
	default:
		return "ORDER BY occurred_at ASC, id ASC"
	}
}

func scanChange(row pgx.CollectableRow) (domain.Change, error) {
	var change domain.Change
	if err := row.Scan(
		&change.ID, &change.BookID, &change.ChangeSetID, &change.OccurredAt,
		&change.Field, &change.Kind, &change.OldValue, &change.NewValue,
	); err != nil {
		return domain.Change{}, err
	}

	change.OccurredAt = change.OccurredAt.UTC()
	return change, nil
}
