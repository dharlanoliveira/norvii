package http

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	documentpostgres "github.com/dharlanoliveira/norvii/apps/api/internal/document/postgres"
	"github.com/google/uuid"
)

type fakeReader struct {
	document documentpostgres.Document
}

func (reader *fakeReader) GetLatest(context.Context, uuid.UUID, uuid.UUID) (documentpostgres.Document, error) {
	return reader.document, nil
}

func (reader *fakeReader) GetVersion(context.Context, uuid.UUID, uuid.UUID, uuid.UUID) (documentpostgres.Document, error) {
	return reader.document, nil
}

func TestGetLatestWritesCompleteDocumentAndUnits(t *testing.T) {
	corpusID := uuid.New()
	sourceID := uuid.New()
	mux := http.NewServeMux()
	NewHandler(&fakeReader{document: documentpostgres.Document{
		ID: uuid.New(), SourceRevisionID: uuid.New(), PipelineVersion: "corpus-ingestion-v1",
		Text: "Persisted legal text.", TextSHA256: strings.Repeat("a", 64), CreatedAt: time.Now(),
		Provenance: documentpostgres.Provenance{
			ContentSHA256: strings.Repeat("c", 64), CapturedAt: time.Now(),
			MediaType: "text/html", ByteSize: 2048,
		},
		Units: []documentpostgres.Unit{{
			ID: uuid.New(), Kind: "document", Locator: "document", ContentSHA256: strings.Repeat("b", 64),
		}},
	}}).Register(mux)
	recorder := httptest.NewRecorder()

	mux.ServeHTTP(recorder, httptest.NewRequest(
		http.MethodGet,
		"/api/v1/corpora/"+corpusID.String()+"/sources/"+sourceID.String()+"/document",
		nil,
	))

	if recorder.Code != http.StatusOK || !strings.Contains(recorder.Body.String(), "Persisted legal text.") ||
		!strings.Contains(recorder.Body.String(), `"mediaType":"text/html"`) {
		t.Fatalf("response status/body = %d/%s", recorder.Code, recorder.Body.String())
	}
}

func TestGetVersionWritesTheRequestedDocumentRoute(t *testing.T) {
	corpusID := uuid.New()
	sourceID := uuid.New()
	documentVersionID := uuid.New()
	mux := http.NewServeMux()
	NewHandler(&fakeReader{document: documentpostgres.Document{ID: documentVersionID, Text: "Immutable legal text."}}).Register(mux)
	recorder := httptest.NewRecorder()
	mux.ServeHTTP(recorder, httptest.NewRequest(
		http.MethodGet,
		"/api/v1/corpora/"+corpusID.String()+"/sources/"+sourceID.String()+"/documents/"+documentVersionID.String(),
		nil,
	))

	if recorder.Code != http.StatusOK || !strings.Contains(recorder.Body.String(), "Immutable legal text.") {
		t.Fatalf("response status/body = %d/%s", recorder.Code, recorder.Body.String())
	}
}
