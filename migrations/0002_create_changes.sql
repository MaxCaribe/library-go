-- +goose Up
CREATE TABLE changes (
    id            BIGSERIAL PRIMARY KEY,
    book_id       UUID NOT NULL REFERENCES books(id) ON DELETE RESTRICT,
    change_set_id UUID NOT NULL,
    occurred_at   TIMESTAMP WITH TIME ZONE NOT NULL,
    field         TEXT NOT NULL,
    kind          TEXT NOT NULL,
    old_value     TEXT,
    new_value     TEXT
);

CREATE INDEX idx_changes_book_time ON changes (book_id, occurred_at, id);
CREATE INDEX idx_changes_book_field_time ON changes (book_id, field, occurred_at, id);

-- +goose Down
DROP TABLE IF EXISTS changes;
