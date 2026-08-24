// Package application coordinates explicit corpus snapshot publication commands.
package application

import (
	"context"
	"time"

	"github.com/dharlanoliveira/norvii/apps/api/internal/snapshot/domain"
	"github.com/dharlanoliveira/norvii/apps/api/internal/snapshot/postgres"
	"github.com/google/uuid"
)

type store interface {
	Active(context.Context, uuid.UUID) (domain.Release, error)
	Get(context.Context, uuid.UUID, uuid.UUID) (domain.Snapshot, error)
	Initialize(context.Context, uuid.UUID, uuid.UUID, string, time.Time) (domain.Publication, error)
	List(context.Context, uuid.UUID) ([]domain.Snapshot, error)
	Publish(context.Context, domain.PublishCommand) (domain.Publication, error)
}

// Service owns application-level IDs and timestamps for snapshot operations.
type Service struct {
	store store
	newID func() uuid.UUID
	now   func() time.Time
}

// NewService constructs snapshot use cases around explicit persistence and clock dependencies.
func NewService(store store, newID func() uuid.UUID, now func() time.Time) *Service {
	return &Service{store: store, newID: newID, now: now}
}

// Publish explicitly promotes one validated candidate document.
func (service *Service) Publish(
	ctx context.Context, corpusID, sourceID, documentID uuid.UUID, expectedReleaseVersion int,
) (domain.Publication, error) {
	return service.store.Publish(ctx, domain.PublishCommand{
		CorpusID: corpusID, SourceID: sourceID, DocumentID: documentID,
		ExpectedReleaseVersion: expectedReleaseVersion, SnapshotID: service.newID(),
		Actor: "local-maintainer", PublishedAt: service.now(),
	})
}

// Initialize creates the first active release for an already ingested corpus.
func (service *Service) Initialize(ctx context.Context, corpusID uuid.UUID) (domain.Publication, error) {
	return service.store.Initialize(ctx, corpusID, service.newID(), "local-maintainer", service.now())
}

// List returns immutable manifests for evaluator inspection.
func (service *Service) List(ctx context.Context, corpusID uuid.UUID) ([]domain.Snapshot, error) {
	return service.store.List(ctx, corpusID)
}

// Get returns one immutable manifest through its corpus boundary.
func (service *Service) Get(ctx context.Context, corpusID, snapshotID uuid.UUID) (domain.Snapshot, error) {
	return service.store.Get(ctx, corpusID, snapshotID)
}

// RepositoryStore documents the concrete store accepted by the API composition root.
var _ store = (*postgres.Repository)(nil)
