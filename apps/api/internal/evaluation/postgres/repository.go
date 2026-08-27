// Package postgres persists corpus-bound evaluation source bindings and resolves legal locators.
package postgres

import (
	"context"
	"errors"
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/dharlanoliveira/norvii/apps/api/internal/evaluation/application"
	"github.com/dharlanoliveira/norvii/apps/api/internal/evaluation/domain"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
)

var (
	// ErrInvalidInput identifies malformed binding or locator inputs.
	ErrInvalidInput = errors.New("evaluation repository input is invalid")
	// ErrSourceRequirementNotFound covers absent and foreign-owned dataset source requirements.
	ErrSourceRequirementNotFound = errors.New("evaluation dataset source requirement not found")
	// ErrCorpusSourceNotFound covers absent and foreign-owned corpus sources.
	ErrCorpusSourceNotFound = errors.New("evaluation corpus source not found")
	// ErrSourceAlreadyBound prevents duplicate binding and later mutation of a source alias.
	ErrSourceAlreadyBound = errors.New("evaluation dataset source alias is already bound")
	// ErrLocatorNotFound covers missing, foreign, or snapshot-nonmember locator targets.
	ErrLocatorNotFound = errors.New("evaluation snapshot locator not found")
)

var (
	sourceAliasPattern      = regexp.MustCompile(`^[a-z0-9][a-z0-9-]{0,127}$`)
	canonicalLocatorPattern = regexp.MustCompile(`^[a-z][a-z-]*:[a-z0-9.-]+(/[a-z][a-z-]*:[a-z0-9.-]+)*$`)
)

type queryer interface {
	Query(context.Context, string, ...any) (pgx.Rows, error)
	QueryRow(context.Context, string, ...any) pgx.Row
}

type database interface {
	queryer
	Begin(context.Context) (pgx.Tx, error)
}

// Repository owns source-alias binding and snapshot-scoped legal-locator resolution.
type Repository struct {
	database database
}

// NewRepository constructs an evaluation repository around caller-owned persistence.
func NewRepository(database database) *Repository { return &Repository{database: database} }

// SourceBinding identifies one immutable manifest alias binding in one dataset corpus.
type SourceBinding struct {
	DatasetRevisionID uuid.UUID
	CorpusID          uuid.UUID
	SourceAlias       string
	CorpusSourceID    uuid.UUID
}

// BindSource binds an unbound manifest source alias exactly once to a source in its own corpus.
func (repository *Repository) BindSource(ctx context.Context, binding SourceBinding) (SourceBinding, error) {
	if err := binding.validate(); err != nil {
		return SourceBinding{}, err
	}

	var persisted SourceBinding
	err := repository.database.QueryRow(ctx, `
		UPDATE evaluation_dataset_source AS requirement
		SET corpus_source_id = source.id
		FROM evaluation_dataset_revision AS revision
		JOIN sources AS source
		  ON source.id = $4
		 AND source.corpus_id = revision.corpus_id
		WHERE requirement.dataset_revision_id = $1
		  AND requirement.corpus_id = $2
		  AND requirement.source_alias = $3
		  AND requirement.corpus_source_id IS NULL
		  AND revision.id = requirement.dataset_revision_id
		  AND revision.corpus_id = requirement.corpus_id
		RETURNING requirement.dataset_revision_id, requirement.corpus_id,
		          requirement.source_alias, requirement.corpus_source_id`,
		binding.DatasetRevisionID, binding.CorpusID, binding.SourceAlias, binding.CorpusSourceID,
	).Scan(&persisted.DatasetRevisionID, &persisted.CorpusID, &persisted.SourceAlias, &persisted.CorpusSourceID)
	if err == nil {
		return persisted, nil
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		return SourceBinding{}, fmt.Errorf("bind evaluation dataset source: %w", err)
	}
	if err := repository.bindingFailure(ctx, binding); err != nil {
		return SourceBinding{}, err
	}
	return SourceBinding{}, fmt.Errorf("bind evaluation dataset source: %w", ErrSourceRequirementNotFound)
}

// BindDatasetSource adapts the repository's explicit binding API to the application boundary.
func (repository *Repository) BindDatasetSource(
	ctx context.Context,
	binding application.SourceBinding,
) (application.SourceBinding, error) {
	persisted, err := repository.BindSource(ctx, SourceBinding{
		DatasetRevisionID: binding.DatasetRevisionID, CorpusID: binding.CorpusID,
		SourceAlias: binding.SourceAlias, CorpusSourceID: binding.CorpusSourceID,
	})
	if err != nil {
		return application.SourceBinding{}, err
	}
	return application.SourceBinding{
		DatasetRevisionID: persisted.DatasetRevisionID, CorpusID: persisted.CorpusID,
		SourceAlias: persisted.SourceAlias, CorpusSourceID: persisted.CorpusSourceID,
	}, nil
}

