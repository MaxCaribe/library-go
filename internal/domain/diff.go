package domain

import (
	"slices"

	"github.com/MaxCaribe/library-go/pkg/utils"
)

// Order is fixed, and the repository must insert in it: rows in a change set
// share an occurred_at, so only their ids preserve order.
func Diff(before, after Book) []Change {
	// A creation changes nothing; books.created_at records that it happened.
	if before.ID == "" {
		return nil
	}

	var changes []Change
	changes = appendScalar(changes, FieldTitle, before.Title, after.Title)
	changes = appendScalar(changes, FieldDescription, before.Description, after.Description)
	changes = appendScalar(changes, FieldPublishedOn, publicationDate(before), publicationDate(after))
	return append(changes, authorChanges(before.Authors, after.Authors)...)
}

func appendScalar(changes []Change, field ChangeField, before, after string) []Change {
	if before == after {
		return changes
	}
	return append(changes, Change{Field: field, Kind: KindSet, OldValue: utils.Reference(before), NewValue: utils.Reference(after)})
}

// Removals come first so a substitution reads as removed-then-added.
func authorChanges(before, after []Author) []Change {
	removed := missing(before, after)
	added := missing(after, before)

	changes := make([]Change, 0, len(removed)+len(added))
	for _, name := range removed {
		changes = append(changes, Change{Field: FieldAuthors, Kind: KindRemoved, OldValue: utils.Reference(name)})
	}
	for _, name := range added {
		changes = append(changes, Change{Field: FieldAuthors, Kind: KindAdded, NewValue: utils.Reference(name)})
	}
	return changes
}

// Sorted, so the result does not depend on map iteration order.
func missing(from, in []Author) []string {
	present := make(map[string]bool, len(in))
	for _, author := range in {
		present[author.Name] = true
	}

	var names []string
	for _, author := range from {
		if !present[author.Name] {
			names = append(names, author.Name)
		}
	}
	slices.Sort(names)
	return names
}

func publicationDate(book Book) string {
	if book.PublishedOn.IsZero() {
		return ""
	}
	return book.PublishedOn.Format(DateLayout)
}
