// Package postgres persists immutable snapshot manifests and active release pointers.
package postgres

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/dharlanoliveira/norvii/apps/api/internal/snapshot/domain"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

type database interface {
	queryer
	Begin(context.Context) (pgx.Tx, error)
}

type queryer interface {
	Query(context.Context, string, ...any) (pgx.Rows, error)
	QueryRow(context.Context, string, ...any) pgx.Row
}

// Repository owns canonical snapshot publication persistence.
type Repository struct {
	database database
}

type snapshotWriteRequest struct {
	corpusID        uuid.UUID
	snapshotID      uuid.UUID
	actor           string
	publishedAt     time.Time
	members         []domain.Member
	previousRelease domain.Release
}

type snapshotStageRequest struct {
	corpusID   uuid.UUID
	snapshotID uuid.UUID
	actor      string
	stagedAt   time.Time
	members    []domain.Member
}

// NewRepository constructs a snapshot repository around caller-owned persistence.
func NewRepository(database database) *Repository { return &Repository{database: database} }

// Active returns the current release for an enabled corpus.
func (repository *Repository) Active(ctx context.Context, corpusID uuid.UUID) (domain.Release, error) {
	return scanRelease(repository.database.QueryRow(ctx, `
		SELECT release.corpus_id, release.snapshot_id, release.version, release.activated_at
		FROM corpus_snapshot_releases AS release
		JOIN corpora AS corpus ON corpus.id = release.corpus_id
		WHERE release.corpus_id = $1
		  AND corpus.status = 'enabled'`, corpusID))
}

// List returns immutable snapshot manifests newest first for one corpus.
func (repository *Repository) List(ctx context.Context, corpusID uuid.UUID) ([]domain.Snapshot, error) {
	rows, err := repository.database.Query(ctx, `
		SELECT id, corpus_id, manifest_sha256, created_by, created_at
		FROM corpus_snapshots
		WHERE corpus_id = $1
		ORDER BY created_at DESC, id DESC`, corpusID)
	if err != nil {
		return nil, fmt.Errorf("list corpus snapshots: %w", err)
	}
	defer rows.Close()
	manifests, err := pgx.CollectRows(rows, func(row pgx.CollectableRow) (domain.Snapshot, error) {
		var snapshot domain.Snapshot
		err := row.Scan(&snapshot.ID, &snapshot.CorpusID, &snapshot.ManifestSHA256, &snapshot.CreatedBy, &snapshot.CreatedAt)
		return snapshot, err
	})
	if err != nil {
		return nil, fmt.Errorf("scan corpus snapshots: %w", err)
	}
	snapshots := make([]domain.Snapshot, 0, len(manifests))
	for _, manifest := range manifests {
		snapshot, readErr := repository.readSnapshot(ctx, repository.database, corpusID, manifest.ID)
		if readErr != nil {
			return nil, fmt.Errorf("read corpus snapshot manifest: %w", readErr)
		}
		snapshots = append(snapshots, snapshot)
	}
	return snapshots, nil
}

// Get returns an immutable manifest only through its owning corpus.
func (repository *Repository) Get(ctx context.Context, corpusID, snapshotID uuid.UUID) (domain.Snapshot, error) {
	snapshot, err := repository.readSnapshot(ctx, repository.database, corpusID, snapshotID)
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.Snapshot{}, domain.ErrNotFound
	}
	if err != nil {
		return domain.Snapshot{}, fmt.Errorf("get corpus snapshot: %w", err)
	}
	return snapshot, nil
}

