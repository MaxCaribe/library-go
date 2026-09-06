package integration

import (
	"context"
	"net/http"
	"sync"
	"testing"

	"github.com/jackc/pgx/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/MaxCaribe/library-go/internal/domain"
	"github.com/MaxCaribe/library-go/internal/infrastructure/httpserver/dto"
	"github.com/MaxCaribe/library-go/test/support"
)

func TestCreateRecordsNoHistory(t *testing.T) {
	api := support.NewBookAPI(t)
	book := decodeData(t, send(t, api.Mux, http.MethodPost, "/books", hobbitBody))

	assert.Empty(t, historyOf(t, api, book.ID),
		"a book that has never changed has no history; created_at records when it appeared")
}

func TestUpdateRecordsOnlyWhatChanged(t *testing.T) {
	api := support.NewBookAPI(t)
	book := decodeData(t, send(t, api.Mux, http.MethodPost, "/books", hobbitBody))

	send(t, api.Mux, http.MethodPut, "/books/"+book.ID, `{"title":"The Hobbit, revised",
		"description":"There and back again.","published_on":"1937-09-21","authors":["J.R.R. Tolkien"]}`)

	changes := historyOf(t, api, book.ID)
	require.Len(t, changes, 2, "only the title and the dropped author changed")

	assert.Equal(t, `Title changed from "The Hobbit" to "The Hobbit, revised"`, dto.ToChangeResponse(changes[0]).Description)
	assert.Equal(t, `Author "Christopher Tolkien" was removed`, dto.ToChangeResponse(changes[1]).Description)

	assert.Equal(t, changes[0].ChangeSetID, changes[1].ChangeSetID, "one request is one change set")
	assert.Equal(t, changes[0].OccurredAt, changes[1].OccurredAt)
}

func TestIdenticalUpdateRecordsNothing(t *testing.T) {
	api := support.NewBookAPI(t)
	book := decodeData(t, send(t, api.Mux, http.MethodPost, "/books", hobbitBody))
	before := historyOf(t, api, book.ID)

	require.Equal(t, http.StatusOK, send(t, api.Mux, http.MethodPut, "/books/"+book.ID, hobbitBody).Code)

	after := historyOf(t, api, book.ID)
	assert.Len(t, after, len(before), "a no-op update must not write history")
	assert.Equal(t, book.UpdatedAt, decodeData(t, send(t, api.Mux, http.MethodGet, "/books/"+book.ID, "")).UpdatedAt,
		"and must not move updated_at")
}

func TestFailedUpdateWritesNoHistory(t *testing.T) {
	api := support.NewBookAPI(t)
	book := decodeData(t, send(t, api.Mux, http.MethodPost, "/books", hobbitBody))
	before := len(historyOf(t, api, book.ID))

	// Rejected by validation, so it never reaches the transaction.
	require.Equal(t, http.StatusBadRequest,
		send(t, api.Mux, http.MethodPut, "/books/"+book.ID, `{"title":"","published_on":"1937-09-21","authors":["A"]}`).Code)

	assert.Len(t, historyOf(t, api, book.ID), before)
}

// TestConcurrentUpdatesRecordAConsistentHistory is the evidence for the
// transactional read-diff-write. Without FOR UPDATE, two updaters read the same
// "before" and each records a change from a state that no longer exists.
func TestConcurrentUpdatesRecordAConsistentHistory(t *testing.T) {
	api := support.NewBookAPI(t)
	book := decodeData(t, send(t, api.Mux, http.MethodPost, "/books", hobbitBody))
	before := len(historyOf(t, api, book.ID))

	const writers = 8
	var wg sync.WaitGroup
	for i := range writers {
		wg.Add(1)
		go func() {
			defer wg.Done()
			body := `{"title":"Title ` + string(rune('A'+i)) + `","description":"There and back again.",
				"published_on":"1937-09-21","authors":["J.R.R. Tolkien","Christopher Tolkien"]}`
			send(t, api.Mux, http.MethodPut, "/books/"+book.ID, body)
		}()
	}
	wg.Wait()

	changes := historyOf(t, api, book.ID)[before:]
	require.Len(t, changes, writers, "each update records exactly one title change")

	// Replaying the title changes in order must reproduce the stored title, and
	// each change must start where the previous one ended.
	final := decodeData(t, send(t, api.Mux, http.MethodGet, "/books/"+book.ID, ""))
	title := "The Hobbit"
	for _, change := range changes {
		require.Equal(t, domain.FieldTitle, change.Field)
		assert.Equal(t, title, *change.OldValue, "a change must start from the previous state, not a stale read")
		title = *change.NewValue
	}
	assert.Equal(t, final.Title, title, "replaying history must reproduce the current book")
}

func historyOf(t *testing.T, api support.API, bookID string) []domain.Change {
	t.Helper()

	rows, err := api.Pool.Query(context.Background(), `
		SELECT id, book_id::text, change_set_id::text, occurred_at, field, kind, old_value, new_value
		FROM changes WHERE book_id = $1 ORDER BY occurred_at, id`, bookID)
	require.NoError(t, err)

	changes, err := pgx.CollectRows(rows, func(row pgx.CollectableRow) (domain.Change, error) {
		var c domain.Change
		err := row.Scan(&c.ID, &c.BookID, &c.ChangeSetID, &c.OccurredAt, &c.Field, &c.Kind, &c.OldValue, &c.NewValue)
		return c, err
	})
	require.NoError(t, err)
	return changes
}
