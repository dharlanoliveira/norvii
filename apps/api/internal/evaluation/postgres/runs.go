package postgres

import (
	"context"
	"errors"
	"fmt"
	"regexp"
	"slices"
	"strings"
	"time"

	"github.com/dharlanoliveira/norvii/apps/api/internal/evaluation/application"
	"github.com/dharlanoliveira/norvii/apps/api/internal/evaluation/domain"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

var runSHA256Pattern = regexp.MustCompile(`^[0-9a-f]{64}$`)

// RunIdentity is the immutable experiment identity persisted before any case is leased.
type RunIdentity struct {
	ID                                uuid.UUID
	DatasetRevisionID                 uuid.UUID
	CorpusID                          uuid.UUID
	SnapshotID                        uuid.UUID
	SnapshotManifestSHA256            string
	DatasetContentSHA256              string
	OrderedCaseSetSHA256              string
	RetrievalStrategy                 string
	RetrievalConfigurationFingerprint string
	ScoringPolicyVersion              string
	AgentBuild                        string
	ChatModelIdentity                 string
	EmbeddingModelIdentity            string
	InitiatedBy                       string
}

// RunCaseDefinition supplies the immutable case ledger entries created with a run.
type RunCaseDefinition struct {
	ID              uuid.UUID
	DatasetCaseID   uuid.UUID
	Position        int
	ExpectedOutcome domain.ExpectedOutcome
}

// ExpectedEvidenceDefinition is a preflight-resolved target copied into the immutable run ledger.
type ExpectedEvidenceDefinition struct {
	ID               uuid.UUID
	RunCaseID        uuid.UUID
	SourceID         uuid.UUID
	SourceRevisionID uuid.UUID
	DocumentID       uuid.UUID
	LegalUnitID      uuid.UUID
	CanonicalLocator string
	DisplayLocator   string
	ContentSHA256    string
	Ordinal          int
}

// CreateRunRequest contains only fixed identities and preflight-approved cases; it cannot select
// an active snapshot or infer cases at execution time.
type CreateRunRequest struct {
	Identity         RunIdentity
	Cases            []RunCaseDefinition
	ExpectedEvidence []ExpectedEvidenceDefinition
}

// RunAggregate makes every execution denominator explicit for later inspection and comparison.
type RunAggregate struct {
	Total         int64
	Eligible      int64
	Scored        int64
	Failed        int64
	Cancelled     int64
	NotApplicable int64
	Metrics       []RunAggregateMetric
}

// RunAggregateMetric is one immutable, component-specific metric persisted after every case in a
// run is terminal. MetricState makes its scoring eligibility explicit; numeric fields are nil for
// metrics that are not scoreable and therefore must never be represented as zero.
type RunAggregateMetric struct {
	Name          application.MetricName
	State         application.MetricState
	Value         *float64
	Numerator     *int64
	Denominator   *int64
	Rationale     string
	ScorerVersion string
}

// CreateRun atomically persists an immutable run, all case ledger records, and its preflight
// evidence plan. A malformed or foreign case prevents the entire run from being created.
func (repository *Repository) CreateRun(ctx context.Context, request CreateRunRequest) error {
	if repository == nil || repository.database == nil || request.validate() != nil {
		return ErrInvalidInput
	}
	transaction, err := repository.database.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin evaluation run creation: %w", err)
	}
	defer func() { _ = transaction.Rollback(ctx) }()

	identity := request.Identity
	if _, err := transaction.Exec(ctx, `
		INSERT INTO evaluation_run (
			id, dataset_revision_id, corpus_id, snapshot_id, snapshot_manifest_sha256,
			dataset_content_sha256, ordered_case_set_sha256, retrieval_strategy,
			retrieval_configuration_fingerprint, scoring_policy_version, agent_build,
			chat_model_identity, embedding_model_identity, initiated_by
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14)`,
		identity.ID, identity.DatasetRevisionID, identity.CorpusID, identity.SnapshotID, identity.SnapshotManifestSHA256,
		identity.DatasetContentSHA256, identity.OrderedCaseSetSHA256, identity.RetrievalStrategy,
		identity.RetrievalConfigurationFingerprint, identity.ScoringPolicyVersion, identity.AgentBuild,
		identity.ChatModelIdentity, identity.EmbeddingModelIdentity, identity.InitiatedBy,
	); err != nil {
		return fmt.Errorf("insert evaluation run: %w", err)
	}
	for _, evaluationCase := range request.Cases {
		command, err := transaction.Exec(ctx, `
			INSERT INTO evaluation_run_case (
				id, run_id, corpus_id, dataset_revision_id, evaluation_case_id, position, expected_outcome
			)
			SELECT $1, $2, $3, $4, evaluation_case.id, $5, $6
			FROM evaluation_dataset_case AS evaluation_case
			WHERE evaluation_case.id = $7
			  AND evaluation_case.corpus_id = $3
			  AND evaluation_case.dataset_revision_id = $4
			  AND evaluation_case.expected_outcome = $6`,
			evaluationCase.ID, identity.ID, identity.CorpusID, identity.DatasetRevisionID, evaluationCase.Position,
			string(evaluationCase.ExpectedOutcome), evaluationCase.DatasetCaseID,
		)
		if err != nil {
			return fmt.Errorf("insert evaluation run case: %w", err)
		}
		if command.RowsAffected() != 1 {
			return ErrInvalidInput
		}
	}
	for _, evidence := range request.ExpectedEvidence {
		command, err := transaction.Exec(ctx, `
			INSERT INTO evaluation_run_expected_evidence (
				id, run_id, run_case_id, corpus_id, snapshot_id, source_id, source_revision_id,
				document_id, legal_unit_id, canonical_locator, display_locator, content_sha256, ordinal
			)
			SELECT $1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13
			WHERE EXISTS (
				SELECT 1
				FROM corpus_snapshot_documents AS member
				JOIN document_units AS unit
				  ON unit.document_id = member.document_id
				 AND unit.id = $9
				 AND unit.canonical_locator = $10
				WHERE member.snapshot_id = $5
				  AND member.corpus_id = $4
				  AND member.source_id = $6
				  AND member.source_revision_id = $7
				  AND member.document_id = $8
				  AND member.content_sha256 = $12
			)`,
			evidence.ID, identity.ID, evidence.RunCaseID, identity.CorpusID, identity.SnapshotID,
			evidence.SourceID, evidence.SourceRevisionID, evidence.DocumentID, evidence.LegalUnitID,
			evidence.CanonicalLocator, evidence.DisplayLocator, evidence.ContentSHA256, evidence.Ordinal,
		)
		if err != nil {
			return fmt.Errorf("insert evaluation run expected evidence: %w", err)
		}
		if command.RowsAffected() != 1 {
			return ErrInvalidInput
		}
	}
	if err := transaction.Commit(ctx); err != nil {
		return fmt.Errorf("commit evaluation run creation: %w", err)
	}
	return nil
}

