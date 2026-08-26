package application

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/dharlanoliveira/norvii/apps/api/internal/evaluation/domain"
	"github.com/google/uuid"
)

var (
	// ErrInvalidWorkerInput identifies a malformed lease, terminal result, or worker configuration.
	ErrInvalidWorkerInput = errors.New("evaluation worker input is invalid")
	// ErrLeaseLost identifies work that another recovery worker has safely reclaimed.
	ErrLeaseLost = errors.New("evaluation case lease is no longer held")
)

// RunCaseState is the durable state for one independently executable evaluation case.
type RunCaseState string

const (
	RunCasePending   RunCaseState = "pending"
	RunCaseLeased    RunCaseState = "leased"
	RunCaseCompleted RunCaseState = "completed"
	RunCaseAbstained RunCaseState = "abstained"
	RunCaseFailed    RunCaseState = "failed"
	RunCaseCancelled RunCaseState = "cancelled"
)

// ClaimedCase contains the complete persisted execution identity and question needed by the
// fixed-snapshot agent adapter. It deliberately has no active-snapshot dependency.
type ClaimedCase struct {
	ID                uuid.UUID
	RunID             uuid.UUID
	CorpusID          uuid.UUID
	SnapshotID        uuid.UUID
	DatasetCaseID     uuid.UUID
	Question          string
	QueryLanguage     domain.QueryLanguage
	ExpectedOutcome   domain.ExpectedOutcome
	Configuration     RetrievalConfiguration
	ExecutionIdentity ExecutionIdentity
	AttemptCount      int
	LeaseToken        uuid.UUID
	LeaseExpiresAt    time.Time
}

// TerminalCaseResult is the safe durable result a worker can record exactly once. It carries
// application-produced evidence and metrics without exposing agent transport payloads.
type TerminalCaseResult struct {
	State                RunCaseState
	Answer               string
	GraphGroundingState  string
	SafeFailureCode      string
	LatencyMilliseconds  *int64
	InputTokens          *int64
	OutputTokens         *int64
	ActualEvidence       []ActualEvidence
	Metrics              []Metric
	ScoringPolicyVersion string
}

// CaseProcessor executes one leased case. It is a narrow seam for the later non-streaming agent
// adapter and makes tests deterministic without calling the chat application.
type CaseProcessor interface {
	Process(context.Context, ClaimedCase) (TerminalCaseResult, error)
}

// WorkStore owns lease concurrency and immutable terminal persistence.
type WorkStore interface {
	Claim(context.Context, string, time.Duration, int) ([]ClaimedCase, error)
	Complete(context.Context, ClaimedCase, TerminalCaseResult) error
	ReleaseForRetry(context.Context, ClaimedCase, string, int) error
}

// Worker processes independent leased cases. A processor failure is released for retry and never
// stops unrelated cases in the same claimed batch.
type Worker struct {
	store       WorkStore
	processor   CaseProcessor
	workerID    string
	leasePeriod time.Duration
	maxAttempts int
	batchSize   int
}

// NewWorker constructs a bounded worker. The caller owns polling and cancellation.
func NewWorker(store WorkStore, processor CaseProcessor, workerID string, leasePeriod time.Duration, maxAttempts, batchSize int) (*Worker, error) {
	if store == nil || processor == nil || strings.TrimSpace(workerID) == "" || leasePeriod <= 0 || maxAttempts < 1 || batchSize < 1 {
		return nil, ErrInvalidWorkerInput
	}
	return &Worker{store: store, processor: processor, workerID: workerID, leasePeriod: leasePeriod, maxAttempts: maxAttempts, batchSize: batchSize}, nil
}

// RunOnce claims one bounded batch and independently completes or releases each case. The returned
// count is the number of claims, including transient failures safely released for another attempt.
func (worker *Worker) RunOnce(ctx context.Context) (int, error) {
	if worker == nil {
		return 0, ErrInvalidWorkerInput
	}
	claimed, err := worker.store.Claim(ctx, worker.workerID, worker.leasePeriod, worker.batchSize)
	if err != nil {
		return 0, fmt.Errorf("claim evaluation cases: %w", err)
	}
	for _, evaluationCase := range claimed {
		if err := worker.processCase(ctx, evaluationCase); err != nil {
			return len(claimed), err
		}
	}
	return len(claimed), nil
}

func (worker *Worker) processCase(ctx context.Context, evaluationCase ClaimedCase) error {
	result, err := worker.processor.Process(ctx, evaluationCase)
	if err != nil {
		return worker.releaseForRetry(ctx, evaluationCase, "execution_retryable")
	}
	if err := validateTerminalCaseResult(result); err != nil {
		return worker.releaseForRetry(ctx, evaluationCase, "invalid_execution_result")
	}
	if err := worker.store.Complete(ctx, evaluationCase, result); err != nil && !errors.Is(err, ErrLeaseLost) {
		return fmt.Errorf("complete evaluation case %s: %w", evaluationCase.ID, err)
	}
	return nil
}

func (worker *Worker) releaseForRetry(ctx context.Context, evaluationCase ClaimedCase, failureCode string) error {
	if err := worker.store.ReleaseForRetry(ctx, evaluationCase, failureCode, worker.maxAttempts); err != nil {
		if errors.Is(err, ErrLeaseLost) {
			return nil
		}
		return fmt.Errorf("release failed evaluation case %s: %w", evaluationCase.ID, err)
	}
	return nil
}

func validateTerminalCaseResult(result TerminalCaseResult) error {
	if result.State != RunCaseCompleted && result.State != RunCaseAbstained && result.State != RunCaseFailed && result.State != RunCaseCancelled {
		return ErrInvalidWorkerInput
	}
	if result.State == RunCaseCompleted && strings.TrimSpace(result.Answer) == "" {
		return ErrInvalidWorkerInput
	}
	if (result.State == RunCaseFailed || result.State == RunCaseCancelled) != (strings.TrimSpace(result.SafeFailureCode) != "") {
		return ErrInvalidWorkerInput
	}
	for _, measurement := range []*int64{result.LatencyMilliseconds, result.InputTokens, result.OutputTokens} {
		if measurement != nil && *measurement < 0 {
			return ErrInvalidWorkerInput
		}
	}
	if len(result.Metrics) > 0 && strings.TrimSpace(result.ScoringPolicyVersion) == "" {
		return ErrInvalidWorkerInput
	}
	for _, metric := range result.Metrics {
		if metric.Name == "" || metric.State == "" || strings.TrimSpace(metric.Rationale) == "" {
			return ErrInvalidWorkerInput
		}
	}
	return nil
}
