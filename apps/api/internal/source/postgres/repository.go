// Package postgres reads corpus-scoped source lifecycle projections.
package postgres

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/dharlanoliveira/norvii/apps/api/internal/source/domain"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgtype"
)

// ErrNotFound intentionally covers absent and foreign-owned source identifiers.
var ErrNotFound = errors.New("source not found")

// ErrStaleState identifies a source version mismatch.
var ErrStaleState = errors.New("source state is stale")

type queryer interface {
	Query(context.Context, string, ...any) (pgx.Rows, error)
	QueryRow(context.Context, string, ...any) pgx.Row
}

type database interface {
	queryer
	Begin(context.Context) (pgx.Tx, error)
}

type rowScanner interface {
	Scan(...any) error
}

// Record is the authoritative source lifecycle read model.
type Record struct {
	ID                    uuid.UUID
	CorpusID              uuid.UUID
	Title                 string
	Kind                  domain.Kind
	ProcessingStatus      domain.Status
	FailureCategory       *string
	LatestReadyDocumentID *uuid.UUID
	Version               int
	CreatedAt             time.Time
	UpdatedAt             time.Time
	Origin                Origin
	LatestAttempt         *Attempt
	Attempts              []Attempt
}

// Origin is the safe source provenance projection without PDF bytes.
type Origin struct {
	SubmittedURL           *string
	NormalizedURL          *string
	OriginalFilename       *string
	MediaType              *string
	ByteSize               *int64
	SHA256                 *string
	FinalURL               *string
	CapturedAt             *time.Time
	ExtractedContentSHA256 *string
}

// Attempt is the newest safe processing-attempt projection.
type Attempt struct {
	Number                   int
	PipelineVersion          string
	Status                   string
	StartedAt                time.Time
	FinishedAt               *time.Time
	FailureCategory          *string
	AcquiredByteCount        *int64
	NormalizedCharacterCount *int64
	UnitCount                *int
	DurationMilliseconds     *int64
}

// Repository reads sources only through their owning corpus boundary.
type Repository struct {
	database database
}

// NewRepository constructs a source repository around a caller-owned database.
func NewRepository(database database) *Repository { return &Repository{database: database} }

// CreateURL atomically persists a source, its origin, and initial work item.
func (repository *Repository) CreateURL(
	ctx context.Context,
	source domain.Source,
	submittedURL string,
	normalizedURL string,
	workID uuid.UUID,
) (Record, error) {
	transaction, err := repository.database.Begin(ctx)
	if err != nil {
		return Record{}, fmt.Errorf("begin URL source creation: %w", err)
	}
	defer func() { _ = transaction.Rollback(ctx) }()

	var corpusStatus string
	if err := transaction.QueryRow(
		ctx, "SELECT status FROM corpora WHERE id = $1 FOR UPDATE", source.CorpusID,
	).Scan(&corpusStatus); errors.Is(err, pgx.ErrNoRows) || corpusStatus != "enabled" {
		return Record{}, domain.ErrCorpusUnavailable
	} else if err != nil {
		return Record{}, fmt.Errorf("lock source corpus: %w", err)
	}
	var count int
	if err := transaction.QueryRow(
		ctx, "SELECT count(*) FROM sources WHERE corpus_id = $1", source.CorpusID,
	).Scan(&count); err != nil {
		return Record{}, fmt.Errorf("count corpus sources: %w", err)
	}
	if err := domain.EnsureCapacity(count); err != nil {
		return Record{}, err
	}

	if _, err := transaction.Exec(ctx, `
		INSERT INTO sources (
			id, corpus_id, title, kind, processing_status, version, created_at, updated_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8)`,
		source.ID, source.CorpusID, source.Title, source.Kind, source.Status,
		source.Version, source.CreatedAt, source.UpdatedAt,
	); err != nil {
		return Record{}, classifyCreateError("insert URL source", err)
	}
	if _, err := transaction.Exec(ctx, `
		INSERT INTO url_origins (
			source_id, corpus_id, submitted_url, normalized_url, created_at
		) VALUES ($1, $2, $3, $4, $5)`,
		source.ID, source.CorpusID, submittedURL, normalizedURL, source.CreatedAt,
	); err != nil {
		return Record{}, classifyCreateError("insert URL origin", err)
	}
	if _, err := transaction.Exec(ctx, `
		INSERT INTO ingestion_work (
			id, source_id, corpus_id, reason, requested_at, created_at, updated_at
		) VALUES ($1, $2, $3, 'initial', $4, $4, $4)`,
		workID, source.ID, source.CorpusID, source.CreatedAt,
	); err != nil {
		return Record{}, classifyCreateError("insert initial ingestion work", err)
	}
	if err := transaction.Commit(ctx); err != nil {
		return Record{}, fmt.Errorf("commit URL source creation: %w", err)
	}
	return Record{
		ID: source.ID, CorpusID: source.CorpusID, Title: source.Title,
		Kind: source.Kind, ProcessingStatus: source.Status, Version: source.Version,
		CreatedAt: source.CreatedAt, UpdatedAt: source.UpdatedAt,
		Origin: Origin{SubmittedURL: &submittedURL, NormalizedURL: &normalizedURL},
	}, nil
}

