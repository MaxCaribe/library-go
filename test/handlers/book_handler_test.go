package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/MaxCaribe/library-go/internal/domain"
	"github.com/MaxCaribe/library-go/internal/infrastructure/httpserver/dto"
	"github.com/MaxCaribe/library-go/internal/infrastructure/httpserver/handlers"
	"github.com/MaxCaribe/library-go/internal/infrastructure/httpserver/middleware"
)

const hobbitBody = `{"title":"The Hobbit","description":"There and back again.",
	"published_on":"1937-09-21","authors":["J.R.R. Tolkien"]}`

func TestCreateBook(t *testing.T) {
	mux := newBookMux(newFakeBookService())

	w := send(t, mux, http.MethodPost, "/books", hobbitBody)
	require.Equal(t, http.StatusCreated, w.Code)

	book := decodeData(t, w)
	assert.NotEmpty(t, book.ID)
	assert.Equal(t, "The Hobbit", book.Title)
	assert.Equal(t, "1937-09-21", book.PublishedOn)
	assert.Equal(t, []string{"J.R.R. Tolkien"}, book.Authors)
	assert.Equal(t, "/books/"+book.ID, w.Header().Get("Location"))
	assert.Contains(t, w.Body.String(), `"published_on":"1937-09-21"`, "a date must go on the wire as YYYY-MM-DD, not a timestamp")
}

func TestCreateBookValidation(t *testing.T) {
	future := time.Now().UTC().AddDate(0, 0, 1).Format(domain.DateLayout)

	tests := map[string]struct {
		body  string
		field string
	}{
		"missing title":     {`{"title":"  ","published_on":"1937-09-21","authors":["Tolkien"]}`, "title"},
		"title too long":    {fmt.Sprintf(`{"title":%q,"published_on":"1937-09-21","authors":["Tolkien"]}`, strings.Repeat("a", 501)), "title"},
		"missing date":      {`{"title":"The Hobbit","authors":["Tolkien"]}`, "published_on"},
		"unparseable date":  {`{"title":"The Hobbit","published_on":"21-09-1937","authors":["Tolkien"]}`, "published_on"},
		"impossible date":   {`{"title":"The Hobbit","published_on":"1937-13-45","authors":["Tolkien"]}`, "published_on"},
		"future date":       {fmt.Sprintf(`{"title":"The Hobbit","published_on":%q,"authors":["Tolkien"]}`, future), "published_on"},
		"no authors":        {`{"title":"The Hobbit","published_on":"1937-09-21","authors":[]}`, "authors"},
		"omitted authors":   {`{"title":"The Hobbit","published_on":"1937-09-21"}`, "authors"},
		"blank author":      {`{"title":"The Hobbit","published_on":"1937-09-21","authors":["Tolkien","  "]}`, "authors[1]"},
		"duplicate authors": {`{"title":"The Hobbit","published_on":"1937-09-21","authors":["Tolkien","Tolkien"]}`, "authors[1]"},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			w := send(t, newBookMux(newFakeBookService()), http.MethodPost, "/books", tc.body)

			require.Equal(t, http.StatusBadRequest, w.Code)
			assert.Contains(t, decodeFields(t, w), tc.field)
		})
	}

	t.Run("reports every invalid field at once", func(t *testing.T) {
		w := send(t, newBookMux(newFakeBookService()), http.MethodPost, "/books", `{"title":"","authors":[]}`)

		require.Equal(t, http.StatusBadRequest, w.Code)
		fields := decodeFields(t, w)
		assert.Len(t, fields, 3)
		assert.Contains(t, fields, "title")
		assert.Contains(t, fields, "published_on")
		assert.Contains(t, fields, "authors")
	})
}

func TestCreateBookMalformedBody(t *testing.T) {
	tests := map[string]struct {
		body    string
		message string
	}{
		"empty body":      {``, "request body is empty"},
		"incomplete json": {`{"title":`, "incomplete json"},
		"broken syntax":   {`{"title" "x"}`, "malformed json at character"},
		"wrong type":      {`{"title":42,"published_on":"1937-09-21","authors":["Tolkien"]}`, `field "title" must be of type string`},
		"unknown field":   {`{"titel":"The Hobbit","published_on":"1937-09-21","authors":["Tolkien"]}`, "invalid json body"},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			w := send(t, newBookMux(newFakeBookService()), http.MethodPost, "/books", tc.body)

			require.Equal(t, http.StatusBadRequest, w.Code)
			assert.Contains(t, decodeError(t, w), tc.message)
		})
	}
}

func TestCreateBookRejectsOversizedBody(t *testing.T) {
	const limit = 64

	handler := middleware.NewBodyLimitMiddleware(limit).Handle(newBookMux(newFakeBookService()))
	w := send(t, handler, http.MethodPost, "/books", hobbitBody)

	require.Equal(t, http.StatusRequestEntityTooLarge, w.Code)
	assert.Contains(t, decodeError(t, w), "must not exceed 64 bytes")
}

