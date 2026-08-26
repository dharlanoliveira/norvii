//go:build integration

package integration_test

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	evaluationapplication "github.com/dharlanoliveira/norvii/apps/api/internal/evaluation/application"
	evaluationhttp "github.com/dharlanoliveira/norvii/apps/api/internal/evaluation/http"
	evaluationpostgres "github.com/dharlanoliveira/norvii/apps/api/internal/evaluation/postgres"
	evaluationcontract "github.com/dharlanoliveira/norvii/apps/api/tests/contract"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

const evaluationHTTPFingerprint = "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"

func TestEvaluationHTTPPersistsOnlyAuthorizedPreflightApprovedRuns(t *testing.T) {
	ctx, connection := openEvaluationSchemaConnection(t)

	t.Run("denies every route before persistence", func(t *testing.T) {
		assertEvaluationHTTPUnauthorizedRoutes(t, ctx, connection)
	})
	t.Run("rejected preflight leaves no run and exposes a safe error", func(t *testing.T) {
		assertEvaluationHTTPRejectedPreflight(t, ctx, connection)
	})
	t.Run("stores and reads the historical execution identity unchanged", func(t *testing.T) {
		assertEvaluationHTTPHistoricalIdentity(t, ctx, connection)
	})
}

func assertEvaluationHTTPUnauthorizedRoutes(t *testing.T, ctx context.Context, connection *pgx.Conn) {
	t.Helper()
	transaction := beginEvaluationSchemaTransaction(t, ctx, connection)
	defer rollbackEvaluationSchemaTransaction(t, ctx, transaction)
	fixture := seedEvaluationHTTPFixture(t, ctx, transaction)
	mux := evaluationHTTPMux(transaction)
	body := evaluationHTTPStartBody(fixture)
	requests := []*http.Request{
		httptest.NewRequest(http.MethodPost, "/api/v1/evaluations", strings.NewReader(body)),
		httptest.NewRequest(http.MethodGet, "/api/v1/evaluations/"+uuid.NewString(), nil),
		httptest.NewRequest(http.MethodGet, "/api/v1/evaluations/"+uuid.NewString()+"/cases/"+uuid.NewString(), nil),
	}
	for _, request := range requests {
		response := httptest.NewRecorder()
		mux.ServeHTTP(response, request)
		assertEvaluationHTTPUnauthorizedResponse(t, request, response)
	}
	assertEvaluationRunCount(t, ctx, transaction, 0)
}

func assertEvaluationHTTPUnauthorizedResponse(t *testing.T, request *http.Request, response *httptest.ResponseRecorder) {
	t.Helper()
	if response.Code != http.StatusUnauthorized || response.Header().Get("WWW-Authenticate") != "Bearer" {
		t.Fatalf("%s %s = %d %q, want denied maintainer route", request.Method, request.URL.Path, response.Code, response.Body.String())
	}
	assertBoundedEvaluationHTTPError(t, response.Body.String())
}

func assertEvaluationHTTPRejectedPreflight(t *testing.T, ctx context.Context, connection *pgx.Conn) {
	t.Helper()
	transaction := beginEvaluationSchemaTransaction(t, ctx, connection)
	defer rollbackEvaluationSchemaTransaction(t, ctx, transaction)
	fixture := seedEvaluationHTTPFixture(t, ctx, transaction)
	if _, err := transaction.Exec(ctx, `DELETE FROM corpus_snapshot_documents WHERE snapshot_id = $1`, fixture.snapshotID); err != nil {
		t.Fatalf("remove required snapshot membership: %v", err)
	}
	response := authorizedEvaluationHTTPPost(evaluationHTTPMux(transaction), evaluationHTTPStartBody(fixture))
	if response.Code != http.StatusUnprocessableEntity {
		t.Fatalf("rejected preflight status = %d: %s", response.Code, response.Body.String())
	}
	assertBoundedEvaluationHTTPError(t, response.Body.String())
	assertEvaluationRunCount(t, ctx, transaction, 0)
}

func assertEvaluationHTTPHistoricalIdentity(t *testing.T, ctx context.Context, connection *pgx.Conn) {
	t.Helper()
	transaction := beginEvaluationSchemaTransaction(t, ctx, connection)
	defer rollbackEvaluationSchemaTransaction(t, ctx, transaction)
	fixture := seedEvaluationHTTPFixture(t, ctx, transaction)
	mux := evaluationHTTPMux(transaction)
	response := authorizedEvaluationHTTPPost(mux, evaluationHTTPStartBody(fixture))
	if response.Code != http.StatusCreated {
		t.Fatalf("start evaluation status = %d: %s", response.Code, response.Body.String())
	}
	var started struct {
		RunID uuid.UUID `json:"runId"`
	}
	if err := json.Unmarshal(response.Body.Bytes(), &started); err != nil || started.RunID == uuid.Nil {
		t.Fatalf("decode started run: %v, payload=%s", err, response.Body.String())
	}
	assertPersistedEvaluationHTTPIdentity(t, ctx, transaction, started.RunID)
	assertHistoricalEvaluationHTTPResponse(t, mux, started.RunID)
}

