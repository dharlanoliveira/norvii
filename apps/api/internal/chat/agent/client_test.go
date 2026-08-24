package agent

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	chatdomain "github.com/dharlanoliveira/norvii/apps/api/internal/chat/domain"
	"github.com/dharlanoliveira/norvii/apps/api/internal/platform/config"
	"github.com/google/uuid"
)

func TestClientRelaysLangGraphEvents(t *testing.T) {
	corpusID := uuid.New()
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		if request.Method != http.MethodPost || !strings.HasSuffix(request.URL.Path, "/chat/stream") {
			t.Fatalf("unexpected agent request: %s %s", request.Method, request.URL.Path)
		}
		var payload map[string]string
		if err := json.NewDecoder(request.Body).Decode(&payload); err != nil {
			t.Fatalf("decode agent request: %v", err)
		}
		if payload["interfaceLanguage"] != "en" {
			t.Fatalf("interfaceLanguage = %q, want en", payload["interfaceLanguage"])
		}
		writer.Header().Set("Content-Type", "text/event-stream")
		_, _ = writer.Write([]byte("event: delta\ndata: {\"type\":\"delta\",\"text\":\"Answer [1].\"}\n\n"))
		_, _ = writer.Write([]byte("event: completed\ndata: {\"type\":\"completed\",\"answer\":\"Answer [1].\",\"references\":[] ,\"telemetry\":{}}\n\n"))
	}))
	defer server.Close()
	client := NewClient(config.AgentConfig{BaseURL: server.URL, Timeout: time.Second})
	var deltas []string
	result, err := client.Ask(context.Background(), chatdomain.Request{
		CorpusID: corpusID, Question: "What applies?", InterfaceLanguage: "en",
	}, func(delta string) { deltas = append(deltas, delta) })

	if err != nil {
		t.Fatalf("Ask() error = %v", err)
	}
	if result.Answer != "Answer [1]." || len(deltas) != 1 {
		t.Fatalf("result = %#v, deltas = %#v", result, deltas)
	}
}

func TestClientMapsAgentTerminalEvents(t *testing.T) {
	tests := []struct {
		name       string
		body       string
		wantErr    error
		wantAnswer string
	}{
		{
			name:    "insufficient evidence",
			body:    "event: abstained\ndata: {\"type\":\"abstained\",\"reason\":\"no_evidence\"}\n\n",
			wantErr: chatdomain.ErrInsufficientEvidence,
		},
		{
			name:    "grounding validation",
			body:    "event: abstained\ndata: {\"type\":\"abstained\",\"reason\":\"grounding_validation_failed\"}\n\n",
			wantErr: chatdomain.ErrGroundingValidation,
		},
		{
			name:    "cancelled",
			body:    "event: cancelled\ndata: {\"type\":\"cancelled\"}\n\n",
			wantErr: context.Canceled,
		},
		{
			name:    "agent error",
			body:    "event: error\ndata: {\"type\":\"error\",\"code\":\"retrieval_failed\"}\n\n",
			wantErr: errors.New("agent request failed: retrieval_failed"),
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, _ *http.Request) {
				_, _ = writer.Write([]byte(test.body))
			}))
			defer server.Close()

			client := NewClient(config.AgentConfig{BaseURL: server.URL, Timeout: time.Second})
			result, err := client.Ask(context.Background(), chatdomain.Request{CorpusID: uuid.New()}, func(string) {})
			if test.wantErr != nil {
				if err == nil || err.Error() != test.wantErr.Error() {
					t.Fatalf("Ask() error = %v, want %v", err, test.wantErr)
				}
				return
			}
			if err != nil || result.Answer != test.wantAnswer {
				t.Fatalf("Ask() = %#v, %v", result, err)
			}
		})
	}
}

