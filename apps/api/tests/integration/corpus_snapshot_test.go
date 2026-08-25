//go:build integration

package integration_test

import (
	"context"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/dharlanoliveira/norvii/apps/api/internal/platform/persistence"
	snapshotdomain "github.com/dharlanoliveira/norvii/apps/api/internal/snapshot/domain"
	snapshotpostgres "github.com/dharlanoliveira/norvii/apps/api/internal/snapshot/postgres"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

func TestSnapshotPublicationPreservesHistoricalManifestAndRejectsDuplicates(t *testing.T) {
	configuration, err := persistence.LoadConfig(os.LookupEnv)
	if err != nil {
		t.Fatalf("LoadConfig() error = %v", err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), configuration.Timeout)
	defer cancel()
	pool, err := persistence.OpenPostgresPool(ctx, configuration.Postgres)
	if err != nil {
		t.Fatalf("OpenPostgresPool() error = %v", err)
	}
	t.Cleanup(pool.Close)

	corpusID, sourceID := uuid.New(), uuid.New()
	seedSnapshotFixture(t, ctx, pool, corpusID, sourceID)
	t.Cleanup(func() { deleteSnapshotFixture(t, context.Background(), pool, corpusID) })
	repository := snapshotpostgres.NewRepository(pool)
	now := time.Date(2026, time.August, 24, 15, 0, 0, 0, time.UTC)

	firstDocumentID := insertReadySnapshotDocument(t, ctx, pool, corpusID, sourceID, "first", now)
	initial, err := repository.Initialize(ctx, corpusID, uuid.New(), "integration-test", now)
	if err != nil {
		t.Fatalf("Initialize() error = %v", err)
	}
	if !initial.Created || initial.Release.Version != 1 || len(initial.Snapshot.Members) != 1 {
		t.Fatalf("initial publication = %+v, want one created release", initial)
	}

	secondDocumentID := insertReadySnapshotDocument(t, ctx, pool, corpusID, sourceID, "second", now.Add(time.Minute))
	published, err := repository.Publish(ctx, snapshotdomain.PublishCommand{
		CorpusID: corpusID, SourceID: sourceID, DocumentID: secondDocumentID,
		SnapshotID: uuid.New(), Actor: "integration-test", PublishedAt: now.Add(2 * time.Minute),
		ExpectedReleaseVersion: initial.Release.Version,
	})
	if err != nil {
		t.Fatalf("Publish() error = %v", err)
	}
	if !published.Created || published.Release.Version != 2 || published.Snapshot.ID == initial.Snapshot.ID {
		t.Fatalf("published release = %+v, want distinct version two release", published)
	}

	historical, err := repository.Get(ctx, corpusID, initial.Snapshot.ID)
	if err != nil {
		t.Fatalf("Get(initial snapshot) error = %v", err)
	}
	if historical.Members[0].DocumentID != firstDocumentID {
		t.Fatalf("historical document = %s, want %s", historical.Members[0].DocumentID, firstDocumentID)
	}
	active, err := repository.Active(ctx, corpusID)
	if err != nil {
		t.Fatalf("Active() error = %v", err)
	}
	if active.SnapshotID != published.Snapshot.ID || active.Version != 2 {
		t.Fatalf("active release = %+v, want published version two", active)
	}

	duplicate, err := repository.Publish(ctx, snapshotdomain.PublishCommand{
		CorpusID: corpusID, SourceID: sourceID, DocumentID: secondDocumentID,
		SnapshotID: uuid.New(), Actor: "integration-test", PublishedAt: now.Add(3 * time.Minute),
		ExpectedReleaseVersion: published.Release.Version,
	})
	if err != nil {
		t.Fatalf("Publish(unchanged candidate) error = %v", err)
	}
	if duplicate.Created || duplicate.Snapshot.ID != published.Snapshot.ID || duplicate.Release.Version != 2 {
		t.Fatalf("duplicate publication = %+v, want existing active release", duplicate)
	}
}

func seedSnapshotFixture(
	t *testing.T,
	ctx context.Context,
	pool *pgxpool.Pool,
	corpusID, sourceID uuid.UUID,
) {
	t.Helper()
	if _, err := pool.Exec(ctx, `
		INSERT INTO corpora (id, name, description, language, jurisdiction)
		VALUES ($1, 'Snapshot integration corpus', 'Snapshot publication integration fixture.', 'en', 'Test')`, corpusID); err != nil {
		t.Fatalf("insert corpus: %v", err)
	}
	if _, err := pool.Exec(ctx, `
		INSERT INTO sources (id, corpus_id, title, kind, processing_status)
		VALUES ($1, $2, 'Snapshot integration source', 'url', 'ready')`, sourceID, corpusID); err != nil {
		t.Fatalf("insert source: %v", err)
	}
	if _, err := pool.Exec(ctx, `
		INSERT INTO url_origins (source_id, corpus_id, submitted_url, normalized_url)
		VALUES ($1, $2, 'https://example.org/snapshot', 'https://example.org/snapshot')`, sourceID, corpusID); err != nil {
		t.Fatalf("insert source origin: %v", err)
	}
}

func insertReadySnapshotDocument(
	t *testing.T,
	ctx context.Context,
	pool *pgxpool.Pool,
	corpusID, sourceID uuid.UUID,
	version string,
	capturedAt time.Time,
) uuid.UUID {
	t.Helper()
	workID, attemptID, revisionID, documentID, unitID, chunkID := uuid.New(), uuid.New(), uuid.New(), uuid.New(), uuid.New(), uuid.New()
	contentHash, textHash := strings.Repeat("a", 64), strings.Repeat("c", 64)
	if version == "second" {
		contentHash, textHash = strings.Repeat("b", 64), strings.Repeat("d", 64)
	}
	if _, err := pool.Exec(ctx, `
		INSERT INTO ingestion_work (id, source_id, corpus_id, reason, status)
		VALUES ($1, $2, $3, 'reprocess', 'succeeded')`, workID, sourceID, corpusID); err != nil {
		t.Fatalf("insert work: %v", err)
	}
	if _, err := pool.Exec(ctx, `
		INSERT INTO processing_attempts (
			id, work_id, source_id, corpus_id, attempt_number, pipeline_version, status,
			lease_token, worker_id, started_at, finished_at
		) VALUES ($1, $2, $3, $4, 1, 'snapshot-test', 'succeeded', $5, 'integration-test', $6, $6)`,
		attemptID, workID, sourceID, corpusID, uuid.New(), capturedAt,
	); err != nil {
		t.Fatalf("insert attempt: %v", err)
	}
	if _, err := pool.Exec(ctx, `
		INSERT INTO source_revisions (
			id, source_id, corpus_id, attempt_id, content_sha256, captured_at, media_type,
			byte_size, pipeline_version, final_url, extracted_content_sha256
		) VALUES ($1, $2, $3, $4, $5, $6, 'text/html', 120, 'snapshot-test', 'https://example.org/snapshot', $5)`,
		revisionID, sourceID, corpusID, attemptID, contentHash, capturedAt,
	); err != nil {
		t.Fatalf("insert source revision: %v", err)
	}
	if _, err := pool.Exec(ctx, `
		INSERT INTO document_versions (
			id, source_revision_id, source_id, corpus_id, pipeline_version, text_content,
			text_sha256, published_at
		) VALUES ($1, $2, $3, $4, 'snapshot-test', $5, $6, $7)`,
		documentID, revisionID, sourceID, corpusID, "Snapshot "+version+" legal text.", textHash, capturedAt,
	); err != nil {
		t.Fatalf("insert document version: %v", err)
	}
	if _, err := pool.Exec(ctx, `
		INSERT INTO document_units (
			id, document_id, kind, ordinal, start_offset, end_offset, locator, content_sha256
		) VALUES ($1, $2, 'article', 1, 0, 28, 'article-1', $3)`, unitID, documentID, textHash); err != nil {
		t.Fatalf("insert document unit: %v", err)
	}
	if _, err := pool.Exec(ctx, `
		INSERT INTO retrieval_chunks (
			id, corpus_id, source_id, document_id, unit_id, ordinal, start_offset, end_offset,
			content, content_sha256, context_locator, embedding, embedding_model, enrichment_status
		) VALUES ($1, $2, $3, $4, $5, 1, 0, 28, $6, $7, 'article-1', $8::vector, 'snapshot-test', 'ready')`,
		chunkID, corpusID, sourceID, documentID, unitID, "Snapshot "+version+" legal text.", textHash, zeroVector(),
	); err != nil {
		t.Fatalf("insert retrieval chunk: %v", err)
	}
	if _, err := pool.Exec(ctx, `
		UPDATE sources
		SET latest_ready_document_id = $3, processing_status = 'ready'
		WHERE corpus_id = $1 AND id = $2`, corpusID, sourceID, documentID); err != nil {
		t.Fatalf("activate ready document: %v", err)
	}
	return documentID
}

func deleteSnapshotFixture(t *testing.T, ctx context.Context, pool *pgxpool.Pool, corpusID uuid.UUID) {
	t.Helper()
	if _, err := pool.Exec(ctx, "DELETE FROM corpus_snapshot_releases WHERE corpus_id = $1", corpusID); err != nil {
		t.Errorf("delete snapshot release: %v", err)
		return
	}
	if _, err := pool.Exec(ctx, "DELETE FROM corpus_snapshot_documents WHERE corpus_id = $1", corpusID); err != nil {
		t.Errorf("delete snapshot members: %v", err)
		return
	}
	if _, err := pool.Exec(ctx, "DELETE FROM corpus_snapshots WHERE corpus_id = $1", corpusID); err != nil {
		t.Errorf("delete snapshots: %v", err)
		return
	}
	if _, err := pool.Exec(ctx, "DELETE FROM retrieval_chunks WHERE corpus_id = $1", corpusID); err != nil {
		t.Errorf("delete retrieval chunks: %v", err)
		return
	}
	if _, err := pool.Exec(ctx, "DELETE FROM document_units WHERE document_id IN (SELECT id FROM document_versions WHERE corpus_id = $1)", corpusID); err != nil {
		t.Errorf("delete document units: %v", err)
		return
	}
	if _, err := pool.Exec(ctx, "UPDATE sources SET latest_ready_document_id = NULL WHERE corpus_id = $1", corpusID); err != nil {
		t.Errorf("clear latest documents: %v", err)
		return
	}
	for _, table := range []string{"document_versions", "source_revisions", "processing_attempts", "ingestion_work", "url_origins", "sources"} {
		if _, err := pool.Exec(ctx, "DELETE FROM "+table+" WHERE corpus_id = $1", corpusID); err != nil {
			t.Errorf("delete %s: %v", table, err)
			return
		}
	}
	if _, err := pool.Exec(ctx, "DELETE FROM corpora WHERE id = $1", corpusID); err != nil {
		t.Errorf("delete corpus: %v", err)
	}
}

func zeroVector() string { return "[" + strings.TrimSuffix(strings.Repeat("0,", 1536), ",") + "]" }