// CreatePDF atomically persists a source, immutable binary origin, and initial work item.
func (repository *Repository) CreatePDF(
	ctx context.Context,
	source domain.Source,
	origin domain.PDFOrigin,
	workID uuid.UUID,
) (Record, error) {
	transaction, err := repository.database.Begin(ctx)
	if err != nil {
		return Record{}, fmt.Errorf("begin PDF source creation: %w", err)
	}
	defer func() { _ = transaction.Rollback(ctx) }()
	if err := lockAvailableCorpus(ctx, transaction, source.CorpusID); err != nil {
		return Record{}, err
	}
	if err := ensureSourceCapacity(ctx, transaction, source.CorpusID); err != nil {
		return Record{}, err
	}
	if err := insertSource(ctx, transaction, source); err != nil {
		return Record{}, err
	}
	if _, err := transaction.Exec(ctx, `
		INSERT INTO pdf_origins (
			source_id, corpus_id, original_filename, delivery_filename,
			declared_media_type, detected_media_type, byte_size, sha256, content, created_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)`,
		source.ID, source.CorpusID, origin.OriginalFilename, origin.DeliveryFilename,
		origin.DeclaredMediaType, origin.DetectedMediaType, origin.ByteSize,
		origin.SHA256, origin.Content, source.CreatedAt,
	); err != nil {
		return Record{}, classifyCreateError("insert PDF origin", err)
	}
	if err := insertInitialWork(ctx, transaction, source, workID); err != nil {
		return Record{}, err
	}
	if err := transaction.Commit(ctx); err != nil {
		return Record{}, fmt.Errorf("commit PDF source creation: %w", err)
	}
	record := recordFromSource(source)
	record.Origin = Origin{
		OriginalFilename: &origin.OriginalFilename, MediaType: &origin.DetectedMediaType,
		ByteSize: &origin.ByteSize, SHA256: &origin.SHA256,
	}
	return record, nil
}

// PDFOriginRecord is the corpus-scoped immutable PDF delivery projection.
type PDFOriginRecord struct {
	DeliveryFilename string
	MediaType        string
	Content          []byte
}

// URLOriginRecord is the safe browser destination for a registered URL source.
type URLOriginRecord struct {
	URL string
}

// GetPDFOrigin returns preserved bytes only through the owning corpus and source identity.
func (repository *Repository) GetPDFOrigin(
	ctx context.Context,
	corpusID uuid.UUID,
	sourceID uuid.UUID,
) (PDFOriginRecord, error) {
	var origin PDFOriginRecord
	err := repository.database.QueryRow(ctx, `
		SELECT p.delivery_filename, p.detected_media_type, p.content
		FROM pdf_origins p
		JOIN sources s ON s.corpus_id = p.corpus_id AND s.id = p.source_id
		WHERE p.corpus_id = $1 AND p.source_id = $2 AND s.kind = 'pdf'`, corpusID, sourceID,
	).Scan(&origin.DeliveryFilename, &origin.MediaType, &origin.Content)
	if errors.Is(err, pgx.ErrNoRows) {
		return PDFOriginRecord{}, ErrNotFound
	}
	if err != nil {
		return PDFOriginRecord{}, fmt.Errorf("get PDF origin: %w", err)
	}
	return origin, nil
}

// GetURLOrigin returns the latest captured destination or submitted URL within corpus scope.
func (repository *Repository) GetURLOrigin(
	ctx context.Context,
	corpusID uuid.UUID,
	sourceID uuid.UUID,
) (URLOriginRecord, error) {
	var origin URLOriginRecord
	err := repository.database.QueryRow(ctx, `
		SELECT COALESCE((
			SELECT r.final_url
			FROM source_revisions r
			WHERE r.corpus_id = u.corpus_id AND r.source_id = u.source_id
			  AND r.final_url IS NOT NULL
			ORDER BY r.captured_at DESC, r.id DESC
			LIMIT 1
		), u.submitted_url)
		FROM url_origins u
		WHERE u.corpus_id = $1 AND u.source_id = $2`, corpusID, sourceID,
	).Scan(&origin.URL)
	if errors.Is(err, pgx.ErrNoRows) {
		return URLOriginRecord{}, ErrNotFound
	}
	if err != nil {
		return URLOriginRecord{}, fmt.Errorf("get URL origin: %w", err)
	}
	return origin, nil
}