// Publish atomically replaces one source member in the active evidence release.
func (repository *Repository) Publish(ctx context.Context, command domain.PublishCommand) (domain.Publication, error) {
	if err := command.Validate(); err != nil {
		return domain.Publication{}, err
	}
	transaction, err := repository.database.Begin(ctx)
	if err != nil {
		return domain.Publication{}, fmt.Errorf("begin snapshot publication: %w", err)
	}
	defer func() { _ = transaction.Rollback(ctx) }()

	release, err := repository.lockRelease(ctx, transaction, command.CorpusID)
	if err != nil {
		return domain.Publication{}, err
	}
	if release.Version != command.ExpectedReleaseVersion {
		return domain.Publication{}, domain.ErrStaleRelease
	}
	members, err := repository.membersForSnapshot(ctx, transaction, release.SnapshotID)
	if err != nil {
		return domain.Publication{}, err
	}
	candidate, err := repository.candidateMember(ctx, transaction, command.CorpusID, command.SourceID, command.DocumentID)
	if err != nil {
		return domain.Publication{}, err
	}
	members = replaceMember(members, candidate)
	publication, err := repository.writeSnapshot(ctx, transaction, snapshotWriteRequest{
		corpusID:        command.CorpusID,
		snapshotID:      command.SnapshotID,
		actor:           command.Actor,
		publishedAt:     command.PublishedAt,
		members:         members,
		previousRelease: release,
	})
	if err != nil {
		return domain.Publication{}, err
	}
	if err := transaction.Commit(ctx); err != nil {
		return domain.Publication{}, fmt.Errorf("commit snapshot publication: %w", err)
	}
	return publication, nil
}

// Stage creates an immutable candidate snapshot while leaving the active release unchanged.
func (repository *Repository) Stage(ctx context.Context, command domain.StageCommand) (domain.Publication, error) {
	if err := command.Validate(); err != nil {
		return domain.Publication{}, err
	}
	transaction, err := repository.database.Begin(ctx)
	if err != nil {
		return domain.Publication{}, fmt.Errorf("begin snapshot staging: %w", err)
	}
	defer func() { _ = transaction.Rollback(ctx) }()

	activeRelease, err := repository.lockRelease(ctx, transaction, command.CorpusID)
	if err != nil && !errors.Is(err, domain.ErrNotFound) {
		return domain.Publication{}, err
	}
	members := make([]domain.Member, 0, 1)
	if err == nil {
		members, err = repository.membersForSnapshot(ctx, transaction, activeRelease.SnapshotID)
		if err != nil {
			return domain.Publication{}, err
		}
	}
	candidate, err := repository.candidateMember(ctx, transaction, command.CorpusID, command.SourceID, command.DocumentID)
	if err != nil {
		return domain.Publication{}, err
	}
	members = replaceMember(members, candidate)
	snapshot, created, err := repository.writeStagedSnapshot(ctx, transaction, snapshotStageRequest{
		corpusID: command.CorpusID, snapshotID: command.SnapshotID, actor: command.Actor,
		stagedAt: command.StagedAt, members: members,
	})
	if err != nil {
		return domain.Publication{}, err
	}
	if err := transaction.Commit(ctx); err != nil {
		return domain.Publication{}, fmt.Errorf("commit snapshot staging: %w", err)
	}
	return domain.Publication{Snapshot: snapshot, Release: activeRelease, Created: created}, nil
}

// Activate promotes only a candidate snapshot with a completed graph release.
func (repository *Repository) Activate(ctx context.Context, command domain.ActivationCommand) (domain.Publication, error) {
	if err := command.Validate(); err != nil {
		return domain.Publication{}, err
	}
	transaction, err := repository.database.Begin(ctx)
	if err != nil {
		return domain.Publication{}, fmt.Errorf("begin snapshot activation: %w", err)
	}
	defer func() { _ = transaction.Rollback(ctx) }()

	activeRelease, activeErr := repository.lockRelease(ctx, transaction, command.CorpusID)
	if activeErr != nil && !errors.Is(activeErr, domain.ErrNotFound) {
		return domain.Publication{}, activeErr
	}
	activeVersion := 0
	if activeErr == nil {
		activeVersion = activeRelease.Version
	}
	if activeVersion != command.ExpectedReleaseVersion {
		return domain.Publication{}, domain.ErrStaleRelease
	}
	snapshot, err := repository.readSnapshot(ctx, transaction, command.CorpusID, command.SnapshotID)
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.Publication{}, domain.ErrNotFound
	}
	if err != nil {
		return domain.Publication{}, fmt.Errorf("read staged snapshot: %w", err)
	}
	if err := repository.requireReadyGraphRelease(ctx, transaction, command.CorpusID, command.SnapshotID); err != nil {
		return domain.Publication{}, err
	}
	if activeErr == nil && activeRelease.SnapshotID == command.SnapshotID {
		if err := transaction.Commit(ctx); err != nil {
			return domain.Publication{}, fmt.Errorf("commit existing snapshot activation: %w", err)
		}
		return domain.Publication{Snapshot: snapshot, Release: activeRelease, Created: false}, nil
	}
	activatedAt := command.ActivatedAt.UTC()
	release := domain.Release{CorpusID: command.CorpusID, SnapshotID: command.SnapshotID, Version: activeVersion + 1, ActivatedAt: activatedAt}
	if activeErr != nil {
		_, err = transaction.Exec(ctx, `
			INSERT INTO corpus_snapshot_releases (corpus_id, snapshot_id, version, activated_at)
			VALUES ($1, $2, $3, $4)`, release.CorpusID, release.SnapshotID, release.Version, release.ActivatedAt)
	} else {
		_, err = transaction.Exec(ctx, `
			UPDATE corpus_snapshot_releases
			SET snapshot_id = $2, version = $3, activated_at = $4
			WHERE corpus_id = $1`, release.CorpusID, release.SnapshotID, release.Version, release.ActivatedAt)
	}
	if err != nil {
		return domain.Publication{}, fmt.Errorf("activate graph-ready snapshot: %w", err)
	}
	if err := transaction.Commit(ctx); err != nil {
		return domain.Publication{}, fmt.Errorf("commit snapshot activation: %w", err)
	}
	return domain.Publication{Snapshot: snapshot, Release: release, Created: true}, nil
}

