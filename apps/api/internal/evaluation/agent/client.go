// Package agent adapts fixed-snapshot evaluation execution to the Python agent port.
package agent

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"mime"
	"net/http"
	"strings"

	"github.com/dharlanoliveira/norvii/apps/api/internal/evaluation/application"
	"github.com/dharlanoliveira/norvii/apps/api/internal/evaluation/domain"
	"github.com/dharlanoliveira/norvii/apps/api/internal/platform/config"
	"github.com/google/uuid"
)

var (
	// ErrUnavailable identifies an unavailable Python evaluation port without exposing its body.
	ErrUnavailable = errors.New("evaluation agent service is unavailable")
	// ErrInvalidRequest identifies a request that cannot safely cross the evaluation boundary.
	ErrInvalidRequest = errors.New("evaluation agent request is invalid")
	// ErrInvalidResponse identifies a response that cannot safely reach scoring or persistence.
	ErrInvalidResponse = errors.New("evaluation agent response is invalid")
	// ErrFrozenIdentityUnavailable identifies an agent that cannot serve a run's persisted identity.
	ErrFrozenIdentityUnavailable = errors.New("evaluation agent cannot serve the frozen run identity")
)

const (
	evaluationPath       = "/v1/evaluations/execute"
	maxResponseBodyBytes = 256 * 1024
	strategyVector       = "vector"
	strategyHybrid       = "hybrid"
	graphNotRequested    = "not_requested"
	graphNotUsed         = "not_used"
	graphGrounded        = "grounded"
	outcomeCompleted     = "completed"
	outcomeAbstained     = "abstained"
	interfaceLanguageEN  = "en"
	interfaceLanguagePT  = "pt"
)

// FrozenRetrievalConfiguration is the immutable retrieval selection persisted with an evaluation
// run. It is deliberately separate from public chat request settings.
type FrozenRetrievalConfiguration struct {
	Strategy    string
	Fingerprint string
}

// Request is one explicit evaluation case against an immutable corpus snapshot.
type Request struct {
	CorpusID               uuid.UUID
	SnapshotID             uuid.UUID
	Question               string
	InterfaceLanguage      string
	RetrievalConfiguration FrozenRetrievalConfiguration
	ExecutionIdentity      application.ExecutionIdentity
}

// CitationMarkerInput preserves the agent's one-based mapping between marker positions and the
// retrieval sequence. The scorer still parses actual answer markers independently.
type CitationMarkerInput struct {
	MarkerPosition int
	EvidenceRank   int
}

// GraphGrounding reports the content-free graph contribution state for one evaluation result.
type GraphGrounding struct {
	Status string
}

// Telemetry contains nullable measurements supplied by the evaluation boundary.
type Telemetry struct {
	RetrievalMilliseconds  *int64
	GenerationMilliseconds *int64
	TotalMilliseconds      *int64
	InputTokens            *int64
	OutputTokens           *int64
}

// Result is a validated evaluation response ready for deterministic scoring or worker handling.
type Result struct {
	Materialized           application.MaterializedCaseResult
	CitationMarkers        []CitationMarkerInput
	GraphGrounding         GraphGrounding
	ModelIdentity          string
	AgentBuildIdentity     string
	EmbeddingModelIdentity string
	Telemetry              Telemetry
}

// Client is the narrow non-streaming HTTP client for the Python evaluation port.
type Client struct {
	baseURL string
	client  *http.Client
}

// NewClient constructs a bounded evaluation client from the shared internal agent settings.
func NewClient(configuration config.AgentConfig) *Client {
	return &Client{
		baseURL: strings.TrimRight(configuration.BaseURL, "/"),
		client:  &http.Client{Timeout: configuration.Timeout},
	}
}

