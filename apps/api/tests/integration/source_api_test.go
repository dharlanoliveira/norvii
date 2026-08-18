//go:build integration

package integration_test

import (
	"bytes"
	"context"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/dharlanoliveira/norvii/apps/api/internal/platform/persistence"
	sourceapplication "github.com/dharlanoliveira/norvii/apps/api/internal/source/application"
	sourcehttp "github.com/dharlanoliveira/norvii/apps/api/internal/source/http"
	sourcepostgres "github.com/dharlanoliveira/norvii/apps/api/internal/source/postgres"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

func TestPDFSourceAPIStoresAndDeliversCorpusScopedBytes(t *testing.T) {
	configuration, err := persistence.LoadConfig(os.LookupEnv)
	if err != nil {
		t.Fatalf("LoadConfig() error = %v", err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), configuration.Timeout)
	t.Cleanup(cancel)
	pool, err := persistence.OpenPostgresPool(ctx, configuration.Postgres)
	if err != nil {
		t.Fatalf("OpenPostgresPool() error = %v", err)
	}
	t.Cleanup(pool.Close)
	corpusID := uuid.New()
	if _, err := pool.Exec(ctx, `
		INSERT INTO corpora (id, name, description, language, jurisdiction)
		VALUES ($1, 'PDF integration', 'Integration test corpus.', 'en', 'Test jurisdiction')`,
		corpusID,
	); err != nil {
		t.Fatalf("insert test corpus error = %v", err)
	}
	t.Cleanup(func() {
		_, _ = pool.Exec(context.Background(), "DELETE FROM ingestion_work WHERE corpus_id = $1", corpusID)
		_, _ = pool.Exec(context.Background(), "DELETE FROM pdf_origins WHERE corpus_id = $1", corpusID)
		_, _ = pool.Exec(context.Background(), "DELETE FROM sources WHERE corpus_id = $1", corpusID)
		_, _ = pool.Exec(context.Background(), "DELETE FROM corpora WHERE id = $1", corpusID)
	})
	repository := sourcepostgres.NewRepository(pool)
	service := sourceapplication.NewService(repository, uuid.New, time.Now)
	mux := http.NewServeMux()
	sourcehttp.NewHandler(repository, service).Register(mux)
	content := []byte("%PDF-generated-integration-test")

	created := postPDFSource(t, mux, corpusID, "Official PDF", `..\unsafe\official.pdf`, content)
	if created.Code != http.StatusAccepted {
		t.Fatalf("PDF response = %d/%s, want accepted", created.Code, created.Body.String())
	}
	var sourceID uuid.UUID
	if err := pool.QueryRow(ctx, "SELECT id FROM sources WHERE corpus_id = $1 AND kind = 'pdf'", corpusID).Scan(&sourceID); err != nil {
		t.Fatalf("query PDF source error = %v", err)
	}
	download := httptest.NewRecorder()
	mux.ServeHTTP(download, httptest.NewRequest(
		http.MethodGet,
		"/api/v1/corpora/"+corpusID.String()+"/sources/"+sourceID.String()+"/origin",
		nil,
	))
	if download.Code != http.StatusOK || !bytes.Equal(download.Body.Bytes(), content) {
		t.Fatalf("download = %d/%q, want preserved PDF", download.Code, download.Body.Bytes())
	}
	if !strings.Contains(download.Header().Get("Content-Disposition"), "official.pdf") {
		t.Fatalf("Content-Disposition = %q, want safe filename", download.Header().Get("Content-Disposition"))
	}
	duplicate := postPDFSource(t, mux, corpusID, "Duplicate PDF", "copy.pdf", content)
	if duplicate.Code != http.StatusConflict || !strings.Contains(duplicate.Body.String(), `"code":"duplicate_source"`) {
		t.Fatalf("duplicate response = %d/%s, want safe conflict", duplicate.Code, duplicate.Body.String())
	}
	foreign := httptest.NewRecorder()
	mux.ServeHTTP(foreign, httptest.NewRequest(
		http.MethodGet,
		"/api/v1/corpora/"+uuid.NewString()+"/sources/"+sourceID.String()+"/origin",
		nil,
	))
	if foreign.Code != http.StatusNotFound {
		t.Fatalf("foreign download status = %d, want 404", foreign.Code)
	}
}

func TestURLSourceAPICommitsOriginAndWorkWithoutCrossCorpusDisclosure(t *testing.T) {
	configuration, err := persistence.LoadConfig(os.LookupEnv)
	if err != nil {
		t.Fatalf("LoadConfig() error = %v", err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), configuration.Timeout)
	t.Cleanup(cancel)
	pool, err := persistence.OpenPostgresPool(ctx, configuration.Postgres)
	if err != nil {
		t.Fatalf("OpenPostgresPool() error = %v", err)
	}
	t.Cleanup(pool.Close)
	corpusIDs := insertURLTestCorpora(t, ctx, pool)
	t.Cleanup(func() {
		_, _ = pool.Exec(context.Background(), "DELETE FROM ingestion_work WHERE corpus_id = ANY($1)", corpusIDs)
		_, _ = pool.Exec(context.Background(), "DELETE FROM url_origins WHERE corpus_id = ANY($1)", corpusIDs)
		_, _ = pool.Exec(context.Background(), "DELETE FROM sources WHERE corpus_id = ANY($1)", corpusIDs)
		_, _ = pool.Exec(context.Background(), "DELETE FROM corpora WHERE id = ANY($1)", corpusIDs)
	})

	repository := sourcepostgres.NewRepository(pool)
	service := sourceapplication.NewService(repository, uuid.New, time.Now)
	mux := http.NewServeMux()
	sourcehttp.NewHandler(repository, service).Register(mux)

	requireResponse(t, postURLSource(t, mux, corpusIDs[0], "Official law", "https://EXAMPLE.org:443/law?b=2&a=1"), http.StatusAccepted, `"processingStatus":"pending"`)
	requireResponse(t, postURLSource(t, mux, corpusIDs[0], "Duplicate", "https://example.org/law?a=1&b=2#section"), http.StatusConflict, `"code":"duplicate_source"`)
	requireResponse(t, postURLSource(t, mux, corpusIDs[1], "Independent law", "https://example.org/law?a=1&b=2"), http.StatusAccepted, "")
	assertURLPersistence(t, ctx, pool, corpusIDs)
	lifecycleSourceID := prepareFailedURLSource(t, ctx, pool, corpusIDs[0])
	retry := postLifecycle(t, mux, corpusIDs[0], lifecycleSourceID, "retry", 2)
	if retry.Code != http.StatusAccepted || !strings.Contains(retry.Body.String(), `"processingStatus":"pending"`) {
		t.Fatalf("retry response = %d/%s, want pending", retry.Code, retry.Body.String())
	}
	stale := postLifecycle(t, mux, corpusIDs[0], lifecycleSourceID, "retry", 2)
	if stale.Code != http.StatusConflict || !strings.Contains(stale.Body.String(), `"code":"stale_state"`) {
		t.Fatalf("stale response = %d/%s, want stale conflict", stale.Code, stale.Body.String())
	}
}

func insertURLTestCorpora(t *testing.T, ctx context.Context, pool *pgxpool.Pool) []uuid.UUID {
	t.Helper()
	corpusIDs := []uuid.UUID{uuid.New(), uuid.New()}
	for index, corpusID := range corpusIDs {
		if _, err := pool.Exec(ctx, `
			INSERT INTO corpora (id, name, description, language, jurisdiction)
			VALUES ($1, $2, 'Integration test corpus.', 'en', 'Test jurisdiction')`,
			corpusID, "URL source integration "+string(rune('A'+index)),
		); err != nil {
			t.Fatalf("insert test corpus error = %v", err)
		}
	}
	return corpusIDs
}

func assertURLPersistence(t *testing.T, ctx context.Context, pool *pgxpool.Pool, corpusIDs []uuid.UUID) {
	t.Helper()
	for _, corpusID := range corpusIDs {
		var sources, origins, work int
		if err := pool.QueryRow(ctx, `
			SELECT count(DISTINCT s.id), count(DISTINCT u.source_id), count(DISTINCT w.id)
			FROM sources s
			LEFT JOIN url_origins u ON u.corpus_id = s.corpus_id AND u.source_id = s.id
			LEFT JOIN ingestion_work w ON w.corpus_id = s.corpus_id AND w.source_id = s.id
			WHERE s.corpus_id = $1`, corpusID,
		).Scan(&sources, &origins, &work); err != nil {
			t.Fatalf("query committed URL slice error = %v", err)
		}
		if sources != 1 || origins != 1 || work != 1 {
			t.Fatalf("committed counts = %d/%d/%d, want 1/1/1", sources, origins, work)
		}
	}
}

func prepareFailedURLSource(t *testing.T, ctx context.Context, pool *pgxpool.Pool, corpusID uuid.UUID) uuid.UUID {
	t.Helper()
	var sourceID uuid.UUID
	if err := pool.QueryRow(ctx,
		"SELECT id FROM sources WHERE corpus_id = $1 AND kind = 'url'", corpusID,
	).Scan(&sourceID); err != nil {
		t.Fatalf("query lifecycle source error = %v", err)
	}
	if _, err := pool.Exec(ctx,
		"UPDATE ingestion_work SET status = 'failed' WHERE source_id = $1", sourceID,
	); err != nil {
		t.Fatalf("prepare failed work error = %v", err)
	}
	if _, err := pool.Exec(ctx, `
		UPDATE sources
		SET processing_status = 'failed', latest_failure_category = 'acquisition_failed', version = 2
		WHERE id = $1`, sourceID); err != nil {
		t.Fatalf("prepare failed source error = %v", err)
	}
	return sourceID
}

func requireResponse(t *testing.T, response *httptest.ResponseRecorder, status int, bodyFragment string) {
	t.Helper()
	if response.Code != status || !strings.Contains(response.Body.String(), bodyFragment) {
		t.Fatalf("response = %d/%s, want status %d containing %q", response.Code, response.Body.String(), status, bodyFragment)
	}
}

func postURLSource(
	t *testing.T,
	handler http.Handler,
	corpusID uuid.UUID,
	title string,
	url string,
) *httptest.ResponseRecorder {
	t.Helper()
	recorder := httptest.NewRecorder()
	body := `{"title":"` + title + `","url":"` + url + `"}`
	handler.ServeHTTP(recorder, httptest.NewRequest(
		http.MethodPost, "/api/v1/corpora/"+corpusID.String()+"/sources/url", strings.NewReader(body),
	))
	return recorder
}

func postPDFSource(
	t *testing.T,
	handler http.Handler,
	corpusID uuid.UUID,
	title string,
	filename string,
	content []byte,
) *httptest.ResponseRecorder {
	t.Helper()
	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	if err := writer.WriteField("title", title); err != nil {
		t.Fatalf("WriteField() error = %v", err)
	}
	part, err := writer.CreateFormFile("file", filename)
	if err != nil {
		t.Fatalf("CreateFormFile() error = %v", err)
	}
	if _, err := part.Write(content); err != nil {
		t.Fatalf("PDF Write() error = %v", err)
	}
	if err := writer.Close(); err != nil {
		t.Fatalf("multipart Close() error = %v", err)
	}
	request := httptest.NewRequest(
		http.MethodPost, "/api/v1/corpora/"+corpusID.String()+"/sources/pdf", &body,
	)
	request.Header.Set("Content-Type", writer.FormDataContentType())
	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, request)
	return recorder
}

func postLifecycle(
	t *testing.T,
	handler http.Handler,
	corpusID uuid.UUID,
	sourceID uuid.UUID,
	action string,
	version int,
) *httptest.ResponseRecorder {
	t.Helper()
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(
		http.MethodPost,
		"/api/v1/corpora/"+corpusID.String()+"/sources/"+sourceID.String()+"/"+action,
		strings.NewReader(`{"version":`+strconv.Itoa(version)+`}`),
	)
	handler.ServeHTTP(recorder, request)
	return recorder
}
