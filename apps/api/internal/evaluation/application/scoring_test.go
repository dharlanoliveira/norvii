package application

import (
	"errors"
	"slices"
	"testing"

	"github.com/dharlanoliveira/norvii/apps/api/internal/evaluation/domain"
	"github.com/google/uuid"
)

func TestMaterializeCaseResultSeparatesRetrievedAndCitedEvidence(t *testing.T) {
	t.Parallel()

	corpusID, snapshotID := scoringID(1), scoringID(2)
	result, err := MaterializeCaseResult(corpusID, snapshotID, AgentCaseResult{
		State: CaseExecutionCompleted, Outcome: AgentOutcomeCompleted, Answer: "A [2], then [1], and again [2].",
		Retrieved: []RetrievedEvidence{{Provenance: scoringEvidence(corpusID, snapshotID, 11)}, {Provenance: scoringEvidence(corpusID, snapshotID, 12)}},
	})
	if err != nil {
		t.Fatalf("MaterializeCaseResult() error = %v", err)
	}
	if got := evidencePositions(result.Retrieved); !slices.Equal(got, []int{1, 2}) {
		t.Fatalf("retrieved positions = %v, want [1 2]", got)
	}
	if got := evidencePositions(result.Cited); !slices.Equal(got, []int{1, 2, 3}) ||
		result.Cited[0].Provenance.UnitID != result.Retrieved[1].Provenance.UnitID ||
		result.Cited[1].Provenance.UnitID != result.Retrieved[0].Provenance.UnitID ||
		result.Cited[2].Provenance.UnitID != result.Retrieved[1].Provenance.UnitID {
		t.Fatalf("cited evidence = %#v, want answer-marker order detached from retrieved order", result.Cited)
	}
	result.Retrieved[0].Provenance.CanonicalLocator = "changed"
	if result.Cited[1].Provenance.CanonicalLocator == "changed" {
		t.Fatal("mutating retrieved materialization changed cited evidence")
	}
}

func TestParseCitationMarkers(t *testing.T) {
	t.Parallel()

	for _, testCase := range []struct {
		name      string
		answer    string
		count     int
		wantRanks []int
		wantValid []bool
	}{
		{name: "ordered repeated markers", answer: "[2] [1] [2]", count: 2, wantRanks: []int{2, 1, 2}, wantValid: []bool{true, true, true}},
		{name: "zero and out of range", answer: "[0] [3]", count: 2, wantRanks: []int{0, 3}, wantValid: []bool{false, false}},
		{name: "non numeric brackets are not markers", answer: "[source] [1]", count: 1, wantRanks: []int{1}, wantValid: []bool{true}},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			markers := ParseCitationMarkers(testCase.answer, testCase.count)
			if len(markers) != len(testCase.wantRanks) {
				t.Fatalf("marker count = %d, want %d", len(markers), len(testCase.wantRanks))
			}
			for index, marker := range markers {
				if marker.Position != index+1 || marker.RetrievedEvidenceRank != testCase.wantRanks[index] || marker.Valid != testCase.wantValid[index] {
					t.Fatalf("marker[%d] = %#v, want rank %d valid %t", index, marker, testCase.wantRanks[index], testCase.wantValid[index])
				}
			}
		})
	}
}

func TestMaterializeCaseResultRejectsUnsafeEvidenceAndResults(t *testing.T) {
	t.Parallel()

	corpusID, snapshotID := scoringID(1), scoringID(2)
	for _, testCase := range []struct {
		name   string
		mutate func(*AgentCaseResult)
		want   error
	}{
		{name: "cross corpus", mutate: func(result *AgentCaseResult) { result.Retrieved[0].Provenance.CorpusID = scoringID(99) }, want: ErrEvaluationEvidenceBoundary},
		{name: "cross snapshot", mutate: func(result *AgentCaseResult) { result.Retrieved[0].Provenance.SnapshotID = scoringID(99) }, want: ErrEvaluationEvidenceBoundary},
		{name: "missing source", mutate: func(result *AgentCaseResult) { result.Retrieved[0].Provenance.SourceID = uuid.Nil }, want: ErrInvalidEvidenceProvenance},
		{name: "missing document", mutate: func(result *AgentCaseResult) { result.Retrieved[0].Provenance.DocumentID = uuid.Nil }, want: ErrInvalidEvidenceProvenance},
		{name: "missing unit", mutate: func(result *AgentCaseResult) { result.Retrieved[0].Provenance.UnitID = uuid.Nil }, want: ErrInvalidEvidenceProvenance},
		{name: "invalid completed outcome", mutate: func(result *AgentCaseResult) { result.Outcome = AgentOutcomeAbstained }, want: ErrInvalidEvaluationResult},
		{name: "negative measurement", mutate: func(result *AgentCaseResult) { negative := int64(-1); result.InputTokens = &negative }, want: ErrInvalidEvaluationResult},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			result := validAgentResult(corpusID, snapshotID)
			testCase.mutate(&result)
			_, err := MaterializeCaseResult(corpusID, snapshotID, result)
			if !errors.Is(err, testCase.want) {
				t.Fatalf("MaterializeCaseResult() error = %v, want %v", err, testCase.want)
			}
		})
	}
}