// Execute sends the caller-selected corpus and snapshot directly to the non-streaming evaluation
// port. It never resolves an active release or invokes public chat orchestration.
func (client *Client) Execute(ctx context.Context, request Request) (Result, error) {
	if client == nil || client.client == nil || strings.TrimSpace(client.baseURL) == "" {
		return Result{}, ErrInvalidRequest
	}
	if err := validateRequest(request); err != nil {
		return Result{}, err
	}
	payload, err := json.Marshal(evaluationRequestPayload{
		CorpusID:          request.CorpusID,
		SnapshotID:        request.SnapshotID,
		Question:          request.Question,
		InterfaceLanguage: request.InterfaceLanguage,
		RetrievalConfiguration: retrievalConfigurationPayload{
			Strategy: request.RetrievalConfiguration.Strategy, Fingerprint: request.RetrievalConfiguration.Fingerprint,
		},
		ExecutionIdentity: executionIdentityPayload{
			AgentBuild: request.ExecutionIdentity.AgentBuild, ChatModelIdentity: request.ExecutionIdentity.ChatModelIdentity,
			EmbeddingModelIdentity: request.ExecutionIdentity.EmbeddingModelIdentity,
		},
	})
	if err != nil {
		return Result{}, fmt.Errorf("encode evaluation agent request: %w", err)
	}

	httpRequest, err := http.NewRequestWithContext(
		ctx, http.MethodPost, client.baseURL+evaluationPath, bytes.NewReader(payload),
	)
	if err != nil {
		return Result{}, fmt.Errorf("create evaluation agent request: %w", err)
	}
	httpRequest.Header.Set("Accept", "application/json")
	httpRequest.Header.Set("Content-Type", "application/json")
	response, err := client.client.Do(httpRequest)
	if err != nil {
		return Result{}, fmt.Errorf("%w: %v", ErrUnavailable, err)
	}
	defer response.Body.Close()
	if response.StatusCode == http.StatusConflict && jsonContentType(response.Header.Get("Content-Type")) {
		return Result{}, frozenIdentityError(response.Body)
	}
	if response.StatusCode < http.StatusOK || response.StatusCode >= http.StatusMultipleChoices {
		return Result{}, fmt.Errorf("%w: agent returned status %d", ErrUnavailable, response.StatusCode)
	}
	if !jsonContentType(response.Header.Get("Content-Type")) {
		return Result{}, ErrInvalidResponse
	}

	contents, err := io.ReadAll(io.LimitReader(response.Body, maxResponseBodyBytes+1))
	if err != nil || len(contents) > maxResponseBodyBytes {
		return Result{}, ErrInvalidResponse
	}
	var payloadResponse evaluationResponsePayload
	decoder := json.NewDecoder(bytes.NewReader(contents))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&payloadResponse); err != nil {
		return Result{}, fmt.Errorf("%w: decode response", ErrInvalidResponse)
	}
	if err := decoder.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
		return Result{}, ErrInvalidResponse
	}
	return validateResponse(request, payloadResponse)
}

type evaluationRequestPayload struct {
	CorpusID               uuid.UUID                     `json:"corpusId"`
	SnapshotID             uuid.UUID                     `json:"snapshotId"`
	Question               string                        `json:"question"`
	InterfaceLanguage      string                        `json:"interfaceLanguage"`
	RetrievalConfiguration retrievalConfigurationPayload `json:"retrievalConfiguration"`
	ExecutionIdentity      executionIdentityPayload      `json:"executionIdentity"`
}

type retrievalConfigurationPayload struct {
	Strategy    string `json:"strategy"`
	Fingerprint string `json:"fingerprint"`
}

type executionIdentityPayload struct {
	AgentBuild             string `json:"agentBuild"`
	ChatModelIdentity      string `json:"chatModelIdentity"`
	EmbeddingModelIdentity string `json:"embeddingModelIdentity"`
}

type evaluationResponsePayload struct {
	Answer                 string                  `json:"answer"`
	Outcome                string                  `json:"outcome"`
	RetrievedEvidence      []evidencePayload       `json:"retrievedEvidence"`
	CitationMarkerInputs   []citationMarkerPayload `json:"citationMarkerInputs"`
	GraphGrounding         graphGroundingPayload   `json:"graphGrounding"`
	ModelIdentity          string                  `json:"modelIdentity"`
	AgentBuildIdentity     string                  `json:"agentBuildIdentity"`
	EmbeddingModelIdentity string                  `json:"embeddingModelIdentity"`
	Telemetry              telemetryPayload        `json:"telemetry"`
}

