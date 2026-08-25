// Package domain defines immutable graph-release values and validation rules.
package domain

import (
	"errors"
	"strings"
	"time"

	"github.com/google/uuid"
)

var (
	// ErrInvalidInput identifies a malformed graph-release command.
	ErrInvalidInput = errors.New("graph release input is invalid")
	// ErrNotFound deliberately covers releases outside the requested corpus and snapshot.
	ErrNotFound = errors.New("graph release not found")
	// ErrUnavailable identifies a graph release that cannot safely support retrieval.
	ErrUnavailable = errors.New("graph release is unavailable")
)

// Status describes the readiness of one immutable graph projection.
type Status string

const (
	StatusBuilding Status = "building"
	StatusReady    Status = "ready"
	StatusFailed   Status = "failed"
)

// Release records a rebuildable graph projection for one corpus snapshot.
type Release struct {
	ID                uuid.UUID
	CorpusID          uuid.UUID
	SnapshotID        uuid.UUID
	ManifestSHA256    string
	BuildVersion      string
	Status            Status
	FailureCategory   string
	EntityCount       int
	RelationshipCount int
	CreatedAt         time.Time
	CompletedAt       *time.Time
}

// Ready reports whether this projection is safe for graph-backed retrieval.
func (release Release) Ready() bool {
	return release.Status == StatusReady && release.ID != uuid.Nil
}

// Validate confirms release identity and status invariants at persistence boundaries.
func (release Release) Validate() error {
	if release.ID == uuid.Nil || release.CorpusID == uuid.Nil || release.SnapshotID == uuid.Nil ||
		strings.TrimSpace(release.ManifestSHA256) == "" || strings.TrimSpace(release.BuildVersion) == "" {
		return ErrInvalidInput
	}
	if release.EntityCount < 0 || release.RelationshipCount < 0 {
		return ErrInvalidInput
	}
	switch release.Status {
	case StatusBuilding:
		if release.CompletedAt != nil || release.FailureCategory != "" {
			return ErrInvalidInput
		}
	case StatusReady:
		if release.CompletedAt == nil || release.FailureCategory != "" {
			return ErrInvalidInput
		}
	case StatusFailed:
		if release.CompletedAt == nil || strings.TrimSpace(release.FailureCategory) == "" {
			return ErrInvalidInput
		}
	default:
		return ErrInvalidInput
	}
	return nil
}
