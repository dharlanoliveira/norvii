package postgres

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"sort"
	"strings"

	"github.com/dharlanoliveira/norvii/apps/api/internal/evaluation/application"
	"github.com/dharlanoliveira/norvii/apps/api/internal/evaluation/domain"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

// CreateEvaluationRun materializes every case and resolved evidence target for a compatibility
// plan. The requested snapshot remains the sole snapshot used for the new ledger.
func (repository *Repository) CreateEvaluationRun(
	ctx context.Context,
	request application.StartRunRequest,
	plan application.PreflightResult,
) (application.Run, error) {
	if repository == nil || repository.database == nil || request.CorpusID == uuid.Nil ||
		request.DatasetRevisionID == uuid.Nil || request.SnapshotID == uuid.Nil ||
		plan.CorpusID != request.CorpusID || plan.DatasetRevisionID != request.DatasetRevisionID || plan.SnapshotID != request.SnapshotID {
		return application.Run{}, ErrInvalidInput
	}
	identity, cases, err := repository.runIdentityAndCases(ctx, request)
	if err != nil {
		return application.Run{}, err
	}
	runCases := make([]RunCaseDefinition, 0, len(cases))
	caseIDs := make(map[uuid.UUID]uuid.UUID, len(cases))
	for _, item := range cases {
		runCaseID := uuid.New()
		caseIDs[item.datasetCaseID] = runCaseID
		runCases = append(runCases, RunCaseDefinition{
			ID: runCaseID, DatasetCaseID: item.datasetCaseID, Position: item.position, ExpectedOutcome: item.expectedOutcome,
		})
	}
	evidence, err := runExpectedEvidence(plan.ResolvedLocators, caseIDs)
	if err != nil {
		return application.Run{}, err
	}
	if err := repository.CreateRun(ctx, CreateRunRequest{Identity: identity, Cases: runCases, ExpectedEvidence: evidence}); err != nil {
		return application.Run{}, err
	}
	return repository.GetEvaluationRun(ctx, identity.ID)
}

type runCaseIdentity struct {
	datasetCaseID   uuid.UUID
	position        int
	expectedOutcome domain.ExpectedOutcome
	checksum        string
}

func (repository *Repository) runIdentityAndCases(ctx context.Context, request application.StartRunRequest) (RunIdentity, []runCaseIdentity, error) {
	var identity RunIdentity
	err := repository.database.QueryRow(ctx, `
		SELECT revision.content_sha256, snapshot.manifest_sha256
		FROM evaluation_dataset_revision AS revision
		JOIN corpus_snapshots AS snapshot ON snapshot.id = $3 AND snapshot.corpus_id = $2
		WHERE revision.id = $1 AND revision.corpus_id = $2`,
		request.DatasetRevisionID, request.CorpusID, request.SnapshotID,
	).Scan(&identity.DatasetContentSHA256, &identity.SnapshotManifestSHA256)
	if errors.Is(err, pgx.ErrNoRows) {
		return RunIdentity{}, nil, application.ErrRunNotFound
	}
	if err != nil {
		return RunIdentity{}, nil, fmt.Errorf("read evaluation run identity: %w", err)
	}
	rows, err := repository.database.Query(ctx, `
		SELECT id, position, expected_outcome, case_checksum
		FROM evaluation_dataset_case
		WHERE dataset_revision_id = $1 AND corpus_id = $2
		ORDER BY position ASC, id ASC`, request.DatasetRevisionID, request.CorpusID)
	if err != nil {
		return RunIdentity{}, nil, fmt.Errorf("read evaluation run cases: %w", err)
	}
	defer rows.Close()
	cases := make([]runCaseIdentity, 0)
	for rows.Next() {
		var item runCaseIdentity
		if err := rows.Scan(&item.datasetCaseID, &item.position, &item.expectedOutcome, &item.checksum); err != nil {
			return RunIdentity{}, nil, fmt.Errorf("scan evaluation run case: %w", err)
		}
		cases = append(cases, item)
	}
	if err := rows.Err(); err != nil {
		return RunIdentity{}, nil, fmt.Errorf("iterate evaluation run cases: %w", err)
	}
	if len(cases) == 0 {
		return RunIdentity{}, nil, ErrInvalidInput
	}
	identity.ID = uuid.New()
	identity.DatasetRevisionID = request.DatasetRevisionID
	identity.CorpusID = request.CorpusID
	identity.SnapshotID = request.SnapshotID
	identity.OrderedCaseSetSHA256 = orderedCaseSetSHA256(cases)
	identity.RetrievalStrategy = request.Configuration.Strategy
	identity.RetrievalConfigurationFingerprint = request.Configuration.Fingerprint
	identity.ScoringPolicyVersion = application.ScoringPolicyV1
	identity.AgentBuild = request.ExecutionIdentity.AgentBuild
	identity.ChatModelIdentity = request.ExecutionIdentity.ChatModelIdentity
	identity.EmbeddingModelIdentity = request.ExecutionIdentity.EmbeddingModelIdentity
	identity.InitiatedBy = "maintainer-http"
	return identity, cases, nil
}

