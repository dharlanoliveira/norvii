// Package http exposes maintainer evaluation run endpoints through the v1 API.
package http

import (
	"context"
	"crypto/subtle"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"time"

	"github.com/dharlanoliveira/norvii/apps/api/internal/evaluation/application"
	"github.com/dharlanoliveira/norvii/apps/api/internal/platform/httpserver"
	"github.com/google/uuid"
)

type service interface {
	Start(context.Context, application.StartRunRequest) (application.Run, error)
	Get(context.Context, uuid.UUID) (application.Run, error)
	GetCase(context.Context, uuid.UUID, uuid.UUID) (application.RunCase, error)
}

type inspectionService interface {
	List(context.Context) ([]application.DatasetCatalogEntry, error)
	Get(context.Context, uuid.UUID) (application.DatasetCatalogEntry, error)
	Check(context.Context, application.PreflightRequest) (application.PreflightResult, error)
}

type comparisonService interface {
	Compare(context.Context, application.ComparisonRequest) (application.ComparisonResult, error)
}

// Authorizer decides whether a request may access maintainer-only evaluation records.
type Authorizer interface {
	Authorize(*http.Request) bool
}

// TokenAuthorizer provides the minimal local maintainer boundary. It fails closed when no token
// is configured and compares presented bearer credentials in constant time.
type TokenAuthorizer struct{ token string }

// NewTokenAuthorizer constructs an explicit maintainer authorization boundary.
func NewTokenAuthorizer(token string) *TokenAuthorizer { return &TokenAuthorizer{token: token} }

// Authorize implements Authorizer.
func (authorizer *TokenAuthorizer) Authorize(request *http.Request) bool {
	if authorizer == nil || authorizer.token == "" {
		return false
	}
	const prefix = "Bearer "
	presented := request.Header.Get("Authorization")
	if len(presented) <= len(prefix) || presented[:len(prefix)] != prefix {
		return false
	}
	value := presented[len(prefix):]
	return len(value) == len(authorizer.token) && subtle.ConstantTimeCompare([]byte(value), []byte(authorizer.token)) == 1
}

// Handler maps maintainer-only evaluation creation and inspection to safe HTTP projections.
type Handler struct {
	service    service
	inspection inspectionService
	comparison comparisonService
	authorizer Authorizer
}

// NewHandler constructs the evaluation endpoint handler.
func NewHandler(service service, authorizer Authorizer, inspections ...inspectionService) *Handler {
	handler := &Handler{service: service, authorizer: authorizer}
	if len(inspections) > 0 {
		handler.inspection = inspections[0]
	}
	return handler
}

// WithComparison attaches the read-only immutable comparison service to this handler.
func (handler *Handler) WithComparison(comparison comparisonService) *Handler {
	if handler != nil {
		handler.comparison = comparison
	}
	return handler
}

// Register adds the evaluation routes to the shared API mux.
func (handler *Handler) Register(mux *http.ServeMux) {
	mux.HandleFunc("POST /api/v1/evaluations", handler.start)
	mux.HandleFunc("GET /api/v1/evaluations/compare", handler.compare)
	mux.HandleFunc("GET /api/v1/evaluations/{runId}", handler.get)
	mux.HandleFunc("GET /api/v1/evaluations/{runId}/cases/{caseId}", handler.getCase)
	mux.HandleFunc("GET /api/v1/evaluation-datasets", handler.listDatasets)
	mux.HandleFunc("GET /api/v1/evaluation-datasets/{datasetRevisionId}", handler.getDataset)
	mux.HandleFunc("GET /api/v1/evaluation-datasets/{datasetRevisionId}/preflight", handler.preflightDataset)
}

func (handler *Handler) compare(writer http.ResponseWriter, request *http.Request) {
	if !handler.authorize(writer, request) {
		return
	}
	if handler.comparison == nil {
		writeProblem(writer, request, http.StatusServiceUnavailable, "unavailable", "Evaluation comparison is temporarily unavailable.", nil)
		return
	}
	leftRunID, rightRunID, ok := comparisonIDs(writer, request)
	if !ok {
		return
	}
	result, err := handler.comparison.Compare(request.Context(), application.ComparisonRequest{
		LeftRunID: leftRunID, RightRunID: rightRunID,
	})
	if err != nil {
		writeComparisonError(writer, request, err)
		return
	}
	httpserver.WriteJSON(writer, http.StatusOK, newComparisonResponse(leftRunID, rightRunID, result))
}

