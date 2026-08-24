package http

import (
	"bytes"
	"context"
	"errors"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/dharlanoliveira/norvii/apps/api/internal/source/domain"
	sourcepostgres "github.com/dharlanoliveira/norvii/apps/api/internal/source/postgres"
	"github.com/google/uuid"
)

type fakeReader struct {
	record    sourcepostgres.Record
	err       error
	pdfOrigin sourcepostgres.PDFOriginRecord
	pdfError  error
	urlOrigin sourcepostgres.URLOriginRecord
	urlError  error
}

type fakeCommands struct {
	title string
	url   string
	err   error
}

func (commands *fakeCommands) CreateURL(
	_ context.Context,
	corpusID uuid.UUID,
	title string,
	url string,
) (sourcepostgres.Record, error) {
	commands.title = title
	commands.url = url
	return sourcepostgres.Record{
		ID: uuid.New(), CorpusID: corpusID, Title: title, Kind: domain.KindURL,
		ProcessingStatus: domain.StatusPending, Version: 1,
		CreatedAt: time.Now().UTC(), UpdatedAt: time.Now().UTC(),
	}, commands.err
}

func (commands *fakeCommands) CreatePDF(
	_ context.Context,
	corpusID uuid.UUID,
	title string,
	_ string,
	_ string,
	_ []byte,
) (sourcepostgres.Record, error) {
	return sourcepostgres.Record{
		ID: uuid.New(), CorpusID: corpusID, Title: title, Kind: domain.KindPDF,
		ProcessingStatus: domain.StatusPending, Version: 1,
		CreatedAt: time.Now().UTC(), UpdatedAt: time.Now().UTC(),
	}, commands.err
}

func (commands *fakeCommands) Retry(
	_ context.Context, corpusID uuid.UUID, sourceID uuid.UUID, version int,
) (sourcepostgres.Record, error) {
	return sourcepostgres.Record{
		ID: sourceID, CorpusID: corpusID, Title: "Retried", Kind: domain.KindURL,
		ProcessingStatus: domain.StatusPending, Version: version + 1,
		CreatedAt: time.Now().UTC(), UpdatedAt: time.Now().UTC(),
	}, commands.err
}

func (commands *fakeCommands) Reprocess(
	ctx context.Context, corpusID uuid.UUID, sourceID uuid.UUID, version int,
) (sourcepostgres.Record, error) {
	return commands.Retry(ctx, corpusID, sourceID, version)
}

func (reader *fakeReader) ListByCorpus(context.Context, uuid.UUID) ([]sourcepostgres.Record, error) {
	return []sourcepostgres.Record{reader.record}, reader.err
}

func (reader *fakeReader) Get(context.Context, uuid.UUID, uuid.UUID) (sourcepostgres.Record, error) {
	return reader.record, reader.err
}

func (reader *fakeReader) GetPDFOrigin(
	context.Context, uuid.UUID, uuid.UUID,
) (sourcepostgres.PDFOriginRecord, error) {
	return reader.pdfOrigin, reader.pdfError
}

func (reader *fakeReader) GetURLOrigin(
	context.Context, uuid.UUID, uuid.UUID,
) (sourcepostgres.URLOriginRecord, error) {
	if reader.urlOrigin.URL == "" {
		return sourcepostgres.URLOriginRecord{URL: "https://example.org/final"}, reader.urlError
	}
	return reader.urlOrigin, reader.urlError
}

func TestListWritesOnlyCorpusOwnedSources(t *testing.T) {
	corpusID := uuid.MustParse("10000000-0000-4000-8000-000000000002")
	activeDocumentID := uuid.New()
	now := time.Date(2026, time.August, 17, 12, 0, 0, 0, time.UTC)
	mux := http.NewServeMux()
	NewHandler(&fakeReader{record: sourcepostgres.Record{
		ID: uuid.New(), CorpusID: corpusID, Title: "Official text", Kind: domain.KindURL,
		ProcessingStatus: domain.StatusPending, ActiveSnapshotDocumentID: &activeDocumentID,
		Version: 1, CreatedAt: now, UpdatedAt: now,
	}}).Register(mux)
	recorder := httptest.NewRecorder()

	mux.ServeHTTP(recorder, httptest.NewRequest(
		http.MethodGet, "/api/v1/corpora/"+corpusID.String()+"/sources", nil,
	))

	if recorder.Code != http.StatusOK || !strings.Contains(recorder.Body.String(), `"kind":"url"`) ||
		!strings.Contains(recorder.Body.String(), `"activeSnapshotDocumentId":"`+activeDocumentID.String()+`"`) {
		t.Fatalf("response status/body = %d/%s", recorder.Code, recorder.Body.String())
	}
}