// Claim safely recovers expired leases and claims at most batchSize independent cases. The case
// update is guarded by its state and occurs in the same transaction as the run transition.
func (repository *Repository) Claim(ctx context.Context, workerID string, leasePeriod time.Duration, batchSize int) ([]application.ClaimedCase, error) {
	if repository == nil || repository.database == nil || strings.TrimSpace(workerID) == "" || leasePeriod <= 0 || batchSize < 1 {
		return nil, ErrInvalidInput
	}
	transaction, err := repository.database.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("begin evaluation case claim: %w", err)
	}
	defer func() { _ = transaction.Rollback(ctx) }()
	rows, err := transaction.Query(ctx, `
		SELECT run_case.id, run_case.run_id, run_case.corpus_id, run.snapshot_id,
		       run_case.evaluation_case_id, dataset_case.question, dataset_case.query_language,
		       run_case.expected_outcome, run_case.attempt_count, run.retrieval_strategy,
		       run.retrieval_configuration_fingerprint, run.agent_build, run.chat_model_identity,
		       run.embedding_model_identity
		FROM evaluation_run_case AS run_case
		JOIN evaluation_run AS run ON run.id = run_case.run_id AND run.corpus_id = run_case.corpus_id
		JOIN evaluation_dataset_case AS dataset_case
		  ON dataset_case.id = run_case.evaluation_case_id
		 AND dataset_case.corpus_id = run_case.corpus_id
		 AND dataset_case.dataset_revision_id = run_case.dataset_revision_id
		WHERE (run_case.state = 'pending' OR (run_case.state = 'leased' AND run_case.lease_expires_at <= now()))
		  AND run.state IN ('queued', 'running')
		ORDER BY run_case.created_at, run_case.id
		FOR UPDATE OF run_case SKIP LOCKED
		LIMIT $1`, batchSize)
	if err != nil {
		return nil, fmt.Errorf("select evaluation cases for claim: %w", err)
	}
	defer rows.Close()
	claims := make([]application.ClaimedCase, 0, batchSize)
	for rows.Next() {
		var claimed application.ClaimedCase
		if err := rows.Scan(&claimed.ID, &claimed.RunID, &claimed.CorpusID, &claimed.SnapshotID, &claimed.DatasetCaseID,
			&claimed.Question, &claimed.QueryLanguage, &claimed.ExpectedOutcome, &claimed.AttemptCount,
			&claimed.Configuration.Strategy, &claimed.Configuration.Fingerprint, &claimed.ExecutionIdentity.AgentBuild,
			&claimed.ExecutionIdentity.ChatModelIdentity, &claimed.ExecutionIdentity.EmbeddingModelIdentity); err != nil {
			return nil, fmt.Errorf("scan evaluation case claim: %w", err)
		}
		claims = append(claims, claimed)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate evaluation case claims: %w", err)
	}
	rows.Close()
	for index := range claims {
		claim := &claims[index]
		claim.LeaseToken = uuid.New()
		command, err := transaction.Exec(ctx, `
			UPDATE evaluation_run
			SET state = 'running', started_at = now()
			WHERE id = $1 AND corpus_id = $2 AND state = 'queued'`, claim.RunID, claim.CorpusID)
		if err != nil {
			return nil, fmt.Errorf("start evaluation run: %w", err)
		}
		_ = command
		err = transaction.QueryRow(ctx, `
			UPDATE evaluation_run_case
			SET state = 'leased', attempt_count = attempt_count + 1, lease_token = $2,
				worker_id = $3, lease_expires_at = now() + $4::interval, started_at = COALESCE(started_at, now())
			WHERE id = $1 AND state IN ('pending', 'leased')
			RETURNING attempt_count, lease_expires_at`, claim.ID, claim.LeaseToken, workerID, postgresInterval(leasePeriod),
		).Scan(&claim.AttemptCount, &claim.LeaseExpiresAt)
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, application.ErrLeaseLost
		}
		if err != nil {
			return nil, fmt.Errorf("lease evaluation case: %w", err)
		}
	}
	if err := transaction.Commit(ctx); err != nil {
		return nil, fmt.Errorf("commit evaluation case claim: %w", err)
	}
	return slices.Clone(claims), nil
}