type startRequest struct {
	DatasetRevisionID uuid.UUID            `json:"datasetRevisionId"`
	CorpusID          uuid.UUID            `json:"corpusId"`
	SnapshotID        uuid.UUID            `json:"snapshotId"`
	Configuration     configurationRequest `json:"configuration"`
}

type configurationRequest struct {
	Strategy    string `json:"strategy"`
	Fingerprint string `json:"fingerprint"`
}

func (handler *Handler) start(writer http.ResponseWriter, request *http.Request) {
	if !handler.authorize(writer, request) {
		return
	}
	var body startRequest
	if err := decodeJSON(request, &body); err != nil {
		writeProblem(writer, request, http.StatusBadRequest, "invalid_input", "The evaluation request is invalid.", nil)
		return
	}
	run, err := handler.service.Start(request.Context(), application.StartRunRequest{
		DatasetRevisionID: body.DatasetRevisionID, CorpusID: body.CorpusID, SnapshotID: body.SnapshotID,
		Configuration: application.RetrievalConfiguration{Strategy: body.Configuration.Strategy, Fingerprint: body.Configuration.Fingerprint},
	})
	if err != nil {
		writeRunError(writer, request, err)
		return
	}
	httpserver.WriteJSON(writer, http.StatusCreated, newStartResponse(run))
}

func (handler *Handler) get(writer http.ResponseWriter, request *http.Request) {
	if !handler.authorize(writer, request) {
		return
	}
	runID, ok := pathID(writer, request, "runId")
	if !ok {
		return
	}
	run, err := handler.service.Get(request.Context(), runID)
	if err != nil {
		writeRunError(writer, request, err)
		return
	}
	httpserver.WriteJSON(writer, http.StatusOK, newRunResponse(run))
}

func (handler *Handler) getCase(writer http.ResponseWriter, request *http.Request) {
	if !handler.authorize(writer, request) {
		return
	}
	runID, ok := pathID(writer, request, "runId")
	if !ok {
		return
	}
	caseID, ok := pathID(writer, request, "caseId")
	if !ok {
		return
	}
	result, err := handler.service.GetCase(request.Context(), runID, caseID)
	if err != nil {
		writeRunError(writer, request, err)
		return
	}
	httpserver.WriteJSON(writer, http.StatusOK, newRunCaseResponse(result))
}

func (handler *Handler) listDatasets(writer http.ResponseWriter, request *http.Request) {
	if !handler.authorize(writer, request) {
		return
	}
	if handler.inspection == nil {
		writeProblem(writer, request, http.StatusServiceUnavailable, "unavailable", "Evaluation data is temporarily unavailable.", nil)
		return
	}
	entries, err := handler.inspection.List(request.Context())
	if err != nil {
		writeDatasetError(writer, request, err)
		return
	}
	response := make([]datasetCatalogResponse, 0, len(entries))
	for _, entry := range entries {
		response = append(response, newDatasetCatalogResponse(entry, false))
	}
	httpserver.WriteJSON(writer, http.StatusOK, response)
}

func (handler *Handler) getDataset(writer http.ResponseWriter, request *http.Request) {
	if !handler.authorize(writer, request) {
		return
	}
	if handler.inspection == nil {
		writeProblem(writer, request, http.StatusServiceUnavailable, "unavailable", "Evaluation data is temporarily unavailable.", nil)
		return
	}
	datasetRevisionID, ok := pathID(writer, request, "datasetRevisionId")
	if !ok {
		return
	}
	entry, err := handler.inspection.Get(request.Context(), datasetRevisionID)
	if err != nil {
		writeDatasetError(writer, request, err)
		return
	}
	httpserver.WriteJSON(writer, http.StatusOK, newDatasetCatalogResponse(entry, true))
}

func (handler *Handler) preflightDataset(writer http.ResponseWriter, request *http.Request) {
	if !handler.authorize(writer, request) {
		return
	}
	if handler.inspection == nil {
		writeProblem(writer, request, http.StatusServiceUnavailable, "unavailable", "Evaluation data is temporarily unavailable.", nil)
		return
	}
	datasetRevisionID, ok := pathID(writer, request, "datasetRevisionId")
	if !ok {
		return
	}
	corpusID, snapshotID, ok := preflightIDs(writer, request)
	if !ok {
		return
	}
	_, err := handler.inspection.Check(request.Context(), application.PreflightRequest{
		CorpusID: corpusID, DatasetRevisionID: datasetRevisionID, SnapshotID: snapshotID,
	})
	if err != nil {
		writeRunError(writer, request, err)
		return
	}
	httpserver.WriteJSON(writer, http.StatusOK, preflightResponse{
		DatasetRevisionID: datasetRevisionID, CorpusID: corpusID, SnapshotID: snapshotID,
		Compatible: true, MissingRequirements: []application.MissingRequirement{},
	})
}

