//go:build integration

package integration_test

import (
	"context"
	"testing"
	"time"

	evaluationapplication "github.com/dharlanoliveira/norvii/apps/api/internal/evaluation/application"
	evaluationdomain "github.com/dharlanoliveira/norvii/apps/api/internal/evaluation/domain"
	evaluationpostgres "github.com/dharlanoliveira/norvii/apps/api/internal/evaluation/postgres"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

func TestEvaluationComparisonRepositoryReadsImmutablePairedHistoricalMetrics(t *testing.T) {
	ctx, connection := openEvaluationSchemaConnection(t)
	transaction := beginEvaluationSchemaTransaction(t, ctx, connection)
	defer rollbackEvaluationSchemaTransaction(t, ctx, transaction)

	fixture := newEvaluationResolutionFixture(evaluationLGPDCorpusID, "20000000-0000-4000-8000-000000000001")
	seedEvaluationResolutionFixture(t, ctx, transaction, &fixture)
	firstCaseID, _ := insertEvaluationCasePair(t, ctx, transaction, fixture.revisionID, uuid.NewString(), uuid.NewString(), 1)
	secondCaseID, _ := insertEvaluationCasePair(t, ctx, transaction, fixture.revisionID, uuid.NewString(), uuid.NewString(), 3)
	repository := evaluationpostgres.NewRepository(transaction)

	leftRunID := uuid.New()
	leftRequest := evaluationComparisonRunRequest(fixture, leftRunID, firstCaseID, secondCaseID, "vector", "left-configuration", "left-chat")
	if err := repository.CreateRun(ctx, leftRequest); err != nil {
		t.Fatalf("CreateRun(left) error = %v", err)
	}
	completeComparisonRun(t, ctx, repository, leftRunID, firstCaseID, secondCaseID, true)

	rightRunID := uuid.New()
	rightRequest := evaluationComparisonRunRequest(fixture, rightRunID, firstCaseID, secondCaseID, "hybrid", "right-configuration", "right-chat")
	if err := repository.CreateRun(ctx, rightRequest); err != nil {
		t.Fatalf("CreateRun(right) error = %v", err)
	}
	completeComparisonRun(t, ctx, repository, rightRunID, firstCaseID, secondCaseID, false)

	comparison, err := evaluationapplication.NewComparisonService(repository).Compare(ctx, evaluationapplication.ComparisonRequest{
		LeftRunID: leftRunID, RightRunID: rightRunID,
	})
	if err != nil {
		t.Fatalf("Compare() error = %v", err)
	}
	if comparison.State != evaluationapplication.ComparisonStateComparable || comparison.Left.RetrievalStrategy != "vector" ||
		comparison.Right.RetrievalStrategy != "hybrid" || comparison.Left.ChatModelIdentity != "left-chat" || comparison.Right.ChatModelIdentity != "right-chat" {
		t.Fatalf("Compare() identity = %#v, want comparable historical configuration and model identities", comparison)
	}
	if comparison.Totals.PairedCases != 2 || comparison.Totals.FailedOrCancelled != 1 {
		t.Fatalf("Compare().Totals = %#v, want failed historical pair kept explicit", comparison.Totals)
	}
	retrieval := integrationComparisonMetric(t, comparison, evaluationapplication.MetricRetrievalCoverage)
	if retrieval.PairedCases != 1 || retrieval.LeftNumerator != 1 || retrieval.LeftDenominator != 1 ||
		retrieval.RightNumerator != 0 || retrieval.RightDenominator != 1 || retrieval.Delta == nil || *retrieval.Delta != -1 {
		t.Fatalf("retrieval comparison = %#v, want only the jointly scored case and right-minus-left delta", retrieval)
	}

	savepoint, err := transaction.Begin(ctx)
	if err != nil {
		t.Fatalf("begin immutable comparison run update savepoint: %v", err)
	}
	expectPostgresErrorCode(t, evaluationSchemaExecError(savepoint.Exec(ctx,
		`UPDATE evaluation_run SET chat_model_identity = 'replacement-model' WHERE id = $1`, leftRunID,
	)), "55000")
	if err := savepoint.Rollback(ctx); err != nil {
		t.Fatalf("rollback immutable comparison run update savepoint: %v", err)
	}
}

func TestEvaluationComparisonMetricLedgerRejectsMalformedTerminalRuns(t *testing.T) {
	ctx, connection := openEvaluationSchemaConnection(t)

	tests := []struct {
		name                        string
		caseState                   string
		runState                    string
		mutateLedger                func([]directMetricLedgerEntry) []directMetricLedgerEntry
		disableScorerVersionTrigger bool
	}{
		{
			name:      "incomplete ledger",
			caseState: "completed",
			runState:  "completed",
			mutateLedger: func(ledger []directMetricLedgerEntry) []directMetricLedgerEntry {
				return ledger[:len(ledger)-1]
			},
		},
		{
			name:      "duplicate ledger component",
			caseState: "completed",
			runState:  "completed",
			mutateLedger: func(ledger []directMetricLedgerEntry) []directMetricLedgerEntry {
				return append(ledger, directMetricLedgerEntry{component: ledger[0].component, scorerVersion: "v2", state: "not_scored"})
			},
			disableScorerVersionTrigger: true,
		},
		{
			name:      "unknown ledger component",
			caseState: "completed",
			runState:  "completed",
			mutateLedger: func(ledger []directMetricLedgerEntry) []directMetricLedgerEntry {
				ledger[len(ledger)-1].component = "unknown_component"
				return ledger
			},
		},
		{
			name:      "mismatched scorer version",
			caseState: "completed",
			runState:  "completed",
			mutateLedger: func(ledger []directMetricLedgerEntry) []directMetricLedgerEntry {
				ledger[0].scorerVersion = "v2"
				return ledger
			},
			disableScorerVersionTrigger: true,
		},
		{
			name:      "scored failed case",
			caseState: "failed",
			runState:  "failed",
			mutateLedger: func(ledger []directMetricLedgerEntry) []directMetricLedgerEntry {
				ledger[0].state = "scored"
				return ledger
			},
		},
		{
			name:      "scored cancelled case",
			caseState: "cancelled",
			runState:  "failed",
			mutateLedger: func(ledger []directMetricLedgerEntry) []directMetricLedgerEntry {
				ledger[0].state = "scored"
				return ledger
			},
		},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			transaction := beginEvaluationSchemaTransaction(t, ctx, connection)
			defer rollbackEvaluationSchemaTransaction(t, ctx, transaction)

			fixture := newEvaluationResolutionFixture(evaluationLGPDCorpusID, "20000000-0000-4000-8000-000000000001")
			seedEvaluationResolutionFixture(t, ctx, transaction, &fixture)
			datasetCaseID, _ := insertEvaluationCasePair(t, ctx, transaction, fixture.revisionID, uuid.NewString(), uuid.NewString(), 1)
			runID, runCaseID := uuid.New(), uuid.New()
			repository := evaluationpostgres.NewRepository(transaction)
			if err := repository.CreateRun(ctx, evaluationRunRequest(fixture, runID, runCaseID, datasetCaseID, evaluationdomain.ExpectedOutcomeAnswer)); err != nil {
				t.Fatalf("CreateRun() error = %v", err)
			}
			claims, err := repository.Claim(ctx, "ledger-constraint-worker", time.Minute, 1)
			if err != nil {
				t.Fatalf("Claim() error = %v", err)
			}
			if len(claims) != 1 || claims[0].ID != runCaseID {
				t.Fatalf("Claim() = %#v, want the fixture case", claims)
			}

			if testCase.name == "mismatched scorer version" {
				assertDirectMetricScorerVersionRejection(t, ctx, transaction, runID, runCaseID, fixture.corpusID)
			}
			if testCase.disableScorerVersionTrigger {
				setEvaluationLedgerTrigger(t, ctx, transaction, "evaluation_run_metric", "evaluation_run_metric_scorer_version_trigger", false)
			}
			insertDirectMetricLedger(t, ctx, transaction, runID, runCaseID, fixture.corpusID, testCase.mutateLedger(newDirectMetricLedger()))
			if testCase.disableScorerVersionTrigger {
				setEvaluationLedgerTrigger(t, ctx, transaction, "evaluation_run_metric", "evaluation_run_metric_scorer_version_trigger", true)
			}

			setEvaluationLedgerTrigger(t, ctx, transaction, "evaluation_run_case", "evaluation_run_case_metric_ledger_trigger", false)
			terminalizeDirectEvaluationCase(t, ctx, transaction, runCaseID, testCase.caseState)
			setEvaluationLedgerTrigger(t, ctx, transaction, "evaluation_run_case", "evaluation_run_case_metric_ledger_trigger", true)

			expectPostgresErrorCode(t, evaluationSchemaExecError(transaction.Exec(ctx, `
				UPDATE evaluation_run
				SET state = $2, completed_at = now()
				WHERE id = $1`, runID, testCase.runState,
			)), "23514")
		})
	}
}