func TestScoreCaseV1Metrics(t *testing.T) {
	t.Parallel()

	corpusID, snapshotID := scoringID(1), scoringID(2)
	expectedOne := scoringEvidence(corpusID, snapshotID, 11)
	expectedTwo := scoringEvidence(corpusID, snapshotID, 12)
	for _, testCase := range []struct {
		name     string
		expected domain.ExpectedOutcome
		result   AgentCaseResult
		metrics  map[MetricName]metricExpectation
	}{
		{
			name: "partial retrieval and citation coverage with valid scope", expected: domain.ExpectedOutcomeAnswer,
			result: AgentCaseResult{State: CaseExecutionCompleted, Outcome: AgentOutcomeCompleted, Answer: "answer [1]",
				Retrieved: []RetrievedEvidence{{Provenance: expectedOne}}},
			metrics: map[MetricName]metricExpectation{
				MetricRetrievalCoverage:  {state: MetricStateScored, numerator: 1, denominator: 2},
				MetricCitationCoverage:   {state: MetricStateScored, numerator: 1, denominator: 2},
				MetricCitationValidity:   {state: MetricStateScored, numerator: 1, denominator: 1},
				MetricCitationScope:      {state: MetricStateScored, numerator: 1, denominator: 1},
				MetricExpectedAbstention: {state: MetricStateNotApplicable},
				MetricExecutionState:     {state: MetricStateScored, numerator: 1, denominator: 1},
				MetricSemanticSupport:    {state: MetricStateNeedsHumanReview},
			},
		},
		{
			name: "invalid marker records invalid citation and scope", expected: domain.ExpectedOutcomeAnswer,
			result: AgentCaseResult{State: CaseExecutionCompleted, Outcome: AgentOutcomeCompleted, Answer: "answer [3]",
				Retrieved: []RetrievedEvidence{{Provenance: expectedOne}}},
			metrics: map[MetricName]metricExpectation{
				MetricCitationCoverage: {state: MetricStateScored, numerator: 0, denominator: 2},
				MetricCitationValidity: {state: MetricStateScored, numerator: 0, denominator: 1},
				MetricCitationScope:    {state: MetricStateScored, numerator: 0, denominator: 1},
			},
		},
		{
			name: "required abstention succeeds", expected: domain.ExpectedOutcomeAbstain,
			result:  AgentCaseResult{State: CaseExecutionAbstained, Outcome: AgentOutcomeAbstained},
			metrics: map[MetricName]metricExpectation{MetricExpectedAbstention: {state: MetricStateScored, numerator: 1, denominator: 1}},
		},
		{
			name: "required abstention answered", expected: domain.ExpectedOutcomeAbstain,
			result:  AgentCaseResult{State: CaseExecutionCompleted, Outcome: AgentOutcomeCompleted, Answer: "answer", Retrieved: []RetrievedEvidence{{Provenance: expectedOne}}},
			metrics: map[MetricName]metricExpectation{MetricExpectedAbstention: {state: MetricStateScored, numerator: 0, denominator: 1}},
		},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			actual, err := MaterializeCaseResult(corpusID, snapshotID, testCase.result)
			if err != nil {
				t.Fatalf("MaterializeCaseResult() error = %v", err)
			}
			score, err := ScoreCaseV1(testCase.expected, []EvidenceProvenance{expectedOne, expectedTwo}, actual)
			if err != nil {
				t.Fatalf("ScoreCaseV1() error = %v", err)
			}
			for name, want := range testCase.metrics {
				assertMetric(t, score, name, want)
			}
		})
	}
}

func TestScoreCaseV1CoverageDeduplicatesExpectedEvidence(t *testing.T) {
	t.Parallel()

	corpusID, snapshotID := scoringID(1), scoringID(2)
	expected := scoringEvidence(corpusID, snapshotID, 11)
	actual, err := MaterializeCaseResult(corpusID, snapshotID, AgentCaseResult{
		State: CaseExecutionCompleted, Outcome: AgentOutcomeCompleted, Answer: "answer [1]",
		Retrieved: []RetrievedEvidence{{Provenance: expected}},
	})
	if err != nil {
		t.Fatalf("MaterializeCaseResult() error = %v", err)
	}

	score, err := ScoreCaseV1(domain.ExpectedOutcomeAnswer, []EvidenceProvenance{expected, expected}, actual)
	if err != nil {
		t.Fatalf("ScoreCaseV1() error = %v", err)
	}
	assertMetric(t, score, MetricRetrievalCoverage, metricExpectation{state: MetricStateScored, numerator: 1, denominator: 1})
	assertMetric(t, score, MetricCitationCoverage, metricExpectation{state: MetricStateScored, numerator: 1, denominator: 1})
}

