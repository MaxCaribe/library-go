# library-go

An HTTP service that manages books and records a history of the changes made to them.

## Prerequisites

- **Go 1.26 or newer** — required by `go.mod`.
- **PostgreSQL** — your own instance, or Docker Compose to run the bundled one.

## Running it

With Docker:

```sh
cp example.env .env
make db-up        # postgres via docker compose
make migrate-up
make server
```

Or against a Postgres you already have — edit `DATABASE_URL` in `.env`, then:

```sh
make migrate-up
make server
```

Either way the service refuses to start without a reachable database, and says so: it retries for about fifteen seconds before reporting `postgres: unreachable`.

Then open <http://localhost:8080/docs>. Swagger UI is generated from the served spec, and because the spec and the API share an origin, **Try it out** calls the real service with no CORS configuration — so every endpoint can be exercised from the browser.

Create a book, change its title and add an author, then fetch its history. The entries read:

```
Title changed from "The Hobbitt" to "The Hobbit"
Author "Christopher Tolkien" was added
```

## API

| Method | Path | Notes |
| --- | --- | --- |
| `POST` | `/books` | 201 with the created book and a `Location` header |
| `GET` | `/books` | paginated |
| `GET` | `/books/{id}` | |
| `PUT` | `/books/{id}` | full replacement; every difference is recorded |
| `GET` | `/books/{id}/history` | paginated, filterable, sortable |
| `GET` | `/heartbeat`, `/docs`, `/openapi.yaml` | |

History query parameters: `page`, `page_size`, `field`, `kind`, `from`, `to`, `sort`, `order`. `field` and `kind` accept a comma-separated list or repeated parameters. `from` is inclusive and `to` exclusive, so adjacent windows neither overlap nor drop a row.

Single resources are returned as `{"data": …}`; collections add `total`, `page`, `page_size`, `total_pages` alongside `data`. Errors are `{"error": "…"}`, and validation failures add a `fields` map keyed by the offending input.

**An unrecognised filter value is a 400, not an ignored filter.** Returning unfiltered data to someone who asked for a subset is a worse failure than refusing.

---

## The change history model


Each row in `changes` is **one field-level difference**:

```
id  book_id  change_set_id  occurred_at  field  kind  old_value  new_value
```

`field` and `kind` are separate columns, so `?field=authors` is an index range rather than a `LIKE` over a combined string. Rows written by one request share a `change_set_id` and an `occurred_at`. Values are text: three of the four fields are natively text, and the fourth is a date stored in canonical `YYYY-MM-DD`, which sorts correctly as a string.

**The prose is rendered at read time, not stored.** `Title changed from "The Hobbitt" to "The Hobbit"` is produced from the columns when the response is built, in `dto`. Storing the sentence instead would make history unfilterable — you can only query it with `LIKE` — and would freeze the wording, so fixing a typo could never apply retroactively and translation would be impossible. Rendering also keeps the decision where it belongs: what changed is domain, how we say it is transport.

The cost is that stored rows outlive the code that renders them. Add a field, deploy, then roll back, and the older binary meets rows naming a field it has never heard of — likewise for a field later dropped from the model. So the renderer never fails on an unrecognised `field` or `kind`: it falls back to a sentence built from the raw column name rather than returning nothing, and a test pins that it never produces an empty description. Storing the sentence would avoid this, because each row would carry its own wording.

### What was rejected

**Storing a snapshot per version.** Trivially correct and gives point-in-time restore for free, but "show me every title change" then means fetching and diffing every version of every book. Filtering and ordering — the two things the history endpoint exists to do — are exactly what it's worst at. And we get point-in-time anyway: because each row holds *both* values, you can take the current book and walk changes backwards to reconstruct any earlier state.

**Event sourcing.** Highest fidelity, and history could never disagree with state because it *is* the state. Disproportionate here: it needs projection and rebuild logic, and `?field=title` becomes a query inside JSON payloads.

**A `change_sets` parent table.** It would have held `book_id`, `occurred_at` and an operation flag — the first two are already denormalised onto `changes` so history queries never join, and the third is derivable. That left a table, a foreign key and an extra insert per write, earning one column. The `change_set_id` column gives the same grouping; if set-level attributes ever appear (see *actors* below), it's the seam to promote.

**Recording creation.** A create now records nothing. Recording it per field meant opening a new book's history and seeing six rows restating what the record plainly shows; recording it as a single marker meant a pseudo-field `book` in an enum where everything else names a real column, and a `new_value` that had to arbitrarily pick the title. `books.created_at` already answers when a book appeared, and the original value of any *edited* field survives in that edit's `old_value`.

---

## Concurrency

The diff requires read-current → compute → write. That sequence must be atomic, or two simultaneous updates both read the same "before" and each records a change from a state that no longer existed — leaving a history that cannot explain the book in front of you.

`BookRepository.Update` runs the whole sequence in one transaction and takes the row lock up front:

