// Package postgres persists and reads immutable corpus opening-suggestion projections.
package postgres

import (
	"context"
	"errors"
	"fmt"

	"github.com/dharlanoliveira/norvii/apps/api/internal/suggestions/domain"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

type queryer interface {
	Query(context.Context, string, ...any) (pgx.Rows, error)
	QueryRow(context.Context, string, ...any) pgx.Row
}

type database interface {
	queryer
	Begin(context.Context) (pgx.Tx, error)
}

// ReadResult is the safe, researcher-facing projection for one corpus at one read instant.
// A nil ActiveSnapshot and an empty Suggestions slice are normal unavailable outcomes.
type ReadResult struct {
	CorpusID       uuid.UUID
	ActiveSnapshot *domain.Snapshot
	Suggestions    []domain.PublicOpeningSuggestion
}

// Repository owns append-only projection persistence and release-compatible reads.
type Repository struct {
	database database
}

// NewRepository constructs opening-suggestion persistence around caller-owned connections.
func NewRepository(database database) *Repository { return &Repository{database: database} }

// AppendProjection atomically rechecks the active release while holding its row lock, then
// appends the fully validated safe projection. A newer release either wins before the lock
// (and is rejected) or after commit (and hides this snapshot-bound projection on reads).
func (repository *Repository) AppendProjection(ctx context.Context, projection domain.Projection) error {
	if repository == nil || repository.database == nil {
		return domain.ErrInvalidInput
	}
	if err := validateProjection(projection); err != nil {
		return err
	}

	transaction, err := repository.database.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin opening suggestion projection append: %w", err)
	}
	defer func() { _ = transaction.Rollback(ctx) }()

	activeSnapshot, err := lockActiveSnapshot(ctx, transaction, projection.Set.CorpusID)
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.ErrSnapshotMismatch
	}
	if err != nil {
		return fmt.Errorf("lock opening suggestion active release: %w", err)
	}
	if !projection.MatchesActiveSnapshot(activeSnapshot) {
		return domain.ErrSnapshotMismatch
	}
	if err := insertProjectionSet(ctx, transaction, projection.Set); err != nil {
		return err
	}
	if err := insertProjectionItems(ctx, transaction, projection.Items); err != nil {
		return err
	}
	if err := transaction.Commit(ctx); err != nil {
		return fmt.Errorf("commit opening suggestion projection append: %w", err)
	}
	return nil
}

// Read returns only the exact-language suggestions for the corpus's active immutable release.
// Missing, disabled, stale, and unpublished states are intentionally represented as empty data.
func (repository *Repository) Read(
	ctx context.Context,
	corpusID uuid.UUID,
	language domain.QueryLanguage,
) (ReadResult, error) {
	result := ReadResult{CorpusID: corpusID, Suggestions: make([]domain.PublicOpeningSuggestion, 0)}
	if repository == nil || repository.database == nil || corpusID == uuid.Nil || !validQueryLanguage(language) {
		return result, domain.ErrInvalidInput
	}

	rows, err := repository.database.Query(ctx, `
		WITH active_release AS (
			SELECT release.corpus_id, release.snapshot_id, snapshot.manifest_sha256
			FROM corpus_snapshot_releases AS release
			JOIN corpora AS corpus ON corpus.id = release.corpus_id
			JOIN corpus_snapshots AS snapshot
			  ON snapshot.id = release.snapshot_id
			 AND snapshot.corpus_id = release.corpus_id
			WHERE release.corpus_id = $1
			  AND corpus.status = 'enabled'
		), latest_set AS (
			SELECT suggestion_set.id, suggestion_set.corpus_id, suggestion_set.dataset_revision_id
			FROM corpus_opening_suggestion_set AS suggestion_set
			JOIN active_release
			  ON active_release.corpus_id = suggestion_set.corpus_id
			 AND active_release.snapshot_id = suggestion_set.snapshot_id
			 AND active_release.manifest_sha256 = suggestion_set.snapshot_manifest_sha256
			ORDER BY suggestion_set.created_at DESC, suggestion_set.id DESC
			LIMIT 1
		), matched_set AS (
			SELECT latest_set.id, latest_set.corpus_id, latest_set.dataset_revision_id
			FROM latest_set
			JOIN LATERAL (
				SELECT publication.review_decision, publication.publication_state
				FROM evaluation_dataset_publication AS publication
				WHERE publication.dataset_revision_id = latest_set.dataset_revision_id
				  AND publication.corpus_id = latest_set.corpus_id
				ORDER BY publication.reviewed_at DESC, publication.id DESC
				LIMIT 1
			) AS latest_publication
			  ON latest_publication.review_decision = 'approved'
			 AND latest_publication.publication_state = 'available'
		)
		SELECT active_release.snapshot_id,
		       active_release.manifest_sha256,
		       COALESCE(item.rank, 0),
		       COALESCE(evaluation_case.external_case_id, ''),
		       COALESCE(item.question, '')
		FROM active_release
		LEFT JOIN matched_set ON true
		LEFT JOIN corpus_opening_suggestion_item AS item
		  ON item.suggestion_set_id = matched_set.id
		 AND item.corpus_id = matched_set.corpus_id
		 AND item.dataset_revision_id = matched_set.dataset_revision_id
		 AND item.query_language = $2
		LEFT JOIN evaluation_dataset_case AS evaluation_case
		  ON evaluation_case.id = item.evaluation_case_id
		 AND evaluation_case.corpus_id = item.corpus_id
		 AND evaluation_case.dataset_revision_id = item.dataset_revision_id
		ORDER BY item.rank NULLS LAST`, corpusID, string(language))
	if err != nil {
		return result, fmt.Errorf("read opening suggestions: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var (
			snapshotID     uuid.UUID
			manifestSHA256 string
			rank           int
			externalCaseID string
			question       string
		)
		if err := rows.Scan(&snapshotID, &manifestSHA256, &rank, &externalCaseID, &question); err != nil {
			return result, fmt.Errorf("scan opening suggestions: %w", err)
		}
		if result.ActiveSnapshot == nil {
			result.ActiveSnapshot = &domain.Snapshot{
				ID: snapshotID, CorpusID: corpusID, ManifestSHA256: domain.SHA256(manifestSHA256),
			}
		}
		if rank == 0 {
			continue
		}
		if externalCaseID == "" || question == "" || rank < 1 || rank > 5 {
			return ReadResult{}, fmt.Errorf("read opening suggestions: %w", domain.ErrInvalidSelection)
		}
		result.Suggestions = append(result.Suggestions, domain.PublicOpeningSuggestion{
			CaseID: externalCaseID, Rank: rank, Question: question,
		})
	}
	if err := rows.Err(); err != nil {
		return result, fmt.Errorf("iterate opening suggestions: %w", err)
	}
	return result, nil
}

