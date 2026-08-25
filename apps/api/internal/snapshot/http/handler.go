// Package http exposes corpus snapshot publication and inspection through the v1 API.
package http

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"time"

	"github.com/dharlanoliveira/norvii/apps/api/internal/platform/httpserver"
	snapshotapplication "github.com/dharlanoliveira/norvii/apps/api/internal/snapshot/application"
	snapshotdomain "github.com/dharlanoliveira/norvii/apps/api/internal/snapshot/domain"
	"github.com/google/uuid"
)

type service interface {
	Get(context.Context, uuid.UUID, uuid.UUID) (snapshotdomain.Snapshot, error)
	List(context.Context, uuid.UUID) ([]snapshotdomain.Snapshot, error)
	Publish(context.Context, uuid.UUID, uuid.UUID, uuid.UUID, int) (snapshotdomain.Publication, error)
	Stage(context.Context, uuid.UUID, uuid.UUID, uuid.UUID) (snapshotdomain.Publication, error)
	Activate(context.Context, uuid.UUID, uuid.UUID, int) (snapshotdomain.Publication, error)
}

// Handler maps explicit snapshot operations to the public HTTP contract.
type Handler struct{ service service }

// NewHandler constructs a snapshot endpoint handler.
func NewHandler(service service) *Handler { return &Handler{service: service} }

// Register adds snapshot routes to the shared API mux.
func (handler *Handler) Register(mux *http.ServeMux) {
	mux.HandleFunc("GET /api/v1/corpora/{corpusId}/snapshots", handler.list)
	mux.HandleFunc("GET /api/v1/corpora/{corpusId}/snapshots/{snapshotId}", handler.get)
	mux.HandleFunc("POST /api/v1/corpora/{corpusId}/snapshots", handler.publish)
	mux.HandleFunc("POST /api/v1/corpora/{corpusId}/snapshots/stage", handler.stage)
	mux.HandleFunc("POST /api/v1/corpora/{corpusId}/snapshots/{snapshotId}/activate", handler.activate)
}

type publishRequest struct {
	SourceID               uuid.UUID `json:"sourceId"`
	DocumentID             uuid.UUID `json:"documentId"`
	ExpectedReleaseVersion int       `json:"expectedReleaseVersion"`
}

type stageRequest struct {
	SourceID   uuid.UUID `json:"sourceId"`
	DocumentID uuid.UUID `json:"documentId"`
}

type activateRequest struct {
	ExpectedReleaseVersion int `json:"expectedReleaseVersion"`
}

type snapshotResponse struct {
	ID             uuid.UUID        `json:"id"`
	CorpusID       uuid.UUID        `json:"corpusId"`
	ManifestSHA256 string           `json:"manifestSha256"`
	CreatedBy      string           `json:"createdBy"`
	CreatedAt      time.Time        `json:"createdAt"`
	Members        []memberResponse `json:"members"`
}

type memberResponse struct {
	SourceID         uuid.UUID `json:"sourceId"`
	SourceRevisionID uuid.UUID `json:"sourceRevisionId"`
	DocumentID       uuid.UUID `json:"documentId"`
	OfficialOrigin   string    `json:"officialOrigin"`
	CapturedAt       time.Time `json:"capturedAt"`
	ContentSHA256    string    `json:"contentSha256"`
}

type publicationResponse struct {
	Snapshot  snapshotResponse `json:"snapshot"`
	Release   releaseResponse  `json:"release"`
	Published bool             `json:"published"`
}

type releaseResponse struct {
	CorpusID    uuid.UUID `json:"corpusId"`
	SnapshotID  uuid.UUID `json:"snapshotId"`
	Version     int       `json:"version"`
	ActivatedAt time.Time `json:"activatedAt"`
}

func (handler *Handler) list(writer http.ResponseWriter, request *http.Request) {
	corpusID, ok := pathID(writer, request, "corpusId")
	if !ok {
		return
	}
	snapshots, err := handler.service.List(request.Context(), corpusID)
	if err != nil {
		writeError(writer, request, err)
		return
	}
	response := make([]snapshotResponse, 0, len(snapshots))
	for _, snapshot := range snapshots {
		response = append(response, newSnapshotResponse(snapshot))
	}
	httpserver.WriteJSON(writer, http.StatusOK, response)
}

func (handler *Handler) get(writer http.ResponseWriter, request *http.Request) {
	corpusID, ok := pathID(writer, request, "corpusId")
	if !ok {
		return
	}
	snapshotID, ok := pathID(writer, request, "snapshotId")
	if !ok {
		return
	}
	snapshot, err := handler.service.Get(request.Context(), corpusID, snapshotID)
	if err != nil {
		writeError(writer, request, err)
		return
	}
	httpserver.WriteJSON(writer, http.StatusOK, newSnapshotResponse(snapshot))
}

func (handler *Handler) publish(writer http.ResponseWriter, request *http.Request) {
	corpusID, ok := pathID(writer, request, "corpusId")
	if !ok {
		return
	}
	var body publishRequest
	if err := decodeJSON(request, &body); err != nil || body.ExpectedReleaseVersion < 1 || body.SourceID == uuid.Nil || body.DocumentID == uuid.Nil {
		httpserver.WriteError(writer, request, httpserver.Problem{Status: http.StatusBadRequest, Code: "invalid_input", Message: "The snapshot publication request is invalid."})
		return
	}
	publication, err := handler.service.Publish(request.Context(), corpusID, body.SourceID, body.DocumentID, body.ExpectedReleaseVersion)
	if err != nil {
		writeError(writer, request, err)
		return
	}
	status := http.StatusCreated
	if !publication.Created {
		status = http.StatusOK
	}
	httpserver.WriteJSON(writer, status, newPublicationResponse(publication))
}

