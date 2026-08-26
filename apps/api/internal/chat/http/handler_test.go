package http

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	chatdomain "github.com/dharlanoliveira/norvii/apps/api/internal/chat/domain"
	"github.com/google/uuid"
)

func TestHandlerStreamsGroundedEventsInOrder(t *testing.T) {
	corpusID := uuid.New()
	service := fakeService{result: chatdomain.Result{
		Answer: "The rule applies [1].",
		Evidence: []chatdomain.Evidence{{
			ID: "evidence-1", CorpusID: corpusID, SourceID: uuid.New(), DocumentID: uuid.New(),
			UnitLocator: "article-1", StartOffset: 0, EndOffset: 20, Excerpt: "Article 1", Rank: 1,
		}},
		Inspection: &chatdomain.Inspection{AssertionPath: []chatdomain.AssertionPathStep{{
			AssertionID: "assertion-1", Predicate: "imposes_duty_on", SubjectLabel: "Authority", ObjectLabel: "Impact report",
			EstablishingLocator: "article-1", EvidenceLocator: "item-1", HierarchyContext: []string{"chapter-1", "article-1"},
		}}},
	}}
	mux := http.NewServeMux()
	NewHandler(service).Register(mux)
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, "/api/v1/corpora/"+corpusID.String()+"/chat/stream", strings.NewReader(`{"question":"What applies?","interfaceLanguage":"en"}`))

	mux.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", recorder.Code)
	}
	body := recorder.Body.String()
	for _, event := range []string{"event: started", "event: evidence", "event: delta", "event: completed"} {
		if !strings.Contains(body, event) {
			t.Fatalf("stream missing %q: %s", event, body)
		}
	}
	if strings.Index(body, "event: evidence") > strings.Index(body, "event: delta") {
		t.Fatalf("evidence event must precede delta: %s", body)
	}
	if !strings.Contains(body, `"unitLocator":"article-1"`) {
		t.Fatalf("evidence event must use the public camelCase contract: %s", body)
	}
	if !strings.Contains(body, `"inspection":{"outcome":"completed"`) ||
		!strings.Contains(body, `"totalMilliseconds":`) {
		t.Fatalf("completed event must expose inspection measurements: %s", body)
	}
	if !strings.Contains(body, `"assertionPath":[{"assertionId":"assertion-1","predicate":"imposes_duty_on"`) {
		t.Fatalf("completed event must expose the assertion path: %s", body)
	}
	if !strings.Contains(body, `"hierarchyContext":["chapter-1","article-1"]`) {
		t.Fatalf("completed event must expose assertion hierarchy context: %s", body)
	}
}

func TestHandlerMapsInsufficientEvidenceToAbstention(t *testing.T) {
	service := fakeService{err: chatdomain.ErrInsufficientEvidence}
	mux := http.NewServeMux()
	NewHandler(service).Register(mux)
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, "/api/v1/corpora/"+uuid.NewString()+"/chat/stream", strings.NewReader(`{"question":"Unknown?"}`))

	mux.ServeHTTP(recorder, request)

	if !strings.Contains(recorder.Body.String(), `"type":"abstained"`) {
		t.Fatalf("body = %s, want abstention event", recorder.Body.String())
	}
}

func TestHandlerRejectsInvalidQuestion(t *testing.T) {
	service := fakeService{}
	mux := http.NewServeMux()
	NewHandler(service).Register(mux)
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, "/api/v1/corpora/not-a-uuid/chat/stream", strings.NewReader(`{"question":""}`))

	mux.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", recorder.Code)
	}
}

func TestHandlerMapsCancellationToCancelledTerminalEvent(t *testing.T) {
	mux := http.NewServeMux()
	NewHandler(fakeService{err: context.Canceled}).Register(mux)
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, "/api/v1/corpora/10000000-0000-4000-8000-000000000001/chat/stream", strings.NewReader(`{"question":"What applies?"}`))

	mux.ServeHTTP(recorder, request)

	if !strings.Contains(recorder.Body.String(), `"type":"cancelled"`) {
		t.Fatalf("body = %s, want cancellation event", recorder.Body.String())
	}
}

