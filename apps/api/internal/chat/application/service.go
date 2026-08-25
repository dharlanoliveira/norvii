// Package application resolves the active snapshot before online chat orchestration.
package application

import (
	"context"
	"errors"
	"fmt"

	chatdomain "github.com/dharlanoliveira/norvii/apps/api/internal/chat/domain"
	snapshotdomain "github.com/dharlanoliveira/norvii/apps/api/internal/snapshot/domain"
	"github.com/google/uuid"
)

type releaseReader interface {
	Active(context.Context, uuid.UUID) (snapshotdomain.Release, error)
}

type asker interface {
	Ask(context.Context, chatdomain.Request, func(string)) (chatdomain.Result, error)
}

// Service enforces the active immutable snapshot at the API-to-agent boundary.
type Service struct {
	releases releaseReader
	agent    asker
}

// NewService constructs chat orchestration from a release reader and agent client.
func NewService(releases releaseReader, agent asker) *Service {
	return &Service{releases: releases, agent: agent}
}

// Ask resolves a corpus release before forwarding the grounded request.
func (service *Service) Ask(
	ctx context.Context, request chatdomain.Request, emit func(string),
) (chatdomain.Result, error) {
	release, err := service.releases.Active(ctx, request.CorpusID)
	if errors.Is(err, snapshotdomain.ErrNotFound) {
		return chatdomain.Result{}, chatdomain.ErrSnapshotUnavailable
	}
	if err != nil {
		return chatdomain.Result{}, fmt.Errorf("resolve active corpus snapshot: %w", err)
	}
	if release.CorpusID != request.CorpusID || release.SnapshotID == uuid.Nil {
		return chatdomain.Result{}, chatdomain.ErrSnapshotUnavailable
	}
	request.SnapshotID = release.SnapshotID
	return service.agent.Ask(ctx, request, emit)
}
