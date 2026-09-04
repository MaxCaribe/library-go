package dto

import (
	"fmt"
	"strings"
	"unicode/utf8"

	"github.com/MaxCaribe/library-go/internal/domain"
	"github.com/MaxCaribe/library-go/pkg/utils"
)

const maxRenderedValueLen = 60

var fieldLabels = map[domain.ChangeField]string{
	domain.FieldTitle:       "Title",
	domain.FieldDescription: "Description",
	domain.FieldPublishedOn: "Publication date",
	domain.FieldAuthors:     "Author",
}

// Must stay total: a row written by an older or newer binary still has to
// render, hence the fallback. Never returns an empty string.
func Render(change domain.Change) string {
	label := fieldLabel(change.Field)

	switch {
	case change.Kind == domain.KindAdded:
		return fmt.Sprintf("%s %s was added", label, quote(utils.Dereference(change.NewValue)))
	case change.Kind == domain.KindRemoved:
		return fmt.Sprintf("%s %s was removed", label, quote(utils.Dereference(change.OldValue)))
	case change.Kind == domain.KindSet:
		return renderSet(label, change)
	}
	return fmt.Sprintf("%s was %s", label, change.Kind)
}

func renderSet(label string, change domain.Change) string {
	// Dates read better unquoted.
	format := quote
	if change.Field == domain.FieldPublishedOn {
		format = func(s string) string { return s }
	}

	if change.OldValue == nil {
		return fmt.Sprintf("%s set to %s", label, format(utils.Dereference(change.NewValue)))
	}
	return fmt.Sprintf("%s changed from %s to %s", label, format(utils.Dereference(change.OldValue)), format(utils.Dereference(change.NewValue)))
}

func fieldLabel(field domain.ChangeField) string {
	if label, ok := fieldLabels[field]; ok {
		return label
	}
	words := strings.ReplaceAll(string(field), "_", " ")
	if words == "" {
		return "Field"
	}
	return strings.ToUpper(words[:1]) + words[1:]
}

// Capped so a long description does not render as a wall of text. Not
// strconv.Quote: it escapes non-ASCII, mangling author names.
func quote(s string) string {
	s = strings.ReplaceAll(s, `"`, `'`)
	if utf8.RuneCountInString(s) > maxRenderedValueLen {
		s = string([]rune(s)[:maxRenderedValueLen]) + "…"
	}
	return `"` + s + `"`
}
