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
	ErrSnapshotUnavailable  = errors.New("active corpus snapshot is unavailable")
)

// Request identifies one ephemeral question within one active corpus.
type Request struct {
	CorpusID          uuid.UUID
	SnapshotID        uuid.UUID
	Question          string
	InterfaceLanguage string
	Strategy          string
}

// Evidence is an immutable, corpus-owned support location.
type Evidence struct {
	ID         string
	CorpusID   uuid.UUID
	SnapshotID uuid.UUID
	SourceID   uuid.UUID
	DocumentID uuid.UUID
	// DocumentVersionID is the immutable document identity used by citation inspection.
	DocumentVersionID uuid.UUID
	SourceRevisionID  uuid.UUID
	PipelineVersion   string
	SourceTitle       string
	UnitLocator       string
	StartOffset       int
	EndOffset         int
	Excerpt           string
	Rank              int
	CosineDistance    *float64
	Contribution      string
}

// RetrievalInspection describes the bounded retrieval operation without exposing provider data.
type RetrievalInspection struct {
	Strategy       string
	TopK           int
	ReturnedCount  int
	EmbeddingModel *string
}

// Measurements contains only values reported by their owning component.
type Measurements struct {
	RetrievalMilliseconds  *int64
	GenerationMilliseconds *int64
	TotalMilliseconds      *int64
	InputTokens            *int64
	OutputTokens           *int64
}

// RetrievalStage records one public, content-free retrieval phase.
type RetrievalStage struct {
	Name                 string
	State                string
	EvidenceCount        int
	DurationMilliseconds *int64
	ReasonCode           *string
	InputTokens          *int64
	OutputTokens         *int64
}

// AssertionPathStep identifies one published normative assertion and its legal provenance.
type AssertionPathStep struct {
	AssertionID         string
	Predicate           string
	SubjectLabel        string
	ObjectLabel         string
	EstablishingLocator string
	EvidenceLocator     string
	HierarchyContext    []string
	Qualifier           *string
}

// Inspection is an ephemeral, safe-to-display answer diagnostic.
type Inspection struct {
	Outcome       string
	Retrieval     *RetrievalInspection
	Measurements  Measurements
	Evidence      []Evidence
	AssertionPath []AssertionPathStep
	ScopeLocator  *string
	Stages        []RetrievalStage
}

// Result is the validated terminal answer and its supporting evidence.
type Result struct {
	Answer     string
	Evidence   []Evidence
	Inspection *Inspection
}