// Complete records a terminal result only while the supplied lease is current. A second completion
// cannot overwrite the first terminal row, even after a worker restart.
func (repository *Repository) Complete(ctx context.Context, claim application.ClaimedCase, result application.TerminalCaseResult) error {
	if repository == nil || repository.database == nil || validateClaim(claim) != nil || validateTerminalResult(result) != nil {
		return ErrInvalidInput
	}
	transaction, err := repository.database.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin evaluation case completion: %w", err)
	}
	defer func() { _ = transaction.Rollback(ctx) }()
	var runID, corpusID uuid.UUID
	err = transaction.QueryRow(ctx, `
		SELECT run_id, corpus_id
		FROM evaluation_run_case
		WHERE id = $1 AND lease_token = $2 AND state = 'leased'
		FOR UPDATE`, claim.ID, claim.LeaseToken,
	).Scan(&runID, &corpusID)
	if errors.Is(err, pgx.ErrNoRows) {
		return application.ErrLeaseLost
	}
	if err != nil {
		return fmt.Errorf("lock evaluation case completion: %w", err)
	}
	if err := persistActualEvidence(ctx, transaction, claim, result.ActualEvidence); err != nil {
		return err
	}
	if err := persistTerminalCaseMetrics(ctx, transaction, claim, result); err != nil {
		return err
	}
	err = transaction.QueryRow(ctx, `
		UPDATE evaluation_run_case
		SET state = $3, lease_token = NULL, worker_id = NULL, lease_expires_at = NULL,
			answer = $4, graph_grounding_state = NULLIF($5, ''), safe_failure_code = NULLIF($6, ''),
			latency_milliseconds = $7, input_tokens = $8, output_tokens = $9, finished_at = now()
		WHERE id = $1 AND lease_token = $2 AND state = 'leased'
		RETURNING run_id, corpus_id`,
		claim.ID, claim.LeaseToken, string(result.State), result.Answer, result.GraphGroundingState,
		result.SafeFailureCode, result.LatencyMilliseconds, result.InputTokens, result.OutputTokens,
	).Scan(&runID, &corpusID)
	if errors.Is(err, pgx.ErrNoRows) {
		return application.ErrLeaseLost
	}
	if err != nil {
		return fmt.Errorf("persist evaluation terminal result: %w", err)
	}
	if err := finalizeRunIfTerminal(ctx, transaction, runID, corpusID); err != nil {
		return err
	}
	if err := transaction.Commit(ctx); err != nil {
		return fmt.Errorf("commit evaluation case completion: %w", err)
	}
	return nil
}

