package docs

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"

	"github.com/MaxCaribe/library-go/api"
	"github.com/MaxCaribe/library-go/internal/infrastructure/httpserver/handlers"
)

func TestSpecIsValidYAML(t *testing.T) {
	spec := parseSpec(t)

	assert.Equal(t, "3.0.3", spec.OpenAPI)
	assert.NotEmpty(t, spec.Info.Title)
	assert.NotEmpty(t, spec.Info.Version)
	assert.NotEmpty(t, spec.Paths)
}

// TestEveryDocumentedRouteIsRegistered walks the spec and resolves each
// documented operation against the real mux. It is the only thing keeping a
// hand-written spec honest: rename a route and the spec stops matching.
func TestEveryDocumentedRouteIsRegistered(t *testing.T) {
	mux := newMux()

	for path, operations := range parseSpec(t).Paths {
		for method := range operations {
			if !isHTTPMethod(method) {
				continue
			}

			t.Run(strings.ToUpper(method)+" "+path, func(t *testing.T) {
				req := httptest.NewRequest(strings.ToUpper(method), substitutePathParams(path), nil)

				_, pattern := mux.Handler(req)
				assert.NotEmpty(t, pattern, "documented in openapi.yaml but not registered on the mux")
			})
		}
	}
}

func TestSpecIsServed(t *testing.T) {
	mux := newMux()

	t.Run("serves the spec", func(t *testing.T) {
		w := httptest.NewRecorder()
		mux.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/openapi.yaml", nil))

		require.Equal(t, http.StatusOK, w.Code)
		assert.Equal(t, "application/yaml", w.Header().Get("Content-Type"))
		assert.Contains(t, w.Body.String(), "openapi: 3.0.3")
	})

	t.Run("serves the ui pointing at the spec", func(t *testing.T) {
		w := httptest.NewRecorder()
		mux.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/docs", nil))

		require.Equal(t, http.StatusOK, w.Code)
		assert.Contains(t, w.Header().Get("Content-Type"), "text/html")
		assert.Contains(t, w.Body.String(), `url: "/openapi.yaml"`)
	})
}

type specDocument struct {
	OpenAPI string `yaml:"openapi"`
	Info    struct {
		Title   string `yaml:"title"`
		Version string `yaml:"version"`
	} `yaml:"info"`
	Paths map[string]map[string]yaml.Node `yaml:"paths"`
}

func parseSpec(t *testing.T) specDocument {
	t.Helper()

	var spec specDocument
	require.NoError(t, yaml.Unmarshal(api.Spec, &spec))
	return spec
}

func newMux() *http.ServeMux {
	mux := http.NewServeMux()
	handlers.NewHeartHandler().RegisterRoutes(mux)
	handlers.NewDocsHandler().RegisterRoutes(mux)
	handlers.NewBookHandler(nil, nil).RegisterRoutes(mux)
	return mux
}

func isHTTPMethod(key string) bool {
	switch strings.ToUpper(key) {
	case http.MethodGet, http.MethodPost, http.MethodPut, http.MethodPatch, http.MethodDelete, http.MethodHead, http.MethodOptions:
		return true
	}
	return false
}

// substitutePathParams turns "/books/{id}" into a concrete path so the mux can
// match it. The value is never used: only the matched pattern is inspected.
func substitutePathParams(path string) string {
	for {
		open := strings.Index(path, "{")
		if open < 0 {
			return path
		}
		close := strings.Index(path[open:], "}")
		if close < 0 {
			return path
		}
		path = path[:open] + "placeholder" + path[open+close+1:]
	}
}
