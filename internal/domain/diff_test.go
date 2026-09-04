package domain_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/MaxCaribe/library-go/internal/domain"
)

func TestDiffDetectsNoChange(t *testing.T) {
	book := hobbit()

	assert.Empty(t, domain.Diff(book, book), "an identical update must record nothing")
}

func TestDiffIgnoresAuthorOrder(t *testing.T) {
	before := hobbit()
	before.Authors = []domain.Author{{Name: "J.R.R. Tolkien"}, {Name: "Christopher Tolkien"}}

	after := hobbit()
	after.Authors = []domain.Author{{Name: "Christopher Tolkien"}, {Name: "J.R.R. Tolkien"}}

	assert.Empty(t, domain.Diff(before, after), "authors are a set: reordering is not a change")
}

func TestDiffScalars(t *testing.T) {
	tests := map[string]struct {
		mutate func(*domain.Book)
		field  domain.ChangeField
		old    string
		new    string
	}{
		"title": {func(b *domain.Book) { b.Title = "The Hobbit, revised" },
			domain.FieldTitle, "The Hobbit", "The Hobbit, revised"},
		"description": {func(b *domain.Book) { b.Description = "A new blurb." },
			domain.FieldDescription, "There and back again.", "A new blurb."},
		"publication date": {func(b *domain.Book) { b.PublishedOn = time.Date(1937, 9, 25, 0, 0, 0, 0, time.UTC) },
			domain.FieldPublishedOn, "1937-09-21", "1937-09-25"},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			after := hobbit()
			tc.mutate(&after)

			changes := domain.Diff(hobbit(), after)
			require.Len(t, changes, 1)
			assert.Equal(t, tc.field, changes[0].Field)
			assert.Equal(t, domain.KindSet, changes[0].Kind)
			assert.Equal(t, tc.old, *changes[0].OldValue)
			assert.Equal(t, tc.new, *changes[0].NewValue)
		})
	}
}

func TestDiffAuthors(t *testing.T) {
	t.Run("added", func(t *testing.T) {
		after := hobbit()
		after.Authors = append(after.Authors, domain.Author{Name: "Christopher Tolkien"})

		changes := domain.Diff(hobbit(), after)
		require.Len(t, changes, 1)
		assert.Equal(t, domain.KindAdded, changes[0].Kind)
		assert.Equal(t, "Christopher Tolkien", *changes[0].NewValue)
		assert.Nil(t, changes[0].OldValue)
	})

	t.Run("removed", func(t *testing.T) {
		before := hobbit()
		before.Authors = append(before.Authors, domain.Author{Name: "Christopher Tolkien"})

		changes := domain.Diff(before, hobbit())
		require.Len(t, changes, 1)
		assert.Equal(t, domain.KindRemoved, changes[0].Kind)
		assert.Equal(t, "Christopher Tolkien", *changes[0].OldValue)
		assert.Nil(t, changes[0].NewValue)
	})

	t.Run("a substitution reads as removed then added", func(t *testing.T) {
		after := hobbit()
		after.Authors = []domain.Author{{Name: "J.R.R. Tolkien (ed.)"}}

		changes := domain.Diff(hobbit(), after)
		require.Len(t, changes, 2)
		assert.Equal(t, domain.KindRemoved, changes[0].Kind)
		assert.Equal(t, domain.KindAdded, changes[1].Kind)
	})
}

func TestDiffEmitsAFixedFieldOrder(t *testing.T) {
	after := domain.Book{
		ID:          hobbit().ID,
		Title:       "New title",
		Description: "New description",
		PublishedOn: time.Date(1950, 1, 1, 0, 0, 0, 0, time.UTC),
		Authors:     []domain.Author{{Name: "Someone Else"}},
	}

	var got []string
	for _, change := range domain.Diff(hobbit(), after) {
		got = append(got, string(change.Field)+"/"+string(change.Kind))
	}

	assert.Equal(t, []string{
		"title/set", "description/set", "published_on/set",
		"authors/removed", "authors/added",
	}, got, "order is fixed so ids encode it and rendering reads sensibly")
}

func TestDiffOfACreationRecordsNothing(t *testing.T) {
	assert.Empty(t, domain.Diff(domain.Book{}, hobbit()),
		"every row describes a field changing; books.created_at records that a book appeared")
}

func TestDiffSortsAuthorsForDeterminism(t *testing.T) {
	after := hobbit()
	after.Authors = append(after.Authors,
		domain.Author{Name: "Zoe"}, domain.Author{Name: "Adam"}, domain.Author{Name: "Mia"})

	var added []string
	for _, change := range domain.Diff(hobbit(), after) {
		added = append(added, *change.NewValue)
	}

	assert.Equal(t, []string{"Adam", "Mia", "Zoe"}, added)
}

func TestDiffPopulatesOnlyFieldLevelFacts(t *testing.T) {
	after := hobbit()
	after.Title = "The Hobbit, revised"

	changes := domain.Diff(hobbit(), after)

	require.NotEmpty(t, changes)
	assert.Empty(t, changes[0].BookID, "identity is stamped by whoever writes the change")
	assert.Empty(t, changes[0].ChangeSetID)
	assert.True(t, changes[0].OccurredAt.IsZero())
}

func hobbit() domain.Book {
	return domain.Book{
		ID:          "01a06d75-1454-72da-a78c-5962e228ee0f",
		Title:       "The Hobbit",
		Description: "There and back again.",
		PublishedOn: time.Date(1937, 9, 21, 0, 0, 0, 0, time.UTC),
		Authors:     []domain.Author{{Name: "J.R.R. Tolkien"}},
	}
}
