// Package http exposes read-only corpus projections through the v1 API.
package http

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"time"

	"github.com/dharlanoliveira/norvii/apps/api/internal/catalog/domain"
	catalogpostgres "github.com/dharlanoliveira/norvii/apps/api/internal/catalog/postgres"
	"github.com/dharlanoliveira/norvii/apps/api/internal/platform/httpserver"
	"github.com/google/uuid"
)

type reader interface {
	List(context.Context, bool) ([]catalogpostgres.Summary, error)
	Get(context.Context, uuid.UUID, bool) (catalogpostgres.Summary, error)
}

type commander interface {
	Create(context.Context, domain.Draft) (catalogpostgres.Summary, error)
	Update(context.Context, uuid.UUID, domain.Draft, int) (catalogpostgres.Summary, error)
	Disable(context.Context, uuid.UUID, int) (catalogpostgres.Summary, error)
	Enable(context.Context, uuid.UUID, int) (catalogpostgres.Summary, error)
}

// Handler maps corpus reads to the stable v1 HTTP contract.
type Handler struct {
	reader   reader
	commands commander
}

// NewHandler constructs a corpus HTTP handler around its read port.
func NewHandler(reader reader, commands ...commander) *Handler {
	handler := &Handler{reader: reader}
	if len(commands) > 0 {
		handler.commands = commands[0]
	}
	return handler
}

// Register adds read-only corpus routes to a shared application mux.
func (handler *Handler) Register(mux *http.ServeMux) {
	mux.HandleFunc("GET /api/v1/corpora", handler.list)
	mux.HandleFunc("GET /api/v1/corpora/{corpusId}", handler.get)
	if handler.commands != nil {
		mux.HandleFunc("POST /api/v1/corpora", handler.create)
		mux.HandleFunc("PATCH /api/v1/corpora/{corpusId}", handler.update)
		mux.HandleFunc("POST /api/v1/corpora/{corpusId}/disable", handler.disable)
		mux.HandleFunc("POST /api/v1/corpora/{corpusId}/enable", handler.enable)
	}
}

type corpusWriteRequest struct {
	Name         string          `json:"name"`
	Description  string          `json:"description"`
	Language     domain.Language `json:"language"`
	Jurisdiction string          `json:"jurisdiction"`
	Version      int             `json:"version,omitempty"`
}

type versionRequest struct {
	Version int `json:"version"`
}

func (request corpusWriteRequest) draft() domain.Draft {
	return domain.Draft{
		Name: request.Name, Description: request.Description,
		Language: request.Language, Jurisdiction: request.Jurisdiction,
	}
}

func (handler *Handler) create(writer http.ResponseWriter, request *http.Request) {
	var body corpusWriteRequest
	if err := decodeJSON(request, &body); err != nil {
		writeInvalidBody(writer, request)
		return
	}
	created, err := handler.commands.Create(request.Context(), body.draft())
	if err != nil {
		writeCommandError(writer, request, err)
		return
	}
	httpserver.WriteJSON(writer, http.StatusCreated, newCorpusResponse(created))
}

func (handler *Handler) update(writer http.ResponseWriter, request *http.Request) {
	id, ok := corpusID(writer, request)
	if !ok {
		return
	}
	var body corpusWriteRequest
	if err := decodeJSON(request, &body); err != nil || body.Version < 1 {
		writeInvalidBody(writer, request)
		return
	}
	updated, err := handler.commands.Update(
		request.Context(), id, body.draft(), body.Version,
	)
	if err != nil {
		writeCommandError(writer, request, err)
		return
	}
	httpserver.WriteJSON(writer, http.StatusOK, newCorpusResponse(updated))
}

func (handler *Handler) disable(writer http.ResponseWriter, request *http.Request) {
	handler.setStatus(writer, request, handler.commands.Disable)
}

func (handler *Handler) enable(writer http.ResponseWriter, request *http.Request) {
	handler.setStatus(writer, request, handler.commands.Enable)
}

func (handler *Handler) setStatus(
	writer http.ResponseWriter,
	request *http.Request,
	command func(context.Context, uuid.UUID, int) (catalogpostgres.Summary, error),
) {
	id, ok := corpusID(writer, request)
	if !ok {
		return
	}
	var body versionRequest
	if err := decodeJSON(request, &body); err != nil || body.Version < 1 {
		writeInvalidBody(writer, request)
		return
	}
	updated, err := command(request.Context(), id, body.Version)
	if err != nil {
		writeCommandError(writer, request, err)
		return
	}
	httpserver.WriteJSON(writer, http.StatusOK, newCorpusResponse(updated))
}