type evidencePayload struct {
	Rank             int           `json:"rank"`
	CorpusID         uuid.UUID     `json:"corpusId"`
	SnapshotID       uuid.UUID     `json:"snapshotId"`
	SourceID         uuid.UUID     `json:"sourceId"`
	SourceRevisionID uuid.UUID     `json:"sourceRevisionId"`
	DocumentID       uuid.UUID     `json:"documentId"`
	UnitID           uuid.UUID     `json:"unitId"`
	CanonicalLocator string        `json:"canonicalLocator"`
	StartOffset      int           `json:"startOffset"`
	EndOffset        int           `json:"endOffset"`
	ContentSHA256    domain.SHA256 `json:"contentSha256"`
}

type citationMarkerPayload struct {
	MarkerPosition int `json:"markerPosition"`
	EvidenceRank   int `json:"evidenceRank"`
}

type graphGroundingPayload struct {
	Status string `json:"status"`
}

type telemetryPayload struct {
	RetrievalMilliseconds  *int64 `json:"retrievalMilliseconds"`
	GenerationMilliseconds *int64 `json:"generationMilliseconds"`
	TotalMilliseconds      *int64 `json:"totalMilliseconds"`
	InputTokens            *int64 `json:"inputTokens"`
	OutputTokens           *int64 `json:"outputTokens"`
}

func validateRequest(request Request) error {
	if request.CorpusID == uuid.Nil || request.SnapshotID == uuid.Nil || strings.TrimSpace(request.Question) == "" ||
		(request.InterfaceLanguage != interfaceLanguageEN && request.InterfaceLanguage != interfaceLanguagePT) ||
		(request.RetrievalConfiguration.Strategy != strategyVector && request.RetrievalConfiguration.Strategy != strategyHybrid) ||
		strings.TrimSpace(request.RetrievalConfiguration.Fingerprint) == "" || strings.TrimSpace(request.ExecutionIdentity.AgentBuild) == "" ||
		strings.TrimSpace(request.ExecutionIdentity.ChatModelIdentity) == "" || strings.TrimSpace(request.ExecutionIdentity.EmbeddingModelIdentity) == "" {
		return ErrInvalidRequest
	}
	return nil
}

func validateResponse(request Request, response evaluationResponsePayload) (Result, error) {
	if strings.TrimSpace(response.ModelIdentity) == "" || strings.TrimSpace(response.AgentBuildIdentity) == "" || strings.TrimSpace(response.EmbeddingModelIdentity) == "" ||
		!validGraphGrounding(request.RetrievalConfiguration.Strategy, response.GraphGrounding.Status) ||
		!validTelemetry(response.Telemetry) || !validEvidenceOrdering(response.RetrievedEvidence) {
		return Result{}, ErrInvalidResponse
	}

	caseResult := application.AgentCaseResult{
		Answer: response.Answer, Retrieved: retrievedEvidence(response.RetrievedEvidence),
		LatencyMilliseconds: cloneMeasurement(response.Telemetry.TotalMilliseconds),
		InputTokens:         cloneMeasurement(response.Telemetry.InputTokens),
		OutputTokens:        cloneMeasurement(response.Telemetry.OutputTokens),
	}
	switch response.Outcome {
	case outcomeCompleted:
		caseResult.State, caseResult.Outcome = application.CaseExecutionCompleted, application.AgentOutcomeCompleted
	case outcomeAbstained:
		caseResult.State, caseResult.Outcome = application.CaseExecutionAbstained, application.AgentOutcomeAbstained
	default:
		return Result{}, ErrInvalidResponse
	}
	if !validCitationMarkerInputs(response.CitationMarkerInputs, len(response.RetrievedEvidence)) {
		return Result{}, ErrInvalidResponse
	}
	materialized, err := application.MaterializeCaseResult(request.CorpusID, request.SnapshotID, caseResult)
	if err != nil {
		return Result{}, fmt.Errorf("%w: validate evaluation result", ErrInvalidResponse)
	}
	return Result{
		Materialized: materialized, CitationMarkers: citationMarkerInputs(response.CitationMarkerInputs),
		GraphGrounding: GraphGrounding{Status: response.GraphGrounding.Status}, ModelIdentity: response.ModelIdentity,
		AgentBuildIdentity: response.AgentBuildIdentity, EmbeddingModelIdentity: response.EmbeddingModelIdentity, Telemetry: Telemetry{
			RetrievalMilliseconds:  cloneMeasurement(response.Telemetry.RetrievalMilliseconds),
			GenerationMilliseconds: cloneMeasurement(response.Telemetry.GenerationMilliseconds),
			TotalMilliseconds:      cloneMeasurement(response.Telemetry.TotalMilliseconds),
			InputTokens:            cloneMeasurement(response.Telemetry.InputTokens),
			OutputTokens:           cloneMeasurement(response.Telemetry.OutputTokens),
		},
	}, nil
}

