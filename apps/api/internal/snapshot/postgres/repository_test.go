package postgres

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/dharlanoliveira/norvii/apps/api/internal/snapshot/domain"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/pashagolub/pgxmock/v4"
)

func TestManifestSHA256IsIndependentOfMemberInputOrder(t *testing.T) {
	first := member()
	second := member()
	second.SourceID = uuid.New()

	forward := manifestSHA256([]domain.Member{first, second})
	reversed := manifestSHA256([]domain.Member{second, first})

	if forward != reversed {
		t.Fatalf("manifest hashes = %s/%s, want stable ordering", forward, reversed)
	}
}

func TestReplaceMemberKeepsAllOtherSnapshotMembers(t *testing.T) {
	first := member()
	second := member()
	second.SourceID = uuid.New()
	replacement := first
	replacement.DocumentID = uuid.New()

	updated := replaceMember([]domain.Member{first, second}, replacement)

	if len(updated) != 2 || updated[0].DocumentID != replacement.DocumentID || updated[1] != second {
		t.Fatalf("replaceMember() = %+v, want replacement and preserved member", updated)
	}
}

func TestListReturnsEachImmutableManifestWithItsMembers(t *testing.T) {
	t.Parallel()
	pool, err := pgxmock.NewPool()
	if err != nil {
		t.Fatalf("create PostgreSQL mock: %v", err)
	}
	defer pool.Close()

	corpusID := uuid.New()
	snapshotID := uuid.New()
	member := member()
	createdAt := time.Date(2026, time.August, 25, 12, 0, 0, 0, time.UTC)
	manifest := "b" + strings.Repeat("a", 63)
	manifestRows := pool.NewRows([]string{"id", "corpus_id", "manifest_sha256", "created_by", "created_at"}).
		AddRow(snapshotID, corpusID, manifest, "test-maintainer", createdAt)
	pool.ExpectQuery("SELECT id, corpus_id, manifest_sha256, created_by, created_at").
		WithArgs(corpusID).
		WillReturnRows(manifestRows)
	pool.ExpectQuery("SELECT id, corpus_id, manifest_sha256, created_by, created_at").
		WithArgs(corpusID, snapshotID).
		WillReturnRows(pool.NewRows([]string{"id", "corpus_id", "manifest_sha256", "created_by", "created_at"}).
			AddRow(snapshotID, corpusID, manifest, "test-maintainer", createdAt))
	pool.ExpectQuery("SELECT source_id, source_revision_id, document_id, official_origin, captured_at, content_sha256").
		WithArgs(snapshotID).
		WillReturnRows(pool.NewRows([]string{"source_id", "source_revision_id", "document_id", "official_origin", "captured_at", "content_sha256"}).
			AddRow(member.SourceID, member.SourceRevisionID, member.DocumentID, member.OfficialOrigin, member.CapturedAt, member.ContentSHA256))

	snapshots, err := NewRepository(pool).List(context.Background(), corpusID)
	if err != nil {
		t.Fatalf("list immutable manifests: %v", err)
	}
	if len(snapshots) != 1 {
		t.Fatalf("snapshot count = %d, want 1", len(snapshots))
	}
	if snapshots[0].ID != snapshotID || len(snapshots[0].Members) != 1 || snapshots[0].Members[0] != member {
		t.Fatalf("snapshot = %+v, want persisted manifest and member", snapshots[0])
	}
	if err := pool.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet PostgreSQL expectations: %v", err)
	}
}

