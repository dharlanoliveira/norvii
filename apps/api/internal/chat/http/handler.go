// Package http exposes the grounded chat stream contract.
package http

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	chatdomain "github.com/dharlanoliveira/norvii/apps/api/internal/chat/domain"
	"github.com/dharlanoliveira/norvii/apps/api/internal/platform/httpserver"
	"github.com/google/uuid"
)

type asker interface {
	Ask(context.Context, chatdomain.Request, func(string)) (chatdomain.Result, error)
}

// Handler maps grounded chat requests to the versioned SSE stream.
type Handler struct {
	service asker
}

// NewHandler constructs a grounded chat handler around an application service.
func NewHandler(service asker) *Handler { return &Handler{service: service} }

// Register adds the active-corpus stream route to a shared mux.
func (handler *Handler) Register(mux *http.ServeMux) {
	mux.HandleFunc("POST /api/v1/corpora/{corpusId}/chat/stream", handler.stream)
}

type requestPayload struct {
	Question          string `json:"question"`
	InterfaceLanguage string `json:"interfaceLanguage"`
	Strategy          string `json:"strategy"`
}

func (handler *Handler) stream(writer http.ResponseWriter, request *http.Request) {
	corpusID, err := uuid.Parse(request.PathValue("corpusId"))
	if err != nil {
		httpserver.WriteError(writer, request, httpserver.Problem{
			Status: http.StatusBadRequest, Code: "invalid_input", Message: "The chat corpus is invalid.",
		})
		return
	}
	var payload requestPayload
	decoder := json.NewDecoder(http.MaxBytesReader(writer, request.Body, 64*1024))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&payload); err != nil || strings.TrimSpace(payload.Question) == "" {
		httpserver.WriteError(writer, request, httpserver.Problem{
			Status: http.StatusBadRequest, Code: "invalid_question", Message: "The research question is invalid.",
		})
		return
	}
	if payload.InterfaceLanguage == "" {
		payload.InterfaceLanguage = "en"
	}
	if payload.InterfaceLanguage != "en" && payload.InterfaceLanguage != "pt" {
		httpserver.WriteError(writer, request, httpserver.Problem{
			Status: http.StatusBadRequest, Code: "invalid_input", Message: "The chat language is invalid.",
		})
		return
	}
	if payload.Strategy == "" {
		payload.Strategy = "vector"
	}
	if payload.Strategy != "vector" && payload.Strategy != "graph" && payload.Strategy != "hybrid" {
		httpserver.WriteError(writer, request, httpserver.Problem{
			Status: http.StatusBadRequest, Code: "invalid_input", Message: "The retrieval strategy is invalid.",
		})
		return
	}

	requestID := uuid.New()
	startedAt := time.Now()
	writer.Header().Set("Content-Type", "text/event-stream")
	writer.Header().Set("Cache-Control", "no-cache")
	writer.Header().Set("Connection", "keep-alive")
	if err := writeEvent(writer, "started", map[string]any{
		"type": "started", "requestId": requestID, "corpusId": corpusID,
	}); err != nil {
		return
	}
	flusher, canFlush := writer.(http.Flusher)
	if canFlush {
		flusher.Flush()
	}
	var deltas []string
	result, err := handler.service.Ask(request.Context(), chatdomain.Request{
		CorpusID: corpusID, Question: payload.Question, InterfaceLanguage: payload.InterfaceLanguage, Strategy: payload.Strategy,
	}, func(delta string) { deltas = append(deltas, delta) })
	if err != nil {
		handler.writeTerminalError(writer, requestID, err, elapsedMilliseconds(startedAt))
		return
	}
	if err := writeEvent(writer, "evidence", map[string]any{
		"type": "evidence", "requestId": requestID, "references": evidenceResponses(result.Evidence),
	}); err != nil {
		return
	}
	for _, delta := range deltas {
		if err := writeEvent(writer, "delta", map[string]any{
			"type": "delta", "requestId": requestID, "text": delta,
		}); err != nil {
			return
		}
	}
	durationMilliseconds := elapsedMilliseconds(startedAt)
	inspection := inspectionResponse(result.Inspection, result.Evidence, durationMilliseconds, "completed")
	_ = writeEvent(writer, "completed", map[string]any{
		"type": "completed", "requestId": requestID,
		"answer": result.Answer, "references": evidenceResponses(result.Evidence),
		"telemetry":  map[string]any{"outcome": "completed", "evidenceCount": len(result.Evidence), "durationMilliseconds": durationMilliseconds},
		"inspection": inspection,
	})
}

func (handler *Handler) writeTerminalError(writer http.ResponseWriter, requestID uuid.UUID, err error, durationMilliseconds int64) {
	if errors.Is(err, context.Canceled) {
		_ = writeEvent(writer, "cancelled", map[string]any{
			"type": "cancelled", "requestId": requestID,
			"telemetry":  map[string]any{"outcome": "cancelled", "evidenceCount": 0, "durationMilliseconds": durationMilliseconds},
			"inspection": inspectionResponse(nil, nil, durationMilliseconds, "cancelled"),
		})
		return
	}
	if errors.Is(err, chatdomain.ErrInsufficientEvidence) || errors.Is(err, chatdomain.ErrGroundingValidation) {
		_ = writeEvent(writer, "abstained", map[string]any{
			"type": "abstained", "requestId": requestID,
			"reason":     "insufficient_evidence",
			"telemetry":  map[string]any{"outcome": "abstained", "evidenceCount": 0, "durationMilliseconds": durationMilliseconds},
			"inspection": inspectionResponse(nil, nil, durationMilliseconds, "abstained"),
		})
		return
	}
	code := "generation_failed"
	if errors.Is(err, chatdomain.ErrRetrievalFailed) {
		code = "retrieval_failed"
	}
	if errors.Is(err, chatdomain.ErrInvalidQuestion) {
		code = "invalid_question"
	}
	if errors.Is(err, chatdomain.ErrSnapshotUnavailable) {
		code = "snapshot_unavailable"
	}
	if errors.Is(err, chatdomain.ErrGraphUnavailable) {
		code = "graph_unavailable"
	}
	_ = writeEvent(writer, "error", map[string]any{
		"type": "error", "requestId": requestID, "code": code,
		"message":    "The grounded chat request could not be completed.",
		"telemetry":  map[string]any{"outcome": "failed", "evidenceCount": 0, "durationMilliseconds": durationMilliseconds},
		"inspection": inspectionResponse(nil, nil, durationMilliseconds, "failed"),
	})
}

