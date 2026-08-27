package postgres

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/dharlanoliveira/norvii/apps/api/internal/suggestions/domain"
	"github.com/google/uuid"
	"github.com/pashagolub/pgxmock/v4"
)

func TestAppendProjectionLocksAndRechecksTheActiveReleaseBeforeAppending(t *testing.T) {
	t.Parallel()
	pool := newPool(t)
	projection := validProjection()
	pool.ExpectBegin()
	pool.ExpectQuery("SELECT release.snapshot_id").
		WithArgs(projection.Set.CorpusID).
		WillReturnRows(pool.NewRows([]string{"snapshot_id", "corpus_id", "manifest_sha256"}).
			AddRow(projection.Set.SnapshotID, projection.Set.CorpusID, string(projection.Set.SnapshotManifestSHA256)))
	expectProjectionInserts(pool, projection)
	pool.ExpectCommit()

	if err := NewRepository(pool).AppendProjection(context.Background(), projection); err != nil {
		t.Fatalf("AppendProjection() error = %v", err)
	}
	if err := pool.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet PostgreSQL expectations: %v", err)
	}
}

func TestAppendProjectionRejectsAStaleActiveReleaseWithoutWriting(t *testing.T) {
	t.Parallel()
	pool := newPool(t)
	projection := validProjection()
	pool.ExpectBegin()
	pool.ExpectQuery("SELECT release.snapshot_id").
		WithArgs(projection.Set.CorpusID).
		WillReturnRows(pool.NewRows([]string{"snapshot_id", "corpus_id", "manifest_sha256"}).
			AddRow(uuid.New(), projection.Set.CorpusID, string(projection.Set.SnapshotManifestSHA256)))
	pool.ExpectRollback()

	err := NewRepository(pool).AppendProjection(context.Background(), projection)
	if !errors.Is(err, domain.ErrSnapshotMismatch) {
		t.Fatalf("AppendProjection() error = %v, want %v", err, domain.ErrSnapshotMismatch)
	}
	if err := pool.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet PostgreSQL expectations: %v", err)
	}
}

func TestAppendProjectionRejectsCrossCorpusItemsBeforeOpeningATransaction(t *testing.T) {
	t.Parallel()
	pool := newPool(t)
	projection := validProjection()
	projection.Items[0].CorpusID = uuid.New()

	err := NewRepository(pool).AppendProjection(context.Background(), projection)
	if !errors.Is(err, domain.ErrCorpusMismatch) {
		t.Fatalf("AppendProjection() error = %v, want %v", err, domain.ErrCorpusMismatch)
	}
	if err := pool.ExpectationsWereMet(); err != nil {
		t.Fatalf("unexpected PostgreSQL operation: %v", err)
	}
}

func TestReadReturnsOnlyRequestedLanguageInRankOrder(t *testing.T) {
	t.Parallel()
	pool := newPool(t)
	corpusID := uuid.New()
	snapshotID := uuid.New()
	manifest := checksum("e")
	pool.ExpectQuery("WITH active_release").
		WithArgs(corpusID, "en").
		WillReturnRows(pool.NewRows([]string{"snapshot_id", "manifest_sha256", "rank", "external_case_id", "question"}).
			AddRow(snapshotID, manifest, 1, "case-001-en", "First synthetic question?").
			AddRow(snapshotID, manifest, 2, "case-002-en", "Second synthetic question?"))

	result, err := NewRepository(pool).Read(context.Background(), corpusID, domain.QueryLanguageEnglish)
	if err != nil {
		t.Fatalf("Read() error = %v", err)
	}
	if result.ActiveSnapshot == nil || result.ActiveSnapshot.ID != snapshotID || result.ActiveSnapshot.ManifestSHA256 != domain.SHA256(manifest) {
		t.Fatalf("Read() active snapshot = %#v, want %s / %s", result.ActiveSnapshot, snapshotID, manifest)
	}
	if len(result.Suggestions) != 2 || result.Suggestions[0].Rank != 1 || result.Suggestions[1].Rank != 2 ||
		result.Suggestions[0].CaseID != "case-001-en" || result.Suggestions[1].CaseID != "case-002-en" {
		t.Fatalf("Read() suggestions = %#v, want rank-ordered English language members", result.Suggestions)
	}
	if err := pool.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet PostgreSQL expectations: %v", err)
	}
}

func TestReadReturnsEmptySuggestionsForStaleOrUnavailableStates(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name         string
		rows         *pgxmock.Rows
		wantSnapshot bool
	}{
		{
			name: "stale projection after a later release",
			rows: pgxmock.NewRows([]string{"snapshot_id", "manifest_sha256", "rank", "external_case_id", "question"}).
				AddRow(uuid.New(), checksum("e"), 0, "", ""),
			wantSnapshot: true,
		},
		{
			name:         "disabled or missing corpus has no active release",
			rows:         pgxmock.NewRows([]string{"snapshot_id", "manifest_sha256", "rank", "external_case_id", "question"}),
			wantSnapshot: false,
		},
	}
	for _, testCase := range tests {
		testCase := testCase
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			pool := newPool(t)
			corpusID := uuid.New()
			pool.ExpectQuery("WITH active_release").WithArgs(corpusID, "pt").WillReturnRows(testCase.rows)

			result, err := NewRepository(pool).Read(context.Background(), corpusID, domain.QueryLanguagePortuguese)
			if err != nil {
				t.Fatalf("Read() error = %v", err)
			}
			if (result.ActiveSnapshot != nil) != testCase.wantSnapshot {
				t.Fatalf("Read() active snapshot = %#v, want present=%t", result.ActiveSnapshot, testCase.wantSnapshot)
			}
			if len(result.Suggestions) != 0 {
				t.Fatalf("Read() suggestions = %#v, want an empty normal outcome", result.Suggestions)
			}
			if err := pool.ExpectationsWereMet(); err != nil {
				t.Fatalf("unmet PostgreSQL expectations: %v", err)
			}
		})
	}
}