func TestPublishReplacesTheActiveSourceMemberWithAReadyCandidate(t *testing.T) {
	t.Parallel()
	pool, err := pgxmock.NewPool()
	if err != nil {
		t.Fatalf("create PostgreSQL mock: %v", err)
	}
	defer pool.Close()

	corpusID, sourceID, documentID, snapshotID := uuid.New(), uuid.New(), uuid.New(), uuid.New()
	activeSnapshotID := uuid.New()
	stagedAt := time.Date(2026, time.August, 25, 12, 0, 0, 0, time.UTC)
	activeMember := member()
	activeMember.SourceID = uuid.New()
	candidate := domain.Member{
		SourceID: sourceID, SourceRevisionID: uuid.New(), DocumentID: documentID,
		OfficialOrigin: "https://official.example/law", CapturedAt: stagedAt.Add(-time.Minute),
		ContentSHA256: strings.Repeat("f", 64),
	}
	updatedMembers := []domain.Member{activeMember, candidate}
	manifest := manifestSHA256(updatedMembers)
	pool.ExpectBegin()
	pool.ExpectQuery("SELECT corpus_id, snapshot_id, version, activated_at").
		WithArgs(corpusID).
		WillReturnRows(pool.NewRows([]string{"corpus_id", "snapshot_id", "version", "activated_at"}).
			AddRow(corpusID, activeSnapshotID, 4, stagedAt.Add(-time.Minute)))
	pool.ExpectQuery("SELECT source_id, source_revision_id, document_id, official_origin, captured_at, content_sha256").
		WithArgs(activeSnapshotID).
		WillReturnRows(pool.NewRows([]string{"source_id", "source_revision_id", "document_id", "official_origin", "captured_at", "content_sha256"}).
			AddRow(activeMember.SourceID, activeMember.SourceRevisionID, activeMember.DocumentID, activeMember.OfficialOrigin, activeMember.CapturedAt, activeMember.ContentSHA256))
	pool.ExpectQuery("SELECT d.source_id, d.source_revision_id, d.id").
		WithArgs(corpusID, sourceID, documentID).
		WillReturnRows(pool.NewRows([]string{"source_id", "source_revision_id", "id", "official_origin", "captured_at", "content_sha256"}).
			AddRow(candidate.SourceID, candidate.SourceRevisionID, candidate.DocumentID, candidate.OfficialOrigin, candidate.CapturedAt, candidate.ContentSHA256))
	pool.ExpectQuery("SELECT[[:space:]]*\\(SELECT count\\(\\*\\) FROM document_units").
		WithArgs(documentID).
		WillReturnRows(pool.NewRows([]string{"unit_count", "chunk_count", "ready_chunk_count"}).AddRow(1, 1, 1))
	pool.ExpectQuery("SELECT id, corpus_id, manifest_sha256, created_by, created_at").
		WithArgs(corpusID, manifest).
		WillReturnError(pgx.ErrNoRows)
	pool.ExpectExec("INSERT INTO corpus_snapshots").
		WithArgs(snapshotID, corpusID, manifest, "ingestion-worker", stagedAt).
		WillReturnResult(pgxmock.NewResult("INSERT", 1))
	for _, snapshotMember := range updatedMembers {
		pool.ExpectExec("INSERT INTO corpus_snapshot_documents").
			WithArgs(snapshotID, corpusID, snapshotMember.SourceID, snapshotMember.SourceRevisionID, snapshotMember.DocumentID, snapshotMember.OfficialOrigin, snapshotMember.CapturedAt, snapshotMember.ContentSHA256).
			WillReturnResult(pgxmock.NewResult("INSERT", 1))
	}
	pool.ExpectExec("UPDATE corpus_snapshot_releases").
		WithArgs(corpusID, snapshotID, 5, stagedAt).
		WillReturnResult(pgxmock.NewResult("UPDATE", 1))
	pool.ExpectCommit()

	publication, err := NewRepository(pool).Publish(context.Background(), domain.PublishCommand{
		CorpusID: corpusID, SourceID: sourceID, DocumentID: documentID, ExpectedReleaseVersion: 4,
		SnapshotID: snapshotID, Actor: "ingestion-worker", PublishedAt: stagedAt,
	})
	if err != nil {
		t.Fatalf("publish snapshot: %v", err)
	}
	if !publication.Created || publication.Release.Version != 5 || publication.Release.SnapshotID != snapshotID {
		t.Fatalf("publication = %+v, want updated active release", publication)
	}
	if len(publication.Snapshot.Members) != 2 || publication.Snapshot.Members[1] != candidate {
		t.Fatalf("snapshot members = %+v, want ready candidate in next manifest", publication.Snapshot.Members)
	}
	if err := pool.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet PostgreSQL expectations: %v", err)
	}
}

func TestActiveAndGetReturnCorpusScopedReleaseAndManifest(t *testing.T) {
	t.Parallel()
	pool, err := pgxmock.NewPool()
	if err != nil {
		t.Fatalf("create PostgreSQL mock: %v", err)
	}
	defer pool.Close()

	corpusID := uuid.New()
	snapshotID := uuid.New()
	member := member()
	createdAt := time.Date(2026, time.August, 25, 12, 0, 0, 0, time.UTC)
	manifest := "d" + strings.Repeat("a", 63)
	pool.ExpectQuery("SELECT release.corpus_id, release.snapshot_id, release.version, release.activated_at").
		WithArgs(corpusID).
		WillReturnRows(pool.NewRows([]string{"corpus_id", "snapshot_id", "version", "activated_at"}).
			AddRow(corpusID, snapshotID, 2, createdAt))
	pool.ExpectQuery("SELECT id, corpus_id, manifest_sha256, created_by, created_at").
		WithArgs(corpusID, snapshotID).
		WillReturnRows(pool.NewRows([]string{"id", "corpus_id", "manifest_sha256", "created_by", "created_at"}).
			AddRow(snapshotID, corpusID, manifest, "test-maintainer", createdAt))
	pool.ExpectQuery("SELECT source_id, source_revision_id, document_id, official_origin, captured_at, content_sha256").
		WithArgs(snapshotID).
		WillReturnRows(pool.NewRows([]string{"source_id", "source_revision_id", "document_id", "official_origin", "captured_at", "content_sha256"}).
			AddRow(member.SourceID, member.SourceRevisionID, member.DocumentID, member.OfficialOrigin, member.CapturedAt, member.ContentSHA256))

	repository := NewRepository(pool)
	release, err := repository.Active(context.Background(), corpusID)
	if err != nil {
		t.Fatalf("get active release: %v", err)
	}
	snapshot, err := repository.Get(context.Background(), corpusID, snapshotID)
	if err != nil {
		t.Fatalf("get snapshot manifest: %v", err)
	}
	if release.Version != 2 || snapshot.ID != snapshotID || len(snapshot.Members) != 1 {
		t.Fatalf("release/snapshot = %+v/%+v, want persisted corpus state", release, snapshot)
	}
	if err := pool.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet PostgreSQL expectations: %v", err)
	}
}

