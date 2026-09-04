package handlers

import (
	"net/http"

	"github.com/MaxCaribe/library-go/api"
)

// docsPage renders the embedded spec with Swagger UI. The assets come from a
// CDN, so /docs needs network access on first load; the spec itself is served
// from this binary. Serving both from one origin is what lets "Try it out"
// call the API without any CORS configuration.
const docsPage = `<!doctype html>
<html lang="en">
<head>
  <meta charset="utf-8">
  <meta name="viewport" content="width=device-width, initial-scale=1">
  <title>Library API</title>
  <link rel="stylesheet" href="https://cdn.jsdelivr.net/npm/swagger-ui-dist@5/swagger-ui.css">
</head>
<body>
  <div id="swagger-ui"></div>
  <script src="https://cdn.jsdelivr.net/npm/swagger-ui-dist@5/swagger-ui-bundle.js" crossorigin></script>
  <script>
    window.onload = () => SwaggerUIBundle({ url: "/openapi.yaml", dom_id: "#swagger-ui" });
  </script>
</body>
</html>
`

type DocsHandler struct{}

func NewDocsHandler() *DocsHandler {
	return &DocsHandler{}
}

func (h *DocsHandler) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("GET /openapi.yaml", h.Spec)
	mux.HandleFunc("GET /docs", h.UI)
}

func (h *DocsHandler) Spec(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "application/yaml")
	w.Write(api.Spec)
}

func (h *DocsHandler) UI(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Write([]byte(docsPage))
}
