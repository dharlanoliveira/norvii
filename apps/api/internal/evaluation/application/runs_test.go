package application

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"
)

func TestRunServiceStartsOnlyAfterSuccessfulFixedSnapshotPreflight(t *testing.T) {
	request := StartRunRequest{
		DatasetRevisionID: uuid.New(), CorpusID: uuid.New(), SnapshotID: uuid.New(),
		Configuration: RetrievalConfiguration{Strategy: "vector", Fingerprint: "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"},
	}
	store := &runStoreStub{run: Run{ID: uuid.New(), SnapshotID: request.SnapshotID}}
	preflight := &preflightStub{result: PreflightResult{CorpusID: request.CorpusID, DatasetRevisionID: request.DatasetRevisionID, SnapshotID: request.SnapshotID}}
	service := NewRunService(preflight, store, runnableConfiguration(request.Configuration))

	result, err := service.Start(context.Background(), request)
	if err != nil {
		t.Fatalf("Start() error = %v", err)
	}
	if result.ID != store.run.ID || preflight.calls != 1 || store.createCalls != 1 {
		t.Fatalf("Start() = %#v, preflight calls = %d, store calls = %d", result, preflight.calls, store.createCalls)
	}
	if store.request.SnapshotID != request.SnapshotID || store.plan.SnapshotID != request.SnapshotID {
		t.Fatalf("Start() changed requested historical snapshot: %#v", store)
	}
}

func TestRunServiceDoesNotCreateLedgerAfterFailedPreflight(t *testing.T) {
	request := StartRunRequest{
		DatasetRevisionID: uuid.New(), CorpusID: uuid.New(), SnapshotID: uuid.New(),
		Configuration: RetrievalConfiguration{Strategy: "hybrid", Fingerprint: "bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb"},
	}
	store := &runStoreStub{}
	service := NewRunService(&preflightStub{err: ErrSnapshotIncompatible}, store, runnableConfiguration(request.Configuration))

	_, err := service.Start(context.Background(), request)
	if !errors.Is(err, ErrSnapshotIncompatible) {
		t.Fatalf("Start() error = %v, want %v", err, ErrSnapshotIncompatible)
	}
	if store.createCalls != 0 {
		t.Fatalf("CreateEvaluationRun calls = %d, want zero after rejected preflight", store.createCalls)
	}
}

func TestRunServiceRejectsConfigurationOutsideRunnableIdentityBeforePreflight(t *testing.T) {
	request := StartRunRequest{
		DatasetRevisionID: uuid.New(), CorpusID: uuid.New(), SnapshotID: uuid.New(),
		Configuration: RetrievalConfiguration{Strategy: "vector", Fingerprint: "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"},
	}
	preflight := &preflightStub{}
	store := &runStoreStub{}
	_, err := NewRunService(preflight, store, runnableConfiguration(RetrievalConfiguration{
		Strategy: "hybrid", Fingerprint: "bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb",
	})).Start(context.Background(), request)
	if !errors.Is(err, ErrInvalidRunRequest) {
		t.Fatalf("Start() error = %v, want %v", err, ErrInvalidRunRequest)
	}
	if preflight.calls != 0 || store.createCalls != 0 {
		t.Fatalf("preflight calls = %d, create calls = %d, want no persisted run", preflight.calls, store.createCalls)
	}
}

func TestRunServiceFreezesRunnableIdentityBeforePersistence(t *testing.T) {
	request := StartRunRequest{
		DatasetRevisionID: uuid.New(), CorpusID: uuid.New(), SnapshotID: uuid.New(),
		Configuration: RetrievalConfiguration{Strategy: "vector", Fingerprint: "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"},
	}
	store := &runStoreStub{run: Run{ID: uuid.New()}}
	service := NewRunService(&preflightStub{result: PreflightResult{CorpusID: request.CorpusID, DatasetRevisionID: request.DatasetRevisionID, SnapshotID: request.SnapshotID}}, store, runnableConfiguration(request.Configuration))
	if _, err := service.Start(context.Background(), request); err != nil {
		t.Fatalf("Start() error = %v", err)
	}
	if store.request.ExecutionIdentity.AgentBuild != "agent-build" || store.request.ExecutionIdentity.ChatModelIdentity != "chat-model" || store.request.ExecutionIdentity.EmbeddingModelIdentity != "embedding-model" {
		t.Fatalf("frozen execution identity = %#v", store.request.ExecutionIdentity)
	}
}

func runnableConfiguration(retrieval RetrievalConfiguration) RunnableConfiguration {
	return RunnableConfiguration{Retrieval: retrieval, Identity: ExecutionIdentity{
		AgentBuild: "agent-build", ChatModelIdentity: "chat-model", EmbeddingModelIdentity: "embedding-model",
	}}
}

type preflightStub struct {
	result PreflightResult
	err    error
	calls  int
}

func (stub *preflightStub) Check(_ context.Context, _ PreflightRequest) (PreflightResult, error) {
	stub.calls++
	return stub.result, stub.err
}

type runStoreStub struct {
	run         Run
	request     StartRunRequest
	plan        PreflightResult
	createCalls int
}

func (stub *runStoreStub) CreateEvaluationRun(_ context.Context, request StartRunRequest, plan PreflightResult) (Run, error) {
	stub.createCalls++
	stub.request, stub.plan = request, plan
	return stub.run, nil
}

func (stub *runStoreStub) GetEvaluationRun(context.Context, uuid.UUID) (Run, error) {
	return Run{}, nil
}
func (stub *runStoreStub) GetEvaluationRunCase(context.Context, uuid.UUID, uuid.UUID) (RunCase, error) {
	return RunCase{}, nil
}
