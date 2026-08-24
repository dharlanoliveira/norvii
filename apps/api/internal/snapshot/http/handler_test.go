package http

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	snapshotdomain "github.com/dharlanoliveira/norvii/apps/api/internal/snapshot/domain"
	"github.com/google/uuid"
)

func TestPublishCreatesAReleaseForAReadyCandidate(t *testing.T) {
	corpusID := uuid.New()
	service := &fakeSnapshotService{publication: publication(corpusID)}
	mux := http.NewServeMux()
	NewHandler(service).Register(mux)

	recorder := serveSnapshotRequest(
		mux,
		http.MethodPost,
		"/api/v1/corpora/"+corpusID.String()+"/snapshots",
		`{"sourceId":"`+uuid.NewString()+`","documentId":"`+uuid.NewString()+`","expectedReleaseVersion":1}`,
	)

	if recorder.Code != http.StatusCreated || !strings.Contains(recorder.Body.String(), `"published":true`) {
		t.Fatalf("response = %d/%s, want created publication", recorder.Code, recorder.Body.String())
	}
	if service.expectedReleaseVersion != 1 {
		t.Fatalf("expected release version = %d, want 1", service.expectedReleaseVersion)
	}
}

func TestSnapshotRoutesMapSafePublicationFailures(t *testing.T) {
	corpusID := uuid.New()
	tests := []struct {
		name   string
		err    error
		status int
		code   string
	}{
		{name: "stale release", err: snapshotdomain.ErrStaleRelease, status: http.StatusConflict, code: "stale_state"},
		{name: "candidate not ready", err: snapshotdomain.ErrCandidateNotReady, status: http.StatusUnprocessableEntity, code: "publication_failed"},
		{name: "missing snapshot", err: snapshotdomain.ErrNotFound, status: http.StatusNotFound, code: "not_found"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			mux := http.NewServeMux()
			NewHandler(&fakeSnapshotService{err: test.err}).Register(mux)
			recorder := serveSnapshotRequest(
				mux,
				http.MethodPost,
				"/api/v1/corpora/"+corpusID.String()+"/snapshots",
				`{"sourceId":"`+uuid.NewString()+`","documentId":"`+uuid.NewString()+`","expectedReleaseVersion":1}`,
			)
			if recorder.Code != test.status || !strings.Contains(recorder.Body.String(), `"code":"`+test.code+`"`) {
				t.Fatalf("response = %d/%s, want %d/%s", recorder.Code, recorder.Body.String(), test.status, test.code)
			}
		})
	}
}

func TestPublishRejectsARequestWithAdditionalJSON(t *testing.T) {
	mux := http.NewServeMux()
	NewHandler(&fakeSnapshotService{}).Register(mux)
	recorder := serveSnapshotRequest(
		mux,
		http.MethodPost,
		"/api/v1/corpora/"+uuid.NewString()+"/snapshots",
		`{"sourceId":"`+uuid.NewString()+`","documentId":"`+uuid.NewString()+`","expectedReleaseVersion":1}{}`,
	)
	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusBadRequest)
	}
}

type fakeSnapshotService struct {
	publication            snapshotdomain.Publication
	err                    error
	expectedReleaseVersion int
}

func (service *fakeSnapshotService) Get(context.Context, uuid.UUID, uuid.UUID) (snapshotdomain.Snapshot, error) {
	return service.publication.Snapshot, service.err
}

func (service *fakeSnapshotService) List(context.Context, uuid.UUID) ([]snapshotdomain.Snapshot, error) {
	return []snapshotdomain.Snapshot{service.publication.Snapshot}, service.err
}

func (service *fakeSnapshotService) Publish(
	_ context.Context,
	_ uuid.UUID,
	_ uuid.UUID,
	_ uuid.UUID,
	expectedReleaseVersion int,
) (snapshotdomain.Publication, error) {
	service.expectedReleaseVersion = expectedReleaseVersion
	return service.publication, service.err
}

func publication(corpusID uuid.UUID) snapshotdomain.Publication {
	now := time.Date(2026, time.August, 24, 12, 0, 0, 0, time.UTC)
	snapshotID := uuid.New()
	return snapshotdomain.Publication{
		Snapshot: snapshotdomain.Snapshot{ID: snapshotID, CorpusID: corpusID, ManifestSHA256: strings.Repeat("a", 64), CreatedBy: "local-maintainer", CreatedAt: now},
		Release:  snapshotdomain.Release{CorpusID: corpusID, SnapshotID: snapshotID, Version: 1, ActivatedAt: now},
		Created:  true,
	}
}

func serveSnapshotRequest(handler http.Handler, method, path, body string) *httptest.ResponseRecorder {
	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, httptest.NewRequest(method, path, strings.NewReader(body)))
	return recorder
}
