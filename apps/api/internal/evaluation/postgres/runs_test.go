package postgres

import (
	"testing"

	"github.com/dharlanoliveira/norvii/apps/api/internal/evaluation/application"
)

func TestMetricPersistenceValuesUsesNullArithmeticForNonScoredMetrics(t *testing.T) {
	scoredValue := 0.5
	tests := []struct {
		name       string
		metric     application.Metric
		wantScored bool
	}{
		{
			name: "scored",
			metric: application.Metric{
				State: application.MetricStateScored, Value: &scoredValue, Numerator: 1, Denominator: 2,
			},
			wantScored: true,
		},
		{
			name:   "not applicable",
			metric: application.Metric{State: application.MetricStateNotApplicable},
		},
		{
			name:   "not scored",
			metric: application.Metric{State: application.MetricStateNotScored},
		},
		{
			name:   "needs human review",
			metric: application.Metric{State: application.MetricStateNeedsHumanReview},
		},
	}
	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			value, numerator, denominator := metricPersistenceValues(testCase.metric)
			if testCase.wantScored {
				if value == nil || numerator == nil || denominator == nil || *value != 0.5 || *numerator != 1 || *denominator != 2 {
					t.Fatalf("metricPersistenceValues() = (%v, %v, %v), want scored arithmetic", value, numerator, denominator)
				}
				return
			}
			if value != nil || numerator != nil || denominator != nil {
				t.Fatalf("metricPersistenceValues() = (%v, %v, %v), want null arithmetic", value, numerator, denominator)
			}
		})
	}
}
