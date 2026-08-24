import { describe, expect, it } from "vitest";

import {
  applyStrategyComparisonEvent,
  beginStrategyComparison,
  finishStrategyComparison,
} from "./strategyComparison";

describe("strategy comparison", () => {
  it("keeps a graph-unavailable result distinct from completed strategies", () => {
    const initial = beginStrategyComparison();
    const graph = initial.results.find((result) => result.strategy === "graph");
    if (graph === undefined) {
      throw new Error("Graph comparison result is required.");
    }

    const updated = applyStrategyComparisonEvent(
      graph,
      {
        type: "error",
        requestId: "request-1",
        code: "graph_unavailable",
        message: "The graph release is unavailable.",
        telemetry: {
          outcome: "failed",
          evidenceCount: 0,
          durationMilliseconds: 1,
        },
      },
      "No answer.",
    );

    expect(updated.status).toBe("unavailable");
    expect(finishStrategyComparison(initial).status).toBe("completed");
  });
});
