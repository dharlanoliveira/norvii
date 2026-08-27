package application

import (
	"errors"
	"fmt"
	"regexp"
	"slices"
	"strconv"
	"strings"

	"github.com/dharlanoliveira/norvii/apps/api/internal/evaluation/domain"
	"github.com/google/uuid"
)

var (
	// ErrInvalidEvaluationResult identifies a malformed result returned at the evaluation boundary.
	ErrInvalidEvaluationResult = errors.New("evaluation agent result is invalid")
	// ErrEvaluationEvidenceBoundary identifies evidence outside the fixed corpus or snapshot.
	ErrEvaluationEvidenceBoundary = errors.New("evaluation evidence is outside the fixed corpus snapshot boundary")
	// ErrInvalidEvidenceProvenance identifies incomplete or internally inconsistent evidence identity.
	ErrInvalidEvidenceProvenance = errors.New("evaluation evidence provenance is invalid")
	// ErrInvalidScoringInput identifies scoring data that cannot safely be evaluated.
	ErrInvalidScoringInput = errors.New("evaluation scoring input is invalid")
)

const ScoringPolicyV1 = "v1"

var (
	citationMarkerPattern    = regexp.MustCompile(`\[([0-9]+)\]`)
	canonicalEvidenceLocator = regexp.MustCompile(`^[a-z][a-z-]*:[a-z0-9.-]+(?:/[a-z][a-z-]*:[a-z0-9.-]+)*$`)
)

// CaseExecutionState is the terminal state of one independent evaluation case.
type CaseExecutionState string

const (
	CaseExecutionCompleted CaseExecutionState = "completed"
	CaseExecutionAbstained CaseExecutionState = "abstained"
	CaseExecutionFailed    CaseExecutionState = "failed"
	CaseExecutionCancelled CaseExecutionState = "cancelled"
)

// AgentOutcome describes the result the non-streaming evaluation agent claims to have produced.
type AgentOutcome string

const (
	AgentOutcomeCompleted AgentOutcome = "completed"
	AgentOutcomeAbstained AgentOutcome = "abstained"
)

// EvidenceProvenance is the complete immutable identity of one evidence location. It never holds
// document text, excerpts, prompts, or provider payloads.
type EvidenceProvenance struct {
	CorpusID         uuid.UUID
	SnapshotID       uuid.UUID
	SourceID         uuid.UUID
	SourceRevisionID uuid.UUID
	DocumentID       uuid.UUID
	UnitID           uuid.UUID
	CanonicalLocator string
	StartOffset      int
	EndOffset        int
	ContentSHA256    domain.SHA256
}

// RetrievedEvidence is one item from the ordered evidence sequence used by answer markers.
type RetrievedEvidence struct {
	Provenance EvidenceProvenance
}

// AgentCaseResult is the safe, transport-independent shape accepted from an evaluation agent.
// Retrieval order is the slice order; markers use one-based positions in that order.
type AgentCaseResult struct {
	State               CaseExecutionState
	Outcome             AgentOutcome
	Answer              string
	Retrieved           []RetrievedEvidence
	LatencyMilliseconds *int64
	InputTokens         *int64
	OutputTokens        *int64
}

// EvidenceKind distinguishes evidence the system retrieved from evidence the answer actually cited.
type EvidenceKind string

const (
	EvidenceKindRetrieved EvidenceKind = "retrieved"
	EvidenceKindCited     EvidenceKind = "cited"
)

// ActualEvidence is a detached persistence-ready evidence record. Position is retrieval order for
// retrieved evidence and answer marker order for cited evidence.
type ActualEvidence struct {
	Kind           EvidenceKind
	Position       int
	MarkerPosition int
	Provenance     EvidenceProvenance
}

// CitationMarker records a parsed answer marker. Invalid markers remain visible to the scorer but
// never produce a cited evidence record.
type CitationMarker struct {
	Position              int
	RetrievedEvidenceRank int
	Valid                 bool
}

// MaterializedCaseResult retains separate retrieved and cited immutable records.
type MaterializedCaseResult struct {
	CorpusID            uuid.UUID
	SnapshotID          uuid.UUID
	State               CaseExecutionState
	Outcome             AgentOutcome
	Answer              string
	Retrieved           []ActualEvidence
	Cited               []ActualEvidence
	Markers             []CitationMarker
	LatencyMilliseconds *int64
	InputTokens         *int64
	OutputTokens        *int64
}