func TestGetBook(t *testing.T) {
	mux := newBookMux(newFakeBookService())
	created := decodeData(t, send(t, mux, http.MethodPost, "/books", hobbitBody))

	t.Run("returns the book", func(t *testing.T) {
		w := send(t, mux, http.MethodGet, "/books/"+created.ID, "")

		require.Equal(t, http.StatusOK, w.Code)
		assert.Equal(t, created.ID, decodeData(t, w).ID)
	})

	t.Run("404s an unknown id", func(t *testing.T) {
		w := send(t, mux, http.MethodGet, "/books/missing", "")

		assert.Equal(t, http.StatusNotFound, w.Code)
	})
}

func TestUpdateBook(t *testing.T) {
	mux := newBookMux(newFakeBookService())
	created := decodeData(t, send(t, mux, http.MethodPost, "/books", hobbitBody))

	t.Run("replaces the book", func(t *testing.T) {
		body := `{"title":"The Hobbit, revised","description":"There and back again.",
			"published_on":"1937-09-21","authors":["J.R.R. Tolkien","Christopher Tolkien"]}`
		w := send(t, mux, http.MethodPut, "/books/"+created.ID, body)

		require.Equal(t, http.StatusOK, w.Code)
		updated := decodeData(t, w)
		assert.Equal(t, "The Hobbit, revised", updated.Title)
		assert.Equal(t, []string{"J.R.R. Tolkien", "Christopher Tolkien"}, updated.Authors)
	})

	t.Run("404s an unknown id", func(t *testing.T) {
		w := send(t, mux, http.MethodPut, "/books/missing", hobbitBody)

		assert.Equal(t, http.StatusNotFound, w.Code)
	})

	t.Run("validates the body", func(t *testing.T) {
		w := send(t, mux, http.MethodPut, "/books/"+created.ID, `{"title":"","published_on":"1937-09-21","authors":["T"]}`)

		require.Equal(t, http.StatusBadRequest, w.Code)
		assert.Contains(t, decodeFields(t, w), "title")
	})
}

func TestListBooks(t *testing.T) {
	mux := newBookMux(newFakeBookService())
	for range 3 {
		require.Equal(t, http.StatusCreated, send(t, mux, http.MethodPost, "/books", hobbitBody).Code)
	}

	w := send(t, mux, http.MethodGet, "/books?page=1&page_size=2", "")
	require.Equal(t, http.StatusOK, w.Code)

	var page dto.PaginatedResponse[dto.BookResponse]
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &page))
	assert.Len(t, page.Data, 2)
	assert.Equal(t, 3, page.Total)
	assert.Equal(t, 2, page.TotalPages)
}

func TestUnsupportedMethod(t *testing.T) {
	mux := newBookMux(newFakeBookService())

	req := httptest.NewRequest(http.MethodDelete, "/books/some-id", bytes.NewReader(nil))
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	assert.Equal(t, http.StatusMethodNotAllowed, w.Code)
	assert.Contains(t, w.Header().Get("Allow"), http.MethodGet)
}

func newBookMux(service handlers.BookService) *http.ServeMux {
	mux := http.NewServeMux()
	handlers.NewBookHandler(service, slog.New(slog.DiscardHandler)).RegisterRoutes(mux)
	return mux
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

func decodeError(t *testing.T, w *httptest.ResponseRecorder) string {
	t.Helper()

	var envelope struct {
		Error string `json:"error"`
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &envelope))
	return envelope.Error
}

func decodeFields(t *testing.T, w *httptest.ResponseRecorder) map[string]string {
	t.Helper()

	var envelope struct {
		Error  string            `json:"error"`
		Fields map[string]string `json:"fields"`
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &envelope))
	assert.Equal(t, "validation failed", envelope.Error)
	return envelope.Fields
}

type fakeBookService struct {
	books  map[string]domain.Book
	order  []string
	nextID int
}

func newFakeBookService() *fakeBookService {
	return &fakeBookService{books: map[string]domain.Book{}}
}

func (f *fakeBookService) Create(_ context.Context, book domain.Book) (domain.Book, error) {
	f.nextID++
	book.ID = fmt.Sprintf("book-%d", f.nextID)
	book.CreatedAt = time.Now().UTC()
	book.UpdatedAt = book.CreatedAt

	f.books[book.ID] = book
	f.order = append(f.order, book.ID)
	return book, nil
}

func (f *fakeBookService) Get(_ context.Context, id string) (domain.Book, error) {
	book, ok := f.books[id]
	if !ok {
		return domain.Book{}, domain.ErrNotFound
	}
	return book, nil
}

func (f *fakeBookService) List(_ context.Context, limit, offset int) ([]domain.Book, int, error) {
	total := len(f.order)
	if offset >= total {
		return nil, total, nil
	}

	end := min(offset+limit, total)
	books := make([]domain.Book, 0, end-offset)
	for _, id := range f.order[offset:end] {
		books = append(books, f.books[id])
	}
	return books, total, nil
}

func (f *fakeBookService) Update(_ context.Context, id string, book domain.Book) (domain.Book, error) {
	current, ok := f.books[id]
	if !ok {
		return domain.Book{}, domain.ErrNotFound
	}

	book.ID = id
	book.CreatedAt = current.CreatedAt
	book.UpdatedAt = time.Now().UTC()

	f.books[id] = book
	return book, nil
}