func validateProjection(projection domain.Projection) error {
	set := projection.Set
	if set.ID == uuid.Nil || set.CorpusID == uuid.Nil || set.SnapshotID == uuid.Nil || set.DatasetRevisionID == uuid.Nil ||
		set.SnapshotManifestSHA256.Validate() != nil || set.DatasetContentSHA256.Validate() != nil ||
		set.SelectionPolicyVersion == "" || set.PublishedBy == "" || len(projection.Items) == 0 {
		return domain.ErrInvalidInput
	}
	for _, item := range projection.Items {
		if item.SuggestionSetID != set.ID || item.CorpusID != set.CorpusID || item.DatasetRevisionID != set.DatasetRevisionID {
			return domain.ErrCorpusMismatch
		}
		if item.CaseID == uuid.Nil || item.Rank < 1 || item.Rank > 5 || item.CaseChecksum.Validate() != nil ||
			!validQueryLanguage(item.QueryLanguage) || item.ExternalCaseID == "" || item.Question == "" {
			return domain.ErrInvalidSelection
		}
	}
	return nil
}

func lockActiveSnapshot(ctx context.Context, transaction pgx.Tx, corpusID uuid.UUID) (domain.Snapshot, error) {
	var snapshot domain.Snapshot
	err := transaction.QueryRow(ctx, `
		SELECT release.snapshot_id, release.corpus_id, snapshot.manifest_sha256
		FROM corpus_snapshot_releases AS release
		JOIN corpora AS corpus ON corpus.id = release.corpus_id
		JOIN corpus_snapshots AS snapshot
		  ON snapshot.id = release.snapshot_id
		 AND snapshot.corpus_id = release.corpus_id
		WHERE release.corpus_id = $1
		  AND corpus.status = 'enabled'
		FOR UPDATE OF release`, corpusID).Scan(&snapshot.ID, &snapshot.CorpusID, &snapshot.ManifestSHA256)
	return snapshot, err
}

func insertProjectionSet(ctx context.Context, transaction pgx.Tx, set domain.OpeningSuggestionSet) error {
	_, err := transaction.Exec(ctx, `
		INSERT INTO corpus_opening_suggestion_set (
			id, corpus_id, snapshot_id, snapshot_manifest_sha256, dataset_revision_id,
			source_dataset_content_sha256, selection_policy_version, published_by
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8)`,
		set.ID, set.CorpusID, set.SnapshotID, string(set.SnapshotManifestSHA256), set.DatasetRevisionID,
		string(set.DatasetContentSHA256), set.SelectionPolicyVersion, set.PublishedBy,
	)
	if err != nil {
		return fmt.Errorf("insert opening suggestion set: %w", err)
	}
	return nil
}

func insertProjectionItems(ctx context.Context, transaction pgx.Tx, items []domain.OpeningSuggestionItem) error {
	for _, item := range items {
		_, err := transaction.Exec(ctx, `
			INSERT INTO corpus_opening_suggestion_item (
				id, suggestion_set_id, corpus_id, dataset_revision_id, rank, evaluation_case_id,
				case_checksum, query_language, question
			) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)`,
			uuid.New(), item.SuggestionSetID, item.CorpusID, item.DatasetRevisionID, item.Rank, item.CaseID,
			string(item.CaseChecksum), string(item.QueryLanguage), item.Question,
		)
		if err != nil {
			return fmt.Errorf("insert opening suggestion item: %w", err)
		}
	}
	return nil
}

func validQueryLanguage(language domain.QueryLanguage) bool {
	return language == domain.QueryLanguageEnglish || language == domain.QueryLanguagePortuguese
}

var _ interface {
	AppendProjection(context.Context, domain.Projection) error
} = (*Repository)(nil)
