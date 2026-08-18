// Package postgres reads immutable published documents through corpus ownership.
package postgres

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
)

// ErrNotFound intentionally covers absent, unavailable, and foreign documents.
var ErrNotFound = errors.New("published document not found")

type queryer interface {
	Query(context.Context, string, ...any) (pgx.Rows, error)
	QueryRow(context.Context, string, ...any) pgx.Row
}

// Unit is an addressable immutable span within a complete document.
type Unit struct {
	ID            uuid.UUID
	ParentID      *uuid.UUID
	Kind          string
	Ordinal       int
	Marker        *string
	Label         *string
	StartOffset   int
	EndOffset     int
	StartPage     *int
	EndPage       *int
	Locator       string
	ContentSHA256 string
}

// Document is the complete published text and its ordered units.
type Document struct {
	ID               uuid.UUID
	SourceRevisionID uuid.UUID
	PipelineVersion  string
	Text             string
	TextSHA256       string
	CreatedAt        time.Time
	Units            []Unit
	Provenance       Provenance
}

// Provenance describes the immutable acquisition behind a published document.
type Provenance struct {
	ContentSHA256          string
	CapturedAt             time.Time
	MediaType              string
	ByteSize               int64
	FinalURL               *string
	ExtractedContentSHA256 *string
}

// Repository reads only the latest published artifact owned by a corpus source.
type Repository struct {
	database queryer
}

// NewRepository constructs a document repository around a caller-owned database.
func NewRepository(database queryer) *Repository { return &Repository{database: database} }

// GetLatest returns the latest ready document with deterministic units.
func (repository *Repository) GetLatest(
	ctx context.Context,
	corpusID uuid.UUID,
	sourceID uuid.UUID,
) (Document, error) {
	var document Document
	err := repository.database.QueryRow(ctx, `
		SELECT d.id, d.source_revision_id, d.pipeline_version,
		       d.text_content, d.text_sha256, d.created_at,
		       r.content_sha256, r.captured_at, r.media_type, r.byte_size,
		       r.final_url, r.extracted_content_sha256
		FROM sources s
		JOIN document_versions d
		  ON d.id = s.latest_ready_document_id
		 AND d.corpus_id = s.corpus_id
		 AND d.source_id = s.id
		JOIN source_revisions r
		  ON r.id = d.source_revision_id
		 AND r.corpus_id = d.corpus_id
		 AND r.source_id = d.source_id
		WHERE s.corpus_id = $1 AND s.id = $2
		  AND s.processing_status IN ('ready', 'pending', 'failed')
		  AND d.publication_status = 'published'`, corpusID, sourceID).Scan(
		&document.ID,
		&document.SourceRevisionID,
		&document.PipelineVersion,
		&document.Text,
		&document.TextSHA256,
		&document.CreatedAt,
		&document.Provenance.ContentSHA256,
		&document.Provenance.CapturedAt,
		&document.Provenance.MediaType,
		&document.Provenance.ByteSize,
		&document.Provenance.FinalURL,
		&document.Provenance.ExtractedContentSHA256,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return Document{}, ErrNotFound
	}
	if err != nil {
		return Document{}, fmt.Errorf("get latest source document: %w", err)
	}
	rows, err := repository.database.Query(ctx, `
		SELECT id, parent_id, kind, ordinal, marker, label,
		       start_offset, end_offset, start_page, end_page,
		       locator, content_sha256
		FROM document_units
		WHERE document_id = $1
		ORDER BY parent_id NULLS FIRST, ordinal, id`, document.ID)
	if err != nil {
		return Document{}, fmt.Errorf("list document units: %w", err)
	}
	defer rows.Close()
	document.Units, err = pgx.CollectRows(rows, scanUnit)
	if err != nil {
		return Document{}, fmt.Errorf("scan document units: %w", err)
	}
	return document, nil
}

func scanUnit(row pgx.CollectableRow) (Unit, error) {
	var unit Unit
	var parentID pgtype.UUID
	err := row.Scan(
		&unit.ID, &parentID, &unit.Kind, &unit.Ordinal, &unit.Marker, &unit.Label,
		&unit.StartOffset, &unit.EndOffset, &unit.StartPage, &unit.EndPage,
		&unit.Locator, &unit.ContentSHA256,
	)
	if err != nil {
		return Unit{}, err
	}
	if parentID.Valid {
		id := uuid.UUID(parentID.Bytes)
		unit.ParentID = &id
	}
	return unit, nil
}
