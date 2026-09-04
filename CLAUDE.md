# CLAUDE.md

Go HTTP service managing books and a history of the changes made to them.

## Architecture

- Layering: `domain` <- `application` <- `infrastructure`, wired by hand in `internal/server/`. No wire/dig.
- `domain` imports nothing from the project. It holds the entities and the pure logic over them — business invariants, diffing, rendering. It does **not** parse or format-check input; that belongs to the DTO layer.
- `application` holds use-case services and **declares the repository interfaces it needs**; `infrastructure` implements them.
- Handlers declare the service interface they depend on, in the handlers package. The concrete service satisfies it.
- Routing is stdlib `http.ServeMux` with method patterns (`"GET /books/{id}"`). No third-party router.
- Persistence sits behind repository interfaces declared in `application`. The backing store has not been chosen yet.

## Where things go

| What | Where |
|---|---|
| Entities and pure domain logic | `internal/domain/` |
| Services + repository interfaces | `internal/application/` |
| Repository implementations | `internal/infrastructure/<store>/` |
| Handlers | `internal/infrastructure/httpserver/handlers/` |
| Request/response shapes, input validation, pagination | `internal/infrastructure/httpserver/dto/` |
| JSON writers, domain-error to status mapping | `internal/infrastructure/httpserver/response/` |
| Strict JSON body decoding | `internal/infrastructure/httpserver/request/` |
| Middleware | `internal/infrastructure/httpserver/middleware/` |
| Dependency wiring | `internal/server/` |
| OpenAPI spec | `api/openapi.yaml` |
| Tests | `test/<layer>/` |

## Conventions

**Comments are off by default.** Write one only for a non-obvious *why* — a subtle invariant, a workaround, a bound not visible from the types. No doc comments on functions: the name and signature are the documentation. No comments on struct fields. When editing, delete comments that no longer earn their keep.

**Validation is split by what it is about.**

- *Input* validation lives in the DTO layer, next to the JSON tags: required fields, sizes, date formats, duplicate entries. It is about the request. `BookRequest.Parse()` trims, validates and converts in one pass, returning the built domain value plus a `map[string]string` of field errors keyed by wire name (`title`, `authors[1]`).
- *Business rules* live on the domain type — invariants that must hold however the value was constructed, whoever built it.

Some checks can only happen in the DTO: a `published_on` that doesn't parse never reaches the domain as a `time.Time`. Domain types carry no validation today because no business rule has needed one yet, not because they shouldn't.

**Error handling.** Services return `domain` sentinels (`ErrNotFound`, `ErrAlreadyExists`, `ErrInvalidInput`, `ErrAccessDenied`). `response.DomainError` is the single place mapping them to status codes — every new sentinel belongs in that switch or handlers will report it as a 500. Handlers decode, delegate, and encode; they do not validate and do not invent statuses.

**Receivers.** Pick one kind per type. Prefer value receivers and methods that return a modified copy (`Normalized() T`) over pointer receivers that mutate. Never mix the two on one type.

**Response shapes.** Single resources go in a `{"data": ...}` envelope via `response.WithData`. Collections use `dto.PaginatedResponse[T]` (`data`, `total`, `page`, `page_size`, `total_pages`). Errors are `{"error": "..."}`; validation failures add `fields`. JSON is snake_case via struct tags.

**DTO conversions.** Outward: `ToXResponse(domain.X) XResponse`, plus `ToXResponses` for slices. Inward: a `Parse()` method on the request type returning the domain value and a field-error map. Name the DTO types `XRequest`/`XResponse` so a signature never has the same type name on both sides.

**Pagination.** `dto.ParsePagination(r)` then `dto.ComputePagination(page, pageSize)` -> `(limit, offset)`. Collections use the generic `dto.PaginatedResponse[T]`, which embeds the generated `PaginationMeta` — the shape is reusable, the fields come from the spec.

**Sets.** `map[string]bool`, not `map[string]struct{}` — the byte saved is not worth the comma-ok read.

**Context.** Propagate `context.Context` through handlers, services and storage. Always use the `...Context` database methods; a single bare `db.Query` breaks it.

## Tests

- Integration-style, driving real handlers through a real `http.ServeMux` and asserting status codes and JSON. Prefer them over unit tests with mocks.
- **Layout: test cases first, then helpers, then fakes and mocks last.** A reader is there for the cases, not the plumbing.
- Table-driven subtests with `t.Run`. `require` for preconditions, `assert` for assertions.

## API documentation

`api/openapi.yaml` is hand-written and is the **source of truth for request and response shapes**. It is embedded with `go:embed`; `/openapi.yaml` serves it and `/docs` renders it with Swagger UI.

`oapi-codegen` generates the DTO structs from it in models-only mode — `make generate` (or `go generate ./api`) rewrites `internal/infrastructure/httpserver/dto/openapi_types.gen.go`. Never hand-edit that file; change the spec and regenerate. Handlers stay hand-written.

**Update the spec in the same change as the route or shape it describes**, then regenerate. Two things then check it: the compiler, because handlers use the generated structs, and `test/docs`, which resolves every documented operation against the real mux.

Spec conventions worth knowing:
- `x-go-type: string` on `BookInput.published_on` keeps it a plain string so an unparseable date is a field-level validation error rather than a whole-body decode failure. The response keeps `openapi_types.Date`.
- One `Error` schema with an optional `fields` map covers both wire shapes. A `oneOf` here would generate a large union type nothing uses.

## Commands

```sh
make server   # go run ./cmd/server/
make test     # go test ./...
go vet ./...
```

## Git

Commit messages are plain imperative with no `feat:`/`chore:` prefix, matching `9cf8b34 bootstrap layered HTTP service skeleton`. **Never run git commands** — the user reviews and commits each step themselves.
