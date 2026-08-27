package http

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/dharlanoliveira/norvii/apps/api/internal/suggestions/domain"
	suggestionspostgres "github.com/dharlanoliveira/norvii/apps/api/internal/suggestions/postgres"
	"github.com/google/uuid"
)

type fakeReader struct {
	results map[domain.QueryLanguage]suggestionspostgres.ReadResult
	err     error
}

func (reader *fakeReader) Read(
	_ context.Context,
	_ uuid.UUID,
	language domain.QueryLanguage,
) (suggestionspostgres.ReadResult, error) {
	if reader.err != nil {
		return suggestionspostgres.ReadResult{}, reader.err
	}
	return reader.results[language], nil
}

func TestGetWritesContractFixturesForEachLanguage(t *testing.T) {
	corpusID := uuid.MustParse("10000000-0000-4000-8000-000000000011")
	snapshotID := uuid.MustParse("20000000-0000-4000-8000-000000000011")
	manifest := domain.SHA256(strings.Repeat("3", 64))
	reader := &fakeReader{results: map[domain.QueryLanguage]suggestionspostgres.ReadResult{
		domain.QueryLanguageEnglish:    fixtureResult(corpusID, snapshotID, manifest, "en"),
		domain.QueryLanguagePortuguese: fixtureResult(corpusID, snapshotID, manifest, "pt"),
	}}
	mux := http.NewServeMux()
	NewHandler(reader).Register(mux)

	for _, language := range []string{"en", "pt"} {
		t.Run(language, func(t *testing.T) {
			recorder := serveRequest(mux, corpusID.String(), "?interfaceLanguage="+language)
			if recorder.Code != http.StatusOK {
				t.Fatalf("status/body = %d/%s, want 200", recorder.Code, recorder.Body.String())
			}
			assertJSONEqual(t, readFixture(t, "suggestions-response-"+language+".json"), recorder.Body.Bytes())
			assertResponseHasNoEvaluationFields(t, recorder.Body.String())
		})
	}
}

func TestGetWritesNormalEmptyResultsForUnavailableStates(t *testing.T) {
	noSnapshotCorpusID := uuid.MustParse("10000000-0000-4000-8000-000000000012")
	staleCorpusID := uuid.MustParse("10000000-0000-4000-8000-000000000011")
	staleSnapshotID := uuid.MustParse("20000000-0000-4000-8000-000000000012")
	reader := &fakeReader{results: map[domain.QueryLanguage]suggestionspostgres.ReadResult{
		domain.QueryLanguageEnglish: {CorpusID: noSnapshotCorpusID, Suggestions: []domain.PublicOpeningSuggestion{}},
		domain.QueryLanguagePortuguese: {
			CorpusID: staleCorpusID,
			ActiveSnapshot: &domain.Snapshot{ID: staleSnapshotID, CorpusID: staleCorpusID,
				ManifestSHA256: domain.SHA256(strings.Repeat("4", 64))},
			Suggestions: []domain.PublicOpeningSuggestion{},
		},
	}}
	mux := http.NewServeMux()
	NewHandler(reader).Register(mux)

	tests := []struct {
		name        string
		corpusID    uuid.UUID
		query       string
		fixtureName string
	}{
		{name: "disabled or missing corpus", corpusID: noSnapshotCorpusID, query: "?interfaceLanguage=en", fixtureName: "suggestions-response-empty-no-active-snapshot.json"},
		{name: "stale projection", corpusID: staleCorpusID, query: "?interfaceLanguage=pt", fixtureName: "suggestions-response-empty-stale-projection.json"},
	}
	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			recorder := serveRequest(mux, testCase.corpusID.String(), testCase.query)
			if recorder.Code != http.StatusOK {
				t.Fatalf("status/body = %d/%s, want 200", recorder.Code, recorder.Body.String())
			}
			assertJSONEqual(t, readFixture(t, testCase.fixtureName), recorder.Body.Bytes())
		})
	}
}

