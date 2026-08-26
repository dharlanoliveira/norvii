package application

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"sort"

	"github.com/google/uuid"
)

var (
	// ErrInvalidComparisonRequest identifies a request that cannot name two persisted runs.
	ErrInvalidComparisonRequest = errors.New("evaluation comparison request is invalid")
	// ErrComparisonRunNotFound identifies an absent historical evaluation run.
	ErrComparisonRunNotFound = errors.New("evaluation comparison run was not found")
	// ErrInvalidComparisonRun identifies a persisted comparison projection that violates its ledger contract.
	ErrInvalidComparisonRun = errors.New("evaluation comparison run is invalid")
)

// ComparisonState makes an incompatible request explicit instead of assigning it quality deltas.
type ComparisonState string

const (
	ComparisonStateComparable    ComparisonState = "comparable"
	ComparisonStateNonComparable ComparisonState = "non_comparable"
)

// ComparisonKey contains every immutable identity that must remain equal for a direct quality
// delta. Dataset and snapshot identity preserve the fixture and bound source set; the policy
// version preserves the deterministic rubric. Model and retrieval configuration are deliberately
// excluded so an experiment can compare them while keeping the fixture fixed.
type ComparisonKey struct {
	DatasetRevisionID      uuid.UUID
	DatasetContentSHA256   string
	CorpusID               uuid.UUID
	SnapshotID             uuid.UUID
	SnapshotManifestSHA256 string
	OrderedCaseSetSHA256   string
	ScoringPolicyVersion   string
}

// ExperimentIdentity names the allowed variables of a comparable experiment.
type ExperimentIdentity struct {
	RetrievalStrategy                 string
	RetrievalConfigurationFingerprint string
	AgentBuild                        string
	ChatModelIdentity                 string
	EmbeddingModelIdentity            string
}

// ComparisonMetric is the stored arithmetic for one case metric. Non-scored states have nil
// arithmetic and are never treated as zero.
type ComparisonMetric struct {
	Name          MetricName
	State         MetricState
	ScorerVersion string
	Numerator     *int64
	Denominator   *int64
}

// ComparisonCase binds stored metrics to the immutable dataset case rather than a run-local case
// ID, which makes pairing stable across repeat runs.
type ComparisonCase struct {
	DatasetCaseID uuid.UUID
	State         RunCaseState
	Metrics       []ComparisonMetric
}

// ComparisonRun is the complete historical input required for a read-only direct comparison.
type ComparisonRun struct {
	ID         uuid.UUID
	State      RunState
	Key        ComparisonKey
	Experiment ExperimentIdentity
	Cases      []ComparisonCase
}

// comparisonRunReader reads immutable historical records. It must not resolve an active snapshot or
// recompute metrics from mutable catalog data.
type comparisonRunReader interface {
	ComparisonRun(context.Context, uuid.UUID) (ComparisonRun, error)
}

// ComparisonRequest identifies the two persisted historical runs to inspect.
type ComparisonRequest struct {
	LeftRunID  uuid.UUID
	RightRunID uuid.UUID
}

// ComparisonDifference identifies one stable key component that prevents direct deltas.
type ComparisonDifference struct {
	Field string
}

// ComparisonTotals reports paired case eligibility across all metric components.
type ComparisonTotals struct {
	LeftCases         int64
	RightCases        int64
	PairedCases       int64
	LeftUnpaired      int64
	RightUnpaired     int64
	FailedOrCancelled int64
	LeftFailed        int64
	LeftCancelled     int64
	RightFailed       int64
	RightCancelled    int64
}

// MetricDelta derives proportional values from summed paired numerators and denominators. Delta
// is right minus left and is nil when the pair has no jointly scored observations.
type MetricDelta struct {
	Name             MetricName
	State            MetricState
	PairedCases      int64
	LeftNumerator    int64
	LeftDenominator  int64
	RightNumerator   int64
	RightDenominator int64
	LeftValue        *float64
	RightValue       *float64
	Delta            *float64
}

