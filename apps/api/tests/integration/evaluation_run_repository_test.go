//go:build integration

package integration_test

import (
	"context"
	"errors"
	"reflect"
	"testing"
	"time"

	evaluationapplication "github.com/dharlanoliveira/norvii/apps/api/internal/evaluation/application"
	evaluationdomain "github.com/dharlanoliveira/norvii/apps/api/internal/evaluation/domain"
	evaluationpostgres "github.com/dharlanoliveira/norvii/apps/api/internal/evaluation/postgres"
	snapshotdomain "github.com/dharlanoliveira/norvii/apps/api/internal/snapshot/domain"
	snapshotpostgres "github.com/dharlanoliveira/norvii/apps/api/internal/snapshot/postgres"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

func TestEvaluationRunRepositoryRecoversLeasesAndPreservesTerminalResults(t *testing.T) {
	ctx, connection := openEvaluationSchemaConnection(t)
	transaction := beginEvaluationSchemaTransaction(t, ctx, connection)
	defer rollbackEvaluationSchemaTransaction(t, ctx, transaction)

	fixture := newEvaluationResolutionFixture(evaluationLGPDCorpusID, "20000000-0000-4000-8000-000000000001")
	seedEvaluationResolutionFixture(t, ctx, transaction, &fixture)
	caseID, _ := insertEvaluationCasePair(t, ctx, transaction, fixture.revisionID, uuid.NewString(), uuid.NewString(), 1)
	repository := evaluationpostgres.NewRepository(transaction)
	runID, runCaseID := uuid.New(), uuid.New()
	if err := repository.CreateRun(ctx, evaluationpostgres.CreateRunRequest{
		Identity: evaluationpostgres.RunIdentity{
			ID: runID, DatasetRevisionID: fixture.revisionID, CorpusID: fixture.corpusID, SnapshotID: fixture.snapshotID,
			SnapshotManifestSHA256: fixtureSHA256(fixture.snapshotID.String()),
			DatasetContentSHA256:   fixtureSHA256("dataset-" + fixture.revisionID.String()),
			OrderedCaseSetSHA256:   fixtureSHA256("ordered-" + runID.String()),
			RetrievalStrategy:      "vector", RetrievalConfigurationFingerprint: "fixture-fingerprint", ScoringPolicyVersion: "v1",
			AgentBuild: "fixture-agent", ChatModelIdentity: "fixture-chat", EmbeddingModelIdentity: "fixture-embedding", InitiatedBy: "integration-test",
		},
		Cases: []evaluationpostgres.RunCaseDefinition{{
			ID: runCaseID, DatasetCaseID: caseID, Position: 1, ExpectedOutcome: evaluationdomain.ExpectedOutcomeAnswer,
		}},
		ExpectedEvidence: []evaluationpostgres.ExpectedEvidenceDefinition{{
			ID: uuid.New(), RunCaseID: runCaseID, SourceID: fixture.sourceID, SourceRevisionID: fixture.sourceRevisionID,
			DocumentID: fixture.documentID, LegalUnitID: fixture.unitID, CanonicalLocator: "article:1", DisplayLocator: "Article 1",
			ContentSHA256: fixture.contentSHA256, Ordinal: 1,
		}},
	}); err != nil {
		t.Fatalf("CreateRun() error = %v", err)
	}

	claimed, err := repository.Claim(ctx, "first-worker", time.Minute, 1)
	if err != nil {
		t.Fatalf("Claim(first worker) error = %v", err)
	}
	if len(claimed) != 1 || claimed[0].ID != runCaseID || claimed[0].AttemptCount != 1 ||
		claimed[0].Configuration.Strategy != "vector" || claimed[0].Configuration.Fingerprint != "fixture-fingerprint" ||
		claimed[0].ExecutionIdentity.AgentBuild != "fixture-agent" || claimed[0].ExecutionIdentity.ChatModelIdentity != "fixture-chat" ||
		claimed[0].ExecutionIdentity.EmbeddingModelIdentity != "fixture-embedding" {
		t.Fatalf("Claim(first worker) = %#v, want one first attempt", claimed)
	}
	if _, err := transaction.Exec(ctx, `UPDATE evaluation_run_case SET lease_expires_at = now() - interval '1 second' WHERE id = $1`, runCaseID); err != nil {
		t.Fatalf("expire evaluation lease: %v", err)
	}
	recovered, err := repository.Claim(ctx, "recovery-worker", time.Minute, 1)
	if err != nil {
		t.Fatalf("Claim(recovery worker) error = %v", err)
	}
	if len(recovered) != 1 || recovered[0].ID != runCaseID || recovered[0].AttemptCount != 2 || recovered[0].LeaseToken == claimed[0].LeaseToken {
		t.Fatalf("Claim(recovery worker) = %#v, want a new second lease", recovered)
	}
	partialCompleted := evaluationapplication.TerminalCaseResult{
		State:  evaluationapplication.RunCaseCompleted,
		Answer: "Partial completed answer must not terminalize the case.",
		Metrics: []evaluationapplication.Metric{{
			Name: evaluationapplication.MetricRetrievalCoverage, State: evaluationapplication.MetricStateScored,
			Value: float64Pointer(1), Numerator: 1, Denominator: 1, Rationale: "incomplete metric set",
		}},
		ScoringPolicyVersion: "v1",
	}
	if err := repository.Complete(ctx, recovered[0], partialCompleted); !errors.Is(err, evaluationpostgres.ErrInvalidInput) {
		t.Fatalf("Complete(partial completed result) error = %v, want %v", err, evaluationpostgres.ErrInvalidInput)
	}
	var state string
	var metricCount int
	if err := transaction.QueryRow(ctx, `
		SELECT state, (SELECT count(*) FROM evaluation_run_metric WHERE run_case_id = evaluation_run_case.id)
		FROM evaluation_run_case
		WHERE id = $1`, runCaseID,
	).Scan(&state, &metricCount); err != nil {
		t.Fatalf("read partial completed case state: %v", err)
	}
	if state != "leased" || metricCount != 0 {
		t.Fatalf("partial completed case = state %q with %d metrics, want leased without metrics", state, metricCount)
	}
	if err := repository.Complete(ctx, recovered[0], evaluationapplication.TerminalCaseResult{
		State: evaluationapplication.RunCaseCompleted, Answer: "Synthetic completed answer.",
		LatencyMilliseconds: int64Pointer(42), InputTokens: int64Pointer(7), OutputTokens: int64Pointer(11),
		ActualEvidence: []evaluationapplication.ActualEvidence{{
			Kind: evaluationapplication.EvidenceKindRetrieved, Position: 1,
			Provenance: evaluationapplication.EvidenceProvenance{
				CorpusID: fixture.corpusID, SnapshotID: fixture.snapshotID, SourceID: fixture.sourceID,
				SourceRevisionID: fixture.sourceRevisionID, DocumentID: fixture.documentID, UnitID: fixture.unitID,
				CanonicalLocator: "article:1", StartOffset: 0, EndOffset: 21,
				ContentSHA256: evaluationdomain.SHA256(fixture.contentSHA256),
			},
		}},
		Metrics:              scorerV1CompletedMetrics(42, 7, 11),
		ScoringPolicyVersion: "v1",
	}); err != nil {
		t.Fatalf("Complete() error = %v", err)
	}
	if err := repository.Complete(ctx, recovered[0], evaluationapplication.TerminalCaseResult{
		State: evaluationapplication.RunCaseCompleted, Answer: "Replacement answer must not persist.",
	}); !errors.Is(err, evaluationapplication.ErrLeaseLost) {
		t.Fatalf("Complete(terminal case again) error = %v, want %v", err, evaluationapplication.ErrLeaseLost)
	}
	directUpdate, err := transaction.Begin(ctx)
	if err != nil {
		t.Fatalf("begin terminal case update savepoint: %v", err)
	}
	expectPostgresErrorCode(t, evaluationSchemaExecError(directUpdate.Exec(ctx,
		`UPDATE evaluation_run_case SET answer = 'Direct replacement must not persist.' WHERE id = $1`, runCaseID,
	)), "55000")
	if err := directUpdate.Rollback(ctx); err != nil {
		t.Fatalf("rollback terminal case update savepoint: %v", err)
	}
	var answer string
	if err := transaction.QueryRow(ctx, `SELECT answer FROM evaluation_run_case WHERE id = $1`, runCaseID).Scan(&answer); err != nil {
		t.Fatalf("read terminal case answer: %v", err)
	}
	if answer != "Synthetic completed answer." {
		t.Fatalf("terminal answer = %q, want the first completed result", answer)
	}
	t.Run("database rejects terminal child inserts", func(t *testing.T) {
		assertRejected := func(statement string, arguments ...any) {
			t.Helper()
			savepoint, err := transaction.Begin(ctx)
			if err != nil {
				t.Fatalf("begin terminal child rejection savepoint: %v", err)
			}
			expectPostgresErrorCode(t, evaluationSchemaExecError(savepoint.Exec(ctx, statement, arguments...)), "55000")
			if err := savepoint.Rollback(ctx); err != nil {
				t.Fatalf("rollback terminal child rejection savepoint: %v", err)
			}
		}
		assertRejected(`
			INSERT INTO evaluation_run_actual_evidence (
				id, run_id, run_case_id, corpus_id, snapshot_id, evidence_kind, position, marker_position,
				source_id, source_revision_id, document_id, legal_unit_id, canonical_locator,
				start_offset, end_offset, content_sha256
			) VALUES ($1, $2, $3, $4, $5, 'cited', 1, 1, $6, $7, $8, $9, 'article:1', 0, 21, $10)`,
			uuid.New(), runID, runCaseID, fixture.corpusID, fixture.snapshotID, fixture.sourceID,
			fixture.sourceRevisionID, fixture.documentID, fixture.unitID, fixture.contentSHA256,
		)
		assertRejected(`
			INSERT INTO evaluation_run_metric (
				id, run_id, run_case_id, corpus_id, component, metric_state, value, numerator,
				denominator, rationale, scorer_version
			) VALUES ($1, $2, $3, $4, 'late_metric', 'scored', 1, 1, 1, 'must not persist', 'v1')`,
			uuid.New(), runID, runCaseID, fixture.corpusID,
		)
		assertRejected(`
			INSERT INTO evaluation_run_expected_evidence (
				id, run_id, run_case_id, corpus_id, snapshot_id, source_id, source_revision_id,
				document_id, legal_unit_id, canonical_locator, display_locator, content_sha256, ordinal
			) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, 'article:1', 'Article 1', $10, 2)`,
			uuid.New(), runID, runCaseID, fixture.corpusID, fixture.snapshotID, fixture.sourceID,
			fixture.sourceRevisionID, fixture.documentID, fixture.unitID, fixture.contentSHA256,
		)
	})
	aggregate, err := repository.Aggregate(ctx, runID, fixture.corpusID)
	if err != nil {
		t.Fatalf("Aggregate() error = %v", err)
	}
	if aggregate.Total != 1 || aggregate.Eligible != 1 || aggregate.Failed != 0 || aggregate.Cancelled != 0 {
		t.Fatalf("Aggregate() = %#v, want explicit terminal denominators", aggregate)
	}
	if len(aggregate.Metrics) != len(requiredScorerV1Metrics) {
		t.Fatalf("Aggregate().Metrics = %#v, want the complete scorer-v1 component set", aggregate.Metrics)
	}
	assertRunAggregateMetric(t, aggregate, evaluationapplication.MetricRetrievalCoverage, 1, 1, 1)
	assertRunAggregateMetric(t, aggregate, evaluationapplication.MetricCitationCoverage, 0, 0, 1)
	assertRunAggregateMetric(t, aggregate, evaluationapplication.MetricLatency, 42, 42, 1)
	assertRunAggregateMetric(t, aggregate, evaluationapplication.MetricInputTokens, 7, 7, 1)
	assertRunAggregateMetric(t, aggregate, evaluationapplication.MetricOutputTokens, 11, 11, 1)
	assertRunAggregateMetricState(t, aggregate, evaluationapplication.MetricCitationValidity, evaluationapplication.MetricStateNotApplicable)
	assertRunAggregateMetricState(t, aggregate, evaluationapplication.MetricCitationScope, evaluationapplication.MetricStateNotApplicable)
	assertRunAggregateMetricState(t, aggregate, evaluationapplication.MetricExpectedAbstention, evaluationapplication.MetricStateNotApplicable)
	assertRunAggregateMetric(t, aggregate, evaluationapplication.MetricExecutionState, 1, 1, 1)
	assertRunAggregateMetricState(t, aggregate, evaluationapplication.MetricSemanticSupport, evaluationapplication.MetricStateNeedsHumanReview)
	var runState string
	if err := transaction.QueryRow(ctx, `SELECT state FROM evaluation_run WHERE id = $1`, runID).Scan(&runState); err != nil {
		t.Fatalf("read evaluation run state: %v", err)
	}
	if runState != "completed" {
		t.Fatalf("run state = %q, want completed", runState)
	}

	t.Run("failed and cancelled completion discard supplied scores", func(t *testing.T) {
		for index, terminal := range []struct {
			name  string
			state evaluationapplication.RunCaseState
		}{
			{name: "failed", state: evaluationapplication.RunCaseFailed},
			{name: "cancelled", state: evaluationapplication.RunCaseCancelled},
		} {
			t.Run(terminal.name, func(t *testing.T) {
				caseID, _ := insertEvaluationCasePair(t, ctx, transaction, fixture.revisionID, uuid.NewString(), uuid.NewString(), 3+index*2)
				runID, runCaseID := uuid.New(), uuid.New()
				if err := repository.CreateRun(ctx, evaluationRunRequest(fixture, runID, runCaseID, caseID, evaluationdomain.ExpectedOutcomeAnswer)); err != nil {
					t.Fatalf("CreateRun(%s) error = %v", terminal.name, err)
				}
				claims, err := repository.Claim(ctx, terminal.name+"-worker", time.Minute, 1)
				if err != nil {
					t.Fatalf("Claim(%s) error = %v", terminal.name, err)
				}
				if len(claims) != 1 || claims[0].ID != runCaseID {
					t.Fatalf("Claim(%s) = %#v, want the terminal case", terminal.name, claims)
				}
				if err := repository.Complete(ctx, claims[0], evaluationapplication.TerminalCaseResult{
					State: terminal.state, SafeFailureCode: "provider_unavailable",
					Metrics: []evaluationapplication.Metric{{
						Name: evaluationapplication.MetricRetrievalCoverage, State: evaluationapplication.MetricStateScored,
						Value: float64Pointer(1), Numerator: 1, Denominator: 1, Rationale: "must be discarded",
					}},
					ScoringPolicyVersion: "v1",
				}); err != nil {
					t.Fatalf("Complete(%s) error = %v", terminal.name, err)
				}
				aggregate, err := repository.Aggregate(ctx, runID, fixture.corpusID)
				if err != nil {
					t.Fatalf("Aggregate(%s) error = %v", terminal.name, err)
				}
				if aggregate.Total != 1 || aggregate.Eligible != 0 || aggregate.Scored != 0 ||
					(terminal.state == evaluationapplication.RunCaseFailed && aggregate.Failed != 1) ||
					(terminal.state == evaluationapplication.RunCaseCancelled && aggregate.Cancelled != 1) {
					t.Fatalf("Aggregate(%s) = %#v, want an ineligible unscored terminal case", terminal.name, aggregate)
				}
				assertUnscoredAggregateContract(t, aggregate)
			})
		}
	})

	t.Run("abstained completion requires the complete scorer-v1 set", func(t *testing.T) {
		abstainedCaseID, _ := insertEvaluationCasePairWithOutcome(
			t, ctx, transaction, fixture.revisionID, uuid.NewString(), uuid.NewString(), 7, "abstain", "insufficient_evidence",
		)
		abstainedRunID, abstainedRunCaseID := uuid.New(), uuid.New()
		if err := repository.CreateRun(ctx, evaluationRunRequest(fixture, abstainedRunID, abstainedRunCaseID, abstainedCaseID, evaluationdomain.ExpectedOutcomeAbstain)); err != nil {
			t.Fatalf("CreateRun(abstained) error = %v", err)
		}
		claims, err := repository.Claim(ctx, "abstained-worker", time.Minute, 1)
		if err != nil {
			t.Fatalf("Claim(abstained) error = %v", err)
		}
		if len(claims) != 1 || claims[0].ID != abstainedRunCaseID {
			t.Fatalf("Claim(abstained) = %#v, want the abstained case", claims)
		}
		partial := evaluationapplication.TerminalCaseResult{
			State: evaluationapplication.RunCaseAbstained,
			Metrics: []evaluationapplication.Metric{{
				Name: evaluationapplication.MetricExpectedAbstention, State: evaluationapplication.MetricStateScored,
				Value: float64Pointer(1), Numerator: 1, Denominator: 1, Rationale: "agent abstained as required",
			}},
			ScoringPolicyVersion: "v1",
		}
		if err := repository.Complete(ctx, claims[0], partial); !errors.Is(err, evaluationpostgres.ErrInvalidInput) {
			t.Fatalf("Complete(partial abstained result) error = %v, want %v", err, evaluationpostgres.ErrInvalidInput)
		}
		if err := repository.Complete(ctx, claims[0], evaluationapplication.TerminalCaseResult{
			State: evaluationapplication.RunCaseAbstained, Metrics: scorerV1AbstainedMetrics(), ScoringPolicyVersion: "v1",
		}); err != nil {
			t.Fatalf("Complete(full abstained result) error = %v", err)
		}
		aggregate, err := repository.Aggregate(ctx, abstainedRunID, fixture.corpusID)
		if err != nil {
			t.Fatalf("Aggregate(abstained) error = %v", err)
		}
		if aggregate.Total != 1 || aggregate.Eligible != 1 || len(aggregate.Metrics) != len(requiredScorerV1Metrics) {
			t.Fatalf("Aggregate(abstained) = %#v, want a complete scoreable terminal contract", aggregate)
		}
		assertRunAggregateMetricState(t, aggregate, evaluationapplication.MetricRetrievalCoverage, evaluationapplication.MetricStateNotApplicable)
		assertRunAggregateMetricState(t, aggregate, evaluationapplication.MetricCitationCoverage, evaluationapplication.MetricStateNotApplicable)
		assertRunAggregateMetricState(t, aggregate, evaluationapplication.MetricCitationValidity, evaluationapplication.MetricStateNotApplicable)
		assertRunAggregateMetricState(t, aggregate, evaluationapplication.MetricCitationScope, evaluationapplication.MetricStateNotApplicable)
		assertRunAggregateMetric(t, aggregate, evaluationapplication.MetricExpectedAbstention, 1, 1, 1)
		assertRunAggregateMetric(t, aggregate, evaluationapplication.MetricExecutionState, 1, 1, 1)
		assertRunAggregateMetricState(t, aggregate, evaluationapplication.MetricLatency, evaluationapplication.MetricStateNotApplicable)
		assertRunAggregateMetricState(t, aggregate, evaluationapplication.MetricInputTokens, evaluationapplication.MetricStateNotApplicable)
		assertRunAggregateMetricState(t, aggregate, evaluationapplication.MetricOutputTokens, evaluationapplication.MetricStateNotApplicable)
		assertRunAggregateMetricState(t, aggregate, evaluationapplication.MetricSemanticSupport, evaluationapplication.MetricStateNeedsHumanReview)
	})
}

func TestEvaluationRunInspectionRetainsHistoricalSnapshotAndImmutableCaseLedger(t *testing.T) {
	ctx, connection := openEvaluationSchemaConnection(t)
	transaction := beginEvaluationSchemaTransaction(t, ctx, connection)
	defer rollbackEvaluationSchemaTransaction(t, ctx, transaction)

	fixture := newEvaluationResolutionFixture(evaluationLGPDCorpusID, "20000000-0000-4000-8000-000000000001")
	seedEvaluationResolutionFixture(t, ctx, transaction, &fixture)
	publishedAt := time.Date(2026, time.August, 26, 15, 45, 0, 0, time.UTC)
	repository := evaluationpostgres.NewRepository(transaction)
	publicationService := evaluationapplication.NewPublicationService(
		repository,
		uuid.New,
		func() time.Time { return publishedAt },
	)
	historicalPublication, err := publicationService.Review(ctx, evaluationapplication.ReviewCommand{
		CorpusID: fixture.corpusID, DatasetRevisionID: fixture.revisionID,
		ReviewDecision: evaluationdomain.ReviewDecisionApproved, ReviewerIdentity: "integration-test",
		ReviewNote:       "Baseline revision reviewed for retention coverage.",
		PublicationState: evaluationdomain.PublicationStateAvailable,
	})
	if err != nil {
		t.Fatalf("Review(baseline dataset revision) error = %v", err)
	}
	var originalReleaseVersion int
	if err := transaction.QueryRow(ctx, `
		INSERT INTO corpus_snapshot_releases (corpus_id, snapshot_id, version, activated_at)
		VALUES ($1, $2, 1, now())
		ON CONFLICT (corpus_id) DO UPDATE
		SET snapshot_id = EXCLUDED.snapshot_id,
			version = corpus_snapshot_releases.version + 1,
			activated_at = EXCLUDED.activated_at
		RETURNING version`, fixture.corpusID, fixture.snapshotID,
	).Scan(&originalReleaseVersion); err != nil {
		t.Fatalf("publish original snapshot: %v", err)
	}
	caseID, _ := insertEvaluationCasePair(t, ctx, transaction, fixture.revisionID, uuid.NewString(), uuid.NewString(), 1)
	runID, runCaseID := uuid.New(), uuid.New()
	request := evaluationRunRequest(fixture, runID, runCaseID, caseID, evaluationdomain.ExpectedOutcomeAnswer)
	request.ExpectedEvidence = []evaluationpostgres.ExpectedEvidenceDefinition{{
		ID: uuid.New(), RunCaseID: runCaseID, SourceID: fixture.sourceID, SourceRevisionID: fixture.sourceRevisionID,
		DocumentID: fixture.documentID, LegalUnitID: fixture.unitID, CanonicalLocator: "article:1", DisplayLocator: "Article 1",
		ContentSHA256: fixture.contentSHA256, Ordinal: 1,
	}}
	if err := repository.CreateRun(ctx, request); err != nil {
		t.Fatalf("CreateRun() error = %v", err)
	}

	claims, err := repository.Claim(ctx, "historical-worker", time.Minute, 1)
	if err != nil {
		t.Fatalf("Claim() error = %v", err)
	}
	if len(claims) != 1 || claims[0].ID != runCaseID {
		t.Fatalf("Claim() = %#v, want the historical run case", claims)
	}
	if err := repository.Complete(ctx, claims[0], evaluationapplication.TerminalCaseResult{
		State: evaluationapplication.RunCaseCompleted, Answer: "Historical completed answer.",
		ActualEvidence: []evaluationapplication.ActualEvidence{{
			Kind: evaluationapplication.EvidenceKindRetrieved, Position: 1,
			Provenance: evaluationapplication.EvidenceProvenance{
				CorpusID: fixture.corpusID, SnapshotID: fixture.snapshotID, SourceID: fixture.sourceID,
				SourceRevisionID: fixture.sourceRevisionID, DocumentID: fixture.documentID, UnitID: fixture.unitID,
				CanonicalLocator: "article:1", StartOffset: 0, EndOffset: 21,
				ContentSHA256: evaluationdomain.SHA256(fixture.contentSHA256),
			},
		}},
		Metrics: scorerV1CompletedMetrics(42, 7, 11), ScoringPolicyVersion: "v1",
	}); err != nil {
		t.Fatalf("Complete() error = %v", err)
	}

	historicalRun, err := repository.GetEvaluationRun(ctx, runID)
	if err != nil {
		t.Fatalf("GetEvaluationRun(before publication) error = %v", err)
	}
	historicalCase, err := repository.GetEvaluationRunCase(ctx, runID, runCaseID)
	if err != nil {
		t.Fatalf("GetEvaluationRunCase(before publication) error = %v", err)
	}

	// Publish a later ready source revision through the repository boundary.
	laterSourceRevisionID, laterDocumentID, laterContentSHA256 := insertLaterReadyEvaluationDocument(t, ctx, transaction, fixture)
	snapshotRepository := snapshotpostgres.NewRepository(transaction)
	laterPublication, err := snapshotRepository.Publish(ctx, snapshotdomain.PublishCommand{
		CorpusID: fixture.corpusID, SourceID: fixture.sourceID, DocumentID: laterDocumentID,
		ExpectedReleaseVersion: originalReleaseVersion, SnapshotID: uuid.New(), Actor: "integration-test",
		PublishedAt: time.Date(2026, time.August, 26, 16, 0, 0, 0, time.UTC),
	})
	if err != nil {
		t.Fatalf("Publish(later snapshot) error = %v", err)
	}
	if !laterPublication.Created || laterPublication.Release.Version != originalReleaseVersion+1 ||
		laterPublication.Snapshot.ID != laterPublication.Release.SnapshotID || len(laterPublication.Snapshot.Members) != 1 ||
		laterPublication.Snapshot.Members[0].SourceRevisionID != laterSourceRevisionID ||
		laterPublication.Snapshot.Members[0].DocumentID != laterDocumentID ||
		laterPublication.Snapshot.Members[0].ContentSHA256 != laterContentSHA256 {
		t.Fatalf("Publish(later snapshot) = %#v, want the later ready source revision in the next release", laterPublication)
	}

	laterDatasetRevisionID := uuid.New()
	insertEvaluationRevision(t, ctx, transaction, laterDatasetRevisionID, evaluationLGPDCorpusID)
	laterPublicationTime := time.Date(2026, time.August, 26, 16, 1, 0, 0, time.UTC)
	laterPublicationService := evaluationapplication.NewPublicationService(
		repository,
		uuid.New,
		func() time.Time { return laterPublicationTime },
	)
	laterDatasetPublication, err := laterPublicationService.Review(ctx, evaluationapplication.ReviewCommand{
		CorpusID: fixture.corpusID, DatasetRevisionID: laterDatasetRevisionID,
		ReviewDecision: evaluationdomain.ReviewDecisionApproved, ReviewerIdentity: "integration-test",
		ReviewNote:       "Later revision reviewed for retention coverage.",
		PublicationState: evaluationdomain.PublicationStateAvailable,
	})
	if err != nil {
		t.Fatalf("Review(later dataset revision) error = %v", err)
	}
	availableLaterPublication, found, err := repository.LatestPublication(ctx, fixture.corpusID, laterDatasetRevisionID)
	if err != nil {
		t.Fatalf("LatestPublication(later dataset revision) error = %v", err)
	}
	if !found || availableLaterPublication.PublicationState != evaluationdomain.PublicationStateAvailable ||
		availableLaterPublication.ReviewDecision != evaluationdomain.ReviewDecisionApproved {
		t.Fatalf("LatestPublication(later dataset revision) = %#v, want the available approved later revision %#v", availableLaterPublication, laterDatasetPublication)
	}
	assertEvaluationPublication(t, availableLaterPublication, laterDatasetPublication)

	var activeSnapshotID uuid.UUID
	var activeReleaseVersion int
	if err := transaction.QueryRow(ctx, `
		SELECT snapshot_id, version FROM corpus_snapshot_releases WHERE corpus_id = $1`, fixture.corpusID,
	).Scan(&activeSnapshotID, &activeReleaseVersion); err != nil {
		t.Fatalf("read later snapshot publication: %v", err)
	}
	if activeSnapshotID != laterPublication.Snapshot.ID || activeReleaseVersion != originalReleaseVersion+1 {
		t.Fatalf("active snapshot publication = (%s, %d), want (%s, %d)", activeSnapshotID, activeReleaseVersion, laterPublication.Snapshot.ID, originalReleaseVersion+1)
	}

	historicalSnapshot, err := snapshotRepository.Get(ctx, fixture.corpusID, fixture.snapshotID)
	if err != nil {
		t.Fatalf("Get(historical snapshot) error = %v", err)
	}
	if len(historicalSnapshot.Members) != 1 || historicalSnapshot.Members[0].DocumentID != fixture.documentID ||
		historicalSnapshot.Members[0].SourceRevisionID != fixture.sourceRevisionID || historicalSnapshot.Members[0].ContentSHA256 != fixture.contentSHA256 {
		t.Fatalf("Get(historical snapshot) = %#v, want the original immutable member", historicalSnapshot)
	}
	publication, found, err := repository.LatestPublication(ctx, fixture.corpusID, fixture.revisionID)
	if err != nil {
		t.Fatalf("LatestPublication(historical dataset revision) error = %v", err)
	}
	if !found {
		t.Fatalf("LatestPublication(historical dataset revision) = %#v, want unchanged publication %#v", publication, historicalPublication)
	}
	assertEvaluationPublication(t, publication, historicalPublication)

	run, err := repository.GetEvaluationRun(ctx, runID)
	if err != nil {
		t.Fatalf("GetEvaluationRun(after publication) error = %v", err)
	}
	if !reflect.DeepEqual(run, historicalRun) {
		t.Fatalf("GetEvaluationRun() after publication = %#v, want unchanged historical run %#v", run, historicalRun)
	}
	result, err := repository.GetEvaluationRunCase(ctx, runID, runCaseID)
	if err != nil {
		t.Fatalf("GetEvaluationRunCase(after publication) error = %v", err)
	}
	if !reflect.DeepEqual(result, historicalCase) {
		t.Fatalf("GetEvaluationRunCase() after publication = %#v, want unchanged historical case %#v", result, historicalCase)
	}
	if result.SnapshotID != fixture.snapshotID || len(result.ExpectedEvidence) != 1 || result.ExpectedEvidence[0].SnapshotID != fixture.snapshotID ||
		result.ExpectedEvidence[0].ContentSHA256 != fixture.contentSHA256 || result.Answer != "Historical completed answer." {
		t.Fatalf("GetEvaluationRunCase() = %#v, want the original immutable ledger and expected evidence", result)
	}
}

func insertLaterReadyEvaluationDocument(
	t *testing.T,
	ctx context.Context,
	transaction pgx.Tx,
	fixture evaluationResolutionFixture,
) (uuid.UUID, uuid.UUID, string) {
	t.Helper()

	workID, attemptID, sourceRevisionID := uuid.New(), uuid.New(), uuid.New()
	documentID, unitID, chunkID := uuid.New(), uuid.New(), uuid.New()
	capturedAt := time.Date(2026, time.August, 26, 15, 30, 0, 0, time.UTC)
	contentSHA256 := fixtureSHA256("later-content-" + sourceRevisionID.String())
	documentSHA256 := fixtureSHA256("later-document-" + documentID.String())
	unitSHA256 := fixtureSHA256("later-unit-" + unitID.String())
	if _, err := transaction.Exec(ctx, `
		INSERT INTO ingestion_work (id, source_id, corpus_id, reason, status)
		VALUES ($1, $2, $3, 'reprocess', 'succeeded')`,
		workID, fixture.sourceID, fixture.corpusID,
	); err != nil {
		t.Fatalf("insert later evaluation work: %v", err)
	}
	if _, err := transaction.Exec(ctx, `
		INSERT INTO processing_attempts (
			id, work_id, source_id, corpus_id, attempt_number, pipeline_version, status,
			lease_token, worker_id, started_at, finished_at
		) VALUES ($1, $2, $3, $4, 1, 'evaluation-retention-test', 'succeeded',
			$5, 'integration-test', $6, $6)`,
		attemptID, workID, fixture.sourceID, fixture.corpusID, uuid.New(), capturedAt,
	); err != nil {
		t.Fatalf("insert later evaluation processing attempt: %v", err)
	}
	if _, err := transaction.Exec(ctx, `
		INSERT INTO source_revisions (
			id, source_id, corpus_id, attempt_id, content_sha256, captured_at, media_type,
			byte_size, pipeline_version, final_url, extracted_content_sha256
		) VALUES ($1, $2, $3, $4, $5, $6, 'text/html', 256, 'evaluation-retention-test',
			'https://example.test/evaluation/later', $5)`,
		sourceRevisionID, fixture.sourceID, fixture.corpusID, attemptID, contentSHA256, capturedAt,
	); err != nil {
		t.Fatalf("insert later evaluation source revision: %v", err)
	}
	if _, err := transaction.Exec(ctx, `
		INSERT INTO document_versions (
			id, source_revision_id, source_id, corpus_id, pipeline_version, text_content,
			text_sha256, published_at
		) VALUES ($1, $2, $3, $4, 'evaluation-retention-test', 'Later Article 1 legal text.', $5, $6)`,
		documentID, sourceRevisionID, fixture.sourceID, fixture.corpusID, documentSHA256, capturedAt,
	); err != nil {
		t.Fatalf("insert later evaluation document: %v", err)
	}
	if _, err := transaction.Exec(ctx, `
		INSERT INTO document_units (
			id, document_id, kind, ordinal, start_offset, end_offset, locator,
			canonical_locator, content_sha256
		) VALUES ($1, $2, 'article', 1, 0, 27, 'article-1', 'article:1', $3)`,
		unitID, documentID, unitSHA256,
	); err != nil {
		t.Fatalf("insert later evaluation legal unit: %v", err)
	}
	if _, err := transaction.Exec(ctx, `
		INSERT INTO retrieval_chunks (
			id, corpus_id, source_id, document_id, unit_id, ordinal, start_offset, end_offset,
			content, content_sha256, context_locator, embedding, embedding_model, enrichment_status
		) VALUES ($1, $2, $3, $4, $5, 1, 0, 27, 'Later Article 1 legal text.', $6,
			'article-1', $7::vector, 'evaluation-retention-test', 'ready')`,
		chunkID, fixture.corpusID, fixture.sourceID, documentID, unitID, documentSHA256, zeroVector(),
	); err != nil {
		t.Fatalf("insert later evaluation retrieval chunk: %v", err)
	}
	return sourceRevisionID, documentID, contentSHA256
}

func assertEvaluationPublication(
	t *testing.T,
	actual, expected evaluationdomain.Publication,
) {
	t.Helper()
	if actual.ID != expected.ID || actual.DatasetRevisionID != expected.DatasetRevisionID ||
		actual.CorpusID != expected.CorpusID || actual.ReviewDecision != expected.ReviewDecision ||
		actual.ReviewerIdentity != expected.ReviewerIdentity || actual.ReviewNote != expected.ReviewNote ||
		actual.PublicationState != expected.PublicationState || !actual.ReviewedAt.Equal(expected.ReviewedAt) {
		t.Fatalf("evaluation dataset publication = %#v, want %#v", actual, expected)
	}
}

func assertRunAggregateMetric(
	t *testing.T,
	aggregate evaluationpostgres.RunAggregate,
	name evaluationapplication.MetricName,
	wantValue float64,
	wantNumerator, wantDenominator int64,
) {
	t.Helper()
	for _, metric := range aggregate.Metrics {
		if metric.Name != name {
			continue
		}
		if metric.State != evaluationapplication.MetricStateScored || metric.Value == nil || metric.Numerator == nil || metric.Denominator == nil ||
			*metric.Value != wantValue || *metric.Numerator != wantNumerator || *metric.Denominator != wantDenominator {
			t.Fatalf("aggregate metric %q = %#v, want scored value %v (%d/%d)", name, metric, wantValue, wantNumerator, wantDenominator)
		}
		return
	}
	t.Fatalf("Aggregate().Metrics does not contain %q: %#v", name, aggregate.Metrics)
}

func assertRunAggregateMetricState(
	t *testing.T,
	aggregate evaluationpostgres.RunAggregate,
	name evaluationapplication.MetricName,
	wantState evaluationapplication.MetricState,
) {
	t.Helper()
	for _, metric := range aggregate.Metrics {
		if metric.Name != name {
			continue
		}
		if metric.State != wantState || metric.Value != nil || metric.Numerator != nil || metric.Denominator != nil {
			t.Fatalf("aggregate metric %q = %#v, want %q without numeric fields", name, metric, wantState)
		}
		return
	}
	t.Fatalf("Aggregate().Metrics does not contain %q: %#v", name, aggregate.Metrics)
}

func assertUnscoredAggregateContract(t *testing.T, aggregate evaluationpostgres.RunAggregate) {
	t.Helper()
	if len(aggregate.Metrics) != len(requiredScorerV1Metrics) {
		t.Fatalf("Aggregate().Metrics = %#v, want every scorer-v1 component", aggregate.Metrics)
	}
	for _, name := range requiredScorerV1Metrics {
		assertRunAggregateMetricState(t, aggregate, name, evaluationapplication.MetricStateNotScored)
	}
}

func evaluationRunRequest(
	fixture evaluationResolutionFixture,
	runID, runCaseID, caseID uuid.UUID,
	expectedOutcome evaluationdomain.ExpectedOutcome,
) evaluationpostgres.CreateRunRequest {
	return evaluationpostgres.CreateRunRequest{
		Identity: evaluationpostgres.RunIdentity{
			ID: runID, DatasetRevisionID: fixture.revisionID, CorpusID: fixture.corpusID, SnapshotID: fixture.snapshotID,
			SnapshotManifestSHA256: fixtureSHA256(fixture.snapshotID.String()),
			DatasetContentSHA256:   fixtureSHA256("dataset-" + fixture.revisionID.String()),
			OrderedCaseSetSHA256:   fixtureSHA256("ordered-" + runID.String()),
			RetrievalStrategy:      "vector", RetrievalConfigurationFingerprint: "fixture-fingerprint", ScoringPolicyVersion: "v1",
			AgentBuild: "fixture-agent", ChatModelIdentity: "fixture-chat", EmbeddingModelIdentity: "fixture-embedding", InitiatedBy: "integration-test",
		},
		Cases: []evaluationpostgres.RunCaseDefinition{{
			ID: runCaseID, DatasetCaseID: caseID, Position: 1, ExpectedOutcome: expectedOutcome,
		}},
	}
}

var requiredScorerV1Metrics = []evaluationapplication.MetricName{
	evaluationapplication.MetricRetrievalCoverage,
	evaluationapplication.MetricCitationCoverage,
	evaluationapplication.MetricCitationValidity,
	evaluationapplication.MetricCitationScope,
	evaluationapplication.MetricExpectedAbstention,
	evaluationapplication.MetricExecutionState,
	evaluationapplication.MetricLatency,
	evaluationapplication.MetricInputTokens,
	evaluationapplication.MetricOutputTokens,
	evaluationapplication.MetricSemanticSupport,
}

func scorerV1CompletedMetrics(latency, inputTokens, outputTokens int64) []evaluationapplication.Metric {
	return []evaluationapplication.Metric{
		{Name: evaluationapplication.MetricRetrievalCoverage, State: evaluationapplication.MetricStateScored, Value: float64Pointer(1), Numerator: 1, Denominator: 1, Rationale: "retrieved the required evidence"},
		{Name: evaluationapplication.MetricCitationCoverage, State: evaluationapplication.MetricStateScored, Value: float64Pointer(0), Numerator: 0, Denominator: 1, Rationale: "no expected evidence was cited"},
		{Name: evaluationapplication.MetricCitationValidity, State: evaluationapplication.MetricStateNotApplicable, Rationale: "answer contains no citation markers"},
		{Name: evaluationapplication.MetricCitationScope, State: evaluationapplication.MetricStateNotApplicable, Rationale: "answer contains no citation markers"},
		{Name: evaluationapplication.MetricExpectedAbstention, State: evaluationapplication.MetricStateNotApplicable, Rationale: "case does not require abstention"},
		{Name: evaluationapplication.MetricExecutionState, State: evaluationapplication.MetricStateScored, Value: float64Pointer(1), Numerator: 1, Denominator: 1, Rationale: "case reached a scoreable terminal state"},
		{Name: evaluationapplication.MetricLatency, State: evaluationapplication.MetricStateScored, Value: float64Pointer(float64(latency)), Numerator: latency, Denominator: 1, Rationale: "reported end-to-end latency"},
		{Name: evaluationapplication.MetricInputTokens, State: evaluationapplication.MetricStateScored, Value: float64Pointer(float64(inputTokens)), Numerator: inputTokens, Denominator: 1, Rationale: "reported input token use"},
		{Name: evaluationapplication.MetricOutputTokens, State: evaluationapplication.MetricStateScored, Value: float64Pointer(float64(outputTokens)), Numerator: outputTokens, Denominator: 1, Rationale: "reported output token use"},
		{Name: evaluationapplication.MetricSemanticSupport, State: evaluationapplication.MetricStateNeedsHumanReview, Rationale: "claim-to-citation support requires a reviewed human rubric"},
	}
}

func scorerV1AbstainedMetrics() []evaluationapplication.Metric {
	return []evaluationapplication.Metric{
		{Name: evaluationapplication.MetricRetrievalCoverage, State: evaluationapplication.MetricStateNotApplicable, Rationale: "no expected evidence targets"},
		{Name: evaluationapplication.MetricCitationCoverage, State: evaluationapplication.MetricStateNotApplicable, Rationale: "no expected evidence targets"},
		{Name: evaluationapplication.MetricCitationValidity, State: evaluationapplication.MetricStateNotApplicable, Rationale: "answer contains no citation markers"},
		{Name: evaluationapplication.MetricCitationScope, State: evaluationapplication.MetricStateNotApplicable, Rationale: "answer contains no citation markers"},
		{Name: evaluationapplication.MetricExpectedAbstention, State: evaluationapplication.MetricStateScored, Value: float64Pointer(1), Numerator: 1, Denominator: 1, Rationale: "agent abstained as required"},
		{Name: evaluationapplication.MetricExecutionState, State: evaluationapplication.MetricStateScored, Value: float64Pointer(1), Numerator: 1, Denominator: 1, Rationale: "case reached a scoreable terminal state"},
		{Name: evaluationapplication.MetricLatency, State: evaluationapplication.MetricStateNotApplicable, Rationale: "measurement was not reported"},
		{Name: evaluationapplication.MetricInputTokens, State: evaluationapplication.MetricStateNotApplicable, Rationale: "measurement was not reported"},
		{Name: evaluationapplication.MetricOutputTokens, State: evaluationapplication.MetricStateNotApplicable, Rationale: "measurement was not reported"},
		{Name: evaluationapplication.MetricSemanticSupport, State: evaluationapplication.MetricStateNeedsHumanReview, Rationale: "claim-to-citation support requires a reviewed human rubric"},
	}
}

func float64Pointer(value float64) *float64 { return &value }

func int64Pointer(value int64) *int64 { return &value }
