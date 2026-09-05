package integration

import (
	"encoding/json"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/MaxCaribe/library-go/internal/infrastructure/httpserver/dto"
	"github.com/MaxCaribe/library-go/test/support"
)

func TestHistoryEndpointReturnsRenderedChanges(t *testing.T) {
	api := support.NewBookAPI(t)
	id := editedBook(t, api)

	page := historyPage(t, api, id, "")

	require.Len(t, page.Data, 2)
	assert.Equal(t, 2, page.Total)

	// Newest first by default.
	assert.Equal(t, `Author "Christopher Tolkien" was removed`, page.Data[0].Description)
	assert.Equal(t, `Title changed from "The Hobbit" to "The Hobbit, revised"`, page.Data[1].Description)

	assert.Equal(t, "title", page.Data[1].Field)
	assert.Equal(t, "set", page.Data[1].Kind)
	assert.Equal(t, "The Hobbit", *page.Data[1].OldValue)
	assert.Equal(t, page.Data[0].ChangeSetID, page.Data[1].ChangeSetID, "one request is one change set")
}

func TestHistoryEndpointFilters(t *testing.T) {
	api := support.NewBookAPI(t)
	id := editedBook(t, api)

	tests := map[string]struct {
		query    string
		expected []string
	}{
		"by field":            {"?field=title", []string{"title"}},
		"by several fields":   {"?field=title,authors", []string{"authors", "title"}},
		"by kind":             {"?kind=removed", []string{"authors"}},
		"field and kind":      {"?field=title&kind=removed", nil},
		"repeated parameters": {"?field=title&field=authors", []string{"authors", "title"}},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			page := historyPage(t, api, id, tc.query)

			var got []string
			for _, change := range page.Data {
				got = append(got, change.Field)
			}
			assert.Equal(t, tc.expected, got)
			assert.Equal(t, len(tc.expected), page.Total)
		})
	}
}

func TestHistoryEndpointOrders(t *testing.T) {
	api := support.NewBookAPI(t)
	id := editedBook(t, api)

	newestFirst := historyPage(t, api, id, "?order=desc")
	oldestFirst := historyPage(t, api, id, "?order=asc")

	require.Len(t, newestFirst.Data, 2)
	assert.Equal(t, newestFirst.Data[0].ID, oldestFirst.Data[1].ID, "asc must be the exact reverse of desc")
	assert.Equal(t, newestFirst.Data[1].ID, oldestFirst.Data[0].ID)
}

// Rows in one change set share an occurred_at, so without the id tiebreaker
// paging could return the same row twice and drop another.
func TestHistoryEndpointPagesWithoutOverlap(t *testing.T) {
	api := support.NewBookAPI(t)
	id := editedBook(t, api)

	first := historyPage(t, api, id, "?page=1&page_size=1")
	second := historyPage(t, api, id, "?page=2&page_size=1")

	require.Len(t, first.Data, 1)
	require.Len(t, second.Data, 1)
	assert.Equal(t, 2, first.TotalPages)
	assert.NotEqual(t, first.Data[0].ID, second.Data[0].ID, "pages must not repeat a row")
}

func TestHistoryEndpointRejectsUnknownFilters(t *testing.T) {
	api := support.NewBookAPI(t)
	id := editedBook(t, api)

	tests := map[string]struct{ query, field string }{
		"unknown field": {"?field=isbn", "field"},
		"unknown kind":  {"?kind=deleted", "kind"},
		"unknown sort":  {"?sort=title", "sort"},
		"unknown order": {"?order=sideways", "order"},
		"bad timestamp": {"?from=yesterday", "from"},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			w := send(t, api.Mux, http.MethodGet, "/books/"+id+"/history"+tc.query, "")

			require.Equal(t, http.StatusBadRequest, w.Code, "an unrecognised filter must not be silently ignored")
			var envelope struct {
				Fields map[string]string `json:"fields"`
			}
			require.NoError(t, json.Unmarshal(w.Body.Bytes(), &envelope))
			assert.Contains(t, envelope.Fields, tc.field)
		})
	}
}

func TestHistoryEndpointOnAnUnknownBook(t *testing.T) {
	api := support.NewBookAPI(t)

	assert.Equal(t, http.StatusNotFound,
		send(t, api.Mux, http.MethodGet, "/books/01a06d75-0000-7000-8000-000000000000/history", "").Code,
		"an empty page would read as 'never changed' rather than 'no such book'")
	assert.Equal(t, http.StatusBadRequest,
		send(t, api.Mux, http.MethodGet, "/books/not-a-uuid/history", "").Code)
}

func TestHistoryEndpointOnAnUneditedBook(t *testing.T) {
	api := support.NewBookAPI(t)
	book := decodeData(t, send(t, api.Mux, http.MethodPost, "/books", hobbitBody))

	page := historyPage(t, api, book.ID, "")

	assert.Empty(t, page.Data)
	assert.Equal(t, 0, page.Total)
}

// editedBook creates a book and makes one edit touching two fields.
func editedBook(t *testing.T, api support.API) string {
	t.Helper()

	book := decodeData(t, send(t, api.Mux, http.MethodPost, "/books", hobbitBody))
	require.Equal(t, http.StatusOK, send(t, api.Mux, http.MethodPut, "/books/"+book.ID, `{"title":"The Hobbit, revised",
		"description":"There and back again.","published_on":"1937-09-21","authors":["J.R.R. Tolkien"]}`).Code)
	return book.ID
}

func historyPage(t *testing.T, api support.API, bookID, query string) dto.PaginatedResponse[dto.ChangeResponse] {
	t.Helper()

	w := send(t, api.Mux, http.MethodGet, "/books/"+bookID+"/history"+query, "")
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())

	var page dto.PaginatedResponse[dto.ChangeResponse]
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &page))
	return page
}
