package domain

import (
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestReleaseValidateAcceptsReadyRelease(t *testing.T) {
	completedAt := time.Now().UTC()
	release := Release{
		ID: uuid.New(), CorpusID: uuid.New(), SnapshotID: uuid.New(),
		ManifestSHA256: "manifest", BuildVersion: "legal-graph-v1", Status: StatusReady,
		CreatedAt: completedAt, CompletedAt: &completedAt,
	}
	if err := release.Validate(); err != nil {
		t.Fatalf("Validate() error = %v", err)
	}
	if !release.Ready() {
		t.Fatal("Ready() = false, want true")
	}
}

func TestReleaseValidateRejectsIncompleteFailedRelease(t *testing.T) {
	release := Release{
		ID: uuid.New(), CorpusID: uuid.New(), SnapshotID: uuid.New(),
		ManifestSHA256: "manifest", BuildVersion: "legal-graph-v1", Status: StatusFailed,
	}
	if err := release.Validate(); err != ErrInvalidInput {
		t.Fatalf("Validate() error = %v, want %v", err, ErrInvalidInput)
	}
}