// ReleaseForRetry clears only a current non-terminal lease. Reaching maxAttempts records a safe
// failed terminal result instead of allowing an unbounded retry loop.
func (repository *Repository) ReleaseForRetry(ctx context.Context, claim application.ClaimedCase, failureCode string, maxAttempts int) error {
	if repository == nil || repository.database == nil || validateClaim(claim) != nil || strings.TrimSpace(failureCode) == "" || maxAttempts < 1 {
		return ErrInvalidInput
	}
	transaction, err := repository.database.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin evaluation retry release: %w", err)
	}
	defer func() { _ = transaction.Rollback(ctx) }()
	var runID, corpusID uuid.UUID
	var terminal bool
	err = transaction.QueryRow(ctx, `
		SELECT run_id, corpus_id, attempt_count >= $3
		FROM evaluation_run_case
		WHERE id = $1 AND lease_token = $2 AND state = 'leased'
		FOR UPDATE`, claim.ID, claim.LeaseToken, maxAttempts,
	).Scan(&runID, &corpusID, &terminal)
	if errors.Is(err, pgx.ErrNoRows) {
		return application.ErrLeaseLost
	}
	if err != nil {
		return fmt.Errorf("lock evaluation case retry release: %w", err)
	}
	if terminal {
		if err := persistUnscoredTerminalMetrics(ctx, transaction, claim); err != nil {
			return err
		}
	}
	err = transaction.QueryRow(ctx, `
		UPDATE evaluation_run_case
		SET state = CASE WHEN attempt_count >= $3 THEN 'failed' ELSE 'pending' END,
			lease_token = NULL, worker_id = NULL, lease_expires_at = NULL,
			safe_failure_code = CASE WHEN attempt_count >= $3 THEN $4 ELSE NULL END,
			finished_at = CASE WHEN attempt_count >= $3 THEN now() ELSE NULL END
		WHERE id = $1 AND lease_token = $2 AND state = 'leased'
		RETURNING run_id, corpus_id`, claim.ID, claim.LeaseToken, maxAttempts, failureCode,
	).Scan(&runID, &corpusID)
	if errors.Is(err, pgx.ErrNoRows) {
		return application.ErrLeaseLost
	}
	if err != nil {
		return fmt.Errorf("release evaluation case for retry: %w", err)
	}
	if err := finalizeRunIfTerminal(ctx, transaction, runID, corpusID); err != nil {
		return err
	}
	if err := transaction.Commit(ctx); err != nil {
		return fmt.Errorf("commit evaluation retry release: %w", err)
	}
	return nil
}

// Aggregate returns explicit terminal case denominators and immutable run-level metric rows. It
// never derives a numeric score from failed or cancelled cases.
func (repository *Repository) Aggregate(ctx context.Context, runID, corpusID uuid.UUID) (RunAggregate, error) {
	if repository == nil || repository.database == nil || runID == uuid.Nil || corpusID == uuid.Nil {
		return RunAggregate{}, ErrInvalidInput
	}
	var aggregate RunAggregate
	err := repository.database.QueryRow(ctx, `
		SELECT count(DISTINCT run_case.id) AS total,
		       count(DISTINCT run_case.id) FILTER (WHERE state IN ('completed', 'abstained')) AS eligible,
		       count(DISTINCT metric.run_case_id) FILTER (WHERE metric.metric_state = 'scored') AS scored,
		       count(DISTINCT run_case.id) FILTER (WHERE state = 'failed') AS failed,
		       count(DISTINCT run_case.id) FILTER (WHERE state = 'cancelled') AS cancelled,
		       count(DISTINCT metric.run_case_id) FILTER (WHERE metric.metric_state = 'not_applicable') AS not_applicable
		FROM evaluation_run_case AS run_case
		LEFT JOIN evaluation_run_metric AS metric
		  ON metric.run_case_id = run_case.id
		 AND metric.run_id = run_case.run_id
		 AND metric.corpus_id = run_case.corpus_id
		WHERE run_case.run_id = $1
		  AND run_case.corpus_id = $2
		  AND run_case.state IN ('completed', 'abstained', 'failed', 'cancelled')`, runID, corpusID,
	).Scan(&aggregate.Total, &aggregate.Eligible, &aggregate.Scored, &aggregate.Failed, &aggregate.Cancelled, &aggregate.NotApplicable)
	if errors.Is(err, pgx.ErrNoRows) {
		return RunAggregate{}, ErrInvalidInput
	}
	if err != nil {
		return RunAggregate{}, fmt.Errorf("aggregate evaluation run: %w", err)
	}
	rows, err := repository.database.Query(ctx, `
		SELECT component, metric_state, value, numerator, denominator, rationale, scorer_version
		FROM evaluation_run_metric
		WHERE run_id = $1 AND corpus_id = $2 AND run_case_id IS NULL
		ORDER BY component, scorer_version`, runID, corpusID)
	if err != nil {
		return RunAggregate{}, fmt.Errorf("read evaluation run aggregate metrics: %w", err)
	}
	defer rows.Close()
	for rows.Next() {
		var metric RunAggregateMetric
		if err := rows.Scan(&metric.Name, &metric.State, &metric.Value, &metric.Numerator, &metric.Denominator,
			&metric.Rationale, &metric.ScorerVersion); err != nil {
			return RunAggregate{}, fmt.Errorf("scan evaluation run aggregate metric: %w", err)
		}
		aggregate.Metrics = append(aggregate.Metrics, metric)
	}
	if err := rows.Err(); err != nil {
		return RunAggregate{}, fmt.Errorf("iterate evaluation run aggregate metrics: %w", err)
	}
	return aggregate, nil
}

