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
	parentID := uuid.New()
	now := time.Date(2026, time.August, 18, 12, 0, 0, 0, time.UTC)
	finalURL := "https://example.org/law"
	extractedHash := "aabb"
	marker := "Article 1"
	label := "Purpose"
	mock.ExpectQuery("FROM sources s").WithArgs(pgxmock.AnyArg(), pgxmock.AnyArg()).WillReturnRows(
		mock.NewRows(documentColumns()).AddRow(
			documentID, revisionID, "corpus-ingestion-v1", "Article 1", "text-hash", now,
			"content-hash", now, "text/html", int64(2048), &finalURL, &extractedHash,
		),
	)
	mock.ExpectQuery("FROM document_units").WithArgs(documentID).WillReturnRows(
		mock.NewRows(unitColumns()).
			AddRow(parentID, nil, "document", 0, nil, nil, 0, 9, nil, nil, "document", "root-hash").
			AddRow(uuid.New(), pgtype.UUID{Bytes: parentID, Valid: true}, "article", 1, &marker, &label, 0, 9, nil, nil, "article-1", "unit-hash"),
	)
	repository := NewRepository(mock)

	document, err := repository.GetLatest(context.Background(), uuid.New(), uuid.New())

	if err != nil {
		t.Fatalf("GetLatest() error = %v", err)
	}
	if document.ID != documentID || len(document.Units) != 2 {
		t.Fatalf("GetLatest() = %+v, want document with two units", document)
	}
	if document.Units[1].ParentID == nil || *document.Units[1].ParentID != parentID {
		t.Fatalf("article parent = %v, want %s", document.Units[1].ParentID, parentID)
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
