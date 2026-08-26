//go:build integration

package integration_test

import (
	"context"
	"errors"
	"testing"
	"time"

	evaluationapplication "github.com/dharlanoliveira/norvii/apps/api/internal/evaluation/application"
	evaluationpostgres "github.com/dharlanoliveira/norvii/apps/api/internal/evaluation/postgres"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

func TestEvaluationPreflightRequiresAvailabilityCorpusSourcesAndLocators(t *testing.T) {
	ctx, connection := openEvaluationSchemaConnection(t)

	for _, testCase := range []struct {
		name      string
		configure func(t *testing.T, ctx context.Context, transaction pgx.Tx, fixture evaluationPreflightFixture)
		request   func(evaluationPreflightFixture) evaluationapplication.PreflightRequest
		locator   string
		want      error
	}{
		{
			name: "draft dataset", configure: func(t *testing.T, ctx context.Context, transaction pgx.Tx, fixture evaluationPreflightFixture) {
				insertEvaluationPreflightPublication(t, ctx, transaction, fixture, "pending", "draft")
			}, want: evaluationapplication.ErrDatasetUnavailable,
		},
		{
			name: "wrong corpus", configure: func(t *testing.T, ctx context.Context, transaction pgx.Tx, fixture evaluationPreflightFixture) {
				insertEvaluationPreflightPublication(t, ctx, transaction, fixture, "approved", "available")
			}, request: func(fixture evaluationPreflightFixture) evaluationapplication.PreflightRequest {
				return evaluationapplication.PreflightRequest{
					CorpusID: uuid.MustParse("10000000-0000-4000-8000-000000000003"), DatasetRevisionID: fixture.revisionID, SnapshotID: fixture.snapshotID,
				}
			}, want: evaluationapplication.ErrPreflightCorpusMismatch,
		},
		{
			name: "missing source", configure: func(t *testing.T, ctx context.Context, transaction pgx.Tx, fixture evaluationPreflightFixture) {
				insertEvaluationPreflightPublication(t, ctx, transaction, fixture, "approved", "available")
				if _, err := transaction.Exec(ctx, `DELETE FROM corpus_snapshot_documents WHERE snapshot_id = $1`, fixture.snapshotID); err != nil {
					t.Fatalf("remove snapshot source membership: %v", err)
				}
			}, want: evaluationapplication.ErrSnapshotIncompatible,
		},
		{
			name: "unresolved locator", configure: func(t *testing.T, ctx context.Context, transaction pgx.Tx, fixture evaluationPreflightFixture) {
				insertEvaluationPreflightPublication(t, ctx, transaction, fixture, "approved", "available")
			}, locator: "article:404", want: evaluationapplication.ErrLocatorUnresolved,
		},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			transaction := beginEvaluationSchemaTransaction(t, ctx, connection)
			defer rollbackEvaluationSchemaTransaction(t, ctx, transaction)

			fixture := seedEvaluationPreflightFixture(t, ctx, transaction, testCase.locator)
			testCase.configure(t, ctx, transaction, fixture)
			request := evaluationapplication.PreflightRequest{
				CorpusID: fixture.corpusID, DatasetRevisionID: fixture.revisionID, SnapshotID: fixture.snapshotID,
			}
			if testCase.request != nil {
				request = testCase.request(fixture)
			}
			model := evaluationModelSentinel{}
			_, err := evaluationapplication.NewPreflightService(evaluationpostgres.NewRepository(transaction)).Check(ctx, request)
			if !errors.Is(err, testCase.want) {
				t.Fatalf("Check() error = %v, want %v", err, testCase.want)
			}
			if model.calls != 0 {
				t.Fatalf("model calls = %d, want zero after rejected preflight", model.calls)
			}
		})
	}
}

func TestEvaluationPreflightReturnsAllResolvedLocatorsOnlyAfterSuccess(t *testing.T) {
	ctx, connection := openEvaluationSchemaConnection(t)
	transaction := beginEvaluationSchemaTransaction(t, ctx, connection)
	defer rollbackEvaluationSchemaTransaction(t, ctx, transaction)

	fixture := seedEvaluationPreflightFixture(t, ctx, transaction, "")
	insertEvaluationPreflightPublication(t, ctx, transaction, fixture, "approved", "available")
	result, err := evaluationapplication.NewPreflightService(evaluationpostgres.NewRepository(transaction)).Check(ctx, evaluationapplication.PreflightRequest{
		CorpusID: fixture.corpusID, DatasetRevisionID: fixture.revisionID, SnapshotID: fixture.snapshotID,
	})
	if err != nil {
		t.Fatalf("Check() error = %v", err)
	}
	if len(result.ResolvedLocators) != 1 || result.ResolvedLocators[0].UnitID != fixture.unitID ||
		result.ResolvedLocators[0].CanonicalLocator != "article:1" {
		t.Fatalf("Check() result = %#v, want complete immutable locator resolution", result)
	}
}