// Initialize creates the first release from all ready documents, or returns its existing release.
func (repository *Repository) Initialize(
	ctx context.Context, corpusID, snapshotID uuid.UUID, actor string, now time.Time,
) (domain.Publication, error) {
	if corpusID == uuid.Nil || snapshotID == uuid.Nil || strings.TrimSpace(actor) == "" {
		return domain.Publication{}, domain.ErrInvalidInput
	}
	transaction, err := repository.database.Begin(ctx)
	if err != nil {
		return domain.Publication{}, fmt.Errorf("begin snapshot initialization: %w", err)
	}
	defer func() { _ = transaction.Rollback(ctx) }()
	if release, err := repository.lockRelease(ctx, transaction, corpusID); err == nil {
		snapshot, readErr := repository.readSnapshot(ctx, transaction, corpusID, release.SnapshotID)
		if readErr != nil {
			return domain.Publication{}, fmt.Errorf("read initialized snapshot: %w", readErr)
		}
		if commitErr := transaction.Commit(ctx); commitErr != nil {
			return domain.Publication{}, fmt.Errorf("commit existing snapshot initialization: %w", commitErr)
		}
		return domain.Publication{Snapshot: snapshot, Release: release, Created: false}, nil
	} else if !errors.Is(err, domain.ErrNotFound) {
		return domain.Publication{}, err
	}
	members, err := repository.readyMembers(ctx, transaction, corpusID)
	if err != nil {
		return domain.Publication{}, err
	}
	publication, err := repository.writeSnapshot(ctx, transaction, snapshotWriteRequest{
		corpusID:        corpusID,
		snapshotID:      snapshotID,
		actor:           actor,
		publishedAt:     now,
		members:         members,
		previousRelease: domain.Release{CorpusID: corpusID},
	})
	if err != nil {
		return domain.Publication{}, err
	}
	if err := transaction.Commit(ctx); err != nil {
		return domain.Publication{}, fmt.Errorf("commit snapshot initialization: %w", err)
	}
	return publication, nil
}

func (repository *Repository) lockRelease(ctx context.Context, transaction pgx.Tx, corpusID uuid.UUID) (domain.Release, error) {
	release, err := scanRelease(transaction.QueryRow(ctx, `
		SELECT corpus_id, snapshot_id, version, activated_at
		FROM corpus_snapshot_releases
		WHERE corpus_id = $1
		FOR UPDATE`, corpusID))
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.Release{}, domain.ErrNotFound
	}
	if err != nil {
		return domain.Release{}, fmt.Errorf("lock snapshot release: %w", err)
	}
	return release, nil
}