func TestActivateRejectsSnapshotWithoutReadyGraphRelease(t *testing.T) {
	t.Parallel()
	pool, err := pgxmock.NewPool()
	if err != nil {
		t.Fatalf("create PostgreSQL mock: %v", err)
	}
	defer pool.Close()

	corpusID := uuid.New()
	snapshotID := uuid.New()
	member := member()
	activatedAt := time.Date(2026, time.August, 25, 12, 0, 0, 0, time.UTC)
	manifest := "c" + strings.Repeat("a", 63)
	pool.ExpectBegin()
	pool.ExpectQuery("SELECT corpus_id, snapshot_id, version, activated_at").
		WithArgs(corpusID).
		WillReturnRows(pool.NewRows([]string{"corpus_id", "snapshot_id", "version", "activated_at"}).
			AddRow(corpusID, uuid.New(), 3, activatedAt.Add(-time.Minute)))
	pool.ExpectQuery("SELECT id, corpus_id, manifest_sha256, created_by, created_at").
		WithArgs(corpusID, snapshotID).
		WillReturnRows(pool.NewRows([]string{"id", "corpus_id", "manifest_sha256", "created_by", "created_at"}).
			AddRow(snapshotID, corpusID, manifest, "test-maintainer", activatedAt))
	pool.ExpectQuery("SELECT source_id, source_revision_id, document_id, official_origin, captured_at, content_sha256").
		WithArgs(snapshotID).
		WillReturnRows(pool.NewRows([]string{"source_id", "source_revision_id", "document_id", "official_origin", "captured_at", "content_sha256"}).
			AddRow(member.SourceID, member.SourceRevisionID, member.DocumentID, member.OfficialOrigin, member.CapturedAt, member.ContentSHA256))
	pool.ExpectQuery("SELECT EXISTS").
		WithArgs(corpusID, snapshotID).
		WillReturnRows(pool.NewRows([]string{"exists"}).AddRow(false))
	pool.ExpectRollback()

	_, err = NewRepository(pool).Activate(context.Background(), domain.ActivationCommand{
		CorpusID: corpusID, SnapshotID: snapshotID, ExpectedReleaseVersion: 3, ActivatedAt: activatedAt,
	})
	if !errors.Is(err, domain.ErrGraphReleaseNotReady) {
		t.Fatalf("activation error = %v, want graph release readiness failure", err)
	}
	if err := pool.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet PostgreSQL expectations: %v", err)
	}
}

