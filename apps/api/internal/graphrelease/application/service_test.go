package application

import (
	"context"
	"testing"

	"github.com/dharlanoliveira/norvii/apps/api/internal/graphrelease/domain"
	"github.com/google/uuid"
)

func TestGetDelegatesToTheCorpusSnapshotReleaseStore(t *testing.T) {
	corpusID := uuid.New()
	snapshotID := uuid.New()
	release := domain.Release{ID: uuid.New(), CorpusID: corpusID, SnapshotID: snapshotID}
	store := &recordingReleaseStore{release: release}
	service := NewService(store)

	actual, err := service.Get(context.Background(), corpusID, snapshotID)

	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}
	if actual != release {
		t.Fatalf("Get() release = %#v, want %#v", actual, release)
	}
	if store.corpusID != corpusID || store.snapshotID != snapshotID {
		t.Fatalf("store lookup = %s/%s, want %s/%s", store.corpusID, store.snapshotID, corpusID, snapshotID)
	}
}

type recordingReleaseStore struct {
	release    domain.Release
	corpusID   uuid.UUID
	snapshotID uuid.UUID
}

func (store *recordingReleaseStore) Get(_ context.Context, corpusID, snapshotID uuid.UUID) (domain.Release, error) {
	store.corpusID = corpusID
	store.snapshotID = snapshotID
	return store.release, nil
}
