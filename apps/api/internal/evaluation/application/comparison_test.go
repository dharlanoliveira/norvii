package application

import (
	"context"
	"errors"
	"math"
	"testing"

	"github.com/google/uuid"
)

func TestComparisonServiceRejectsEveryMismatchedStableKey(t *testing.T) {
	left := comparisonTestRun()
	right := left
	right.ID = uuid.New()
	right.Experiment = ExperimentIdentity{
		RetrievalStrategy: "hybrid", RetrievalConfigurationFingerprint: "different-configuration",
		AgentBuild: "agent-b", ChatModelIdentity: "chat-b", EmbeddingModelIdentity: "embedding-b",
	}

	result, err := NewComparisonService(comparisonStore{runs: map[uuid.UUID]ComparisonRun{
		left.ID: left, right.ID: right,
	}}).Compare(context.Background(), ComparisonRequest{LeftRunID: left.ID, RightRunID: right.ID})
	if err != nil {
		t.Fatalf("Compare() error = %v", err)
	}
	if result.State != ComparisonStateComparable || len(result.Differences) != 0 {
		t.Fatalf("Compare() = %#v, want comparable runs with different model/configuration identities", result)
	}

	for _, mismatch := range []struct {
		name  string
		field string
		apply func(*ComparisonKey)
	}{
		{"dataset revision", "dataset_revision_id", func(key *ComparisonKey) { key.DatasetRevisionID = uuid.New() }},
		{"dataset content", "dataset_content_sha256", func(key *ComparisonKey) { key.DatasetContentSHA256 = "other-dataset" }},
		{"corpus", "corpus_id", func(key *ComparisonKey) { key.CorpusID = uuid.New() }},
		{"snapshot", "snapshot_id", func(key *ComparisonKey) { key.SnapshotID = uuid.New() }},
		{"snapshot manifest", "snapshot_manifest_sha256", func(key *ComparisonKey) { key.SnapshotManifestSHA256 = "other-manifest" }},
		{"case set", "ordered_case_set_sha256", func(key *ComparisonKey) { key.OrderedCaseSetSHA256 = "other-case-set" }},
		{"scoring policy", "scoring_policy_version", func(key *ComparisonKey) { key.ScoringPolicyVersion = "v2" }},
	} {
		t.Run(mismatch.name, func(t *testing.T) {
			mismatched := right
			mismatched.Cases = cloneComparisonCases(right.Cases)
			mismatch.apply(&mismatched.Key)
			if mismatch.field == "scoring_policy_version" {
				for caseIndex := range mismatched.Cases {
					for metricIndex := range mismatched.Cases[caseIndex].Metrics {
						mismatched.Cases[caseIndex].Metrics[metricIndex].ScorerVersion = mismatched.Key.ScoringPolicyVersion
					}
				}
			}
			result, err := NewComparisonService(comparisonStore{runs: map[uuid.UUID]ComparisonRun{
				left.ID: left, mismatched.ID: mismatched,
			}}).Compare(context.Background(), ComparisonRequest{LeftRunID: left.ID, RightRunID: mismatched.ID})
			if err != nil {
				t.Fatalf("Compare() error = %v", err)
			}
			if result.State != ComparisonStateNonComparable || len(result.Metrics) != 0 || len(result.Differences) != 1 || result.Differences[0].Field != mismatch.field {
				t.Fatalf("Compare() = %#v, want explicit %q incompatibility without metrics", result, mismatch.field)
			}
		})
	}
}