func finalizeRunIfTerminal(ctx context.Context, transaction pgx.Tx, runID, corpusID uuid.UUID) error {
	var open, completed, abstained, failed, cancelled int64
	if err := transaction.QueryRow(ctx, `
		SELECT count(*) FILTER (WHERE state IN ('pending', 'leased')),
		       count(*) FILTER (WHERE state = 'completed'), count(*) FILTER (WHERE state = 'abstained'),
		       count(*) FILTER (WHERE state = 'failed'), count(*) FILTER (WHERE state = 'cancelled')
		FROM evaluation_run_case
		WHERE run_id = $1 AND corpus_id = $2`, runID, corpusID,
	).Scan(&open, &completed, &abstained, &failed, &cancelled); err != nil {
		return fmt.Errorf("read evaluation run terminal state: %w", err)
	}
	if open != 0 {
		return nil
	}
	if err := persistRunAggregateMetrics(ctx, transaction, runID, corpusID); err != nil {
		return err
	}
	state := "completed"
	if completed+abstained == 0 {
		state = "failed"
	} else if failed+cancelled > 0 {
		state = "completed_with_failures"
	}
	if _, err := transaction.Exec(ctx, `
		UPDATE evaluation_run SET state = $3, completed_at = now()
		WHERE id = $1 AND corpus_id = $2 AND state = 'running'`, runID, corpusID, state); err != nil {
		return fmt.Errorf("finalize evaluation run: %w", err)
	}
	return nil
}

func (request CreateRunRequest) validate() error {
	identity := request.Identity
	if identity.ID == uuid.Nil || identity.DatasetRevisionID == uuid.Nil || identity.CorpusID == uuid.Nil || identity.SnapshotID == uuid.Nil ||
		!runSHA256Pattern.MatchString(identity.SnapshotManifestSHA256) || !runSHA256Pattern.MatchString(identity.DatasetContentSHA256) ||
		!runSHA256Pattern.MatchString(identity.OrderedCaseSetSHA256) || !allNonBlank(identity.RetrievalStrategy, identity.RetrievalConfigurationFingerprint,
		identity.ScoringPolicyVersion, identity.AgentBuild, identity.ChatModelIdentity, identity.EmbeddingModelIdentity, identity.InitiatedBy) || len(request.Cases) == 0 {
		return ErrInvalidInput
	}
	caseIDs := make(map[uuid.UUID]struct{}, len(request.Cases))
	positions := make(map[int]struct{}, len(request.Cases))
	runCaseIDs := make(map[uuid.UUID]struct{}, len(request.Cases))
	for _, evaluationCase := range request.Cases {
		if evaluationCase.ID == uuid.Nil || evaluationCase.DatasetCaseID == uuid.Nil || evaluationCase.Position < 1 ||
			(evaluationCase.ExpectedOutcome != domain.ExpectedOutcomeAnswer && evaluationCase.ExpectedOutcome != domain.ExpectedOutcomeAbstain) {
			return ErrInvalidInput
		}
		if _, found := caseIDs[evaluationCase.DatasetCaseID]; found {
			return ErrInvalidInput
		}
		if _, found := positions[evaluationCase.Position]; found {
			return ErrInvalidInput
		}
		caseIDs[evaluationCase.DatasetCaseID] = struct{}{}
		positions[evaluationCase.Position] = struct{}{}
		runCaseIDs[evaluationCase.ID] = struct{}{}
	}
	for _, evidence := range request.ExpectedEvidence {
		if evidence.ID == uuid.Nil || evidence.RunCaseID == uuid.Nil || evidence.SourceID == uuid.Nil || evidence.SourceRevisionID == uuid.Nil ||
			evidence.DocumentID == uuid.Nil || evidence.LegalUnitID == uuid.Nil || evidence.Ordinal < 1 ||
			!canonicalLocatorPattern.MatchString(evidence.CanonicalLocator) || !runSHA256Pattern.MatchString(evidence.ContentSHA256) || !allNonBlank(evidence.DisplayLocator) {
			return ErrInvalidInput
		}
		if _, found := runCaseIDs[evidence.RunCaseID]; !found {
			return ErrInvalidInput
		}
	}
	return nil
}