// MaterializeCaseResult validates the fixed corpus/snapshot boundary, parses markers in answer
// order, and returns detached evidence records. It intentionally does not persist or call agents.
func MaterializeCaseResult(corpusID, snapshotID uuid.UUID, result AgentCaseResult) (MaterializedCaseResult, error) {
	if corpusID == uuid.Nil || snapshotID == uuid.Nil || !validResultState(result) ||
		!validMeasurement(result.LatencyMilliseconds) || !validMeasurement(result.InputTokens) || !validMeasurement(result.OutputTokens) {
		return MaterializedCaseResult{}, ErrInvalidEvaluationResult
	}

	retrieved := make([]ActualEvidence, 0, len(result.Retrieved))
	for index, item := range result.Retrieved {
		if err := validateEvidenceProvenance(corpusID, snapshotID, item.Provenance); err != nil {
			return MaterializedCaseResult{}, err
		}
		retrieved = append(retrieved, ActualEvidence{
			Kind: EvidenceKindRetrieved, Position: index + 1, Provenance: item.Provenance,
		})
	}

	markers := ParseCitationMarkers(result.Answer, len(retrieved))
	cited := citedEvidence(markers, retrieved)
	return MaterializedCaseResult{
		CorpusID: corpusID, SnapshotID: snapshotID, State: result.State, Outcome: result.Outcome,
		Answer:    result.Answer,
		Retrieved: cloneActualEvidence(retrieved), Cited: cloneActualEvidence(cited), Markers: slices.Clone(markers),
		LatencyMilliseconds: cloneMeasurement(result.LatencyMilliseconds), InputTokens: cloneMeasurement(result.InputTokens),
		OutputTokens: cloneMeasurement(result.OutputTokens),
	}, nil
}

// ParseCitationMarkers maps every numeric [n] marker from left to right to the ordered retrieved
// evidence list. A marker is valid only when n is a one-based in-range evidence position.
func ParseCitationMarkers(answer string, retrievedCount int) []CitationMarker {
	matches := citationMarkerPattern.FindAllStringSubmatch(answer, -1)
	markers := make([]CitationMarker, 0, len(matches))
	for index, match := range matches {
		rank, err := strconv.Atoi(match[1])
		markers = append(markers, CitationMarker{
			Position: index + 1, RetrievedEvidenceRank: rank, Valid: err == nil && rank >= 1 && rank <= retrievedCount,
		})
	}
	return markers
}

func validResultState(result AgentCaseResult) bool {
	switch result.State {
	case CaseExecutionCompleted:
		return result.Outcome == AgentOutcomeCompleted && strings.TrimSpace(result.Answer) != ""
	case CaseExecutionAbstained:
		return result.Outcome == AgentOutcomeAbstained
	case CaseExecutionFailed, CaseExecutionCancelled:
		return result.Outcome == ""
	default:
		return false
	}
}

func validMeasurement(value *int64) bool {
	return value == nil || *value >= 0
}

func validateEvidenceProvenance(corpusID, snapshotID uuid.UUID, provenance EvidenceProvenance) error {
	if provenance.CorpusID != corpusID || provenance.SnapshotID != snapshotID {
		return ErrEvaluationEvidenceBoundary
	}
	if provenance.SourceID == uuid.Nil || provenance.SourceRevisionID == uuid.Nil || provenance.DocumentID == uuid.Nil ||
		provenance.UnitID == uuid.Nil || !canonicalEvidenceLocator.MatchString(provenance.CanonicalLocator) ||
		provenance.StartOffset < 0 || provenance.EndOffset <= provenance.StartOffset {
		return ErrInvalidEvidenceProvenance
	}
	if err := provenance.ContentSHA256.Validate(); err != nil {
		return fmt.Errorf("validate evaluation evidence content hash: %w", ErrInvalidEvidenceProvenance)
	}
	return nil
}

func citedEvidence(markers []CitationMarker, retrieved []ActualEvidence) []ActualEvidence {
	cited := make([]ActualEvidence, 0, len(markers))
	for _, marker := range markers {
		if !marker.Valid {
			continue
		}
		retrievedEvidence := retrieved[marker.RetrievedEvidenceRank-1]
		cited = append(cited, ActualEvidence{
			Kind: EvidenceKindCited, Position: len(cited) + 1, MarkerPosition: marker.Position,
			Provenance: retrievedEvidence.Provenance,
		})
	}
	return cited
}

func cloneActualEvidence(evidence []ActualEvidence) []ActualEvidence {
	return slices.Clone(evidence)
}

func cloneMeasurement(value *int64) *int64 {
	if value == nil {
		return nil
	}
	copy := *value
	return &copy
}

// MetricName identifies one deterministic scorer-v1 component.
type MetricName string

