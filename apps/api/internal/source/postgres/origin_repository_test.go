package postgres

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/dharlanoliveira/norvii/apps/api/internal/source/domain"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/pashagolub/pgxmock/v4"
)

func TestCreateURLPersistsSourceOriginAndInitialIngestionWork(t *testing.T) {
	t.Parallel()
	pool, err := pgxmock.NewPool()
	if err != nil {
		t.Fatalf("create PostgreSQL mock: %v", err)
	}
	defer pool.Close()

	createdAt := time.Date(2026, time.August, 25, 12, 0, 0, 0, time.UTC)
	source, err := domain.NewSource(uuid.New(), uuid.New(), "Official law", domain.KindURL, createdAt)
	if err != nil {
		t.Fatalf("create source fixture: %v", err)
	}
	submittedURL := "https://official.example/law"
	normalizedURL := "https://official.example/law"
	workID := uuid.New()
	pool.ExpectBegin()
	pool.ExpectQuery("SELECT status FROM corpora").
		WithArgs(source.CorpusID).
		WillReturnRows(pool.NewRows([]string{"status"}).AddRow("enabled"))
	pool.ExpectQuery("SELECT count\\(\\*\\) FROM sources").
		WithArgs(source.CorpusID).
		WillReturnRows(pool.NewRows([]string{"count"}).AddRow(0))
	pool.ExpectExec("INSERT INTO sources").
		WithArgs(source.ID, source.CorpusID, source.Title, source.Kind, source.Status, source.Version, source.CreatedAt, source.UpdatedAt).
		WillReturnResult(pgxmock.NewResult("INSERT", 1))
	pool.ExpectExec("INSERT INTO url_origins").
		WithArgs(source.ID, source.CorpusID, submittedURL, normalizedURL, source.CreatedAt).
		WillReturnResult(pgxmock.NewResult("INSERT", 1))
	pool.ExpectExec("INSERT INTO ingestion_work").
		WithArgs(workID, source.ID, source.CorpusID, source.CreatedAt).
		WillReturnResult(pgxmock.NewResult("INSERT", 1))
	pool.ExpectCommit()

	record, err := NewRepository(pool).CreateURL(
		context.Background(), source, submittedURL, normalizedURL, workID,
	)
	if err != nil {
		t.Fatalf("create URL source: %v", err)
	}
	if record.ID != source.ID || record.Origin.SubmittedURL == nil || *record.Origin.SubmittedURL != submittedURL {
		t.Fatalf("record = %+v, want newly pending URL source", record)
	}
	if err := pool.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet PostgreSQL expectations: %v", err)
	}
}

func TestGetPDFOriginReturnsCorpusScopedBinaryMetadata(t *testing.T) {
	t.Parallel()
	pool, err := pgxmock.NewPool()
	if err != nil {
		t.Fatalf("create PostgreSQL mock: %v", err)
	}
	defer pool.Close()

	corpusID := uuid.New()
	sourceID := uuid.New()
	pool.ExpectQuery("SELECT p.delivery_filename, p.detected_media_type, p.content").
		WithArgs(corpusID, sourceID).
		WillReturnRows(pool.NewRows([]string{"delivery_filename", "detected_media_type", "content"}).
			AddRow("official-law.pdf", "application/pdf", []byte("%PDF")))

	origin, err := NewRepository(pool).GetPDFOrigin(context.Background(), corpusID, sourceID)
	if err != nil {
		t.Fatalf("get PDF origin: %v", err)
	}
	if origin.DeliveryFilename != "official-law.pdf" || origin.MediaType != "application/pdf" || string(origin.Content) != "%PDF" {
		t.Fatalf("origin = %+v, want scoped PDF metadata and bytes", origin)
	}
	if err := pool.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet PostgreSQL expectations: %v", err)
	}
}

func TestGetURLOriginTreatsMissingOrForeignSourceAsNotFound(t *testing.T) {
	t.Parallel()
	pool, err := pgxmock.NewPool()
	if err != nil {
		t.Fatalf("create PostgreSQL mock: %v", err)
	}
	defer pool.Close()

	corpusID := uuid.New()
	sourceID := uuid.New()
	pool.ExpectQuery("SELECT COALESCE").
		WithArgs(corpusID, sourceID).
		WillReturnError(pgx.ErrNoRows)

	_, err = NewRepository(pool).GetURLOrigin(context.Background(), corpusID, sourceID)
	if !errors.Is(err, ErrNotFound) {
		t.Fatalf("get URL origin error = %v, want not found", err)
	}
	if err := pool.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet PostgreSQL expectations: %v", err)
	}
}

func TestGetURLOriginPrefersTheCapturedOfficialDestination(t *testing.T) {
	t.Parallel()
	pool, err := pgxmock.NewPool()
	if err != nil {
		t.Fatalf("create PostgreSQL mock: %v", err)
	}
	defer pool.Close()

	corpusID := uuid.New()
	sourceID := uuid.New()
	pool.ExpectQuery("SELECT COALESCE").
		WithArgs(corpusID, sourceID).
		WillReturnRows(pool.NewRows([]string{"url"}).AddRow("https://official.example/law"))

	origin, err := NewRepository(pool).GetURLOrigin(context.Background(), corpusID, sourceID)
	if err != nil {
		t.Fatalf("get URL origin: %v", err)
	}
	if origin.URL != "https://official.example/law" {
		t.Fatalf("URL = %q, want latest captured official destination", origin.URL)
	}
	if err := pool.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet PostgreSQL expectations: %v", err)
	}
}