// AppendPublication stores one immutable review/publication record. Lifecycle validation is owned
// by the application service; the database append-only trigger protects persisted history.
func (repository *Repository) AppendPublication(ctx context.Context, publication domain.Publication) error {
	if repository == nil || repository.database == nil || publication.Validate() != nil {
		return application.ErrPublicationStore
	}
	transaction, err := repository.database.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin evaluation dataset publication append: %w", err)
	}
	defer func() { _ = transaction.Rollback(ctx) }()
	var revisionID uuid.UUID
	err = transaction.QueryRow(ctx, `
		SELECT id
		FROM evaluation_dataset_revision
		WHERE id = $1 AND corpus_id = $2
		FOR UPDATE`, publication.DatasetRevisionID, publication.CorpusID,
	).Scan(&revisionID)
	if errors.Is(err, pgx.ErrNoRows) {
		return application.ErrPublicationStore
	}
	if err != nil {
		return fmt.Errorf("lock evaluation dataset publication revision: %w", err)
	}
	latest, found, err := readLatestPublication(ctx, transaction, publication.CorpusID, publication.DatasetRevisionID)
	if err != nil {
		return err
	}
	if found && domain.ValidatePublicationTransition(latest.PublicationState, publication.PublicationState) != nil {
		return domain.ErrInvalidLifecycleTransition
	}
	var persistedID uuid.UUID
	err = transaction.QueryRow(ctx, `
		INSERT INTO evaluation_dataset_publication (
			id, dataset_revision_id, corpus_id, review_decision, reviewer_identity,
			review_note, publication_state, reviewed_at
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		RETURNING id`,
		publication.ID, publication.DatasetRevisionID, publication.CorpusID, string(publication.ReviewDecision),
		publication.ReviewerIdentity, publication.ReviewNote, string(publication.PublicationState), publication.ReviewedAt,
	).Scan(&persistedID)
	if errors.Is(err, pgx.ErrNoRows) {
		return application.ErrPublicationStore
	}
	if err != nil {
		return fmt.Errorf("insert evaluation dataset publication: %w", err)
	}
	if err := transaction.Commit(ctx); err != nil {
		return fmt.Errorf("commit evaluation dataset publication append: %w", err)
	}
	return nil
}

// LatestPublication returns the one state that controls dataset availability. Ordering by review
// timestamp and record ID matches the append-only catalog convention used by suggestion reads.
func (repository *Repository) LatestPublication(
	ctx context.Context,
	corpusID uuid.UUID,
	datasetRevisionID uuid.UUID,
) (domain.Publication, bool, error) {
	if repository == nil || repository.database == nil || corpusID == uuid.Nil || datasetRevisionID == uuid.Nil {
		return domain.Publication{}, false, application.ErrPublicationStore
	}
	publication, found, err := readLatestPublication(ctx, repository.database, corpusID, datasetRevisionID)
	if err != nil {
		return domain.Publication{}, false, err
	}
	return publication, found, nil
}

func readLatestPublication(
	ctx context.Context,
	database queryer,
	corpusID uuid.UUID,
	datasetRevisionID uuid.UUID,
) (domain.Publication, bool, error) {
	var publication domain.Publication
	err := database.QueryRow(ctx, `
		SELECT id, dataset_revision_id, corpus_id, review_decision, reviewer_identity,
		       review_note, publication_state, reviewed_at
		FROM evaluation_dataset_publication
		WHERE corpus_id = $1 AND dataset_revision_id = $2
		ORDER BY reviewed_at DESC, id DESC
		LIMIT 1`, corpusID, datasetRevisionID,
	).Scan(
		&publication.ID, &publication.DatasetRevisionID, &publication.CorpusID, &publication.ReviewDecision,
		&publication.ReviewerIdentity, &publication.ReviewNote, &publication.PublicationState, &publication.ReviewedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.Publication{}, false, nil
	}
	if err != nil {
		return domain.Publication{}, false, fmt.Errorf("read latest evaluation dataset publication: %w", err)
	}
	return publication, true, nil
}

