package dto

import (
	"fmt"
	"net/http"
	"slices"
	"strings"
	"time"

	"github.com/MaxCaribe/library-go/internal/application"
	"github.com/MaxCaribe/library-go/internal/domain"
	"github.com/MaxCaribe/library-go/pkg/utils"
)

var (
	changeFields = []domain.ChangeField{domain.FieldTitle, domain.FieldDescription, domain.FieldPublishedOn, domain.FieldAuthors}
	changeKinds  = []domain.ChangeKind{domain.KindSet, domain.KindAdded, domain.KindRemoved}
	sortFields   = []application.ChangeSortField{application.SortByOccurredAt, application.SortByField}
)

// ParseChangeFilter reads the history filters. An unrecognised value is a field
// error, never a silently ignored filter: returning unfiltered data to someone
// who asked for a subset is worse than refusing.
func ParseChangeFilter(r *http.Request, bookID string) (application.ChangeFilter, map[string]string) {
	fields := map[string]string{}
	values := r.URL.Query()

	filter := application.ChangeFilter{
		BookID:     bookID,
		Fields:     parseEnums(values, "field", changeFields, fields),
		Kinds:      parseEnums(values, "kind", changeKinds, fields),
		From:       parseInstant(values.Get("from"), "from", fields),
		To:         parseInstant(values.Get("to"), "to", fields),
		SortBy:     parseSort(values.Get("sort"), fields),
		Descending: parseDescending(values.Get("order"), fields),
	}

	if len(fields) > 0 {
		return application.ChangeFilter{}, fields
	}
	return filter, nil
}

func ToChangeResponse(change domain.Change) ChangeResponse {
	return ChangeResponse{
		ID:          change.ID,
		ChangeSetID: change.ChangeSetID,
		OccurredAt:  change.OccurredAt,
		Field:       string(change.Field),
		Kind:        string(change.Kind),
		OldValue:    change.OldValue,
		NewValue:    change.NewValue,
		Description: render(change),
	}
}

func ToChangeResponses(changes []domain.Change) []ChangeResponse {
	out := make([]ChangeResponse, len(changes))
	for i := range changes {
		out[i] = ToChangeResponse(changes[i])
	}
	return out
}

// parseEnums accepts both repeated params and comma-separated values, so
// ?field=title&field=authors and ?field=title,authors mean the same thing.
func parseEnums[T ~string](values map[string][]string, key string, allowed []T, fields map[string]string) []T {
	var parsed []T

	for _, raw := range values[key] {
		for entry := range strings.SplitSeq(raw, ",") {
			entry = strings.TrimSpace(entry)
			if entry == "" {
				continue
			}
			candidate := T(entry)
			if !slices.Contains(allowed, candidate) {
				fields[key] = fmt.Sprintf("must be one of %s", join(allowed))
				continue
			}
			parsed = append(parsed, candidate)
		}
	}
	return parsed
}

func parseInstant(raw, key string, fields map[string]string) *time.Time {
	if raw == "" {
		return nil
	}

	instant, err := time.Parse(time.RFC3339, raw)
	if err != nil {
		fields[key] = "must be an RFC3339 timestamp"
		return nil
	}

	return new(instant.UTC())
}

func parseSort(raw string, fields map[string]string) application.ChangeSortField {
	if raw == "" {
		return application.SortByOccurredAt
	}
	if candidate := application.ChangeSortField(raw); slices.Contains(sortFields, candidate) {
		return candidate
	}

	fields["sort"] = fmt.Sprintf("must be one of %s", join(sortFields))
	return application.SortByOccurredAt
}

func parseDescending(raw string, fields map[string]string) bool {
	switch raw {
	case "", "desc":
		return true
	case "asc":
		return false
	}

	fields["order"] = "must be one of asc, desc"
	return true
}

func join[T ~string](values []T) string {
	return strings.Join(utils.ToStrings(values), ", ")
}