func validateClaim(claim application.ClaimedCase) error {
	if claim.ID == uuid.Nil || claim.RunID == uuid.Nil || claim.CorpusID == uuid.Nil || claim.SnapshotID == uuid.Nil ||
		claim.DatasetCaseID == uuid.Nil || claim.LeaseToken == uuid.Nil || claim.AttemptCount < 1 || claim.LeaseExpiresAt.IsZero() ||
		strings.TrimSpace(claim.Question) == "" || strings.TrimSpace(claim.Configuration.Strategy) == "" ||
		strings.TrimSpace(claim.Configuration.Fingerprint) == "" || strings.TrimSpace(claim.ExecutionIdentity.AgentBuild) == "" ||
		strings.TrimSpace(claim.ExecutionIdentity.ChatModelIdentity) == "" || strings.TrimSpace(claim.ExecutionIdentity.EmbeddingModelIdentity) == "" {
		return ErrInvalidInput
	}
	return nil
}

func validateTerminalResult(result application.TerminalCaseResult) error {
	if result.State != application.RunCaseCompleted && result.State != application.RunCaseAbstained &&
		result.State != application.RunCaseFailed && result.State != application.RunCaseCancelled {
		return ErrInvalidInput
	}
	if result.State == application.RunCaseCompleted && strings.TrimSpace(result.Answer) == "" {
		return ErrInvalidInput
	}
	if (result.State == application.RunCaseFailed || result.State == application.RunCaseCancelled) != (strings.TrimSpace(result.SafeFailureCode) != "") {
		return ErrInvalidInput
	}
	for _, measurement := range []*int64{result.LatencyMilliseconds, result.InputTokens, result.OutputTokens} {
		if measurement != nil && *measurement < 0 {
			return ErrInvalidInput
		}
	}
	if len(result.Metrics) > 0 && strings.TrimSpace(result.ScoringPolicyVersion) == "" {
		return ErrInvalidInput
	}
	for _, metric := range result.Metrics {
		if err := validateMetric(metric); err != nil {
			return err
		}
	}
	return nil
}

func persistActualEvidence(ctx context.Context, transaction pgx.Tx, claim application.ClaimedCase, evidence []application.ActualEvidence) error {
	nextPosition := map[application.EvidenceKind]int{application.EvidenceKindRetrieved: 1, application.EvidenceKindCited: 1}
	for _, item := range evidence {
		if item.Kind != application.EvidenceKindRetrieved && item.Kind != application.EvidenceKindCited ||
			item.Position != nextPosition[item.Kind] || (item.Kind == application.EvidenceKindRetrieved && item.MarkerPosition != 0) ||
			(item.Kind == application.EvidenceKindCited && item.MarkerPosition < 1) ||
			item.Provenance.CorpusID != claim.CorpusID || item.Provenance.SnapshotID != claim.SnapshotID ||
			item.Provenance.SourceID == uuid.Nil || item.Provenance.SourceRevisionID == uuid.Nil || item.Provenance.DocumentID == uuid.Nil ||
			item.Provenance.UnitID == uuid.Nil || item.Provenance.StartOffset < 0 || item.Provenance.EndOffset <= item.Provenance.StartOffset ||
			!canonicalLocatorPattern.MatchString(item.Provenance.CanonicalLocator) || item.Provenance.ContentSHA256.Validate() != nil {
			return ErrInvalidInput
		}
		nextPosition[item.Kind]++
		command, err := transaction.Exec(ctx, `
			INSERT INTO evaluation_run_actual_evidence (
				id, run_id, run_case_id, corpus_id, snapshot_id, evidence_kind, position, marker_position,
				source_id, source_revision_id, document_id, legal_unit_id, canonical_locator,
				start_offset, end_offset, content_sha256
			)
			SELECT $1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16
			WHERE EXISTS (
				SELECT 1
				FROM corpus_snapshot_documents AS member
				JOIN document_units AS unit
				  ON unit.document_id = member.document_id
				 AND unit.id = $12
				 AND unit.canonical_locator = $13
				WHERE member.snapshot_id = $5
				  AND member.corpus_id = $4
				  AND member.source_id = $9
				  AND member.source_revision_id = $10
				  AND member.document_id = $11
				  AND member.content_sha256 = $16
			)`,
			uuid.New(), claim.RunID, claim.ID, claim.CorpusID, claim.SnapshotID, string(item.Kind), item.Position, item.MarkerPosition,
			item.Provenance.SourceID, item.Provenance.SourceRevisionID, item.Provenance.DocumentID, item.Provenance.UnitID,
			item.Provenance.CanonicalLocator, item.Provenance.StartOffset, item.Provenance.EndOffset, string(item.Provenance.ContentSHA256),
		)
		if err != nil {
			return fmt.Errorf("persist evaluation actual evidence: %w", err)
		}
		if command.RowsAffected() != 1 {
			return ErrInvalidInput
		}
	}
	return nil
}