func (repository *Repository) writeSnapshot(
	ctx context.Context,
	transaction pgx.Tx,
	request snapshotWriteRequest,
) (domain.Publication, error) {
	if len(request.members) == 0 {
		return domain.Publication{}, domain.ErrCandidateNotReady
	}
	manifest := manifestSHA256(request.members)
	existing, err := repository.findByManifest(ctx, transaction, request.corpusID, manifest)
	if err == nil {
		if request.previousRelease.SnapshotID != existing.ID {
			return domain.Publication{}, domain.ErrStaleRelease
		}
		return domain.Publication{Snapshot: existing, Release: request.previousRelease, Created: false}, nil
	}
	if !errors.Is(err, domain.ErrNotFound) {
		return domain.Publication{}, err
	}
	publishedAt := request.publishedAt.UTC()
	if _, err := transaction.Exec(ctx, `
		INSERT INTO corpus_snapshots (id, corpus_id, manifest_sha256, created_by, created_at)
		VALUES ($1, $2, $3, $4, $5)`, request.snapshotID, request.corpusID, manifest, request.actor, publishedAt); err != nil {
		return domain.Publication{}, fmt.Errorf("insert corpus snapshot: %w", err)
	}
	for _, member := range request.members {
		if _, err := transaction.Exec(ctx, `
			INSERT INTO corpus_snapshot_documents (
				snapshot_id, corpus_id, source_id, source_revision_id, document_id,
				official_origin, captured_at, content_sha256
			) VALUES ($1, $2, $3, $4, $5, $6, $7, $8)`,
			request.snapshotID, request.corpusID, member.SourceID, member.SourceRevisionID, member.DocumentID,
			member.OfficialOrigin, member.CapturedAt.UTC(), member.ContentSHA256,
		); err != nil {
			return domain.Publication{}, fmt.Errorf("insert snapshot member: %w", err)
		}
	}
	release := domain.Release{CorpusID: request.corpusID, SnapshotID: request.snapshotID, Version: 1, ActivatedAt: publishedAt}
	if request.previousRelease.SnapshotID == uuid.Nil {
		if _, err := transaction.Exec(ctx, `
			INSERT INTO corpus_snapshot_releases (corpus_id, snapshot_id, version, activated_at)
			VALUES ($1, $2, $3, $4)`, request.corpusID, request.snapshotID, release.Version, publishedAt); err != nil {
			return domain.Publication{}, fmt.Errorf("insert snapshot release: %w", err)
		}
	} else {
		release.Version = request.previousRelease.Version + 1
		if _, err := transaction.Exec(ctx, `
			UPDATE corpus_snapshot_releases
			SET snapshot_id = $2, version = $3, activated_at = $4
			WHERE corpus_id = $1`, request.corpusID, request.snapshotID, release.Version, publishedAt); err != nil {
			return domain.Publication{}, fmt.Errorf("activate corpus snapshot: %w", err)
		}
	}
	return domain.Publication{
		Snapshot: domain.Snapshot{ID: request.snapshotID, CorpusID: request.corpusID, ManifestSHA256: manifest, CreatedBy: request.actor, CreatedAt: publishedAt, Members: request.members},
		Release:  release,
		Created:  true,
	}, nil
}

func (repository *Repository) writeStagedSnapshot(
	ctx context.Context,
	transaction pgx.Tx,
	request snapshotStageRequest,
) (domain.Snapshot, bool, error) {
	if len(request.members) == 0 {
		return domain.Snapshot{}, false, domain.ErrCandidateNotReady
	}
	manifest := manifestSHA256(request.members)
	existing, err := repository.findByManifest(ctx, transaction, request.corpusID, manifest)
	if err == nil {
		return existing, false, nil
	}
	if !errors.Is(err, domain.ErrNotFound) {
		return domain.Snapshot{}, false, err
	}
	stagedAt := request.stagedAt.UTC()
	if _, err := transaction.Exec(ctx, `
		INSERT INTO corpus_snapshots (id, corpus_id, manifest_sha256, created_by, created_at)
		VALUES ($1, $2, $3, $4, $5)`, request.snapshotID, request.corpusID, manifest, request.actor, stagedAt); err != nil {
		return domain.Snapshot{}, false, fmt.Errorf("insert staged corpus snapshot: %w", err)
	}
	for _, member := range request.members {
		if _, err := transaction.Exec(ctx, `
			INSERT INTO corpus_snapshot_documents (
				snapshot_id, corpus_id, source_id, source_revision_id, document_id,
				official_origin, captured_at, content_sha256
			) VALUES ($1, $2, $3, $4, $5, $6, $7, $8)`,
			request.snapshotID, request.corpusID, member.SourceID, member.SourceRevisionID, member.DocumentID,
			member.OfficialOrigin, member.CapturedAt.UTC(), member.ContentSHA256,
		); err != nil {
			return domain.Snapshot{}, false, fmt.Errorf("insert staged snapshot member: %w", err)
		}
	}
	return domain.Snapshot{
		ID: request.snapshotID, CorpusID: request.corpusID, ManifestSHA256: manifest,
		CreatedBy: request.actor, CreatedAt: stagedAt, Members: request.members,
	}, true, nil
}

