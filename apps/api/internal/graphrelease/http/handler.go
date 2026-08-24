// Package http exposes graph-release readiness through the public API.
package http

import (
	"context"
	"errors"
	"net/http"
	"time"

	graphapplication "github.com/dharlanoliveira/norvii/apps/api/internal/graphrelease/application"
	graphdomain "github.com/dharlanoliveira/norvii/apps/api/internal/graphrelease/domain"
	"github.com/dharlanoliveira/norvii/apps/api/internal/platform/httpserver"
	"github.com/google/uuid"
)

type service interface {
	Get(context.Context, uuid.UUID, uuid.UUID) (graphdomain.Release, error)
}

// Handler maps graph-release inspection to one corpus-scoped endpoint.
type Handler struct{ service service }

// NewHandler constructs a graph-release handler.
func NewHandler(service service) *Handler { return &Handler{service: service} }

// Register adds the immutable snapshot graph-release route.
func (handler *Handler) Register(mux *http.ServeMux) {
	mux.HandleFunc("GET /api/v1/corpora/{corpusId}/snapshots/{snapshotId}/graph-release", handler.get)
}

type response struct {
	ID                uuid.UUID          `json:"id"`
	CorpusID          uuid.UUID          `json:"corpusId"`
	SnapshotID        uuid.UUID          `json:"snapshotId"`
	ManifestSHA256    string             `json:"manifestSha256"`
	BuildVersion      string             `json:"buildVersion"`
	Status            graphdomain.Status `json:"status"`
	FailureCategory   string             `json:"failureCategory,omitempty"`
	EntityCount       int                `json:"entityCount"`
	RelationshipCount int                `json:"relationshipCount"`
	CreatedAt         time.Time          `json:"createdAt"`
	CompletedAt       *time.Time         `json:"completedAt,omitempty"`
}

func (handler *Handler) get(writer http.ResponseWriter, request *http.Request) {
	corpusID, corpusErr := uuid.Parse(request.PathValue("corpusId"))
	snapshotID, snapshotErr := uuid.Parse(request.PathValue("snapshotId"))
	if corpusErr != nil || snapshotErr != nil {
		httpserver.WriteError(writer, request, httpserver.Problem{Status: http.StatusBadRequest, Code: "invalid_input", Message: "The graph release identifier is invalid."})
		return
	}
	release, err := handler.service.Get(request.Context(), corpusID, snapshotID)
	if errors.Is(err, graphdomain.ErrNotFound) {
		httpserver.WriteError(writer, request, httpserver.Problem{Status: http.StatusNotFound, Code: "not_found", Message: "The graph release was not found."})
		return
	}
	if err != nil {
		httpserver.WriteError(writer, request, httpserver.Problem{Status: http.StatusServiceUnavailable, Code: "unavailable", Message: "Graph-release inspection is temporarily unavailable."})
		return
	}
	httpserver.WriteJSON(writer, http.StatusOK, response{
		ID: release.ID, CorpusID: release.CorpusID, SnapshotID: release.SnapshotID,
		ManifestSHA256: release.ManifestSHA256, BuildVersion: release.BuildVersion,
		Status: release.Status, FailureCategory: release.FailureCategory,
		EntityCount: release.EntityCount, RelationshipCount: release.RelationshipCount,
		CreatedAt: release.CreatedAt, CompletedAt: release.CompletedAt,
	})
}

var _ service = (*graphapplication.Service)(nil)