func (handler *Handler) authorize(writer http.ResponseWriter, request *http.Request) bool {
	if handler != nil && handler.authorizer != nil && handler.authorizer.Authorize(request) {
		return true
	}
	writer.Header().Set("WWW-Authenticate", "Bearer")
	writeProblem(writer, request, http.StatusUnauthorized, "maintainer_authorization_required", "Maintainer authorization is required.", nil)
	return false
}

func pathID(writer http.ResponseWriter, request *http.Request, name string) (uuid.UUID, bool) {
	identifier, err := uuid.Parse(request.PathValue(name))
	if err != nil || identifier == uuid.Nil {
		writeProblem(writer, request, http.StatusBadRequest, "invalid_input", "The evaluation identifier is invalid.", nil)
		return uuid.Nil, false
	}
	return identifier, true
}

func preflightIDs(writer http.ResponseWriter, request *http.Request) (uuid.UUID, uuid.UUID, bool) {
	corpusID, corpusErr := uuid.Parse(request.URL.Query().Get("corpusId"))
	snapshotID, snapshotErr := uuid.Parse(request.URL.Query().Get("snapshotId"))
	if corpusErr != nil || snapshotErr != nil || corpusID == uuid.Nil || snapshotID == uuid.Nil {
		writeProblem(writer, request, http.StatusBadRequest, "invalid_input", "The evaluation compatibility request is invalid.", nil)
		return uuid.Nil, uuid.Nil, false
	}
	return corpusID, snapshotID, true
}

func comparisonIDs(writer http.ResponseWriter, request *http.Request) (uuid.UUID, uuid.UUID, bool) {
	query := request.URL.Query()
	leftRunID, leftErr := uuid.Parse(query.Get("left"))
	rightRunID, rightErr := uuid.Parse(query.Get("right"))
	if leftErr != nil || rightErr != nil || leftRunID == uuid.Nil || rightRunID == uuid.Nil {
		writeProblem(writer, request, http.StatusBadRequest, "invalid_input", "The evaluation comparison request is invalid.", nil)
		return uuid.Nil, uuid.Nil, false
	}
	return leftRunID, rightRunID, true
}

func decodeJSON(request *http.Request, destination any) error {
	decoder := json.NewDecoder(request.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(destination); err != nil {
		return err
	}
	if err := decoder.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
		return errors.New("evaluation request must contain one JSON object")
	}
	return nil
}

func writeRunError(writer http.ResponseWriter, request *http.Request, err error) {
	var compatibility *application.CompatibilityError
	switch {
	case errors.As(err, &compatibility):
		code, status, message := "snapshot_incompatible", http.StatusUnprocessableEntity, "The selected snapshot is incompatible with the dataset."
		switch {
		case errors.Is(err, application.ErrDatasetUnavailable):
			code, message = "dataset_not_available", "The evaluation dataset is not available."
		case errors.Is(err, application.ErrPreflightCorpusMismatch):
			code, message = "corpus_mismatch", "The dataset does not belong to the selected corpus."
		case errors.Is(err, application.ErrLocatorUnresolved):
			code, message = "locator_unresolved", "A required legal locator did not resolve in the selected snapshot."
		}
		writeProblem(writer, request, status, code, message, compatibility.MissingRequirements())
	case errors.Is(err, application.ErrDatasetUnavailable), errors.Is(err, application.ErrPreflightCorpusMismatch),
		errors.Is(err, application.ErrSnapshotIncompatible), errors.Is(err, application.ErrLocatorUnresolved):
		code, message := "snapshot_incompatible", "The selected snapshot is incompatible with the dataset."
		switch {
		case errors.Is(err, application.ErrDatasetUnavailable):
			code, message = "dataset_not_available", "The evaluation dataset is not available."
		case errors.Is(err, application.ErrPreflightCorpusMismatch):
			code, message = "corpus_mismatch", "The dataset does not belong to the selected corpus."
		case errors.Is(err, application.ErrLocatorUnresolved):
			code, message = "locator_unresolved", "A required legal locator did not resolve in the selected snapshot."
		}
		writeProblem(writer, request, http.StatusUnprocessableEntity, code, message, nil)
	case errors.Is(err, application.ErrInvalidRunRequest), errors.Is(err, application.ErrInvalidPreflightRequest):
		writeProblem(writer, request, http.StatusBadRequest, "invalid_configuration", "The evaluation configuration is invalid.", nil)
	case errors.Is(err, application.ErrDatasetNotFound):
		writeProblem(writer, request, http.StatusNotFound, "not_found", "The evaluation dataset was not found.", nil)
	case errors.Is(err, application.ErrRunNotFound):
		writeProblem(writer, request, http.StatusNotFound, "not_found", "The evaluation run was not found.", nil)
	default:
		writeProblem(writer, request, http.StatusServiceUnavailable, "unavailable", "Evaluation data is temporarily unavailable.", nil)
	}
}