const (
	MetricRetrievalCoverage  MetricName = "retrieval_coverage"
	MetricCitationCoverage   MetricName = "citation_coverage"
	MetricCitationValidity   MetricName = "citation_validity"
	MetricCitationScope      MetricName = "citation_scope_validity"
	MetricExpectedAbstention MetricName = "expected_abstention_outcome"
	MetricExecutionState     MetricName = "execution_outcome"
	MetricLatency            MetricName = "latency_milliseconds"
	MetricInputTokens        MetricName = "input_tokens"
	MetricOutputTokens       MetricName = "output_tokens"
	MetricSemanticSupport    MetricName = "semantic_claim_support"
)

// MetricState declares whether a metric has a numeric score, is not applicable, needs review, or
// is deliberately absent because the case never completed.
type MetricState string

const (
	MetricStateScored           MetricState = "scored"
	MetricStateNotApplicable    MetricState = "not_applicable"
	MetricStateNotScored        MetricState = "not_scored"
	MetricStateNeedsHumanReview MetricState = "needs_human_review"
)

// Metric records explicit score arithmetic. Value is nil whenever State is not scored.
type Metric struct {
	Name        MetricName
	State       MetricState
	Value       *float64
	Numerator   int64
	Denominator int64
	Rationale   string
}

// CaseScore is the complete deterministic v1 result for one materialized case.
type CaseScore struct {
	ScoringPolicyVersion string
	Metrics              []Metric
}

// ScoreCaseV1 calculates evidence and operational metrics without judging answer prose. Expected
// evidence must already be fixed to the run snapshot by preflight.
func ScoreCaseV1(expectedOutcome domain.ExpectedOutcome, expected []EvidenceProvenance, actual MaterializedCaseResult) (CaseScore, error) {
	if !validExpectedOutcome(expectedOutcome) || actual.CorpusID == uuid.Nil || actual.SnapshotID == uuid.Nil ||
		!validMaterializedResult(actual) {
		return CaseScore{}, ErrInvalidScoringInput
	}
	for _, item := range expected {
		if err := validateEvidenceProvenance(actual.CorpusID, actual.SnapshotID, item); err != nil {
			return CaseScore{}, fmt.Errorf("validate expected evaluation evidence: %w", err)
		}
	}
	if actual.State == CaseExecutionFailed || actual.State == CaseExecutionCancelled {
		return CaseScore{ScoringPolicyVersion: ScoringPolicyV1, Metrics: notScoredMetrics(actual.State)}, nil
	}

	retrievalNumerator, retrievalDenominator := expectedCoverage(expected, actual.Retrieved)
	citationNumerator, citationDenominator := expectedCoverage(expected, actual.Cited)
	metrics := []Metric{
		coverageMetric(MetricRetrievalCoverage, retrievalNumerator, retrievalDenominator, "unique expected evidence retrieved from the fixed snapshot"),
		coverageMetric(MetricCitationCoverage, citationNumerator, citationDenominator, "unique expected evidence cited by answer markers"),
		citationValidityMetric(actual.Markers),
		citationScopeMetric(actual),
		expectedAbstentionMetric(expectedOutcome, actual.Outcome),
		scoredMetric(MetricExecutionState, 1, 1, "case reached a scoreable terminal state"),
		telemetryMetric(MetricLatency, actual.LatencyMilliseconds, "reported end-to-end latency"),
		telemetryMetric(MetricInputTokens, actual.InputTokens, "reported input token use"),
		telemetryMetric(MetricOutputTokens, actual.OutputTokens, "reported output token use"),
		{Name: MetricSemanticSupport, State: MetricStateNeedsHumanReview, Rationale: "claim-to-citation support requires a reviewed human rubric"},
	}
	return CaseScore{ScoringPolicyVersion: ScoringPolicyV1, Metrics: metrics}, nil
}

func validExpectedOutcome(outcome domain.ExpectedOutcome) bool {
	return outcome == domain.ExpectedOutcomeAnswer || outcome == domain.ExpectedOutcomeAbstain
}

func validMaterializedResult(actual MaterializedCaseResult) bool {
	result := AgentCaseResult{
		State: actual.State, Outcome: actual.Outcome, Answer: actual.Answer,
		LatencyMilliseconds: actual.LatencyMilliseconds, InputTokens: actual.InputTokens, OutputTokens: actual.OutputTokens,
	}
	if !validResultState(result) || !validMeasurement(actual.LatencyMilliseconds) || !validMeasurement(actual.InputTokens) || !validMeasurement(actual.OutputTokens) {
		return false
	}
	for index, evidence := range actual.Retrieved {
		if evidence.Kind != EvidenceKindRetrieved || evidence.Position != index+1 || evidence.MarkerPosition != 0 ||
			validateEvidenceProvenance(actual.CorpusID, actual.SnapshotID, evidence.Provenance) != nil {
			return false
		}
	}
	if !slices.Equal(actual.Markers, ParseCitationMarkers(actual.Answer, len(actual.Retrieved))) {
		return false
	}
	if !slices.Equal(actual.Cited, citedEvidence(actual.Markers, actual.Retrieved)) {
		return false
	}
	for index, evidence := range actual.Cited {
		if evidence.Kind != EvidenceKindCited || evidence.Position != index+1 || evidence.MarkerPosition < 1 ||
			validateEvidenceProvenance(actual.CorpusID, actual.SnapshotID, evidence.Provenance) != nil {
			return false
		}
	}
	return true
}

