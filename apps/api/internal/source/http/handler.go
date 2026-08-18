// Package http exposes corpus-scoped source projections through the v1 API.
package http

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"mime"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/dharlanoliveira/norvii/apps/api/internal/platform/httpserver"
	"github.com/dharlanoliveira/norvii/apps/api/internal/source/domain"
	sourcepostgres "github.com/dharlanoliveira/norvii/apps/api/internal/source/postgres"
	"github.com/google/uuid"
)

type reader interface {
	ListByCorpus(context.Context, uuid.UUID) ([]sourcepostgres.Record, error)
	Get(context.Context, uuid.UUID, uuid.UUID) (sourcepostgres.Record, error)
}

type commander interface {
	CreateURL(context.Context, uuid.UUID, string, string) (sourcepostgres.Record, error)
	CreatePDF(context.Context, uuid.UUID, string, string, string, []byte) (sourcepostgres.Record, error)
	Retry(context.Context, uuid.UUID, uuid.UUID, int) (sourcepostgres.Record, error)
	Reprocess(context.Context, uuid.UUID, uuid.UUID, int) (sourcepostgres.Record, error)
}

type originReader interface {
	GetPDFOrigin(context.Context, uuid.UUID, uuid.UUID) (sourcepostgres.PDFOriginRecord, error)
	GetURLOrigin(context.Context, uuid.UUID, uuid.UUID) (sourcepostgres.URLOriginRecord, error)
}

// Handler maps source reads to the stable v1 HTTP contract.
type Handler struct {
	reader   reader
	commands commander
	origins  originReader
}

// NewHandler constructs a source HTTP handler around its read port.
func NewHandler(reader reader, commands ...commander) *Handler {
	handler := &Handler{reader: reader}
	handler.origins, _ = reader.(originReader)
	if len(commands) > 0 {
		handler.commands = commands[0]
	}
	return handler
}

// Register adds corpus-scoped source routes to a shared application mux.
func (handler *Handler) Register(mux *http.ServeMux) {
	mux.HandleFunc("GET /api/v1/corpora/{corpusId}/sources", handler.list)
	mux.HandleFunc("GET /api/v1/corpora/{corpusId}/sources/{sourceId}", handler.get)
	if handler.commands != nil {
		mux.HandleFunc("POST /api/v1/corpora/{corpusId}/sources/url", handler.createURL)
		mux.HandleFunc("POST /api/v1/corpora/{corpusId}/sources/pdf", handler.createPDF)
		mux.HandleFunc("POST /api/v1/corpora/{corpusId}/sources/{sourceId}/retry", handler.retry)
		mux.HandleFunc("POST /api/v1/corpora/{corpusId}/sources/{sourceId}/reprocess", handler.reprocess)
	}
	if handler.origins != nil {
		mux.HandleFunc("GET /api/v1/corpora/{corpusId}/sources/{sourceId}/origin", handler.getOrigin)
		mux.HandleFunc("GET /api/v1/corpora/{corpusId}/sources/{sourceId}/origin/pdf", handler.getPDFOrigin)
		mux.HandleFunc("GET /api/v1/corpora/{corpusId}/sources/{sourceId}/origin/url", handler.getURLOrigin)
	}
}

type versionRequest struct {
	Version int `json:"version"`
}

func (handler *Handler) retry(writer http.ResponseWriter, request *http.Request) {
	handler.queueLifecycle(writer, request, handler.commands.Retry)
}

func (handler *Handler) reprocess(writer http.ResponseWriter, request *http.Request) {
	handler.queueLifecycle(writer, request, handler.commands.Reprocess)
}

func (handler *Handler) queueLifecycle(
	writer http.ResponseWriter,
	request *http.Request,
	command func(context.Context, uuid.UUID, uuid.UUID, int) (sourcepostgres.Record, error),
) {
	corpusID, sourceID, ok := pathIDs(writer, request)
	if !ok {
		return
	}
	var body versionRequest
	if err := decodeJSON(request, &body); err != nil || body.Version < 1 {
		writeSourceProblem(writer, request, http.StatusBadRequest, "invalid_input", "The lifecycle request is invalid.")
		return
	}
	record, err := command(request.Context(), corpusID, sourceID, body.Version)
	if err != nil {
		handler.writeCommandError(writer, request, err)
		return
	}
	httpserver.WriteJSON(writer, http.StatusAccepted, newSourceResponse(record))
}

type urlSourceRequest struct {
	Title string `json:"title"`
	URL   string `json:"url"`
}