func TestGetWritesOriginAndLatestSafeAttempt(t *testing.T) {
	corpusID, sourceID := uuid.New(), uuid.New()
	submittedURL := "https://example.org/law"
	now := time.Date(2026, time.August, 17, 12, 0, 0, 0, time.UTC)
	mux := http.NewServeMux()
	NewHandler(&fakeReader{record: sourcepostgres.Record{
		ID: sourceID, CorpusID: corpusID, Title: "Official text", Kind: domain.KindURL,
		ProcessingStatus: domain.StatusFailed, Version: 2, CreatedAt: now, UpdatedAt: now,
		Origin: sourcepostgres.Origin{SubmittedURL: &submittedURL},
		LatestAttempt: &sourcepostgres.Attempt{
			Number: 1, PipelineVersion: "corpus-ingestion-v1", Status: "failed", StartedAt: now,
		},
		Attempts: []sourcepostgres.Attempt{
			{Number: 2, PipelineVersion: "corpus-ingestion-v1", Status: "failed", StartedAt: now},
			{
				Number: 1, PipelineVersion: "corpus-ingestion-v1", Status: "failed",
				StartedAt: now.Add(-time.Hour),
			},
		},
	}}).Register(mux)
	recorder := httptest.NewRecorder()

	mux.ServeHTTP(recorder, httptest.NewRequest(
		http.MethodGet,
		"/api/v1/corpora/"+corpusID.String()+"/sources/"+sourceID.String(),
		nil,
	))

	body := recorder.Body.String()
	if recorder.Code != http.StatusOK || !strings.Contains(body, `"submittedUrl":"https://example.org/law"`) ||
		!strings.Contains(body, `"attempts":[`) || strings.Count(body, `"pipelineVersion"`) != 3 {
		t.Fatalf("response = %d/%s, want safe source detail", recorder.Code, body)
	}
}

func TestGetURLOriginWritesMetadataInsteadOfRedirecting(t *testing.T) {
	corpusID, sourceID := uuid.New(), uuid.New()
	finalURL := "https://example.org/final"
	mux := http.NewServeMux()
	NewHandler(&fakeReader{record: sourcepostgres.Record{
		ID: sourceID, CorpusID: corpusID, Kind: domain.KindURL,
		Origin: sourcepostgres.Origin{FinalURL: &finalURL},
	}}).Register(mux)
	recorder := httptest.NewRecorder()

	mux.ServeHTTP(recorder, httptest.NewRequest(
		http.MethodGet,
		"/api/v1/corpora/"+corpusID.String()+"/sources/"+sourceID.String()+"/origin/url",
		nil,
	))

	if recorder.Code != http.StatusOK || !strings.Contains(recorder.Body.String(), `"finalUrl":"https://example.org/final"`) {
		t.Fatalf("response = %d/%s, want URL metadata", recorder.Code, recorder.Body.String())
	}
}

func TestCreateURLAcceptsPendingSource(t *testing.T) {
	corpusID := uuid.New()
	commands := &fakeCommands{}
	mux := http.NewServeMux()
	NewHandler(&fakeReader{}, commands).Register(mux)
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(
		http.MethodPost,
		"/api/v1/corpora/"+corpusID.String()+"/sources/url",
		strings.NewReader(`{"title":"Official law","url":"https://example.org/law"}`),
	)

	mux.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusAccepted || commands.title != "Official law" {
		t.Fatalf("response/command = %d/%s, want accepted source", recorder.Code, commands.title)
	}
	if !strings.Contains(recorder.Body.String(), `"processingStatus":"pending"`) {
		t.Fatalf("response = %s, want pending lifecycle", recorder.Body.String())
	}
}

