//go:build integration

package integration_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/dharlanoliveira/norvii/apps/api/internal/suggestions/domain"
	suggestionspostgres "github.com/dharlanoliveira/norvii/apps/api/internal/suggestions/postgres"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

func TestOpeningSuggestionRepositoryUsesOnlyTheCurrentCorpusRelease(t *testing.T) {
	ctx, connection := openEvaluationSchemaConnection(t)
	transaction := beginEvaluationSchemaTransaction(t, ctx, connection)
	defer rollbackEvaluationSchemaTransaction(t, ctx, transaction)

	projection := insertSuggestionRepositoryFixture(t, ctx, transaction)
	if _, err := transaction.Exec(ctx, "SET CONSTRAINTS ALL IMMEDIATE"); err != nil {
		t.Fatalf("validate opening-suggestion fixture: %v", err)
	}
	if _, err := transaction.Exec(ctx, "SET CONSTRAINTS ALL DEFERRED"); err != nil {
		t.Fatalf("defer opening-suggestion projection constraints: %v", err)
	}

	repository := suggestionspostgres.NewRepository(transaction)
	if err := repository.AppendProjection(ctx, projection); err != nil {
		t.Fatalf("AppendProjection() error = %v", err)
	}
	if _, err := transaction.Exec(ctx, "SET CONSTRAINTS ALL IMMEDIATE"); err != nil {
		t.Fatalf("validate appended opening suggestions: %v", err)
	}

	english, err := repository.Read(ctx, projection.Set.CorpusID, domain.QueryLanguageEnglish)
	if err != nil {
		t.Fatalf("Read(en) error = %v", err)
	}
	assertSuggestionRead(t, english, projection.Set.SnapshotID, 5, "Synthetic English question")

	portuguese, err := repository.Read(ctx, projection.Set.CorpusID, domain.QueryLanguagePortuguese)
	if err != nil {
		t.Fatalf("Read(pt) error = %v", err)
	}
	assertSuggestionRead(t, portuguese, projection.Set.SnapshotID, 5, "Synthetic Portuguese question")

	foreignProjection := projection
	foreignProjection.Items = append([]domain.OpeningSuggestionItem(nil), projection.Items...)
	foreignProjection.Items[0].CorpusID = uuid.New()
	if err := repository.AppendProjection(ctx, foreignProjection); !errors.Is(err, domain.ErrCorpusMismatch) {
		t.Fatalf("AppendProjection(foreign corpus) error = %v, want %v", err, domain.ErrCorpusMismatch)
	}

	staleSnapshotID := uuid.New()
	if _, err := transaction.Exec(ctx, `
		INSERT INTO corpus_snapshots (id, corpus_id, manifest_sha256, created_by)
		VALUES ($1, $2, $3, 'integration-test')`, staleSnapshotID, projection.Set.CorpusID, hashOf("f")); err != nil {
		t.Fatalf("insert later snapshot: %v", err)
	}
	if _, err := transaction.Exec(ctx, `
		INSERT INTO corpus_snapshot_releases (corpus_id, snapshot_id, version, activated_at)
		VALUES ($1, $2, 1, now())
		ON CONFLICT (corpus_id) DO UPDATE
		SET snapshot_id = EXCLUDED.snapshot_id,
		    version = corpus_snapshot_releases.version + 1,
		    activated_at = EXCLUDED.activated_at`, projection.Set.CorpusID, staleSnapshotID); err != nil {
		t.Fatalf("activate later snapshot: %v", err)
	}
	if err := repository.AppendProjection(ctx, projection); !errors.Is(err, domain.ErrSnapshotMismatch) {
		t.Fatalf("AppendProjection(stale release) error = %v, want %v", err, domain.ErrSnapshotMismatch)
	}

	stale, err := repository.Read(ctx, projection.Set.CorpusID, domain.QueryLanguageEnglish)
	if err != nil {
		t.Fatalf("Read(stale projection) error = %v", err)
	}
	if stale.ActiveSnapshot == nil || stale.ActiveSnapshot.ID != staleSnapshotID || len(stale.Suggestions) != 0 {
		t.Fatalf("Read(stale projection) = %#v, want later release identity and empty suggestions", stale)
	}

	missing, err := repository.Read(ctx, uuid.New(), domain.QueryLanguageEnglish)
	if err != nil {
		t.Fatalf("Read(missing corpus) error = %v", err)
	}
	if missing.ActiveSnapshot != nil || len(missing.Suggestions) != 0 {
		t.Fatalf("Read(missing corpus) = %#v, want empty normal outcome", missing)
	}
}

