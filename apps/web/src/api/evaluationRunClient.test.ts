import { describe, expect, it, vi } from "vitest";

import comparisonFixture from "../../../../contracts/evaluation/v1/fixtures/evaluation-comparison-response.json?raw";
import caseFixture from "../../../../contracts/evaluation/v1/fixtures/evaluation-run-case-response.json?raw";
import summaryFixture from "../../../../contracts/evaluation/v1/fixtures/evaluation-run-summary-response.json?raw";

import { createEvaluationRunClient } from "./evaluationRunClient";

const runId = "40000000-0000-4000-8000-000000000031";
const otherRunId = "40000000-0000-4000-8000-000000000032";
const caseId = "50000000-0000-4000-8000-000000000031";

describe("evaluation run HTTP client", () => {
  it("uses the authenticated same-origin session for immutable inspection requests", async () => {
    const fetchResponse = vi.fn<typeof fetch>().mockImplementation((input) => {
      const url = requestUrl(input);
      if (url.includes("/compare?")) {
        return Promise.resolve(jsonResponse(JSON.parse(comparisonFixture)));
      }
      if (url.includes("/cases/")) {
        return Promise.resolve(jsonResponse(JSON.parse(caseFixture)));
      }
      return Promise.resolve(jsonResponse(JSON.parse(summaryFixture)));
    });
    const client = createEvaluationRunClient({
      baseUrl: "https://api.example.test/api/v1",
      fetch: fetchResponse,
    });
    const signal = new AbortController().signal;

    await expect(client.getRun(runId, signal)).resolves.toMatchObject({
      id: runId,
    });
    await expect(client.getCase(runId, caseId, signal)).resolves.toMatchObject({
      id: caseId,
      runId,
    });
    await expect(
      client.compareRuns(runId, otherRunId, signal),
    ).resolves.toMatchObject({ comparisonState: "comparable" });

    expect(
      fetchResponse.mock.calls.map(([input]) => requestUrl(input)),
    ).toEqual([
      `https://api.example.test/api/v1/evaluations/${runId}`,
      `https://api.example.test/api/v1/evaluations/${runId}/cases/${caseId}`,
      `https://api.example.test/api/v1/evaluations/compare?left=${runId}&right=${otherRunId}`,
    ]);
    expect(fetchResponse).toHaveBeenCalledWith(
      `https://api.example.test/api/v1/evaluations/${runId}`,
      expect.objectContaining({
        method: "GET",
        signal,
        credentials: "include",
      }),
    );
    expect(fetchResponse.mock.calls[0]?.[1]).not.toHaveProperty("headers");
  });

  it("rejects malformed requested identities before a request", async () => {
    const fetchResponse = vi.fn<typeof fetch>();
    const client = createEvaluationRunClient({ fetch: fetchResponse });

    await expect(
      client.getRun("invalid", new AbortController().signal),
    ).rejects.toThrow("UUID");
    expect(fetchResponse).not.toHaveBeenCalled();
  });
});

function requestUrl(input: RequestInfo | URL): string {
  if (typeof input === "string") return input;
  return input instanceof URL ? input.href : input.url;
}

function jsonResponse(body: unknown): Response {
  return new Response(JSON.stringify(body), {
    headers: { "Content-Type": "application/json" },
  });
}
