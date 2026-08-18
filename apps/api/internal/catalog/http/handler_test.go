package http

import (
	"context"
	"errors"
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
	err     error
}

type fakeCommands struct {
	created domain.Draft
	result  catalogpostgres.Summary
	err     error
}

func (commands *fakeCommands) Create(_ context.Context, draft domain.Draft) (catalogpostgres.Summary, error) {
	commands.created = draft
	if commands.err != nil {
		return catalogpostgres.Summary{}, commands.err
	}
	corpus, err := domain.NewCorpus(uuid.New(), draft, time.Now())
	return catalogpostgres.Summary{Corpus: corpus}, err
}

func (commands *fakeCommands) Update(
	context.Context, uuid.UUID, domain.Draft, int,
) (catalogpostgres.Summary, error) {
	return commands.result, commands.err
}

func (commands *fakeCommands) Disable(context.Context, uuid.UUID, int) (catalogpostgres.Summary, error) {
	return commands.result, commands.err
}

func (commands *fakeCommands) Enable(context.Context, uuid.UUID, int) (catalogpostgres.Summary, error) {
	return commands.result, commands.err
}

func (reader *fakeReader) List(context.Context, bool) ([]catalogpostgres.Summary, error) {
	return reader.corpora, reader.err
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
	if reader.err != nil {
		return catalogpostgres.Summary{}, reader.err
	}
	if len(reader.corpora) == 0 {
		return catalogpostgres.Summary{}, catalogpostgres.ErrNotFound
	}
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

func TestMutationAndGetRoutesReturnVersionedCorpus(t *testing.T) {
	result := corpusSummary()
	commands := &fakeCommands{result: result}
	mux := http.NewServeMux()
	NewHandler(&fakeReader{corpora: []catalogpostgres.Summary{result}}, commands).Register(mux)
	corpusID := result.ID.String()
	tests := []struct {
		method string
		path   string
		body   string
	}{
		{method: http.MethodGet, path: "/api/v1/corpora/" + corpusID},
		{method: http.MethodPatch, path: "/api/v1/corpora/" + corpusID, body: `{"name":"Privacy","description":"Official materials.","language":"en","jurisdiction":"EU","version":1}`},
		{method: http.MethodPost, path: "/api/v1/corpora/" + corpusID + "/disable", body: `{"version":1}`},
		{method: http.MethodPost, path: "/api/v1/corpora/" + corpusID + "/enable", body: `{"version":1}`},
	}
	for _, test := range tests {
		t.Run(test.method+test.path, func(t *testing.T) {
			recorder := serveCatalogRequest(mux, test.method, test.path, test.body)
			if recorder.Code != http.StatusOK || !strings.Contains(recorder.Body.String(), `"version":1`) {
				t.Fatalf("response = %d/%s, want versioned corpus", recorder.Code, recorder.Body.String())
			}
		})
	}
}

func TestCatalogRoutesMapSafeFailures(t *testing.T) {
	tests := []struct {
		name     string
		reader   *fakeReader
		commands *fakeCommands
		method   string
		path     string
		body     string
		status   int
		code     string
	}{
		{name: "invalid identifier", reader: &fakeReader{}, method: http.MethodGet, path: "/api/v1/corpora/not-a-uuid", status: http.StatusBadRequest, code: "invalid_input"},
		{name: "missing corpus", reader: &fakeReader{}, method: http.MethodGet, path: "/api/v1/corpora/" + uuid.NewString(), status: http.StatusNotFound, code: "not_found"},
		{name: "read unavailable", reader: &fakeReader{err: errors.New("database unavailable")}, method: http.MethodGet, path: "/api/v1/corpora", status: http.StatusServiceUnavailable, code: "unavailable"},
		{name: "invalid mutation body", reader: &fakeReader{}, commands: &fakeCommands{}, method: http.MethodPatch, path: "/api/v1/corpora/" + uuid.NewString(), body: `{"version":0}`, status: http.StatusBadRequest, code: "invalid_input"},
		{name: "stale mutation", reader: &fakeReader{}, commands: &fakeCommands{err: catalogpostgres.ErrStaleState}, method: http.MethodPost, path: "/api/v1/corpora/" + uuid.NewString() + "/disable", body: `{"version":1}`, status: http.StatusConflict, code: "stale_state"},
		{name: "missing mutation target", reader: &fakeReader{}, commands: &fakeCommands{err: catalogpostgres.ErrNotFound}, method: http.MethodPost, path: "/api/v1/corpora/" + uuid.NewString() + "/enable", body: `{"version":1}`, status: http.StatusNotFound, code: "not_found"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			mux := http.NewServeMux()
			if test.commands == nil {
				NewHandler(test.reader).Register(mux)
			} else {
				NewHandler(test.reader, test.commands).Register(mux)
			}
			recorder := serveCatalogRequest(mux, test.method, test.path, test.body)
			if recorder.Code != test.status || !strings.Contains(recorder.Body.String(), `"code":"`+test.code+`"`) {
				t.Fatalf("response = %d/%s, want %d/%s", recorder.Code, recorder.Body.String(), test.status, test.code)
			}
		})
	}
}

func serveCatalogRequest(handler http.Handler, method string, path string, body string) *httptest.ResponseRecorder {
	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, httptest.NewRequest(method, path, strings.NewReader(body)))
	return recorder
}

func corpusSummary() catalogpostgres.Summary {
	now := time.Date(2026, time.August, 18, 12, 0, 0, 0, time.UTC)
	return catalogpostgres.Summary{Corpus: domain.Corpus{
		ID: uuid.New(), Name: "Privacy", Description: "Official materials.",
		Language: domain.LanguageEnglish, Jurisdiction: "European Union",
		Status: domain.StatusEnabled, Version: 1, CreatedAt: now, UpdatedAt: now,
	}}
}