func TestCreateURLMapsDuplicateWithoutDisclosingAnotherSource(t *testing.T) {
	commands := &fakeCommands{err: domain.ErrDuplicateSource}
	mux := http.NewServeMux()
	NewHandler(&fakeReader{}, commands).Register(mux)
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(
		http.MethodPost,
		"/api/v1/corpora/"+uuid.NewString()+"/sources/url",
		strings.NewReader(`{"title":"Duplicate","url":"https://example.org/law"}`),
	)

	mux.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusConflict || !strings.Contains(recorder.Body.String(), `"code":"duplicate_source"`) {
		t.Fatalf("response = %d/%s, want safe duplicate conflict", recorder.Code, recorder.Body.String())
	}
}

func TestCreatePDFStreamsMultipartAndReturnsPendingSource(t *testing.T) {
	corpusID := uuid.New()
	commands := &fakeCommands{}
	mux := http.NewServeMux()
	NewHandler(&fakeReader{}, commands).Register(mux)
	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	if err := writer.WriteField("title", "Official PDF"); err != nil {
		t.Fatalf("WriteField() error = %v", err)
	}
	file, err := writer.CreateFormFile("file", "official.pdf")
	if err != nil {
		t.Fatalf("CreateFormFile() error = %v", err)
	}
	if _, err := file.Write([]byte("%PDF-generated-test")); err != nil {
		t.Fatalf("file Write() error = %v", err)
	}
	if err := writer.Close(); err != nil {
		t.Fatalf("multipart Close() error = %v", err)
	}
	request := httptest.NewRequest(
		http.MethodPost, "/api/v1/corpora/"+corpusID.String()+"/sources/pdf", &body,
	)
	request.Header.Set("Content-Type", writer.FormDataContentType())
	recorder := httptest.NewRecorder()

	mux.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusAccepted || !strings.Contains(recorder.Body.String(), `"kind":"pdf"`) {
		t.Fatalf("response = %d/%s, want accepted PDF", recorder.Code, recorder.Body.String())
	}
}

func TestRetryAcceptsCurrentVersionAndReturnsPendingSource(t *testing.T) {
	corpusID, sourceID := uuid.New(), uuid.New()
	mux := http.NewServeMux()
	NewHandler(&fakeReader{}, &fakeCommands{}).Register(mux)
	recorder := httptest.NewRecorder()

	mux.ServeHTTP(recorder, httptest.NewRequest(
		http.MethodPost,
		"/api/v1/corpora/"+corpusID.String()+"/sources/"+sourceID.String()+"/retry",
		strings.NewReader(`{"version":3}`),
	))

	if recorder.Code != http.StatusAccepted || !strings.Contains(recorder.Body.String(), `"version":4`) {
		t.Fatalf("response = %d/%s, want accepted versioned retry", recorder.Code, recorder.Body.String())
	}
}

func TestReprocessAcceptsReadySource(t *testing.T) {
	corpusID, sourceID := uuid.New(), uuid.New()
	mux := http.NewServeMux()
	NewHandler(&fakeReader{}, &fakeCommands{}).Register(mux)

	recorder := serveSourceRequest(
		mux, http.MethodPost,
		"/api/v1/corpora/"+corpusID.String()+"/sources/"+sourceID.String()+"/reprocess",
		`{"version":3}`,
	)

	if recorder.Code != http.StatusAccepted || !strings.Contains(recorder.Body.String(), `"version":4`) {
		t.Fatalf("response = %d/%s, want accepted reprocessing", recorder.Code, recorder.Body.String())
	}
}

func TestPDFOriginIsDeliveredAsSafeAttachment(t *testing.T) {
	corpusID, sourceID := uuid.New(), uuid.New()
	reader := &fakeReader{pdfOrigin: sourcepostgres.PDFOriginRecord{
		DeliveryFilename: "official.pdf", MediaType: "application/pdf", Content: []byte("%PDF-safe"),
	}}
	mux := http.NewServeMux()
	NewHandler(reader).Register(mux)

	recorder := serveSourceRequest(
		mux, http.MethodGet,
		"/api/v1/corpora/"+corpusID.String()+"/sources/"+sourceID.String()+"/origin/pdf", "",
	)

	if recorder.Code != http.StatusOK || recorder.Body.String() != "%PDF-safe" {
		t.Fatalf("response = %d/%q, want preserved PDF", recorder.Code, recorder.Body.String())
	}
	if !strings.Contains(recorder.Header().Get("Content-Disposition"), "official.pdf") {
		t.Fatalf("Content-Disposition = %q, want safe filename", recorder.Header().Get("Content-Disposition"))
	}
}