func (handler *Handler) createURL(writer http.ResponseWriter, request *http.Request) {
	corpusID, err := uuid.Parse(request.PathValue("corpusId"))
	if err != nil {
		writeInvalidID(writer, request)
		return
	}
	var body urlSourceRequest
	if err := decodeJSON(request, &body); err != nil {
		writeSourceProblem(writer, request, http.StatusBadRequest, "invalid_input", "The source request is invalid.")
		return
	}
	record, err := handler.commands.CreateURL(request.Context(), corpusID, body.Title, body.URL)
	if err != nil {
		handler.writeCommandError(writer, request, err)
		return
	}
	httpserver.WriteJSON(writer, http.StatusAccepted, newSourceResponse(record))
}

func (handler *Handler) createPDF(writer http.ResponseWriter, request *http.Request) {
	corpusID, err := uuid.Parse(request.PathValue("corpusId"))
	if err != nil {
		writeInvalidID(writer, request)
		return
	}
	title, filename, mediaType, content, err := readPDFMultipart(request)
	if errors.Is(err, domain.ErrPayloadTooLarge) {
		writeSourceProblem(writer, request, http.StatusRequestEntityTooLarge, "payload_too_large", "The PDF exceeds the supported size.")
		return
	}
	if err != nil {
		writeSourceProblem(writer, request, http.StatusBadRequest, "invalid_input", "The PDF request is invalid.")
		return
	}
	record, err := handler.commands.CreatePDF(
		request.Context(), corpusID, title, filename, mediaType, content,
	)
	if err != nil {
		handler.writeCommandError(writer, request, err)
		return
	}
	httpserver.WriteJSON(writer, http.StatusAccepted, newSourceResponse(record))
}

func readPDFMultipart(request *http.Request) (string, string, string, []byte, error) {
	reader, err := request.MultipartReader()
	if err != nil {
		return "", "", "", nil, err
	}
	var title, filename, mediaType string
	var content []byte
	for {
		part, partError := reader.NextPart()
		if errors.Is(partError, io.EOF) {
			break
		}
		if partError != nil {
			return "", "", "", nil, partError
		}
		switch part.FormName() {
		case "title":
			value, readError := io.ReadAll(io.LimitReader(part, 1001))
			if readError != nil || len(value) > 1000 {
				return "", "", "", nil, domain.ErrInvalidInput
			}
			title = string(value)
		case "file":
			filename = part.FileName()
			mediaType = part.Header.Get("Content-Type")
			value, readError := io.ReadAll(io.LimitReader(part, domain.MaxOriginBytes+1))
			if readError != nil {
				return "", "", "", nil, readError
			}
			if len(value) > domain.MaxOriginBytes {
				return "", "", "", nil, domain.ErrPayloadTooLarge
			}
			content = value
		}
		_ = part.Close()
	}
	if strings.TrimSpace(title) == "" || filename == "" || len(content) == 0 {
		return "", "", "", nil, domain.ErrInvalidInput
	}
	return title, filename, mediaType, content, nil
}

func (handler *Handler) getOrigin(writer http.ResponseWriter, request *http.Request) {
	corpusID, sourceID, ok := pathIDs(writer, request)
	if !ok {
		return
	}
	origin, err := handler.origins.GetPDFOrigin(request.Context(), corpusID, sourceID)
	if errors.Is(err, sourcepostgres.ErrNotFound) {
		urlOrigin, urlError := handler.origins.GetURLOrigin(request.Context(), corpusID, sourceID)
		if errors.Is(urlError, sourcepostgres.ErrNotFound) {
			writeNotFound(writer, request)
			return
		}
		if urlError != nil {
			writeUnavailable(writer, request)
			return
		}
		http.Redirect(writer, request, urlOrigin.URL, http.StatusSeeOther)
		return
	}
	if err != nil {
		writeUnavailable(writer, request)
		return
	}
	writePDFOrigin(writer, origin)
}

func (handler *Handler) getPDFOrigin(writer http.ResponseWriter, request *http.Request) {
	corpusID, sourceID, ok := pathIDs(writer, request)
	if !ok {
		return
	}
	origin, err := handler.origins.GetPDFOrigin(request.Context(), corpusID, sourceID)
	if errors.Is(err, sourcepostgres.ErrNotFound) {
		writeNotFound(writer, request)
		return
	}
	if err != nil {
		writeUnavailable(writer, request)
		return
	}
	writePDFOrigin(writer, origin)
}

func (handler *Handler) getURLOrigin(writer http.ResponseWriter, request *http.Request) {
	corpusID, sourceID, ok := pathIDs(writer, request)
	if !ok {
		return
	}
	record, err := handler.reader.Get(request.Context(), corpusID, sourceID)
	if errors.Is(err, sourcepostgres.ErrNotFound) {
		writeNotFound(writer, request)
		return
	}
	if err != nil {
		writeUnavailable(writer, request)
		return
	}
	if record.Kind != domain.KindURL {
		writeNotFound(writer, request)
		return
	}
	httpserver.WriteJSON(writer, http.StatusOK, newOriginResponse(record))
}