func frozenIdentityError(body io.Reader) error {
	contents, err := io.ReadAll(io.LimitReader(body, maxResponseBodyBytes+1))
	if err != nil || len(contents) > maxResponseBodyBytes {
		return ErrUnavailable
	}
	var response struct {
		Code string `json:"code"`
	}
	decoder := json.NewDecoder(bytes.NewReader(contents))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&response); err != nil || response.Code != "frozen_identity_unavailable" {
		return ErrUnavailable
	}
	if err := decoder.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
		return ErrUnavailable
	}
	return ErrFrozenIdentityUnavailable
}

func retrievedEvidence(evidence []evidencePayload) []application.RetrievedEvidence {
	retrieved := make([]application.RetrievedEvidence, 0, len(evidence))
	for _, item := range evidence {
		retrieved = append(retrieved, application.RetrievedEvidence{Provenance: application.EvidenceProvenance{
			CorpusID: item.CorpusID, SnapshotID: item.SnapshotID, SourceID: item.SourceID,
			SourceRevisionID: item.SourceRevisionID, DocumentID: item.DocumentID, UnitID: item.UnitID,
			CanonicalLocator: item.CanonicalLocator, StartOffset: item.StartOffset, EndOffset: item.EndOffset,
			ContentSHA256: item.ContentSHA256,
		}})
	}
	return retrieved
}

func validEvidenceOrdering(evidence []evidencePayload) bool {
	for index, item := range evidence {
		if item.Rank != index+1 {
			return false
		}
	}
	return true
}

func validCitationMarkerInputs(inputs []citationMarkerPayload, evidenceCount int) bool {
	if len(inputs) != evidenceCount {
		return false
	}
	for index, input := range inputs {
		if input.MarkerPosition != index+1 || input.EvidenceRank != index+1 {
			return false
		}
	}
	return true
}

func citationMarkerInputs(inputs []citationMarkerPayload) []CitationMarkerInput {
	result := make([]CitationMarkerInput, 0, len(inputs))
	for _, input := range inputs {
		result = append(result, CitationMarkerInput{MarkerPosition: input.MarkerPosition, EvidenceRank: input.EvidenceRank})
	}
	return result
}

func validGraphGrounding(strategy, status string) bool {
	if strategy == strategyVector {
		return status == graphNotRequested
	}
	return status == graphNotUsed || status == graphGrounded
}

func validTelemetry(telemetry telemetryPayload) bool {
	for _, measurement := range []*int64{
		telemetry.RetrievalMilliseconds, telemetry.GenerationMilliseconds, telemetry.TotalMilliseconds,
		telemetry.InputTokens, telemetry.OutputTokens,
	} {
		if measurement != nil && *measurement < 0 {
			return false
		}
	}
	return true
}

func cloneMeasurement(value *int64) *int64 {
	if value == nil {
		return nil
	}
	copy := *value
	return &copy
}

func jsonContentType(value string) bool {
	mediaType, _, err := mime.ParseMediaType(value)
	return err == nil && mediaType == "application/json"
}
