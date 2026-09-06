package repositories

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/MaxCaribe/library-go/internal/domain"
)

const bookColumns = `id::text, title, description, published_on, authors, created_at, updated_at`

type BookRepository struct {
	pool *pgxpool.Pool
}

func NewBookRepository(pool *pgxpool.Pool) *BookRepository {
	return &BookRepository{pool: pool}
}

func (r *BookRepository) Create(ctx context.Context, book domain.Book) error {
	_, err := r.pool.Exec(ctx, `
		INSERT INTO books (id, title, description, published_on, authors, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7)`,
		book.ID, book.Title, book.Description, book.PublishedOn,
		authorNames(book.Authors), book.CreatedAt, book.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("insert book: %w", err)
	}
	return nil
}

func (r *BookRepository) GetByID(ctx context.Context, id string) (domain.Book, error) {
	rows, err := r.pool.Query(ctx, `SELECT `+bookColumns+` FROM books WHERE id = $1`, id)
	if err != nil {
		return domain.Book{}, err
	}

	book, err := pgx.CollectExactlyOneRow(rows, scanBook)
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.Book{}, domain.ErrNotFound
	}
	return book, err
}

func (r *BookRepository) Exists(ctx context.Context, id string) (bool, error) {
	var exists bool
	if err := r.pool.QueryRow(ctx, `SELECT EXISTS (SELECT 1 FROM books WHERE id = $1)`, id).Scan(&exists); err != nil {
		return false, fmt.Errorf("check book exists: %w", err)
	}
	return exists, nil
}

func (r *BookRepository) List(ctx context.Context, limit, offset int) ([]domain.Book, int, error) {
	var total int
	if err := r.pool.QueryRow(ctx, `SELECT count(*) FROM books`).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count books: %w", err)
	}
	if total == 0 {
		return nil, 0, nil
	}

	rows, err := r.pool.Query(ctx, `
		SELECT `+bookColumns+` FROM books
		ORDER BY created_at DESC, id DESC
		LIMIT $1 OFFSET $2`, limit, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("select page: %w", err)
	}

	books, err := pgx.CollectRows(rows, scanBook)
	if err != nil {
		return nil, 0, fmt.Errorf("select page: %w", err)
	}
	return books, total, nil
}

// FOR UPDATE is what makes the diff correct: under READ COMMITTED two
// concurrent updates would otherwise both read the same "before" and each
// record a change from a state that no longer existed.
func (r *BookRepository) Update(ctx context.Context, book domain.Book) (domain.Book, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return domain.Book{}, fmt.Errorf("begin transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	rows, err := tx.Query(ctx, `SELECT `+bookColumns+` FROM books WHERE id = $1 FOR UPDATE`, book.ID)
	if err != nil {
		return domain.Book{}, fmt.Errorf("lock book: %w", err)
	}

	current, err := pgx.CollectExactlyOneRow(rows, scanBook)
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.Book{}, domain.ErrNotFound
	}
	if err != nil {
		return domain.Book{}, fmt.Errorf("lock book: %w", err)
	}

	// Read while holding the lock, so occurred_at orders as the updates serialised.
	now := time.Now().UTC()

	changes := domain.Diff(current, book)
	if len(changes) == 0 {
		return current, tx.Commit(ctx)
	}

	rows, err = tx.Query(ctx, `
		UPDATE books
		SET title = $2, description = $3, published_on = $4, authors = $5, updated_at = $6
		WHERE id = $1
		RETURNING `+bookColumns,
		book.ID, book.Title, book.Description, book.PublishedOn,
		authorNames(book.Authors), now,
	)
	if err != nil {
		return domain.Book{}, fmt.Errorf("update book row: %w", err)
	}

	updated, err := pgx.CollectExactlyOneRow(rows, scanBook)
	if err != nil {
		return domain.Book{}, fmt.Errorf("update book row: %w", err)
	}

	if err := recordChanges(ctx, tx, book.ID, changes, now); err != nil {
		return domain.Book{}, err
	}

	if err := tx.Commit(ctx); err != nil {
		return domain.Book{}, fmt.Errorf("commit: %w", err)
	}
	return updated, nil
}

// Inserted in Diff's order, so the ascending id encodes order within the set.
func recordChanges(ctx context.Context, tx pgx.Tx, bookID string, changes []domain.Change, occurredAt time.Time) error {
	if len(changes) == 0 {
		return nil
	}

	changeSetID, err := uuid.NewV7()
	if err != nil {
		return fmt.Errorf("generate change set id: %w", err)
	}

	for _, change := range changes {
		if _, err := tx.Exec(ctx, `
			INSERT INTO changes (book_id, change_set_id, occurred_at, field, kind, old_value, new_value)
			VALUES ($1, $2, $3, $4, $5, $6, $7)`,
			bookID, changeSetID.String(), occurredAt, change.Field, change.Kind, change.OldValue, change.NewValue,
		); err != nil {
			return fmt.Errorf("record change: %w", err)
		}
	}
	return nil
}

func scanBook(row pgx.CollectableRow) (domain.Book, error) {
	var book domain.Book
	var names []string

	if err := row.Scan(
		&book.ID, &book.Title, &book.Description, &book.PublishedOn,
		&names, &book.CreatedAt, &book.UpdatedAt,
	); err != nil {
		return domain.Book{}, err
	}

	book.PublishedOn = book.PublishedOn.UTC()
	book.CreatedAt = book.CreatedAt.UTC()
	book.UpdatedAt = book.UpdatedAt.UTC()
	book.Authors = toAuthors(names)
	return book, nil
}

func authorNames(authors []domain.Author) []string {
	names := make([]string, len(authors))
	for i, author := range authors {
		names[i] = author.Name
	}
	return names
}

func toAuthors(names []string) []domain.Author {
	authors := make([]domain.Author, len(names))
	for i, name := range names {
		authors[i] = domain.Author{Name: name}
	}
	return authors
}