func TestHandlerEmitsExactlyOneTerminalEventAfterOrderedStreamEvents(t *testing.T) {
	corpusID := uuid.New()
	tests := []struct {
		name         string
		service      fakeService
		wantEvents   []string
		wantTerminal string
	}{
		{
			name: "completed",
			service: fakeService{result: chatdomain.Result{
				Answer: "The rule applies.",
				Evidence: []chatdomain.Evidence{{
					ID: "evidence-1", CorpusID: corpusID, SnapshotID: uuid.New(), SourceID: uuid.New(), DocumentID: uuid.New(),
					UnitLocator: "article-1", StartOffset: 0, EndOffset: 10, Excerpt: "The rule.", Rank: 1,
				}},
			}},
			wantEvents:   []string{"started", "evidence", "delta", "completed"},
			wantTerminal: "completed",
		},
		{
			name:         "abstained",
			service:      fakeService{err: chatdomain.ErrInsufficientEvidence},
			wantEvents:   []string{"started", "abstained"},
			wantTerminal: "abstained",
		},
		{
			name:         "cancelled",
			service:      fakeService{err: context.Canceled},
			wantEvents:   []string{"started", "cancelled"},
			wantTerminal: "cancelled",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			mux := http.NewServeMux()
			NewHandler(test.service).Register(mux)
			recorder := httptest.NewRecorder()
			request := httptest.NewRequest(
				http.MethodPost,
				"/api/v1/corpora/"+corpusID.String()+"/chat/stream",
				strings.NewReader(`{"question":"What applies?"}`),
			)

			mux.ServeHTTP(recorder, request)

			events := streamEventTypes(t, recorder.Body.String())
			if strings.Join(events, ",") != strings.Join(test.wantEvents, ",") {
				t.Fatalf("events = %v, want %v", events, test.wantEvents)
			}
			if terminalEventCount(events) != 1 || events[len(events)-1] != test.wantTerminal {
				t.Fatalf("terminal events = %v, want exactly one %q terminal", events, test.wantTerminal)
			}
		})
	}
}

func TestHandlerRejectsInvalidPayloads(t *testing.T) {
	tests := []struct {
		name string
		body string
		want string
	}{
		{name: "malformed JSON", body: "{", want: "invalid_question"},
		{name: "unknown field", body: `{"question":"What applies?","unexpected":true}`, want: "invalid_question"},
		{name: "invalid language", body: `{"question":"What applies?","interfaceLanguage":"fr"}`, want: "invalid_input"},
		{name: "removed graph strategy", body: `{"question":"What applies?","strategy":"graph"}`, want: "invalid_input"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			recorder := httptest.NewRecorder()
			request := httptest.NewRequest(http.MethodPost, "/api/v1/corpora/"+uuid.NewString()+"/chat/stream", strings.NewReader(test.body))
			mux := http.NewServeMux()
			NewHandler(fakeService{}).Register(mux)

			mux.ServeHTTP(recorder, request)
			if !strings.Contains(recorder.Body.String(), `"code":"`+test.want+`"`) {
				t.Fatalf("body = %s, want code %q", recorder.Body.String(), test.want)
			}
		})
	}
}

func TestHandlerMapsTerminalFailures(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want string
	}{
		{name: "grounding validation", err: chatdomain.ErrGroundingValidation, want: "abstained"},
		{name: "retrieval", err: chatdomain.ErrRetrievalFailed, want: "retrieval_failed"},
		{name: "invalid question", err: chatdomain.ErrInvalidQuestion, want: "invalid_question"},
		{name: "missing active snapshot", err: chatdomain.ErrSnapshotUnavailable, want: "snapshot_unavailable"},
		{name: "generation", err: chatdomain.ErrGenerationFailed, want: "generation_failed"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			recorder := httptest.NewRecorder()
			request := httptest.NewRequest(http.MethodPost, "/api/v1/corpora/"+uuid.NewString()+"/chat/stream", strings.NewReader(`{"question":"What applies?"}`))
			mux := http.NewServeMux()
			NewHandler(fakeService{err: test.err}).Register(mux)

			mux.ServeHTTP(recorder, request)
			if !strings.Contains(recorder.Body.String(), `"type":"`+test.want+`"`) && !strings.Contains(recorder.Body.String(), `"code":"`+test.want+`"`) {
				t.Fatalf("body = %s, want terminal %q", recorder.Body.String(), test.want)
			}
		})
	}
}

type fakeService struct {
	result chatdomain.Result
	err    error
}

func streamEventTypes(t *testing.T, body string) []string {
	t.Helper()
	types := make([]string, 0)
	for _, record := range strings.Split(strings.TrimSpace(body), "\n\n") {
		var event struct {
			Type string `json:"type"`
		}
		data := strings.TrimPrefix(strings.Split(record, "\n")[1], "data: ")
		if err := json.Unmarshal([]byte(data), &event); err != nil {
			t.Fatalf("decode SSE event %q: %v", record, err)
		}
		types = append(types, event.Type)
	}
	return types
}

func terminalEventCount(events []string) int {
	terminals := map[string]bool{
		"completed": true,
		"abstained": true,
		"cancelled": true,
		"error":     true,
	}
	count := 0
	for _, event := range events {
		if terminals[event] {
			count++
		}
	}
	return count
}

func (fake fakeService) Ask(_ context.Context, _ chatdomain.Request, emit func(string)) (chatdomain.Result, error) {
	if fake.err != nil {
		return chatdomain.Result{}, fake.err
	}
	emit(fake.result.Answer)
	return fake.result, nil
}
