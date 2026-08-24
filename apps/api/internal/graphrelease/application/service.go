// Package application owns public graph-release inspection use cases.
package application

import (
	"context"

	"github.com/dharlanoliveira/norvii/apps/api/internal/graphrelease/domain"
	"github.com/dharlanoliveira/norvii/apps/api/internal/graphrelease/postgres"
	"github.com/google/uuid"
)

type store interface {
	Get(context.Context, uuid.UUID, uuid.UUID) (domain.Release, error)
}

// Service exposes one corpus- and snapshot-scoped graph-release lookup.
type Service struct{ store store }

// NewService constructs a graph-release inspection service.
func NewService(store store) *Service { return &Service{store: store} }

// Get returns an inspectable release record.
func (service *Service) Get(ctx context.Context, corpusID, snapshotID uuid.UUID) (domain.Release, error) {
	return service.store.Get(ctx, corpusID, snapshotID)
}

// RepositoryStore verifies the concrete persistence adapter satisfies the public port.
var _ store = (*postgres.Repository)(nil)
