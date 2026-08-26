package agent

import (
	"context"
	"errors"
	"fmt"

	"github.com/dharlanoliveira/norvii/apps/api/internal/evaluation/application"
)

// executionClient is the narrow fixed-snapshot agent port used by the managed worker.
type executionClient interface {
	Execute(context.Context, Request) (Result, error)
}

// Processor translates one leased case to the fixed-snapshot agent protocol and accepts only
// results matching the identity frozen when the run was created.
type Processor struct {
	client executionClient
}

// NewProcessor constructs a worker case processor without any public-chat dependency.
func NewProcessor(client executionClient) (*Processor, error) {
	if client == nil {
		return nil, application.ErrInvalidWorkerInput
	}
	return &Processor{client: client}, nil
}

// Process executes only the corpus, snapshot, question, and retrieval selection carried by the
// lease. Both the request and returned identity must match the identity persisted on the run.
func (processor *Processor) Process(ctx context.Context, claim application.ClaimedCase) (application.TerminalCaseResult, error) {
	if processor == nil || processor.client == nil {
		return application.TerminalCaseResult{}, application.ErrInvalidWorkerInput
	}
	result, err := processor.client.Execute(ctx, Request{
		CorpusID: claim.CorpusID, SnapshotID: claim.SnapshotID, Question: claim.Question,
		InterfaceLanguage:      string(claim.QueryLanguage),
		RetrievalConfiguration: FrozenRetrievalConfiguration{Strategy: claim.Configuration.Strategy, Fingerprint: claim.Configuration.Fingerprint},
		ExecutionIdentity:      claim.ExecutionIdentity,
	})
	if err != nil {
		if errors.Is(err, ErrFrozenIdentityUnavailable) {
			return application.TerminalCaseResult{State: application.RunCaseFailed, SafeFailureCode: "frozen_identity_unavailable"}, nil
		}
		return application.TerminalCaseResult{}, fmt.Errorf("execute fixed-snapshot evaluation: %w", err)
	}
	if result.AgentBuildIdentity != claim.ExecutionIdentity.AgentBuild || result.ModelIdentity != claim.ExecutionIdentity.ChatModelIdentity ||
		result.EmbeddingModelIdentity != claim.ExecutionIdentity.EmbeddingModelIdentity {
		return application.TerminalCaseResult{}, fmt.Errorf("evaluation result identity does not match the frozen run identity")
	}
	state := application.RunCaseCompleted
	if result.Materialized.Outcome == application.AgentOutcomeAbstained {
		state = application.RunCaseAbstained
	}
	return application.TerminalCaseResult{
		State: state, Answer: result.Materialized.Answer, GraphGroundingState: result.GraphGrounding.Status,
		LatencyMilliseconds: result.Materialized.LatencyMilliseconds, InputTokens: result.Materialized.InputTokens,
		OutputTokens: result.Materialized.OutputTokens, ActualEvidence: append(result.Materialized.Retrieved, result.Materialized.Cited...),
	}, nil
}

var _ application.CaseProcessor = (*Processor)(nil)
