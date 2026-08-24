package application

import (
	"context"
	"testing"
	"time"

	"github.com/dharlanoliveira/norvii/apps/api/internal/snapshot/domain"
	"github.com/google/uuid"
)

func TestPublishBuildsAnExplicitPublicationCommand(t *testing.T) {
	store := &fakeStore{}
	snapshotID := uuid.New()
	now := time.Date(2026, time.August, 24, 12, 0, 0, 0, time.UTC)
	service := NewService(store, func() uuid.UUID { return snapshotID }, func() time.Time { return now })
	corpusID, sourceID, documentID := uuid.New(), uuid.New(), uuid.New()

	_, err := service.Publish(context.Background(), corpusID, sourceID, documentID, 3)

	if err != nil {
		t.Fatalf("Publish() error = %v", err)
	}
	if store.command.SnapshotID != snapshotID || store.command.ExpectedReleaseVersion != 3 || store.command.PublishedAt != now {
		t.Fatalf("publication command = %+v, want generated snapshot and version", store.command)
	}
}

func TestInitializeDelegatesTheGeneratedSnapshotIdentity(t *testing.T) {
	store := &fakeStore{}
	snapshotID := uuid.New()
	service := NewService(store, func() uuid.UUID { return snapshotID }, time.Now)
	corpusID := uuid.New()

	_, err := service.Initialize(context.Background(), corpusID)

	if err != nil {
		t.Fatalf("Initialize() error = %v", err)
	}
	if store.initializedSnapshotID != snapshotID || store.initializedCorpusID != corpusID {
		t.Fatalf("initialized corpus/snapshot = %s/%s, want %s/%s", store.initializedCorpusID, store.initializedSnapshotID, corpusID, snapshotID)
	}
}

type fakeStore struct {
	command               domain.PublishCommand
	initializedCorpusID   uuid.UUID
	initializedSnapshotID uuid.UUID
}

func (store *fakeStore) Active(context.Context, uuid.UUID) (domain.Release, error) {
	return domain.Release{}, nil
}

func (store *fakeStore) Get(context.Context, uuid.UUID, uuid.UUID) (domain.Snapshot, error) {
	return domain.Snapshot{}, nil
}

func (store *fakeStore) Initialize(
	_ context.Context,
	corpusID, snapshotID uuid.UUID,
	_ string,
	_ time.Time,
) (domain.Publication, error) {
	store.initializedCorpusID = corpusID
	store.initializedSnapshotID = snapshotID
	return domain.Publication{}, nil
}

func (store *fakeStore) List(context.Context, uuid.UUID) ([]domain.Snapshot, error) {
	return nil, nil
}

func (store *fakeStore) Publish(
	_ context.Context,
	command domain.PublishCommand,
) (domain.Publication, error) {
	store.command = command
	return domain.Publication{}, nil
}