type directMetricLedgerEntry struct {
	component     string
	scorerVersion string
	state         string
}

func newDirectMetricLedger() []directMetricLedgerEntry {
	components := []string{
		"retrieval_coverage",
		"citation_coverage",
		"citation_validity",
		"citation_scope_validity",
		"expected_abstention_outcome",
		"execution_outcome",
		"latency_milliseconds",
		"input_tokens",
		"output_tokens",
		"semantic_claim_support",
	}
	ledger := make([]directMetricLedgerEntry, 0, len(components))
	for _, component := range components {
		ledger = append(ledger, directMetricLedgerEntry{component: component, scorerVersion: "v1", state: "not_scored"})
	}
	return ledger
}

func assertDirectMetricScorerVersionRejection(
	t *testing.T,
	ctx context.Context,
	transaction pgx.Tx,
	runID, runCaseID, corpusID uuid.UUID,
) {
	t.Helper()
	savepoint, err := transaction.Begin(ctx)
	if err != nil {
		t.Fatalf("begin scorer version savepoint: %v", err)
	}
	insertDirectMetricLedgerExpectingError(t, ctx, savepoint, runID, runCaseID, corpusID, []directMetricLedgerEntry{{
		component: "retrieval_coverage", scorerVersion: "v2", state: "not_scored",
	}})
	if err := savepoint.Rollback(ctx); err != nil {
		t.Fatalf("rollback scorer version savepoint: %v", err)
	}
}