func TestActivatePromotesAGraphReadySnapshot(t *testing.T) {
	t.Parallel()
	pool, err := pgxmock.NewPool()
	if err != nil {
		t.Fatalf("create PostgreSQL mock: %v", err)
	}
	defer pool.Close()

	corpusID, snapshotID, activeSnapshotID := uuid.New(), uuid.New(), uuid.New()
	member := member()
	activatedAt := time.Date(2026, time.August, 25, 12, 0, 0, 0, time.UTC)
	manifest := "e" + strings.Repeat("a", 63)
	pool.ExpectBegin()
	pool.ExpectQuery("SELECT corpus_id, snapshot_id, version, activated_at").
		WithArgs(corpusID).
		WillReturnRows(pool.NewRows([]string{"corpus_id", "snapshot_id", "version", "activated_at"}).
			AddRow(corpusID, activeSnapshotID, 3, activatedAt.Add(-time.Minute)))
	pool.ExpectQuery("SELECT id, corpus_id, manifest_sha256, created_by, created_at").
		WithArgs(corpusID, snapshotID).
		WillReturnRows(pool.NewRows([]string{"id", "corpus_id", "manifest_sha256", "created_by", "created_at"}).
			AddRow(snapshotID, corpusID, manifest, "test-maintainer", activatedAt))
	pool.ExpectQuery("SELECT source_id, source_revision_id, document_id, official_origin, captured_at, content_sha256").
		WithArgs(snapshotID).
		WillReturnRows(pool.NewRows([]string{"source_id", "source_revision_id", "document_id", "official_origin", "captured_at", "content_sha256"}).
			AddRow(member.SourceID, member.SourceRevisionID, member.DocumentID, member.OfficialOrigin, member.CapturedAt, member.ContentSHA256))
	pool.ExpectQuery("SELECT EXISTS").
		WithArgs(corpusID, snapshotID).
		WillReturnRows(pool.NewRows([]string{"exists"}).AddRow(true))
	pool.ExpectExec("UPDATE corpus_snapshot_releases").
		WithArgs(corpusID, snapshotID, 4, activatedAt).
		WillReturnResult(pgxmock.NewResult("UPDATE", 1))
	pool.ExpectCommit()

	publication, err := NewRepository(pool).Activate(context.Background(), domain.ActivationCommand{
		CorpusID: corpusID, SnapshotID: snapshotID, ExpectedReleaseVersion: 3, ActivatedAt: activatedAt,
	})
	if err != nil {
		t.Fatalf("activate graph-ready snapshot: %v", err)
	}
	if !publication.Created || publication.Release.Version != 4 || publication.Release.SnapshotID != snapshotID {
		t.Fatalf("publication = %+v, want newly active graph-ready snapshot", publication)
	}
	if err := pool.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet PostgreSQL expectations: %v", err)
	}
}

func TestActivateCreatesTheFirstReleaseForAGraphReadySnapshot(t *testing.T) {
	t.Parallel()
	pool, err := pgxmock.NewPool()
	if err != nil {
		t.Fatalf("create PostgreSQL mock: %v", err)
	}
	defer pool.Close()

	corpusID, snapshotID := uuid.New(), uuid.New()
	member := member()
	activatedAt := time.Date(2026, time.August, 25, 12, 0, 0, 0, time.UTC)
	manifest := "f" + strings.Repeat("b", 63)
	pool.ExpectBegin()
	pool.ExpectQuery("SELECT corpus_id, snapshot_id, version, activated_at").
		WithArgs(corpusID).
		WillReturnError(pgx.ErrNoRows)
	pool.ExpectQuery("SELECT id, corpus_id, manifest_sha256, created_by, created_at").
		WithArgs(corpusID, snapshotID).
		WillReturnRows(pool.NewRows([]string{"id", "corpus_id", "manifest_sha256", "created_by", "created_at"}).
			AddRow(snapshotID, corpusID, manifest, "test-maintainer", activatedAt))
	pool.ExpectQuery("SELECT source_id, source_revision_id, document_id, official_origin, captured_at, content_sha256").
		WithArgs(snapshotID).
		WillReturnRows(pool.NewRows([]string{"source_id", "source_revision_id", "document_id", "official_origin", "captured_at", "content_sha256"}).
			AddRow(member.SourceID, member.SourceRevisionID, member.DocumentID, member.OfficialOrigin, member.CapturedAt, member.ContentSHA256))
	pool.ExpectQuery("SELECT EXISTS").
		WithArgs(corpusID, snapshotID).
		WillReturnRows(pool.NewRows([]string{"exists"}).AddRow(true))
	pool.ExpectExec("INSERT INTO corpus_snapshot_releases").
		WithArgs(corpusID, snapshotID, 1, activatedAt).
		WillReturnResult(pgxmock.NewResult("INSERT", 1))
	pool.ExpectCommit()

	publication, err := NewRepository(pool).Activate(context.Background(), domain.ActivationCommand{
		CorpusID: corpusID, SnapshotID: snapshotID, ExpectedReleaseVersion: 0, ActivatedAt: activatedAt,
	})
	if err != nil {
		t.Fatalf("activate first graph-ready snapshot: %v", err)
	}
	if !publication.Created || publication.Release.Version != 1 || publication.Release.SnapshotID != snapshotID {
		t.Fatalf("publication = %+v, want first active graph-ready snapshot", publication)
	}
	if err := pool.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet PostgreSQL expectations: %v", err)
	}
}

