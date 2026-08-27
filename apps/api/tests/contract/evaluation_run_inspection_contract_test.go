package contract_test

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"

	evaluationcontract "github.com/dharlanoliveira/norvii/apps/api/tests/contract"
)

func TestEvaluationRunInspectionFixturesSeparateEvidenceAndKeepDiagnosticsSafe(t *testing.T) {
	var summary evaluationcontract.EvaluationRunSummaryResponse
	decodeDatasetInspectionFixture(t, "evaluation-run-summary-response.json", &summary)
	assertRunSummaryFixture(t, summary)

	var result evaluationcontract.EvaluationRunCaseResponse
	decodeDatasetInspectionFixture(t, "evaluation-run-case-response.json", &result)
	assertRunCaseFixture(t, result)
	assertEvaluationFixtureDoesNotExposeProviderData(t, summary, result)
}

func assertRunSummaryFixture(t *testing.T, summary evaluationcontract.EvaluationRunSummaryResponse) {
	t.Helper()
	if !validDatasetFixtureUUID(summary.ID) || !validDatasetFixtureUUID(summary.CorpusID) || !validDatasetFixtureUUID(summary.SnapshotID) || !validDatasetFixtureHash(summary.DatasetRevision.ContentSHA256) || !validDatasetFixtureHash(summary.SnapshotManifestSHA256) || !validDatasetFixtureHash(summary.OrderedCaseSetSHA256) || summary.Aggregate.Total != 2 || summary.Aggregate.Failed != 1 {
		t.Fatalf("invalid run summary fixture: %#v", summary)
	}
	if len(summary.Aggregate.Metrics) != 1 || strings.TrimSpace(summary.Aggregate.Metrics[0].Rationale) == "" || summary.Aggregate.Metrics[0].Numerator == nil || summary.Aggregate.Metrics[0].Denominator == nil {
		t.Fatalf("run metric is missing rationale or arithmetic: %#v", summary.Aggregate.Metrics)
	}
	if len(summary.Cases) != 2 || summary.Cases[1].FailureCode != "provider_unavailable" {
		t.Fatalf("run cases do not preserve safe failure state: %#v", summary.Cases)
	}
}

func assertRunCaseFixture(t *testing.T, result evaluationcontract.EvaluationRunCaseResponse) {
	t.Helper()
	if len(result.ExpectedEvidence) != 1 || len(result.ActualEvidence) != 2 || result.ExpectedEvidence[0].Kind != "expected" || result.ActualEvidence[0].Kind != "retrieved" || result.ActualEvidence[1].Kind != "cited" {
		t.Fatalf("case evidence is not explicitly separated: expected=%#v actual=%#v", result.ExpectedEvidence, result.ActualEvidence)
	}
	assertFixtureEvidenceProvenance(t, result)
	assertFixtureTelemetry(t, result)
	assertFixtureEvidenceOffsets(t, result)
	if len(result.Metrics) != 1 || strings.TrimSpace(result.Metrics[0].Rationale) == "" {
		t.Fatalf("case metric rationale is absent: %#v", result.Metrics)
	}
}

func assertFixtureEvidenceProvenance(t *testing.T, result evaluationcontract.EvaluationRunCaseResponse) {
	t.Helper()
	for _, item := range append(result.ExpectedEvidence, result.ActualEvidence...) {
		if item.CorpusID != result.CorpusID || item.SnapshotID != result.SnapshotID || !validDatasetFixtureUUID(item.SourceID) || !validDatasetFixtureHash(item.ContentSHA256) {
			t.Fatalf("evidence does not preserve immutable run provenance: %#v", item)
		}
	}
}

func assertFixtureTelemetry(t *testing.T, result evaluationcontract.EvaluationRunCaseResponse) {
	t.Helper()
	if result.GraphGroundingState != "not_requested" || result.LatencyMilliseconds == nil || *result.LatencyMilliseconds != 42 || result.InputTokens == nil || *result.InputTokens != 7 || result.OutputTokens == nil || *result.OutputTokens != 11 {
		t.Fatalf("case telemetry or graph grounding is absent: %#v", result)
	}
}

func assertFixtureEvidenceOffsets(t *testing.T, result evaluationcontract.EvaluationRunCaseResponse) {
	t.Helper()
	if result.ExpectedEvidence[0].DisplayLocator != "Synthetic section 1" || result.ActualEvidence[0].DisplayLocator != "Synthetic section 1" || result.ActualEvidence[0].StartOffset == nil || *result.ActualEvidence[0].StartOffset != 0 || result.ActualEvidence[0].EndOffset == nil || *result.ActualEvidence[0].EndOffset != 42 {
		t.Fatalf("case evidence display locator or offsets are absent: expected=%#v actual=%#v", result.ExpectedEvidence, result.ActualEvidence)
	}
}

func TestEvaluationComparisonFixturesPreventDeltasForMismatchedIdentities(t *testing.T) {
	var comparable evaluationcontract.EvaluationComparisonResponse
	decodeDatasetInspectionFixture(t, "evaluation-comparison-response.json", &comparable)
	if comparable.ComparisonState != "comparable" || comparable.Totals == nil || len(comparable.Differences) != 0 || len(comparable.Metrics) != 1 ||
		comparable.Metrics[0].Delta == nil || comparable.Metrics[0].PairedCases != 1 || comparable.Totals.FailedOrCancelled != 1 {
		t.Fatalf("invalid comparable fixture: %#v", comparable)
	}

	var nonComparable evaluationcontract.EvaluationComparisonResponse
	decodeDatasetInspectionFixture(t, "evaluation-comparison-non-comparable-response.json", &nonComparable)
	if nonComparable.ComparisonState != "non_comparable" || nonComparable.Totals != nil || len(nonComparable.Differences) != 1 ||
		nonComparable.Differences[0].Field != "snapshot_manifest_sha256" || len(nonComparable.Metrics) != 0 {
		t.Fatalf("non-comparable fixture exposed direct quality arithmetic: %#v", nonComparable)
	}
	assertEvaluationFixtureDoesNotExposeProviderData(t, comparable, nonComparable)
}

func assertEvaluationFixtureDoesNotExposeProviderData(t *testing.T, fixtures ...any) {
	t.Helper()
	for _, fixture := range fixtures {
		encoded, err := json.Marshal(fixture)
		if err != nil {
			t.Fatalf("marshal fixture: %v", err)
		}
		for _, forbidden := range []string{"providerpayload", "prompt", "credential", "rawdocument", "systemmessage"} {
			if bytes.Contains(bytes.ToLower(encoded), []byte(forbidden)) {
				t.Fatalf("fixture disclosed %q: %s", forbidden, encoded)
			}
		}
	}
}
