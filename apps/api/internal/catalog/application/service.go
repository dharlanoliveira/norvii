// Package application coordinates validated corpus management use cases.
package application

import (
	"context"
	"time"

	"github.com/dharlanoliveira/norvii/apps/api/internal/catalog/domain"
	catalogpostgres "github.com/dharlanoliveira/norvii/apps/api/internal/catalog/postgres"
	"github.com/google/uuid"
)

type mutations interface {
	Create(context.Context, domain.Corpus) (catalogpostgres.Summary, error)
	Update(context.Context, uuid.UUID, domain.Draft, int, time.Time) (catalogpostgres.Summary, error)
	SetStatus(context.Context, uuid.UUID, domain.Status, int, time.Time) (catalogpostgres.Summary, error)
}

// Service owns corpus validation, identity, time, and lifecycle orchestration.
type Service struct {
	repository mutations
	newID      func() uuid.UUID
	now        func() time.Time
}

// NewService constructs corpus management around injected deterministic dependencies.
func NewService(
	repository mutations,
	newID func() uuid.UUID,
	now func() time.Time,
) *Service {
	return &Service{repository: repository, newID: newID, now: now}
}

// Create validates and persists one new enabled corpus.
func (service *Service) Create(ctx context.Context, draft domain.Draft) (catalogpostgres.Summary, error) {
	corpus, err := domain.NewCorpus(service.newID(), draft, service.now())
	if err != nil {
		return catalogpostgres.Summary{}, err
	}
	return service.repository.Create(ctx, corpus)
}

// Update validates and atomically replaces mutable corpus metadata.
func (service *Service) Update(
	ctx context.Context,
	id uuid.UUID,
	draft domain.Draft,
	expectedVersion int,
) (catalogpostgres.Summary, error) {
	normalized, err := domain.NormalizeDraft(draft)
	if err != nil {
		return catalogpostgres.Summary{}, err
	}
	return service.repository.Update(ctx, id, normalized, expectedVersion, service.now())
}

// Disable makes one corpus unavailable without deleting owned data.
func (service *Service) Disable(
	ctx context.Context,
	id uuid.UUID,
	expectedVersion int,
) (catalogpostgres.Summary, error) {
	return service.repository.SetStatus(
		ctx, id, domain.StatusDisabled, expectedVersion, service.now(),
	)
}

// Enable restores one disabled corpus for researcher selection.
func (service *Service) Enable(
	ctx context.Context,
	id uuid.UUID,
	expectedVersion int,
) (catalogpostgres.Summary, error) {
	return service.repository.SetStatus(
		ctx, id, domain.StatusEnabled, expectedVersion, service.now(),
	)
}