func orderedCaseSetSHA256(cases []runCaseIdentity) string {
	hash := sha256.New()
	for _, item := range cases {
		_, _ = fmt.Fprintf(hash, "%d:%s:%s\n", item.position, item.datasetCaseID, item.checksum)
	}
	return hex.EncodeToString(hash.Sum(nil))
}

func runExpectedEvidence(resolved []application.ResolvedLocator, runCaseIDs map[uuid.UUID]uuid.UUID) ([]ExpectedEvidenceDefinition, error) {
	if len(resolved) == 0 {
		return nil, ErrInvalidInput
	}
	evidence := make([]ExpectedEvidenceDefinition, 0, len(resolved))
	ordinalByCase := make(map[uuid.UUID]int, len(runCaseIDs))
	seen := make(map[uuid.UUID]struct{}, len(resolved))
	for _, locator := range resolved {
		runCaseID, found := runCaseIDs[locator.CaseID]
		if !found || locator.ExpectedEvidenceID == uuid.Nil || locator.SourceID == uuid.Nil || locator.SourceRevisionID == uuid.Nil ||
			locator.DocumentID == uuid.Nil || locator.UnitID == uuid.Nil || strings.TrimSpace(locator.ContentSHA256) == "" {
			return nil, ErrInvalidInput
		}
		if _, duplicate := seen[locator.ExpectedEvidenceID]; duplicate {
			return nil, ErrInvalidInput
		}
		seen[locator.ExpectedEvidenceID] = struct{}{}
		ordinalByCase[runCaseID]++
		evidence = append(evidence, ExpectedEvidenceDefinition{
			ID: uuid.New(), RunCaseID: runCaseID, SourceID: locator.SourceID, SourceRevisionID: locator.SourceRevisionID,
			DocumentID: locator.DocumentID, LegalUnitID: locator.UnitID, CanonicalLocator: locator.CanonicalLocator,
			DisplayLocator: locator.DisplayLocator, ContentSHA256: locator.ContentSHA256, Ordinal: ordinalByCase[runCaseID],
		})
	}
	return evidence, nil
}

// GetEvaluationRun reads the saved identity rather than an active release and includes safe
// aggregate and case-status projections.
func (repository *Repository) GetEvaluationRun(ctx context.Context, runID uuid.UUID) (application.Run, error) {
	if repository == nil || repository.database == nil || runID == uuid.Nil {
		return application.Run{}, ErrInvalidInput
	}
	run, err := repository.readRun(ctx, runID)
	if err != nil {
		return application.Run{}, err
	}
	aggregate, err := repository.Aggregate(ctx, run.ID, run.CorpusID)
	if err != nil {
		return application.Run{}, err
	}
	run.Aggregate = application.RunAggregate{
		Total: aggregate.Total, Eligible: aggregate.Eligible, Scored: aggregate.Scored, Failed: aggregate.Failed,
		Cancelled: aggregate.Cancelled, NotApplicable: aggregate.NotApplicable, Metrics: aggregateMetrics(aggregate.Metrics),
	}
	cases, err := repository.readRunCaseSummaries(ctx, run.ID)
	if err != nil {
		return application.Run{}, err
	}
	run.Cases = cases
	return run, nil
}

