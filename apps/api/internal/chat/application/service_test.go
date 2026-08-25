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

func TestAskUsesAReadyGraphReleaseForGraphBackedStrategies(t *testing.T) {
	corpusID := uuid.New()
	snapshotID := uuid.New()
	for _, strategy := range []string{"graph", "hybrid"} {
		t.Run(strategy, func(t *testing.T) {
			graphReleases := &recordingGraphReleases{release: readyGraphRelease(corpusID, snapshotID)}
			agent := &fakeAgent{}
			service := NewService(
				fakeReleases{release: snapshotdomain.Release{CorpusID: corpusID, SnapshotID: snapshotID, Version: 2, ActivatedAt: time.Now()}},
				graphReleases,
				agent,
			)

			_, err := service.Ask(context.Background(), chatdomain.Request{CorpusID: corpusID, Question: "What applies?", Strategy: strategy}, func(string) {})

			if err != nil {
				t.Fatalf("Ask() error = %v", err)
			}
			if graphReleases.calls != 1 {
				t.Fatalf("graph release calls = %d, want 1", graphReleases.calls)
			}
			if agent.request.SnapshotID != snapshotID {
				t.Fatalf("agent snapshot = %s, want %s", agent.request.SnapshotID, snapshotID)
			}
		})
	}
}

func TestAskRejectsAnIncompleteGraphRelease(t *testing.T) {
	corpusID := uuid.New()
	snapshotID := uuid.New()
	service := NewService(
		fakeReleases{release: snapshotdomain.Release{CorpusID: corpusID, SnapshotID: snapshotID, Version: 2, ActivatedAt: time.Now()}},
		fakeGraphReleases{release: graphdomain.Release{ID: uuid.New(), Status: graphdomain.StatusBuilding}},
		&fakeAgent{},
	)

	_, err := service.Ask(context.Background(), chatdomain.Request{CorpusID: corpusID, Question: "What applies?", Strategy: "graph"}, func(string) {})

	if !errors.Is(err, chatdomain.ErrGraphUnavailable) {
		t.Fatalf("Ask() error = %v, want %v", err, chatdomain.ErrGraphUnavailable)
	}
}

func TestAskPreservesUnexpectedGraphReleaseFailures(t *testing.T) {
	corpusID := uuid.New()
	snapshotID := uuid.New()
	graphFailure := errors.New("graph store unavailable")
	service := NewService(
		fakeReleases{release: snapshotdomain.Release{CorpusID: corpusID, SnapshotID: snapshotID, Version: 2, ActivatedAt: time.Now()}},
		fakeGraphReleases{err: graphFailure},
		&fakeAgent{},
	)

	_, err := service.Ask(context.Background(), chatdomain.Request{CorpusID: corpusID, Question: "What applies?", Strategy: "hybrid"}, func(string) {})

	if !errors.Is(err, graphFailure) {
		t.Fatalf("Ask() error = %v, want wrapped graph failure", err)
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

type recordingGraphReleases struct {
	release graphdomain.Release
	calls   int
}

func (releases *recordingGraphReleases) Get(context.Context, uuid.UUID, uuid.UUID) (graphdomain.Release, error) {
	releases.calls++
	return releases.release, nil
}

func readyGraphRelease(corpusID, snapshotID uuid.UUID) graphdomain.Release {
	completedAt := time.Now()
	return graphdomain.Release{
		ID:             uuid.New(),
		CorpusID:       corpusID,
		SnapshotID:     snapshotID,
		Status:         graphdomain.StatusReady,
		CompletedAt:    &completedAt,
		BuildVersion:   "legal-graph-v1",
		ManifestSHA256: "manifest",
	}
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