type evidenceResponse struct {
	ID                string    `json:"id"`
	CorpusID          uuid.UUID `json:"corpusId"`
	SnapshotID        uuid.UUID `json:"snapshotId"`
	SourceID          uuid.UUID `json:"sourceId"`
	DocumentID        uuid.UUID `json:"documentId"`
	DocumentVersionID uuid.UUID `json:"documentVersionId,omitempty"`
	SourceRevisionID  uuid.UUID `json:"sourceRevisionId,omitempty"`
	PipelineVersion   string    `json:"pipelineVersion,omitempty"`
	SourceTitle       string    `json:"sourceTitle,omitempty"`
	UnitLocator       string    `json:"unitLocator"`
	StartOffset       int       `json:"startOffset"`
	EndOffset         int       `json:"endOffset"`
	Excerpt           string    `json:"excerpt"`
	Rank              int       `json:"rank"`
	CosineDistance    *float64  `json:"cosineDistance"`
}

func evidenceResponses(evidence []chatdomain.Evidence) []evidenceResponse {
	responses := make([]evidenceResponse, 0, len(evidence))
	for _, item := range evidence {
		responses = append(responses, evidenceResponse{
			ID: item.ID, CorpusID: item.CorpusID, SnapshotID: item.SnapshotID, SourceID: item.SourceID,
			DocumentID: item.DocumentID, DocumentVersionID: item.DocumentVersionID,
			SourceRevisionID: item.SourceRevisionID, PipelineVersion: item.PipelineVersion,
			SourceTitle: item.SourceTitle, UnitLocator: item.UnitLocator,
			StartOffset: item.StartOffset, EndOffset: item.EndOffset,
			Excerpt: item.Excerpt, Rank: item.Rank, CosineDistance: item.CosineDistance,
		})
	}
	return responses
}

type inspectionEventResponse struct {
	Outcome      string              `json:"outcome"`
	Retrieval    *retrievalResponse  `json:"retrieval,omitempty"`
	Measurements measurementResponse `json:"measurements"`
	Evidence     []evidenceResponse  `json:"evidence,omitempty"`
}

type retrievalResponse struct {
	Strategy       string  `json:"strategy"`
	TopK           int     `json:"topK"`
	ReturnedCount  int     `json:"returnedCount"`
	EmbeddingModel *string `json:"embeddingModel"`
}

type measurementResponse struct {
	RetrievalMilliseconds  *int64 `json:"retrievalMilliseconds"`
	GenerationMilliseconds *int64 `json:"generationMilliseconds"`
	TotalMilliseconds      *int64 `json:"totalMilliseconds"`
	InputTokens            *int64 `json:"inputTokens"`
	OutputTokens           *int64 `json:"outputTokens"`
}

func inspectionResponse(inspection *chatdomain.Inspection, evidence []chatdomain.Evidence, total int64, outcome string) inspectionEventResponse {
	if inspection == nil {
		inspection = &chatdomain.Inspection{Outcome: outcome, Evidence: evidence}
	}
	inspectionCopy := *inspection
	inspectionCopy.Outcome = outcome
	inspectionCopy.Measurements.TotalMilliseconds = &total
	response := inspectionEventResponse{
		Outcome: inspectionCopy.Outcome,
		Measurements: measurementResponse{
			RetrievalMilliseconds:  inspectionCopy.Measurements.RetrievalMilliseconds,
			GenerationMilliseconds: inspectionCopy.Measurements.GenerationMilliseconds,
			TotalMilliseconds:      inspectionCopy.Measurements.TotalMilliseconds,
			InputTokens:            inspectionCopy.Measurements.InputTokens,
			OutputTokens:           inspectionCopy.Measurements.OutputTokens,
		},
	}
	if inspectionCopy.Retrieval != nil {
		response.Retrieval = &retrievalResponse{
			Strategy: inspectionCopy.Retrieval.Strategy, TopK: inspectionCopy.Retrieval.TopK,
			ReturnedCount: inspectionCopy.Retrieval.ReturnedCount, EmbeddingModel: inspectionCopy.Retrieval.EmbeddingModel,
		}
	}
	if outcome == "completed" {
		response.Evidence = evidenceResponses(inspectionCopy.Evidence)
	}
	return response
}

func elapsedMilliseconds(startedAt time.Time) int64 {
	return time.Since(startedAt).Milliseconds()
}

func writeEvent(writer http.ResponseWriter, name string, value any) error {
	payload, err := json.Marshal(value)
	if err != nil {
		return fmt.Errorf("encode %s chat event: %w", name, err)
	}
	if _, err := fmt.Fprintf(writer, "event: %s\ndata: %s\n\n", name, payload); err != nil {
		return fmt.Errorf("write %s chat event: %w", name, err)
	}
	if flusher, ok := writer.(http.Flusher); ok {
		flusher.Flush()
	}
	return nil
}
