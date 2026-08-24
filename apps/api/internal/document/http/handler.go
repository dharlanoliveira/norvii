// Package http exposes corpus-scoped immutable documents through the v1 API.
package http

import (
	"context"
	"errors"
	"net/http"
	"time"

	documentpostgres "github.com/dharlanoliveira/norvii/apps/api/internal/document/postgres"
	"github.com/dharlanoliveira/norvii/apps/api/internal/platform/httpserver"
	"github.com/google/uuid"
)

type reader interface {
	GetLatest(context.Context, uuid.UUID, uuid.UUID) (documentpostgres.Document, error)
	GetVersion(context.Context, uuid.UUID, uuid.UUID, uuid.UUID) (documentpostgres.Document, error)
}

// Handler maps immutable document reads to the stable v1 HTTP contract.
type Handler struct {
	reader reader
}

// NewHandler constructs a document HTTP handler around its read port.
func NewHandler(reader reader) *Handler { return &Handler{reader: reader} }

// Register adds the latest ready document route to a shared application mux.
func (handler *Handler) Register(mux *http.ServeMux) {
	mux.HandleFunc(
		"GET /api/v1/corpora/{corpusId}/sources/{sourceId}/document",
		handler.getLatest,
	)
	mux.HandleFunc(
		"GET /api/v1/corpora/{corpusId}/sources/{sourceId}/documents/{documentVersionId}",
		handler.getVersion,
	)
}

func (handler *Handler) getLatest(writer http.ResponseWriter, request *http.Request) {
	corpusID, corpusError := uuid.Parse(request.PathValue("corpusId"))
	sourceID, sourceError := uuid.Parse(request.PathValue("sourceId"))
	if corpusError != nil || sourceError != nil {
		writeInvalidID(writer, request)
		return
	}
	document, err := handler.reader.GetLatest(request.Context(), corpusID, sourceID)
	if errors.Is(err, documentpostgres.ErrNotFound) {
		writeNotFound(writer, request)
		return
	}
	if err != nil {
		writeUnavailable(writer, request)
		return
	}
	httpserver.WriteJSON(writer, http.StatusOK, newDocumentResponse(document))
}

func (handler *Handler) getVersion(writer http.ResponseWriter, request *http.Request) {
	corpusID, corpusError := uuid.Parse(request.PathValue("corpusId"))
	sourceID, sourceError := uuid.Parse(request.PathValue("sourceId"))
	documentVersionID, documentError := uuid.Parse(request.PathValue("documentVersionId"))
	if corpusError != nil || sourceError != nil || documentError != nil {
		writeInvalidID(writer, request)
		return
	}
	document, err := handler.reader.GetVersion(
		request.Context(), corpusID, sourceID, documentVersionID,
	)
	if errors.Is(err, documentpostgres.ErrNotFound) {
		writeNotFound(writer, request)
		return
	}
	if err != nil {
		writeUnavailable(writer, request)
		return
	}
	httpserver.WriteJSON(writer, http.StatusOK, newDocumentResponse(document))
}

type unitResponse struct {
	ID            uuid.UUID  `json:"id"`
	ParentID      *uuid.UUID `json:"parentId"`
	Kind          string     `json:"kind"`
	Ordinal       int        `json:"ordinal"`
	Marker        *string    `json:"marker"`
	Label         *string    `json:"label"`
	StartOffset   int        `json:"startOffset"`
	EndOffset     int        `json:"endOffset"`
	StartPage     *int       `json:"startPage,omitempty"`
	EndPage       *int       `json:"endPage,omitempty"`
	Locator       string     `json:"locator"`
	ContentSHA256 string     `json:"contentSha256"`
}

type documentResponse struct {
	ID               uuid.UUID          `json:"id"`
	SourceRevisionID uuid.UUID          `json:"sourceRevisionId"`
	PipelineVersion  string             `json:"pipelineVersion"`
	Text             string             `json:"text"`
	TextSHA256       string             `json:"textSha256"`
	CreatedAt        time.Time          `json:"createdAt"`
	Units            []unitResponse     `json:"units"`
	Provenance       provenanceResponse `json:"provenance"`
}

type provenanceResponse struct {
	ContentSHA256          string    `json:"contentSha256"`
	CapturedAt             time.Time `json:"capturedAt"`
	MediaType              string    `json:"mediaType"`
	ByteSize               int64     `json:"byteSize"`
	FinalURL               *string   `json:"finalUrl"`
	ExtractedContentSHA256 *string   `json:"extractedContentSha256"`
}

func newDocumentResponse(document documentpostgres.Document) documentResponse {
	units := make([]unitResponse, 0, len(document.Units))
	for _, unit := range document.Units {
		units = append(units, unitResponse{
			ID: unit.ID, ParentID: unit.ParentID, Kind: unit.Kind, Ordinal: unit.Ordinal,
			Marker: unit.Marker, Label: unit.Label, StartOffset: unit.StartOffset,
			EndOffset: unit.EndOffset, StartPage: unit.StartPage, EndPage: unit.EndPage,
			Locator: unit.Locator, ContentSHA256: unit.ContentSHA256,
		})
	}
	return documentResponse{
		ID: document.ID, SourceRevisionID: document.SourceRevisionID,
		PipelineVersion: document.PipelineVersion, Text: document.Text,
		TextSHA256: document.TextSHA256, CreatedAt: document.CreatedAt, Units: units,
		Provenance: provenanceResponse{
			ContentSHA256: document.Provenance.ContentSHA256,
			CapturedAt:    document.Provenance.CapturedAt, MediaType: document.Provenance.MediaType,
			ByteSize: document.Provenance.ByteSize, FinalURL: document.Provenance.FinalURL,
			ExtractedContentSHA256: document.Provenance.ExtractedContentSHA256,
		},
	}
}

func writeInvalidID(writer http.ResponseWriter, request *http.Request) {
	httpserver.WriteError(writer, request, httpserver.Problem{
		Status: http.StatusBadRequest, Code: "invalid_input", Message: "The document path is invalid.",
	})
}

func writeNotFound(writer http.ResponseWriter, request *http.Request) {
	httpserver.WriteError(writer, request, httpserver.Problem{
		Status: http.StatusNotFound, Code: "not_found", Message: "A ready document was not found.",
	})
}

func writeUnavailable(writer http.ResponseWriter, request *http.Request) {
	httpserver.WriteError(writer, request, httpserver.Problem{
		Status: http.StatusServiceUnavailable, Code: "unavailable",
		Message: "The document store is temporarily unavailable.",
	})
}
