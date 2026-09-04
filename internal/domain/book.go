package domain

import "time"

const DateLayout = "2006-01-02"

type Author struct {
	Name string
}

type Book struct {
	ID          string
	Title       string
	Description string
	PublishedOn time.Time
	Authors     []Author
	CreatedAt   time.Time
	UpdatedAt   time.Time
}