func assertPersistedEvaluationHTTPIdentity(t *testing.T, ctx context.Context, transaction pgx.Tx, runID uuid.UUID) {
	t.Helper()
	var agentBuild, chatModel, embeddingModel, fingerprint string
	if err := transaction.QueryRow(ctx, `
		SELECT agent_build, chat_model_identity, embedding_model_identity, retrieval_configuration_fingerprint
		FROM evaluation_run WHERE id = $1`, runID,
	).Scan(&agentBuild, &chatModel, &embeddingModel, &fingerprint); err != nil {
		t.Fatalf("read persisted evaluation identity: %v", err)
	}
	if agentBuild != "historical-agent" || chatModel != "historical-chat" || embeddingModel != "historical-embedding" || fingerprint != evaluationHTTPFingerprint {
		t.Fatalf("persisted execution identity = %q %q %q %q", agentBuild, chatModel, embeddingModel, fingerprint)
	}
}

func assertHistoricalEvaluationHTTPResponse(t *testing.T, mux *http.ServeMux, runID uuid.UUID) {
	t.Helper()
	getRequest := httptest.NewRequest(http.MethodGet, "/api/v1/evaluations/"+runID.String(), nil)
	getRequest.Header.Set("Authorization", "Bearer test-maintainer")
	getResponse := httptest.NewRecorder()
	mux.ServeHTTP(getResponse, getRequest)
	if getResponse.Code != http.StatusOK || !strings.Contains(getResponse.Body.String(), "historical-agent") ||
		!strings.Contains(getResponse.Body.String(), "historical-chat") || !strings.Contains(getResponse.Body.String(), "historical-embedding") ||
		!strings.Contains(getResponse.Body.String(), evaluationHTTPFingerprint) {
		t.Fatalf("historical run inspection = %d %s", getResponse.Code, getResponse.Body.String())
	}
}

func TestEvaluationDatasetInspectionHTTPPreservesPersistedStarterMetadataAndPreflightSafety(t *testing.T) {
	ctx, connection := openEvaluationSchemaConnection(t)
	transaction := beginEvaluationSchemaTransaction(t, ctx, connection)
	defer rollbackEvaluationSchemaTransaction(t, ctx, transaction)

	fixture := seedEvaluationHTTPFixture(t, ctx, transaction)
	insertEvaluationHTTPStarterPair(t, ctx, transaction, fixture)
	mux := evaluationHTTPMux(transaction)

	assertEvaluationHTTPDatasetCatalogList(t, mux, fixture)
	assertEvaluationHTTPDatasetCatalogDetail(t, mux, fixture)
	assertEvaluationHTTPDatasetPreflightSafety(t, ctx, transaction, mux, fixture)
}

func assertEvaluationHTTPDatasetCatalogList(t *testing.T, mux *http.ServeMux, fixture evaluationPreflightFixture) {
	t.Helper()
	listRequest := httptest.NewRequest(http.MethodGet, "/api/v1/evaluation-datasets", nil)
	listRequest.Header.Set("Authorization", "Bearer test-maintainer")
	listResponse := httptest.NewRecorder()
	mux.ServeHTTP(listResponse, listRequest)
	if listResponse.Code != http.StatusOK || strings.Contains(listResponse.Body.String(), "starters") || strings.Contains(listResponse.Body.String(), "Synthetic question") {
		t.Fatalf("catalog list = %d %s", listResponse.Code, listResponse.Body.String())
	}
	var catalog []evaluationcontract.DatasetCatalogResponse
	decodeEvaluationHTTPResponse(t, listResponse, &catalog)
	for index := range catalog {
		if catalog[index].Revision.ID == fixture.revisionID.String() {
			entry := catalog[index]
			if entry.Review == nil || entry.Review.Decision != "approved" || entry.Review.PublicationState != "available" {
				t.Fatalf("catalog response = %#v, want persisted fixture revision and review", catalog)
			}
			return
		}
	}
	t.Fatalf("catalog response = %#v, want persisted fixture revision and review", catalog)
}

