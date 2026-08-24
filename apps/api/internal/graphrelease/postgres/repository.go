// Package postgres persists canonical graph-release manifests.
package postgres

import (
	"context"
	"errors"
	"fmt"

	"github.com/dharlanoliveira/norvii/apps/api/internal/graphrelease/domain"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

type queryer interface {
	QueryRow(context.Context, string, ...any) pgx.Row
}

// Repository owns graph-release reads at the Go public boundary.
type Repository struct{ database queryer }

// NewRepository constructs graph-release persistence from caller-owned connections.
func NewRepository(database queryer) *Repository { return &Repository{database: database} }

// Get returns the newest release built for one snapshot, without crossing corpus boundaries.
func (repository *Repository) Get(
	ctx context.Context, corpusID, snapshotID uuid.UUID,
) (domain.Release, error) {
	release, err := scanRelease(repository.database.QueryRow(ctx, `
		SELECT id, corpus_id, snapshot_id, manifest_sha256, build_version, status,
		       COALESCE(failure_category, ''), entity_count, relationship_count, created_at, completed_at
		FROM graph_releases
		WHERE corpus_id = $1 AND snapshot_id = $2
		ORDER BY created_at DESC, id DESC
		LIMIT 1`, corpusID, snapshotID))
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.Release{}, domain.ErrNotFound
	}
	if err != nil {
		return domain.Release{}, fmt.Errorf("get graph release: %w", err)
	}
	return release, nil
}

func scanRelease(row pgx.Row) (domain.Release, error) {
	var release domain.Release
	if err := row.Scan(
		&release.ID, &release.CorpusID, &release.SnapshotID, &release.ManifestSHA256,
		&release.BuildVersion, &release.Status, &release.FailureCategory, &release.EntityCount,
		&release.RelationshipCount, &release.CreatedAt, &release.CompletedAt,
	); err != nil {
		return domain.Release{}, err
	}
	if err := release.Validate(); err != nil {
		return domain.Release{}, fmt.Errorf("validate graph release: %w", err)
	}
	return release, nil
}
