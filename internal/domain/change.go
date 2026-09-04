package domain

import "time"

type ChangeField string

const (
	FieldTitle       ChangeField = "title"
	FieldDescription ChangeField = "description"
	FieldPublishedOn ChangeField = "published_on"
	FieldAuthors     ChangeField = "authors"
)

type ChangeKind string

const (
	KindSet     ChangeKind = "set"
	KindAdded   ChangeKind = "added"
	KindRemoved ChangeKind = "removed"
)

type Change struct {
	ID          int64
	BookID      string
	ChangeSetID string
	OccurredAt  time.Time
	Field       ChangeField
	Kind        ChangeKind
	OldValue    *string
	NewValue    *string
}
