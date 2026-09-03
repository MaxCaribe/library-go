# library-go

A starting skeleton for a Go HTTP service: configuration, structured logging, middleware, graceful shutdown, and a
layered layout with the wiring points already in place. It carries no domain code — the first resource you add is
your own.

## Quick start

```sh
cp example.env .env
make server
curl -s localhost:8080/heartbeat
```

`make test` runs the suite, `make build` produces a static binary in `bin/`.

## Layout

```
cmd/server/                              entrypoint: config -> App -> Start
internal/
  config/                                flags + env vars, one Config struct
  domain/                                entities, sentinel errors, validation — imports nothing from the project
  infrastructure/
    httpserver/
      server/                            http.Server lifecycle (timeouts, graceful stop)
      handlers/                          decode -> call service -> encode; each owns its route registration
      dto/                               request/response shapes, domain conversions, pagination
      response/                          JSON writers and the domain-error -> HTTP status map
      middleware/                        recovery, body limit, request ID, logging
  server/                                dependency wiring: app.go, infrastructure.go, services.go, http.go
pkg/logging/                             slog root logger + request-ID context helpers
test/handlers/                           handler tests driven through a real ServeMux
```

Two layers have no files yet, because they only exist once there is a domain to serve:

- `internal/application/` — use-case services, plus the repository interfaces they depend on
- `internal/infrastructure/<store>/repositories/` — implementations of those interfaces

## The dependency rule

```
domain  <-  application  <-  infrastructure
                    ^             |
                    |             |
                    +-- server ---+   (wires concrete adapters into services)
```

`domain` imports nothing from the project. `application` depends on `domain` and **declares the interfaces it
needs**; `infrastructure` implements them. A service therefore has no idea whether its data lives in memory, in
Postgres, or behind another API — which is what makes it testable without any of them running.

`internal/server/` is the only place that knows about concrete types. `newInfra` constructs adapters, `newServices`
injects them into services, `newHTTPHandler` mounts handlers and middleware. Changing a datastore is a change to
`newInfra` and nothing else.

## Adding a resource

1. `internal/domain/<thing>.go` — the entity and its `Validate()`; new sentinel errors go in `errors.go`
2. `internal/application/interfaces.go` — the repository interface the service needs, written from the service's
   point of view, not the database's
3. `internal/application/<thing>_service.go` — the use cases; return `domain` sentinels, never HTTP concerns
4. `internal/infrastructure/<store>/repositories/<thing>_repository.go` — implement that interface
5. `internal/infrastructure/httpserver/dto/<thing>_dto.go` — request/response shapes and `ToXResponse` converters
6. `internal/infrastructure/httpserver/handlers/<thing>_handler.go` — handlers plus `RegisterRoutes(mux)`
7. `internal/server/{infrastructure,services,http}.go` — one line each to construct and wire it
8. `test/handlers/<thing>_handler_test.go` — drive the routes end to end

Map every new sentinel error to a status code in `response.DomainError`, or handlers will report it as a 500.

## Conventions

- Handlers decode, delegate, and encode. Validation belongs on the domain type; error-to-status mapping belongs in
  `response.DomainError`. A handler that does its own validation is a handler that will disagree with the next one
- Single resources are returned as `{"data": {...}}` via `response.WithData`; lists as `dto.PaginatedResponse`
  (`{"data": [...], "total": n, "page": n, "page_size": n, "total_pages": n}`); errors as `{"error": "..."}`
- Pagination: `dto.ParsePagination(r)` -> `dto.ComputePagination(page, pageSize)` -> `(limit, offset)`
- JSON fields are snake_case via struct tags
- Routing is stdlib `http.ServeMux` with method patterns (`"GET /things/{id}"`) — no third-party router
- `*slog.Logger` is passed as a dependency; nothing uses the package-level default
- Tests drive real handlers through a real `ServeMux` and assert on status codes and JSON, rather than mocking the
  HTTP layer

## Middleware

Applied outermost to innermost in `internal/server/http.go`:

1. **Recovery** — a panic in any layer becomes a logged 500 instead of a dropped connection
2. **Body limit** — `http.MaxBytesReader`, so oversized payloads die before decoding
3. **Request ID** — reuses an inbound `X-Request-ID` or mints one, echoes it back, puts it in the context
4. **Logging** — one structured line per request: method, path, status, duration, request ID

## Configuration

| Env | Flag | Default | Purpose |
| --- | --- | --- | --- |
| `DEBUG` | `--debug` | `true` | Text logs when true, JSON when false |
| `HTTP_PORT` | `--http.port` | `8080` | Listen port |

`.env` is loaded at startup if present; `--help` lists everything.

Only values that differ between deployments live here. Server timeouts, the request body cap, and the shutdown grace
period are constants beside the code they govern — in `httpserver/server`, `internal/server/http.go`, and
`internal/server/app.go` respectively.
