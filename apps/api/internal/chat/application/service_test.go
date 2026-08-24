package application

import (
	"context"
	"errors"
	"testing"
	"time"

	chatdomain "github.com/dharlanoliveira/norvii/apps/api/internal/chat/domain"
	graphdomain "github.com/dharlanoliveira/norvii/apps/api/internal/graphrelease/domain"
	snapshotdomain "github.com/dharlanoliveira/norvii/apps/api/internal/snapshot/domain"
	"github.com/google/uuid"
)

func TestAskForwardsOnlyTheActiveSnapshot(t *testing.T) {
	corpusID := uuid.New()
	snapshotID := uuid.New()
	agent := &fakeAgent{}
	service := NewService(
		fakeReleases{release: snapshotdomain.Release{CorpusID: corpusID, SnapshotID: snapshotID, Version: 2, ActivatedAt: time.Now()}},
		fakeGraphReleases{},
		agent,
	)

	_, err := service.Ask(context.Background(), chatdomain.Request{CorpusID: corpusID, Question: "What applies?"}, func(string) {})

	if err != nil {
		t.Fatalf("Ask() error = %v", err)
	}
	if agent.request.SnapshotID != snapshotID {
		t.Fatalf("agent snapshot = %s, want %s", agent.request.SnapshotID, snapshotID)
	}
}

func TestAskRejectsCorpusWithoutAnActiveSnapshot(t *testing.T) {
	service := NewService(fakeReleases{err: snapshotdomain.ErrNotFound}, fakeGraphReleases{}, &fakeAgent{})

	_, err := service.Ask(context.Background(), chatdomain.Request{CorpusID: uuid.New(), Question: "What applies?"}, func(string) {})

	if !errors.Is(err, chatdomain.ErrSnapshotUnavailable) {
		t.Fatalf("Ask() error = %v, want %v", err, chatdomain.ErrSnapshotUnavailable)
	}
}

func TestAskRejectsGraphStrategyWithoutAReadyRelease(t *testing.T) {
	corpusID := uuid.New()
	snapshotID := uuid.New()
	service := NewService(
		fakeReleases{release: snapshotdomain.Release{CorpusID: corpusID, SnapshotID: snapshotID, Version: 2, ActivatedAt: time.Now()}},
		fakeGraphReleases{err: graphdomain.ErrNotFound},
		&fakeAgent{},
	)

	_, err := service.Ask(context.Background(), chatdomain.Request{CorpusID: corpusID, Question: "What applies?", Strategy: "graph"}, func(string) {})

	if !errors.Is(err, chatdomain.ErrGraphUnavailable) {
		t.Fatalf("Ask() error = %v, want %v", err, chatdomain.ErrGraphUnavailable)
	}
}

type fakeReleases struct {
	release snapshotdomain.Release
	err     error
}

func (releases fakeReleases) Active(context.Context, uuid.UUID) (snapshotdomain.Release, error) {
	return releases.release, releases.err
}

type fakeGraphReleases struct {
	release graphdomain.Release
	err     error
}

func (releases fakeGraphReleases) Get(context.Context, uuid.UUID, uuid.UUID) (graphdomain.Release, error) {
	return releases.release, releases.err
}

type fakeAgent struct{ request chatdomain.Request }

func (agent *fakeAgent) Ask(
	_ context.Context,
	request chatdomain.Request,
	_ func(string),
) (chatdomain.Result, error) {
	agent.request = request
	return chatdomain.Result{}, nil
}