// ComparisonResult is either a direct delta set or an explicit non-comparable outcome.
type ComparisonResult struct {
	State       ComparisonState
	Left        ExperimentIdentity
	Right       ExperimentIdentity
	Differences []ComparisonDifference
	Totals      ComparisonTotals
	Metrics     []MetricDelta
}

// ComparisonService compares immutable run records loaded by the caller-owned persistence store.
type ComparisonService struct{ store comparisonRunReader }

// NewComparisonService constructs a read-only historical comparison capability.
func NewComparisonService(store comparisonRunReader) *ComparisonService {
	return &ComparisonService{store: store}
}

// Compare returns an explicit non-comparable result for a mismatched stable identity. For a
// compatible identity it returns every required metric, with arithmetic only when both matching
// dataset cases have scored values, preventing failures, cancellations, and non-scored values
// from becoming fabricated zeros.
func (service *ComparisonService) Compare(ctx context.Context, request ComparisonRequest) (ComparisonResult, error) {
	if service == nil || service.store == nil || request.LeftRunID == uuid.Nil || request.RightRunID == uuid.Nil {
		return ComparisonResult{}, ErrInvalidComparisonRequest
	}
	left, err := service.store.ComparisonRun(ctx, request.LeftRunID)
	if err != nil {
		return ComparisonResult{}, fmt.Errorf("read left evaluation comparison run: %w", err)
	}
	right, err := service.store.ComparisonRun(ctx, request.RightRunID)
	if err != nil {
		return ComparisonResult{}, fmt.Errorf("read right evaluation comparison run: %w", err)
	}
	if err := validateComparisonRun(left, request.LeftRunID); err != nil {
		return ComparisonResult{}, err
	}
	if err := validateComparisonRun(right, request.RightRunID); err != nil {
		return ComparisonResult{}, err
	}

	result := ComparisonResult{Left: left.Experiment, Right: right.Experiment}
	if differences := comparisonDifferences(left.Key, right.Key); len(differences) > 0 {
		result.State = ComparisonStateNonComparable
		result.Differences = differences
		return result, nil
	}
	result.State = ComparisonStateComparable
	result.Totals, result.Metrics = compareCases(left.Cases, right.Cases)
	return result, nil
}

func validateComparisonRun(run ComparisonRun, wantID uuid.UUID) error {
	if run.ID != wantID || run.Key.DatasetRevisionID == uuid.Nil || run.Key.CorpusID == uuid.Nil || run.Key.SnapshotID == uuid.Nil ||
		run.Key.DatasetContentSHA256 == "" || run.Key.SnapshotManifestSHA256 == "" || run.Key.OrderedCaseSetSHA256 == "" || run.Key.ScoringPolicyVersion == "" {
		return ErrInvalidComparisonRun
	}
	if run.State != RunCompleted && run.State != RunCompletedWithFailures && run.State != RunFailed {
		return ErrInvalidComparisonRun
	}
	if err := validateComparisonCases(run.Cases, run.Key.ScoringPolicyVersion); err != nil {
		return err
	}
	return nil
}

func validateComparisonCases(cases []ComparisonCase, scoringPolicyVersion string) error {
	if len(cases) == 0 {
		return ErrInvalidComparisonRun
	}
	seenCases := make(map[uuid.UUID]struct{}, len(cases))
	for _, evaluationCase := range cases {
		if err := validateComparisonCase(evaluationCase, scoringPolicyVersion, seenCases); err != nil {
			return err
		}
	}
	return nil
}

func validateComparisonCase(evaluationCase ComparisonCase, scoringPolicyVersion string, seenCases map[uuid.UUID]struct{}) error {
	if evaluationCase.DatasetCaseID == uuid.Nil || !terminalComparisonCaseState(evaluationCase.State) {
		return ErrInvalidComparisonRun
	}
	if _, found := seenCases[evaluationCase.DatasetCaseID]; found {
		return ErrInvalidComparisonRun
	}
	seenCases[evaluationCase.DatasetCaseID] = struct{}{}
	return validateComparisonMetrics(evaluationCase, scoringPolicyVersion)
}

