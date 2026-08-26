//go:build integration

package integration_test

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"testing"
	"time"

	evaluationpostgres "github.com/dharlanoliveira/norvii/apps/api/internal/evaluation/postgres"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

func TestEvaluationLocatorResolutionIsBoundToOneCorpusSnapshot(t *testing.T) {
	ctx, connection := openEvaluationSchemaConnection(t)
	transaction := beginEvaluationSchemaTransaction(t, ctx, connection)
	defer rollbackEvaluationSchemaTransaction(t, ctx, transaction)

	repository := evaluationpostgres.NewRepository(transaction)
	fixtures := []evaluationResolutionFixture{
		newEvaluationResolutionFixture(evaluationLGPDCorpusID, "20000000-0000-4000-8000-000000000001"),
		newEvaluationResolutionFixture("10000000-0000-4000-8000-000000000003", "20000000-0000-4000-8000-000000000003"),
		newEvaluationResolutionFixture("10000000-0000-4000-8000-000000000004", "20000000-0000-4000-8000-000000000008"),
	}
	for index := range fixtures {
		seedEvaluationResolutionFixture(t, ctx, transaction, &fixtures[index])
		bindEvaluationFixtureSource(t, ctx, repository, fixtures[index])
	}

	for index, fixture := range fixtures {
		t.Run(fmt.Sprintf("fixture corpus %d resolves only its named snapshot", index+1), func(t *testing.T) {
			resolved, err := repository.ResolveLocator(ctx, fixture.request())
			if err != nil {
				t.Fatalf("ResolveLocator() error = %v", err)
			}
			assertResolvedEvaluationFixture(t, resolved, fixture)

			foreignSnapshot := fixtures[(index+1)%len(fixtures)].snapshotID
			foreignRequest := fixture.request()
			foreignRequest.SnapshotID = foreignSnapshot
			if _, err := repository.ResolveLocator(ctx, foreignRequest); !errors.Is(err, evaluationpostgres.ErrLocatorNotFound) {
				t.Fatalf("ResolveLocator(foreign snapshot) error = %v, want ErrLocatorNotFound", err)
			}

			nonmemberRequest := fixture.request()
			nonmemberRequest.SnapshotID = fixture.nonmemberSnapshotID
			if _, err := repository.ResolveLocator(ctx, nonmemberRequest); !errors.Is(err, evaluationpostgres.ErrLocatorNotFound) {
				t.Fatalf("ResolveLocator(nonmember snapshot) error = %v, want ErrLocatorNotFound", err)
			}

			absentRequest := fixture.request()
			absentRequest.CanonicalLocator = "article:404"
			absentRequest.DisplayLocator = "Article 404"
			if _, err := repository.ResolveLocator(ctx, absentRequest); !errors.Is(err, evaluationpostgres.ErrLocatorNotFound) {
				t.Fatalf("ResolveLocator(absent canonical locator) error = %v, want ErrLocatorNotFound", err)
			}
		})
	}

	t.Run("binding rejects foreign source, duplicate alias, and unknown alias", func(t *testing.T) {
		fixture := fixtures[0]
		foreign := fixture.binding()
		foreign.CorpusSourceID = fixtures[1].sourceID
		if _, err := repository.BindSource(ctx, foreign); !errors.Is(err, evaluationpostgres.ErrSourceAlreadyBound) {
			t.Fatalf("BindSource(already bound alias) error = %v, want ErrSourceAlreadyBound", err)
		}

		unboundFixture := newEvaluationResolutionFixture(evaluationLGPDCorpusID, "20000000-0000-4000-8000-000000000001")
		unboundFixture.sourceAlias = "unbound-source"
		seedEvaluationDatasetRequirement(t, ctx, transaction, unboundFixture)
		foreign = unboundFixture.binding()
		foreign.CorpusSourceID = fixtures[1].sourceID
		if _, err := repository.BindSource(ctx, foreign); !errors.Is(err, evaluationpostgres.ErrCorpusSourceNotFound) {
			t.Fatalf("BindSource(foreign corpus source) error = %v, want ErrCorpusSourceNotFound", err)
		}

		unknown := fixture.binding()
		unknown.SourceAlias = "missing-source"
		if _, err := repository.BindSource(ctx, unknown); !errors.Is(err, evaluationpostgres.ErrSourceRequirementNotFound) {
			t.Fatalf("BindSource(unknown alias) error = %v, want ErrSourceRequirementNotFound", err)
		}
	})

	t.Run("resolution rejects compound canonical locators", func(t *testing.T) {
		request := fixtures[0].request()
		request.CanonicalLocator = "article:1,article:2"
		if _, err := repository.ResolveLocator(ctx, request); !errors.Is(err, evaluationpostgres.ErrInvalidInput) {
			t.Fatalf("ResolveLocator(compound locator) error = %v, want ErrInvalidInput", err)
		}
	})
}