func writePDFOrigin(writer http.ResponseWriter, origin sourcepostgres.PDFOriginRecord) {
	disposition := mime.FormatMediaType("attachment", map[string]string{"filename": origin.DeliveryFilename})
	writer.Header().Set("Content-Type", origin.MediaType)
	writer.Header().Set("Content-Disposition", disposition)
	writer.Header().Set("X-Content-Type-Options", "nosniff")
	writer.Header().Set("Content-Length", strconv.Itoa(len(origin.Content)))
	writer.WriteHeader(http.StatusOK)
	_, _ = writer.Write(origin.Content)
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

func (handler *Handler) writeCommandError(writer http.ResponseWriter, request *http.Request, err error) {
	switch {
	case errors.Is(err, domain.ErrInvalidInput), errors.Is(err, domain.ErrInvalidURL):
		writeSourceProblem(writer, request, http.StatusBadRequest, "invalid_input", "The source metadata is invalid.")
	case errors.Is(err, domain.ErrDuplicateSource):
		writeSourceProblem(writer, request, http.StatusConflict, "duplicate_source", "The source is already registered in this corpus.")
	case errors.Is(err, domain.ErrSourceLimit):
		writeSourceProblem(writer, request, http.StatusConflict, "payload_too_large", "This corpus reached its source limit.")
	case errors.Is(err, domain.ErrCorpusUnavailable):
		writeSourceProblem(writer, request, http.StatusNotFound, "not_found", "The corpus was not found.")
	case errors.Is(err, domain.ErrPayloadTooLarge):
		writeSourceProblem(writer, request, http.StatusRequestEntityTooLarge, "payload_too_large", "The source exceeds the supported size.")
	case errors.Is(err, domain.ErrUnsupportedContent):
		writeSourceProblem(writer, request, http.StatusBadRequest, "unsupported_content", "The source content is unsupported.")
	case errors.Is(err, sourcepostgres.ErrStaleState):
		writeSourceProblem(writer, request, http.StatusConflict, "stale_state", "The source changed; reload and retry.")
	case errors.Is(err, domain.ErrInvalidTransition):
		writeSourceProblem(writer, request, http.StatusConflict, "unavailable", "The source action is unavailable in its current state.")
	case errors.Is(err, sourcepostgres.ErrNotFound):
		writeNotFound(writer, request)
	default:
		writeUnavailable(writer, request)
	}
}

func writeSourceProblem(
	writer http.ResponseWriter,
	request *http.Request,
	status int,
	code string,
	message string,
) {
	httpserver.WriteError(writer, request, httpserver.Problem{Status: status, Code: code, Message: message})
}

func (handler *Handler) list(writer http.ResponseWriter, request *http.Request) {
	corpusID, err := uuid.Parse(request.PathValue("corpusId"))
	if err != nil {
		writeInvalidID(writer, request)
		return
	}
	records, err := handler.reader.ListByCorpus(request.Context(), corpusID)
	if err != nil {
		writeUnavailable(writer, request)
		return
	}
	responses := make([]sourceResponse, 0, len(records))
	for _, record := range records {
		responses = append(responses, newSourceResponse(record))
	}
	httpserver.WriteJSON(writer, http.StatusOK, responses)
}

func (handler *Handler) get(writer http.ResponseWriter, request *http.Request) {
	corpusID, sourceID, ok := pathIDs(writer, request)
	if !ok {
		return
	}
	record, err := handler.reader.Get(request.Context(), corpusID, sourceID)
	if errors.Is(err, sourcepostgres.ErrNotFound) {
		writeNotFound(writer, request)
		return
	}
	if err != nil {
		writeUnavailable(writer, request)
		return
	}
	httpserver.WriteJSON(writer, http.StatusOK, newSourceResponse(record))
}

type sourceResponse struct {
	ID                    uuid.UUID         `json:"id"`
	CorpusID              uuid.UUID         `json:"corpusId"`
	Title                 string            `json:"title"`
	Kind                  string            `json:"kind"`
	ProcessingStatus      string            `json:"processingStatus"`
	FailureCategory       *string           `json:"failureCategory"`
	LatestReadyDocumentID *uuid.UUID        `json:"latestReadyDocumentId"`
	Version               int               `json:"version"`
	CreatedAt             time.Time         `json:"createdAt"`
	UpdatedAt             time.Time         `json:"updatedAt"`
	Origin                originResponse    `json:"origin"`
	LatestAttempt         *attemptResponse  `json:"latestAttempt"`
	Attempts              []attemptResponse `json:"attempts"`
}

type originResponse struct {
	Kind                   string     `json:"kind"`
	SubmittedURL           *string    `json:"submittedUrl,omitempty"`
	NormalizedURL          *string    `json:"normalizedUrl,omitempty"`
	OriginalFilename       *string    `json:"originalFilename,omitempty"`
	MediaType              *string    `json:"mediaType,omitempty"`
	ByteSize               *int64     `json:"byteSize,omitempty"`
	SHA256                 *string    `json:"sha256,omitempty"`
	FinalURL               *string    `json:"finalUrl,omitempty"`
	CapturedAt             *time.Time `json:"capturedAt,omitempty"`
	ExtractedContentSHA256 *string    `json:"extractedContentSha256,omitempty"`
}

type attemptResponse struct {
	Number                   int        `json:"number"`
	PipelineVersion          string     `json:"pipelineVersion"`
	Status                   string     `json:"status"`
	StartedAt                time.Time  `json:"startedAt"`
	FinishedAt               *time.Time `json:"finishedAt"`
	FailureCategory          *string    `json:"failureCategory"`
	AcquiredByteCount        *int64     `json:"acquiredByteCount"`
	NormalizedCharacterCount *int64     `json:"normalizedCharacterCount"`
	UnitCount                *int       `json:"unitCount"`
	DurationMilliseconds     *int64     `json:"durationMilliseconds"`
}

func newSourceResponse(record sourcepostgres.Record) sourceResponse {
	response := sourceResponse{
		ID: record.ID, CorpusID: record.CorpusID, Title: record.Title,
		Kind: string(record.Kind), ProcessingStatus: string(record.ProcessingStatus),
		FailureCategory:       record.FailureCategory,
		LatestReadyDocumentID: record.LatestReadyDocumentID,
		Version:               record.Version, CreatedAt: record.CreatedAt, UpdatedAt: record.UpdatedAt,
		Origin: newOriginResponse(record), Attempts: make([]attemptResponse, 0, len(record.Attempts)),
	}
	for _, attempt := range record.Attempts {
		response.Attempts = append(response.Attempts, newAttemptResponse(attempt))
	}
	if attempt := record.LatestAttempt; attempt != nil {
		latest := newAttemptResponse(*attempt)
		response.LatestAttempt = &latest
	}
	return response
}

func newAttemptResponse(attempt sourcepostgres.Attempt) attemptResponse {
	return attemptResponse{
		Number: attempt.Number, PipelineVersion: attempt.PipelineVersion, Status: attempt.Status,
		StartedAt: attempt.StartedAt, FinishedAt: attempt.FinishedAt,
		FailureCategory: attempt.FailureCategory, AcquiredByteCount: attempt.AcquiredByteCount,
		NormalizedCharacterCount: attempt.NormalizedCharacterCount, UnitCount: attempt.UnitCount,
		DurationMilliseconds: attempt.DurationMilliseconds,
	}
}

func newOriginResponse(record sourcepostgres.Record) originResponse {
	return originResponse{
		Kind: string(record.Kind), SubmittedURL: record.Origin.SubmittedURL,
		NormalizedURL:    record.Origin.NormalizedURL,
		OriginalFilename: record.Origin.OriginalFilename, MediaType: record.Origin.MediaType,
		ByteSize: record.Origin.ByteSize, SHA256: record.Origin.SHA256,
		FinalURL: record.Origin.FinalURL, CapturedAt: record.Origin.CapturedAt,
		ExtractedContentSHA256: record.Origin.ExtractedContentSHA256,
	}
}

func pathIDs(writer http.ResponseWriter, request *http.Request) (uuid.UUID, uuid.UUID, bool) {
	corpusID, corpusError := uuid.Parse(request.PathValue("corpusId"))
	sourceID, sourceError := uuid.Parse(request.PathValue("sourceId"))
	if corpusError != nil || sourceError != nil {
		writeInvalidID(writer, request)
		return uuid.Nil, uuid.Nil, false
	}
	return corpusID, sourceID, true
}

func writeInvalidID(writer http.ResponseWriter, request *http.Request) {
	httpserver.WriteError(writer, request, httpserver.Problem{
		Status: http.StatusBadRequest, Code: "invalid_input", Message: "The source identifier is invalid.",
	})
}

func writeNotFound(writer http.ResponseWriter, request *http.Request) {
	httpserver.WriteError(writer, request, httpserver.Problem{
		Status: http.StatusNotFound, Code: "not_found", Message: "The source was not found.",
	})
}

func writeUnavailable(writer http.ResponseWriter, request *http.Request) {
	httpserver.WriteError(writer, request, httpserver.Problem{
		Status: http.StatusServiceUnavailable, Code: "unavailable",
		Message: "The source catalog is temporarily unavailable.",
	})
}