func writeDatasetError(writer http.ResponseWriter, request *http.Request, err error) {
	switch {
	case errors.Is(err, application.ErrDatasetNotFound):
		writeProblem(writer, request, http.StatusNotFound, "not_found", "The evaluation dataset was not found.", nil)
	default:
		writeProblem(writer, request, http.StatusServiceUnavailable, "unavailable", "Evaluation data is temporarily unavailable.", nil)
	}
}

func writeComparisonError(writer http.ResponseWriter, request *http.Request, err error) {
	switch {
	case errors.Is(err, application.ErrInvalidComparisonRequest):
		writeProblem(writer, request, http.StatusBadRequest, "invalid_input", "The evaluation comparison request is invalid.", nil)
	case errors.Is(err, application.ErrComparisonRunNotFound):
		writeProblem(writer, request, http.StatusNotFound, "not_found", "An evaluation run was not found.", nil)
	default:
		writeProblem(writer, request, http.StatusServiceUnavailable, "unavailable", "Evaluation comparison is temporarily unavailable.", nil)
	}
}

func writeProblem(writer http.ResponseWriter, request *http.Request, status int, code, message string, missing []application.MissingRequirement) {
	problem := httpserver.Problem{Status: status, Code: code, Message: message}
	if len(missing) > 0 {
		problem.Details = map[string]any{"missingRequirements": missing}
	}
	httpserver.WriteError(writer, request, problem)
}

type startResponse struct {
	RunID                uuid.UUID        `json:"runId"`
	State                string           `json:"state"`
	DatasetRevision      revisionResponse `json:"datasetRevision"`
	CorpusID             uuid.UUID        `json:"corpusId"`
	SnapshotID           uuid.UUID        `json:"snapshotId"`
	ScoringPolicyVersion string           `json:"scoringPolicyVersion"`
}

type revisionResponse struct {
	ID            uuid.UUID `json:"id"`
	ContentSHA256 string    `json:"contentSha256"`
}

type datasetCatalogResponse struct {
	Revision  datasetRevisionResponse `json:"revision"`
	Available bool                    `json:"available"`
	Review    *datasetReviewResponse  `json:"review"`
	Sources   []datasetSourceResponse `json:"sources,omitempty"`
	Starters  []starterCaseResponse   `json:"starters,omitempty"`
}

type datasetRevisionResponse struct {
	ID                            uuid.UUID `json:"id"`
	CorpusID                      uuid.UUID `json:"corpusId"`
	DatasetKey                    string    `json:"datasetKey"`
	SemanticRevision              string    `json:"semanticRevision"`
	Jurisdiction                  string    `json:"jurisdiction"`
	ManifestSHA256                string    `json:"manifestSha256"`
	JSONLSHA256                   string    `json:"jsonlSha256"`
	ContentSHA256                 string    `json:"contentSha256"`
	DeclaredSnapshotDate          string    `json:"declaredSnapshotDate"`
	QueryLanguages                []string  `json:"queryLanguages"`
	AuthoritativeEvidenceLanguage string    `json:"authoritativeEvidenceLanguage"`
}

type datasetReviewResponse struct {
	Decision         string `json:"decision"`
	PublicationState string `json:"publicationState"`
	ReviewedAt       string `json:"reviewedAt"`
}

