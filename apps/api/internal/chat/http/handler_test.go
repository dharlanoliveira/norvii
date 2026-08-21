package http

import (
	"context"
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

func TestHandlerRejectsInvalidPayloads(t *testing.T) {
	tests := []struct {
		name string
		body string
		want string
	}{
		{name: "malformed JSON", body: "{", want: "invalid_question"},
		{name: "unknown field", body: `{"question":"What applies?","unexpected":true}`, want: "invalid_question"},
		{name: "invalid language", body: `{"question":"What applies?","interfaceLanguage":"fr"}`, want: "invalid_input"},
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

func (fake fakeService) Ask(_ context.Context, _ chatdomain.Request, emit func(string)) (chatdomain.Result, error) {
	if fake.err != nil {
		return chatdomain.Result{}, fake.err
	}
	emit(fake.result.Answer)
	return fake.result, nil
}