func validateComparisonMetrics(evaluationCase ComparisonCase, scoringPolicyVersion string) error {
	if len(evaluationCase.Metrics) != len(requiredComparisonMetrics) {
		return ErrInvalidComparisonRun
	}
	seenMetrics := make(map[MetricName]struct{}, len(evaluationCase.Metrics))
	for _, metric := range evaluationCase.Metrics {
		if err := validateComparisonMetric(evaluationCase.State, metric, scoringPolicyVersion, seenMetrics); err != nil {
			return err
		}
	}
	if len(seenMetrics) != len(requiredComparisonMetrics) {
		return ErrInvalidComparisonRun
	}
	return nil
}

func validateComparisonMetric(caseState RunCaseState, metric ComparisonMetric, scoringPolicyVersion string, seen map[MetricName]struct{}) error {
	if _, required := requiredComparisonMetrics[metric.Name]; !required || metric.ScorerVersion != scoringPolicyVersion {
		return ErrInvalidComparisonRun
	}
	if _, found := seen[metric.Name]; found {
		return ErrInvalidComparisonRun
	}
	seen[metric.Name] = struct{}{}
	if terminalFailure(caseState) && metric.State == MetricStateScored {
		return ErrInvalidComparisonRun
	}
	if metric.State == MetricStateScored && !scoredComparisonMetric(metric) {
		return ErrInvalidComparisonRun
	}
	if metric.State != MetricStateScored && !validUnscoredComparisonMetric(metric) {
		return ErrInvalidComparisonRun
	}
	return nil
}

func validUnscoredComparisonMetric(metric ComparisonMetric) bool {
	return (metric.State == MetricStateNotApplicable || metric.State == MetricStateNotScored || metric.State == MetricStateNeedsHumanReview) &&
		metric.Numerator == nil && metric.Denominator == nil
}

var requiredComparisonMetrics = map[MetricName]struct{}{
	MetricRetrievalCoverage: {}, MetricCitationCoverage: {}, MetricCitationValidity: {}, MetricCitationScope: {},
	MetricExpectedAbstention: {}, MetricExecutionState: {}, MetricLatency: {}, MetricInputTokens: {},
	MetricOutputTokens: {}, MetricSemanticSupport: {},
}

func terminalComparisonCaseState(state RunCaseState) bool {
	return state == RunCaseCompleted || state == RunCaseAbstained || state == RunCaseFailed || state == RunCaseCancelled
}

func comparisonDifferences(left, right ComparisonKey) []ComparisonDifference {
	differences := make([]ComparisonDifference, 0, 7)
	for _, difference := range []struct {
		field string
		equal bool
	}{
		{"dataset_revision_id", left.DatasetRevisionID == right.DatasetRevisionID},
		{"dataset_content_sha256", left.DatasetContentSHA256 == right.DatasetContentSHA256},
		{"corpus_id", left.CorpusID == right.CorpusID},
		{"snapshot_id", left.SnapshotID == right.SnapshotID},
		{"snapshot_manifest_sha256", left.SnapshotManifestSHA256 == right.SnapshotManifestSHA256},
		{"ordered_case_set_sha256", left.OrderedCaseSetSHA256 == right.OrderedCaseSetSHA256},
		{"scoring_policy_version", left.ScoringPolicyVersion == right.ScoringPolicyVersion},
	} {
		if !difference.equal {
			differences = append(differences, ComparisonDifference{Field: difference.field})
		}
	}
	return differences
}