func TestScoreCaseV1TelemetryAndFailureOutcomes(t *testing.T) {
	t.Parallel()

	corpusID, snapshotID := scoringID(1), scoringID(2)
	latency, input, output := int64(120), int64(7), int64(11)
	for _, testCase := range []struct {
		name       string
		result     AgentCaseResult
		wantState  MetricState
		wantMetric metricExpectation
	}{
		{
			name:       "reported telemetry has explicit observation denominator",
			result:     AgentCaseResult{State: CaseExecutionCompleted, Outcome: AgentOutcomeCompleted, Answer: "answer", LatencyMilliseconds: &latency, InputTokens: &input, OutputTokens: &output},
			wantMetric: metricExpectation{state: MetricStateScored, numerator: 120, denominator: 1},
		},
		{
			name:       "missing telemetry is not applicable",
			result:     AgentCaseResult{State: CaseExecutionCompleted, Outcome: AgentOutcomeCompleted, Answer: "answer"},
			wantMetric: metricExpectation{state: MetricStateNotApplicable},
		},
		{
			name:   "failed case is never synthetic zero",
			result: AgentCaseResult{State: CaseExecutionFailed}, wantState: MetricStateNotScored,
		},
		{
			name:   "cancelled case is never synthetic zero",
			result: AgentCaseResult{State: CaseExecutionCancelled}, wantState: MetricStateNotScored,
		},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			assertTelemetryOutcome(t, corpusID, snapshotID, testCase)
		})
	}
}

func assertTelemetryOutcome(t *testing.T, corpusID, snapshotID uuid.UUID, testCase struct {
	name       string
	result     AgentCaseResult
	wantState  MetricState
	wantMetric metricExpectation
}) {
	t.Helper()
	actual, err := MaterializeCaseResult(corpusID, snapshotID, testCase.result)
	if err != nil {
		t.Fatalf("MaterializeCaseResult() error = %v", err)
	}
	score, err := ScoreCaseV1(domain.ExpectedOutcomeAnswer, nil, actual)
	if err != nil {
		t.Fatalf("ScoreCaseV1() error = %v", err)
	}
	if testCase.wantState != "" {
		assertAllMetricsNotScored(t, score, testCase.wantState)
		return
	}
	assertMetric(t, score, MetricLatency, testCase.wantMetric)
	if testCase.wantMetric.state == MetricStateScored {
		assertMetric(t, score, MetricInputTokens, metricExpectation{state: MetricStateScored, numerator: 7, denominator: 1})
		assertMetric(t, score, MetricOutputTokens, metricExpectation{state: MetricStateScored, numerator: 11, denominator: 1})
	}
}

func assertAllMetricsNotScored(t *testing.T, score CaseScore, want MetricState) {
	t.Helper()
	for _, metric := range score.Metrics {
		if metric.State != want || metric.Value != nil || metric.Numerator != 0 || metric.Denominator != 0 {
			t.Fatalf("metric = %#v, want not_scored without a synthetic value", metric)
		}
	}
}

type metricExpectation struct {
	state       MetricState
	numerator   int64
	denominator int64
}

func assertMetric(t *testing.T, score CaseScore, name MetricName, want metricExpectation) {
	t.Helper()
	for _, metric := range score.Metrics {
		if metric.Name == name {
			if metric.State != want.state || metric.Numerator != want.numerator || metric.Denominator != want.denominator {
				t.Fatalf("metric %q = %#v, want state %q numerator %d denominator %d", name, metric, want.state, want.numerator, want.denominator)
			}
			return
		}
	}
	t.Fatalf("metric %q was not produced", name)
}

func validAgentResult(corpusID, snapshotID uuid.UUID) AgentCaseResult {
	return AgentCaseResult{
		State: CaseExecutionCompleted, Outcome: AgentOutcomeCompleted, Answer: "answer [1]",
		Retrieved: []RetrievedEvidence{{Provenance: scoringEvidence(corpusID, snapshotID, 11)}},
	}
}

func scoringEvidence(corpusID, snapshotID uuid.UUID, seed byte) EvidenceProvenance {
	return EvidenceProvenance{
		CorpusID: corpusID, SnapshotID: snapshotID, SourceID: scoringID(seed), SourceRevisionID: scoringID(seed + 20),
		DocumentID: scoringID(seed + 40), UnitID: scoringID(seed + 60), CanonicalLocator: "article:1/item:a",
		StartOffset: 0, EndOffset: 10, ContentSHA256: domain.SHA256("aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"),
	}
}

func scoringID(seed byte) uuid.UUID {
	return uuid.UUID{15: seed}
}

func evidencePositions(evidence []ActualEvidence) []int {
	positions := make([]int, 0, len(evidence))
	for _, item := range evidence {
		positions = append(positions, item.Position)
	}
	return positions
}
