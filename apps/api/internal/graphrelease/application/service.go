// Package application owns public graph-release inspection use cases.
package application

import (
	"context"

	"github.com/dharlanoliveira/norvii/apps/api/internal/graphrelease/domain"
	"github.com/dharlanoliveira/norvii/apps/api/internal/graphrelease/postgres"
	"github.com/google/uuid"
)

type releaseGetter interface {
	Get(context.Context, uuid.UUID, uuid.UUID) (domain.Release, error)
}

// Service exposes one corpus- and snapshot-scoped graph-release lookup.
type Service struct{ releaseGetter releaseGetter }

// NewService constructs a graph-release inspection service.
func NewService(releaseGetter releaseGetter) *Service { return &Service{releaseGetter: releaseGetter} }

// Get returns an inspectable release record.
func (service *Service) Get(ctx context.Context, corpusID, snapshotID uuid.UUID) (domain.Release, error) {
	return service.releaseGetter.Get(ctx, corpusID, snapshotID)
}

// RepositoryStore verifies the concrete persistence adapter satisfies the public port.
var _ releaseGetter = (*postgres.Repository)(nil)