func TestStageCreatesCandidateSnapshotFromReadyDocument(t *testing.T) {
	t.Parallel()
	pool, err := pgxmock.NewPool()
	if err != nil {
		t.Fatalf("create PostgreSQL mock: %v", err)
	}
	defer pool.Close()

	corpusID, sourceID, documentID, snapshotID := uuid.New(), uuid.New(), uuid.New(), uuid.New()
	sourceRevisionID := uuid.New()
	stagedAt := time.Date(2026, time.August, 25, 12, 0, 0, 0, time.UTC)
	candidate := domain.Member{
		SourceID: sourceID, SourceRevisionID: sourceRevisionID, DocumentID: documentID,
		OfficialOrigin: "https://official.example/law", CapturedAt: stagedAt.Add(-time.Minute),
		ContentSHA256: strings.Repeat("d", 64),
	}
	manifest := manifestSHA256([]domain.Member{candidate})
	pool.ExpectBegin()
	pool.ExpectQuery("SELECT corpus_id, snapshot_id, version, activated_at").
		WithArgs(corpusID).
		WillReturnError(pgx.ErrNoRows)
	pool.ExpectQuery("SELECT d.source_id, d.source_revision_id, d.id").
		WithArgs(corpusID, sourceID, documentID).
		WillReturnRows(pool.NewRows([]string{"source_id", "source_revision_id", "id", "official_origin", "captured_at", "content_sha256"}).
			AddRow(candidate.SourceID, candidate.SourceRevisionID, candidate.DocumentID, candidate.OfficialOrigin, candidate.CapturedAt, candidate.ContentSHA256))
	pool.ExpectQuery("SELECT[[:space:]]*\\(SELECT count\\(\\*\\) FROM document_units").
		WithArgs(documentID).
		WillReturnRows(pool.NewRows([]string{"unit_count", "chunk_count", "ready_chunk_count"}).AddRow(1, 1, 1))
	pool.ExpectQuery("SELECT id, corpus_id, manifest_sha256, created_by, created_at").
		WithArgs(corpusID, manifest).
		WillReturnError(pgx.ErrNoRows)
	pool.ExpectExec("INSERT INTO corpus_snapshots").
		WithArgs(snapshotID, corpusID, manifest, "ingestion-worker", stagedAt).
		WillReturnResult(pgxmock.NewResult("INSERT", 1))
	pool.ExpectExec("INSERT INTO corpus_snapshot_documents").
		WithArgs(snapshotID, corpusID, candidate.SourceID, candidate.SourceRevisionID, candidate.DocumentID, candidate.OfficialOrigin, candidate.CapturedAt, candidate.ContentSHA256).
		WillReturnResult(pgxmock.NewResult("INSERT", 1))
	pool.ExpectCommit()

	publication, err := NewRepository(pool).Stage(context.Background(), domain.StageCommand{
		CorpusID: corpusID, SourceID: sourceID, DocumentID: documentID,
		SnapshotID: snapshotID, Actor: "ingestion-worker", StagedAt: stagedAt,
	})
	if err != nil {
		t.Fatalf("stage snapshot: %v", err)
	}
	if !publication.Created || publication.Snapshot.ID != snapshotID || publication.Release.Version != 0 {
		t.Fatalf("publication = %+v, want an unpublished graph candidate", publication)
	}
	if err := pool.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet PostgreSQL expectations: %v", err)
	}
}