func (repository *Repository) readRun(ctx context.Context, runID uuid.UUID) (application.Run, error) {
	var run application.Run
	var state string
	err := repository.database.QueryRow(ctx, `
		SELECT id, dataset_revision_id, dataset_content_sha256, corpus_id, snapshot_id, snapshot_manifest_sha256,
		       ordered_case_set_sha256, retrieval_strategy, retrieval_configuration_fingerprint, scoring_policy_version,
		       agent_build, chat_model_identity, embedding_model_identity, initiated_by, state, created_at, started_at, completed_at
		FROM evaluation_run WHERE id = $1`, runID,
	).Scan(&run.ID, &run.DatasetRevisionID, &run.DatasetContentSHA256, &run.CorpusID, &run.SnapshotID, &run.SnapshotManifestSHA256,
		&run.OrderedCaseSetSHA256, &run.Configuration.Strategy, &run.Configuration.Fingerprint, &run.ScoringPolicyVersion,
		&run.AgentBuild, &run.ChatModelIdentity, &run.EmbeddingModelIdentity, &run.InitiatedBy, &state, &run.CreatedAt, &run.StartedAt, &run.CompletedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return application.Run{}, application.ErrRunNotFound
	}
	if err != nil {
		return application.Run{}, fmt.Errorf("read evaluation run: %w", err)
	}
	run.State = application.RunState(state)
	return run, nil
}

func (repository *Repository) readRunCaseSummaries(ctx context.Context, runID uuid.UUID) ([]application.RunCaseSummary, error) {
	rows, err := repository.database.Query(ctx, `
		SELECT id, evaluation_case_id, position, state, attempt_count, finished_at, COALESCE(safe_failure_code, '')
		FROM evaluation_run_case WHERE run_id = $1 ORDER BY position ASC`, runID)
	if err != nil {
		return nil, fmt.Errorf("read evaluation run case summaries: %w", err)
	}
	defer rows.Close()
	summaries := make([]application.RunCaseSummary, 0)
	for rows.Next() {
		var summary application.RunCaseSummary
		if err := rows.Scan(&summary.ID, &summary.DatasetCaseID, &summary.Position, &summary.State, &summary.AttemptCount, &summary.FinishedAt, &summary.FailureCode); err != nil {
			return nil, fmt.Errorf("scan evaluation run case summary: %w", err)
		}
		summaries = append(summaries, summary)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate evaluation run case summaries: %w", err)
	}
	return summaries, nil
}

// GetEvaluationRunCase reads one immutable case ledger record; it never invokes execution.
func (repository *Repository) GetEvaluationRunCase(ctx context.Context, runID, runCaseID uuid.UUID) (application.RunCase, error) {
	if repository == nil || repository.database == nil || runID == uuid.Nil || runCaseID == uuid.Nil {
		return application.RunCase{}, ErrInvalidInput
	}
	var result application.RunCase
	err := repository.database.QueryRow(ctx, `
		SELECT run_case.id, run_case.evaluation_case_id, run_case.position, run_case.state, run_case.attempt_count,
		       run_case.finished_at, COALESCE(run_case.safe_failure_code, ''), run_case.run_id, run_case.corpus_id,
		       run.snapshot_id, run_case.dataset_revision_id, dataset_case.question, dataset_case.reference_answer,
		       run_case.expected_outcome, COALESCE(run_case.answer, ''), COALESCE(run_case.graph_grounding_state, ''),
		       run_case.latency_milliseconds, run_case.input_tokens, run_case.output_tokens
		FROM evaluation_run_case AS run_case
		JOIN evaluation_run AS run ON run.id = run_case.run_id AND run.corpus_id = run_case.corpus_id
		JOIN evaluation_dataset_case AS dataset_case ON dataset_case.id = run_case.evaluation_case_id
		 AND dataset_case.corpus_id = run_case.corpus_id AND dataset_case.dataset_revision_id = run_case.dataset_revision_id
		WHERE run_case.run_id = $1 AND run_case.id = $2`, runID, runCaseID,
	).Scan(&result.ID, &result.DatasetCaseID, &result.Position, &result.State, &result.AttemptCount, &result.FinishedAt,
		&result.FailureCode, &result.RunID, &result.CorpusID, &result.SnapshotID, &result.DatasetRevisionID, &result.Question,
		&result.ReferenceAnswer, &result.ExpectedOutcome, &result.Answer, &result.GraphGroundingState, &result.LatencyMilliseconds,
		&result.InputTokens, &result.OutputTokens)
	if errors.Is(err, pgx.ErrNoRows) {
		return application.RunCase{}, application.ErrRunNotFound
	}
	if err != nil {
		return application.RunCase{}, fmt.Errorf("read evaluation run case: %w", err)
	}
	var evidenceErr error
	if result.ExpectedEvidence, evidenceErr = repository.readExpectedEvidence(ctx, runID, runCaseID); evidenceErr != nil {
		return application.RunCase{}, evidenceErr
	}
	if result.ActualEvidence, evidenceErr = repository.readActualEvidence(ctx, runID, runCaseID); evidenceErr != nil {
		return application.RunCase{}, evidenceErr
	}
	metrics, err := repository.readCaseMetrics(ctx, runID, runCaseID)
	if err != nil {
		return application.RunCase{}, err
	}
	result.Metrics = metrics
	return result, nil
}

