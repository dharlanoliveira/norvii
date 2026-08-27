package agent

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/dharlanoliveira/norvii/apps/api/internal/evaluation/application"
	"github.com/dharlanoliveira/norvii/apps/api/internal/platform/config"
	"github.com/google/uuid"
)

func TestClientExecutesFixedSnapshotEvaluationContract(t *testing.T) {
	t.Parallel()

	request := evaluationRequest()
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, received *http.Request) {
		if received.Method != http.MethodPost || received.URL.Path != evaluationPath {
			t.Fatalf("request = %s %s, want POST %s", received.Method, received.URL.Path, evaluationPath)
		}
		if received.Header.Get("Accept") != "application/json" || received.Header.Get("Content-Type") != "application/json" {
			t.Fatalf("headers = %#v, want non-streaming JSON headers", received.Header)
		}
		var payload evaluationRequestPayload
		if err := json.NewDecoder(received.Body).Decode(&payload); err != nil {
			t.Fatalf("decode request: %v", err)
		}
		if payload.CorpusID != request.CorpusID || payload.SnapshotID != request.SnapshotID ||
			payload.Question != request.Question || payload.InterfaceLanguage != request.InterfaceLanguage ||
			payload.RetrievalConfiguration.Strategy != strategyVector ||
			payload.RetrievalConfiguration.Fingerprint != request.RetrievalConfiguration.Fingerprint ||
			payload.ExecutionIdentity.AgentBuild != request.ExecutionIdentity.AgentBuild ||
			payload.ExecutionIdentity.ChatModelIdentity != request.ExecutionIdentity.ChatModelIdentity ||
			payload.ExecutionIdentity.EmbeddingModelIdentity != request.ExecutionIdentity.EmbeddingModelIdentity {
			t.Fatalf("evaluation transport payload = %#v, want fixed request %#v", payload, request)
		}
		writer.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(writer).Encode(validResponse(request)); err != nil {
			t.Fatalf("encode response: %v", err)
		}
	}))
	defer server.Close()

	result, err := NewClient(config.AgentConfig{BaseURL: server.URL, Timeout: time.Second}).Execute(context.Background(), request)
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if result.Materialized.CorpusID != request.CorpusID || result.Materialized.SnapshotID != request.SnapshotID ||
		result.Materialized.Answer != "The fixed snapshot applies [1]." || len(result.Materialized.Retrieved) != 1 ||
		len(result.Materialized.Cited) != 1 {
		t.Fatalf("materialized evaluation = %#v, want fixed, cited result", result.Materialized)
	}
	if result.GraphGrounding.Status != graphNotRequested || result.ModelIdentity != "test-model" ||
		result.AgentBuildIdentity != "agent-build-test" || result.EmbeddingModelIdentity != "embedding-model-test" || result.Telemetry.TotalMilliseconds == nil ||
		*result.Telemetry.TotalMilliseconds != 18 {
		t.Fatalf("result metadata = %#v, want validated agent metadata", result)
	}
	if len(result.CitationMarkers) != 1 || result.CitationMarkers[0] != (CitationMarkerInput{MarkerPosition: 1, EvidenceRank: 1}) {
		t.Fatalf("citation markers = %#v, want ordered mapping", result.CitationMarkers)
	}
}

func TestClientRejectsUnsafeEvaluationResponses(t *testing.T) {
	t.Parallel()

	request := evaluationRequest()
	tests := []struct {
		name   string
		mutate func(map[string]any)
	}{
		{
			name: "cross corpus evidence",
			mutate: func(payload map[string]any) {
				payload["retrievedEvidence"].([]map[string]any)[0]["corpusId"] = uuid.New().String()
			},
		},
		{
			name: "cross snapshot evidence",
			mutate: func(payload map[string]any) {
				payload["retrievedEvidence"].([]map[string]any)[0]["snapshotId"] = uuid.New().String()
			},
		},
		{
			name: "out of order evidence",
			mutate: func(payload map[string]any) {
				payload["retrievedEvidence"].([]map[string]any)[0]["rank"] = 2
			},
		},
		{
			name: "citation mapping mismatch",
			mutate: func(payload map[string]any) {
				payload["citationMarkerInputs"].([]map[string]any)[0]["evidenceRank"] = 2
			},
		},
		{
			name: "invalid vector graph state",
			mutate: func(payload map[string]any) {
				payload["graphGrounding"].(map[string]any)["status"] = graphGrounded
			},
		},
		{
			name: "negative telemetry",
			mutate: func(payload map[string]any) {
				payload["telemetry"].(map[string]any)["inputTokens"] = -1
			},
		},
	}
	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			payload := validResponse(request)
			testCase.mutate(payload)
			server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, _ *http.Request) {
				writer.Header().Set("Content-Type", "application/json")
				if err := json.NewEncoder(writer).Encode(payload); err != nil {
					t.Fatalf("encode response: %v", err)
				}
			}))
			defer server.Close()

			_, err := NewClient(config.AgentConfig{BaseURL: server.URL, Timeout: time.Second}).Execute(context.Background(), request)
			if !errors.Is(err, ErrInvalidResponse) {
				t.Fatalf("Execute() error = %v, want ErrInvalidResponse", err)
			}
		})
	}
}