```sql
SELECT … FROM books WHERE id = $1 FOR UPDATE
```

Without `FOR UPDATE`, Postgres' default `READ COMMITTED` lets both transactions read before either writes. `TestConcurrentUpdatesRecordAConsistentHistory` runs eight concurrent updates and asserts that replaying the resulting changes in order reproduces the stored title — every change starting exactly where the previous one ended.

**`occurred_at` is read inside the lock**, not before it. A writer that took the clock early and acquired the lock late would be stamped ahead of changes it actually followed: the values would still chain correctly, but the ordering would contradict them. Reading it while holding the lock keeps `occurred_at` consistent with the order the updates serialised.

---

## Indexes

```sql
idx_changes_book_time       (book_id, occurred_at, id)
idx_changes_book_field_time (book_id, field, occurred_at, id)
```

Equality column first, then the sort column, then the tiebreaker. Measured against 20,000 rows:

```
Limit
  ->  Index Only Scan Backward using idx_changes_book_time on changes
        Index Cond: (book_id = '…'::uuid)
```

No sort step. Two details make that possible.

**The `id` tiebreaker.** Rows in a change set share `occurred_at`, so `ORDER BY occurred_at` alone leaves their relative order undefined — the same row can appear on page one *and* page two while another vanishes. `id` is a `BIGSERIAL` assigned in the order `Diff` emits changes, so it is both unique and meaningful.

**Uniform direction.** `ORDER BY occurred_at DESC, id DESC` — both terms descending. Postgres scans an index backwards only when every term agrees; a mixed `occurred_at DESC, id ASC` would force a sort and discard the index.

There is no index on `kind`: four distinct values make it a poor leading column, and it costs nothing as a residual filter on rows the composite index has already narrowed.

---

## Other decisions

**Raw SQL, no ORM.** The interesting parts of this service are transaction scope, index design and ordering stability. An ORM would have made books CRUD faster to write and those decisions harder to see or make. `ent`'s mutation hooks would have given field-level diffs almost free — but the easy half is detecting changes, not deciding transaction boundaries.

**No sqlc either.** Its largest saving comes from using its generated structs directly, which would let the database schema define the domain model. Confine them to the repository and map at the boundary, and what's left is ten lines of scanning. Its durable benefit — verifying SQL against the schema at build time — is covered here by integration tests that exercise every query against a real database. Dynamic filtering and ordering are also where generation helps least: `ORDER BY $1` silently sorts nothing, and the usual `CASE` workaround degrades to a sequential scan and sort once Postgres switches a prepared statement to a generic plan.

**Authors are an ordered `text[]` column,** not a child table. An author here is a value object with no identity, attributes or lifecycle. The array preserves order natively, needs no join, and turns update into a single column assignment. It also suits the history model: an audit trail should record *values*, not references — with shared author entities, renaming one would silently change what every book looks like with no change set to explain it. Duplicate detection moves to the application layer as a result. Expanding to a join table later is about a dozen lines of SQL using `unnest(authors) WITH ORDINALITY`.

**The OpenAPI spec owns the wire shapes.** `oapi-codegen` generates the request and response structs from `api/openapi.yaml` in models-only mode, so a renamed field is a compile error rather than a silent lie. Handlers stay hand-written. A test resolves every documented operation against the real mux, and wire-format tests pin the exact JSON, since a changed struct tag breaks clients without breaking the build.

---

## Deliberately left out

- **`DELETE`** — deleting a book would destroy the history that is the point of the service. The schema says so too: `changes.book_id` is `ON DELETE RESTRICT`.
- **`PATCH`** — `PUT` as full replacement keeps diff semantics unambiguous. Adding `PATCH` is a DTO change: pointer fields applied onto the current book inside the same transaction.
- **A cross-book `/history` feed.** One more index for a reader that doesn't exist.
- **Author reordering and rename detection.** Authors are diffed as a set, so reordering persists but records nothing, and a rename is a removal plus an addition. Fuzzy rename pairing is the kind of cleverness that renders a confidently wrong sentence.

**The one property to preserve:** the history is complete only while there is exactly one write path. Everything goes through `BookRepository.Update`, which writes the book and its changes in a single transaction. A bulk importer or a hand-run `UPDATE` would leave a gap that nothing detects.

---

## Tests

```sh
make test              # everything that needs no database
make test-integration  # adds a Postgres container, or set TEST_DATABASE_URL
make cover             # coverage across internal/, including from test/integration
```

Unit tests sit beside the code they cover. `test/integration` drives the real stack — pool, migrations, repository, service, handler — with no fakes or mocks below HTTP. It gets its database from `TEST_DATABASE_URL` if set, otherwise a testcontainer, and skips with an instruction if neither is available, so `go test ./...` works anywhere.

`make cover` rather than `go test -cover`: integration tests exercise `application` and `repositories` from outside those directories, so without `-coverpkg` they report as untested.