func TestComparisonServiceUsesPairedProportionalMetricDenominators(t *testing.T) {
	left := comparisonTestRun()
	right := left
	right.ID = uuid.New()
	caseA, caseB, leftOnly, rightOnly := uuid.New(), uuid.New(), uuid.New(), uuid.New()
	left.Cases = []ComparisonCase{
		comparisonCase(caseA, RunCaseCompleted,
			comparisonScoredMetric(MetricRetrievalCoverage, 1, 2),
			comparisonScoredMetric(MetricLatency, 20, 1),
		),
		comparisonCase(caseB, RunCaseFailed),
		comparisonCase(leftOnly, RunCaseCompleted,
			comparisonScoredMetric(MetricRetrievalCoverage, 1, 1),
		),
	}
	right.Cases = []ComparisonCase{
		comparisonCase(caseA, RunCaseCompleted,
			comparisonScoredMetric(MetricRetrievalCoverage, 2, 3),
			comparisonScoredMetric(MetricLatency, 30, 1),
		),
		comparisonCase(caseB, RunCaseCancelled),
		comparisonCase(rightOnly, RunCaseCompleted,
			comparisonScoredMetric(MetricRetrievalCoverage, 1, 1),
		),
	}

	result, err := NewComparisonService(comparisonStore{runs: map[uuid.UUID]ComparisonRun{
		left.ID: left, right.ID: right,
	}}).Compare(context.Background(), ComparisonRequest{LeftRunID: left.ID, RightRunID: right.ID})
	if err != nil {
		t.Fatalf("Compare() error = %v", err)
	}
	if result.Totals != (ComparisonTotals{
		LeftCases: 3, RightCases: 3, PairedCases: 2, LeftUnpaired: 1, RightUnpaired: 1,
		FailedOrCancelled: 1, LeftFailed: 1, RightCancelled: 1,
	}) {
		t.Fatalf("Compare().Totals = %#v, want paired case denominators and explicit failures", result.Totals)
	}
	retrieval := comparisonMetric(t, result, MetricRetrievalCoverage)
	if retrieval.State != MetricStateScored || retrieval.PairedCases != 1 || retrieval.LeftNumerator != 1 || retrieval.LeftDenominator != 2 ||
		retrieval.RightNumerator != 2 || retrieval.RightDenominator != 3 || retrieval.LeftValue == nil || retrieval.RightValue == nil || retrieval.Delta == nil ||
		*retrieval.LeftValue != 0.5 || math.Abs(*retrieval.RightValue-2.0/3.0) > 1e-12 || math.Abs(*retrieval.Delta-(2.0/3.0-0.5)) > 1e-12 {
		t.Fatalf("retrieval comparison = %#v, want proportional arithmetic from the one jointly scored case", retrieval)
	}
	latency := comparisonMetric(t, result, MetricLatency)
	if latency.State != MetricStateScored || latency.PairedCases != 1 || latency.LeftNumerator != 20 || latency.RightNumerator != 30 || latency.Delta == nil || *latency.Delta != 10 {
		t.Fatalf("latency comparison = %#v, want paired telemetry delta", latency)
	}
}

func TestComparisonServiceRejectsMalformedTerminalMetricLedgers(t *testing.T) {
	tests := []struct {
		name    string
		corrupt func(*ComparisonRun)
	}{
		{
			name: "scored failed case",
			corrupt: func(run *ComparisonRun) {
				run.Cases[0].State = RunCaseFailed
				run.Cases[0].Metrics[0] = comparisonScoredMetric(MetricRetrievalCoverage, 1, 1)
			},
		},
		{
			name:    "mismatched scorer version",
			corrupt: func(run *ComparisonRun) { run.Cases[0].Metrics[0].ScorerVersion = "v2" },
		},
		{
			name:    "incomplete ledger",
			corrupt: func(run *ComparisonRun) { run.Cases[0].Metrics = run.Cases[0].Metrics[:len(run.Cases[0].Metrics)-1] },
		},
		{
			name: "duplicate metric",
			corrupt: func(run *ComparisonRun) {
				run.Cases[0].Metrics[len(run.Cases[0].Metrics)-1] = run.Cases[0].Metrics[0]
			},
		},
		{
			name:    "unknown metric",
			corrupt: func(run *ComparisonRun) { run.Cases[0].Metrics[0].Name = "unknown_metric" },
		},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			left := comparisonTestRun()
			right := comparisonTestRun()
			right.ID = uuid.New()
			testCase.corrupt(&left)
			_, err := NewComparisonService(comparisonStore{runs: map[uuid.UUID]ComparisonRun{
				left.ID: left, right.ID: right,
			}}).Compare(context.Background(), ComparisonRequest{LeftRunID: left.ID, RightRunID: right.ID})
			if !errors.Is(err, ErrInvalidComparisonRun) {
				t.Fatalf("Compare() error = %v, want %v", err, ErrInvalidComparisonRun)
			}
		})
	}
}

func TestComparisonServiceReturnsNonScoredResultsForEveryMetricWithoutJointScores(t *testing.T) {
	left := comparisonTestRun()
	right := comparisonTestRun()
	right.ID = uuid.New()
	right.Key = left.Key
	right.Cases[0].DatasetCaseID = left.Cases[0].DatasetCaseID

	result, err := NewComparisonService(comparisonStore{runs: map[uuid.UUID]ComparisonRun{
		left.ID: left, right.ID: right,
	}}).Compare(context.Background(), ComparisonRequest{LeftRunID: left.ID, RightRunID: right.ID})
	if err != nil {
		t.Fatalf("Compare() error = %v", err)
	}
	if len(result.Metrics) != len(requiredComparisonMetrics) {
		t.Fatalf("Compare().Metrics = %#v, want every required metric", result.Metrics)
	}
	for name := range requiredComparisonMetrics {
		metric := comparisonMetric(t, result, name)
		if metric.State != MetricStateNotScored || metric.PairedCases != 0 || metric.LeftValue != nil || metric.RightValue != nil || metric.Delta != nil {
			t.Fatalf("comparison metric %q = %#v, want an explicit non-scored result without arithmetic", name, metric)
		}
	}
}

