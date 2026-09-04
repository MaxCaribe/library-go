package dto_test

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/MaxCaribe/library-go/internal/domain"
	"github.com/MaxCaribe/library-go/internal/infrastructure/httpserver/dto"
)

// The two sentences the brief gives as examples, rendered from stored rows.
func TestRenderMatchesTheExamplesFromTheBrief(t *testing.T) {
	titleFix := domain.Change{
		Field: domain.FieldTitle, Kind: domain.KindSet,
		OldValue: ptr("The Hobbitt"), NewValue: ptr("The Hobbit"),
	}
	authorAdded := domain.Change{
		Field: domain.FieldAuthors, Kind: domain.KindAdded, NewValue: ptr("J.R.R. Tolkien"),
	}

	assert.Equal(t, `Title changed from "The Hobbitt" to "The Hobbit"`, dto.Render(titleFix))
	assert.Equal(t, `Author "J.R.R. Tolkien" was added`, dto.Render(authorAdded))
}

func TestRender(t *testing.T) {
	tests := map[string]struct {
		change   domain.Change
		expected string
	}{
		"title set": {
			domain.Change{Field: domain.FieldTitle, Kind: domain.KindSet, NewValue: ptr("The Hobbit")},
			`Title set to "The Hobbit"`,
		},
		"description changed": {
			domain.Change{Field: domain.FieldDescription, Kind: domain.KindSet, OldValue: ptr("Old"), NewValue: ptr("New")},
			`Description changed from "Old" to "New"`,
		},
		"date set": {
			domain.Change{Field: domain.FieldPublishedOn, Kind: domain.KindSet, NewValue: ptr("1937-09-21")},
			"Publication date set to 1937-09-21",
		},
		"date changed": {
			domain.Change{Field: domain.FieldPublishedOn, Kind: domain.KindSet, OldValue: ptr("1937-09-21"), NewValue: ptr("1937-09-25")},
			"Publication date changed from 1937-09-21 to 1937-09-25",
		},
		"author removed": {
			domain.Change{Field: domain.FieldAuthors, Kind: domain.KindRemoved, OldValue: ptr("Christopher Tolkien")},
			`Author "Christopher Tolkien" was removed`,
		},
		"cleared to empty": {
			domain.Change{Field: domain.FieldDescription, Kind: domain.KindSet, OldValue: ptr("Old"), NewValue: ptr("")},
			`Description changed from "Old" to ""`,
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			assert.Equal(t, tc.expected, dto.Render(tc.change))
		})
	}
}

func TestRenderIsTotal(t *testing.T) {
	tests := map[string]domain.Change{
		"unknown field":          {Field: "isbn", Kind: domain.KindSet, NewValue: ptr("978-0")},
		"unknown kind":           {Field: domain.FieldTitle, Kind: "reordered"},
		"unknown field and kind": {Field: "series_position", Kind: "recalculated"},
		"entirely zero":          {},
		"missing values":         {Field: domain.FieldTitle, Kind: domain.KindSet},
	}

	for name, change := range tests {
		t.Run(name, func(t *testing.T) {
			rendered := dto.Render(change)

			assert.NotEmpty(t, rendered, "a row written by another binary must still read")
			assert.NotContains(t, rendered, "%!", "no formatting verbs should leak")
		})
	}

	assert.Equal(t, `Isbn set to "978-0"`, dto.Render(domain.Change{Field: "isbn", Kind: domain.KindSet, NewValue: ptr("978-0")}))
	assert.Equal(t, "Title was reordered", dto.Render(domain.Change{Field: domain.FieldTitle, Kind: "reordered"}))
	assert.Equal(t, "Series position was recalculated", dto.Render(domain.Change{Field: "series_position", Kind: "recalculated"}))
}

func TestRenderTruncatesLongValues(t *testing.T) {
	long := strings.Repeat("a", 500)
	rendered := dto.Render(domain.Change{Field: domain.FieldDescription, Kind: domain.KindSet, NewValue: ptr(long)})

	assert.Less(t, len(rendered), 100, "a long description must not render as a wall of text")
	assert.Contains(t, rendered, "…")
}

func TestRenderKeepsNonASCIIReadable(t *testing.T) {
	rendered := dto.Render(domain.Change{Field: domain.FieldAuthors, Kind: domain.KindAdded, NewValue: ptr("Аркадий Стругацкий")})

	assert.Equal(t, `Author "Аркадий Стругацкий" was added`, rendered, "strconv.Quote would escape this into \\u sequences")
}

func TestRenderNeutralisesEmbeddedQuotes(t *testing.T) {
	rendered := dto.Render(domain.Change{Field: domain.FieldTitle, Kind: domain.KindSet, NewValue: ptr(`The "Hobbit"`)})

	assert.Equal(t, `Title set to "The 'Hobbit'"`, rendered)
}

func ptr(s string) *string {
	return &s
}