func TestReadRequiresTheLatestDatasetPublicationToBeApprovedAndAvailable(t *testing.T) {
	t.Parallel()
	pool := newPool(t)
	corpusID := uuid.New()
	snapshotID := uuid.New()
	manifest := checksum("e")
	pool.ExpectQuery("WITH active_release").
		WithArgs(corpusID, "en").
		WillReturnRows(pool.NewRows([]string{"snapshot_id", "manifest_sha256", "rank", "external_case_id", "question"}).
			AddRow(snapshotID, manifest, 0, "", ""))

	result, err := NewRepository(pool).Read(context.Background(), corpusID, domain.QueryLanguageEnglish)
	if err != nil {
		t.Fatalf("Read() error = %v", err)
	}
	if result.ActiveSnapshot == nil || result.ActiveSnapshot.ID != snapshotID {
		t.Fatalf("Read() active snapshot = %#v, want %s", result.ActiveSnapshot, snapshotID)
	}
	if len(result.Suggestions) != 0 {
		t.Fatalf("Read() suggestions = %#v, want an empty normal outcome", result.Suggestions)
	}
	if err := pool.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet PostgreSQL expectations: %v", err)
	}
}

func TestReadRejectsUnsupportedLanguage(t *testing.T) {
	t.Parallel()
	pool := newPool(t)
	_, err := NewRepository(pool).Read(context.Background(), uuid.New(), domain.QueryLanguage("fr"))
	if !errors.Is(err, domain.ErrInvalidInput) {
		t.Fatalf("Read() error = %v, want %v", err, domain.ErrInvalidInput)
	}
	if err := pool.ExpectationsWereMet(); err != nil {
		t.Fatalf("unexpected PostgreSQL operation: %v", err)
	}
}

func expectProjectionInserts(pool pgxmock.PgxPoolIface, projection domain.Projection) {
	pool.ExpectExec("INSERT INTO corpus_opening_suggestion_set").
		WithArgs(
			projection.Set.ID, projection.Set.CorpusID, projection.Set.SnapshotID,
			string(projection.Set.SnapshotManifestSHA256), projection.Set.DatasetRevisionID,
			string(projection.Set.DatasetContentSHA256), projection.Set.SelectionPolicyVersion, projection.Set.PublishedBy,
		).
		WillReturnResult(pgxmock.NewResult("INSERT", 1))
	for _, item := range projection.Items {
		pool.ExpectExec("INSERT INTO corpus_opening_suggestion_item").
			WithArgs(
				pgxmock.AnyArg(), item.SuggestionSetID, item.CorpusID, item.DatasetRevisionID, item.Rank, item.CaseID,
				string(item.CaseChecksum), string(item.QueryLanguage), item.Question,
			).
			WillReturnResult(pgxmock.NewResult("INSERT", 1))
	}
}

func validProjection() domain.Projection {
	corpusID := uuid.New()
	setID := uuid.New()
	revisionID := uuid.New()
	projection := domain.Projection{
		Set: domain.OpeningSuggestionSet{
			ID: setID, CorpusID: corpusID, SnapshotID: uuid.New(), SnapshotManifestSHA256: domain.SHA256(checksum("e")),
			DatasetRevisionID: revisionID, DatasetContentSHA256: domain.SHA256(checksum("d")),
			SelectionPolicyVersion: "opening-suggestions-v1", PublishedBy: "integration-test",
		},
		Items: make([]domain.OpeningSuggestionItem, 0, 10),
	}
	for rank := 1; rank <= 5; rank++ {
		for _, language := range []domain.QueryLanguage{domain.QueryLanguageEnglish, domain.QueryLanguagePortuguese} {
			checksumCharacter := "a"
			if language == domain.QueryLanguagePortuguese {
				checksumCharacter = "b"
			}
			projection.Items = append(projection.Items, domain.OpeningSuggestionItem{
				SuggestionSetID: setID, CorpusID: corpusID, DatasetRevisionID: revisionID, Rank: rank, CaseID: uuid.New(),
				CaseChecksum: domain.SHA256(checksum(checksumCharacter)), QueryLanguage: language,
				ExternalCaseID: "case-" + strings.Repeat("x", rank) + "-" + string(language),
				Question:       "Synthetic opening question?",
			})
		}
	}
	return projection
}

func checksum(character string) string { return strings.Repeat(character, 64) }

func newPool(t *testing.T) pgxmock.PgxPoolIface {
	t.Helper()
	pool, err := pgxmock.NewPool()
	if err != nil {
		t.Fatalf("create PostgreSQL mock: %v", err)
	}
	t.Cleanup(func() { pool.Close() })
	return pool
}