func TestComparisonServiceRejectsInvalidRequestsAndMissingRuns(t *testing.T) {
	service := NewComparisonService(comparisonStore{})
	if _, err := service.Compare(context.Background(), ComparisonRequest{}); !errors.Is(err, ErrInvalidComparisonRequest) {
		t.Fatalf("Compare(empty request) error = %v, want %v", err, ErrInvalidComparisonRequest)
	}
	if _, err := service.Compare(context.Background(), ComparisonRequest{LeftRunID: uuid.New(), RightRunID: uuid.New()}); !errors.Is(err, ErrComparisonRunNotFound) {
		t.Fatalf("Compare(missing runs) error = %v, want %v", err, ErrComparisonRunNotFound)
	}
}

type comparisonStore struct{ runs map[uuid.UUID]ComparisonRun }

func (store comparisonStore) ComparisonRun(_ context.Context, runID uuid.UUID) (ComparisonRun, error) {
	run, found := store.runs[runID]
	if !found {
		return ComparisonRun{}, ErrComparisonRunNotFound
	}
	return run, nil
}

func comparisonTestRun() ComparisonRun {
	caseID := uuid.New()
	return ComparisonRun{
		ID: uuid.New(), State: RunCompleted,
		Key: ComparisonKey{
			DatasetRevisionID: uuid.New(), DatasetContentSHA256: "dataset", CorpusID: uuid.New(), SnapshotID: uuid.New(),
			SnapshotManifestSHA256: "manifest", OrderedCaseSetSHA256: "case-set", ScoringPolicyVersion: "v1",
		},
		Experiment: ExperimentIdentity{
			RetrievalStrategy: "vector", RetrievalConfigurationFingerprint: "configuration", AgentBuild: "agent-a",
			ChatModelIdentity: "chat-a", EmbeddingModelIdentity: "embedding-a",
		},
		Cases: []ComparisonCase{comparisonCase(caseID, RunCaseCompleted)},
	}
}

func cloneComparisonCases(cases []ComparisonCase) []ComparisonCase {
	clone := make([]ComparisonCase, len(cases))
	for caseIndex, evaluationCase := range cases {
		clone[caseIndex] = evaluationCase
		clone[caseIndex].Metrics = append([]ComparisonMetric(nil), evaluationCase.Metrics...)
	}
	return clone
}

func comparisonScoredMetric(name MetricName, numerator, denominator int64) ComparisonMetric {
	return ComparisonMetric{Name: name, State: MetricStateScored, ScorerVersion: ScoringPolicyV1, Numerator: &numerator, Denominator: &denominator}
}

func comparisonCase(datasetCaseID uuid.UUID, state RunCaseState, overrides ...ComparisonMetric) ComparisonCase {
	metrics := comparisonMetricLedger()
	indices := make(map[MetricName]int, len(metrics))
	for index, metric := range metrics {
		indices[metric.Name] = index
	}
	for _, metric := range overrides {
		metrics[indices[metric.Name]] = metric
	}
	return ComparisonCase{DatasetCaseID: datasetCaseID, State: state, Metrics: metrics}
}

func comparisonMetricLedger() []ComparisonMetric {
	names := []MetricName{
		MetricRetrievalCoverage, MetricCitationCoverage, MetricCitationValidity, MetricCitationScope, MetricExpectedAbstention,
		MetricExecutionState, MetricLatency, MetricInputTokens, MetricOutputTokens, MetricSemanticSupport,
	}
	metrics := make([]ComparisonMetric, 0, len(names))
	for _, name := range names {
		metrics = append(metrics, ComparisonMetric{Name: name, State: MetricStateNotScored, ScorerVersion: ScoringPolicyV1})
	}
	return metrics
}

func comparisonMetric(t *testing.T, result ComparisonResult, name MetricName) MetricDelta {
	t.Helper()
	for _, metric := range result.Metrics {
		if metric.Name == name {
			return metric
		}
	}
	t.Fatalf("comparison metrics = %#v, want %q", result.Metrics, name)
	return MetricDelta{}
}
