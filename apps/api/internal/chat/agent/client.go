// Package agent adapts the internal Python LangGraph stream to the Go API boundary.
package agent

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"

	chatdomain "github.com/dharlanoliveira/norvii/apps/api/internal/chat/domain"
	"github.com/dharlanoliveira/norvii/apps/api/internal/platform/config"
	"github.com/google/uuid"
)

var ErrUnavailable = errors.New("agent service is unavailable")

// Client is the Go facade's narrow client for the Python LangGraph service.
type Client struct {
	baseURL string
	client  *http.Client
}

// NewClient constructs an internal agent client with a bounded timeout.
func NewClient(configuration config.AgentConfig) *Client {
	return &Client{
		baseURL: strings.TrimRight(configuration.BaseURL, "/"),
		client:  &http.Client{Timeout: configuration.Timeout},
	}
}

// Ask forwards one corpus-scoped request and relays model deltas to the facade.
func (client *Client) Ask(
	ctx context.Context,
	researchRequest chatdomain.Request,
	emit func(string),
) (chatdomain.Result, error) {
	body, err := json.Marshal(map[string]string{
		"question":          researchRequest.Question,
		"interfaceLanguage": researchRequest.InterfaceLanguage,
		"snapshotId":        researchRequest.SnapshotID.String(),
		"strategy":          researchRequest.Strategy,
	})
	if err != nil {
		return chatdomain.Result{}, fmt.Errorf("encode agent request: %w", err)
	}
	request, err := http.NewRequestWithContext(
		ctx,
		http.MethodPost,
		fmt.Sprintf("%s/v1/corpora/%s/chat/stream", client.baseURL, researchRequest.CorpusID),
		strings.NewReader(string(body)),
	)
	if err != nil {
		return chatdomain.Result{}, fmt.Errorf("create agent request: %w", err)
	}
	request.Header.Set("Content-Type", "application/json")
	response, err := client.client.Do(request)
	if err != nil {
		return chatdomain.Result{}, fmt.Errorf("%w: %v", ErrUnavailable, err)
	}
	defer response.Body.Close()
	if response.StatusCode < http.StatusOK || response.StatusCode >= http.StatusMultipleChoices {
		return chatdomain.Result{}, fmt.Errorf("%w: agent returned status %d", ErrUnavailable, response.StatusCode)
	}
	var result chatdomain.Result
	var terminal bool
	scanner := bufio.NewScanner(response.Body)
	scanner.Buffer(make([]byte, 1024), 256*1024)
	var eventName string
	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, "event: ") {
			eventName = strings.TrimPrefix(line, "event: ")
			continue
		}
		if !strings.HasPrefix(line, "data: ") {
			continue
		}
		if err := client.handleEvent(eventName, strings.TrimPrefix(line, "data:"), &result, &terminal, emit); err != nil {
			return chatdomain.Result{}, err
		}
	}
	if err := scanner.Err(); err != nil {
		return chatdomain.Result{}, fmt.Errorf("read agent stream: %w", err)
	}
	if !terminal {
		return chatdomain.Result{}, errors.New("agent stream ended without a terminal event")
	}
	return result, nil
}

func (client *Client) handleEvent(
	eventName string,
	data string,
	result *chatdomain.Result,
	terminal *bool,
	emit func(string),
) error {
	var event struct {
		Type       string              `json:"type"`
		Text       string              `json:"text"`
		Answer     string              `json:"answer"`
		Reason     string              `json:"reason"`
		Code       string              `json:"code"`
		References []evidenceReference `json:"references"`
		Inspection *inspectionPayload  `json:"inspection"`
	}
	if err := json.Unmarshal([]byte(strings.TrimSpace(data)), &event); err != nil {
		return fmt.Errorf("decode agent %s event: %w", eventName, err)
	}
	switch event.Type {
	case "delta":
		emit(event.Text)
	case "evidence":
		result.Evidence = evidenceValues(event.References)
	case "completed":
		result.Answer = event.Answer
		result.Evidence = evidenceValues(event.References)
		result.Inspection = inspectionValue(event.Inspection, result.Evidence, "completed")
		*terminal = true
	case "abstained":
		*terminal = true
		if event.Reason == "grounding_validation_failed" {
			return chatdomain.ErrGroundingValidation
		}
		return chatdomain.ErrInsufficientEvidence
	case "cancelled":
		*terminal = true
		return context.Canceled
	case "error":
		*terminal = true
		return fmt.Errorf("agent request failed: %s", event.Code)
	}
	return nil
}