func insertDirectMetricLedgerExpectingError(
	t *testing.T,
	ctx context.Context,
	transaction pgx.Tx,
	runID, runCaseID, corpusID uuid.UUID,
	ledger []directMetricLedgerEntry,
) {
	t.Helper()
	metric := ledger[0]
	expectPostgresErrorCode(t, evaluationSchemaExecError(transaction.Exec(ctx, `
		INSERT INTO evaluation_run_metric (
			id, run_id, run_case_id, corpus_id, component, metric_state, value, numerator,
			denominator, rationale, scorer_version
		) VALUES ($1, $2, $3, $4, $5, $6, NULL, NULL, NULL, 'direct constraint fixture', $7)`,
		uuid.New(), runID, runCaseID, corpusID, metric.component, metric.state, metric.scorerVersion,
	)), "23514")
}

func insertDirectMetricLedger(
	t *testing.T,
	ctx context.Context,
	transaction pgx.Tx,
	runID, runCaseID, corpusID uuid.UUID,
	ledger []directMetricLedgerEntry,
) {
	t.Helper()
	for _, metric := range ledger {
		value, numerator, denominator := directMetricValues(metric.state)
		if _, err := transaction.Exec(ctx, `
			INSERT INTO evaluation_run_metric (
				id, run_id, run_case_id, corpus_id, component, metric_state, value, numerator,
				denominator, rationale, scorer_version
			) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, 'direct constraint fixture', $10)`,
			uuid.New(), runID, runCaseID, corpusID, metric.component, metric.state, value, numerator, denominator, metric.scorerVersion,
		); err != nil {
			t.Fatalf("insert direct metric ledger entry %q: %v", metric.component, err)
		}
	}
}

func directMetricValues(state string) (*float64, *int64, *int64) {
	if state != "scored" {
		return nil, nil, nil
	}
	value := float64(1)
	numerator, denominator := int64(1), int64(1)
	return &value, &numerator, &denominator
}

func setEvaluationLedgerTrigger(t *testing.T, ctx context.Context, transaction pgx.Tx, table, trigger string, enabled bool) {
	t.Helper()
	action := "DISABLE"
	if enabled {
		action = "ENABLE"
	}
	statement := "ALTER TABLE " + table + " " + action + " TRIGGER " + trigger
	if _, err := transaction.Exec(ctx, statement); err != nil {
		t.Fatalf("%s: %v", statement, err)
	}
}

func terminalizeDirectEvaluationCase(t *testing.T, ctx context.Context, transaction pgx.Tx, runCaseID uuid.UUID, state string) {
	t.Helper()
	if _, err := transaction.Exec(ctx, `
		UPDATE evaluation_run_case
		SET state = $2,
			lease_token = NULL,
			worker_id = NULL,
			lease_expires_at = NULL,
			answer = CASE WHEN $2 IN ('completed', 'abstained') THEN 'direct terminal fixture answer' ELSE NULL END,
			safe_failure_code = CASE WHEN $2 IN ('failed', 'cancelled') THEN 'direct_fixture_failure' ELSE NULL END,
			finished_at = now()
		WHERE id = $1`, runCaseID, state,
	); err != nil {
		t.Fatalf("terminalize direct evaluation case: %v", err)
	}
}

