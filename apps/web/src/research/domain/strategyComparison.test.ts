import { describe, expect, it } from "vitest";

import {
  applyStrategyComparisonEvent,
  beginStrategyComparison,
  finishStrategyComparison,
  newStrategyComparisonResult,
} from "./strategyComparison";

describe("strategy comparison", () => {
  it("keeps a failed Hybrid result distinct from completed strategies", () => {
    const initial = beginStrategyComparison();
    const hybrid = initial.results.find(
      (result) => result.strategy === "hybrid",
    );
    if (hybrid === undefined) {
      throw new Error("Hybrid comparison result is required.");
    }

    const updated = applyStrategyComparisonEvent(
      hybrid,
      {
        type: "error",
        requestId: "request-1",
        code: "generation_failed",
        message: "The request could not be completed.",
        telemetry: {
          outcome: "failed",
          evidenceCount: 0,
          durationMilliseconds: 1,
        },
      },
      "No answer.",
    );

    expect(updated.status).toBe("failed");
    expect(finishStrategyComparison(initial).status).toBe("completed");
  });

  it("reconciles evidence, streamed text, abstention, cancellation, and provider failures", () => {
    const result = newStrategyComparisonResult("vector");
    const reference = {
      id: "reference-1",
      corpusId: "corpus-1",
      sourceId: "source-1",
      documentId: "document-1",
      unitLocator: "Article 1",
      startOffset: 0,
      endOffset: 10,
      excerpt: "Scope",
      rank: 1,
    };
    const telemetry = {
      outcome: "completed",
      evidenceCount: 1,
      durationMilliseconds: 1,
    };

    const withEvidence = applyStrategyComparisonEvent(
      result,
      { type: "evidence", requestId: "request-1", references: [reference] },
      "No answer.",
    );
    const withDelta = applyStrategyComparisonEvent(
      withEvidence,
      { type: "delta", requestId: "request-1", text: "Partial answer." },
      "No answer.",
    );
    const abstained = applyStrategyComparisonEvent(
      withDelta,
      {
        type: "abstained",
        requestId: "request-1",
        reason: "Not enough evidence.",
        telemetry,
      },
      "No answer.",
    );
    const cancelled = applyStrategyComparisonEvent(
      result,
      { type: "cancelled", requestId: "request-1", telemetry },
      "No answer.",
    );
    const failed = applyStrategyComparisonEvent(
      result,
      {
        type: "error",
        requestId: "request-1",
        code: "provider_failed",
        message: "Provider failed.",
        telemetry,
      },
      "No answer.",
    );

    expect(withEvidence.references).toEqual([reference]);
    expect(withDelta.answer).toBe("Partial answer.");
    expect(abstained).toMatchObject({
      status: "abstained",
      answer: "No answer.",
    });
    expect(cancelled.status).toBe("failed");
    expect(failed).toMatchObject({
      status: "failed",
      answer: "Provider failed.",
    });
  });
});
