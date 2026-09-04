package request_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/MaxCaribe/library-go/internal/infrastructure/httpserver/middleware"
	"github.com/MaxCaribe/library-go/internal/infrastructure/httpserver/request"
)

func TestDecodeJSONRejectsMalformedBodies(t *testing.T) {
	tests := map[string]struct {
		body    string
		message string
	}{
		"empty body":      {``, "request body is empty"},
		"incomplete json": {`{"title":`, "request body contains incomplete json"},
		"broken syntax":   {`{"title" "x"}`, "malformed json at character"},
		"wrong type":      {`{"title":42}`, `field "title" must be of type string`},
		"unknown field":   {`{"titel":"The Hobbit"}`, "invalid json body"},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			w, ok := decode(tc.body, nil)

			assert.False(t, ok, "decoding should have failed")
			require.Equal(t, http.StatusBadRequest, w.Code)
			assert.Contains(t, errorMessage(t, w), tc.message)
		})
	}
}

func TestDecodeJSONAcceptsAValidBody(t *testing.T) {
	w, ok := decode(`{"title":"The Hobbit"}`, nil)

	assert.True(t, ok)
	assert.Empty(t, w.Body.String(), "nothing should be written on success")
}

func TestDecodeJSONReportsAnOversizedBody(t *testing.T) {
	const limit = 32

	w, ok := decode(`{"title":"a title comfortably longer than the limit"}`, middleware.NewBodyLimitMiddleware(limit).Handle)

	assert.False(t, ok)
	require.Equal(t, http.StatusRequestEntityTooLarge, w.Code)
	assert.Contains(t, errorMessage(t, w), "must not exceed 32 bytes")
}

// decode runs DecodeJSON behind an optional middleware, which is how the body
// limit reaches it in production.
func decode(body string, wrap func(http.Handler) http.Handler) (*httptest.ResponseRecorder, bool) {
	var target struct {
		Title string `json:"title"`
	}
	ok := false

	var handler http.Handler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ok = request.DecodeJSON(w, r, &target)
	})
	if wrap != nil {
		handler = wrap(handler)
	}

	req := httptest.NewRequest(http.MethodPost, "/books", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)
	return w, ok
}

func errorMessage(t *testing.T, w *httptest.ResponseRecorder) string {
	t.Helper()

	var envelope struct {
		Error string `json:"error"`
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &envelope))
	return envelope.Error
}