type evaluationPreflightFixture struct {
	corpusID           uuid.UUID
	sourceID           uuid.UUID
	revisionID         uuid.UUID
	snapshotID         uuid.UUID
	sourceRevisionID   uuid.UUID
	documentID         uuid.UUID
	unitID             uuid.UUID
	caseID             uuid.UUID
	pairedCaseID       uuid.UUID
	caseChecksum       string
	pairedCaseChecksum string
	expectedEvidenceID uuid.UUID
}

func seedEvaluationPreflightFixture(
	t *testing.T,
	ctx context.Context,
	transaction pgx.Tx,
	locator string,
) evaluationPreflightFixture {
	t.Helper()
	resolution := newEvaluationResolutionFixture(evaluationLGPDCorpusID, "20000000-0000-4000-8000-000000000001")
	seedEvaluationResolutionFixture(t, ctx, transaction, &resolution)
	repository := evaluationpostgres.NewRepository(transaction)
	bindEvaluationFixtureSource(t, ctx, repository, resolution)

	fixture := evaluationPreflightFixture{
		corpusID: resolution.corpusID, sourceID: resolution.sourceID, revisionID: resolution.revisionID,
		snapshotID: resolution.snapshotID, sourceRevisionID: resolution.sourceRevisionID, documentID: resolution.documentID,
		unitID: resolution.unitID, caseID: uuid.New(), expectedEvidenceID: uuid.New(),
	}
	if locator == "" {
		locator = "article:1"
	}
	// The case table enforces reciprocal records, so insert a valid pair in one deferred transaction.
	pairID := uuid.New()
	externalID, pairExternalID := "preflight-"+fixture.caseID.String(), "preflight-"+pairID.String()
	fixture.pairedCaseID = pairID
	fixture.caseChecksum = fixtureSHA256("preflight-case-en-" + fixture.caseID.String())
	fixture.pairedCaseChecksum = fixtureSHA256("preflight-case-pt-" + pairID.String())
	if _, err := transaction.Exec(ctx, `
		INSERT INTO evaluation_dataset_case (
			id, dataset_revision_id, corpus_id, position, external_case_id, query_language,
			asset_language, question, reference_answer, category, authoritative_evidence_language,
			expected_outcome, reciprocal_case_external_id, case_checksum
		) VALUES
			($1, $2, $3, 1, $4, 'en', 'en', 'Synthetic question?', 'Synthetic answer.', 'Synthetic category', 'en', 'answer', $5, $6),
			($7, $2, $3, 2, $5, 'pt', 'pt-BR', 'Synthetic Portuguese question?', 'Synthetic Portuguese answer.', 'Synthetic category', 'en', 'answer', $4, $8)`,
		fixture.caseID, fixture.revisionID, fixture.corpusID, externalID, pairExternalID, fixture.caseChecksum,
		pairID, fixture.pairedCaseChecksum,
	); err != nil {
		t.Fatalf("insert evaluation preflight case pair: %v", err)
	}
	if _, err := transaction.Exec(ctx, `
		INSERT INTO evaluation_case_expected_evidence (
			id, dataset_revision_id, corpus_id, evaluation_case_id, source_alias, ordinal, display_locator, canonical_locator, required_propositions
		) VALUES ($1, $2, $3, $4, 'official-source', 1, 'Article 1', $5, '["synthetic proposition"]'::jsonb)`,
		fixture.expectedEvidenceID, fixture.revisionID, fixture.corpusID, fixture.caseID, locator,
	); err != nil {
		t.Fatalf("insert evaluation preflight evidence: %v", err)
	}
	return fixture
}

func insertEvaluationPreflightPublication(
	t *testing.T,
	ctx context.Context,
	transaction pgx.Tx,
	fixture evaluationPreflightFixture,
	decision string,
	state string,
) {
	t.Helper()
	if _, err := transaction.Exec(ctx, `
		INSERT INTO evaluation_dataset_publication (
			id, dataset_revision_id, corpus_id, review_decision, reviewer_identity, publication_state, reviewed_at
		) VALUES ($1, $2, $3, $4, 'integration-test', $5, $6)`,
		uuid.New(), fixture.revisionID, fixture.corpusID, decision, state, time.Date(2026, time.August, 26, 16, 0, 0, 0, time.UTC),
	); err != nil {
		t.Fatalf("insert evaluation preflight publication: %v", err)
	}
}

type evaluationModelSentinel struct{ calls int }
