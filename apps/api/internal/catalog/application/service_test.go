package application

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/dharlanoliveira/norvii/apps/api/internal/catalog/domain"
	catalogpostgres "github.com/dharlanoliveira/norvii/apps/api/internal/catalog/postgres"
	"github.com/google/uuid"
)

type fakeMutations struct {
	created domain.Corpus
	updated domain.Draft
	status  domain.Status
	err     error
}

func (repository *fakeMutations) Create(_ context.Context, corpus domain.Corpus) (catalogpostgres.Summary, error) {
	repository.created = corpus
	return catalogpostgres.Summary{Corpus: corpus}, repository.err
}

func (repository *fakeMutations) Update(
	_ context.Context,
	_ uuid.UUID,
	draft domain.Draft,
	_ int,
	_ time.Time,
) (catalogpostgres.Summary, error) {
	repository.updated = draft
	return catalogpostgres.Summary{}, repository.err
}

func (repository *fakeMutations) SetStatus(
	_ context.Context,
	_ uuid.UUID,
	status domain.Status,
	_ int,
	_ time.Time,
) (catalogpostgres.Summary, error) {
	repository.status = status
	return catalogpostgres.Summary{}, repository.err
}

func TestCreateValidatesNormalizesAndAssignsIdentity(t *testing.T) {
	repository := &fakeMutations{}
	id := uuid.New()
	now := time.Date(2026, time.August, 17, 12, 0, 0, 0, time.UTC)
	service := NewService(repository, func() uuid.UUID { return id }, func() time.Time { return now })

	_, err := service.Create(context.Background(), domain.Draft{
		Name: " Privacy ", Description: " Official materials ",
		Language: domain.LanguageEnglish, Jurisdiction: " European Union ",
	})

	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}
	if repository.created.ID != id || repository.created.Name != "Privacy" {
		t.Fatalf("created corpus = %+v, want assigned normalized aggregate", repository.created)
	}
}

func TestUpdateRejectsInvalidMetadataBeforeRepositoryMutation(t *testing.T) {
	repository := &fakeMutations{}
	service := NewService(repository, uuid.New, time.Now)

	_, err := service.Update(context.Background(), uuid.New(), domain.Draft{}, 1)

	if err == nil || repository.updated.Name != "" {
		t.Fatalf("Update() error/repository = %v/%+v, want validation without mutation", err, repository.updated)
	}
}

func TestDisablePreservesTypedStaleState(t *testing.T) {
	repository := &fakeMutations{err: catalogpostgres.ErrStaleState}
	service := NewService(repository, uuid.New, time.Now)

	_, err := service.Disable(context.Background(), uuid.New(), 2)

	if !errors.Is(err, catalogpostgres.ErrStaleState) {
		t.Fatalf("Disable() error = %v, want stale state", err)
	}
}
