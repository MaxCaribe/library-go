package integration

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/MaxCaribe/library-go/internal/infrastructure/httpserver/dto"
	"github.com/MaxCaribe/library-go/test/support"
)

const hobbitBody = `{"title":"The Hobbit","description":"There and back again.",
	"published_on":"1937-09-21","authors":["J.R.R. Tolkien","Christopher Tolkien"]}`

func TestBookLifecycle(t *testing.T) {
	mux := support.NewBookAPI(t).Mux

	created := send(t, mux, http.MethodPost, "/books", hobbitBody)
	require.Equal(t, http.StatusCreated, created.Code)

	book := decodeData(t, created)
	assert.NotEmpty(t, book.ID)
	assert.Equal(t, "/books/"+book.ID, created.Header().Get("Location"))
	assert.Equal(t, "1937-09-21", book.PublishedOn)
	assert.Equal(t, []string{"J.R.R. Tolkien", "Christopher Tolkien"}, book.Authors,
		"author order must survive the round trip through the array column")

	fetched := send(t, mux, http.MethodGet, "/books/"+book.ID, "")
	require.Equal(t, http.StatusOK, fetched.Code)
	assert.Equal(t, book, decodeData(t, fetched))

	updated := send(t, mux, http.MethodPut, "/books/"+book.ID, `{"title":"The Hobbit, revised",
		"description":"There and back again.","published_on":"1937-09-21","authors":["J.R.R. Tolkien"]}`)
	require.Equal(t, http.StatusOK, updated.Code)

	after := decodeData(t, updated)
	assert.Equal(t, "The Hobbit, revised", after.Title)
	assert.Equal(t, []string{"J.R.R. Tolkien"}, after.Authors)
	assert.Equal(t, book.CreatedAt, after.CreatedAt, "created_at must not move on update")
	assert.True(t, after.UpdatedAt.After(book.UpdatedAt), "updated_at must advance")

	assert.Equal(t, after, decodeData(t, send(t, mux, http.MethodGet, "/books/"+book.ID, "")),
		"the update must be persisted, not just echoed")
}

func TestUnknownBook(t *testing.T) {
	mux := support.NewBookAPI(t).Mux

	const unknown = "01a06d75-0000-7000-8000-000000000000"

	assert.Equal(t, http.StatusNotFound, send(t, mux, http.MethodGet, "/books/"+unknown, "").Code)
	assert.Equal(t, http.StatusNotFound, send(t, mux, http.MethodPut, "/books/"+unknown, hobbitBody).Code)
}

func TestMalformedBookID(t *testing.T) {
	mux := support.NewBookAPI(t).Mux

	// An id the column cannot hold is a client error, caught before storage.
	w := send(t, mux, http.MethodGet, "/books/not-a-uuid", "")

	require.Equal(t, http.StatusBadRequest, w.Code)
	var envelope struct {
		Fields map[string]string `json:"fields"`
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &envelope))
	assert.Equal(t, "must be a uuid", envelope.Fields["id"])
}

func TestCreateRejectsInvalidBook(t *testing.T) {
	mux := support.NewBookAPI(t).Mux

	w := send(t, mux, http.MethodPost, "/books", `{"title":"","published_on":"1937-09-21","authors":[]}`)

	require.Equal(t, http.StatusBadRequest, w.Code)
	var envelope struct {
		Error  string            `json:"error"`
		Fields map[string]string `json:"fields"`
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &envelope))
	assert.Equal(t, "validation failed", envelope.Error)
	assert.Contains(t, envelope.Fields, "title")
	assert.Contains(t, envelope.Fields, "authors")
}

func TestListBooksPaginates(t *testing.T) {
	mux := support.NewBookAPI(t).Mux

	for _, title := range []string{"First", "Second", "Third"} {
		body := `{"title":"` + title + `","published_on":"2001-01-01","authors":["A"]}`
		require.Equal(t, http.StatusCreated, send(t, mux, http.MethodPost, "/books", body).Code)
	}

	first := decodePage(t, send(t, mux, http.MethodGet, "/books?page=1&page_size=2", ""))
	assert.Len(t, first.Data, 2)
	assert.Equal(t, 3, first.Total)
	assert.Equal(t, 2, first.TotalPages)

	second := decodePage(t, send(t, mux, http.MethodGet, "/books?page=2&page_size=2", ""))
	assert.Len(t, second.Data, 1)

	assert.NotContains(t, ids(second.Data), ids(first.Data)[0], "pages must not overlap")
}

func TestEmptyListIsAnEmptyPage(t *testing.T) {
	page := decodePage(t, send(t, support.NewBookAPI(t).Mux, http.MethodGet, "/books", ""))

	assert.Empty(t, page.Data)
	assert.Equal(t, 0, page.Total)
}

func TestUnsupportedMethod(t *testing.T) {
	w := send(t, support.NewBookAPI(t).Mux, http.MethodDelete, "/books/01a06d75-0000-7000-8000-000000000000", "")

	assert.Equal(t, http.StatusMethodNotAllowed, w.Code)
	assert.Contains(t, w.Header().Get("Allow"), http.MethodGet)
}

func send(t *testing.T, handler http.Handler, method, path, body string) *httptest.ResponseRecorder {
	t.Helper()

	req := httptest.NewRequest(method, path, strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)
	return w
}

func decodeData(t *testing.T, w *httptest.ResponseRecorder) dto.BookResponse {
	t.Helper()

	var envelope struct {
		Data dto.BookResponse `json:"data"`
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &envelope))
	return envelope.Data
}

func decodePage(t *testing.T, w *httptest.ResponseRecorder) dto.PaginatedResponse[dto.BookResponse] {
	t.Helper()

	require.Equal(t, http.StatusOK, w.Code)
	var page dto.PaginatedResponse[dto.BookResponse]
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &page))
	return page
}

func ids(books []dto.BookResponse) []string {
	out := make([]string, len(books))
	for i, book := range books {
		out[i] = book.ID
	}
	return out
}
