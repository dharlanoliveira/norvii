import { describe, expect, it } from "vitest";

import comparisonFixture from "../../../../contracts/evaluation/v1/fixtures/evaluation-comparison-response.json?raw";
import nonComparableFixture from "../../../../contracts/evaluation/v1/fixtures/evaluation-comparison-non-comparable-response.json?raw";
import caseFixture from "../../../../contracts/evaluation/v1/fixtures/evaluation-run-case-response.json?raw";
import summaryFixture from "../../../../contracts/evaluation/v1/fixtures/evaluation-run-summary-response.json?raw";

import {
  parseEvaluationComparison,
  parseEvaluationRunCase,
  parseEvaluationRunSummary,
} from "./evaluationRun";

describe("evaluation run contracts", () => {
  it("parses immutable run and case evidence identities", () => {
    const summary = parseEvaluationRunSummary(JSON.parse(summaryFixture));
    const result = parseEvaluationRunCase(JSON.parse(caseFixture));

    expect(summary).toMatchObject({
      id: "40000000-0000-4000-8000-000000000031",
      aggregate: { failed: 1, scored: 1 },
    });
    expect(result.expectedEvidence[0]).toMatchObject({ kind: "expected" });
    expect(result.actualEvidence.map((evidence) => evidence.kind)).toEqual([
      "retrieved",
      "cited",
    ]);
  });

  it("does not accept direct comparison data in a non-comparable response", () => {
    const nonComparable = JSON.parse(nonComparableFixture) as Record<
      string,
      unknown
    >;

    expect(parseEvaluationComparison(nonComparable)).toMatchObject({
      comparisonState: "non_comparable",
      totals: null,
      metrics: [],
    });
    expect(() =>
      parseEvaluationComparison({
        ...nonComparable,
        totals: {},
      }),
    ).toThrow("must not include totals");
  });

  it("parses paired comparison arithmetic", () => {
    expect(
      parseEvaluationComparison(JSON.parse(comparisonFixture)),
    ).toMatchObject({
      comparisonState: "comparable",
      totals: { pairedCases: 2 },
      metrics: [{ name: "citation_validity", delta: -1 }],
    });
  });

  it("rejects evidence outside the immutable run corpus or snapshot", () => {
    const response = caseResponse();
    evidenceAt(response, "expectedEvidence", 0).corpusId =
      "10000000-0000-4000-8000-000000000099";

    expect(() => parseEvaluationRunCase(response)).toThrow(
      "case run corpus and snapshot",
    );

    const snapshotResponse = caseResponse();
    evidenceAt(snapshotResponse, "actualEvidence", 0).snapshotId =
      "20000000-0000-4000-8000-000000000099";
    expect(() => parseEvaluationRunCase(snapshotResponse)).toThrow(
      "case run corpus and snapshot",
    );
  });

  it("rejects malformed evidence markers and offset ranges", () => {
    const markerResponse = caseResponse();
    evidenceAt(markerResponse, "actualEvidence", 0).markerPosition = 1;
    expect(() => parseEvaluationRunCase(markerResponse)).toThrow(
      "Only cited evaluation evidence",
    );

    const missingMarkerResponse = caseResponse();
    delete evidenceAt(missingMarkerResponse, "actualEvidence", 1)
      .markerPosition;
    expect(() => parseEvaluationRunCase(missingMarkerResponse)).toThrow(
      "requires a marker position",
    );

    const offsetResponse = caseResponse();
    evidenceAt(offsetResponse, "actualEvidence", 0).endOffset = 0;
    expect(() => parseEvaluationRunCase(offsetResponse)).toThrow(
      "increasing source offset range",
    );
  });

  it("rejects unsafe failure codes and unknown metric names", () => {
    const failureResponse = caseResponse();
    failureResponse.state = "failed";
    failureResponse.failureCode = "provider diagnostic secret";
    expect(() => parseEvaluationRunCase(failureResponse)).toThrow(
      "failure code is unsupported",
    );

    const misplacedFailureResponse = caseResponse();
    misplacedFailureResponse.failureCode = "provider_unavailable";
    expect(() => parseEvaluationRunCase(misplacedFailureResponse)).toThrow(
      "must match failed or cancelled states",
    );

    const metricResponse = caseResponse();
    metricAt(metricResponse, "metrics", 0).name = "unreviewed_metric";
    expect(() => parseEvaluationRunCase(metricResponse)).toThrow(
      "metric name is unsupported",
    );

    const comparisonResponse = JSON.parse(comparisonFixture) as Record<
      string,
      unknown
    >;
    metricAt(comparisonResponse, "metrics", 0).name = "unreviewed_metric";
    expect(() => parseEvaluationComparison(comparisonResponse)).toThrow(
      "comparison metric name is unsupported",
    );
  });
});

function caseResponse(): Record<string, unknown> {
  return JSON.parse(caseFixture) as Record<string, unknown>;
}

function evidenceAt(
  response: Record<string, unknown>,
  field: "expectedEvidence" | "actualEvidence",
  index: number,
): Record<string, unknown> {
  return recordAt(response, field, index);
}

function metricAt(
  response: Record<string, unknown>,
  field: "metrics",
  index: number,
): Record<string, unknown> {
  return recordAt(response, field, index);
}

function recordAt(
  response: Record<string, unknown>,
  field: string,
  index: number,
): Record<string, unknown> {
  const values = response[field];
  if (!Array.isArray(values)) throw new Error(`${field} must be an array.`);
  const value: unknown = values.at(index);
  if (value === undefined || value === null || typeof value !== "object") {
    throw new Error(`${field} entry must be an object.`);
  }
  return value as Record<string, unknown>;
}
