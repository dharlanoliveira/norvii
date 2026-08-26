//go:build integration

package integration_test

import (
	"testing"

	evaluationpostgres "github.com/dharlanoliveira/norvii/apps/api/internal/evaluation/postgres"
	"github.com/google/uuid"
)

func TestEvaluationDatasetCatalogReadsImmutableManifestBindingAndUnavailableReview(t *testing.T) {
	ctx, connection := openEvaluationSchemaConnection(t)
	transaction := beginEvaluationSchemaTransaction(t, ctx, connection)
	defer rollbackEvaluationSchemaTransaction(t, ctx, transaction)

	fixture := newEvaluationResolutionFixture(evaluationLGPDCorpusID, "20000000-0000-4000-8000-000000000001")
	seedEvaluationResolutionFixture(t, ctx, transaction, &fixture)
	repository := evaluationpostgres.NewRepository(transaction)
	bindEvaluationFixtureSource(t, ctx, repository, fixture)
	if _, err := transaction.Exec(ctx, `
		INSERT INTO evaluation_dataset_publication (
			id, dataset_revision_id, corpus_id, review_decision, reviewer_identity, publication_state
		) VALUES ($1, $2, $3, 'pending', 'integration-test', 'draft')`,
		uuid.New(), fixture.revisionID, fixture.corpusID,
	); err != nil {
		t.Fatalf("insert unavailable dataset review: %v", err)
	}

	entry, err := repository.GetDatasetCatalog(ctx, fixture.revisionID)
	if err != nil {
		t.Fatalf("GetDatasetCatalog() error = %v", err)
	}
	if entry.Revision.ID != fixture.revisionID || entry.Revision.CorpusID != fixture.corpusID || entry.Revision.ManifestSHA256 != fixtureSHA256("manifest-"+fixture.revisionID.String()) ||
		len(entry.Revision.QueryLanguages) != 1 || entry.Revision.QueryLanguages[0] != "en" {
		t.Fatalf("catalog revision = %#v, want immutable manifest and language identity", entry.Revision)
	}
	if entry.Available() || entry.Review == nil || entry.Review.Decision != "pending" || entry.Review.PublicationState != "draft" {
		t.Fatalf("catalog review = %#v, want unavailable pending draft", entry.Review)
	}
	if len(entry.Sources) != 1 || entry.Sources[0].CorpusSourceID == nil || *entry.Sources[0].CorpusSourceID != fixture.sourceID ||
		entry.Sources[0].IssuingAuthority != "Test authority" || entry.Sources[0].AuthorityRole != "statute" {
		t.Fatalf("catalog sources = %#v, want bound source authority", entry.Sources)
	}

	entries, err := repository.ListDatasetCatalog(ctx)
	if err != nil {
		t.Fatalf("ListDatasetCatalog() error = %v", err)
	}
	if len(entries) != 1 || entries[0].Revision.CorpusID != fixture.corpusID || entries[0].Revision.ID != fixture.revisionID {
		t.Fatalf("catalog list = %#v, want one corpus-bound revision", entries)
	}
}