type evidenceReference struct {
	ID                string    `json:"id"`
	CorpusID          uuid.UUID `json:"corpusId"`
	SnapshotID        uuid.UUID `json:"snapshotId"`
	SourceID          uuid.UUID `json:"sourceId"`
	DocumentID        uuid.UUID `json:"documentId"`
	DocumentVersionID uuid.UUID `json:"documentVersionId"`
	SourceRevisionID  uuid.UUID `json:"sourceRevisionId"`
	PipelineVersion   string    `json:"pipelineVersion"`
	SourceTitle       string    `json:"sourceTitle"`
	UnitLocator       string    `json:"unitLocator"`
	StartOffset       int       `json:"startOffset"`
	EndOffset         int       `json:"endOffset"`
	Excerpt           string    `json:"excerpt"`
	Rank              int       `json:"rank"`
	CosineDistance    *float64  `json:"cosineDistance"`
}

type inspectionPayload struct {
	Outcome      string               `json:"outcome"`
	Retrieval    *retrievalInspection `json:"retrieval"`
	Measurements measurementPayload   `json:"measurements"`
	Evidence     []evidenceReference  `json:"evidence"`
}

type retrievalInspection struct {
	Strategy       string  `json:"strategy"`
	TopK           int     `json:"topK"`
	ReturnedCount  int     `json:"returnedCount"`
	EmbeddingModel *string `json:"embeddingModel"`
}

type measurementPayload struct {
	RetrievalMilliseconds  *int64 `json:"retrievalMilliseconds"`
	GenerationMilliseconds *int64 `json:"generationMilliseconds"`
	TotalMilliseconds      *int64 `json:"totalMilliseconds"`
	InputTokens            *int64 `json:"inputTokens"`
	OutputTokens           *int64 `json:"outputTokens"`
}

func evidenceValues(references []evidenceReference) []chatdomain.Evidence {
	evidence := make([]chatdomain.Evidence, 0, len(references))
	for _, reference := range references {
		evidence = append(evidence, chatdomain.Evidence{
			ID: reference.ID, CorpusID: reference.CorpusID, SnapshotID: reference.SnapshotID, SourceID: reference.SourceID,
			DocumentID: reference.DocumentID, DocumentVersionID: reference.DocumentVersionID,
			SourceRevisionID: reference.SourceRevisionID, PipelineVersion: reference.PipelineVersion,
			SourceTitle: reference.SourceTitle, UnitLocator: reference.UnitLocator,
			StartOffset: reference.StartOffset, EndOffset: reference.EndOffset,
			Excerpt: reference.Excerpt, Rank: reference.Rank, CosineDistance: reference.CosineDistance,
		})
	}
	return evidence
}

func inspectionValue(payload *inspectionPayload, evidence []chatdomain.Evidence, outcome string) *chatdomain.Inspection {
	if payload == nil {
		return &chatdomain.Inspection{Outcome: outcome, Evidence: evidence}
	}
	inspection := &chatdomain.Inspection{
		Outcome: payload.Outcome, Evidence: evidence,
		Measurements: chatdomain.Measurements{
			RetrievalMilliseconds:  payload.Measurements.RetrievalMilliseconds,
			GenerationMilliseconds: payload.Measurements.GenerationMilliseconds,
			TotalMilliseconds:      payload.Measurements.TotalMilliseconds,
			InputTokens:            payload.Measurements.InputTokens,
			OutputTokens:           payload.Measurements.OutputTokens,
		},
	}
	if inspection.Outcome == "" {
		inspection.Outcome = outcome
	}
	if payload.Retrieval != nil {
		inspection.Retrieval = &chatdomain.RetrievalInspection{
			Strategy: payload.Retrieval.Strategy, TopK: payload.Retrieval.TopK,
			ReturnedCount: payload.Retrieval.ReturnedCount, EmbeddingModel: payload.Retrieval.EmbeddingModel,
		}
	}
	if len(payload.Evidence) > 0 {
		inspection.Evidence = evidenceValues(payload.Evidence)
	}
	return inspection
}
