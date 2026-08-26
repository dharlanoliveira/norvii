package agent

import (
	"context"
	"testing"
	"time"

	"github.com/dharlanoliveira/norvii/apps/api/internal/evaluation/application"
	"github.com/dharlanoliveira/norvii/apps/api/internal/evaluation/domain"
	"github.com/google/uuid"
)

func TestProcessorExecutesOnlyClaimedFixedSnapshotAndAcceptsFrozenIdentity(t *testing.T) {
	claim := processorClaim()
	materialized, err := application.MaterializeCaseResult(claim.CorpusID, claim.SnapshotID, application.AgentCaseResult{
		State: application.CaseExecutionCompleted, Outcome: application.AgentOutcomeCompleted, Answer: "A safe answer.",
	})
	if err != nil {
		t.Fatalf("MaterializeCaseResult() error = %v", err)
	}
	client := &processorClient{result: Result{Materialized: materialized, ModelIdentity: "chat-model", AgentBuildIdentity: "agent-build", EmbeddingModelIdentity: "embedding-model", GraphGrounding: GraphGrounding{Status: graphNotRequested}}}
	processor, err := NewProcessor(client)
	if err != nil {
		t.Fatalf("NewProcessor() error = %v", err)
	}
	result, err := processor.Process(context.Background(), claim)
	if err != nil {
		t.Fatalf("Process() error = %v", err)
	}
	if result.State != application.RunCaseCompleted || client.request.CorpusID != claim.CorpusID || client.request.SnapshotID != claim.SnapshotID || client.request.RetrievalConfiguration.Fingerprint != claim.Configuration.Fingerprint || client.request.ExecutionIdentity != claim.ExecutionIdentity {
		t.Fatalf("Process() result = %#v, request = %#v", result, client.request)
	}
}

func TestProcessorRejectsAgentIdentityDifferentFromFrozenRunIdentity(t *testing.T) {
	claim := processorClaim()
	materialized, err := application.MaterializeCaseResult(claim.CorpusID, claim.SnapshotID, application.AgentCaseResult{
		State: application.CaseExecutionCompleted, Outcome: application.AgentOutcomeCompleted, Answer: "A safe answer.",
	})
	if err != nil {
		t.Fatalf("MaterializeCaseResult() error = %v", err)
	}
	processor, err := NewProcessor(&processorClient{result: Result{Materialized: materialized, ModelIdentity: "other-model", AgentBuildIdentity: "agent-build", EmbeddingModelIdentity: "embedding-model"}})
	if err != nil {
		t.Fatalf("NewProcessor() error = %v", err)
	}
	if _, err := processor.Process(context.Background(), claim); err == nil {
		t.Fatal("Process() error = nil, want frozen identity rejection")
	}
}

func TestProcessorFailsClosedWhenAgentCannotServeFrozenIdentity(t *testing.T) {
	claim := processorClaim()
	processor, err := NewProcessor(&processorClient{err: ErrFrozenIdentityUnavailable})
	if err != nil {
		t.Fatalf("NewProcessor() error = %v", err)
	}
	result, err := processor.Process(context.Background(), claim)
	if err != nil {
		t.Fatalf("Process() error = %v", err)
	}
	if result.State != application.RunCaseFailed || result.SafeFailureCode != "frozen_identity_unavailable" {
		t.Fatalf("Process() = %#v, want bounded terminal frozen identity failure", result)
	}
}

type processorClient struct {
	request Request
	result  Result
	err     error
}

func (client *processorClient) Execute(_ context.Context, request Request) (Result, error) {
	client.request = request
	return client.result, client.err
}

func processorClaim() application.ClaimedCase {
	return application.ClaimedCase{
		ID: uuid.New(), RunID: uuid.New(), CorpusID: uuid.New(), SnapshotID: uuid.New(), DatasetCaseID: uuid.New(),
		Question: "Synthetic question?", QueryLanguage: domain.QueryLanguageEnglish, ExpectedOutcome: domain.ExpectedOutcomeAnswer,
		Configuration:     application.RetrievalConfiguration{Strategy: "vector", Fingerprint: "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"},
		ExecutionIdentity: application.ExecutionIdentity{AgentBuild: "agent-build", ChatModelIdentity: "chat-model", EmbeddingModelIdentity: "embedding-model"},
		LeaseToken:        uuid.New(), LeaseExpiresAt: time.Now().Add(time.Minute),
	}
}