func persistTerminalCaseMetrics(ctx context.Context, transaction pgx.Tx, claim application.ClaimedCase, result application.TerminalCaseResult) error {
	if result.State == application.RunCaseFailed || result.State == application.RunCaseCancelled {
		return persistUnscoredTerminalMetrics(ctx, transaction, claim)
	}

	if len(result.Metrics) > 0 {
		policyVersion, err := scoringPolicyVersion(ctx, transaction, claim.RunID, claim.CorpusID)
		if err != nil {
			return err
		}
		if result.ScoringPolicyVersion != policyVersion {
			return ErrInvalidInput
		}
	}
	if err := validateRequiredTerminalMetricSet(result.Metrics); err != nil {
		return err
	}
	return persistCaseMetrics(ctx, transaction, claim, result.Metrics, result.ScoringPolicyVersion)
}

func persistCaseMetrics(
	ctx context.Context,
	transaction pgx.Tx,
	claim application.ClaimedCase,
	metrics []application.Metric,
	scoringPolicyVersion string,
) error {
	for _, metric := range metrics {
		value, numerator, denominator := metricPersistenceValues(metric)
		if _, err := transaction.Exec(ctx, `
			INSERT INTO evaluation_run_metric (
				id, run_id, run_case_id, corpus_id, component, metric_state, value, numerator,
				denominator, rationale, scorer_version
			) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)`,
			uuid.New(), claim.RunID, claim.ID, claim.CorpusID, string(metric.Name), string(metric.State), value,
			numerator, denominator, metric.Rationale, scoringPolicyVersion,
		); err != nil {
			return fmt.Errorf("persist evaluation case metric: %w", err)
		}
	}
	return nil
}

// metricPersistenceValues maps non-scored domain metrics to the nullable storage contract.
func metricPersistenceValues(metric application.Metric) (*float64, *int64, *int64) {
	if metric.State != application.MetricStateScored {
		return nil, nil, nil
	}
	return metric.Value, &metric.Numerator, &metric.Denominator
}

func persistUnscoredTerminalMetrics(
	ctx context.Context,
	transaction pgx.Tx,
	claim application.ClaimedCase,
) error {
	policyVersion, err := scoringPolicyVersion(ctx, transaction, claim.RunID, claim.CorpusID)
	if err != nil {
		return err
	}
	metrics := make([]application.Metric, 0, len(terminalMetricNames))
	for _, name := range terminalMetricNames {
		metrics = append(metrics, application.Metric{
			Name: name, State: application.MetricStateNotScored,
			Rationale: "terminal failure or cancellation is not assigned a synthetic score",
		})
	}
	return persistCaseMetrics(ctx, transaction, claim, metrics, policyVersion)
}

func scoringPolicyVersion(ctx context.Context, transaction pgx.Tx, runID, corpusID uuid.UUID) (string, error) {
	var version string
	if err := transaction.QueryRow(ctx, `
		SELECT scoring_policy_version
		FROM evaluation_run
		WHERE id = $1 AND corpus_id = $2`, runID, corpusID,
	).Scan(&version); err != nil {
		return "", fmt.Errorf("read evaluation scoring policy version: %w", err)
	}
	return version, nil
}

var terminalMetricNames = []application.MetricName{
	application.MetricRetrievalCoverage,
	application.MetricCitationCoverage,
	application.MetricCitationValidity,
	application.MetricCitationScope,
	application.MetricExpectedAbstention,
	application.MetricExecutionState,
	application.MetricLatency,
	application.MetricInputTokens,
	application.MetricOutputTokens,
	application.MetricSemanticSupport,
}

func validateRequiredTerminalMetricSet(metrics []application.Metric) error {
	if len(metrics) != len(terminalMetricNames) {
		return ErrInvalidInput
	}
	required := make(map[application.MetricName]struct{}, len(terminalMetricNames))
	for _, name := range terminalMetricNames {
		required[name] = struct{}{}
	}
	for _, metric := range metrics {
		if _, found := required[metric.Name]; !found {
			return ErrInvalidInput
		}
		delete(required, metric.Name)
	}
	if len(required) != 0 {
		return ErrInvalidInput
	}
	return nil
}

type metricAggregateCounts struct {
	name             application.MetricName
	scorerVersion    string
	scored           int64
	numerator        int64
	denominator      int64
	notApplicable    int64
	notScored        int64
	needsHumanReview int64
}

