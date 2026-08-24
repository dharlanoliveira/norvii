// Package postgres persists and reads corpus aggregates from the canonical store.
package postgres

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/dharlanoliveira/norvii/apps/api/internal/catalog/domain"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

// ErrNotFound identifies an absent or unavailable corpus without foreign metadata.
var ErrNotFound = errors.New("corpus not found")

// ErrStaleState identifies an optimistic concurrency mismatch.
var ErrStaleState = errors.New("corpus state is stale")

type queryer interface {
	Query(context.Context, string, ...any) (pgx.Rows, error)
	QueryRow(context.Context, string, ...any) pgx.Row
}

type rowScanner interface {
	Scan(...any) error
}

// Summary combines a corpus aggregate with its authoritative source count.
type Summary struct {
	domain.Corpus
	SourceCount    int
	ActiveSnapshot *ActiveSnapshot
}

// ActiveSnapshot is the compact immutable release projection shown to researchers.
type ActiveSnapshot struct {
	ID             uuid.UUID
	ManifestSHA256 string
	CreatedAt      time.Time
	ActivatedAt    time.Time
	ReleaseVersion int
}

// Repository reads corpora through explicit projections and deterministic ordering.
type Repository struct {
	database queryer
}

// NewRepository constructs a corpus repository around the caller-owned database.
func NewRepository(database queryer) *Repository { return &Repository{database: database} }

// List returns enabled corpora unless maintainers explicitly include disabled entries.
func (repository *Repository) List(ctx context.Context, includeDisabled bool) ([]Summary, error) {
	rows, err := repository.database.Query(ctx, `
		SELECT c.id, c.name, c.description, c.language, c.jurisdiction,
		       c.status, c.version, c.created_at, c.updated_at, count(s.id),
		       release.snapshot_id, snapshot.manifest_sha256, snapshot.created_at,
		       release.activated_at, release.version
		FROM corpora c
		LEFT JOIN sources s ON s.corpus_id = c.id
		LEFT JOIN corpus_snapshot_releases release ON release.corpus_id = c.id
		LEFT JOIN corpus_snapshots snapshot ON snapshot.id = release.snapshot_id
		WHERE $1 OR c.status = 'enabled'
		GROUP BY c.id, release.snapshot_id, snapshot.manifest_sha256, snapshot.created_at,
		         release.activated_at, release.version
		ORDER BY c.language, c.name, c.id`, includeDisabled)
	if err != nil {
		return nil, fmt.Errorf("list corpora: %w", err)
	}
	defer rows.Close()
	corpora, err := pgx.CollectRows(rows, func(row pgx.CollectableRow) (Summary, error) {
		return scanSummary(row)
	})
	if err != nil {
		return nil, fmt.Errorf("scan corpus list: %w", err)
	}
	return corpora, nil
}

// Get returns one corpus while enforcing researcher availability when requested.
func (repository *Repository) Get(
	ctx context.Context,
	id uuid.UUID,
	includeDisabled bool,
) (Summary, error) {
	row := repository.database.QueryRow(ctx, `
		SELECT c.id, c.name, c.description, c.language, c.jurisdiction,
		       c.status, c.version, c.created_at, c.updated_at, count(s.id),
		       release.snapshot_id, snapshot.manifest_sha256, snapshot.created_at,
		       release.activated_at, release.version
		FROM corpora c
		LEFT JOIN sources s ON s.corpus_id = c.id
		LEFT JOIN corpus_snapshot_releases release ON release.corpus_id = c.id
		LEFT JOIN corpus_snapshots snapshot ON snapshot.id = release.snapshot_id
		WHERE c.id = $1 AND ($2 OR c.status = 'enabled')
		GROUP BY c.id, release.snapshot_id, snapshot.manifest_sha256, snapshot.created_at,
		         release.activated_at, release.version`, id, includeDisabled)
	corpus, err := scanSummary(row)
	if errors.Is(err, pgx.ErrNoRows) {
		return Summary{}, ErrNotFound
	}
	if err != nil {
		return Summary{}, fmt.Errorf("get corpus: %w", err)
	}
	return corpus, nil
}

// Create inserts one validated corpus aggregate atomically.
func (repository *Repository) Create(
	ctx context.Context,
	corpus domain.Corpus,
) (Summary, error) {
	row := repository.database.QueryRow(ctx, `
		WITH inserted AS (
			INSERT INTO corpora (
				id, name, description, language, jurisdiction, status,
				version, created_at, updated_at
			) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
			RETURNING *
		)
		SELECT id, name, description, language, jurisdiction,
		       status, version, created_at, updated_at, 0,
		       NULL::uuid, NULL::text, NULL::timestamptz, NULL::timestamptz, NULL::integer
		FROM inserted`,
		corpus.ID, corpus.Name, corpus.Description, corpus.Language,
		corpus.Jurisdiction, corpus.Status, corpus.Version,
		corpus.CreatedAt, corpus.UpdatedAt,
	)
	created, err := scanSummary(row)
	if err != nil {
		return Summary{}, fmt.Errorf("create corpus: %w", err)
	}
	return created, nil
}