// DatasetCorpus returns the immutable owner of a dataset revision without accepting a caller
// supplied corpus filter, so preflight can identify cross-corpus selection explicitly.
func (repository *Repository) DatasetCorpus(ctx context.Context, datasetRevisionID uuid.UUID) (uuid.UUID, error) {
	if repository == nil || repository.database == nil || datasetRevisionID == uuid.Nil {
		return uuid.Nil, application.ErrPreflightCorpusMismatch
	}
	var corpusID uuid.UUID
	err := repository.database.QueryRow(ctx, `
		SELECT corpus_id
		FROM evaluation_dataset_revision
		WHERE id = $1`, datasetRevisionID,
	).Scan(&corpusID)
	if errors.Is(err, pgx.ErrNoRows) {
		return uuid.Nil, application.ErrDatasetNotFound
	}
	if err != nil {
		return uuid.Nil, fmt.Errorf("read evaluation dataset corpus: %w", err)
	}
	return corpusID, nil
}

// SourceRequirements returns all manifest sources, including those not used by an initial case
// set, because each one must be bound and present in a compatible snapshot.
func (repository *Repository) SourceRequirements(
	ctx context.Context,
	corpusID uuid.UUID,
	datasetRevisionID uuid.UUID,
) ([]application.SourceRequirement, error) {
	if repository == nil || repository.database == nil || corpusID == uuid.Nil || datasetRevisionID == uuid.Nil {
		return nil, application.ErrInvalidPreflightRequest
	}
	rows, err := repository.database.Query(ctx, `
		SELECT id, corpus_id, dataset_revision_id, source_alias, corpus_source_id
		FROM evaluation_dataset_source
		WHERE corpus_id = $1 AND dataset_revision_id = $2
		ORDER BY source_alias ASC`, corpusID, datasetRevisionID)
	if err != nil {
		return nil, fmt.Errorf("read evaluation dataset source requirements: %w", err)
	}
	defer rows.Close()
	requirements := make([]application.SourceRequirement, 0)
	for rows.Next() {
		var requirement application.SourceRequirement
		if err := rows.Scan(&requirement.ID, &requirement.CorpusID, &requirement.DatasetRevisionID, &requirement.SourceAlias, &requirement.CorpusSourceID); err != nil {
			return nil, fmt.Errorf("scan evaluation dataset source requirement: %w", err)
		}
		requirements = append(requirements, requirement)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate evaluation dataset source requirements: %w", err)
	}
	return requirements, nil
}

// ExpectedEvidence returns every reviewed target with its stored exact canonical locator.
// No title, URL, text, or structural-label fallback is introduced by preflight.
func (repository *Repository) ExpectedEvidence(
	ctx context.Context,
	corpusID uuid.UUID,
	datasetRevisionID uuid.UUID,
) ([]application.ExpectedEvidenceRequirement, error) {
	if repository == nil || repository.database == nil || corpusID == uuid.Nil || datasetRevisionID == uuid.Nil {
		return nil, application.ErrInvalidPreflightRequest
	}
	rows, err := repository.database.Query(ctx, `
		SELECT id, corpus_id, dataset_revision_id, evaluation_case_id, source_alias, display_locator,
		       COALESCE(canonical_locator, '')
		FROM evaluation_case_expected_evidence
		WHERE corpus_id = $1 AND dataset_revision_id = $2
		ORDER BY evaluation_case_id ASC, ordinal ASC`, corpusID, datasetRevisionID)
	if err != nil {
		return nil, fmt.Errorf("read evaluation expected evidence: %w", err)
	}
	defer rows.Close()
	requirements := make([]application.ExpectedEvidenceRequirement, 0)
	for rows.Next() {
		var requirement application.ExpectedEvidenceRequirement
		if err := rows.Scan(&requirement.ID, &requirement.CorpusID, &requirement.DatasetRevisionID, &requirement.CaseID, &requirement.SourceAlias, &requirement.DisplayLocator, &requirement.CanonicalLocator); err != nil {
			return nil, fmt.Errorf("scan evaluation expected evidence: %w", err)
		}
		requirements = append(requirements, requirement)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate evaluation expected evidence: %w", err)
	}
	return requirements, nil
}

