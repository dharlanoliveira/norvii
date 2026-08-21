// Package domain contains corpus-scoped grounded chat rules independent of transports and vendors.
package domain

import (
	"errors"

	"github.com/google/uuid"
)

var (
	ErrInvalidQuestion      = errors.New("research question is invalid")
	ErrInsufficientEvidence = errors.New("research evidence is insufficient")
	ErrGroundingValidation  = errors.New("generated answer failed grounding validation")
	ErrRetrievalFailed      = errors.New("grounded evidence retrieval failed")
	ErrGenerationFailed     = errors.New("grounded answer generation failed")
)

// Request identifies one ephemeral question within one active corpus.
type Request struct {
	CorpusID          uuid.UUID
	Question          string
	InterfaceLanguage string
}

// Evidence is an immutable, corpus-owned support location.
type Evidence struct {
	ID          string
	CorpusID    uuid.UUID
	SourceID    uuid.UUID
	DocumentID  uuid.UUID
	UnitLocator string
	StartOffset int
	EndOffset   int
	Excerpt     string
	Rank        int
}

// Result is the validated terminal answer and its supporting evidence.
type Result struct {
	Answer   string
	Evidence []Evidence
}