type datasetSourceResponse struct {
	ID               uuid.UUID  `json:"id"`
	SourceAlias      string     `json:"sourceAlias"`
	Title            string     `json:"title"`
	OfficialURL      string     `json:"officialUrl"`
	IssuingAuthority string     `json:"issuingAuthority"`
	DocumentType     string     `json:"documentType"`
	AuthorityRole    string     `json:"authorityRole"`
	CorpusSourceID   *uuid.UUID `json:"corpusSourceId"`
	Bound            bool       `json:"bound"`
}

type starterCaseResponse struct {
	ID             uuid.UUID `json:"id"`
	CaseID         uuid.UUID `json:"caseId"`
	Rank           int       `json:"rank"`
	QueryLanguage  string    `json:"queryLanguage"`
	ReviewEligible bool      `json:"reviewEligible"`
}

type preflightResponse struct {
	DatasetRevisionID   uuid.UUID                        `json:"datasetRevisionId"`
	CorpusID            uuid.UUID                        `json:"corpusId"`
	SnapshotID          uuid.UUID                        `json:"snapshotId"`
	Compatible          bool                             `json:"compatible"`
	MissingRequirements []application.MissingRequirement `json:"missingRequirements"`
}

func newDatasetCatalogResponse(entry application.DatasetCatalogEntry, includeDetails bool) datasetCatalogResponse {
	revision := entry.Revision
	response := datasetCatalogResponse{
		Revision: datasetRevisionResponse{
			ID: revision.ID, CorpusID: revision.CorpusID, DatasetKey: revision.DatasetKey, SemanticRevision: revision.SemanticRevision,
			Jurisdiction: revision.Jurisdiction, ManifestSHA256: revision.ManifestSHA256, JSONLSHA256: revision.JSONLSHA256,
			ContentSHA256: revision.ContentSHA256, DeclaredSnapshotDate: revision.DeclaredSnapshotDate.UTC().Format("2006-01-02"),
			QueryLanguages: append([]string(nil), revision.QueryLanguages...), AuthoritativeEvidenceLanguage: revision.AuthoritativeEvidenceLanguage,
		},
		Available: entry.Available(),
	}
	if entry.Review != nil {
		response.Review = &datasetReviewResponse{Decision: entry.Review.Decision, PublicationState: entry.Review.PublicationState, ReviewedAt: entry.Review.ReviewedAt.UTC().Format(time.RFC3339)}
	}
	if !includeDetails {
		return response
	}
	response.Sources = make([]datasetSourceResponse, 0, len(entry.Sources))
	for _, source := range entry.Sources {
		response.Sources = append(response.Sources, datasetSourceResponse{ID: source.ID, SourceAlias: source.SourceAlias, Title: source.Title,
			OfficialURL: source.OfficialURL, IssuingAuthority: source.IssuingAuthority, DocumentType: source.DocumentType,
			AuthorityRole: source.AuthorityRole, CorpusSourceID: source.CorpusSourceID, Bound: source.CorpusSourceID != nil})
	}
	response.Starters = make([]starterCaseResponse, 0, len(entry.Starters))
	for _, starter := range entry.Starters {
		response.Starters = append(response.Starters, starterCaseResponse{ID: starter.ID, CaseID: starter.CaseID, Rank: starter.Rank,
			QueryLanguage: starter.QueryLanguage, ReviewEligible: starter.ReviewEligible})
	}
	return response
}

func newStartResponse(run application.Run) startResponse {
	return startResponse{RunID: run.ID, State: string(run.State), DatasetRevision: revisionResponse{ID: run.DatasetRevisionID, ContentSHA256: run.DatasetContentSHA256}, CorpusID: run.CorpusID, SnapshotID: run.SnapshotID, ScoringPolicyVersion: run.ScoringPolicyVersion}
}

type runResponse struct {
	ID                     uuid.UUID             `json:"id"`
	DatasetRevision        revisionResponse      `json:"datasetRevision"`
	CorpusID               uuid.UUID             `json:"corpusId"`
	SnapshotID             uuid.UUID             `json:"snapshotId"`
	SnapshotManifestSHA256 string                `json:"snapshotManifestSha256"`
	OrderedCaseSetSHA256   string                `json:"orderedCaseSetSha256"`
	Configuration          configurationResponse `json:"configuration"`
	ScoringPolicyVersion   string                `json:"scoringPolicyVersion"`
	AgentBuild             string                `json:"agentBuild"`
	ChatModelIdentity      string                `json:"chatModelIdentity"`
	EmbeddingModelIdentity string                `json:"embeddingModelIdentity"`
	InitiatedBy            string                `json:"initiatedBy"`
	State                  string                `json:"state"`
	CreatedAt              string                `json:"createdAt"`
	StartedAt              *string               `json:"startedAt,omitempty"`
	CompletedAt            *string               `json:"completedAt,omitempty"`
	Aggregate              aggregateResponse     `json:"aggregate"`
	Cases                  []caseSummaryResponse `json:"cases"`
}