// LifecycleCommand describes one optimistic source lifecycle transition.
type LifecycleCommand struct {
	CorpusID        uuid.UUID
	SourceID        uuid.UUID
	ExpectedVersion int
	RequiredStatus  domain.Status
	Reason          string
	WorkID          uuid.UUID
	RequestedAt     time.Time
}

// QueueLifecycle atomically transitions a source to pending and creates explicit work.
func (repository *Repository) QueueLifecycle(
	ctx context.Context,
	command LifecycleCommand,
) (Record, error) {
	transaction, err := repository.database.Begin(ctx)
	if err != nil {
		return Record{}, fmt.Errorf("begin source lifecycle command: %w", err)
	}
	defer func() { _ = transaction.Rollback(ctx) }()
	result, err := transaction.Exec(ctx, `
		UPDATE sources
		SET processing_status = 'pending', latest_failure_category = NULL,
		    version = version + 1, updated_at = $5
		WHERE corpus_id = $1 AND id = $2 AND version = $3 AND processing_status = $4`,
		command.CorpusID, command.SourceID, command.ExpectedVersion,
		command.RequiredStatus, command.RequestedAt.UTC(),
	)
	if err != nil {
		return Record{}, fmt.Errorf("transition source lifecycle: %w", err)
	}
	if result.RowsAffected() != 1 {
		return Record{}, repository.classifyLifecycleMiss(
			ctx, transaction, command.CorpusID, command.SourceID, command.ExpectedVersion,
		)
	}
	if _, err := transaction.Exec(ctx, `
		INSERT INTO ingestion_work (
			id, source_id, corpus_id, reason, requested_at, created_at, updated_at
		) VALUES ($1, $2, $3, $4, $5, $5, $5)`,
		command.WorkID, command.SourceID, command.CorpusID,
		command.Reason, command.RequestedAt.UTC(),
	); err != nil {
		return Record{}, classifyCreateError("insert lifecycle ingestion work", err)
	}
	record, err := scanRecord(transaction.QueryRow(ctx, `
		`+sourceProjectionSQL+`
		WHERE s.corpus_id = $1 AND s.id = $2`, command.CorpusID, command.SourceID))
	if err != nil {
		return Record{}, fmt.Errorf("read transitioned source: %w", err)
	}
	record.Attempts, err = listAttempts(
		ctx, transaction, command.CorpusID, command.SourceID,
	)
	if err != nil {
		return Record{}, err
	}
	setLatestAttempt(&record)
	if err := transaction.Commit(ctx); err != nil {
		return Record{}, fmt.Errorf("commit source lifecycle command: %w", err)
	}
	return record, nil
}

func (repository *Repository) classifyLifecycleMiss(
	ctx context.Context,
	transaction pgx.Tx,
	corpusID uuid.UUID,
	sourceID uuid.UUID,
	expectedVersion int,
) error {
	var version int
	err := transaction.QueryRow(ctx,
		"SELECT version FROM sources WHERE corpus_id = $1 AND id = $2", corpusID, sourceID,
	).Scan(&version)
	if errors.Is(err, pgx.ErrNoRows) {
		return ErrNotFound
	}
	if err != nil {
		return fmt.Errorf("classify source lifecycle miss: %w", err)
	}
	if version != expectedVersion {
		return ErrStaleState
	}
	return domain.ErrInvalidTransition
}

func classifyCreateError(operation string, err error) error {
	var postgresError *pgconn.PgError
	if errors.As(err, &postgresError) && postgresError.Code == "23505" {
		return domain.ErrDuplicateSource
	}
	return fmt.Errorf("%s: %w", operation, err)
}

func lockAvailableCorpus(ctx context.Context, transaction pgx.Tx, corpusID uuid.UUID) error {
	var status string
	if err := transaction.QueryRow(
		ctx, "SELECT status FROM corpora WHERE id = $1 FOR UPDATE", corpusID,
	).Scan(&status); errors.Is(err, pgx.ErrNoRows) || status != "enabled" {
		return domain.ErrCorpusUnavailable
	} else if err != nil {
		return fmt.Errorf("lock source corpus: %w", err)
	}
	return nil
}