// SnapshotContainsSource verifies named immutable snapshot membership for a bound source without
// falling back to the corpus's active snapshot.
func (repository *Repository) SnapshotContainsSource(
	ctx context.Context,
	corpusID uuid.UUID,
	snapshotID uuid.UUID,
	sourceID uuid.UUID,
) (bool, error) {
	if repository == nil || repository.database == nil || corpusID == uuid.Nil || snapshotID == uuid.Nil || sourceID == uuid.Nil {
		return false, application.ErrInvalidPreflightRequest
	}
	var found bool
	err := repository.database.QueryRow(ctx, `
		SELECT EXISTS (
			SELECT 1
			FROM corpus_snapshots AS snapshot
			JOIN corpus_snapshot_documents AS member
			  ON member.snapshot_id = snapshot.id
			 AND member.corpus_id = snapshot.corpus_id
			WHERE snapshot.id = $1
			  AND snapshot.corpus_id = $2
			  AND member.source_id = $3
		)`, snapshotID, corpusID, sourceID,
	).Scan(&found)
	if err != nil {
		return false, fmt.Errorf("check evaluation snapshot source membership: %w", err)
	}
	return found, nil
}

// ResolvePreflightLocator adapts the Task 007 repository API to the application port without
// exposing a persistence package dependency to application code.
func (repository *Repository) ResolvePreflightLocator(
	ctx context.Context,
	request application.LocatorRequest,
) (application.ResolvedLocator, error) {
	resolved, err := repository.ResolveLocator(ctx, LocatorRequest{
		DatasetRevisionID: request.DatasetRevisionID, CorpusID: request.CorpusID, SnapshotID: request.SnapshotID,
		SourceAlias: request.SourceAlias, CanonicalLocator: request.CanonicalLocator, DisplayLocator: request.DisplayLocator,
	})
	if err != nil {
		return application.ResolvedLocator{}, err
	}
	return application.ResolvedLocator{
		CorpusID: resolved.CorpusID, SnapshotID: resolved.SnapshotID, SourceID: resolved.SourceID,
		SourceRevisionID: resolved.SourceRevisionID, DocumentID: resolved.DocumentID, UnitID: resolved.UnitID,
		CanonicalLocator: resolved.CanonicalLocator, DisplayLocator: resolved.DisplayLocator,
		ContentSHA256: resolved.ContentProvenance.ContentSHA256,
	}, nil
}

func (binding SourceBinding) validate() error {
	if binding.DatasetRevisionID == uuid.Nil || binding.CorpusID == uuid.Nil || binding.CorpusSourceID == uuid.Nil ||
		binding.SourceAlias != strings.TrimSpace(binding.SourceAlias) || !sourceAliasPattern.MatchString(binding.SourceAlias) {
		return ErrInvalidInput
	}
	return nil
}

func (repository *Repository) bindingFailure(ctx context.Context, binding SourceBinding) error {
	var bound pgtype.UUID
	err := repository.database.QueryRow(ctx, `
		SELECT requirement.corpus_source_id
		FROM evaluation_dataset_source AS requirement
		JOIN evaluation_dataset_revision AS revision
		  ON revision.id = requirement.dataset_revision_id
		 AND revision.corpus_id = requirement.corpus_id
		WHERE requirement.dataset_revision_id = $1
		  AND requirement.corpus_id = $2
		  AND requirement.source_alias = $3`,
		binding.DatasetRevisionID, binding.CorpusID, binding.SourceAlias,
	).Scan(&bound)
	if errors.Is(err, pgx.ErrNoRows) {
		return ErrSourceRequirementNotFound
	}
	if err != nil {
		return fmt.Errorf("read evaluation dataset source binding: %w", err)
	}
	if bound.Valid {
		return ErrSourceAlreadyBound
	}

	var sourceExists bool
	err = repository.database.QueryRow(ctx, `
		SELECT EXISTS (
			SELECT 1
			FROM sources AS source
			JOIN corpora AS corpus ON corpus.id = source.corpus_id
			WHERE source.id = $1 AND source.corpus_id = $2
		)`, binding.CorpusSourceID, binding.CorpusID,
	).Scan(&sourceExists)
	if err != nil {
		return fmt.Errorf("verify evaluation corpus source ownership: %w", err)
	}
	if !sourceExists {
		return ErrCorpusSourceNotFound
	}
	return ErrSourceAlreadyBound
}

// LocatorRequest identifies one reviewed atomic legal locator in an explicit immutable snapshot.
type LocatorRequest struct {
	DatasetRevisionID uuid.UUID
	CorpusID          uuid.UUID
	SnapshotID        uuid.UUID
	SourceAlias       string
	CanonicalLocator  string
	DisplayLocator    string
}

// ResolvedLocator preserves the immutable provenance of one snapshot member and legal unit.
type ResolvedLocator struct {
	CorpusID          uuid.UUID
	SnapshotID        uuid.UUID
	SourceID          uuid.UUID
	SourceRevisionID  uuid.UUID
	DocumentID        uuid.UUID
	UnitID            uuid.UUID
	UnitStartOffset   int
	UnitEndOffset     int
	CanonicalLocator  string
	DisplayLocator    string
	ContentProvenance ContentProvenance
}

