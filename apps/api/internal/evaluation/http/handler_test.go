package http

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/dharlanoliveira/norvii/apps/api/internal/evaluation/application"
	"github.com/google/uuid"
)

func TestStartUsesCallerSelectedSnapshotAndReturnsContractShape(t *testing.T) {
	requestID, corpusID, datasetID, snapshotID, runID := uuid.New(), uuid.New(), uuid.New(), uuid.New(), uuid.New()
	service := &serviceStub{run: application.Run{ID: runID, State: application.RunQueued, DatasetRevisionID: datasetID,
		DatasetContentSHA256: strings.Repeat("a", 64), CorpusID: corpusID, SnapshotID: snapshotID, ScoringPolicyVersion: "v1"}}
	mux := http.NewServeMux()
	NewHandler(service, NewTokenAuthorizer("test-token")).Register(mux)
	body := `{"datasetRevisionId":"` + datasetID.String() + `","corpusId":"` + corpusID.String() + `","snapshotId":"` + snapshotID.String() + `","configuration":{"strategy":"vector","fingerprint":"` + strings.Repeat("b", 64) + `"}}`
	request := httptest.NewRequest(http.MethodPost, "/api/v1/evaluations", strings.NewReader(body))
	request.Header.Set("Authorization", "Bearer test-token")
	request.Header.Set("X-Request-ID", requestID.String())
	response := httptest.NewRecorder()

	mux.ServeHTTP(response, request)
	if response.Code != http.StatusCreated {
		t.Fatalf("status = %d, want %d: %s", response.Code, http.StatusCreated, response.Body.String())
	}
	if service.startRequest.SnapshotID != snapshotID || service.startRequest.CorpusID != corpusID || service.startRequest.DatasetRevisionID != datasetID {
		t.Fatalf("Start request = %#v, want immutable caller selections", service.startRequest)
	}
	var payload map[string]any
	if err := json.Unmarshal(response.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if payload["runId"] != runID.String() || payload["snapshotId"] != snapshotID.String() || payload["state"] != "queued" {
		t.Fatalf("start response = %#v", payload)
	}
}

func TestRunAndCaseReadsExposeHistoricalIdentityAndSafeDiagnostics(t *testing.T) {
	runID, runCaseID, datasetCaseID, corpusID, snapshotID := uuid.New(), uuid.New(), uuid.New(), uuid.New(), uuid.New()
	finished := time.Date(2026, time.August, 26, 19, 0, 0, 0, time.UTC)
	metricValue, metricNumerator, metricDenominator := 0.5, int64(1), int64(2)
	latency, inputTokens, outputTokens := int64(42), int64(7), int64(11)
	startOffset, endOffset := 0, 42
	service := &serviceStub{
		run: application.Run{ID: runID, DatasetRevisionID: uuid.New(), DatasetContentSHA256: strings.Repeat("a", 64),
			CorpusID: corpusID, SnapshotID: snapshotID, SnapshotManifestSHA256: strings.Repeat("c", 64), OrderedCaseSetSHA256: strings.Repeat("d", 64),
			Configuration: application.RetrievalConfiguration{Strategy: "vector", Fingerprint: strings.Repeat("e", 64)}, ScoringPolicyVersion: "v1",
			AgentBuild: "recorded-build", ChatModelIdentity: "recorded-chat", EmbeddingModelIdentity: "recorded-embedding", InitiatedBy: "maintainer-http", State: application.RunCompleted, CreatedAt: finished,
			Aggregate: application.RunAggregate{Total: 1, Eligible: 1, Scored: 1, Metrics: []application.RunMetric{{Name: "citation_validity", State: "scored", Value: &metricValue, Numerator: &metricNumerator, Denominator: &metricDenominator, Rationale: "One cited evidence identity resolved in the run snapshot.", ScorerVersion: "v1"}}},
			Cases:     []application.RunCaseSummary{{ID: runCaseID, DatasetCaseID: datasetCaseID, Position: 1, State: application.RunCaseCompleted, FinishedAt: &finished}}},
		caseResult: application.RunCase{RunCaseSummary: application.RunCaseSummary{ID: runCaseID, DatasetCaseID: datasetCaseID, Position: 1, State: application.RunCaseFailed, FailureCode: "provider_unavailable"},
			RunID: runID, CorpusID: corpusID, SnapshotID: snapshotID, DatasetRevisionID: uuid.New(), Question: "A maintained question?", ReferenceAnswer: "A reviewed answer.", ExpectedOutcome: "answer",
			ExpectedEvidence:    []application.EvidenceIdentity{{Kind: "expected", Position: 1, CorpusID: corpusID, SnapshotID: snapshotID, SourceID: uuid.New(), SourceRevisionID: uuid.New(), DocumentID: uuid.New(), LegalUnitID: uuid.New(), CanonicalLocator: "article:1", DisplayLocator: "Article 1", ContentSHA256: strings.Repeat("f", 64)}},
			ActualEvidence:      []application.EvidenceIdentity{{Kind: "cited", Position: 1, MarkerPosition: 1, CorpusID: corpusID, SnapshotID: snapshotID, SourceID: uuid.New(), SourceRevisionID: uuid.New(), DocumentID: uuid.New(), LegalUnitID: uuid.New(), CanonicalLocator: "article:1", DisplayLocator: "Article 1", StartOffset: &startOffset, EndOffset: &endOffset, ContentSHA256: strings.Repeat("f", 64)}},
			GraphGroundingState: "not_requested", LatencyMilliseconds: &latency, InputTokens: &inputTokens, OutputTokens: &outputTokens,
			Metrics: []application.RunMetric{{Name: "citation_validity", State: "not_scored", Rationale: "The provider failed before citation scoring.", ScorerVersion: "v1"}}},
	}
	mux := http.NewServeMux()
	NewHandler(service, NewTokenAuthorizer("test-token")).Register(mux)

	statusResponse := httptest.NewRecorder()
	statusRequest := httptest.NewRequest(http.MethodGet, "/api/v1/evaluations/"+runID.String(), nil)
	statusRequest.Header.Set("Authorization", "Bearer test-token")
	mux.ServeHTTP(statusResponse, statusRequest)
	if statusResponse.Code != http.StatusOK || !strings.Contains(statusResponse.Body.String(), snapshotID.String()) || !strings.Contains(statusResponse.Body.String(), "rationale") {
		t.Fatalf("run status = %d %s", statusResponse.Code, statusResponse.Body.String())
	}
	caseResponse := httptest.NewRecorder()
	caseRequest := httptest.NewRequest(http.MethodGet, "/api/v1/evaluations/"+runID.String()+"/cases/"+runCaseID.String(), nil)
	caseRequest.Header.Set("Authorization", "Bearer test-token")
	mux.ServeHTTP(caseResponse, caseRequest)
	for _, forbidden := range []string{"providerPayload", "prompt", "credential"} {
		if strings.Contains(caseResponse.Body.String(), forbidden) {
			t.Fatalf("case result disclosed %q: %s", forbidden, caseResponse.Body.String())
		}
	}
	for _, required := range []string{"provider_unavailable", "expectedEvidence", "actualEvidence", "citation_validity", "rationale", snapshotID.String()} {
		if !strings.Contains(caseResponse.Body.String(), required) {
			t.Fatalf("case result missing %q: %s", required, caseResponse.Body.String())
		}
	}
	if caseResponse.Code != http.StatusOK {
		t.Fatalf("case result = %d %s", caseResponse.Code, caseResponse.Body.String())
	}
	var casePayload runCaseResponse
	if err := json.Unmarshal(caseResponse.Body.Bytes(), &casePayload); err != nil {
		t.Fatalf("decode case response: %v", err)
	}
	if casePayload.GraphGroundingState != "not_requested" || casePayload.LatencyMilliseconds == nil || *casePayload.LatencyMilliseconds != latency ||
		casePayload.InputTokens == nil || *casePayload.InputTokens != inputTokens || casePayload.OutputTokens == nil || *casePayload.OutputTokens != outputTokens ||
		len(casePayload.ActualEvidence) != 1 || casePayload.ActualEvidence[0].DisplayLocator != "Article 1" ||
		casePayload.ActualEvidence[0].StartOffset == nil || *casePayload.ActualEvidence[0].StartOffset != startOffset ||
		casePayload.ActualEvidence[0].EndOffset == nil || *casePayload.ActualEvidence[0].EndOffset != endOffset {
		t.Fatalf("case response omitted durable execution detail: %#v", casePayload)
	}
	if service.getCaseRunID != runID || service.getCaseID != runCaseID {
		t.Fatalf("GetCase IDs = %s %s", service.getCaseRunID, service.getCaseID)
	}
}

func TestComparisonUsesImmutableServiceAndOmitsDeltasForNonComparableRuns(t *testing.T) {
	leftRunID, rightRunID := uuid.New(), uuid.New()
	value := 0.5
	comparison := &comparisonStub{result: application.ComparisonResult{
		State:   application.ComparisonStateComparable,
		Left:    application.ExperimentIdentity{RetrievalStrategy: "vector", RetrievalConfigurationFingerprint: strings.Repeat("a", 64), AgentBuild: "left-build", ChatModelIdentity: "left-chat", EmbeddingModelIdentity: "left-embedding"},
		Right:   application.ExperimentIdentity{RetrievalStrategy: "hybrid", RetrievalConfigurationFingerprint: strings.Repeat("b", 64), AgentBuild: "right-build", ChatModelIdentity: "right-chat", EmbeddingModelIdentity: "right-embedding"},
		Totals:  application.ComparisonTotals{LeftCases: 2, RightCases: 2, PairedCases: 2, FailedOrCancelled: 1},
		Metrics: []application.MetricDelta{{Name: application.MetricCitationValidity, State: application.MetricStateScored, PairedCases: 1, LeftNumerator: 1, LeftDenominator: 2, RightNumerator: 2, RightDenominator: 2, LeftValue: &value, RightValue: &value, Delta: &value}},
	}}
	mux := http.NewServeMux()
	NewHandler(&serviceStub{}, NewTokenAuthorizer("test-token")).WithComparison(comparison).Register(mux)

	request := httptest.NewRequest(http.MethodGet, "/api/v1/evaluations/compare?left="+leftRunID.String()+"&right="+rightRunID.String(), nil)
	request.Header.Set("Authorization", "Bearer test-token")
	response := httptest.NewRecorder()
	mux.ServeHTTP(response, request)
	if response.Code != http.StatusOK || comparison.request.LeftRunID != leftRunID || comparison.request.RightRunID != rightRunID {
		t.Fatalf("comparison = %d %s request=%#v", response.Code, response.Body.String(), comparison.request)
	}
	for _, required := range []string{`"comparisonState":"comparable"`, `"pairedCases":2`, `"citation_validity"`, `"retrievalStrategy":"hybrid"`} {
		if !strings.Contains(response.Body.String(), required) {
			t.Fatalf("comparison missing %q: %s", required, response.Body.String())
		}
	}

	comparison.result = application.ComparisonResult{
		State:       application.ComparisonStateNonComparable,
		Differences: []application.ComparisonDifference{{Field: "snapshot_manifest_sha256"}},
		Totals:      application.ComparisonTotals{LeftCases: 99, RightCases: 99, PairedCases: 99},
		Metrics: []application.MetricDelta{{
			Name: application.MetricCitationValidity, State: application.MetricStateScored, PairedCases: 99,
			LeftNumerator: 99, LeftDenominator: 99, RightNumerator: 99, RightDenominator: 99,
			LeftValue: &value, RightValue: &value, Delta: &value,
		}},
	}
	nonComparableResponse := httptest.NewRecorder()
	mux.ServeHTTP(nonComparableResponse, request)
	var nonComparablePayload comparisonResponse
	if err := json.Unmarshal(nonComparableResponse.Body.Bytes(), &nonComparablePayload); err != nil {
		t.Fatalf("decode non-comparable response: %v", err)
	}
	if nonComparableResponse.Code != http.StatusOK || !strings.Contains(nonComparableResponse.Body.String(), `"comparisonState":"non_comparable"`) ||
		!strings.Contains(nonComparableResponse.Body.String(), "snapshot_manifest_sha256") || nonComparablePayload.Totals != nil || len(nonComparablePayload.Metrics) != 0 ||
		strings.Contains(nonComparableResponse.Body.String(), `"delta"`) {
		t.Fatalf("non-comparable response = %d %s", nonComparableResponse.Code, nonComparableResponse.Body.String())
	}
}

func TestComparisonRequiresAuthorizationAndValidRunIdentifiers(t *testing.T) {
	comparison := &comparisonStub{}
	mux := http.NewServeMux()
	NewHandler(&serviceStub{}, NewTokenAuthorizer("test-token")).WithComparison(comparison).Register(mux)

	unauthorized := httptest.NewRecorder()
	mux.ServeHTTP(unauthorized, httptest.NewRequest(http.MethodGet, "/api/v1/evaluations/compare?left="+uuid.NewString()+"&right="+uuid.NewString(), nil))
	if unauthorized.Code != http.StatusUnauthorized || comparison.calls != 0 || strings.Contains(unauthorized.Body.String(), "leftRunId") {
		t.Fatalf("unauthorized comparison = %d %s", unauthorized.Code, unauthorized.Body.String())
	}

	invalidRequest := httptest.NewRequest(http.MethodGet, "/api/v1/evaluations/compare?left=invalid&right="+uuid.NewString(), nil)
	invalidRequest.Header.Set("Authorization", "Bearer test-token")
	invalid := httptest.NewRecorder()
	mux.ServeHTTP(invalid, invalidRequest)
	if invalid.Code != http.StatusBadRequest || comparison.calls != 0 || !strings.Contains(invalid.Body.String(), `"code":"invalid_input"`) {
		t.Fatalf("invalid comparison = %d %s", invalid.Code, invalid.Body.String())
	}
}

func TestEvaluationEndpointsRequireMaintainerAuthorization(t *testing.T) {
	mux := http.NewServeMux()
	NewHandler(&serviceStub{}, NewTokenAuthorizer("test-token")).Register(mux)

	response := httptest.NewRecorder()
	mux.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/api/v1/evaluations/"+uuid.New().String(), nil))
	if response.Code != http.StatusUnauthorized || !strings.Contains(response.Body.String(), "maintainer_authorization_required") || response.Header().Get("WWW-Authenticate") != "Bearer" {
		t.Fatalf("unauthorized response = %d %q headers=%#v", response.Code, response.Body.String(), response.Header())
	}
}

func TestDatasetInspectionRoutesRequireMaintainerAuthorizationWithoutDisclosure(t *testing.T) {
	revisionID, corpusID, sourceID, starterID, caseID := uuid.New(), uuid.New(), uuid.New(), uuid.New(), uuid.New()
	reviewedAt := time.Date(2026, time.August, 26, 20, 0, 0, 0, time.UTC)
	inspection := &inspectionStub{entry: application.DatasetCatalogEntry{
		Revision: application.DatasetRevisionSummary{ID: revisionID, CorpusID: corpusID, DatasetKey: "lgpd-evaluation",
			SemanticRevision: "v1", Jurisdiction: "Brazil", ManifestSHA256: strings.Repeat("a", 64), JSONLSHA256: strings.Repeat("b", 64),
			ContentSHA256: strings.Repeat("c", 64), DeclaredSnapshotDate: reviewedAt, QueryLanguages: []string{"en", "pt"}, AuthoritativeEvidenceLanguage: "pt-BR"},
		Review: &application.DatasetReview{Decision: "approved", PublicationState: "available", ReviewedAt: reviewedAt},
		Sources: []application.DatasetSource{{ID: sourceID, SourceAlias: "lgpd", Title: "LGPD", OfficialURL: "https://example.test/lgpd",
			IssuingAuthority: "Presidency", DocumentType: "statute", AuthorityRole: "statute", CorpusSourceID: &sourceID}},
		Starters: []application.StarterCase{{ID: starterID, CaseID: caseID, Rank: 1, QueryLanguage: "en", ReviewEligible: true}},
	}}
	mux := http.NewServeMux()
	NewHandler(&serviceStub{}, NewTokenAuthorizer("test-token"), inspection).Register(mux)

	for _, request := range []*http.Request{
		httptest.NewRequest(http.MethodGet, "/api/v1/evaluation-datasets", nil),
		httptest.NewRequest(http.MethodGet, "/api/v1/evaluation-datasets/"+revisionID.String(), nil),
		httptest.NewRequest(http.MethodGet, "/api/v1/evaluation-datasets/"+revisionID.String()+"/preflight?corpusId="+corpusID.String()+"&snapshotId="+uuid.NewString(), nil),
	} {
		response := httptest.NewRecorder()
		mux.ServeHTTP(response, request)
		if response.Code != http.StatusUnauthorized || response.Header().Get("WWW-Authenticate") != "Bearer" ||
			strings.Contains(response.Body.String(), starterID.String()) || strings.Contains(response.Body.String(), "LGPD") {
			t.Fatalf("unauthorized %s %s = %d %s", request.Method, request.URL, response.Code, response.Body.String())
		}
	}
	if inspection.listCalls != 0 || inspection.getCalls != 0 || inspection.checkCalls != 0 {
		t.Fatalf("unauthorized inspection calls = list:%d get:%d check:%d", inspection.listCalls, inspection.getCalls, inspection.checkCalls)
	}
}

func TestDatasetCatalogListProjectsOnlyRevisionAndReview(t *testing.T) {
	revisionID, corpusID := uuid.New(), uuid.New()
	reviewedAt := time.Date(2026, time.August, 26, 20, 0, 0, 0, time.UTC)
	inspection := &inspectionStub{entry: application.DatasetCatalogEntry{
		Revision: application.DatasetRevisionSummary{ID: revisionID, CorpusID: corpusID, DatasetKey: "lgpd-evaluation",
			SemanticRevision: "v1", Jurisdiction: "Brazil", ManifestSHA256: strings.Repeat("a", 64), JSONLSHA256: strings.Repeat("b", 64),
			ContentSHA256: strings.Repeat("c", 64), DeclaredSnapshotDate: reviewedAt, QueryLanguages: []string{"en", "pt"}, AuthoritativeEvidenceLanguage: "pt-BR"},
		Review:   &application.DatasetReview{Decision: "approved", PublicationState: "available", ReviewedAt: reviewedAt},
		Sources:  []application.DatasetSource{{SourceAlias: "must-not-appear"}},
		Starters: []application.StarterCase{{ID: uuid.New()}},
	}}
	mux := http.NewServeMux()
	NewHandler(&serviceStub{}, NewTokenAuthorizer("test-token"), inspection).Register(mux)
	request := httptest.NewRequest(http.MethodGet, "/api/v1/evaluation-datasets", nil)
	request.Header.Set("Authorization", "Bearer test-token")
	response := httptest.NewRecorder()
	mux.ServeHTTP(response, request)
	if response.Code != http.StatusOK {
		t.Fatalf("catalog list = %d %s", response.Code, response.Body.String())
	}
	for _, value := range []string{"manifestSha256", "queryLanguages", "publicationState", `"available":true`} {
		if !strings.Contains(response.Body.String(), value) {
			t.Fatalf("catalog list missing %q: %s", value, response.Body.String())
		}
	}
	for _, value := range []string{"sources", "starters", "must-not-appear"} {
		if strings.Contains(response.Body.String(), value) {
			t.Fatalf("catalog list disclosed detail %q: %s", value, response.Body.String())
		}
	}
	if inspection.listCalls != 1 || inspection.getCalls != 0 {
		t.Fatalf("catalog calls = list:%d get:%d", inspection.listCalls, inspection.getCalls)
	}
}

func TestDatasetDetailExposesMaintainerMetadataAndRejectsInvalidOrUnknownIdentifiers(t *testing.T) {
	revisionID, corpusID, sourceID, starterID, caseID := uuid.New(), uuid.New(), uuid.New(), uuid.New(), uuid.New()
	reviewedAt := time.Date(2026, time.August, 26, 20, 0, 0, 0, time.UTC)
	inspection := &inspectionStub{entry: application.DatasetCatalogEntry{
		Revision: application.DatasetRevisionSummary{ID: revisionID, CorpusID: corpusID, DatasetKey: "lgpd-evaluation",
			SemanticRevision: "v1", Jurisdiction: "Brazil", ManifestSHA256: strings.Repeat("a", 64), JSONLSHA256: strings.Repeat("b", 64),
			ContentSHA256: strings.Repeat("c", 64), DeclaredSnapshotDate: reviewedAt, QueryLanguages: []string{"en", "pt"}, AuthoritativeEvidenceLanguage: "pt-BR"},
		Review: &application.DatasetReview{Decision: "approved", PublicationState: "available", ReviewedAt: reviewedAt},
		Sources: []application.DatasetSource{{ID: sourceID, SourceAlias: "lgpd", Title: "LGPD", OfficialURL: "https://example.test/lgpd",
			IssuingAuthority: "Presidency", DocumentType: "statute", AuthorityRole: "statute", CorpusSourceID: &sourceID}},
		Starters: []application.StarterCase{{ID: starterID, CaseID: caseID, Rank: 1, QueryLanguage: "en", ReviewEligible: true}},
	}}
	mux := http.NewServeMux()
	NewHandler(&serviceStub{}, NewTokenAuthorizer("test-token"), inspection).Register(mux)

	invalid := httptest.NewRequest(http.MethodGet, "/api/v1/evaluation-datasets/not-a-uuid", nil)
	invalid.Header.Set("Authorization", "Bearer test-token")
	invalidResponse := httptest.NewRecorder()
	mux.ServeHTTP(invalidResponse, invalid)
	if invalidResponse.Code != http.StatusBadRequest || inspection.getCalls != 0 {
		t.Fatalf("invalid detail = %d %s, get calls = %d", invalidResponse.Code, invalidResponse.Body.String(), inspection.getCalls)
	}

	request := httptest.NewRequest(http.MethodGet, "/api/v1/evaluation-datasets/"+revisionID.String(), nil)
	request.Header.Set("Authorization", "Bearer test-token")
	response := httptest.NewRecorder()
	mux.ServeHTTP(response, request)
	if response.Code != http.StatusOK {
		t.Fatalf("dataset inspection = %d %s", response.Code, response.Body.String())
	}
	for _, value := range []string{"manifestSha256", "queryLanguages", "issuingAuthority", "corpusSourceId", "publicationState", starterID.String()} {
		if !strings.Contains(response.Body.String(), value) {
			t.Fatalf("dataset inspection missing %q: %s", value, response.Body.String())
		}
	}

	inspection.err = application.ErrDatasetNotFound
	notFound := httptest.NewRequest(http.MethodGet, "/api/v1/evaluation-datasets/"+uuid.NewString(), nil)
	notFound.Header.Set("Authorization", "Bearer test-token")
	notFoundResponse := httptest.NewRecorder()
	mux.ServeHTTP(notFoundResponse, notFound)
	if notFoundResponse.Code != http.StatusNotFound || strings.Contains(notFoundResponse.Body.String(), starterID.String()) {
		t.Fatalf("unknown detail = %d %s", notFoundResponse.Code, notFoundResponse.Body.String())
	}
}

func TestDatasetPreflightReturnsCompatibilityWithoutStartingRun(t *testing.T) {
	revisionID, corpusID, snapshotID := uuid.New(), uuid.New(), uuid.New()
	inspection := &inspectionStub{}
	mux := http.NewServeMux()
	runs := &serviceStub{}
	NewHandler(runs, NewTokenAuthorizer("test-token"), inspection).Register(mux)
	request := httptest.NewRequest(http.MethodGet, "/api/v1/evaluation-datasets/"+revisionID.String()+"/preflight?corpusId="+corpusID.String()+"&snapshotId="+snapshotID.String(), nil)
	request.Header.Set("Authorization", "Bearer test-token")
	response := httptest.NewRecorder()
	mux.ServeHTTP(response, request)
	if response.Code != http.StatusOK || !strings.Contains(response.Body.String(), `"compatible":true`) || !strings.Contains(response.Body.String(), `"missingRequirements":[]`) {
		t.Fatalf("dataset preflight = %d %s", response.Code, response.Body.String())
	}
	if inspection.request.DatasetRevisionID != revisionID || inspection.request.CorpusID != corpusID || inspection.request.SnapshotID != snapshotID || runs.startCalls != 0 {
		t.Fatalf("preflight request = %#v, run starts = %d", inspection.request, runs.startCalls)
	}
}

func TestDatasetPreflightRejectsInvalidIdentifiersAndReturnsSafeCompatibilityFailure(t *testing.T) {
	revisionID, corpusID, snapshotID := uuid.New(), uuid.New(), uuid.New()
	inspection := &inspectionStub{err: application.ErrSnapshotIncompatible}
	mux := http.NewServeMux()
	runs := &serviceStub{}
	NewHandler(runs, NewTokenAuthorizer("test-token"), inspection).Register(mux)

	invalid := httptest.NewRequest(http.MethodGet, "/api/v1/evaluation-datasets/not-a-uuid/preflight?corpusId="+corpusID.String()+"&snapshotId="+snapshotID.String(), nil)
	invalid.Header.Set("Authorization", "Bearer test-token")
	invalidResponse := httptest.NewRecorder()
	mux.ServeHTTP(invalidResponse, invalid)
	if invalidResponse.Code != http.StatusBadRequest || inspection.checkCalls != 0 {
		t.Fatalf("invalid preflight identifier = %d %s, checks=%d", invalidResponse.Code, invalidResponse.Body.String(), inspection.checkCalls)
	}
	invalidQuery := httptest.NewRequest(http.MethodGet, "/api/v1/evaluation-datasets/"+revisionID.String()+"/preflight?corpusId=not-a-uuid&snapshotId="+snapshotID.String(), nil)
	invalidQuery.Header.Set("Authorization", "Bearer test-token")
	invalidQueryResponse := httptest.NewRecorder()
	mux.ServeHTTP(invalidQueryResponse, invalidQuery)
	if invalidQueryResponse.Code != http.StatusBadRequest || inspection.checkCalls != 0 {
		t.Fatalf("invalid preflight query = %d %s, checks=%d", invalidQueryResponse.Code, invalidQueryResponse.Body.String(), inspection.checkCalls)
	}

	request := httptest.NewRequest(http.MethodGet, "/api/v1/evaluation-datasets/"+revisionID.String()+"/preflight?corpusId="+corpusID.String()+"&snapshotId="+snapshotID.String(), nil)
	request.Header.Set("Authorization", "Bearer test-token")
	response := httptest.NewRecorder()
	mux.ServeHTTP(response, request)
	if response.Code != http.StatusUnprocessableEntity || inspection.checkCalls != 1 || runs.startCalls != 0 || strings.Contains(response.Body.String(), "providerPayload") {
		t.Fatalf("rejected preflight response = %d %q, checks=%d starts=%d", response.Code, response.Body.String(), inspection.checkCalls, runs.startCalls)
	}
}

func TestDatasetPreflightReturnsNotFoundForAuthorizedAbsentRevision(t *testing.T) {
	revisionID, corpusID, snapshotID := uuid.New(), uuid.New(), uuid.New()
	inspection := &inspectionStub{err: application.ErrDatasetNotFound}
	mux := http.NewServeMux()
	runs := &serviceStub{}
	NewHandler(runs, NewTokenAuthorizer("test-token"), inspection).Register(mux)

	request := httptest.NewRequest(http.MethodGet, "/api/v1/evaluation-datasets/"+revisionID.String()+"/preflight?corpusId="+corpusID.String()+"&snapshotId="+snapshotID.String(), nil)
	request.Header.Set("Authorization", "Bearer test-token")
	response := httptest.NewRecorder()
	mux.ServeHTTP(response, request)

	if response.Code != http.StatusNotFound || inspection.checkCalls != 1 || runs.startCalls != 0 ||
		!strings.Contains(response.Body.String(), `"code":"not_found"`) {
		t.Fatalf("absent dataset preflight = %d %q, checks=%d starts=%d", response.Code, response.Body.String(), inspection.checkCalls, runs.startCalls)
	}
}

type serviceStub struct {
	run          application.Run
	caseResult   application.RunCase
	startRequest application.StartRunRequest
	getCaseRunID uuid.UUID
	getCaseID    uuid.UUID
	startError   error
	startCalls   int
}

func (stub *serviceStub) Start(_ context.Context, request application.StartRunRequest) (application.Run, error) {
	stub.startCalls++
	stub.startRequest = request
	return stub.run, stub.startError
}
func (stub *serviceStub) Get(context.Context, uuid.UUID) (application.Run, error) {
	return stub.run, nil
}
func (stub *serviceStub) GetCase(_ context.Context, runID, caseID uuid.UUID) (application.RunCase, error) {
	stub.getCaseRunID, stub.getCaseID = runID, caseID
	return stub.caseResult, nil
}

type comparisonStub struct {
	result  application.ComparisonResult
	err     error
	request application.ComparisonRequest
	calls   int
}

func (stub *comparisonStub) Compare(_ context.Context, request application.ComparisonRequest) (application.ComparisonResult, error) {
	stub.calls++
	stub.request = request
	return stub.result, stub.err
}

type inspectionStub struct {
	entry      application.DatasetCatalogEntry
	request    application.PreflightRequest
	err        error
	listCalls  int
	getCalls   int
	checkCalls int
}

func (stub *inspectionStub) List(context.Context) ([]application.DatasetCatalogEntry, error) {
	stub.listCalls++
	return []application.DatasetCatalogEntry{stub.entry}, stub.err
}

func (stub *inspectionStub) Get(context.Context, uuid.UUID) (application.DatasetCatalogEntry, error) {
	stub.getCalls++
	return stub.entry, stub.err
}

func (stub *inspectionStub) Check(_ context.Context, request application.PreflightRequest) (application.PreflightResult, error) {
	stub.checkCalls++
	stub.request = request
	return application.PreflightResult{CorpusID: request.CorpusID, DatasetRevisionID: request.DatasetRevisionID, SnapshotID: request.SnapshotID}, stub.err
}
