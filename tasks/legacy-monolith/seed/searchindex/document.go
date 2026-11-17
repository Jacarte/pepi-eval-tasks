package searchindex

import (
	"fmt"
	"strings"
	"time"
)

// Document is a unit of indexed content.
//
// The zero value is not valid; build one with all required fields and run
// Validate before handing it to a store.
type Document struct {
	ID        string
	Kind      string // product, category or page
	Title     string
	Body      string
	Boost     int
	Tags      []string
	Status    DocumentStatus
	CreatedAt time.Time
	UpdatedAt time.Time
}

// DocumentStatus is the lifecycle state of a document.
type DocumentStatus int

const (
	DocumentStatusUnknown DocumentStatus = iota
	DocumentStatusPending
	DocumentStatusIndexed
	DocumentStatusStale
)

var documentStatusNames = []string{"unknown", "pending", "indexed", "stale"}

func (s DocumentStatus) String() string {
	if s < 0 || int(s) >= len(documentStatusNames) {
		return fmt.Sprintf("DocumentStatus(%d)", int(s))
	}
	return documentStatusNames[s]
}

// ParseDocumentStatus parses the wire name of a status. Matching is
// case-insensitive because the old importer wrote upper-case values.
func ParseDocumentStatus(v string) (DocumentStatus, error) {
	v = strings.ToLower(strings.TrimSpace(v))
	for i, n := range documentStatusNames {
		if i > 0 && n == v {
			return DocumentStatus(i), nil
		}
	}
	return DocumentStatusUnknown, fmt.Errorf("%w: unknown document status %q", ErrInvalid, v)
}

var documentTransitions = map[DocumentStatus][]DocumentStatus{
	DocumentStatusPending: {DocumentStatusIndexed, DocumentStatusStale},
	DocumentStatusIndexed: {DocumentStatusStale},
}

// CanTransition reports whether a document may move from s to next.
func (s DocumentStatus) CanTransition(next DocumentStatus) bool {
	for _, t := range documentTransitions[s] {
		if t == next {
			return true
		}
	}
	return false
}

// Terminal reports whether no further transitions are possible.
func (s DocumentStatus) Terminal() bool { return len(documentTransitions[s]) == 0 }

// Validate checks the invariants every stored document must satisfy.
func (v *Document) Validate() error {
	if strings.TrimSpace(v.ID) == "" {
		return &ValidationError{Entity: "document", Field: "id", Reason: "is required"}
	}
	if strings.TrimSpace(v.Kind) == "" {
		return &ValidationError{Entity: "document", Field: "kind", Reason: "is required"}
	}
	if strings.TrimSpace(v.Title) == "" {
		return &ValidationError{Entity: "document", Field: "title", Reason: "is required"}
	}
	if v.Boost < 0 {
		return &ValidationError{Entity: "document", Field: "boost", Reason: "must not be negative"}
	}
	for _, tag := range v.Tags {
		if strings.TrimSpace(tag) == "" {
			return &ValidationError{Entity: "document", Field: "tags", Reason: "contains an empty tag"}
		}
	}
	if v.Status <= DocumentStatusUnknown || int(v.Status) >= len(documentStatusNames) {
		return &ValidationError{Entity: "document", Field: "status", Reason: "is not a known status"}
	}
	return nil
}

// Clone returns a deep copy so callers can mutate the result freely.
func (v *Document) Clone() *Document {
	if v == nil {
		return nil
	}
	cp := *v
	if v.Tags != nil {
		cp.Tags = append([]string(nil), v.Tags...)
	}
	return &cp
}