func TestStagePreservesTheCurrentReleaseMembersInTheCandidateManifest(t *testing.T) {
	t.Parallel()
	pool, err := pgxmock.NewPool()
	if err != nil {
		t.Fatalf("create PostgreSQL mock: %v", err)
	}
	defer pool.Close()

	corpusID, sourceID, documentID, snapshotID := uuid.New(), uuid.New(), uuid.New(), uuid.New()
	activeSnapshotID := uuid.New()
	stagedAt := time.Date(2026, time.August, 25, 12, 0, 0, 0, time.UTC)
	activeMember := member()
	activeMember.SourceID = uuid.New()
	candidate := domain.Member{
		SourceID: sourceID, SourceRevisionID: uuid.New(), DocumentID: documentID,
		OfficialOrigin: "https://official.example/law", CapturedAt: stagedAt.Add(-time.Minute),
		ContentSHA256: strings.Repeat("a", 64),
	}
	members := []domain.Member{activeMember, candidate}
	manifest := manifestSHA256(members)
	pool.ExpectBegin()
	pool.ExpectQuery("SELECT corpus_id, snapshot_id, version, activated_at").
		WithArgs(corpusID).
		WillReturnRows(pool.NewRows([]string{"corpus_id", "snapshot_id", "version", "activated_at"}).
			AddRow(corpusID, activeSnapshotID, 4, stagedAt.Add(-time.Minute)))
	pool.ExpectQuery("SELECT source_id, source_revision_id, document_id, official_origin, captured_at, content_sha256").
		WithArgs(activeSnapshotID).
		WillReturnRows(pool.NewRows([]string{"source_id", "source_revision_id", "document_id", "official_origin", "captured_at", "content_sha256"}).
			AddRow(activeMember.SourceID, activeMember.SourceRevisionID, activeMember.DocumentID, activeMember.OfficialOrigin, activeMember.CapturedAt, activeMember.ContentSHA256))
	pool.ExpectQuery("SELECT d.source_id, d.source_revision_id, d.id").
		WithArgs(corpusID, sourceID, documentID).
		WillReturnRows(pool.NewRows([]string{"source_id", "source_revision_id", "id", "official_origin", "captured_at", "content_sha256"}).
			AddRow(candidate.SourceID, candidate.SourceRevisionID, candidate.DocumentID, candidate.OfficialOrigin, candidate.CapturedAt, candidate.ContentSHA256))
	pool.ExpectQuery("SELECT[[:space:]]*\\(SELECT count\\(\\*\\) FROM document_units").
		WithArgs(documentID).
		WillReturnRows(pool.NewRows([]string{"unit_count", "chunk_count", "ready_chunk_count"}).AddRow(1, 1, 1))
	pool.ExpectQuery("SELECT id, corpus_id, manifest_sha256, created_by, created_at").
		WithArgs(corpusID, manifest).
		WillReturnError(pgx.ErrNoRows)
	pool.ExpectExec("INSERT INTO corpus_snapshots").
		WithArgs(snapshotID, corpusID, manifest, "ingestion-worker", stagedAt).
		WillReturnResult(pgxmock.NewResult("INSERT", 1))
	for _, snapshotMember := range members {
		pool.ExpectExec("INSERT INTO corpus_snapshot_documents").
			WithArgs(snapshotID, corpusID, snapshotMember.SourceID, snapshotMember.SourceRevisionID, snapshotMember.DocumentID, snapshotMember.OfficialOrigin, snapshotMember.CapturedAt, snapshotMember.ContentSHA256).
			WillReturnResult(pgxmock.NewResult("INSERT", 1))
	}
	pool.ExpectCommit()

	publication, err := NewRepository(pool).Stage(context.Background(), domain.StageCommand{
		CorpusID: corpusID, SourceID: sourceID, DocumentID: documentID,
		SnapshotID: snapshotID, Actor: "ingestion-worker", StagedAt: stagedAt,
	})
	if err != nil {
		t.Fatalf("stage candidate snapshot: %v", err)
	}
	if !publication.Created || len(publication.Snapshot.Members) != 2 || publication.Snapshot.Members[0] != activeMember {
		t.Fatalf("publication = %+v, want candidate that preserves the active manifest", publication)
	}
	if err := pool.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet PostgreSQL expectations: %v", err)
	}
}

func TestStageReusesAnExistingCandidateManifest(t *testing.T) {
	t.Parallel()
	pool, err := pgxmock.NewPool()
	if err != nil {
		t.Fatalf("create PostgreSQL mock: %v", err)
	}
	defer pool.Close()

	corpusID, sourceID, documentID, existingSnapshotID := uuid.New(), uuid.New(), uuid.New(), uuid.New()
	stagedAt := time.Date(2026, time.August, 25, 12, 0, 0, 0, time.UTC)
	candidate := domain.Member{
		SourceID: sourceID, SourceRevisionID: uuid.New(), DocumentID: documentID,
		OfficialOrigin: "https://official.example/law", CapturedAt: stagedAt.Add(-time.Minute),
		ContentSHA256: strings.Repeat("b", 64),
	}
	manifest := manifestSHA256([]domain.Member{candidate})
	pool.ExpectBegin()
	pool.ExpectQuery("SELECT corpus_id, snapshot_id, version, activated_at").
		WithArgs(corpusID).
		WillReturnError(pgx.ErrNoRows)
	pool.ExpectQuery("SELECT d.source_id, d.source_revision_id, d.id").
		WithArgs(corpusID, sourceID, documentID).
		WillReturnRows(pool.NewRows([]string{"source_id", "source_revision_id", "id", "official_origin", "captured_at", "content_sha256"}).
			AddRow(candidate.SourceID, candidate.SourceRevisionID, candidate.DocumentID, candidate.OfficialOrigin, candidate.CapturedAt, candidate.ContentSHA256))
	pool.ExpectQuery("SELECT[[:space:]]*\\(SELECT count\\(\\*\\) FROM document_units").
		WithArgs(documentID).
		WillReturnRows(pool.NewRows([]string{"unit_count", "chunk_count", "ready_chunk_count"}).AddRow(1, 1, 1))
	pool.ExpectQuery("SELECT id, corpus_id, manifest_sha256, created_by, created_at").
		WithArgs(corpusID, manifest).
		WillReturnRows(pool.NewRows([]string{"id", "corpus_id", "manifest_sha256", "created_by", "created_at"}).
			AddRow(existingSnapshotID, corpusID, manifest, "ingestion-worker", stagedAt))
	pool.ExpectQuery("SELECT source_id, source_revision_id, document_id, official_origin, captured_at, content_sha256").
		WithArgs(existingSnapshotID).
		WillReturnRows(pool.NewRows([]string{"source_id", "source_revision_id", "document_id", "official_origin", "captured_at", "content_sha256"}).
			AddRow(candidate.SourceID, candidate.SourceRevisionID, candidate.DocumentID, candidate.OfficialOrigin, candidate.CapturedAt, candidate.ContentSHA256))
	pool.ExpectCommit()

	publication, err := NewRepository(pool).Stage(context.Background(), domain.StageCommand{
		CorpusID: corpusID, SourceID: sourceID, DocumentID: documentID,
		SnapshotID: uuid.New(), Actor: "ingestion-worker", StagedAt: stagedAt,
	})
	if err != nil {
		t.Fatalf("reuse candidate snapshot: %v", err)
	}
	if publication.Created || publication.Snapshot.ID != existingSnapshotID {
		t.Fatalf("publication = %+v, want existing candidate manifest", publication)
	}
	if err := pool.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet PostgreSQL expectations: %v", err)
	}
}

