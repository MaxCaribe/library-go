# CLAUDE.md

Go HTTP service managing books and a history of the changes made to them.

## Architecture

- Layering: `domain` <- `application` <- `infrastructure`, wired by hand in `internal/server/`. No wire/dig.
- `domain` imports nothing from the project. It holds the entities and the pure logic over them — business invariants, diffing, rendering. It does **not** parse or format-check input; that belongs to the DTO layer.
- `application` holds use-case services and **declares the repository interfaces it needs**; `infrastructure` implements them.
- Handlers declare the service interface they depend on, in the handlers package. The concrete service satisfies it.
- Routing is stdlib `http.ServeMux` with method patterns (`"GET /books/{id}"`). No third-party router.
- Persistence is Postgres, reached through `database/sql` over the `jackc/pgx/v5/stdlib` driver. Repositories sit behind interfaces declared in `application`.

## Where things go

| What | Where |
|---|---|
| Entities and pure domain logic | `internal/domain/` |
| Services + repository interfaces | `internal/application/` |
| Pool, repositories | `internal/infrastructure/postgres/` |
| Migration runner | `internal/infrastructure/dbmigrate/` |
| Migration SQL | `migrations/` |
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

**Repositories return values, not pointers** — `(domain.Book, error)`, `[]domain.Book`. Applied to every repository without exception: mixing the two across a codebase is a worse cost than either choice, because each call site becomes a question. A miss is always `domain.ErrNotFound`, never a nil value with a nil error.

**Receivers.** Pick one kind per type and never mix them. Prefer value receivers that return a modified copy over pointer receivers that mutate; use pointer receivers for handles to shared state (`*BookRepository`, `*BookService`).

**Response shapes.** Single resources go in a `{"data": ...}` envelope via `response.WithData`. Collections use `dto.PaginatedResponse[T]` (`data`, `total`, `page`, `page_size`, `total_pages`). Errors are `{"error": "..."}`; validation failures add `fields`. JSON is snake_case via struct tags.

**DTO conversions.** Outward: `ToXResponse(domain.X) XResponse`, plus `ToXResponses` for slices. Inward: a `Parse()` method on the request type returning the domain value and a field-error map. Name the DTO types `XRequest`/`XResponse` so a signature never has the same type name on both sides.

**Pagination.** `dto.ParsePagination(r)` then `dto.ComputePagination(page, pageSize)` -> `(limit, offset)`. Collections use the generic `dto.PaginatedResponse[T]`, which embeds the generated `PaginationMeta` — the shape is reusable, the fields come from the spec.

**Sets.** `map[string]bool`, not `map[string]struct{}` — the byte saved is not worth the comma-ok read.

**Context.** Propagate `context.Context` through handlers, services and storage. Always use the `...Context` database methods; a single bare `db.Query` breaks it.

## Tests

- **Unit tests live beside the code they test**, in the same directory as `package <name>_test`. The external test package keeps them honest about the exported surface, and coverage is attributed to the package under test — a test in a separate directory leaves its target reporting 0%.
- **`test/` is only for tests that need something outside the process.** Today that is `test/integration` (a real database) plus `test/support` (its harness). If a test needs no external resource, it does not belong there.
- Integration tests get their database from `TEST_DATABASE_URL` if set, otherwise a Postgres testcontainer (one per package, truncated between tests). With neither Docker nor the variable they **skip** with an instruction rather than failing, so `go test ./...` works anywhere.
- No fakes or mocks for the service or repository — integration tests drive the real stack.
- **Layout: test cases first, then helpers, then fakes and mocks last.** A reader is there for the cases, not the plumbing.
- Table-driven subtests with `t.Run`. `require` for preconditions, `assert` for assertions.

## API documentation

`api/openapi.yaml` is hand-written and is the **source of truth for request and response shapes**. It is embedded with `go:embed`; `/openapi.yaml` serves it and `/docs` renders it with Swagger UI.

`oapi-codegen` generates the DTO structs from it in models-only mode — `make generate` (or `go generate ./api`) rewrites `internal/infrastructure/httpserver/dto/openapi_types.gen.go`. Never hand-edit that file; change the spec and regenerate. Handlers stay hand-written.

**Update the spec in the same change as the route or shape it describes**, then regenerate. Two things then check it: the compiler, because handlers use the generated structs, and `test/docs`, which resolves every documented operation against the real mux.

Spec conventions worth knowing:
- `x-go-type: string` on `BookInput.published_on` keeps it a plain string so an unparseable date is a field-level validation error rather than a whole-body decode failure. The response keeps `openapi_types.Date`.
- One `Error` schema with an optional `fields` map covers both wire shapes. A `oneOf` here would generate a large union type nothing uses.

## Migrations

goose, with `migrations/*.sql` embedded into the binary. One file per change, named `NNNN_snake_case.sql`, each with `-- +goose Up` and `-- +goose Down` sections; the Down section is required so a migration can be rolled back. Never edit an applied migration — add a new one.

The provider runs with `WithAllowOutofOrder(true)`: goose records every applied version, so a branch merged after a higher version has been applied still runs rather than being silently skipped.

```sh
make db-up            # docker compose up -d postgres
make migrate-up       # apply pending migrations
make migrate-status   # per-migration state
make migrate-down     # roll back one
make create-migration name=add_book_isbn
```

## Commands

```sh
make server   # go run ./cmd/server/
make test     # go test ./...
make cover    # coverage across internal/, including from test/integration
make generate # regenerate DTOs from the OpenAPI spec
go vet ./...
```

Measure coverage with `make cover`, never plain `go test -cover`: `test/integration` exercises `application` and `repositories` from outside those directories, so without `-coverpkg=./internal/...` they report as untested.

## Git

Commit messages are plain imperative with no `feat:`/`chore:` prefix, matching `9cf8b34 bootstrap layered HTTP service skeleton`. **Never run git commands** — the user reviews and commits each step themselves.