type evaluationResolutionFixture struct {
	corpusID            uuid.UUID
	sourceID            uuid.UUID
	revisionID          uuid.UUID
	snapshotID          uuid.UUID
	nonmemberSnapshotID uuid.UUID
	sourceRevisionID    uuid.UUID
	documentID          uuid.UUID
	unitID              uuid.UUID
	sourceAlias         string
	contentSHA256       string
	documentSHA256      string
	unitSHA256          string
}

func newEvaluationResolutionFixture(corpusIDText, sourceIDText string) evaluationResolutionFixture {
	return evaluationResolutionFixture{
		corpusID:            uuid.MustParse(corpusIDText),
		sourceID:            uuid.MustParse(sourceIDText),
		revisionID:          uuid.New(),
		snapshotID:          uuid.New(),
		nonmemberSnapshotID: uuid.New(),
		sourceRevisionID:    uuid.New(),
		documentID:          uuid.New(),
		unitID:              uuid.New(),
		sourceAlias:         "official-source",
		contentSHA256:       fixtureSHA256("content-" + uuid.NewString()),
		documentSHA256:      fixtureSHA256("document-" + uuid.NewString()),
		unitSHA256:          fixtureSHA256("unit-" + uuid.NewString()),
	}
}

func (fixture evaluationResolutionFixture) binding() evaluationpostgres.SourceBinding {
	return evaluationpostgres.SourceBinding{
		DatasetRevisionID: fixture.revisionID,
		CorpusID:          fixture.corpusID,
		SourceAlias:       fixture.sourceAlias,
		CorpusSourceID:    fixture.sourceID,
	}
}

func (fixture evaluationResolutionFixture) request() evaluationpostgres.LocatorRequest {
	return evaluationpostgres.LocatorRequest{
		DatasetRevisionID: fixture.revisionID,
		CorpusID:          fixture.corpusID,
		SnapshotID:        fixture.snapshotID,
		SourceAlias:       fixture.sourceAlias,
		CanonicalLocator:  "article:1",
		DisplayLocator:    "Article 1",
	}
}

func seedEvaluationResolutionFixture(
	t *testing.T,
	ctx context.Context,
	transaction pgx.Tx,
	fixture *evaluationResolutionFixture,
) {
	t.Helper()
	seedEvaluationDatasetRequirement(t, ctx, transaction, *fixture)

	workID, attemptID := uuid.New(), uuid.New()
	now := time.Date(2026, time.August, 26, 15, 0, 0, 0, time.UTC)
	if _, err := transaction.Exec(ctx, `
		INSERT INTO ingestion_work (id, source_id, corpus_id, reason, status)
		VALUES ($1, $2, $3, 'reprocess', 'succeeded')`,
		workID, fixture.sourceID, fixture.corpusID,
	); err != nil {
		t.Fatalf("insert resolution work: %v", err)
	}
	if _, err := transaction.Exec(ctx, `
		INSERT INTO processing_attempts (
			id, work_id, source_id, corpus_id, attempt_number, pipeline_version, status,
			lease_token, worker_id, started_at, finished_at
		) VALUES ($1, $2, $3, $4, 1, 'evaluation-resolution-test', 'succeeded',
			$5, 'integration-test', $6, $6)`,
		attemptID, workID, fixture.sourceID, fixture.corpusID, uuid.New(), now,
	); err != nil {
		t.Fatalf("insert resolution processing attempt: %v", err)
	}
	if _, err := transaction.Exec(ctx, `
		INSERT INTO source_revisions (
			id, source_id, corpus_id, attempt_id, content_sha256, captured_at, media_type,
			byte_size, pipeline_version, final_url, extracted_content_sha256
		) VALUES ($1, $2, $3, $4, $5, $6, 'text/html', 128, 'evaluation-resolution-test',
			'https://example.test/evaluation', $5)`,
		fixture.sourceRevisionID, fixture.sourceID, fixture.corpusID, attemptID, fixture.contentSHA256, now,
	); err != nil {
		t.Fatalf("insert resolution source revision: %v", err)
	}
	if _, err := transaction.Exec(ctx, `
		INSERT INTO document_versions (
			id, source_revision_id, source_id, corpus_id, pipeline_version, text_content,
			text_sha256, published_at
		) VALUES ($1, $2, $3, $4, 'evaluation-resolution-test', 'Article 1 legal text.', $5, $6)`,
		fixture.documentID, fixture.sourceRevisionID, fixture.sourceID, fixture.corpusID, fixture.documentSHA256, now,
	); err != nil {
		t.Fatalf("insert resolution document: %v", err)
	}
	if _, err := transaction.Exec(ctx, `
		INSERT INTO document_units (
			id, document_id, kind, ordinal, start_offset, end_offset, locator,
			canonical_locator, content_sha256
		) VALUES ($1, $2, 'article', 1, 0, 21, 'article-1', 'article:1', $3)`,
		fixture.unitID, fixture.documentID, fixture.unitSHA256,
	); err != nil {
		t.Fatalf("insert resolution legal unit: %v", err)
	}
	for _, snapshotID := range []uuid.UUID{fixture.snapshotID, fixture.nonmemberSnapshotID} {
		if _, err := transaction.Exec(ctx, `
			INSERT INTO corpus_snapshots (id, corpus_id, manifest_sha256, created_by)
			VALUES ($1, $2, $3, 'integration-test')`,
			snapshotID, fixture.corpusID, fixtureSHA256(snapshotID.String()),
		); err != nil {
			t.Fatalf("insert resolution snapshot: %v", err)
		}
	}
	if _, err := transaction.Exec(ctx, `
		INSERT INTO corpus_snapshot_documents (
			snapshot_id, corpus_id, source_id, source_revision_id, document_id, official_origin,
			captured_at, content_sha256
		) VALUES ($1, $2, $3, $4, $5, 'official', $6, $7)`,
		fixture.snapshotID, fixture.corpusID, fixture.sourceID, fixture.sourceRevisionID,
		fixture.documentID, now, fixture.contentSHA256,
	); err != nil {
		t.Fatalf("insert resolution snapshot member: %v", err)
	}
}

