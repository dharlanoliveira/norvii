package application

import (
	"context"
	"testing"
	"time"

	"github.com/dharlanoliveira/norvii/apps/api/internal/source/domain"
	sourcepostgres "github.com/dharlanoliveira/norvii/apps/api/internal/source/postgres"
	"github.com/google/uuid"
)

type fakeURLRepository struct {
	source        domain.Source
	submittedURL  string
	normalizedURL string
	workID        uuid.UUID
	queuedStatus  domain.Status
	queuedReason  string
}

func (repository *fakeURLRepository) CreateURL(
	_ context.Context,
	source domain.Source,
	submittedURL string,
	normalizedURL string,
	workID uuid.UUID,
) (sourcepostgres.Record, error) {
	repository.source = source
	repository.submittedURL = submittedURL
	repository.normalizedURL = normalizedURL
	repository.workID = workID
	return sourcepostgres.Record{
		ID: source.ID, CorpusID: source.CorpusID, Title: source.Title,
		Kind: source.Kind, ProcessingStatus: source.Status, Version: source.Version,
		CreatedAt: source.CreatedAt, UpdatedAt: source.UpdatedAt,
	}, nil
}

func (repository *fakeURLRepository) CreatePDF(
	context.Context, domain.Source, domain.PDFOrigin, uuid.UUID,
) (sourcepostgres.Record, error) {
	return sourcepostgres.Record{}, nil
}

func (repository *fakeURLRepository) QueueLifecycle(
	_ context.Context,
	_ uuid.UUID,
	_ uuid.UUID,
	_ int,
	status domain.Status,
	reason string,
	_ uuid.UUID,
	_ time.Time,
) (sourcepostgres.Record, error) {
	repository.queuedStatus = status
	repository.queuedReason = reason
	return sourcepostgres.Record{}, nil
}

func TestCreateURLBuildsPendingSourceAndInitialWork(t *testing.T) {
	repository := &fakeURLRepository{}
	identities := []uuid.UUID{uuid.New(), uuid.New()}
	index := 0
	now := time.Date(2026, time.August, 17, 12, 0, 0, 0, time.UTC)
	service := NewService(repository, func() uuid.UUID {
		identity := identities[index]
		index++
		return identity
	}, func() time.Time { return now })
	corpusID := uuid.New()

	created, err := service.CreateURL(
		context.Background(), corpusID, " Official text ",
		"HTTPS://EXAMPLE.COM:443/law?b=2&a=1#part",
	)

	if err != nil {
		t.Fatalf("CreateURL() error = %v", err)
	}
	if created.ID != identities[0] || repository.workID != identities[1] {
		t.Fatalf("identities = source:%s work:%s, want injected identities", created.ID, repository.workID)
	}
	if repository.source.Status != domain.StatusPending || repository.normalizedURL != "https://example.com/law?a=1&b=2" {
		t.Fatalf("creation = %+v/%q, want pending canonical URL", repository.source, repository.normalizedURL)
	}
}

func TestRetryAndReprocessRequireTheirExplicitLifecycleStates(t *testing.T) {
	repository := &fakeURLRepository{}
	service := NewService(repository, uuid.New, time.Now)

	if _, err := service.Retry(context.Background(), uuid.New(), uuid.New(), 2); err != nil {
		t.Fatalf("Retry() error = %v", err)
	}
	if repository.queuedStatus != domain.StatusFailed || repository.queuedReason != "retry" {
		t.Fatalf("retry queue = %s/%s, want failed/retry", repository.queuedStatus, repository.queuedReason)
	}
	if _, err := service.Reprocess(context.Background(), uuid.New(), uuid.New(), 3); err != nil {
		t.Fatalf("Reprocess() error = %v", err)
	}
	if repository.queuedStatus != domain.StatusReady || repository.queuedReason != "reprocess" {
		t.Fatalf("reprocess queue = %s/%s, want ready/reprocess", repository.queuedStatus, repository.queuedReason)
	}
}