func expectedCoverage(expected []EvidenceProvenance, actual []ActualEvidence) (covered, denominator int) {
	expectedIdentities := make(map[string]struct{}, len(expected))
	for _, item := range expected {
		expectedIdentities[evidenceIdentity(item)] = struct{}{}
	}
	denominator = len(expectedIdentities)
	if denominator == 0 {
		return 0, 0
	}

	actualIdentities := make(map[string]struct{}, len(actual))
	for _, item := range actual {
		actualIdentities[evidenceIdentity(item.Provenance)] = struct{}{}
	}
	for identity := range expectedIdentities {
		if _, found := actualIdentities[identity]; found {
			covered++
		}
	}
	return covered, denominator
}

func evidenceIdentity(provenance EvidenceProvenance) string {
	return strings.Join([]string{
		provenance.SourceID.String(), provenance.SourceRevisionID.String(), provenance.DocumentID.String(), provenance.UnitID.String(),
		provenance.CanonicalLocator, strconv.Itoa(provenance.StartOffset), strconv.Itoa(provenance.EndOffset), string(provenance.ContentSHA256),
	}, "|")
}

func coverageMetric(name MetricName, numerator, denominator int, rationale string) Metric {
	if denominator == 0 {
		return Metric{Name: name, State: MetricStateNotApplicable, Rationale: "no expected evidence targets"}
	}
	return scoredMetric(name, int64(numerator), int64(denominator), rationale)
}

func citationValidityMetric(markers []CitationMarker) Metric {
	if len(markers) == 0 {
		return Metric{Name: MetricCitationValidity, State: MetricStateNotApplicable, Rationale: "answer contains no citation markers"}
	}
	valid := 0
	for _, marker := range markers {
		if marker.Valid {
			valid++
		}
	}
	return scoredMetric(MetricCitationValidity, int64(valid), int64(len(markers)), "citation markers resolve to ordered retrieved evidence")
}

func citationScopeMetric(actual MaterializedCaseResult) Metric {
	if len(actual.Markers) == 0 {
		return Metric{Name: MetricCitationScope, State: MetricStateNotApplicable, Rationale: "answer contains no citation markers"}
	}
	inScope := 0
	for _, marker := range actual.Markers {
		if marker.Valid {
			inScope++
		}
	}
	return scoredMetric(MetricCitationScope, int64(inScope), int64(len(actual.Markers)), "cited evidence remains within the fixed corpus snapshot")
}

func expectedAbstentionMetric(expected domain.ExpectedOutcome, actual AgentOutcome) Metric {
	if expected != domain.ExpectedOutcomeAbstain {
		return Metric{Name: MetricExpectedAbstention, State: MetricStateNotApplicable, Rationale: "case does not require abstention"}
	}
	if actual == AgentOutcomeAbstained {
		return scoredMetric(MetricExpectedAbstention, 1, 1, "agent abstained as required")
	}
	return scoredMetric(MetricExpectedAbstention, 0, 1, "agent answered when abstention was required")
}

func telemetryMetric(name MetricName, value *int64, rationale string) Metric {
	if value == nil {
		return Metric{Name: name, State: MetricStateNotApplicable, Rationale: "measurement was not reported"}
	}
	metricValue := float64(*value)
	return Metric{Name: name, State: MetricStateScored, Value: &metricValue, Numerator: *value, Denominator: 1, Rationale: rationale}
}

func scoredMetric(name MetricName, numerator, denominator int64, rationale string) Metric {
	value := float64(numerator) / float64(denominator)
	return Metric{Name: name, State: MetricStateScored, Value: &value, Numerator: numerator, Denominator: denominator, Rationale: rationale}
}

func notScoredMetrics(state CaseExecutionState) []Metric {
	names := []MetricName{
		MetricRetrievalCoverage, MetricCitationCoverage, MetricCitationValidity, MetricCitationScope, MetricExpectedAbstention,
		MetricExecutionState, MetricLatency, MetricInputTokens, MetricOutputTokens, MetricSemanticSupport,
	}
	metrics := make([]Metric, 0, len(names))
	for _, name := range names {
		metrics = append(metrics, Metric{Name: name, State: MetricStateNotScored, Rationale: string(state) + " cases are never assigned synthetic scores"})
	}
	return metrics
}
