// Package application resolves the active snapshot before online chat orchestration.
package application

import (
	"context"
	"errors"
	"fmt"

	chatdomain "github.com/dharlanoliveira/norvii/apps/api/internal/chat/domain"
	graphdomain "github.com/dharlanoliveira/norvii/apps/api/internal/graphrelease/domain"
	snapshotdomain "github.com/dharlanoliveira/norvii/apps/api/internal/snapshot/domain"
	"github.com/google/uuid"
)

type releaseReader interface {
	Active(context.Context, uuid.UUID) (snapshotdomain.Release, error)
}

type asker interface {
	Ask(context.Context, chatdomain.Request, func(string)) (chatdomain.Result, error)
}

type graphReleaseReader interface {
	Get(context.Context, uuid.UUID, uuid.UUID) (graphdomain.Release, error)
}

// Service enforces the active immutable snapshot at the API-to-agent boundary.
type Service struct {
	releases      releaseReader
	graphReleases graphReleaseReader
	agent         asker
}

// NewService constructs chat orchestration from a release reader and agent client.
func NewService(releases releaseReader, graphReleases graphReleaseReader, agent asker) *Service {
	return &Service{releases: releases, graphReleases: graphReleases, agent: agent}
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
	request.SnapshotID = release.SnapshotID
	if request.Strategy == "graph" || request.Strategy == "hybrid" {
		graphRelease, graphErr := service.graphReleases.Get(ctx, request.CorpusID, release.SnapshotID)
		if graphErr != nil {
			if errors.Is(graphErr, graphdomain.ErrNotFound) {
				return chatdomain.Result{}, chatdomain.ErrGraphUnavailable
			}
			return chatdomain.Result{}, fmt.Errorf("resolve active graph release: %w", graphErr)
		}
		if !graphRelease.Ready() {
			return chatdomain.Result{}, chatdomain.ErrGraphUnavailable
		}
	}
	return service.agent.Ask(ctx, request, emit)
}