func persistRunAggregateMetrics(ctx context.Context, transaction pgx.Tx, runID, corpusID uuid.UUID) error {
	rows, err := transaction.Query(ctx, `
		SELECT metric.component,
		       metric.scorer_version,
		       count(*) FILTER (WHERE metric.metric_state = 'scored') AS scored,
		       COALESCE(sum(metric.numerator) FILTER (WHERE metric.metric_state = 'scored'), 0) AS numerator,
		       COALESCE(sum(metric.denominator) FILTER (WHERE metric.metric_state = 'scored'), 0) AS denominator,
		       count(*) FILTER (WHERE metric.metric_state = 'not_applicable') AS not_applicable,
		       count(*) FILTER (WHERE metric.metric_state = 'not_scored') AS not_scored,
		       count(*) FILTER (WHERE metric.metric_state = 'needs_human_review') AS needs_human_review
		FROM evaluation_run_metric AS metric
		JOIN evaluation_run_case AS run_case
		  ON run_case.id = metric.run_case_id
		 AND run_case.run_id = metric.run_id
		 AND run_case.corpus_id = metric.corpus_id
		WHERE metric.run_id = $1
		  AND metric.corpus_id = $2
		  AND metric.run_case_id IS NOT NULL
		  AND run_case.state IN ('completed', 'abstained', 'failed', 'cancelled')
		GROUP BY metric.component, metric.scorer_version
		ORDER BY metric.component, metric.scorer_version`, runID, corpusID)
	if err != nil {
		return fmt.Errorf("read terminal evaluation metrics for aggregate: %w", err)
	}
	defer rows.Close()
	countsByMetric := make([]metricAggregateCounts, 0)
	for rows.Next() {
		var counts metricAggregateCounts
		if err := rows.Scan(&counts.name, &counts.scorerVersion, &counts.scored, &counts.numerator,
			&counts.denominator, &counts.notApplicable, &counts.notScored, &counts.needsHumanReview); err != nil {
			return fmt.Errorf("scan terminal evaluation metric aggregate: %w", err)
		}
		countsByMetric = append(countsByMetric, counts)
	}
	if err := rows.Err(); err != nil {
		return fmt.Errorf("iterate terminal evaluation metrics for aggregate: %w", err)
	}
	rows.Close()
	for _, counts := range countsByMetric {
		if err := persistRunAggregateMetric(ctx, transaction, runID, corpusID, counts); err != nil {
			return err
		}
	}
	return nil
}

func persistRunAggregateMetric(
	ctx context.Context,
	transaction pgx.Tx,
	runID, corpusID uuid.UUID,
	counts metricAggregateCounts,
) error {
	state, value, numerator, denominator := aggregateMetricValue(counts)
	rationale := fmt.Sprintf(
		"aggregated from %d scored, %d not-applicable, %d not-scored, and %d needs-human-review terminal cases",
		counts.scored, counts.notApplicable, counts.notScored, counts.needsHumanReview,
	)
	if _, err := transaction.Exec(ctx, `
		INSERT INTO evaluation_run_metric (
			id, run_id, run_case_id, corpus_id, component, metric_state, value, numerator,
			denominator, rationale, scorer_version
		) VALUES ($1, $2, NULL, $3, $4, $5, $6, $7, $8, $9, $10)`,
		uuid.New(), runID, corpusID, string(counts.name), string(state), value, numerator, denominator,
		rationale, counts.scorerVersion,
	); err != nil {
		return fmt.Errorf("persist evaluation run aggregate metric: %w", err)
	}
	return nil
}

func aggregateMetricValue(counts metricAggregateCounts) (application.MetricState, *float64, *int64, *int64) {
	if counts.scored > 0 {
		value := float64(counts.numerator) / float64(counts.denominator)
		return application.MetricStateScored, &value, &counts.numerator, &counts.denominator
	}
	if counts.notScored > 0 {
		return application.MetricStateNotScored, nil, nil, nil
	}
	if counts.needsHumanReview > 0 {
		return application.MetricStateNeedsHumanReview, nil, nil, nil
	}
	return application.MetricStateNotApplicable, nil, nil, nil
}

func validateMetric(metric application.Metric) error {
	if metric.Name == "" || strings.TrimSpace(metric.Rationale) == "" {
		return ErrInvalidInput
	}
	switch metric.State {
	case application.MetricStateScored:
		if metric.Value == nil || metric.Denominator <= 0 || metric.Numerator < 0 {
			return ErrInvalidInput
		}
	case application.MetricStateNotApplicable, application.MetricStateNotScored, application.MetricStateNeedsHumanReview:
		if metric.Value != nil || metric.Numerator != 0 || metric.Denominator != 0 {
			return ErrInvalidInput
		}
	default:
		return ErrInvalidInput
	}
	return nil
}

func allNonBlank(values ...string) bool {
	for _, value := range values {
		if strings.TrimSpace(value) == "" {
			return false
		}
	}
	return true
}

func postgresInterval(duration time.Duration) string {
	return fmt.Sprintf("%f seconds", duration.Seconds())
}

var _ application.WorkStore = (*Repository)(nil)