func TestStageRejectsADocumentWithoutReadyRetrievalArtifacts(t *testing.T) {
	t.Parallel()
	pool, err := pgxmock.NewPool()
	if err != nil {
		t.Fatalf("create PostgreSQL mock: %v", err)
	}
	defer pool.Close()

	corpusID, sourceID, documentID := uuid.New(), uuid.New(), uuid.New()
	stagedAt := time.Date(2026, time.August, 25, 12, 0, 0, 0, time.UTC)
	candidate := domain.Member{
		SourceID: sourceID, SourceRevisionID: uuid.New(), DocumentID: documentID,
		OfficialOrigin: "https://official.example/law", CapturedAt: stagedAt.Add(-time.Minute),
		ContentSHA256: strings.Repeat("c", 64),
	}
	pool.ExpectBegin()
	pool.ExpectQuery("SELECT corpus_id, snapshot_id, version, activated_at").
		WithArgs(corpusID).
		WillReturnError(pgx.ErrNoRows)
	pool.ExpectQuery("SELECT d.source_id, d.source_revision_id, d.id").
		WithArgs(corpusID, sourceID, documentID).
		WillReturnRows(pool.NewRows([]string{"source_id", "source_revision_id", "id", "official_origin", "captured_at", "content_sha256"}).
			AddRow(candidate.SourceID, candidate.SourceRevisionID, candidate.DocumentID, candidate.OfficialOrigin, candidate.CapturedAt, candidate.ContentSHA256))
	pool.ExpectQuery("SELECT[[:space:]]*\\(SELECT count\\(\\*\\) FROM document_units").
		WithArgs(documentID).
		WillReturnRows(pool.NewRows([]string{"unit_count", "chunk_count", "ready_chunk_count"}).AddRow(1, 2, 1))
	pool.ExpectRollback()

	_, err = NewRepository(pool).Stage(context.Background(), domain.StageCommand{
		CorpusID: corpusID, SourceID: sourceID, DocumentID: documentID,
		SnapshotID: uuid.New(), Actor: "ingestion-worker", StagedAt: stagedAt,
	})
	if !errors.Is(err, domain.ErrCandidateNotReady) {
		t.Fatalf("stage error = %v, want readiness failure", err)
	}
	if err := pool.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet PostgreSQL expectations: %v", err)
	}
}