type configurationResponse struct {
	Strategy    string `json:"strategy"`
	Fingerprint string `json:"fingerprint"`
}
type aggregateResponse struct {
	Total         int64            `json:"total"`
	Eligible      int64            `json:"eligible"`
	Scored        int64            `json:"scored"`
	Failed        int64            `json:"failed"`
	Cancelled     int64            `json:"cancelled"`
	NotApplicable int64            `json:"notApplicable"`
	Metrics       []metricResponse `json:"metrics"`
}
type metricResponse struct {
	Name          string   `json:"name"`
	State         string   `json:"state"`
	Value         *float64 `json:"value,omitempty"`
	Numerator     *int64   `json:"numerator,omitempty"`
	Denominator   *int64   `json:"denominator,omitempty"`
	Rationale     string   `json:"rationale"`
	ScorerVersion string   `json:"scorerVersion"`
}
type caseSummaryResponse struct {
	ID            uuid.UUID `json:"id"`
	DatasetCaseID uuid.UUID `json:"datasetCaseId"`
	Position      int       `json:"position"`
	State         string    `json:"state"`
	AttemptCount  int       `json:"attemptCount"`
	FinishedAt    *string   `json:"finishedAt,omitempty"`
	FailureCode   string    `json:"failureCode,omitempty"`
}

func newRunResponse(run application.Run) runResponse {
	response := runResponse{ID: run.ID, DatasetRevision: revisionResponse{ID: run.DatasetRevisionID, ContentSHA256: run.DatasetContentSHA256}, CorpusID: run.CorpusID, SnapshotID: run.SnapshotID, SnapshotManifestSHA256: run.SnapshotManifestSHA256, OrderedCaseSetSHA256: run.OrderedCaseSetSHA256, Configuration: configurationResponse{Strategy: run.Configuration.Strategy, Fingerprint: run.Configuration.Fingerprint}, ScoringPolicyVersion: run.ScoringPolicyVersion, AgentBuild: run.AgentBuild, ChatModelIdentity: run.ChatModelIdentity, EmbeddingModelIdentity: run.EmbeddingModelIdentity, InitiatedBy: run.InitiatedBy, State: string(run.State), CreatedAt: run.CreatedAt.UTC().Format("2006-01-02T15:04:05Z07:00"), Aggregate: newAggregateResponse(run.Aggregate), Cases: make([]caseSummaryResponse, 0, len(run.Cases))}
	response.StartedAt = timeString(run.StartedAt)
	response.CompletedAt = timeString(run.CompletedAt)
	for _, item := range run.Cases {
		response.Cases = append(response.Cases, newCaseSummaryResponse(item))
	}
	return response
}

func newAggregateResponse(aggregate application.RunAggregate) aggregateResponse {
	response := aggregateResponse{Total: aggregate.Total, Eligible: aggregate.Eligible, Scored: aggregate.Scored, Failed: aggregate.Failed, Cancelled: aggregate.Cancelled, NotApplicable: aggregate.NotApplicable, Metrics: make([]metricResponse, 0, len(aggregate.Metrics))}
	for _, metric := range aggregate.Metrics {
		response.Metrics = append(response.Metrics, newMetricResponse(metric))
	}
	return response
}
func newMetricResponse(metric application.RunMetric) metricResponse {
	return metricResponse{Name: metric.Name, State: metric.State, Value: metric.Value, Numerator: metric.Numerator, Denominator: metric.Denominator, Rationale: metric.Rationale, ScorerVersion: metric.ScorerVersion}
}
func newCaseSummaryResponse(item application.RunCaseSummary) caseSummaryResponse {
	return caseSummaryResponse{ID: item.ID, DatasetCaseID: item.DatasetCaseID, Position: item.Position, State: string(item.State), AttemptCount: item.AttemptCount, FinishedAt: timeString(item.FinishedAt), FailureCode: item.FailureCode}
}