func TestGenericOriginRedirectsToPreservedOfficialURL(t *testing.T) {
	corpusID, sourceID := uuid.New(), uuid.New()
	reader := &fakeReader{
		pdfError:  sourcepostgres.ErrNotFound,
		urlOrigin: sourcepostgres.URLOriginRecord{URL: "https://example.org/final"},
	}
	mux := http.NewServeMux()
	NewHandler(reader).Register(mux)

	recorder := serveSourceRequest(
		mux, http.MethodGet,
		"/api/v1/corpora/"+corpusID.String()+"/sources/"+sourceID.String()+"/origin", "",
	)

	if recorder.Code != http.StatusSeeOther || recorder.Header().Get("Location") != "https://example.org/final" {
		t.Fatalf("response = %d/%s, want safe official redirect", recorder.Code, recorder.Header().Get("Location"))
	}
}

func TestSourceCommandsMapDomainFailures(t *testing.T) {
	tests := []struct {
		name   string
		err    error
		status int
		code   string
	}{
		{name: "invalid", err: domain.ErrInvalidInput, status: http.StatusBadRequest, code: "invalid_input"},
		{name: "limit", err: domain.ErrSourceLimit, status: http.StatusConflict, code: "payload_too_large"},
		{name: "missing corpus", err: domain.ErrCorpusUnavailable, status: http.StatusNotFound, code: "not_found"},
		{name: "unsupported", err: domain.ErrUnsupportedContent, status: http.StatusBadRequest, code: "unsupported_content"},
		{name: "stale", err: sourcepostgres.ErrStaleState, status: http.StatusConflict, code: "stale_state"},
		{name: "transition", err: domain.ErrInvalidTransition, status: http.StatusConflict, code: "unavailable"},
		{name: "unavailable", err: errors.New("database unavailable"), status: http.StatusServiceUnavailable, code: "unavailable"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			mux := http.NewServeMux()
			NewHandler(&fakeReader{}, &fakeCommands{err: test.err}).Register(mux)
			recorder := serveSourceRequest(
				mux, http.MethodPost, "/api/v1/corpora/"+uuid.NewString()+"/sources/url",
				`{"title":"Official law","url":"https://example.org/law"}`,
			)
			if recorder.Code != test.status || !strings.Contains(recorder.Body.String(), `"code":"`+test.code+`"`) {
				t.Fatalf("response = %d/%s, want %d/%s", recorder.Code, recorder.Body.String(), test.status, test.code)
			}
		})
	}
}

func TestSourceReadsMapInvalidAndUnavailableRequests(t *testing.T) {
	tests := []struct {
		name   string
		reader *fakeReader
		path   string
		status int
	}{
		{name: "invalid corpus", reader: &fakeReader{}, path: "/api/v1/corpora/not-a-uuid/sources", status: http.StatusBadRequest},
		{name: "unavailable list", reader: &fakeReader{err: errors.New("database unavailable")}, path: "/api/v1/corpora/" + uuid.NewString() + "/sources", status: http.StatusServiceUnavailable},
		{name: "missing source", reader: &fakeReader{err: sourcepostgres.ErrNotFound}, path: "/api/v1/corpora/" + uuid.NewString() + "/sources/" + uuid.NewString(), status: http.StatusNotFound},
		{name: "unavailable source", reader: &fakeReader{err: errors.New("database unavailable")}, path: "/api/v1/corpora/" + uuid.NewString() + "/sources/" + uuid.NewString(), status: http.StatusServiceUnavailable},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			mux := http.NewServeMux()
			NewHandler(test.reader).Register(mux)
			recorder := serveSourceRequest(mux, http.MethodGet, test.path, "")
			if recorder.Code != test.status {
				t.Fatalf("response status = %d, want %d", recorder.Code, test.status)
			}
		})
	}
}

func serveSourceRequest(handler http.Handler, method string, path string, body string) *httptest.ResponseRecorder {
	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, httptest.NewRequest(method, path, strings.NewReader(body)))
	return recorder
}
