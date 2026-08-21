package postgres

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/pashagolub/pgxmock/v4"
)

func TestGetLatestReturnsPublishedDocumentWithOrderedUnits(t *testing.T) {
	mock := newPoolMock(t)
	documentID := uuid.New()
	revisionID := uuid.New()
	rootID := uuid.New()
	titleID := uuid.New()
	articleOneID := uuid.New()
	articleTwoID := uuid.New()
	now := time.Date(2026, time.August, 18, 12, 0, 0, 0, time.UTC)
	finalURL := "https://example.org/law"
	extractedHash := "aabb"
	articleOneMarker := "Article 1"
	articleTwoMarker := "Article 2"
	mock.ExpectQuery("FROM sources s").WithArgs(pgxmock.AnyArg(), pgxmock.AnyArg()).WillReturnRows(
		mock.NewRows(documentColumns()).AddRow(
			documentID, revisionID, "corpus-ingestion-v1", "Article 1", "text-hash", now,
			"content-hash", now, "text/html", int64(2048), &finalURL, &extractedHash,
		),
	)
	mock.ExpectQuery("FROM document_units").WithArgs(documentID).WillReturnRows(
		mock.NewRows(unitColumns()).
			AddRow(articleTwoID, pgtype.UUID{Bytes: rootID, Valid: true}, "article", 1, &articleTwoMarker, nil, 50, 100, nil, nil, "article-2", "article-2-hash").
			AddRow(rootID, nil, "document", 0, nil, nil, 0, 100, nil, nil, "document", "root-hash").
			AddRow(articleOneID, pgtype.UUID{Bytes: rootID, Valid: true}, "article", 0, &articleOneMarker, nil, 20, 50, nil, nil, "article-1", "article-1-hash").
			AddRow(titleID, pgtype.UUID{Bytes: rootID, Valid: true}, "title", 0, nil, nil, 0, 20, nil, nil, "title", "title-hash"),
	)
	repository := NewRepository(mock)

	document, err := repository.GetLatest(context.Background(), uuid.New(), uuid.New())

	if err != nil {
		t.Fatalf("GetLatest() error = %v", err)
	}
	if document.ID != documentID || len(document.Units) != 4 {
		t.Fatalf("GetLatest() = %+v, want document with four units", document)
	}
	wantLocators := []string{"document", "title", "article-1", "article-2"}
	for index, wantLocator := range wantLocators {
		if document.Units[index].Locator != wantLocator {
			t.Fatalf("unit %d locator = %q, want %q", index, document.Units[index].Locator, wantLocator)
		}
	}
	if document.Units[2].ParentID == nil || *document.Units[2].ParentID != rootID {
		t.Fatalf("article parent = %v, want %s", document.Units[2].ParentID, rootID)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("database expectations = %v", err)
	}
}

func TestGetLatestHidesMissingAndForeignDocuments(t *testing.T) {
	mock := newPoolMock(t)
	mock.ExpectQuery("FROM sources s").WithArgs(pgxmock.AnyArg(), pgxmock.AnyArg()).WillReturnError(pgx.ErrNoRows)
	repository := NewRepository(mock)

	_, err := repository.GetLatest(context.Background(), uuid.New(), uuid.New())

	if !errors.Is(err, ErrNotFound) {
		t.Fatalf("GetLatest() error = %v, want ErrNotFound", err)
	}
}

func TestGetLatestReportsUnitQueryFailure(t *testing.T) {
	mock := newPoolMock(t)
	now := time.Now().UTC()
	expected := errors.New("query unavailable")
	mock.ExpectQuery("FROM sources s").WithArgs(pgxmock.AnyArg(), pgxmock.AnyArg()).WillReturnRows(
		mock.NewRows(documentColumns()).AddRow(
			uuid.New(), uuid.New(), "corpus-ingestion-v1", "Article 1", "text-hash", now,
			"content-hash", now, "text/html", int64(2048), nil, nil,
		),
	)
	mock.ExpectQuery("FROM document_units").WithArgs(pgxmock.AnyArg()).WillReturnError(expected)
	repository := NewRepository(mock)

	_, err := repository.GetLatest(context.Background(), uuid.New(), uuid.New())

	if !errors.Is(err, expected) {
		t.Fatalf("GetLatest() error = %v, want wrapped unit query failure", err)
	}
}

func newPoolMock(t *testing.T) pgxmock.PgxPoolIface {
	t.Helper()
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatalf("NewPool() error = %v", err)
	}
	t.Cleanup(mock.Close)
	return mock
}

func documentColumns() []string {
	return []string{
		"id", "source_revision_id", "pipeline_version", "text_content", "text_sha256", "created_at",
		"content_sha256", "captured_at", "media_type", "byte_size", "final_url", "extracted_content_sha256",
	}
}

func unitColumns() []string {
	return []string{
		"id", "parent_id", "kind", "ordinal", "marker", "label", "start_offset", "end_offset",
		"start_page", "end_page", "locator", "content_sha256",
	}
}
