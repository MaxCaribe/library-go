package dto

import (
	"fmt"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/google/uuid"

	"github.com/MaxCaribe/library-go/internal/domain"
)

const (
	maxTitleLen        = 500
	maxDescriptionLen  = 2000
	maxAuthorLen       = 200
	minPublicationYear = 1000
)

func ValidateBookID(id string) map[string]string {
	if _, err := uuid.Parse(id); err != nil {
		return map[string]string{"id": "must be a uuid"}
	}
	return nil
}

func (r BookRequest) Parse() (domain.Book, map[string]string) {
	fields := map[string]string{}

	title := strings.TrimSpace(r.Title)
	switch length := utf8.RuneCountInString(title); {
	case length == 0:
		fields["title"] = "is required"
	case length > maxTitleLen:
		fields["title"] = fmt.Sprintf("must be at most %d characters", maxTitleLen)
	}

	description := strings.TrimSpace(r.Description)
	if utf8.RuneCountInString(description) > maxDescriptionLen {
		fields["description"] = fmt.Sprintf("must be at most %d characters", maxDescriptionLen)
	}

	publishedOn := parsePublicationDate(strings.TrimSpace(r.PublishedOn), fields)
	authors := parseAuthors(r.Authors, fields)

	if len(fields) > 0 {
		return domain.Book{}, fields
	}

	return domain.Book{
		Title:       title,
		Description: description,
		PublishedOn: publishedOn,
		Authors:     authors,
	}, nil
}

func parsePublicationDate(value string, fields map[string]string) time.Time {
	if value == "" {
		fields["published_on"] = "is required"
		return time.Time{}
	}

	date, err := time.Parse(domain.DateLayout, value)
	if err != nil {
		fields["published_on"] = "must be a date in YYYY-MM-DD format"
		return time.Time{}
	}

	switch {
	case date.Year() < minPublicationYear:
		fields["published_on"] = fmt.Sprintf("must not be earlier than year %d", minPublicationYear)
	case date.After(todayUTC()):
		fields["published_on"] = "must not be in the future"
	}
	return date
}

func parseAuthors(values []string, fields map[string]string) []domain.Author {
	if len(values) == 0 {
		fields["authors"] = "must contain at least one author"
		return nil
	}

	authors := make([]domain.Author, 0, len(values))
	seen := make(map[string]bool, len(values))

	for i, value := range values {
		name := strings.TrimSpace(value)
		key := fmt.Sprintf("authors[%d]", i)

		switch length := utf8.RuneCountInString(name); {
		case length == 0:
			fields[key] = "must not be empty"
			continue
		case length > maxAuthorLen:
			fields[key] = fmt.Sprintf("must be at most %d characters", maxAuthorLen)
			continue
		}

		if seen[name] {
			fields[key] = fmt.Sprintf("duplicates %q", name)
			continue
		}

		seen[name] = true
		authors = append(authors, domain.Author{Name: name})
	}
	return authors
}

func todayUTC() time.Time {
	year, month, day := time.Now().UTC().Date()
	return time.Date(year, month, day, 0, 0, 0, 0, time.UTC)
}

func ToBookResponse(book domain.Book) BookResponse {
	authors := make([]string, len(book.Authors))
	for i, author := range book.Authors {
		authors[i] = author.Name
	}

	publishedOn := ""
	if !book.PublishedOn.IsZero() {
		publishedOn = book.PublishedOn.Format(domain.DateLayout)
	}

	return BookResponse{
		ID:          book.ID,
		Title:       book.Title,
		Description: book.Description,
		PublishedOn: publishedOn,
		Authors:     authors,
		CreatedAt:   book.CreatedAt,
		UpdatedAt:   book.UpdatedAt,
	}
}

func ToBookResponses(books []domain.Book) []BookResponse {
	out := make([]BookResponse, len(books))
	for i := range books {
		out[i] = ToBookResponse(books[i])
	}
	return out
}