type runCaseResponse struct {
	caseSummaryResponse
	RunID               uuid.UUID          `json:"runId"`
	CorpusID            uuid.UUID          `json:"corpusId"`
	SnapshotID          uuid.UUID          `json:"snapshotId"`
	DatasetRevisionID   uuid.UUID          `json:"datasetRevisionId"`
	Question            string             `json:"question"`
	ReferenceAnswer     string             `json:"referenceAnswer"`
	ExpectedOutcome     string             `json:"expectedOutcome"`
	ExpectedEvidence    []evidenceResponse `json:"expectedEvidence"`
	ActualEvidence      []evidenceResponse `json:"actualEvidence"`
	Answer              string             `json:"answer,omitempty"`
	GraphGroundingState string             `json:"graphGroundingState,omitempty"`
	LatencyMilliseconds *int64             `json:"latencyMilliseconds,omitempty"`
	InputTokens         *int64             `json:"inputTokens,omitempty"`
	OutputTokens        *int64             `json:"outputTokens,omitempty"`
	Metrics             []metricResponse   `json:"metrics"`
}

type evidenceResponse struct {
	Kind             string    `json:"kind"`
	Position         int       `json:"position"`
	MarkerPosition   int       `json:"markerPosition,omitempty"`
	CorpusID         uuid.UUID `json:"corpusId"`
	SnapshotID       uuid.UUID `json:"snapshotId"`
	SourceID         uuid.UUID `json:"sourceId"`
	SourceRevisionID uuid.UUID `json:"sourceRevisionId"`
	DocumentID       uuid.UUID `json:"documentId"`
	LegalUnitID      uuid.UUID `json:"legalUnitId"`
	CanonicalLocator string    `json:"canonicalLocator"`
	DisplayLocator   string    `json:"displayLocator,omitempty"`
	StartOffset      *int      `json:"startOffset,omitempty"`
	EndOffset        *int      `json:"endOffset,omitempty"`
	ContentSHA256    string    `json:"contentSha256"`
}

type comparisonResponse struct {
	ComparisonState string                         `json:"comparisonState"`
	LeftRunID       uuid.UUID                      `json:"leftRunId"`
	RightRunID      uuid.UUID                      `json:"rightRunId"`
	Left            comparisonExperimentResponse   `json:"left"`
	Right           comparisonExperimentResponse   `json:"right"`
	Differences     []comparisonDifferenceResponse `json:"differences"`
	Totals          *comparisonTotalsResponse      `json:"totals,omitempty"`
	Metrics         []comparisonMetricResponse     `json:"metrics"`
}

type comparisonExperimentResponse struct {
	RetrievalStrategy                 string `json:"retrievalStrategy"`
	RetrievalConfigurationFingerprint string `json:"retrievalConfigurationFingerprint"`
	AgentBuild                        string `json:"agentBuild"`
	ChatModelIdentity                 string `json:"chatModelIdentity"`
	EmbeddingModelIdentity            string `json:"embeddingModelIdentity"`
}

type comparisonDifferenceResponse struct {
	Field string `json:"field"`
}

type comparisonTotalsResponse struct {
	LeftCases         int64 `json:"leftCases"`
	RightCases        int64 `json:"rightCases"`
	PairedCases       int64 `json:"pairedCases"`
	LeftUnpaired      int64 `json:"leftUnpaired"`
	RightUnpaired     int64 `json:"rightUnpaired"`
	FailedOrCancelled int64 `json:"failedOrCancelled"`
	LeftFailed        int64 `json:"leftFailed"`
	LeftCancelled     int64 `json:"leftCancelled"`
	RightFailed       int64 `json:"rightFailed"`
	RightCancelled    int64 `json:"rightCancelled"`
}

type comparisonMetricResponse struct {
	Name             string   `json:"name"`
	State            string   `json:"state"`
	PairedCases      int64    `json:"pairedCases"`
	LeftNumerator    int64    `json:"leftNumerator"`
	LeftDenominator  int64    `json:"leftDenominator"`
	RightNumerator   int64    `json:"rightNumerator"`
	RightDenominator int64    `json:"rightDenominator"`
	LeftValue        *float64 `json:"leftValue,omitempty"`
	RightValue       *float64 `json:"rightValue,omitempty"`
	Delta            *float64 `json:"delta,omitempty"`
}