func TestGetRejectsMalformedRequestAndHidesReaderFailures(t *testing.T) {
	corpusID := uuid.New()
	tests := []struct {
		name   string
		reader *fakeReader
		path   string
		status int
		code   string
	}{
		{name: "malformed corpus identifier", reader: &fakeReader{}, path: "/api/v1/corpora/not-a-uuid/opening-suggestions?interfaceLanguage=en", status: http.StatusBadRequest, code: "invalid_input"},
		{name: "zero corpus identifier", reader: &fakeReader{}, path: "/api/v1/corpora/00000000-0000-0000-0000-000000000000/opening-suggestions?interfaceLanguage=en", status: http.StatusBadRequest, code: "invalid_input"},
		{name: "missing language", reader: &fakeReader{}, path: "/api/v1/corpora/" + corpusID.String() + "/opening-suggestions", status: http.StatusBadRequest, code: "invalid_input"},
		{name: "unsupported language", reader: &fakeReader{}, path: "/api/v1/corpora/" + corpusID.String() + "/opening-suggestions?interfaceLanguage=fr", status: http.StatusBadRequest, code: "invalid_input"},
		{name: "repeated language", reader: &fakeReader{}, path: "/api/v1/corpora/" + corpusID.String() + "/opening-suggestions?interfaceLanguage=en&interfaceLanguage=pt", status: http.StatusBadRequest, code: "invalid_input"},
		{name: "reader failure", reader: &fakeReader{err: errors.New("database credentials exposed")}, path: "/api/v1/corpora/" + corpusID.String() + "/opening-suggestions?interfaceLanguage=en", status: http.StatusServiceUnavailable, code: "unavailable"},
	}
	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			mux := http.NewServeMux()
			NewHandler(testCase.reader).Register(mux)
			recorder := httptest.NewRecorder()
			mux.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, testCase.path, nil))
			if recorder.Code != testCase.status || !strings.Contains(recorder.Body.String(), `"code":"`+testCase.code+`"`) {
				t.Fatalf("status/body = %d/%s, want %d/%s", recorder.Code, recorder.Body.String(), testCase.status, testCase.code)
			}
			if strings.Contains(recorder.Body.String(), "credentials") {
				t.Fatalf("response leaked reader failure: %s", recorder.Body.String())
			}
		})
	}
}

func fixtureResult(
	corpusID, snapshotID uuid.UUID,
	manifest domain.SHA256,
	language string,
) suggestionspostgres.ReadResult {
	suggestions := make([]domain.PublicOpeningSuggestion, 0, 5)
	questions := map[string][]string{
		"en": {
			"Which synthetic obligation applies to a controller?", "When must a synthetic notice be provided?",
			"Which synthetic actor receives the request?", "What synthetic condition limits the processing activity?",
			"Which synthetic safeguard must be observed?",
		},
		"pt": {
			"Qual obrigacao sintetica se aplica ao controlador?", "Quando um aviso sintetico deve ser fornecido?",
			"Qual agente sintetico recebe a solicitacao?", "Qual condicao sintetica limita a atividade de tratamento?",
			"Qual salvaguarda sintetica deve ser observada?",
		},
	}
	for index, question := range questions[language] {
		suggestions = append(suggestions, domain.PublicOpeningSuggestion{
			CaseID: "brazil-data-protection-00" + string(rune('1'+index)) + "-" + language,
			Rank:   index + 1, Question: question,
		})
	}
	return suggestionspostgres.ReadResult{
		CorpusID:       corpusID,
		ActiveSnapshot: &domain.Snapshot{ID: snapshotID, CorpusID: corpusID, ManifestSHA256: manifest},
		Suggestions:    suggestions,
	}
}

func serveRequest(mux *http.ServeMux, corpusID, query string) *httptest.ResponseRecorder {
	recorder := httptest.NewRecorder()
	mux.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/api/v1/corpora/"+corpusID+"/opening-suggestions"+query, nil))
	return recorder
}

func readFixture(t *testing.T, name string) []byte {
	t.Helper()
	payload, err := os.ReadFile(filepath.Join("..", "..", "..", "..", "..", "contracts", "corpus-opening-suggestions", "v1", "fixtures", name))
	if err != nil {
		t.Fatalf("ReadFile(%q) error = %v", name, err)
	}
	return payload
}

func assertJSONEqual(t *testing.T, expected, actual []byte) {
	t.Helper()
	var expectedValue any
	if err := json.Unmarshal(expected, &expectedValue); err != nil {
		t.Fatalf("unmarshal expected JSON: %v", err)
	}
	var actualValue any
	if err := json.Unmarshal(actual, &actualValue); err != nil {
		t.Fatalf("unmarshal actual JSON: %v", err)
	}
	if !equalJSON(expectedValue, actualValue) {
		t.Fatalf("response = %s, want %s", actual, expected)
	}
}

func equalJSON(left, right any) bool {
	leftJSON, _ := json.Marshal(left)
	rightJSON, _ := json.Marshal(right)
	return string(leftJSON) == string(rightJSON)
}

func assertResponseHasNoEvaluationFields(t *testing.T, response string) {
	t.Helper()
	for _, field := range []string{
		"answer", "referenceAnswer", "expectedEvidence", "datasetRevision", "score",
		"configuration", "provider", "prompt",
	} {
		if strings.Contains(response, `"`+field+`"`) {
			t.Fatalf("response leaked evaluation field %q: %s", field, response)
		}
	}
}