func (handler *Handler) stage(writer http.ResponseWriter, request *http.Request) {
	corpusID, ok := pathID(writer, request, "corpusId")
	if !ok {
		return
	}
	var body stageRequest
	if err := decodeJSON(request, &body); err != nil || body.SourceID == uuid.Nil || body.DocumentID == uuid.Nil {
		httpserver.WriteError(writer, request, httpserver.Problem{Status: http.StatusBadRequest, Code: "invalid_input", Message: "The snapshot staging request is invalid."})
		return
	}
	publication, err := handler.service.Stage(request.Context(), corpusID, body.SourceID, body.DocumentID)
	if err != nil {
		writeError(writer, request, err)
		return
	}
	status := http.StatusCreated
	if !publication.Created {
		status = http.StatusOK
	}
	httpserver.WriteJSON(writer, status, newPublicationResponse(publication))
}

func (handler *Handler) activate(writer http.ResponseWriter, request *http.Request) {
	corpusID, ok := pathID(writer, request, "corpusId")
	if !ok {
		return
	}
	snapshotID, ok := pathID(writer, request, "snapshotId")
	if !ok {
		return
	}
	var body activateRequest
	if err := decodeJSON(request, &body); err != nil || body.ExpectedReleaseVersion < 0 {
		httpserver.WriteError(writer, request, httpserver.Problem{Status: http.StatusBadRequest, Code: "invalid_input", Message: "The snapshot activation request is invalid."})
		return
	}
	publication, err := handler.service.Activate(request.Context(), corpusID, snapshotID, body.ExpectedReleaseVersion)
	if err != nil {
		writeError(writer, request, err)
		return
	}
	status := http.StatusCreated
	if !publication.Created {
		status = http.StatusOK
	}
	httpserver.WriteJSON(writer, status, newPublicationResponse(publication))
}

func pathID(writer http.ResponseWriter, request *http.Request, key string) (uuid.UUID, bool) {
	id, err := uuid.Parse(request.PathValue(key))
	if err != nil {
		httpserver.WriteError(writer, request, httpserver.Problem{Status: http.StatusBadRequest, Code: "invalid_input", Message: "The snapshot identifier is invalid."})
		return uuid.Nil, false
	}
	return id, true
}

func decodeJSON(request *http.Request, destination any) error {
	decoder := json.NewDecoder(request.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(destination); err != nil {
		return err
	}
	if err := decoder.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
		return errors.New("snapshot request must contain one JSON object")
	}
	return nil
}

func writeError(writer http.ResponseWriter, request *http.Request, err error) {
	switch {
	case errors.Is(err, snapshotdomain.ErrNotFound):
		httpserver.WriteError(writer, request, httpserver.Problem{Status: http.StatusNotFound, Code: "not_found", Message: "The corpus snapshot was not found."})
	case errors.Is(err, snapshotdomain.ErrStaleRelease):
		httpserver.WriteError(writer, request, httpserver.Problem{Status: http.StatusConflict, Code: "stale_state", Message: "The active snapshot changed; reload and retry."})
	case errors.Is(err, snapshotdomain.ErrCandidateNotReady):
		httpserver.WriteError(writer, request, httpserver.Problem{Status: http.StatusUnprocessableEntity, Code: "publication_failed", Message: "The source candidate is not ready for publication."})
	case errors.Is(err, snapshotdomain.ErrGraphReleaseNotReady):
		httpserver.WriteError(writer, request, httpserver.Problem{Status: http.StatusUnprocessableEntity, Code: "graph_release_not_ready", Message: "The snapshot graph release is not ready for activation."})
	case errors.Is(err, snapshotdomain.ErrInvalidInput):
		httpserver.WriteError(writer, request, httpserver.Problem{Status: http.StatusBadRequest, Code: "invalid_input", Message: "The snapshot publication request is invalid."})
	default:
		httpserver.WriteError(writer, request, httpserver.Problem{Status: http.StatusServiceUnavailable, Code: "unavailable", Message: "Snapshot publication is temporarily unavailable."})
	}
}

func newSnapshotResponse(snapshot snapshotdomain.Snapshot) snapshotResponse {
	members := make([]memberResponse, 0, len(snapshot.Members))
	for _, member := range snapshot.Members {
		members = append(members, memberResponse{
			SourceID: member.SourceID, SourceRevisionID: member.SourceRevisionID,
			DocumentID: member.DocumentID, OfficialOrigin: member.OfficialOrigin,
			CapturedAt: member.CapturedAt, ContentSHA256: member.ContentSHA256,
		})
	}
	return snapshotResponse{
		ID: snapshot.ID, CorpusID: snapshot.CorpusID, ManifestSHA256: snapshot.ManifestSHA256,
		CreatedBy: snapshot.CreatedBy, CreatedAt: snapshot.CreatedAt, Members: members,
	}
}

func newPublicationResponse(publication snapshotdomain.Publication) publicationResponse {
	return publicationResponse{
		Snapshot: newSnapshotResponse(publication.Snapshot), Published: publication.Created,
		Release: releaseResponse{CorpusID: publication.Release.CorpusID, SnapshotID: publication.Release.SnapshotID, Version: publication.Release.Version, ActivatedAt: publication.Release.ActivatedAt},
	}
}

// Compile-time check keeps the concrete API service aligned with the endpoint's consumer port.
var _ service = (*snapshotapplication.Service)(nil)
