package dto_test

import (
	"encoding/json"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/MaxCaribe/library-go/internal/domain"
	"github.com/MaxCaribe/library-go/internal/infrastructure/httpserver/dto"
)

func TestBookRequestValidation(t *testing.T) {
	future := time.Now().UTC().AddDate(0, 0, 1).Format(domain.DateLayout)

	tests := map[string]struct {
		body  string
		field string
	}{
		"missing title":        {`{"title":"  ","published_on":"1937-09-21","authors":["Tolkien"]}`, "title"},
		"title too long":       {fmt.Sprintf(`{"title":%q,"published_on":"1937-09-21","authors":["Tolkien"]}`, strings.Repeat("a", 501)), "title"},
		"description too long": {fmt.Sprintf(`{"title":"The Hobbit","description":%q,"published_on":"1937-09-21","authors":["Tolkien"]}`, strings.Repeat("a", 2001)), "description"},
		"missing date":         {`{"title":"The Hobbit","authors":["Tolkien"]}`, "published_on"},
		"unparseable date":     {`{"title":"The Hobbit","published_on":"21-09-1937","authors":["Tolkien"]}`, "published_on"},
		"impossible date":      {`{"title":"The Hobbit","published_on":"1937-13-45","authors":["Tolkien"]}`, "published_on"},
		"date before 1000":     {`{"title":"The Hobbit","published_on":"0999-01-01","authors":["Tolkien"]}`, "published_on"},
		"future date":          {fmt.Sprintf(`{"title":"The Hobbit","published_on":%q,"authors":["Tolkien"]}`, future), "published_on"},
		"no authors":           {`{"title":"The Hobbit","published_on":"1937-09-21","authors":[]}`, "authors"},
		"omitted authors":      {`{"title":"The Hobbit","published_on":"1937-09-21"}`, "authors"},
		"blank author":         {`{"title":"The Hobbit","published_on":"1937-09-21","authors":["Tolkien","  "]}`, "authors[1]"},
		"author too long":      {fmt.Sprintf(`{"title":"The Hobbit","published_on":"1937-09-21","authors":[%q]}`, strings.Repeat("a", 201)), "authors[0]"},
		"duplicate authors":    {`{"title":"The Hobbit","published_on":"1937-09-21","authors":["Tolkien","Tolkien"]}`, "authors[1]"},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			_, fields := parse(t, tc.body)

			require.NotEmpty(t, fields)
			assert.Contains(t, fields, tc.field)
		})
	}
}

func TestBookRequestReportsEveryInvalidFieldAtOnce(t *testing.T) {
	_, fields := parse(t, `{"title":"","authors":[]}`)

	assert.Len(t, fields, 3)
	assert.Contains(t, fields, "title")
	assert.Contains(t, fields, "published_on")
	assert.Contains(t, fields, "authors")
}

func TestBookRequestTrimsBeforeValidating(t *testing.T) {
	book, fields := parse(t, `{"title":"  The Hobbit ","description":" There and back again. ","published_on":" 1937-09-21 ","authors":["  J.R.R. Tolkien "]}`)

	require.Empty(t, fields)
	assert.Equal(t, "The Hobbit", book.Title)
	assert.Equal(t, "There and back again.", book.Description)
	assert.Equal(t, []domain.Author{{Name: "J.R.R. Tolkien"}}, book.Authors)
}

func TestBookRequestAcceptsAnOmittedDescription(t *testing.T) {
	book, fields := parse(t, `{"title":"The Hobbit","published_on":"1937-09-21","authors":["Tolkien"]}`)

	require.Empty(t, fields)
	assert.Equal(t, "", book.Description)
}

func parse(t *testing.T, body string) (domain.Book, map[string]string) {
	t.Helper()

	var request dto.BookRequest
	require.NoError(t, json.Unmarshal([]byte(body), &request))
	return request.Parse()
}
