package dto_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/MaxCaribe/library-go/internal/domain"
	"github.com/MaxCaribe/library-go/internal/infrastructure/httpserver/dto"
	"github.com/MaxCaribe/library-go/internal/infrastructure/httpserver/response"
)

// The response types are generated from the OpenAPI spec, so a generator
// upgrade or a spec edit can change the wire format without breaking the
// build. These tests pin the exact JSON clients receive.

func TestBookResponseWireFormat(t *testing.T) {
	w := httptest.NewRecorder()
	response.WithData(w, http.StatusOK, dto.ToBookResponse(hobbit()))

	assert.JSONEq(t, `{
		"data": {
			"id": "0199a1b2-c3d4-7e5f-8a9b-0c1d2e3f4a5b",
			"title": "The Hobbit",
			"description": "There and back again.",
			"published_on": "1937-09-21",
			"authors": ["J.R.R. Tolkien", "Christopher Tolkien"],
			"created_at": "2026-01-02T03:04:05Z",
			"updated_at": "2026-03-04T05:06:07Z"
		}
	}`, w.Body.String())
}

func TestBookListResponseWireFormat(t *testing.T) {
	encoded, err := json.Marshal(dto.NewPaginatedResponse(dto.ToBookResponses([]domain.Book{hobbit()}), 42, 2, 10))
	require.NoError(t, err)

	assert.JSONEq(t, `{
		"data": [{
			"id": "0199a1b2-c3d4-7e5f-8a9b-0c1d2e3f4a5b",
			"title": "The Hobbit",
			"description": "There and back again.",
			"published_on": "1937-09-21",
			"authors": ["J.R.R. Tolkien", "Christopher Tolkien"],
			"created_at": "2026-01-02T03:04:05Z",
			"updated_at": "2026-03-04T05:06:07Z"
		}],
		"total": 42,
		"page": 2,
		"page_size": 10,
		"total_pages": 5
	}`, string(encoded))
}

func TestEmptyBookListIsAnArrayNotNull(t *testing.T) {
	encoded, err := json.Marshal(dto.NewPaginatedResponse(dto.ToBookResponses(nil), 0, 1, 10))
	require.NoError(t, err)

	assert.JSONEq(t, `{"data": [], "total": 0, "page": 1, "page_size": 10, "total_pages": 0}`, string(encoded))
}

func TestBookRequestDecodesEveryField(t *testing.T) {
	var request dto.BookRequest
	require.NoError(t, json.Unmarshal([]byte(`{
		"title": "The Hobbit",
		"description": "There and back again.",
		"published_on": "1937-09-21",
		"authors": ["J.R.R. Tolkien"]
	}`), &request))

	book, fields := request.Parse()
	require.Empty(t, fields)
	assert.Equal(t, "The Hobbit", book.Title)
	assert.Equal(t, "There and back again.", book.Description)
	assert.Equal(t, time.Date(1937, 9, 21, 0, 0, 0, 0, time.UTC), book.PublishedOn)
	assert.Equal(t, []domain.Author{{Name: "J.R.R. Tolkien"}}, book.Authors)
}

func hobbit() domain.Book {
	return domain.Book{
		ID:          "0199a1b2-c3d4-7e5f-8a9b-0c1d2e3f4a5b",
		Title:       "The Hobbit",
		Description: "There and back again.",
		PublishedOn: time.Date(1937, 9, 21, 0, 0, 0, 0, time.UTC),
		Authors:     []domain.Author{{Name: "J.R.R. Tolkien"}, {Name: "Christopher Tolkien"}},
		CreatedAt:   time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC),
		UpdatedAt:   time.Date(2026, 3, 4, 5, 6, 7, 0, time.UTC),
	}
}
