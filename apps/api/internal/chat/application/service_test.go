package application

import (
	"context"
	"errors"
	"testing"
	"time"

	chatdomain "github.com/dharlanoliveira/norvii/apps/api/internal/chat/domain"
	snapshotdomain "github.com/dharlanoliveira/norvii/apps/api/internal/snapshot/domain"
	"github.com/google/uuid"
)

func TestAskForwardsOnlyTheActiveSnapshot(t *testing.T) {
	corpusID := uuid.New()
	snapshotID := uuid.New()
	agent := &fakeAgent{}
	service := NewService(
		fakeReleases{release: activeRelease(corpusID, snapshotID)},
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
	service := NewService(fakeReleases{err: snapshotdomain.ErrNotFound}, &fakeAgent{})

	_, err := service.Ask(context.Background(), chatdomain.Request{CorpusID: uuid.New(), Question: "What applies?"}, func(string) {})

	if !errors.Is(err, chatdomain.ErrSnapshotUnavailable) {
		t.Fatalf("Ask() error = %v, want %v", err, chatdomain.ErrSnapshotUnavailable)
	}
}

func TestAskAllowsHybridWithoutAPreflightGraphRelease(t *testing.T) {
	corpusID := uuid.New()
	snapshotID := uuid.New()
	agent := &fakeAgent{}
	service := NewService(fakeReleases{release: activeRelease(corpusID, snapshotID)}, agent)

	_, err := service.Ask(
		context.Background(),
		chatdomain.Request{CorpusID: corpusID, Question: "What applies?", Strategy: "hybrid"},
		func(string) {},
	)

	if err != nil {
		t.Fatalf("Ask() error = %v", err)
	}
	if agent.request.Strategy != "hybrid" {
		t.Fatalf("agent strategy = %q, want hybrid", agent.request.Strategy)
	}
}

type fakeReleases struct {
	release snapshotdomain.Release
	err     error
}

func (releases fakeReleases) Active(context.Context, uuid.UUID) (snapshotdomain.Release, error) {
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

func activeRelease(corpusID, snapshotID uuid.UUID) snapshotdomain.Release {
	return snapshotdomain.Release{
		CorpusID: corpusID, SnapshotID: snapshotID, Version: 2, ActivatedAt: time.Now(),
	}
}