func corpusID(writer http.ResponseWriter, request *http.Request) (uuid.UUID, bool) {
	id, err := uuid.Parse(request.PathValue("corpusId"))
	if err != nil {
		writeInvalidID(writer, request)
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
		return errors.New("request must contain one JSON value")
	}
	return nil
}

func (handler *Handler) list(writer http.ResponseWriter, request *http.Request) {
	includeDisabled := request.URL.Query().Get("includeDisabled") == "true"
	corpora, err := handler.reader.List(request.Context(), includeDisabled)
	if err != nil {
		writeUnavailable(writer, request)
		return
	}
	responses := make([]corpusResponse, 0, len(corpora))
	for _, corpus := range corpora {
		responses = append(responses, newCorpusResponse(corpus))
	}
	httpserver.WriteJSON(writer, http.StatusOK, responses)
}

func (handler *Handler) get(writer http.ResponseWriter, request *http.Request) {
	id, err := uuid.Parse(request.PathValue("corpusId"))
	if err != nil {
		writeInvalidID(writer, request)
		return
	}
	includeDisabled := request.URL.Query().Get("includeDisabled") == "true"
	corpus, err := handler.reader.Get(request.Context(), id, includeDisabled)
	if errors.Is(err, catalogpostgres.ErrNotFound) {
		writeNotFound(writer, request)
		return
	}
	if err != nil {
		writeUnavailable(writer, request)
		return
	}
	httpserver.WriteJSON(writer, http.StatusOK, newCorpusResponse(corpus))
}

type corpusResponse struct {
	ID           uuid.UUID `json:"id"`
	Name         string    `json:"name"`
	Description  string    `json:"description"`
	Language     string    `json:"language"`
	Jurisdiction string    `json:"jurisdiction"`
	Status       string    `json:"status"`
	SourceCount  int       `json:"sourceCount"`
	Version      int       `json:"version"`
	CreatedAt    time.Time `json:"createdAt"`
	UpdatedAt    time.Time `json:"updatedAt"`
}

func newCorpusResponse(corpus catalogpostgres.Summary) corpusResponse {
	return corpusResponse{
		ID: corpus.ID, Name: corpus.Name, Description: corpus.Description,
		Language: string(corpus.Language), Jurisdiction: corpus.Jurisdiction,
		Status: string(corpus.Status), SourceCount: corpus.SourceCount, Version: corpus.Version,
		CreatedAt: corpus.CreatedAt, UpdatedAt: corpus.UpdatedAt,
	}
}

func writeInvalidID(writer http.ResponseWriter, request *http.Request) {
	httpserver.WriteError(writer, request, httpserver.Problem{
		Status: http.StatusBadRequest, Code: "invalid_input", Message: "The corpus identifier is invalid.",
	})
}

func writeInvalidBody(writer http.ResponseWriter, request *http.Request) {
	httpserver.WriteError(writer, request, httpserver.Problem{
		Status: http.StatusBadRequest, Code: "invalid_input",
		Message: "The corpus request is invalid.",
	})
}

func writeCommandError(writer http.ResponseWriter, request *http.Request, err error) {
	var domainError *domain.Error
	switch {
	case errors.As(err, &domainError):
		httpserver.WriteError(writer, request, httpserver.Problem{
			Status: http.StatusBadRequest, Code: string(domainError.Code),
			Message: domainError.Message, Fields: domainError.Fields,
		})
	case errors.Is(err, catalogpostgres.ErrStaleState):
		httpserver.WriteError(writer, request, httpserver.Problem{
			Status: http.StatusConflict, Code: "stale_state",
			Message: "The corpus changed; reload and retry.",
		})
	case errors.Is(err, catalogpostgres.ErrNotFound):
		writeNotFound(writer, request)
	default:
		writeUnavailable(writer, request)
	}
}

func writeNotFound(writer http.ResponseWriter, request *http.Request) {
	httpserver.WriteError(writer, request, httpserver.Problem{
		Status: http.StatusNotFound, Code: "not_found", Message: "The corpus was not found.",
	})
}

func writeUnavailable(writer http.ResponseWriter, request *http.Request) {
	httpserver.WriteError(writer, request, httpserver.Problem{
		Status: http.StatusServiceUnavailable, Code: "unavailable",
		Message: "The corpus catalog is temporarily unavailable.",
	})
}