func ensureSourceCapacity(ctx context.Context, transaction pgx.Tx, corpusID uuid.UUID) error {
	var count int
	if err := transaction.QueryRow(
		ctx, "SELECT count(*) FROM sources WHERE corpus_id = $1", corpusID,
	).Scan(&count); err != nil {
		return fmt.Errorf("count corpus sources: %w", err)
	}
	return domain.EnsureCapacity(count)
}

func insertSource(ctx context.Context, transaction pgx.Tx, source domain.Source) error {
	if _, err := transaction.Exec(ctx, `
		INSERT INTO sources (
			id, corpus_id, title, kind, processing_status, version, created_at, updated_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8)`,
		source.ID, source.CorpusID, source.Title, source.Kind, source.Status,
		source.Version, source.CreatedAt, source.UpdatedAt,
	); err != nil {
		return classifyCreateError("insert source", err)
	}
	return nil
}

func insertInitialWork(
	ctx context.Context,
	transaction pgx.Tx,
	source domain.Source,
	workID uuid.UUID,
) error {
	if _, err := transaction.Exec(ctx, `
		INSERT INTO ingestion_work (
			id, source_id, corpus_id, reason, requested_at, created_at, updated_at
		) VALUES ($1, $2, $3, 'initial', $4, $4, $4)`,
		workID, source.ID, source.CorpusID, source.CreatedAt,
	); err != nil {
		return classifyCreateError("insert initial ingestion work", err)
	}
	return nil
}

func recordFromSource(source domain.Source) Record {
	return Record{
		ID: source.ID, CorpusID: source.CorpusID, Title: source.Title,
		Kind: source.Kind, ProcessingStatus: source.Status, Version: source.Version,
		CreatedAt: source.CreatedAt, UpdatedAt: source.UpdatedAt,
	}
}

// ListByCorpus returns deterministic source projections owned by one corpus.
func (repository *Repository) ListByCorpus(ctx context.Context, corpusID uuid.UUID) ([]Record, error) {
	rows, err := repository.database.Query(ctx, `
		`+sourceProjectionSQL+`
		WHERE s.corpus_id = $1
		ORDER BY s.created_at, s.id`, corpusID)
	if err != nil {
		return nil, fmt.Errorf("list corpus sources: %w", err)
	}
	defer rows.Close()
	records, err := pgx.CollectRows(rows, func(row pgx.CollectableRow) (Record, error) {
		return scanRecord(row)
	})
	if err != nil {
		return nil, fmt.Errorf("scan corpus sources: %w", err)
	}
	for index := range records {
		records[index].Attempts, err = listAttempts(
			ctx, repository.database, corpusID, records[index].ID,
		)
		if err != nil {
			return nil, err
		}
		setLatestAttempt(&records[index])
	}
	return records, nil
}

// Get returns a source only when both corpus and source identity match.
func (repository *Repository) Get(
	ctx context.Context,
	corpusID uuid.UUID,
	sourceID uuid.UUID,
) (Record, error) {
	record, err := scanRecord(repository.database.QueryRow(ctx, `
		`+sourceProjectionSQL+`
		WHERE s.corpus_id = $1 AND s.id = $2`, corpusID, sourceID))
	if errors.Is(err, pgx.ErrNoRows) {
		return Record{}, ErrNotFound
	}
	if err != nil {
		return Record{}, fmt.Errorf("get corpus source: %w", err)
	}
	record.Attempts, err = listAttempts(ctx, repository.database, corpusID, sourceID)
	if err != nil {
		return Record{}, err
	}
	setLatestAttempt(&record)
	return record, nil
}

func listAttempts(
	ctx context.Context, database queryer, corpusID uuid.UUID, sourceID uuid.UUID,
) ([]Attempt, error) {
	rows, err := database.Query(ctx, `
		SELECT attempt_number, pipeline_version, status, started_at, finished_at,
		       failure_category, acquired_byte_count, normalized_character_count,
		       unit_count, duration_milliseconds
		FROM processing_attempts
		WHERE corpus_id = $1 AND source_id = $2
		ORDER BY started_at DESC, id DESC`, corpusID, sourceID)
	if err != nil {
		return nil, fmt.Errorf("list source processing attempts: %w", err)
	}
	defer rows.Close()
	attempts, err := pgx.CollectRows(rows, func(row pgx.CollectableRow) (Attempt, error) {
		var attempt Attempt
		err := row.Scan(
			&attempt.Number, &attempt.PipelineVersion, &attempt.Status, &attempt.StartedAt,
			&attempt.FinishedAt, &attempt.FailureCategory, &attempt.AcquiredByteCount,
			&attempt.NormalizedCharacterCount, &attempt.UnitCount, &attempt.DurationMilliseconds,
		)
		return attempt, err
	})
	if err != nil {
		return nil, fmt.Errorf("scan source processing attempts: %w", err)
	}
	return attempts, nil
}