func TestOpeningSuggestionRepositoryHidesProjectionAfterLatestPublicationBecomesUnavailable(t *testing.T) {
	ctx, connection := openEvaluationSchemaConnection(t)
	transaction := beginEvaluationSchemaTransaction(t, ctx, connection)
	defer rollbackEvaluationSchemaTransaction(t, ctx, transaction)

	availableReviewedAt := time.Date(2026, time.August, 26, 12, 0, 0, 0, time.UTC)
	withdrawnReviewedAt := availableReviewedAt.Add(time.Second)
	availablePublicationID := uuid.MustParse("ffffffff-ffff-4fff-8fff-ffffffffffff")
	withdrawnPublicationID := uuid.MustParse("00000000-0000-4000-8000-000000000001")
	if !withdrawnReviewedAt.After(availableReviewedAt) {
		t.Fatal("withdrawn publication timestamp must be later than the available publication")
	}
	if availablePublicationID.String() <= withdrawnPublicationID.String() {
		t.Fatal("fixture must make UUID tie-breaking select the available publication")
	}

	projection := insertSuggestionRepositoryFixtureWithPublication(
		t,
		ctx,
		transaction,
		availablePublicationID,
		availableReviewedAt,
	)
	if _, err := transaction.Exec(ctx, "SET CONSTRAINTS ALL IMMEDIATE"); err != nil {
		t.Fatalf("validate opening-suggestion fixture: %v", err)
	}
	if _, err := transaction.Exec(ctx, "SET CONSTRAINTS ALL DEFERRED"); err != nil {
		t.Fatalf("defer opening-suggestion projection constraints: %v", err)
	}

	repository := suggestionspostgres.NewRepository(transaction)
	if err := repository.AppendProjection(ctx, projection); err != nil {
		t.Fatalf("AppendProjection() error = %v", err)
	}
	if _, err := transaction.Exec(ctx, "SET CONSTRAINTS ALL IMMEDIATE"); err != nil {
		t.Fatalf("validate appended opening suggestions: %v", err)
	}

	if _, err := transaction.Exec(ctx, `
		INSERT INTO evaluation_dataset_publication (
			id, dataset_revision_id, corpus_id, review_decision, reviewer_identity, publication_state, reviewed_at
		) VALUES ($1, $2, $3, 'approved', 'integration-test', 'withdrawn', $4)`,
		withdrawnPublicationID, projection.Set.DatasetRevisionID, projection.Set.CorpusID, withdrawnReviewedAt); err != nil {
		t.Fatalf("insert later unavailable evaluation dataset publication: %v", err)
	}

	result, err := repository.Read(ctx, projection.Set.CorpusID, domain.QueryLanguageEnglish)
	if err != nil {
		t.Fatalf("Read() after unavailable publication error = %v", err)
	}
	if result.ActiveSnapshot == nil || result.ActiveSnapshot.ID != projection.Set.SnapshotID {
		t.Fatalf("Read() active snapshot = %#v, want %s", result.ActiveSnapshot, projection.Set.SnapshotID)
	}
	if len(result.Suggestions) != 0 {
		t.Fatalf("Read() suggestions = %#v, want an empty normal outcome after an unavailable publication", result.Suggestions)
	}
}

func insertSuggestionRepositoryFixture(t *testing.T, ctx context.Context, transaction pgx.Tx) domain.Projection {
	t.Helper()
	return insertSuggestionRepositoryFixtureWithPublication(t, ctx, transaction, uuid.New(), time.Now().UTC())
}

