// Package domain defines immutable corpus snapshot values and validation rules.
package domain

import (
	"errors"
	"strings"
	"time"

	"github.com/google/uuid"
)

var (
	// ErrInvalidInput identifies malformed snapshot commands.
	ErrInvalidInput = errors.New("snapshot input is invalid")
	// ErrCandidateNotReady identifies a candidate without complete retrieval artifacts.
	ErrCandidateNotReady = errors.New("snapshot candidate is not ready")
	// ErrStaleRelease identifies an optimistic release-version conflict.
	ErrStaleRelease = errors.New("snapshot release is stale")
	// ErrNotFound intentionally covers snapshots outside the requested corpus.
	ErrNotFound = errors.New("snapshot not found")
)

// Member identifies one immutable source revision selected by a snapshot.
type Member struct {
	SourceID         uuid.UUID
	SourceRevisionID uuid.UUID
	DocumentID       uuid.UUID
	OfficialOrigin   string
	CapturedAt       time.Time
	ContentSHA256    string
}

// Snapshot is one immutable evidence manifest.
type Snapshot struct {
	ID             uuid.UUID
	CorpusID       uuid.UUID
	ManifestSHA256 string
	CreatedBy      string
	CreatedAt      time.Time
	Members        []Member
}

// Release selects one active immutable snapshot for a corpus.
type Release struct {
	CorpusID    uuid.UUID
	SnapshotID  uuid.UUID
	Version     int
	ActivatedAt time.Time
}

// Publication reports the active release after an explicit publish operation.
type Publication struct {
	Snapshot Snapshot
	Release  Release
	Created  bool
}

// PublishCommand explicitly promotes one ready candidate into a new release.
type PublishCommand struct {
	CorpusID               uuid.UUID
	SourceID               uuid.UUID
	DocumentID             uuid.UUID
	ExpectedReleaseVersion int
	SnapshotID             uuid.UUID
	Actor                  string
	PublishedAt            time.Time
}

// Validate normalizes the application boundary before persistence.
func (command PublishCommand) Validate() error {
	if command.CorpusID == uuid.Nil || command.SourceID == uuid.Nil || command.DocumentID == uuid.Nil || command.SnapshotID == uuid.Nil {
		return ErrInvalidInput
	}
	if command.ExpectedReleaseVersion < 1 || strings.TrimSpace(command.Actor) == "" {
		return ErrInvalidInput
	}
	return nil
}