// ContentProvenance is the immutable acquisition and extracted-content identity of one resolution.
type ContentProvenance struct {
	ContentSHA256          string
	CapturedAt             time.Time
	MediaType              string
	ByteSize               int64
	FinalURL               *string
	ExtractedContentSHA256 *string
	DocumentTextSHA256     string
	UnitContentSHA256      string
}

// ResolveLocator resolves only a normalized atomic canonical locator through a named snapshot member.
//
// The snapshot membership and canonical-locator uniqueness constraints guarantee
// at most one target for a valid request.
func (repository *Repository) ResolveLocator(ctx context.Context, request LocatorRequest) (ResolvedLocator, error) {
	if err := request.validate(); err != nil {
		return ResolvedLocator{}, err
	}
	var resolved ResolvedLocator
	err := repository.database.QueryRow(ctx, `
		SELECT member.corpus_id, member.snapshot_id, member.source_id,
		       member.source_revision_id, member.document_id, unit.id,
		       unit.start_offset, unit.end_offset, unit.canonical_locator,
		       revision.content_sha256, revision.captured_at, revision.media_type,
		       revision.byte_size, revision.final_url, revision.extracted_content_sha256,
		       document.text_sha256, unit.content_sha256
		FROM evaluation_dataset_source AS requirement
		JOIN evaluation_dataset_revision AS dataset
		  ON dataset.id = requirement.dataset_revision_id
		 AND dataset.corpus_id = requirement.corpus_id
		JOIN corpus_snapshots AS snapshot
		  ON snapshot.id = $3
		 AND snapshot.corpus_id = dataset.corpus_id
		JOIN corpus_snapshot_documents AS member
		  ON member.snapshot_id = snapshot.id
		 AND member.corpus_id = snapshot.corpus_id
		 AND member.source_id = requirement.corpus_source_id
		JOIN document_versions AS document
		  ON document.id = member.document_id
		 AND document.corpus_id = member.corpus_id
		 AND document.source_id = member.source_id
		 AND document.source_revision_id = member.source_revision_id
		JOIN source_revisions AS revision
		  ON revision.id = member.source_revision_id
		 AND revision.corpus_id = member.corpus_id
		 AND revision.source_id = member.source_id
		 AND revision.content_sha256 = member.content_sha256
		JOIN document_units AS unit
		  ON unit.document_id = document.id
		 AND unit.canonical_locator = $5
		WHERE requirement.dataset_revision_id = $1
		  AND requirement.corpus_id = $2
		  AND requirement.source_alias = $4
		  AND requirement.corpus_source_id IS NOT NULL`,
		request.DatasetRevisionID, request.CorpusID, request.SnapshotID, request.SourceAlias, request.CanonicalLocator,
	).Scan(
		&resolved.CorpusID,
		&resolved.SnapshotID,
		&resolved.SourceID,
		&resolved.SourceRevisionID,
		&resolved.DocumentID,
		&resolved.UnitID,
		&resolved.UnitStartOffset,
		&resolved.UnitEndOffset,
		&resolved.CanonicalLocator,
		&resolved.ContentProvenance.ContentSHA256,
		&resolved.ContentProvenance.CapturedAt,
		&resolved.ContentProvenance.MediaType,
		&resolved.ContentProvenance.ByteSize,
		&resolved.ContentProvenance.FinalURL,
		&resolved.ContentProvenance.ExtractedContentSHA256,
		&resolved.ContentProvenance.DocumentTextSHA256,
		&resolved.ContentProvenance.UnitContentSHA256,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return ResolvedLocator{}, ErrLocatorNotFound
	}
	if err != nil {
		return ResolvedLocator{}, fmt.Errorf("scan evaluation snapshot locator: %w", err)
	}
	resolved.DisplayLocator = request.DisplayLocator
	return resolved, nil
}

func (request LocatorRequest) validate() error {
	if request.DatasetRevisionID == uuid.Nil || request.CorpusID == uuid.Nil || request.SnapshotID == uuid.Nil ||
		request.SourceAlias != strings.TrimSpace(request.SourceAlias) || !sourceAliasPattern.MatchString(request.SourceAlias) ||
		request.CanonicalLocator != strings.TrimSpace(request.CanonicalLocator) || !canonicalLocatorPattern.MatchString(request.CanonicalLocator) ||
		strings.TrimSpace(request.DisplayLocator) == "" {
		return ErrInvalidInput
	}
	return nil
}