func TestClientRejectsStreamingAndUnsafeFailureBodies(t *testing.T) {
	t.Parallel()

	request := evaluationRequest()
	for _, testCase := range []struct {
		name        string
		statusCode  int
		contentType string
		body        string
		want        error
	}{
		{name: "streaming response", statusCode: http.StatusOK, contentType: "text/event-stream", body: "event: completed\n", want: ErrInvalidResponse},
		{name: "agent failure", statusCode: http.StatusBadGateway, contentType: "application/json", body: `{"message":"private provider payload"}`, want: ErrUnavailable},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, _ *http.Request) {
				writer.Header().Set("Content-Type", testCase.contentType)
				writer.WriteHeader(testCase.statusCode)
				_, _ = writer.Write([]byte(testCase.body))
			}))
			defer server.Close()

			_, err := NewClient(config.AgentConfig{BaseURL: server.URL, Timeout: time.Second}).Execute(context.Background(), request)
			if !errors.Is(err, testCase.want) {
				t.Fatalf("Execute() error = %v, want %v", err, testCase.want)
			}
			if strings.Contains(err.Error(), "private provider payload") {
				t.Fatalf("Execute() exposed an unsafe failure body: %v", err)
			}
		})
	}
}

func TestClientMapsBoundedFrozenIdentityFailure(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, _ *http.Request) {
		writer.Header().Set("Content-Type", "application/json")
		writer.WriteHeader(http.StatusConflict)
		_, _ = writer.Write([]byte(`{"code":"frozen_identity_unavailable"}`))
	}))
	defer server.Close()

	_, err := NewClient(config.AgentConfig{BaseURL: server.URL, Timeout: time.Second}).Execute(context.Background(), evaluationRequest())
	if !errors.Is(err, ErrFrozenIdentityUnavailable) {
		t.Fatalf("Execute() error = %v, want %v", err, ErrFrozenIdentityUnavailable)
	}
}

func TestEvaluationAdapterDoesNotImportPublicChatApplicationService(t *testing.T) {
	t.Parallel()

	_, testFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("resolve adapter source path")
	}
	contents, err := os.ReadFile(filepath.Join(filepath.Dir(testFile), "client.go"))
	if err != nil {
		t.Fatalf("read adapter source: %v", err)
	}
	if strings.Contains(string(contents), "internal/chat/application") || strings.Contains(string(contents), "chatapplication.") {
		t.Fatal("evaluation adapter must not import or call the public chat application service")
	}
}

func evaluationRequest() Request {
	return Request{
		CorpusID:          uuid.MustParse("10000000-0000-4000-8000-000000000001"),
		SnapshotID:        uuid.MustParse("20000000-0000-4000-8000-000000000001"),
		Question:          "Which fixed rule applies?",
		InterfaceLanguage: interfaceLanguageEN,
		RetrievalConfiguration: FrozenRetrievalConfiguration{
			Strategy: strategyVector, Fingerprint: strings.Repeat("f", 64),
		},
		ExecutionIdentity: application.ExecutionIdentity{AgentBuild: "agent-build-test", ChatModelIdentity: "test-model", EmbeddingModelIdentity: "embedding-model-test"},
	}
}

func validResponse(request Request) map[string]any {
	return map[string]any{
		"answer":  "The fixed snapshot applies [1].",
		"outcome": outcomeCompleted,
		"retrievedEvidence": []map[string]any{{
			"rank":             1,
			"corpusId":         request.CorpusID.String(),
			"snapshotId":       request.SnapshotID.String(),
			"sourceId":         "30000000-0000-4000-8000-000000000001",
			"sourceRevisionId": "40000000-0000-4000-8000-000000000001",
			"documentId":       "50000000-0000-4000-8000-000000000001",
			"unitId":           "60000000-0000-4000-8000-000000000001",
			"canonicalLocator": "article:1/item:a",
			"startOffset":      0,
			"endOffset":        10,
			"contentSha256":    "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
		}},
		"citationMarkerInputs":   []map[string]any{{"markerPosition": 1, "evidenceRank": 1}},
		"graphGrounding":         map[string]any{"status": graphNotRequested},
		"modelIdentity":          "test-model",
		"agentBuildIdentity":     "agent-build-test",
		"embeddingModelIdentity": "embedding-model-test",
		"telemetry": map[string]any{
			"retrievalMilliseconds":  5,
			"generationMilliseconds": 11,
			"totalMilliseconds":      18,
			"inputTokens":            7,
			"outputTokens":           nil,
		},
	}
}
