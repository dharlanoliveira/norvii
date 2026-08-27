package application

import (
	"context"
	"errors"
	"regexp"
	"strings"
	"time"

	"github.com/google/uuid"
)

var (
	// ErrInvalidRunRequest identifies a malformed maintainer request before any run is created.
	ErrInvalidRunRequest = errors.New("evaluation run request is invalid")
	// ErrRunNotFound covers absent evaluation runs and cases outside the selected run.
	ErrRunNotFound = errors.New("evaluation run was not found")
)

var runFingerprintPattern = regexp.MustCompile(`^[0-9a-f]{64}$`)

// RetrievalConfiguration is the frozen evaluation-only selection recorded with a run.
type RetrievalConfiguration struct {
	Strategy    string
	Fingerprint string
}

// ExecutionIdentity is the complete runnable evaluator identity frozen before a run exists.
// Agent responses are verified against this identity by the managed worker.
type ExecutionIdentity struct {
	AgentBuild             string
	ChatModelIdentity      string
	EmbeddingModelIdentity string
}

// RunnableConfiguration is the server-owned evaluation configuration. User input may select
// only the exact retrieval configuration this process can execute.
type RunnableConfiguration struct {
	Retrieval RetrievalConfiguration
	Identity  ExecutionIdentity
}

// StartRunRequest selects only immutable inputs. It has no active-snapshot field or fallback.
type StartRunRequest struct {
	DatasetRevisionID uuid.UUID
	CorpusID          uuid.UUID
	SnapshotID        uuid.UUID
	Configuration     RetrievalConfiguration
	ExecutionIdentity ExecutionIdentity
}

// RunState describes the durable run lifecycle.
type RunState string

const (
	RunQueued                RunState = "queued"
	RunRunning               RunState = "running"
	RunCompleted             RunState = "completed"
	RunCompletedWithFailures RunState = "completed_with_failures"
	RunFailed                RunState = "failed"
)

// Run is the safe, immutable run inspection projection.
type Run struct {
	ID                     uuid.UUID
	DatasetRevisionID      uuid.UUID
	DatasetContentSHA256   string
	CorpusID               uuid.UUID
	SnapshotID             uuid.UUID
	SnapshotManifestSHA256 string
	OrderedCaseSetSHA256   string
	Configuration          RetrievalConfiguration
	ScoringPolicyVersion   string
	AgentBuild             string
	ChatModelIdentity      string
	EmbeddingModelIdentity string
	InitiatedBy            string
	State                  RunState
	CreatedAt              time.Time
	StartedAt              *time.Time
	CompletedAt            *time.Time
	Aggregate              RunAggregate
	Cases                  []RunCaseSummary
}

// RunAggregate reports explicit execution and scoring denominators.
type RunAggregate struct {
	Total         int64
	Eligible      int64
	Scored        int64
	Failed        int64
	Cancelled     int64
	NotApplicable int64
	Metrics       []RunMetric
}

// RunMetric is a safe deterministic scoring component.
type RunMetric struct {
	Name          string
	State         string
	Value         *float64
	Numerator     *int64
	Denominator   *int64
	Rationale     string
	ScorerVersion string
}

// RunCaseSummary supports stable status polling without exposing prompts or provider payloads.
type RunCaseSummary struct {
	ID            uuid.UUID
	DatasetCaseID uuid.UUID
	Position      int
	State         RunCaseState
	AttemptCount  int
	FinishedAt    *time.Time
	FailureCode   string
}

// EvidenceIdentity contains immutable provenance only, never source text or excerpts.
type EvidenceIdentity struct {
	Kind             string
	Position         int
	MarkerPosition   int
	CorpusID         uuid.UUID
	SnapshotID       uuid.UUID
	SourceID         uuid.UUID
	SourceRevisionID uuid.UUID
	DocumentID       uuid.UUID
	LegalUnitID      uuid.UUID
	CanonicalLocator string
	DisplayLocator   string
	StartOffset      *int
	EndOffset        *int
	ContentSHA256    string
}

