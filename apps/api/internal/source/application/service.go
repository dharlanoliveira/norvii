// Package application coordinates source registration and lifecycle use cases.
package application

import (
	"context"
	"strings"
	"time"

	"github.com/dharlanoliveira/norvii/apps/api/internal/source/domain"
	sourcepostgres "github.com/dharlanoliveira/norvii/apps/api/internal/source/postgres"
	"github.com/google/uuid"
)

type urlRepository interface {
	CreateURL(context.Context, domain.Source, string, string, uuid.UUID) (sourcepostgres.Record, error)
	CreatePDF(context.Context, domain.Source, domain.PDFOrigin, uuid.UUID) (sourcepostgres.Record, error)
	QueueLifecycle(
		context.Context, uuid.UUID, uuid.UUID, int, domain.Status, string, uuid.UUID, time.Time,
	) (sourcepostgres.Record, error)
}

// Retry queues a failed source under the caller's expected version.
func (service *Service) Retry(
	ctx context.Context, corpusID uuid.UUID, sourceID uuid.UUID, version int,
) (sourcepostgres.Record, error) {
	return service.repository.QueueLifecycle(
		ctx, corpusID, sourceID, version, domain.StatusFailed, "retry", service.newID(), service.now(),
	)
}

// Reprocess queues a ready source while preserving its latest ready document.
func (service *Service) Reprocess(
	ctx context.Context, corpusID uuid.UUID, sourceID uuid.UUID, version int,
) (sourcepostgres.Record, error) {
	return service.repository.QueueLifecycle(
		ctx, corpusID, sourceID, version, domain.StatusReady, "reprocess", service.newID(), service.now(),
	)
}

// CreatePDF registers one preserved PDF and its initial ingestion work atomically.
func (service *Service) CreatePDF(
	ctx context.Context,
	corpusID uuid.UUID,
	title string,
	filename string,
	declaredMediaType string,
	content []byte,
) (sourcepostgres.Record, error) {
	origin, err := domain.NewPDFOrigin(filename, declaredMediaType, content)
	if err != nil {
		return sourcepostgres.Record{}, err
	}
	source, err := domain.NewSource(service.newID(), corpusID, title, domain.KindPDF, service.now())
	if err != nil {
		return sourcepostgres.Record{}, err
	}
	return service.repository.CreatePDF(ctx, source, origin, service.newID())
}

// Service owns source validation, identities, and work creation orchestration.
type Service struct {
	repository urlRepository
	newID      func() uuid.UUID
	now        func() time.Time
}

// NewService constructs source commands around caller-owned dependencies.
func NewService(repository urlRepository, newID func() uuid.UUID, now func() time.Time) *Service {
	return &Service{repository: repository, newID: newID, now: now}
}

// CreateURL registers one HTTPS origin and its initial ingestion work atomically.
func (service *Service) CreateURL(
	ctx context.Context,
	corpusID uuid.UUID,
	title string,
	submittedURL string,
) (sourcepostgres.Record, error) {
	submittedURL = strings.TrimSpace(submittedURL)
	normalizedURL, err := domain.NormalizeURL(submittedURL)
	if err != nil {
		return sourcepostgres.Record{}, err
	}
	source, err := domain.NewSource(service.newID(), corpusID, title, domain.KindURL, service.now())
	if err != nil {
		return sourcepostgres.Record{}, err
	}
	return service.repository.CreateURL(ctx, source, submittedURL, normalizedURL, service.newID())
}