func assertEvaluationHTTPDatasetCatalogDetail(t *testing.T, mux *http.ServeMux, fixture evaluationPreflightFixture) {
	t.Helper()
	detailRequest := httptest.NewRequest(http.MethodGet, "/api/v1/evaluation-datasets/"+fixture.revisionID.String(), nil)
	detailRequest.Header.Set("Authorization", "Bearer test-maintainer")
	detailResponse := httptest.NewRecorder()
	mux.ServeHTTP(detailResponse, detailRequest)
	if detailResponse.Code != http.StatusOK || strings.Contains(detailResponse.Body.String(), "Synthetic question") || strings.Contains(detailResponse.Body.String(), "Synthetic answer") {
		t.Fatalf("catalog detail = %d %s", detailResponse.Code, detailResponse.Body.String())
	}
	var detail evaluationcontract.DatasetDetailResponse
	decodeEvaluationHTTPResponse(t, detailResponse, &detail)
	if detail.Revision.ID != fixture.revisionID.String() || len(detail.Starters) != 2 {
		t.Fatalf("catalog detail = %#v, want revision and two persisted starters", detail)
	}
	startersByLanguage := make(map[string]evaluationcontract.DatasetStarterResponse, len(detail.Starters))
	for _, starter := range detail.Starters {
		startersByLanguage[starter.QueryLanguage] = starter
	}
	for language, caseID := range map[string]string{"en": fixture.caseID.String(), "pt": fixture.pairedCaseID.String()} {
		starter, found := startersByLanguage[language]
		if !found || starter.CaseID != caseID || starter.Rank != 1 || !starter.ReviewEligible {
			t.Fatalf("persisted %s starter = %#v, want case=%s rank=1 review eligible", language, starter, caseID)
		}
	}
}

func assertEvaluationHTTPDatasetPreflightSafety(t *testing.T, ctx context.Context, transaction pgx.Tx, mux *http.ServeMux, fixture evaluationPreflightFixture) {
	t.Helper()
	preflightPath := "/api/v1/evaluation-datasets/" + fixture.revisionID.String() + "/preflight?corpusId=" + fixture.corpusID.String() + "&snapshotId=" + fixture.snapshotID.String()
	preflightResponse := evaluationHTTPGet(mux, preflightPath)
	if preflightResponse.Code != http.StatusOK {
		t.Fatalf("successful preflight = %d %s", preflightResponse.Code, preflightResponse.Body.String())
	}
	var preflight evaluationcontract.DatasetPreflightResponse
	decodeEvaluationHTTPResponse(t, preflightResponse, &preflight)
	if !preflight.Compatible || len(preflight.MissingRequirements) != 0 || preflight.DatasetRevisionID != fixture.revisionID.String() || preflight.CorpusID != fixture.corpusID.String() || preflight.SnapshotID != fixture.snapshotID.String() {
		t.Fatalf("successful preflight payload = %#v", preflight)
	}
	assertEvaluationRunCount(t, ctx, transaction, 0)

	absentResponse := evaluationHTTPGet(mux, "/api/v1/evaluation-datasets/"+uuid.NewString()+"/preflight?corpusId="+fixture.corpusID.String()+"&snapshotId="+fixture.snapshotID.String())
	if absentResponse.Code != http.StatusNotFound {
		t.Fatalf("absent preflight = %d %s", absentResponse.Code, absentResponse.Body.String())
	}
	var absent evaluationcontract.DatasetPreflightErrorResponse
	decodeEvaluationHTTPResponse(t, absentResponse, &absent)
	if absent.Error.Code != "not_found" || len(absent.Error.MissingRequirements) != 0 {
		t.Fatalf("absent preflight payload = %#v", absent)
	}
	assertEvaluationRunCount(t, ctx, transaction, 0)

	if _, err := transaction.Exec(ctx, `DELETE FROM corpus_snapshot_documents WHERE snapshot_id = $1`, fixture.snapshotID); err != nil {
		t.Fatalf("remove required snapshot membership: %v", err)
	}
	rejectedResponse := evaluationHTTPGet(mux, preflightPath)
	if rejectedResponse.Code != http.StatusUnprocessableEntity || strings.Contains(rejectedResponse.Body.String(), "Synthetic question") || strings.Contains(rejectedResponse.Body.String(), "Synthetic answer") {
		t.Fatalf("rejected preflight = %d %s", rejectedResponse.Code, rejectedResponse.Body.String())
	}
	var rejected evaluationcontract.DatasetPreflightErrorResponse
	decodeEvaluationHTTPResponse(t, rejectedResponse, &rejected)
	if rejected.Error.Code != "snapshot_incompatible" || len(rejected.Error.MissingRequirements) != 2 || rejected.Error.MissingRequirements[0].Reason != "The required source is not a member of the selected snapshot." || rejected.Error.MissingRequirements[1].Reason != "The locator did not resolve uniquely." {
		t.Fatalf("rejected preflight payload = %#v, want source and locator diagnostics", rejected)
	}
	assertBoundedEvaluationHTTPError(t, rejectedResponse.Body.String())
	assertEvaluationRunCount(t, ctx, transaction, 0)
}