func TestClientHandlesEvidenceAndUnknownEvents(t *testing.T) {
	corpusID := uuid.New()
	sourceID := uuid.New()
	documentID := uuid.New()
	reference := `{"id":"ref-1","corpusId":"` + corpusID.String() + `","sourceId":"` + sourceID.String() + `","documentId":"` + documentID.String() + `","unitLocator":"article-1","startOffset":1,"endOffset":8,"excerpt":"text","rank":1}`
	body := strings.Join([]string{
		"event: evidence\ndata: {\"type\":\"evidence\",\"references\":[" + reference + "]}\n\n",
		"event: ignored\ndata: {\"type\":\"progress\"}\n\n",
		"event: completed\ndata: {\"type\":\"completed\",\"answer\":\"Answer\",\"references\":[" + reference + "]}\n\n",
	}, "")
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, _ *http.Request) {
		_, _ = writer.Write([]byte(body))
	}))
	defer server.Close()

	result, err := NewClient(config.AgentConfig{BaseURL: server.URL, Timeout: time.Second}).Ask(
		context.Background(), chatdomain.Request{CorpusID: corpusID}, func(string) {},
	)
	if err != nil {
		t.Fatalf("Ask() error = %v", err)
	}
	if len(result.Evidence) != 1 || result.Evidence[0].UnitLocator != "article-1" {
		t.Fatalf("evidence = %#v", result.Evidence)
	}
}

func TestClientMapsInspectionMetadata(t *testing.T) {
	corpusID := uuid.New()
	sourceID := uuid.New()
	documentID := uuid.New()
	reference := `{"id":"ref-1","corpusId":"` + corpusID.String() + `","sourceId":"` + sourceID.String() + `","documentId":"` + documentID.String() + `","documentVersionId":"` + documentID.String() + `","unitLocator":"article-1","startOffset":1,"endOffset":8,"excerpt":"text","rank":1,"cosineDistance":0.18}`
	body := "event: completed\ndata: {\"type\":\"completed\",\"answer\":\"Answer\",\"references\":[" + reference + "],\"inspection\":{\"outcome\":\"completed\",\"retrieval\":{\"strategy\":\"vector\",\"topK\":8,\"returnedCount\":1,\"embeddingModel\":null},\"measurements\":{\"retrievalMilliseconds\":12,\"generationMilliseconds\":34,\"totalMilliseconds\":50,\"inputTokens\":null,\"outputTokens\":7},\"evidence\":[" + reference + "]}}\n\n"
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, _ *http.Request) {
		_, _ = writer.Write([]byte(body))
	}))
	defer server.Close()

	result, err := NewClient(config.AgentConfig{BaseURL: server.URL, Timeout: time.Second}).Ask(
		context.Background(), chatdomain.Request{CorpusID: corpusID}, func(string) {},
	)
	if err != nil {
		t.Fatalf("Ask() error = %v", err)
	}
	if result.Inspection == nil || result.Inspection.Retrieval == nil {
		t.Fatalf("inspection = %#v, want retrieval metadata", result.Inspection)
	}
	if result.Inspection.Measurements.TotalMilliseconds == nil || *result.Inspection.Measurements.TotalMilliseconds != 50 {
		t.Fatalf("total milliseconds = %#v, want 50", result.Inspection.Measurements.TotalMilliseconds)
	}
	if result.Evidence[0].CosineDistance == nil || *result.Evidence[0].CosineDistance != 0.18 {
		t.Fatalf("cosine distance = %#v, want 0.18", result.Evidence[0].CosineDistance)
	}
}

func TestClientRejectsInvalidAndIncompleteAgentStreams(t *testing.T) {
	tests := []struct {
		name string
		body string
	}{
		{name: "invalid event", body: "event: delta\ndata: {not-json}\n\n"},
		{name: "missing terminal", body: "event: delta\ndata: {\"type\":\"delta\",\"text\":\"partial\"}\n\n"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, _ *http.Request) {
				_, _ = writer.Write([]byte(test.body))
			}))
			defer server.Close()

			_, err := NewClient(config.AgentConfig{BaseURL: server.URL, Timeout: time.Second}).Ask(
				context.Background(), chatdomain.Request{CorpusID: uuid.New()}, func(string) {},
			)
			if err == nil {
				t.Fatal("Ask() error = nil, want error")
			}
		})
	}
}

func TestClientRejectsAgentHTTPFailure(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, _ *http.Request) {
		writer.WriteHeader(http.StatusBadGateway)
	}))
	defer server.Close()

	_, err := NewClient(config.AgentConfig{BaseURL: server.URL, Timeout: time.Second}).Ask(
		context.Background(), chatdomain.Request{CorpusID: uuid.New()}, func(string) {},
	)
	if !errors.Is(err, ErrUnavailable) {
		t.Fatalf("Ask() error = %v, want ErrUnavailable", err)
	}
}
