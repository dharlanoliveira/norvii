package agent

import (
	"context"
	"encoding/json"
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