func evaluationHTTPGet(mux *http.ServeMux, path string) *httptest.ResponseRecorder {
	request := httptest.NewRequest(http.MethodGet, path, nil)
	request.Header.Set("Authorization", "Bearer test-maintainer")
	response := httptest.NewRecorder()
	mux.ServeHTTP(response, request)
	return response
}

func seedEvaluationHTTPFixture(t *testing.T, ctx context.Context, transaction pgx.Tx) evaluationPreflightFixture {
	t.Helper()
	fixture := seedEvaluationPreflightFixture(t, ctx, transaction, "")
	insertEvaluationPreflightPublication(t, ctx, transaction, fixture, "approved", "available")
	return fixture
}

func evaluationHTTPMux(transaction pgx.Tx) *http.ServeMux {
	repository := evaluationpostgres.NewRepository(transaction)
	preflight := evaluationapplication.NewPreflightService(repository)
	runnable := evaluationapplication.RunnableConfiguration{
		Retrieval: evaluationapplication.RetrievalConfiguration{Strategy: "vector", Fingerprint: evaluationHTTPFingerprint},
		Identity: evaluationapplication.ExecutionIdentity{
			AgentBuild: "historical-agent", ChatModelIdentity: "historical-chat", EmbeddingModelIdentity: "historical-embedding",
		},
	}
	mux := http.NewServeMux()
	evaluationhttp.NewHandler(
		evaluationapplication.NewRunService(preflight, repository, runnable),
		evaluationhttp.NewTokenAuthorizer("test-maintainer"),
		evaluationapplication.NewCatalogService(repository, preflight),
	).Register(mux)
	return mux
}

func insertEvaluationHTTPStarterPair(t *testing.T, ctx context.Context, transaction pgx.Tx, fixture evaluationPreflightFixture) {
	t.Helper()
	if _, err := transaction.Exec(ctx, `
		INSERT INTO evaluation_dataset_starter_case (
			id, dataset_revision_id, corpus_id, evaluation_case_id, rank, query_language, case_checksum, is_review_eligible
		) VALUES
			($1, $2, $3, $4, 1, 'en', $5, true),
			($6, $2, $3, $7, 1, 'pt', $8, true)`,
		uuid.New(), fixture.revisionID, fixture.corpusID, fixture.caseID, fixture.caseChecksum,
		uuid.New(), fixture.pairedCaseID, fixture.pairedCaseChecksum,
	); err != nil {
		t.Fatalf("insert evaluation starter pair: %v", err)
	}
}

func evaluationHTTPStartBody(fixture evaluationPreflightFixture) string {
	return `{"datasetRevisionId":"` + fixture.revisionID.String() + `","corpusId":"` + fixture.corpusID.String() +
		`","snapshotId":"` + fixture.snapshotID.String() + `","configuration":{"strategy":"vector","fingerprint":"` + evaluationHTTPFingerprint + `"}}`
}

func authorizedEvaluationHTTPPost(mux *http.ServeMux, body string) *httptest.ResponseRecorder {
	request := httptest.NewRequest(http.MethodPost, "/api/v1/evaluations", strings.NewReader(body))
	request.Header.Set("Authorization", "Bearer test-maintainer")
	response := httptest.NewRecorder()
	mux.ServeHTTP(response, request)
	return response
}

func assertEvaluationRunCount(t *testing.T, ctx context.Context, transaction pgx.Tx, want int) {
	t.Helper()
	var count int
	if err := transaction.QueryRow(ctx, `SELECT count(*) FROM evaluation_run`).Scan(&count); err != nil {
		t.Fatalf("count evaluation runs: %v", err)
	}
	if count != want {
		t.Fatalf("evaluation run count = %d, want %d", count, want)
	}
}

func assertBoundedEvaluationHTTPError(t *testing.T, body string) {
	t.Helper()
	if len(body) == 0 || len(body) > 1024 || strings.Contains(body, "providerPayload") || strings.Contains(body, "Synthetic question") {
		t.Fatalf("evaluation error is not bounded and safe: %q", body)
	}
}

func decodeEvaluationHTTPResponse(t *testing.T, response *httptest.ResponseRecorder, destination any) {
	t.Helper()
	decoder := json.NewDecoder(bytes.NewReader(response.Body.Bytes()))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(destination); err != nil {
		t.Fatalf("decode HTTP response: %v, payload=%s", err, response.Body.String())
	}
	if err := decoder.Decode(&struct{}{}); err != io.EOF {
		t.Fatalf("decode HTTP response trailing content: %v, payload=%s", err, response.Body.String())
	}
}