func evaluationComparisonRunRequest(
	fixture evaluationResolutionFixture,
	runID, firstCaseID, secondCaseID uuid.UUID,
	strategy, configurationFingerprint, chatModel string,
) evaluationpostgres.CreateRunRequest {
	return evaluationpostgres.CreateRunRequest{
		Identity: evaluationpostgres.RunIdentity{
			ID: runID, DatasetRevisionID: fixture.revisionID, CorpusID: fixture.corpusID, SnapshotID: fixture.snapshotID,
			SnapshotManifestSHA256: fixtureSHA256(fixture.snapshotID.String()),
			DatasetContentSHA256:   fixtureSHA256("dataset-" + fixture.revisionID.String()),
			OrderedCaseSetSHA256:   fixtureSHA256("comparison-case-set"),
			RetrievalStrategy:      strategy, RetrievalConfigurationFingerprint: configurationFingerprint, ScoringPolicyVersion: "v1",
			AgentBuild: "comparison-agent", ChatModelIdentity: chatModel, EmbeddingModelIdentity: "comparison-embedding", InitiatedBy: "integration-test",
		},
		Cases: []evaluationpostgres.RunCaseDefinition{
			{ID: uuid.New(), DatasetCaseID: firstCaseID, Position: 1, ExpectedOutcome: evaluationdomain.ExpectedOutcomeAnswer},
			{ID: uuid.New(), DatasetCaseID: secondCaseID, Position: 3, ExpectedOutcome: evaluationdomain.ExpectedOutcomeAnswer},
		},
	}
}

func completeComparisonRun(
	t *testing.T,
	ctx context.Context,
	repository *evaluationpostgres.Repository,
	runID, firstCaseID, secondCaseID uuid.UUID,
	left bool,
) {
	t.Helper()
	claims, err := repository.Claim(ctx, "comparison-worker-"+runID.String(), time.Minute, 2)
	if err != nil {
		t.Fatalf("Claim(%s) error = %v", runID, err)
	}
	if len(claims) != 2 {
		t.Fatalf("Claim(%s) = %#v, want two cases", runID, claims)
	}
	for _, claim := range claims {
		switch claim.DatasetCaseID {
		case firstCaseID:
			metrics := scorerV1CompletedMetrics(10, 1, 1)
			if !left {
				metrics[0].Value = float64Pointer(0)
				metrics[0].Numerator = 0
			}
			if err := repository.Complete(ctx, claim, evaluationapplication.TerminalCaseResult{
				State: evaluationapplication.RunCaseCompleted, Answer: "Synthetic comparison answer.", Metrics: metrics, ScoringPolicyVersion: "v1",
			}); err != nil {
				t.Fatalf("Complete(first case) error = %v", err)
			}
		case secondCaseID:
			if left {
				if err := repository.Complete(ctx, claim, evaluationapplication.TerminalCaseResult{
					State: evaluationapplication.RunCaseFailed, SafeFailureCode: "provider_unavailable",
				}); err != nil {
					t.Fatalf("Complete(failed second case) error = %v", err)
				}
				continue
			}
			if err := repository.Complete(ctx, claim, evaluationapplication.TerminalCaseResult{
				State: evaluationapplication.RunCaseCompleted, Answer: "Synthetic unpaired-score answer.", Metrics: scorerV1CompletedMetrics(10, 1, 1), ScoringPolicyVersion: "v1",
			}); err != nil {
				t.Fatalf("Complete(right second case) error = %v", err)
			}
		default:
			t.Fatalf("Claim(%s) returned unexpected dataset case %s", runID, claim.DatasetCaseID)
		}
	}
}

func integrationComparisonMetric(t *testing.T, comparison evaluationapplication.ComparisonResult, name evaluationapplication.MetricName) evaluationapplication.MetricDelta {
	t.Helper()
	for _, metric := range comparison.Metrics {
		if metric.Name == name {
			return metric
		}
	}
	t.Fatalf("Compare().Metrics = %#v, want %q", comparison.Metrics, name)
	return evaluationapplication.MetricDelta{}
}