// Update atomically replaces mutable metadata for the expected version.
func (repository *Repository) Update(
	ctx context.Context,
	id uuid.UUID,
	draft domain.Draft,
	expectedVersion int,
	now time.Time,
) (Summary, error) {
	row := repository.database.QueryRow(ctx, `
		WITH updated AS (
			UPDATE corpora
			SET name = $2, description = $3, language = $4, jurisdiction = $5,
			    version = version + 1, updated_at = $6
			WHERE id = $1 AND version = $7
			RETURNING *
		)
		SELECT u.id, u.name, u.description, u.language, u.jurisdiction,
		       u.status, u.version, u.created_at, u.updated_at, count(s.id),
		       release.snapshot_id, snapshot.manifest_sha256, snapshot.created_at,
		       release.activated_at, release.version
		FROM updated u
		LEFT JOIN sources s ON s.corpus_id = u.id
		LEFT JOIN corpus_snapshot_releases release ON release.corpus_id = u.id
		LEFT JOIN corpus_snapshots snapshot ON snapshot.id = release.snapshot_id
		GROUP BY u.id, u.name, u.description, u.language, u.jurisdiction,
		         u.status, u.version, u.created_at, u.updated_at, release.snapshot_id,
		         snapshot.manifest_sha256, snapshot.created_at, release.activated_at, release.version`,
		id, draft.Name, draft.Description, draft.Language, draft.Jurisdiction,
		now.UTC(), expectedVersion,
	)
	updated, err := scanSummary(row)
	if errors.Is(err, pgx.ErrNoRows) {
		return Summary{}, repository.mutationMiss(ctx, id)
	}
	if err != nil {
		return Summary{}, fmt.Errorf("update corpus: %w", err)
	}
	return updated, nil
}

// SetStatus atomically performs a reversible lifecycle mutation.
func (repository *Repository) SetStatus(
	ctx context.Context,
	id uuid.UUID,
	status domain.Status,
	expectedVersion int,
	now time.Time,
) (Summary, error) {
	row := repository.database.QueryRow(ctx, `
		WITH updated AS (
			UPDATE corpora
			SET status = $2, version = version + 1, updated_at = $3
			WHERE id = $1 AND version = $4 AND status <> $2
			RETURNING *
		)
		SELECT u.id, u.name, u.description, u.language, u.jurisdiction,
		       u.status, u.version, u.created_at, u.updated_at, count(s.id),
		       release.snapshot_id, snapshot.manifest_sha256, snapshot.created_at,
		       release.activated_at, release.version
		FROM updated u
		LEFT JOIN sources s ON s.corpus_id = u.id
		LEFT JOIN corpus_snapshot_releases release ON release.corpus_id = u.id
		LEFT JOIN corpus_snapshots snapshot ON snapshot.id = release.snapshot_id
		GROUP BY u.id, u.name, u.description, u.language, u.jurisdiction,
		         u.status, u.version, u.created_at, u.updated_at, release.snapshot_id,
		         snapshot.manifest_sha256, snapshot.created_at, release.activated_at, release.version`,
		id, status, now.UTC(), expectedVersion,
	)
	updated, err := scanSummary(row)
	if errors.Is(err, pgx.ErrNoRows) {
		return Summary{}, repository.mutationMiss(ctx, id)
	}
	if err != nil {
		return Summary{}, fmt.Errorf("set corpus status: %w", err)
	}
	return updated, nil
}

func (repository *Repository) mutationMiss(ctx context.Context, id uuid.UUID) error {
	var exists bool
	if err := repository.database.QueryRow(
		ctx, "SELECT EXISTS (SELECT 1 FROM corpora WHERE id = $1)", id,
	).Scan(&exists); err != nil {
		return fmt.Errorf("classify corpus mutation conflict: %w", err)
	}
	if exists {
		return ErrStaleState
	}
	return ErrNotFound
}

func scanSummary(row rowScanner) (Summary, error) {
	var summary Summary
	var snapshotID *uuid.UUID
	var manifestSHA256 *string
	var createdAt *time.Time
	var activatedAt *time.Time
	var releaseVersion *int
	err := row.Scan(
		&summary.ID,
		&summary.Name,
		&summary.Description,
		&summary.Language,
		&summary.Jurisdiction,
		&summary.Status,
		&summary.Version,
		&summary.CreatedAt,
		&summary.UpdatedAt,
		&summary.SourceCount,
		&snapshotID,
		&manifestSHA256,
		&createdAt,
		&activatedAt,
		&releaseVersion,
	)
	if err != nil {
		return Summary{}, err
	}
	if snapshotID != nil && manifestSHA256 != nil && createdAt != nil && activatedAt != nil && releaseVersion != nil {
		summary.ActiveSnapshot = &ActiveSnapshot{
			ID: *snapshotID, ManifestSHA256: *manifestSHA256, CreatedAt: *createdAt,
			ActivatedAt: *activatedAt, ReleaseVersion: *releaseVersion,
		}
	}
	return summary, nil
}
