package postgres

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/dharlanoliveira/norvii/apps/api/internal/evaluation/application"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

// ListDatasetCatalog returns all immutable evaluation revisions, including unavailable revisions.
func (repository *Repository) ListDatasetCatalog(ctx context.Context) ([]application.DatasetCatalogEntry, error) {
	if repository == nil || repository.database == nil {
		return nil, ErrInvalidInput
	}
	rows, err := repository.database.Query(ctx, datasetCatalogQuery+` ORDER BY revision.imported_at DESC, revision.id DESC`)
	if err != nil {
		return nil, fmt.Errorf("list evaluation dataset catalog: %w", err)
	}
	defer rows.Close()
	entries := make([]application.DatasetCatalogEntry, 0)
	for rows.Next() {
		entry, err := scanDatasetCatalogEntry(rows)
		if err != nil {
			return nil, err
		}
		entries = append(entries, entry)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate evaluation dataset catalog: %w", err)
	}
	return entries, nil
}

// GetDatasetCatalog returns the one immutable revision and every maintainer-only inspection record.
func (repository *Repository) GetDatasetCatalog(ctx context.Context, datasetRevisionID uuid.UUID) (application.DatasetCatalogEntry, error) {
	if repository == nil || repository.database == nil || datasetRevisionID == uuid.Nil {
		return application.DatasetCatalogEntry{}, application.ErrDatasetNotFound
	}
	entry, err := readDatasetCatalogEntry(ctx, repository.database, datasetRevisionID)
	if err != nil {
		return application.DatasetCatalogEntry{}, err
	}
	if entry.Sources, err = repository.readDatasetCatalogSources(ctx, entry.Revision.ID, entry.Revision.CorpusID); err != nil {
		return application.DatasetCatalogEntry{}, err
	}
	if entry.Starters, err = repository.readDatasetCatalogStarters(ctx, entry.Revision.ID, entry.Revision.CorpusID); err != nil {
		return application.DatasetCatalogEntry{}, err
	}
	return entry, nil
}

const datasetCatalogQuery = `
	SELECT revision.id, revision.corpus_id, revision.dataset_key, revision.semantic_revision, revision.jurisdiction,
	       revision.manifest_sha256, revision.jsonl_sha256, revision.content_sha256, revision.declared_snapshot_date,
	       revision.query_languages, revision.authoritative_evidence_language,
	       publication.review_decision, publication.publication_state, publication.reviewed_at
	FROM evaluation_dataset_revision AS revision
	LEFT JOIN LATERAL (
		SELECT review_decision, publication_state, reviewed_at
		FROM evaluation_dataset_publication
		WHERE dataset_revision_id = revision.id AND corpus_id = revision.corpus_id
		ORDER BY reviewed_at DESC, id DESC
		LIMIT 1
	) AS publication ON true`

func readDatasetCatalogEntry(ctx context.Context, database queryer, datasetRevisionID uuid.UUID) (application.DatasetCatalogEntry, error) {
	row := database.QueryRow(ctx, datasetCatalogQuery+` WHERE revision.id = $1`, datasetRevisionID)
	entry, err := scanDatasetCatalogEntry(row)
	if errors.Is(err, pgx.ErrNoRows) {
		return application.DatasetCatalogEntry{}, application.ErrDatasetNotFound
	}
	if err != nil {
		return application.DatasetCatalogEntry{}, fmt.Errorf("read evaluation dataset catalog entry: %w", err)
	}
	return entry, nil
}

type datasetCatalogScanner interface{ Scan(...any) error }

func scanDatasetCatalogEntry(scanner datasetCatalogScanner) (application.DatasetCatalogEntry, error) {
	var entry application.DatasetCatalogEntry
	var review application.DatasetReview
	var decision, state *string
	var reviewedAtValue *time.Time
	err := scanner.Scan(
		&entry.Revision.ID, &entry.Revision.CorpusID, &entry.Revision.DatasetKey, &entry.Revision.SemanticRevision, &entry.Revision.Jurisdiction,
		&entry.Revision.ManifestSHA256, &entry.Revision.JSONLSHA256, &entry.Revision.ContentSHA256, &entry.Revision.DeclaredSnapshotDate,
		&entry.Revision.QueryLanguages, &entry.Revision.AuthoritativeEvidenceLanguage,
		&decision, &state, &reviewedAtValue,
	)
	if err != nil {
		return application.DatasetCatalogEntry{}, err
	}
	if decision != nil && state != nil && reviewedAtValue != nil {
		review.Decision, review.PublicationState, review.ReviewedAt = *decision, *state, *reviewedAtValue
		entry.Review = &review
	}
	return entry, nil
}

func (repository *Repository) readDatasetCatalogSources(ctx context.Context, revisionID, corpusID uuid.UUID) ([]application.DatasetSource, error) {
	rows, err := repository.database.Query(ctx, `
		SELECT id, source_alias, title, official_url, issuing_authority, document_type, authority_role, corpus_source_id
		FROM evaluation_dataset_source
		WHERE dataset_revision_id = $1 AND corpus_id = $2
		ORDER BY source_alias ASC`, revisionID, corpusID)
	if err != nil {
		return nil, fmt.Errorf("read evaluation dataset catalog sources: %w", err)
	}
	defer rows.Close()
	sources := make([]application.DatasetSource, 0)
	for rows.Next() {
		var source application.DatasetSource
		if err := rows.Scan(&source.ID, &source.SourceAlias, &source.Title, &source.OfficialURL, &source.IssuingAuthority,
			&source.DocumentType, &source.AuthorityRole, &source.CorpusSourceID); err != nil {
			return nil, fmt.Errorf("scan evaluation dataset catalog source: %w", err)
		}
		sources = append(sources, source)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate evaluation dataset catalog sources: %w", err)
	}
	return sources, nil
}

func (repository *Repository) readDatasetCatalogStarters(ctx context.Context, revisionID, corpusID uuid.UUID) ([]application.StarterCase, error) {
	rows, err := repository.database.Query(ctx, `
		SELECT id, evaluation_case_id, rank, query_language, is_review_eligible
		FROM evaluation_dataset_starter_case
		WHERE dataset_revision_id = $1 AND corpus_id = $2
		ORDER BY rank ASC, query_language ASC, id ASC`, revisionID, corpusID)
	if err != nil {
		return nil, fmt.Errorf("read evaluation dataset catalog starters: %w", err)
	}
	defer rows.Close()
	starters := make([]application.StarterCase, 0)
	for rows.Next() {
		var starter application.StarterCase
		if err := rows.Scan(&starter.ID, &starter.CaseID, &starter.Rank, &starter.QueryLanguage, &starter.ReviewEligible); err != nil {
			return nil, fmt.Errorf("scan evaluation dataset catalog starter: %w", err)
		}
		starters = append(starters, starter)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate evaluation dataset catalog starters: %w", err)
	}
	return starters, nil
}

var _ application.DatasetCatalogStore = (*Repository)(nil)