func (repository *Repository) readExpectedEvidence(ctx context.Context, runID, runCaseID uuid.UUID) ([]application.EvidenceIdentity, error) {
	rows, err := repository.database.Query(ctx, `
		SELECT ordinal, corpus_id, snapshot_id, source_id, source_revision_id, document_id, legal_unit_id,
		       canonical_locator, display_locator, content_sha256
		FROM evaluation_run_expected_evidence WHERE run_id = $1 AND run_case_id = $2 ORDER BY ordinal`, runID, runCaseID)
	if err != nil {
		return nil, fmt.Errorf("read expected evaluation evidence: %w", err)
	}
	defer rows.Close()
	items := make([]application.EvidenceIdentity, 0)
	for rows.Next() {
		item := application.EvidenceIdentity{Kind: "expected"}
		if err := rows.Scan(&item.Position, &item.CorpusID, &item.SnapshotID, &item.SourceID, &item.SourceRevisionID, &item.DocumentID,
			&item.LegalUnitID, &item.CanonicalLocator, &item.DisplayLocator, &item.ContentSHA256); err != nil {
			return nil, fmt.Errorf("scan expected evaluation evidence: %w", err)
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate expected evaluation evidence: %w", err)
	}
	return items, nil
}

func (repository *Repository) readActualEvidence(ctx context.Context, runID, runCaseID uuid.UUID) ([]application.EvidenceIdentity, error) {
	rows, err := repository.database.Query(ctx, `
		SELECT evidence_kind, position, marker_position, corpus_id, snapshot_id, source_id, source_revision_id,
		       document_id, legal_unit_id, canonical_locator, start_offset, end_offset, content_sha256
		FROM evaluation_run_actual_evidence WHERE run_id = $1 AND run_case_id = $2 ORDER BY evidence_kind, position`, runID, runCaseID)
	if err != nil {
		return nil, fmt.Errorf("read actual evaluation evidence: %w", err)
	}
	defer rows.Close()
	items := make([]application.EvidenceIdentity, 0)
	for rows.Next() {
		var item application.EvidenceIdentity
		if err := rows.Scan(&item.Kind, &item.Position, &item.MarkerPosition, &item.CorpusID, &item.SnapshotID, &item.SourceID,
			&item.SourceRevisionID, &item.DocumentID, &item.LegalUnitID, &item.CanonicalLocator, &item.StartOffset, &item.EndOffset,
			&item.ContentSHA256); err != nil {
			return nil, fmt.Errorf("scan actual evaluation evidence: %w", err)
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate actual evaluation evidence: %w", err)
	}
	return items, nil
}

func (repository *Repository) readCaseMetrics(ctx context.Context, runID, runCaseID uuid.UUID) ([]application.RunMetric, error) {
	rows, err := repository.database.Query(ctx, `
		SELECT component, metric_state, value, numerator, denominator, rationale, scorer_version
		FROM evaluation_run_metric WHERE run_id = $1 AND run_case_id = $2 ORDER BY component, scorer_version`, runID, runCaseID)
	if err != nil {
		return nil, fmt.Errorf("read evaluation case metrics: %w", err)
	}
	defer rows.Close()
	metrics := make([]application.RunMetric, 0)
	for rows.Next() {
		var metric application.RunMetric
		if err := rows.Scan(&metric.Name, &metric.State, &metric.Value, &metric.Numerator, &metric.Denominator, &metric.Rationale, &metric.ScorerVersion); err != nil {
			return nil, fmt.Errorf("scan evaluation case metric: %w", err)
		}
		metrics = append(metrics, metric)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate evaluation case metrics: %w", err)
	}
	return metrics, nil
}

func aggregateMetrics(metrics []RunAggregateMetric) []application.RunMetric {
	result := make([]application.RunMetric, 0, len(metrics))
	for _, metric := range metrics {
		result = append(result, application.RunMetric{Name: string(metric.Name), State: string(metric.State), Value: metric.Value,
			Numerator: metric.Numerator, Denominator: metric.Denominator, Rationale: metric.Rationale, ScorerVersion: metric.ScorerVersion})
	}
	sort.Slice(result, func(left, right int) bool { return result[left].Name < result[right].Name })
	return result
}

var _ application.RunStore = (*Repository)(nil)