func seedEvaluationDatasetRequirement(
	t *testing.T,
	ctx context.Context,
	transaction pgx.Tx,
	fixture evaluationResolutionFixture,
) {
	t.Helper()
	if _, err := transaction.Exec(ctx, `
		INSERT INTO evaluation_dataset_revision (
			id, corpus_id, dataset_key, semantic_revision, jurisdiction, manifest_sha256,
			jsonl_sha256, content_sha256, manifest_path, jsonl_path, declared_snapshot_date,
			query_languages, authoritative_evidence_language, importer_version
		) VALUES ($1, $2, $3, 'v1', 'Test', $4, $5, $6, 'fixtures/manifest.json',
			'fixtures/cases.jsonl', $7, ARRAY['en'], 'en', 'integration-test')`,
		fixture.revisionID, fixture.corpusID, "resolution-"+fixture.revisionID.String(),
		fixtureSHA256("manifest-"+fixture.revisionID.String()), fixtureSHA256("jsonl-"+fixture.revisionID.String()),
		fixtureSHA256("dataset-"+fixture.revisionID.String()), time.Date(2026, time.August, 26, 0, 0, 0, 0, time.UTC),
	); err != nil {
		t.Fatalf("insert resolution dataset revision: %v", err)
	}
	if _, err := transaction.Exec(ctx, `
		INSERT INTO evaluation_dataset_source (
			id, dataset_revision_id, corpus_id, source_alias, title, official_url,
			issuing_authority, document_type, authority_role
		) VALUES ($1, $2, $3, $4, 'Synthetic source', 'https://example.test/evaluation',
			'Test authority', 'statute', 'statute')`,
		uuid.New(), fixture.revisionID, fixture.corpusID, fixture.sourceAlias,
	); err != nil {
		t.Fatalf("insert resolution dataset source requirement: %v", err)
	}
}

func bindEvaluationFixtureSource(
	t *testing.T,
	ctx context.Context,
	repository *evaluationpostgres.Repository,
	fixture evaluationResolutionFixture,
) {
	t.Helper()
	persisted, err := repository.BindSource(ctx, fixture.binding())
	if err != nil {
		t.Fatalf("BindSource() error = %v", err)
	}
	if persisted != fixture.binding() {
		t.Fatalf("BindSource() = %+v, want %+v", persisted, fixture.binding())
	}
}

func assertResolvedEvaluationFixture(
	t *testing.T,
	resolved evaluationpostgres.ResolvedLocator,
	fixture evaluationResolutionFixture,
) {
	t.Helper()
	if resolved.CorpusID != fixture.corpusID || resolved.SnapshotID != fixture.snapshotID ||
		resolved.SourceID != fixture.sourceID || resolved.SourceRevisionID != fixture.sourceRevisionID ||
		resolved.DocumentID != fixture.documentID || resolved.UnitID != fixture.unitID {
		t.Fatalf("resolved identity = %+v, want fixture-owned source, revision, document, and unit", resolved)
	}
	if resolved.CanonicalLocator != "article:1" || resolved.DisplayLocator != "Article 1" ||
		resolved.UnitStartOffset != 0 || resolved.UnitEndOffset != 21 {
		t.Fatalf("resolved locator = %+v, want atomic canonical and requested display locator", resolved)
	}
	if resolved.ContentProvenance.ContentSHA256 != fixture.contentSHA256 ||
		resolved.ContentProvenance.DocumentTextSHA256 != fixture.documentSHA256 ||
		resolved.ContentProvenance.UnitContentSHA256 != fixture.unitSHA256 {
		t.Fatalf("resolved content provenance = %+v, want fixture hashes", resolved.ContentProvenance)
	}
}

func fixtureSHA256(value string) string {
	sum := sha256.Sum256([]byte(value))
	return hex.EncodeToString(sum[:])
}