// RunCase is the safe detailed ledger projection for one immutable run case.
type RunCase struct {
	RunCaseSummary
	RunID               uuid.UUID
	CorpusID            uuid.UUID
	SnapshotID          uuid.UUID
	DatasetRevisionID   uuid.UUID
	Question            string
	ReferenceAnswer     string
	ExpectedOutcome     string
	ExpectedEvidence    []EvidenceIdentity
	ActualEvidence      []EvidenceIdentity
	Answer              string
	GraphGroundingState string
	LatencyMilliseconds *int64
	InputTokens         *int64
	OutputTokens        *int64
	Metrics             []RunMetric
}

// RunStore materializes a preflight-approved immutable ledger and reads its projections.
type RunStore interface {
	CreateEvaluationRun(context.Context, StartRunRequest, PreflightResult) (Run, error)
	GetEvaluationRun(context.Context, uuid.UUID) (Run, error)
	GetEvaluationRunCase(context.Context, uuid.UUID, uuid.UUID) (RunCase, error)
}

// RunService separates compatibility validation from run persistence and reads.
type RunService struct {
	preflight interface {
		Check(context.Context, PreflightRequest) (PreflightResult, error)
	}
	store    RunStore
	runnable RunnableConfiguration
}

// NewRunService composes the all-or-nothing preflight boundary with immutable run persistence.
func NewRunService(preflight interface {
	Check(context.Context, PreflightRequest) (PreflightResult, error)
}, store RunStore, runnable RunnableConfiguration) *RunService {
	return &RunService{preflight: preflight, store: store, runnable: runnable}
}

// Start checks the caller-selected historical snapshot before creating any ledger record.
func (service *RunService) Start(ctx context.Context, request StartRunRequest) (Run, error) {
	if service == nil || service.preflight == nil || service.store == nil || validateStartRunRequest(request) != nil ||
		validateRunnableConfiguration(service.runnable) != nil || request.Configuration != service.runnable.Retrieval {
		return Run{}, ErrInvalidRunRequest
	}
	request.ExecutionIdentity = service.runnable.Identity
	plan, err := service.preflight.Check(ctx, PreflightRequest{
		CorpusID: request.CorpusID, DatasetRevisionID: request.DatasetRevisionID, SnapshotID: request.SnapshotID,
	})
	if err != nil {
		return Run{}, err
	}
	return service.store.CreateEvaluationRun(ctx, request, plan)
}

func validateRunnableConfiguration(configuration RunnableConfiguration) error {
	if (configuration.Retrieval.Strategy != "vector" && configuration.Retrieval.Strategy != "hybrid") ||
		!runFingerprintPattern.MatchString(configuration.Retrieval.Fingerprint) ||
		strings.TrimSpace(configuration.Retrieval.Fingerprint) != configuration.Retrieval.Fingerprint ||
		strings.TrimSpace(configuration.Identity.AgentBuild) == "" ||
		strings.TrimSpace(configuration.Identity.ChatModelIdentity) == "" ||
		strings.TrimSpace(configuration.Identity.EmbeddingModelIdentity) == "" {
		return ErrInvalidRunRequest
	}
	return nil
}

// Get returns a historical ledger projection without resolving the active snapshot.
func (service *RunService) Get(ctx context.Context, runID uuid.UUID) (Run, error) {
	if service == nil || service.store == nil || runID == uuid.Nil {
		return Run{}, ErrInvalidRunRequest
	}
	return service.store.GetEvaluationRun(ctx, runID)
}

// GetCase returns a case only from its run's immutable ledger.
func (service *RunService) GetCase(ctx context.Context, runID, runCaseID uuid.UUID) (RunCase, error) {
	if service == nil || service.store == nil || runID == uuid.Nil || runCaseID == uuid.Nil {
		return RunCase{}, ErrInvalidRunRequest
	}
	return service.store.GetEvaluationRunCase(ctx, runID, runCaseID)
}

func validateStartRunRequest(request StartRunRequest) error {
	if request.CorpusID == uuid.Nil || request.DatasetRevisionID == uuid.Nil || request.SnapshotID == uuid.Nil ||
		(request.Configuration.Strategy != "vector" && request.Configuration.Strategy != "hybrid") ||
		!runFingerprintPattern.MatchString(request.Configuration.Fingerprint) ||
		strings.TrimSpace(request.Configuration.Fingerprint) != request.Configuration.Fingerprint {
		return ErrInvalidRunRequest
	}
	return nil
}
