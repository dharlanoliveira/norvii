// Package http exposes corpus opening suggestions through the v1 API.
package http

import (
	"context"
	"net/http"

	"github.com/dharlanoliveira/norvii/apps/api/internal/platform/httpserver"
	"github.com/dharlanoliveira/norvii/apps/api/internal/suggestions/domain"
	suggestionspostgres "github.com/dharlanoliveira/norvii/apps/api/internal/suggestions/postgres"
	"github.com/google/uuid"
)

type reader interface {
	Read(context.Context, uuid.UUID, domain.QueryLanguage) (suggestionspostgres.ReadResult, error)
}

// Handler maps safe, snapshot-bound suggestion reads to the public v1 contract.
type Handler struct {
	reader reader
}

// NewHandler constructs an opening-suggestions handler around its read port.
func NewHandler(reader reader) *Handler { return &Handler{reader: reader} }

// Register adds the read-only opening-suggestions route to a shared mux.
func (handler *Handler) Register(mux *http.ServeMux) {
	mux.HandleFunc("GET /api/v1/corpora/{corpusId}/opening-suggestions", handler.get)
}

func (handler *Handler) get(writer http.ResponseWriter, request *http.Request) {
	corpusID, err := uuid.Parse(request.PathValue("corpusId"))
	if err != nil || corpusID == uuid.Nil {
		writeInvalidRequest(writer, request)
		return
	}
	language, ok := requestLanguage(request)
	if !ok {
		writeInvalidRequest(writer, request)
		return
	}

	result, err := handler.reader.Read(request.Context(), corpusID, language)
	if err != nil {
		httpserver.WriteError(writer, request, httpserver.Problem{
			Status: http.StatusServiceUnavailable, Code: "unavailable",
			Message: "Opening suggestions are temporarily unavailable.",
		})
		return
	}
	httpserver.WriteJSON(writer, http.StatusOK, newResponse(result, language))
}

func requestLanguage(request *http.Request) (domain.QueryLanguage, bool) {
	values, present := request.URL.Query()["interfaceLanguage"]
	if !present || len(values) != 1 {
		return "", false
	}
	language := domain.QueryLanguage(values[0])
	if language != domain.QueryLanguageEnglish && language != domain.QueryLanguagePortuguese {
		return "", false
	}
	return language, true
}

func writeInvalidRequest(writer http.ResponseWriter, request *http.Request) {
	httpserver.WriteError(writer, request, httpserver.Problem{
		Status: http.StatusBadRequest, Code: "invalid_input",
		Message: "The opening suggestion request is invalid.",
	})
}

type response struct {
	CorpusID                     uuid.UUID            `json:"corpusId"`
	ActiveSnapshotID             *uuid.UUID           `json:"activeSnapshotId"`
	ActiveSnapshotManifestSHA256 *string              `json:"activeSnapshotManifestSha256"`
	InterfaceLanguage            domain.QueryLanguage `json:"interfaceLanguage"`
	Suggestions                  []suggestionResponse `json:"suggestions"`
}

type suggestionResponse struct {
	CaseID   string `json:"caseId"`
	Rank     int    `json:"rank"`
	Question string `json:"question"`
}

func newResponse(result suggestionspostgres.ReadResult, language domain.QueryLanguage) response {
	response := response{
		CorpusID: result.CorpusID, InterfaceLanguage: language,
		Suggestions: make([]suggestionResponse, 0, len(result.Suggestions)),
	}
	if result.ActiveSnapshot != nil {
		snapshotID := result.ActiveSnapshot.ID
		manifest := string(result.ActiveSnapshot.ManifestSHA256)
		response.ActiveSnapshotID = &snapshotID
		response.ActiveSnapshotManifestSHA256 = &manifest
	}
	for _, suggestion := range result.Suggestions {
		response.Suggestions = append(response.Suggestions, suggestionResponse{
			CaseID: suggestion.CaseID, Rank: suggestion.Rank, Question: suggestion.Question,
		})
	}
	return response
}
