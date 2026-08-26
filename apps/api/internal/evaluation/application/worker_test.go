package application

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/dharlanoliveira/norvii/apps/api/internal/evaluation/domain"
	"github.com/google/uuid"
)

func TestWorkerContinuesAfterIndependentFailureAndRetriesOnlyLeasedCase(t *testing.T) {
	first := claimedCase(1)
	second := claimedCase(2)
	store := &workerStoreFake{claims: []ClaimedCase{first, second}}
	processor := processorFake{results: map[uuid.UUID]processorResult{
		first.ID:  {err: errors.New("temporary provider failure")},
		second.ID: {result: TerminalCaseResult{State: RunCaseCompleted, Answer: "Safe answer."}},
	}}
	worker, err := NewWorker(store, processor, "test-worker", time.Minute, 3, 2)
	if err != nil {
		t.Fatalf("NewWorker() error = %v", err)
	}

	processed, err := worker.RunOnce(context.Background())
	if err != nil {
		t.Fatalf("RunOnce() error = %v", err)
	}
	if processed != 2 {
		t.Fatalf("RunOnce() processed = %d, want 2", processed)
	}
	if len(store.released) != 1 || store.released[0].ID != first.ID || store.released[0].failureCode != "execution_retryable" {
		t.Fatalf("released cases = %#v, want only the failed lease", store.released)
	}
	if len(store.completed) != 1 || store.completed[0].claim.ID != second.ID || store.completed[0].result.State != RunCaseCompleted {
		t.Fatalf("completed cases = %#v, want the unrelated completed case", store.completed)
	}
}

func TestWorkerReleasesInvalidTerminalResultAndDoesNotWriteIt(t *testing.T) {
	claim := claimedCase(3)
	store := &workerStoreFake{claims: []ClaimedCase{claim}}
	worker, err := NewWorker(store, processorFake{results: map[uuid.UUID]processorResult{
		claim.ID: {result: TerminalCaseResult{State: RunCaseCompleted}},
	}}, "test-worker", time.Minute, 1, 1)
	if err != nil {
		t.Fatalf("NewWorker() error = %v", err)
	}
	if _, err := worker.RunOnce(context.Background()); err != nil {
		t.Fatalf("RunOnce() error = %v", err)
	}
	if len(store.completed) != 0 {
		t.Fatalf("completed cases = %#v, want no invalid terminal write", store.completed)
	}
	if len(store.released) != 1 || store.released[0].failureCode != "invalid_execution_result" {
		t.Fatalf("released cases = %#v, want invalid result recovery", store.released)
	}
}

func TestNewWorkerRejectsMissingExecutionBoundary(t *testing.T) {
	if _, err := NewWorker(&workerStoreFake{}, nil, "worker", time.Second, 1, 1); !errors.Is(err, ErrInvalidWorkerInput) {
		t.Fatalf("NewWorker() error = %v, want %v", err, ErrInvalidWorkerInput)
	}
}

type workerStoreFake struct {
	claims    []ClaimedCase
	completed []completedCase
	released  []releasedCase
}

type completedCase struct {
	claim  ClaimedCase
	result TerminalCaseResult
}

type releasedCase struct {
	ClaimedCase
	failureCode string
	maxAttempts int
}

func (store *workerStoreFake) Claim(context.Context, string, time.Duration, int) ([]ClaimedCase, error) {
	return store.claims, nil
}

func (store *workerStoreFake) Complete(_ context.Context, claim ClaimedCase, result TerminalCaseResult) error {
	store.completed = append(store.completed, completedCase{claim: claim, result: result})
	return nil
}

func (store *workerStoreFake) ReleaseForRetry(_ context.Context, claim ClaimedCase, failureCode string, maxAttempts int) error {
	store.released = append(store.released, releasedCase{ClaimedCase: claim, failureCode: failureCode, maxAttempts: maxAttempts})
	return nil
}

type processorFake struct{ results map[uuid.UUID]processorResult }

type processorResult struct {
	result TerminalCaseResult
	err    error
}

func (processor processorFake) Process(_ context.Context, claim ClaimedCase) (TerminalCaseResult, error) {
	result := processor.results[claim.ID]
	return result.result, result.err
}

func claimedCase(sequence byte) ClaimedCase {
	return ClaimedCase{
		ID: uuid.New(), RunID: uuid.New(), CorpusID: uuid.New(), SnapshotID: uuid.New(), DatasetCaseID: uuid.New(),
		Question: "Synthetic question?", QueryLanguage: domain.QueryLanguageEnglish,
		ExpectedOutcome: domain.ExpectedOutcomeAnswer, AttemptCount: 1, LeaseToken: uuid.New(),
		Configuration:     RetrievalConfiguration{Strategy: "vector", Fingerprint: "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"},
		ExecutionIdentity: ExecutionIdentity{AgentBuild: "agent-build", ChatModelIdentity: "chat-model", EmbeddingModelIdentity: "embedding-model"},
		LeaseExpiresAt:    time.Date(2026, time.August, 26, 18, 0, int(sequence), 0, time.UTC),
	}
}
