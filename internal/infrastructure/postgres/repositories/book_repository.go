package repositories

import (
	"context"
	"errors"
	"fmt"

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
		return fmt.Errorf("create book: %w", err)
	}
	return nil
}

func (r *BookRepository) GetByID(ctx context.Context, id string) (domain.Book, error) {
	rows, err := r.pool.Query(ctx, `SELECT `+bookColumns+` FROM books WHERE id = $1`, id)
	if err != nil {
		return domain.Book{}, fmt.Errorf("get book: %w", err)
	}

	book, err := pgx.CollectExactlyOneRow(rows, scanBook)
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.Book{}, domain.ErrNotFound
	}
	if err != nil {
		return domain.Book{}, fmt.Errorf("get book: %w", err)
	}
	return book, nil
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
		return nil, 0, fmt.Errorf("list books: %w", err)
	}

	books, err := pgx.CollectRows(rows, scanBook)
	if err != nil {
		return nil, 0, fmt.Errorf("list books: %w", err)
	}
	return books, total, nil
}

func (r *BookRepository) Update(ctx context.Context, book domain.Book) (domain.Book, error) {
	rows, err := r.pool.Query(ctx, `
		UPDATE books
		SET title = $2, description = $3, published_on = $4, authors = $5, updated_at = $6
		WHERE id = $1
		RETURNING `+bookColumns,
		book.ID, book.Title, book.Description, book.PublishedOn,
		authorNames(book.Authors), book.UpdatedAt,
	)
	if err != nil {
		return domain.Book{}, fmt.Errorf("update book: %w", err)
	}

	updated, err := pgx.CollectExactlyOneRow(rows, scanBook)
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.Book{}, domain.ErrNotFound
	}
	if err != nil {
		return domain.Book{}, fmt.Errorf("update book: %w", err)
	}
	return updated, nil
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