func setLatestAttempt(record *Record) {
	if len(record.Attempts) > 0 {
		record.LatestAttempt = &record.Attempts[0]
	}
}

func scanRecord(row rowScanner) (Record, error) {
	var record Record
	var latestDocument pgtype.UUID
	var attemptNumber *int
	var attemptPipeline, attemptStatus *string
	var attemptStarted *time.Time
	var attemptFinished *time.Time
	var attemptFailure *string
	var acquiredBytes, normalizedCharacters, durationMilliseconds *int64
	var unitCount *int
	err := row.Scan(
		&record.ID,
		&record.CorpusID,
		&record.Title,
		&record.Kind,
		&record.ProcessingStatus,
		&record.FailureCategory,
		&latestDocument,
		&record.Version,
		&record.CreatedAt,
		&record.UpdatedAt,
		&record.Origin.SubmittedURL,
		&record.Origin.NormalizedURL,
		&record.Origin.OriginalFilename,
		&record.Origin.MediaType,
		&record.Origin.ByteSize,
		&record.Origin.SHA256,
		&record.Origin.FinalURL,
		&record.Origin.CapturedAt,
		&record.Origin.ExtractedContentSHA256,
		&attemptNumber,
		&attemptPipeline,
		&attemptStatus,
		&attemptStarted,
		&attemptFinished,
		&attemptFailure,
		&acquiredBytes,
		&normalizedCharacters,
		&unitCount,
		&durationMilliseconds,
	)
	if err != nil {
		return Record{}, err
	}
	if latestDocument.Valid {
		id := uuid.UUID(latestDocument.Bytes)
		record.LatestReadyDocumentID = &id
	}
	if attemptNumber != nil && attemptPipeline != nil && attemptStatus != nil && attemptStarted != nil {
		record.LatestAttempt = &Attempt{
			Number: *attemptNumber, PipelineVersion: *attemptPipeline, Status: *attemptStatus,
			StartedAt: *attemptStarted, FinishedAt: attemptFinished, FailureCategory: attemptFailure,
			AcquiredByteCount: acquiredBytes, NormalizedCharacterCount: normalizedCharacters,
			UnitCount: unitCount, DurationMilliseconds: durationMilliseconds,
		}
	}
	return record, nil
}

const sourceProjectionSQL = `
	SELECT s.id, s.corpus_id, s.title, s.kind, s.processing_status,
	       s.latest_failure_category, s.latest_ready_document_id,
	       s.version, s.created_at, s.updated_at,
	       u.submitted_url, u.normalized_url, p.original_filename,
	       COALESCE(r.media_type, p.detected_media_type),
	       COALESCE(r.byte_size, p.byte_size),
	       COALESCE(r.content_sha256, p.sha256),
	       r.final_url, r.captured_at, r.extracted_content_sha256,
	       a.attempt_number, a.pipeline_version, a.status, a.started_at, a.finished_at,
	       a.failure_category, a.acquired_byte_count, a.normalized_character_count,
	       a.unit_count, a.duration_milliseconds
	FROM sources s
	LEFT JOIN url_origins u ON u.corpus_id = s.corpus_id AND u.source_id = s.id
	LEFT JOIN pdf_origins p ON p.corpus_id = s.corpus_id AND p.source_id = s.id
	LEFT JOIN LATERAL (
		SELECT content_sha256, captured_at, media_type, byte_size, final_url,
		       extracted_content_sha256
		FROM source_revisions
		WHERE corpus_id = s.corpus_id AND source_id = s.id
		ORDER BY captured_at DESC, id DESC LIMIT 1
	) r ON true
	LEFT JOIN LATERAL (
		SELECT attempt_number, pipeline_version, status, started_at, finished_at,
		       failure_category, acquired_byte_count, normalized_character_count,
		       unit_count, duration_milliseconds
		FROM processing_attempts
		WHERE corpus_id = s.corpus_id AND source_id = s.id
		ORDER BY started_at DESC, id DESC LIMIT 1
	) a ON true`
