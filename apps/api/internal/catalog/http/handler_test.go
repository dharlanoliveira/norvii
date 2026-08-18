package http

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/dharlanoliveira/norvii/apps/api/internal/catalog/domain"
	catalogpostgres "github.com/dharlanoliveira/norvii/apps/api/internal/catalog/postgres"
	"github.com/google/uuid"
)

type fakeReader struct {
	corpora []catalogpostgres.Summary
}

type fakeCommands struct {
	created domain.Draft
}

func (commands *fakeCommands) Create(_ context.Context, draft domain.Draft) (catalogpostgres.Summary, error) {
	commands.created = draft
	corpus, err := domain.NewCorpus(uuid.New(), draft, time.Now())
	return catalogpostgres.Summary{Corpus: corpus}, err
}

func (commands *fakeCommands) Update(
	context.Context, uuid.UUID, domain.Draft, int,
) (catalogpostgres.Summary, error) {
	return catalogpostgres.Summary{}, nil
}

func (commands *fakeCommands) Disable(context.Context, uuid.UUID, int) (catalogpostgres.Summary, error) {
	return catalogpostgres.Summary{}, nil
}

func (commands *fakeCommands) Enable(context.Context, uuid.UUID, int) (catalogpostgres.Summary, error) {
	return catalogpostgres.Summary{}, nil
}

func (reader *fakeReader) List(context.Context, bool) ([]catalogpostgres.Summary, error) {
	return reader.corpora, nil
}

func TestCreateValidatesAndWritesCreatedCorpus(t *testing.T) {
	commands := &fakeCommands{}
	mux := http.NewServeMux()
	NewHandler(&fakeReader{}, commands).Register(mux)
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(
		http.MethodPost,
		"/api/v1/corpora",
		strings.NewReader(`{
			"name":"Privacy","description":"Official materials.",
			"language":"en","jurisdiction":"European Union"
		}`),
	)

	mux.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusCreated {
		t.Fatalf("status/body = %d/%s, want 201", recorder.Code, recorder.Body.String())
	}
	if commands.created.Name != "Privacy" {
		t.Fatalf("Create() draft = %+v, want request metadata", commands.created)
	}
}

func (reader *fakeReader) Get(context.Context, uuid.UUID, bool) (catalogpostgres.Summary, error) {
	return reader.corpora[0], nil
}

func TestListWritesVersionedCorpusProjection(t *testing.T) {
	now := time.Date(2026, time.August, 17, 12, 0, 0, 0, time.UTC)
	reader := &fakeReader{corpora: []catalogpostgres.Summary{{
		Corpus: domain.Corpus{
			ID:   uuid.MustParse("10000000-0000-4000-8000-000000000002"),
			Name: "EU Privacy Law", Description: "Official materials.",
			Language: domain.LanguageEnglish, Jurisdiction: "European Union",
			Status: domain.StatusEnabled, Version: 1, CreatedAt: now, UpdatedAt: now,
		},
		SourceCount: 1,
	}}}
	mux := http.NewServeMux()
	NewHandler(reader).Register(mux)
	recorder := httptest.NewRecorder()

	mux.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/api/v1/corpora", nil))

	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", recorder.Code)
	}
	if !strings.Contains(recorder.Body.String(), `"sourceCount":1`) {
		t.Fatalf("response = %s, want source count", recorder.Body.String())
	}
}