func compareCases(left, right []ComparisonCase) (ComparisonTotals, []MetricDelta) {
	leftByDatasetCase := indexComparisonCases(left)
	rightByDatasetCase := indexComparisonCases(right)
	totals := ComparisonTotals{LeftCases: int64(len(leftByDatasetCase)), RightCases: int64(len(rightByDatasetCase))}
	metricTotals := newMetricDeltas()
	for caseID, leftCase := range leftByDatasetCase {
		rightCase, paired := rightByDatasetCase[caseID]
		if !paired {
			totals.LeftUnpaired++
			continue
		}
		totals.PairedCases++
		if terminalFailure(leftCase.State) || terminalFailure(rightCase.State) {
			totals.FailedOrCancelled++
			continue
		}
		compareCaseMetrics(leftCase, rightCase, metricTotals)
	}
	for caseID := range rightByDatasetCase {
		if _, paired := leftByDatasetCase[caseID]; !paired {
			totals.RightUnpaired++
		}
	}
	for _, evaluationCase := range leftByDatasetCase {
		switch evaluationCase.State {
		case RunCaseFailed:
			totals.LeftFailed++
		case RunCaseCancelled:
			totals.LeftCancelled++
		}
	}
	for _, evaluationCase := range rightByDatasetCase {
		switch evaluationCase.State {
		case RunCaseFailed:
			totals.RightFailed++
		case RunCaseCancelled:
			totals.RightCancelled++
		}
	}
	return totals, sortedMetricDeltas(metricTotals)
}

func newMetricDeltas() map[MetricName]*MetricDelta {
	deltas := make(map[MetricName]*MetricDelta, len(requiredComparisonMetrics))
	for name := range requiredComparisonMetrics {
		deltas[name] = &MetricDelta{Name: name, State: MetricStateNotScored}
	}
	return deltas
}

func indexComparisonCases(cases []ComparisonCase) map[uuid.UUID]ComparisonCase {
	indexed := make(map[uuid.UUID]ComparisonCase, len(cases))
	for _, evaluationCase := range cases {
		if evaluationCase.DatasetCaseID != uuid.Nil {
			indexed[evaluationCase.DatasetCaseID] = evaluationCase
		}
	}
	return indexed
}

func terminalFailure(state RunCaseState) bool {
	return state == RunCaseFailed || state == RunCaseCancelled
}

func compareCaseMetrics(left, right ComparisonCase, metricTotals map[MetricName]*MetricDelta) {
	rightByName := make(map[MetricName]ComparisonMetric, len(right.Metrics))
	for _, metric := range right.Metrics {
		rightByName[metric.Name] = metric
	}
	for _, leftMetric := range left.Metrics {
		rightMetric, found := rightByName[leftMetric.Name]
		if !found || !scoredComparisonMetric(leftMetric) || !scoredComparisonMetric(rightMetric) {
			continue
		}
		delta := metricTotals[leftMetric.Name]
		delta.PairedCases++
		delta.State = MetricStateScored
		delta.LeftNumerator += *leftMetric.Numerator
		delta.LeftDenominator += *leftMetric.Denominator
		delta.RightNumerator += *rightMetric.Numerator
		delta.RightDenominator += *rightMetric.Denominator
	}
}

func scoredComparisonMetric(metric ComparisonMetric) bool {
	return metric.Name != "" && metric.State == MetricStateScored && metric.Numerator != nil && metric.Denominator != nil &&
		*metric.Numerator >= 0 && *metric.Denominator > 0
}

func sortedMetricDeltas(metricTotals map[MetricName]*MetricDelta) []MetricDelta {
	metrics := make([]MetricDelta, 0, len(metricTotals))
	for _, metric := range metricTotals {
		if metric.State != MetricStateScored {
			metrics = append(metrics, *metric)
			continue
		}
		leftValue := float64(metric.LeftNumerator) / float64(metric.LeftDenominator)
		rightValue := float64(metric.RightNumerator) / float64(metric.RightDenominator)
		deltaValue := rightValue - leftValue
		metric.LeftValue = &leftValue
		metric.RightValue = &rightValue
		metric.Delta = &deltaValue
		metrics = append(metrics, *metric)
	}
	sort.Slice(metrics, func(left, right int) bool { return metrics[left].Name < metrics[right].Name })
	return slices.Clone(metrics)
}