func TestInitializeCreatesTheFirstActiveReleaseFromReadyDocuments(t *testing.T) {
	t.Parallel()
	pool, err := pgxmock.NewPool()
	if err != nil {
		t.Fatalf("create PostgreSQL mock: %v", err)
	}
	defer pool.Close()

	corpusID, sourceID, documentID, snapshotID := uuid.New(), uuid.New(), uuid.New(), uuid.New()
	sourceRevisionID := uuid.New()
	createdAt := time.Date(2026, time.August, 25, 12, 0, 0, 0, time.UTC)
	candidate := domain.Member{
		SourceID: sourceID, SourceRevisionID: sourceRevisionID, DocumentID: documentID,
		OfficialOrigin: "https://official.example/law", CapturedAt: createdAt.Add(-time.Minute),
		ContentSHA256: strings.Repeat("e", 64),
	}
	manifest := manifestSHA256([]domain.Member{candidate})
	pool.ExpectBegin()
	pool.ExpectQuery("SELECT corpus_id, snapshot_id, version, activated_at").
		WithArgs(corpusID).
		WillReturnError(pgx.ErrNoRows)
	pool.ExpectQuery("SELECT s.id, d.id").
		WithArgs(corpusID).
		WillReturnRows(pool.NewRows([]string{"id", "id"}).AddRow(sourceID, documentID))
	pool.ExpectQuery("SELECT d.source_id, d.source_revision_id, d.id").
		WithArgs(corpusID, sourceID, documentID).
		WillReturnRows(pool.NewRows([]string{"source_id", "source_revision_id", "id", "official_origin", "captured_at", "content_sha256"}).
			AddRow(candidate.SourceID, candidate.SourceRevisionID, candidate.DocumentID, candidate.OfficialOrigin, candidate.CapturedAt, candidate.ContentSHA256))
	pool.ExpectQuery("SELECT[[:space:]]*\\(SELECT count\\(\\*\\) FROM document_units").
		WithArgs(documentID).
		WillReturnRows(pool.NewRows([]string{"unit_count", "chunk_count", "ready_chunk_count"}).AddRow(1, 1, 1))
	pool.ExpectQuery("SELECT id, corpus_id, manifest_sha256, created_by, created_at").
		WithArgs(corpusID, manifest).
		WillReturnError(pgx.ErrNoRows)
	pool.ExpectExec("INSERT INTO corpus_snapshots").
		WithArgs(snapshotID, corpusID, manifest, "local-maintainer", createdAt).
		WillReturnResult(pgxmock.NewResult("INSERT", 1))
	pool.ExpectExec("INSERT INTO corpus_snapshot_documents").
		WithArgs(snapshotID, corpusID, candidate.SourceID, candidate.SourceRevisionID, candidate.DocumentID, candidate.OfficialOrigin, candidate.CapturedAt, candidate.ContentSHA256).
		WillReturnResult(pgxmock.NewResult("INSERT", 1))
	pool.ExpectExec("INSERT INTO corpus_snapshot_releases").
		WithArgs(corpusID, snapshotID, 1, createdAt).
		WillReturnResult(pgxmock.NewResult("INSERT", 1))
	pool.ExpectCommit()

	publication, err := NewRepository(pool).Initialize(
		context.Background(), corpusID, snapshotID, "local-maintainer", createdAt,
	)
	if err != nil {
		t.Fatalf("initialize snapshot release: %v", err)
	}
	if !publication.Created || publication.Release.Version != 1 || publication.Release.SnapshotID != snapshotID {
		t.Fatalf("publication = %+v, want first active release", publication)
	}
	if err := pool.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet PostgreSQL expectations: %v", err)
	}
}

func TestInitializeReturnsTheExistingReleaseWithoutCreatingANewManifest(t *testing.T) {
	t.Parallel()
	pool, err := pgxmock.NewPool()
	if err != nil {
		t.Fatalf("create PostgreSQL mock: %v", err)
	}
	defer pool.Close()

	corpusID, snapshotID := uuid.New(), uuid.New()
	member := member()
	createdAt := time.Date(2026, time.August, 25, 12, 0, 0, 0, time.UTC)
	manifest := "f" + strings.Repeat("a", 63)
	pool.ExpectBegin()
	pool.ExpectQuery("SELECT corpus_id, snapshot_id, version, activated_at").
		WithArgs(corpusID).
		WillReturnRows(pool.NewRows([]string{"corpus_id", "snapshot_id", "version", "activated_at"}).
			AddRow(corpusID, snapshotID, 1, createdAt))
	pool.ExpectQuery("SELECT id, corpus_id, manifest_sha256, created_by, created_at").
		WithArgs(corpusID, snapshotID).
		WillReturnRows(pool.NewRows([]string{"id", "corpus_id", "manifest_sha256", "created_by", "created_at"}).
			AddRow(snapshotID, corpusID, manifest, "local-maintainer", createdAt))
	pool.ExpectQuery("SELECT source_id, source_revision_id, document_id, official_origin, captured_at, content_sha256").
		WithArgs(snapshotID).
		WillReturnRows(pool.NewRows([]string{"source_id", "source_revision_id", "document_id", "official_origin", "captured_at", "content_sha256"}).
			AddRow(member.SourceID, member.SourceRevisionID, member.DocumentID, member.OfficialOrigin, member.CapturedAt, member.ContentSHA256))
	pool.ExpectCommit()

	publication, err := NewRepository(pool).Initialize(
		context.Background(), corpusID, uuid.New(), "local-maintainer", createdAt,
	)
	if err != nil {
		t.Fatalf("return existing snapshot release: %v", err)
	}
	if publication.Created || publication.Release.SnapshotID != snapshotID || publication.Release.Version != 1 {
		t.Fatalf("publication = %+v, want the existing release", publication)
	}
	if err := pool.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet PostgreSQL expectations: %v", err)
	}
}

func member() domain.Member {
	return domain.Member{
		SourceID: uuid.New(), SourceRevisionID: uuid.New(), DocumentID: uuid.New(),
		OfficialOrigin: "https://example.org/law", CapturedAt: time.Date(2026, time.August, 24, 12, 0, 0, 0, time.UTC),
		ContentSHA256: "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
	}
}