func newRunCaseResponse(result application.RunCase) runCaseResponse {
	response := runCaseResponse{
		caseSummaryResponse: newCaseSummaryResponse(result.RunCaseSummary), RunID: result.RunID, CorpusID: result.CorpusID,
		SnapshotID: result.SnapshotID, DatasetRevisionID: result.DatasetRevisionID, Question: result.Question,
		ReferenceAnswer: result.ReferenceAnswer, ExpectedOutcome: result.ExpectedOutcome, Answer: result.Answer,
		GraphGroundingState: result.GraphGroundingState, LatencyMilliseconds: result.LatencyMilliseconds,
		InputTokens: result.InputTokens, OutputTokens: result.OutputTokens,
		ExpectedEvidence: make([]evidenceResponse, 0, len(result.ExpectedEvidence)),
		ActualEvidence:   make([]evidenceResponse, 0, len(result.ActualEvidence)), Metrics: make([]metricResponse, 0, len(result.Metrics)),
	}
	for _, item := range result.ExpectedEvidence {
		response.ExpectedEvidence = append(response.ExpectedEvidence, newEvidenceResponse(item))
	}
	for _, item := range result.ActualEvidence {
		response.ActualEvidence = append(response.ActualEvidence, newEvidenceResponse(item))
	}
	for _, metric := range result.Metrics {
		response.Metrics = append(response.Metrics, newMetricResponse(metric))
	}
	return response
}

func newEvidenceResponse(item application.EvidenceIdentity) evidenceResponse {
	return evidenceResponse{Kind: item.Kind, Position: item.Position, MarkerPosition: item.MarkerPosition, CorpusID: item.CorpusID,
		SnapshotID: item.SnapshotID, SourceID: item.SourceID, SourceRevisionID: item.SourceRevisionID, DocumentID: item.DocumentID,
		LegalUnitID: item.LegalUnitID, CanonicalLocator: item.CanonicalLocator, DisplayLocator: item.DisplayLocator,
		StartOffset: item.StartOffset, EndOffset: item.EndOffset, ContentSHA256: item.ContentSHA256}
}

func newComparisonResponse(leftRunID, rightRunID uuid.UUID, result application.ComparisonResult) comparisonResponse {
	response := comparisonResponse{
		ComparisonState: string(result.State), LeftRunID: leftRunID, RightRunID: rightRunID,
		Left: newComparisonExperimentResponse(result.Left), Right: newComparisonExperimentResponse(result.Right),
		Differences: make([]comparisonDifferenceResponse, 0, len(result.Differences)),
		Metrics:     make([]comparisonMetricResponse, 0, len(result.Metrics)),
	}
	if result.State == application.ComparisonStateComparable {
		totals := newComparisonTotalsResponse(result.Totals)
		response.Totals = &totals
	}
	for _, difference := range result.Differences {
		response.Differences = append(response.Differences, comparisonDifferenceResponse{Field: difference.Field})
	}
	if result.State != application.ComparisonStateComparable {
		return response
	}
	for _, metric := range result.Metrics {
		response.Metrics = append(response.Metrics, comparisonMetricResponse{
			Name: string(metric.Name), State: string(metric.State), PairedCases: metric.PairedCases,
			LeftNumerator: metric.LeftNumerator, LeftDenominator: metric.LeftDenominator,
			RightNumerator: metric.RightNumerator, RightDenominator: metric.RightDenominator,
			LeftValue: metric.LeftValue, RightValue: metric.RightValue, Delta: metric.Delta,
		})
	}
	return response
}

func newComparisonExperimentResponse(identity application.ExperimentIdentity) comparisonExperimentResponse {
	return comparisonExperimentResponse{
		RetrievalStrategy:                 identity.RetrievalStrategy,
		RetrievalConfigurationFingerprint: identity.RetrievalConfigurationFingerprint,
		AgentBuild:                        identity.AgentBuild,
		ChatModelIdentity:                 identity.ChatModelIdentity,
		EmbeddingModelIdentity:            identity.EmbeddingModelIdentity,
	}
}

func newComparisonTotalsResponse(totals application.ComparisonTotals) comparisonTotalsResponse {
	return comparisonTotalsResponse{
		LeftCases: totals.LeftCases, RightCases: totals.RightCases, PairedCases: totals.PairedCases,
		LeftUnpaired: totals.LeftUnpaired, RightUnpaired: totals.RightUnpaired,
		FailedOrCancelled: totals.FailedOrCancelled, LeftFailed: totals.LeftFailed,
		LeftCancelled: totals.LeftCancelled, RightFailed: totals.RightFailed, RightCancelled: totals.RightCancelled,
	}
}

func timeString(value *time.Time) *string {
	if value == nil {
		return nil
	}
	formatted := value.UTC().Format(time.RFC3339)
	return &formatted
}
