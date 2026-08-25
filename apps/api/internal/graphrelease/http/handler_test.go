package http

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	graphdomain "github.com/dharlanoliveira/norvii/apps/api/internal/graphrelease/domain"
	"github.com/google/uuid"
)

func TestHandlerReturnsSnapshotScopedGraphRelease(t *testing.T) {
	now := time.Date(2026, time.August, 24, 12, 0, 0, 0, time.UTC)
	completed := now.Add(time.Minute)
	release := graphdomain.Release{
		ID: uuid.New(), CorpusID: uuid.New(), SnapshotID: uuid.New(),
		ManifestSHA256: "manifest", BuildVersion: "legal-graph-v1", Status: graphdomain.StatusReady,
		EntityCount: 2, RelationshipCount: 1, CreatedAt: now, CompletedAt: &completed,
	}
	mux := http.NewServeMux()
	NewHandler(fakeGraphReleaseService{release: release}).Register(mux)
	recorder := httptest.NewRecorder()
	mux.ServeHTTP(recorder, httptest.NewRequest(
		http.MethodGet,
		"/api/v1/corpora/"+release.CorpusID.String()+"/snapshots/"+release.SnapshotID.String()+"/graph-release",
		nil,
	))
	if recorder.Code != http.StatusOK || !strings.Contains(recorder.Body.String(), `"status":"ready"`) {
		t.Fatalf("response = %d/%s, want ready release", recorder.Code, recorder.Body.String())
	}
}

func TestHandlerMapsGraphReleaseFailures(t *testing.T) {
	tests := []struct {
		name string
		err  error
		code string
	}{
		{name: "missing", err: graphdomain.ErrNotFound, code: "not_found"},
		{name: "unavailable", err: errors.New("database unavailable"), code: "unavailable"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			mux := http.NewServeMux()
			NewHandler(fakeGraphReleaseService{err: test.err}).Register(mux)
			recorder := httptest.NewRecorder()
			mux.ServeHTTP(recorder, httptest.NewRequest(
				http.MethodGet,
				"/api/v1/corpora/"+uuid.NewString()+"/snapshots/"+uuid.NewString()+"/graph-release",
				nil,
			))
			if !strings.Contains(recorder.Body.String(), `"code":"`+test.code+`"`) {
				t.Fatalf("response = %s, want %q", recorder.Body.String(), test.code)
			}
		})
	}
}

type fakeGraphReleaseService struct {
	release graphdomain.Release
	err     error
}

func (fake fakeGraphReleaseService) Get(_ context.Context, _ uuid.UUID, _ uuid.UUID) (graphdomain.Release, error) {
	return fake.release, fake.err
}
