package http

import (
	"context"
	"errors"
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

func TestStageThenActivateUsesDistinctSnapshotOperations(t *testing.T) {
	corpusID, snapshotID := uuid.New(), uuid.New()
	service := &fakeSnapshotService{publication: publication(corpusID)}
	mux := http.NewServeMux()
	NewHandler(service).Register(mux)

	staged := serveSnapshotRequest(
		mux,
		http.MethodPost,
		"/api/v1/corpora/"+corpusID.String()+"/snapshots/stage",
		`{"sourceId":"`+uuid.NewString()+`","documentId":"`+uuid.NewString()+`"}`,
	)
	if staged.Code != http.StatusCreated || !service.staged {
		t.Fatalf("stage response = %d/%s, want created stage", staged.Code, staged.Body.String())
	}
	activated := serveSnapshotRequest(
		mux,
		http.MethodPost,
		"/api/v1/corpora/"+corpusID.String()+"/snapshots/"+snapshotID.String()+"/activate",
		`{"expectedReleaseVersion":0}`,
	)
	if activated.Code != http.StatusCreated || service.activatedSnapshotID != snapshotID || service.expectedReleaseVersion != 0 {
		t.Fatalf("activation response = %d/%s, want graph-ready activation", activated.Code, activated.Body.String())
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
		{name: "graph not ready", err: snapshotdomain.ErrGraphReleaseNotReady, status: http.StatusUnprocessableEntity, code: "graph_release_not_ready"},
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

func TestSnapshotRoutesListAndGetImmutableSnapshots(t *testing.T) {
	corpusID := uuid.New()
	snapshot := publication(corpusID).Snapshot
	snapshot.Members = []snapshotdomain.Member{{
		SourceID: uuid.New(), SourceRevisionID: uuid.New(), DocumentID: uuid.New(),
		OfficialOrigin: "https://example.org/law", CapturedAt: time.Now().UTC(), ContentSHA256: strings.Repeat("b", 64),
	}}
	service := &fakeSnapshotService{publication: snapshotdomain.Publication{Snapshot: snapshot}}
	mux := http.NewServeMux()
	NewHandler(service).Register(mux)

	list := serveSnapshotRequest(mux, http.MethodGet, "/api/v1/corpora/"+corpusID.String()+"/snapshots", "")
	if list.Code != http.StatusOK || !strings.Contains(list.Body.String(), `"sourceRevisionId"`) {
		t.Fatalf("list response = %d/%s, want serialized snapshots", list.Code, list.Body.String())
	}

	get := serveSnapshotRequest(mux, http.MethodGet, "/api/v1/corpora/"+corpusID.String()+"/snapshots/"+snapshot.ID.String(), "")
	if get.Code != http.StatusOK || !strings.Contains(get.Body.String(), `"id":"`+snapshot.ID.String()) {
		t.Fatalf("get response = %d/%s, want snapshot", get.Code, get.Body.String())
	}
}

func TestSnapshotRoutesRejectInvalidIdentifiersAndUnavailableService(t *testing.T) {
	tests := []struct {
		name    string
		method  string
		path    string
		body    string
		service *fakeSnapshotService
		status  int
		code    string
	}{
		{name: "invalid corpus", method: http.MethodGet, path: "/api/v1/corpora/not-a-uuid/snapshots", service: &fakeSnapshotService{}, status: http.StatusBadRequest, code: "invalid_input"},
		{name: "invalid snapshot", method: http.MethodGet, path: "/api/v1/corpora/" + uuid.NewString() + "/snapshots/not-a-uuid", service: &fakeSnapshotService{}, status: http.StatusBadRequest, code: "invalid_input"},
		{name: "invalid publication", method: http.MethodPost, path: "/api/v1/corpora/" + uuid.NewString() + "/snapshots", body: `{}`, service: &fakeSnapshotService{}, status: http.StatusBadRequest, code: "invalid_input"},
		{name: "unavailable list", method: http.MethodGet, path: "/api/v1/corpora/" + uuid.NewString() + "/snapshots", service: &fakeSnapshotService{err: errors.New("database unavailable")}, status: http.StatusServiceUnavailable, code: "unavailable"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			mux := http.NewServeMux()
			NewHandler(test.service).Register(mux)
			recorder := serveSnapshotRequest(mux, test.method, test.path, test.body)
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
	staged                 bool
	activatedSnapshotID    uuid.UUID
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

func (service *fakeSnapshotService) Stage(
	_ context.Context,
	_ uuid.UUID,
	_ uuid.UUID,
	_ uuid.UUID,
) (snapshotdomain.Publication, error) {
	service.staged = true
	return service.publication, service.err
}

func (service *fakeSnapshotService) Activate(
	_ context.Context,
	_ uuid.UUID,
	snapshotID uuid.UUID,
	expectedReleaseVersion int,
) (snapshotdomain.Publication, error) {
	service.activatedSnapshotID = snapshotID
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