func insertSuggestionRepositoryFixtureWithPublication(
	t *testing.T,
	ctx context.Context,
	transaction pgx.Tx,
	availablePublicationID uuid.UUID,
	availableReviewedAt time.Time,
) domain.Projection {
	t.Helper()
	corpusID := uuid.MustParse(evaluationLGPDCorpusID)
	revisionID := uuid.New()
	snapshotID := uuid.New()
	setID := uuid.New()
	insertEvaluationRevision(t, ctx, transaction, revisionID, evaluationLGPDCorpusID)

	projection := domain.Projection{
		Set: domain.OpeningSuggestionSet{
			ID:                     setID,
			CorpusID:               corpusID,
			SnapshotID:             snapshotID,
			SnapshotManifestSHA256: domain.SHA256(hashOf("e")),
			DatasetRevisionID:      revisionID,
			DatasetContentSHA256:   domain.SHA256(hashOf("c")),
			SelectionPolicyVersion: "opening-suggestions-v1",
			PublishedBy:            "integration-test",
		},
		Items: make([]domain.OpeningSuggestionItem, 0, 10),
	}
	for rank := 1; rank <= 5; rank++ {
		englishCaseID := uuid.New()
		portugueseCaseID := uuid.New()
		insertSuggestionCasePair(t, ctx, transaction, revisionID, englishCaseID, portugueseCaseID, rank*2)
		insertEvaluationStarterSelection(t, ctx, transaction, uuid.New().String(), revisionID, englishCaseID, rank, "en", hashOf("a"))
		insertEvaluationStarterSelection(t, ctx, transaction, uuid.New().String(), revisionID, portugueseCaseID, rank, "pt", hashOf("b"))
		projection.Items = append(projection.Items,
			domain.OpeningSuggestionItem{
				SuggestionSetID: setID, CorpusID: corpusID, DatasetRevisionID: revisionID, Rank: rank, CaseID: englishCaseID,
				CaseChecksum: domain.SHA256(hashOf("a")), QueryLanguage: domain.QueryLanguageEnglish,
				ExternalCaseID: "case-" + englishCaseID.String() + "-en", Question: "Synthetic English question",
			},
			domain.OpeningSuggestionItem{
				SuggestionSetID: setID, CorpusID: corpusID, DatasetRevisionID: revisionID, Rank: rank, CaseID: portugueseCaseID,
				CaseChecksum: domain.SHA256(hashOf("b")), QueryLanguage: domain.QueryLanguagePortuguese,
				ExternalCaseID: "case-" + portugueseCaseID.String() + "-pt", Question: "Synthetic Portuguese question",
			},
		)
	}
	if _, err := transaction.Exec(ctx, `
		INSERT INTO evaluation_dataset_publication (
			id, dataset_revision_id, corpus_id, review_decision, reviewer_identity, publication_state, reviewed_at
		) VALUES ($1, $2, $3, 'approved', 'integration-test', 'available', $4)`,
		availablePublicationID, revisionID, corpusID, availableReviewedAt); err != nil {
		t.Fatalf("insert available evaluation dataset publication: %v", err)
	}
	if _, err := transaction.Exec(ctx, `
		INSERT INTO corpus_snapshots (id, corpus_id, manifest_sha256, created_by)
		VALUES ($1, $2, $3, 'integration-test')`, snapshotID, corpusID, hashOf("e")); err != nil {
		t.Fatalf("insert opening-suggestion snapshot: %v", err)
	}
	if _, err := transaction.Exec(ctx, `
		INSERT INTO corpus_snapshot_releases (corpus_id, snapshot_id, version, activated_at)
		VALUES ($1, $2, 1, now())
		ON CONFLICT (corpus_id) DO UPDATE
		SET snapshot_id = EXCLUDED.snapshot_id,
		    version = corpus_snapshot_releases.version + 1,
		    activated_at = EXCLUDED.activated_at`, corpusID, snapshotID); err != nil {
		t.Fatalf("activate opening-suggestion snapshot: %v", err)
	}
	return projection
}

func assertSuggestionRead(
	t *testing.T,
	result suggestionspostgres.ReadResult,
	wantSnapshotID uuid.UUID,
	wantSuggestions int,
	wantQuestion string,
) {
	t.Helper()
	if result.ActiveSnapshot == nil || result.ActiveSnapshot.ID != wantSnapshotID {
		t.Fatalf("Read() active snapshot = %#v, want %s", result.ActiveSnapshot, wantSnapshotID)
	}
	if len(result.Suggestions) != wantSuggestions {
		t.Fatalf("Read() suggestion count = %d, want %d", len(result.Suggestions), wantSuggestions)
	}
	for index, suggestion := range result.Suggestions {
		if suggestion.Rank != index+1 || suggestion.CaseID == "" || suggestion.Question != wantQuestion {
			t.Fatalf("Read() suggestion %d = %#v, want a rank-ordered safe question", index, suggestion)
		}
	}
}

func insertSuggestionCasePair(
	t *testing.T,
	ctx context.Context,
	transaction pgx.Tx,
	revisionID, englishCaseID, portugueseCaseID uuid.UUID,
	firstPosition int,
) {
	t.Helper()
	englishExternalID := "case-" + englishCaseID.String() + "-en"
	portugueseExternalID := "case-" + portugueseCaseID.String() + "-pt"
	for _, fixture := range []struct {
		id            uuid.UUID
		externalID    string
		reciprocalID  string
		queryLanguage string
		assetLanguage string
		position      int
		question      string
		checksum      string
	}{
		{englishCaseID, englishExternalID, portugueseExternalID, "en", "en", firstPosition, "Synthetic English question", hashOf("a")},
		{portugueseCaseID, portugueseExternalID, englishExternalID, "pt", "pt-BR", firstPosition + 1, "Synthetic Portuguese question", hashOf("b")},
	} {
		if _, err := transaction.Exec(ctx, `
			INSERT INTO evaluation_dataset_case (
				id, dataset_revision_id, corpus_id, position, external_case_id, query_language,
				asset_language, question, reference_answer, category, authoritative_evidence_language,
				reciprocal_case_external_id, case_checksum
			) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, 'Synthetic answer',
				'fixture', 'pt-BR', $9, $10)`,
			fixture.id, revisionID, uuid.MustParse(evaluationLGPDCorpusID), fixture.position, fixture.externalID,
			fixture.queryLanguage, fixture.assetLanguage, fixture.question, fixture.reciprocalID, fixture.checksum,
		); err != nil {
			t.Fatalf("insert opening-suggestion evaluation case %s: %v", fixture.externalID, err)
		}
	}
}
