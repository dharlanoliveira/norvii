// Package domain defines corpus invariants without transport or persistence dependencies.
package domain

import (
	"strings"
	"time"

	"github.com/google/uuid"
)

// Language identifies the preserved language of corpus legal content.
type Language string

const (
	// LanguageEnglish identifies English legal content.
	LanguageEnglish Language = "en"
	// LanguagePortuguese identifies Portuguese legal content.
	LanguagePortuguese Language = "pt"
)

// Status is the reversible corpus lifecycle state.
type Status string

const (
	// StatusEnabled permits researcher selection.
	StatusEnabled Status = "enabled"
	// StatusDisabled hides the corpus from researcher selection.
	StatusDisabled Status = "disabled"
)

// ErrorCode is a stable public failure category.
type ErrorCode string

const (
	// ErrorInvalidInput identifies rejected caller-controlled metadata.
	ErrorInvalidInput ErrorCode = "invalid_input"
	// ErrorStaleState identifies an optimistic concurrency conflict.
	ErrorStaleState ErrorCode = "stale_state"
)

// Error is a safe domain failure suitable for mapping to the public error contract.
type Error struct {
	Code     ErrorCode
	Message  string
	Fields   map[string]string
	Internal error
}

// Error returns only the safe public message.
func (err *Error) Error() string { return err.Message }

// Unwrap exposes an internal cause to trusted error handling only.
func (err *Error) Unwrap() error { return err.Internal }

// ErrStaleState is returned when an expected version no longer matches.
var ErrStaleState = &Error{Code: ErrorStaleState, Message: "The corpus changed; reload and retry."}

// Draft contains mutable corpus metadata.
type Draft struct {
	Name         string
	Description  string
	Language     Language
	Jurisdiction string
}

// Corpus is the aggregate root for one isolated legal collection.
type Corpus struct {
	ID           uuid.UUID
	Name         string
	Description  string
	Language     Language
	Jurisdiction string
	Status       Status
	Version      int
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

// NewCorpus validates and creates an enabled corpus with stable identity.
func NewCorpus(id uuid.UUID, draft Draft, now time.Time) (Corpus, error) {
	normalized, err := NormalizeDraft(draft)
	if err != nil {
		return Corpus{}, err
	}
	if id == uuid.Nil {
		return Corpus{}, invalidField("id", "A corpus identifier is required.")
	}
	now = now.UTC()
	return Corpus{
		ID:           id,
		Name:         normalized.Name,
		Description:  normalized.Description,
		Language:     normalized.Language,
		Jurisdiction: normalized.Jurisdiction,
		Status:       StatusEnabled,
		Version:      1,
		CreatedAt:    now,
		UpdatedAt:    now,
	}, nil
}

// Update replaces mutable metadata when the caller has the current version.
func (corpus *Corpus) Update(draft Draft, expectedVersion int, now time.Time) error {
	if err := corpus.checkVersion(expectedVersion); err != nil {
		return err
	}
	normalized, err := NormalizeDraft(draft)
	if err != nil {
		return err
	}
	corpus.Name = normalized.Name
	corpus.Description = normalized.Description
	corpus.Language = normalized.Language
	corpus.Jurisdiction = normalized.Jurisdiction
	corpus.recordMutation(now)
	return nil
}

// Disable makes a corpus unavailable without deleting its identity or sources.
func (corpus *Corpus) Disable(expectedVersion int, now time.Time) error {
	if err := corpus.checkVersion(expectedVersion); err != nil {
		return err
	}
	if corpus.Status != StatusDisabled {
		corpus.Status = StatusDisabled
		corpus.recordMutation(now)
	}
	return nil
}

// Enable restores a disabled corpus for researcher selection.
func (corpus *Corpus) Enable(expectedVersion int, now time.Time) error {
	if err := corpus.checkVersion(expectedVersion); err != nil {
		return err
	}
	if corpus.Status != StatusEnabled {
		corpus.Status = StatusEnabled
		corpus.recordMutation(now)
	}
	return nil
}

func (corpus *Corpus) checkVersion(expected int) error {
	if corpus.Version != expected {
		return ErrStaleState
	}
	return nil
}

func (corpus *Corpus) recordMutation(now time.Time) {
	corpus.Version++
	corpus.UpdatedAt = now.UTC()
}

// NormalizeDraft trims and validates every mutable corpus metadata field.
func NormalizeDraft(draft Draft) (Draft, error) {
	draft.Name = strings.TrimSpace(draft.Name)
	draft.Description = strings.TrimSpace(draft.Description)
	draft.Jurisdiction = strings.TrimSpace(draft.Jurisdiction)
	fields := []struct {
		name  string
		value string
	}{
		{name: "name", value: draft.Name},
		{name: "description", value: draft.Description},
		{name: "jurisdiction", value: draft.Jurisdiction},
	}
	for _, field := range fields {
		if field.value == "" {
			return Draft{}, invalidField(field.name, "This field is required.")
		}
	}
	if draft.Language != LanguageEnglish && draft.Language != LanguagePortuguese {
		return Draft{}, invalidField("language", "Language must be English or Portuguese.")
	}
	return draft, nil
}

func invalidField(field, message string) *Error {
	return &Error{
		Code:    ErrorInvalidInput,
		Message: "The corpus metadata is invalid.",
		Fields:  map[string]string{field: message},
	}
}