func (repository *Repository) requireReadyGraphRelease(
	ctx context.Context,
	transaction pgx.Tx,
	corpusID, snapshotID uuid.UUID,
) error {
	var ready bool
	if err := transaction.QueryRow(ctx, `
		SELECT EXISTS (
			SELECT 1 FROM graph_releases
			WHERE corpus_id = $1 AND snapshot_id = $2 AND status = 'ready'
		)`, corpusID, snapshotID).Scan(&ready); err != nil {
		return fmt.Errorf("verify graph release readiness: %w", err)
	}
	if !ready {
		return domain.ErrGraphReleaseNotReady
	}
	return nil
}

func (repository *Repository) findByManifest(ctx context.Context, queryer queryer, corpusID uuid.UUID, manifest string) (domain.Snapshot, error) {
	snapshot, err := repository.readSnapshot(ctx, queryer, corpusID, uuid.Nil, manifest)
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.Snapshot{}, domain.ErrNotFound
	}
	return snapshot, err
}

func (repository *Repository) readSnapshot(ctx context.Context, queryer queryer, corpusID, snapshotID uuid.UUID, manifest ...string) (domain.Snapshot, error) {
	query := `SELECT id, corpus_id, manifest_sha256, created_by, created_at FROM corpus_snapshots WHERE corpus_id = $1`
	args := []any{corpusID}
	if len(manifest) > 0 {
		query += " AND manifest_sha256 = $2"
		args = append(args, manifest[0])
	} else {
		query += " AND id = $2"
		args = append(args, snapshotID)
	}
	var snapshot domain.Snapshot
	if err := queryer.QueryRow(ctx, query, args...).Scan(&snapshot.ID, &snapshot.CorpusID, &snapshot.ManifestSHA256, &snapshot.CreatedBy, &snapshot.CreatedAt); err != nil {
		return domain.Snapshot{}, err
	}
	members, err := repository.membersForSnapshot(ctx, queryer, snapshot.ID)
	if err != nil {
		return domain.Snapshot{}, err
	}
	snapshot.Members = members
	return snapshot, nil
}

func (repository *Repository) membersForSnapshot(ctx context.Context, queryer queryer, snapshotID uuid.UUID) ([]domain.Member, error) {
	rows, err := queryer.Query(ctx, `
		SELECT source_id, source_revision_id, document_id, official_origin, captured_at, content_sha256
		FROM corpus_snapshot_documents
		WHERE snapshot_id = $1
		ORDER BY source_id`, snapshotID)
	if err != nil {
		return nil, fmt.Errorf("list snapshot members: %w", err)
	}
	defer rows.Close()
	members, err := pgx.CollectRows(rows, func(row pgx.CollectableRow) (domain.Member, error) {
		var member domain.Member
		err := row.Scan(&member.SourceID, &member.SourceRevisionID, &member.DocumentID, &member.OfficialOrigin, &member.CapturedAt, &member.ContentSHA256)
		return member, err
	})
	if err != nil {
		return nil, fmt.Errorf("scan snapshot members: %w", err)
	}
	return members, nil
}

func (repository *Repository) readyMembers(ctx context.Context, queryer queryer, corpusID uuid.UUID) ([]domain.Member, error) {
	rows, err := queryer.Query(ctx, `
		SELECT s.id, d.id
		FROM sources s
		JOIN document_versions d ON d.id = s.latest_ready_document_id
		WHERE s.corpus_id = $1 AND d.publication_status = 'published'
		ORDER BY s.id`, corpusID)
	if err != nil {
		return nil, fmt.Errorf("list ready snapshot candidates: %w", err)
	}
	defer rows.Close()
	var pairs []struct{ sourceID, documentID uuid.UUID }
	for rows.Next() {
		var pair struct{ sourceID, documentID uuid.UUID }
		if err := rows.Scan(&pair.sourceID, &pair.documentID); err != nil {
			return nil, fmt.Errorf("scan ready snapshot candidate: %w", err)
		}
		pairs = append(pairs, pair)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate ready snapshot candidates: %w", err)
	}
	members := make([]domain.Member, 0, len(pairs))
	for _, pair := range pairs {
		member, err := repository.candidateMember(ctx, queryer, corpusID, pair.sourceID, pair.documentID)
		if err != nil {
			return nil, err
		}
		members = append(members, member)
	}
	return members, nil
}

