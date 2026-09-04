-- +goose Up
CREATE TABLE books (
    id           UUID PRIMARY KEY,
    title        TEXT NOT NULL,
    description  TEXT NOT NULL DEFAULT '',
    published_on DATE NOT NULL,
    authors      TEXT[] NOT NULL,
    created_at   TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at   TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_books_created_at ON books (created_at DESC, id DESC);

-- +goose Down
DROP TABLE IF EXISTS books;