func (repository *Repository) candidateMember(ctx context.Context, queryer queryer, corpusID, sourceID, documentID uuid.UUID) (domain.Member, error) {
	var member domain.Member
	err := queryer.QueryRow(ctx, `
		SELECT d.source_id, d.source_revision_id, d.id,
		       COALESCE(u.normalized_url, p.sha256), r.captured_at, r.content_sha256
		FROM document_versions d
		JOIN source_revisions r ON r.id = d.source_revision_id
		  AND r.corpus_id = d.corpus_id AND r.source_id = d.source_id
		LEFT JOIN url_origins u ON u.corpus_id = d.corpus_id AND u.source_id = d.source_id
		LEFT JOIN pdf_origins p ON p.corpus_id = d.corpus_id AND p.source_id = d.source_id
		WHERE d.corpus_id = $1 AND d.source_id = $2 AND d.id = $3
		  AND d.publication_status = 'published'`, corpusID, sourceID, documentID,
	).Scan(&member.SourceID, &member.SourceRevisionID, &member.DocumentID, &member.OfficialOrigin, &member.CapturedAt, &member.ContentSHA256)
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.Member{}, domain.ErrNotFound
	}
	if err != nil {
		return domain.Member{}, fmt.Errorf("read snapshot candidate: %w", err)
	}
	if err := repository.validateCandidate(ctx, queryer, member.DocumentID); err != nil {
		return domain.Member{}, err
	}
	return member, nil
}

func (repository *Repository) validateCandidate(ctx context.Context, queryer queryer, documentID uuid.UUID) error {
	var unitCount, chunkCount, readyChunkCount int
	if err := queryer.QueryRow(ctx, `
		SELECT
			(SELECT count(*) FROM document_units WHERE document_id = $1),
			(SELECT count(*) FROM retrieval_chunks WHERE document_id = $1),
			(SELECT count(*) FROM retrieval_chunks WHERE document_id = $1 AND enrichment_status = 'ready' AND embedding IS NOT NULL)`, documentID,
	).Scan(&unitCount, &chunkCount, &readyChunkCount); err != nil {
		return fmt.Errorf("validate snapshot candidate: %w", err)
	}
	if unitCount == 0 || chunkCount == 0 || chunkCount != readyChunkCount {
		return domain.ErrCandidateNotReady
	}
	return nil
}

func manifestSHA256(members []domain.Member) string {
	ordered := append([]domain.Member(nil), members...)
	sort.Slice(ordered, func(left, right int) bool { return ordered[left].SourceID.String() < ordered[right].SourceID.String() })
	lines := make([]string, 0, len(ordered))
	for _, member := range ordered {
		lines = append(lines, strings.Join([]string{
			member.SourceID.String(), member.SourceRevisionID.String(), member.DocumentID.String(),
			member.OfficialOrigin, member.CapturedAt.UTC().Format(time.RFC3339Nano), member.ContentSHA256,
		}, "|"))
	}
	digest := sha256.Sum256([]byte(strings.Join(lines, "\n")))
	return hex.EncodeToString(digest[:])
}

func replaceMember(members []domain.Member, candidate domain.Member) []domain.Member {
	updated := append([]domain.Member(nil), members...)
	for index, member := range updated {
		if member.SourceID == candidate.SourceID {
			updated[index] = candidate
			return updated
		}
	}
	return append(updated, candidate)
}

func scanRelease(row interface{ Scan(...any) error }) (domain.Release, error) {
	var release domain.Release
	err := row.Scan(&release.CorpusID, &release.SnapshotID, &release.Version, &release.ActivatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.Release{}, domain.ErrNotFound
	}
	return release, err
}
