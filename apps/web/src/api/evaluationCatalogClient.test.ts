import { describe, expect, it, vi } from "vitest";

import catalogFixture from "../../../../contracts/evaluation/v1/fixtures/dataset-catalog-response.json?raw";
import detailFixture from "../../../../contracts/evaluation/v1/fixtures/dataset-detail-response.json?raw";
import incompatibleFixture from "../../../../contracts/evaluation/v1/fixtures/dataset-preflight-error-snapshot-incompatible.json?raw";
import preflightFixture from "../../../../contracts/evaluation/v1/fixtures/dataset-preflight-success.json?raw";

import { createEvaluationCatalogClient } from "./evaluationCatalogClient";

const datasetRevisionId = "30000000-0000-4000-8000-000000000021";
const corpusId = "10000000-0000-4000-8000-000000000021";
const snapshotId = "20000000-0000-4000-8000-000000000021";

describe("evaluation dataset catalog HTTP client", () => {
  it("uses the authenticated same-origin boundary for immutable dataset inspection", async () => {
    const fetchResponse = vi.fn<typeof fetch>().mockImplementation((input) => {
      const url = requestUrl(input);
      if (url.endsWith("/evaluation-datasets")) {
        return Promise.resolve(jsonResponse(JSON.parse(catalogFixture)));
      }
      if (url.includes("/preflight?")) {
        return Promise.resolve(jsonResponse(JSON.parse(preflightFixture)));
      }
      return Promise.resolve(jsonResponse(JSON.parse(detailFixture)));
    });
    const client = createEvaluationCatalogClient({
      baseUrl: "https://api.example.test/api/v1",
      fetch: fetchResponse,
    });
    const signal = new AbortController().signal;

    await expect(client.listDatasets(signal)).resolves.toHaveLength(1);
    await expect(
      client.getDataset(datasetRevisionId, signal),
    ).resolves.toMatchObject({
      revision: { id: datasetRevisionId },
    });
    await expect(
      client.preflightDataset(
        { datasetRevisionId, corpusId, snapshotId },
        signal,
      ),
    ).resolves.toMatchObject({
      datasetRevisionId,
      corpusId,
      snapshotId,
      compatible: true,
    });

    const requests = fetchResponse.mock.calls.map(([input]) =>
      requestUrl(input),
    );
    expect(requests).toContain(
      "https://api.example.test/api/v1/evaluation-datasets",
    );
    expect(requests).toContain(
      `https://api.example.test/api/v1/evaluation-datasets/${datasetRevisionId}`,
    );
    expect(requests).toContain(
      `https://api.example.test/api/v1/evaluation-datasets/${datasetRevisionId}/preflight?corpusId=${corpusId}&snapshotId=${snapshotId}`,
    );
    expect(fetchResponse).toHaveBeenCalledWith(
      "https://api.example.test/api/v1/evaluation-datasets",
      expect.objectContaining({
        method: "GET",
        signal,
        credentials: "include",
      }),
    );
    expect(fetchResponse.mock.calls[0]?.[1]).not.toHaveProperty("headers");
  });

  it("fails before a request when a requested identity is malformed", async () => {
    const fetchResponse = vi.fn<typeof fetch>();
    const client = createEvaluationCatalogClient({ fetch: fetchResponse });

    await expect(
      client.getDataset("not-a-uuid", new AbortController().signal),
    ).rejects.toThrow("UUID");
    expect(fetchResponse).not.toHaveBeenCalled();
  });

  it("does not accept a response for a different immutable selection", async () => {
    const response = JSON.parse(preflightFixture) as Record<string, unknown>;
    const fetchResponse = vi.fn<typeof fetch>().mockResolvedValue(
      jsonResponse({
        ...response,
        snapshotId: "20000000-0000-4000-8000-000000000022",
      }),
    );
    const client = createEvaluationCatalogClient({ fetch: fetchResponse });

    await expect(
      client.preflightDataset(
        { datasetRevisionId, corpusId, snapshotId },
        new AbortController().signal,
      ),
    ).rejects.toThrow("do not match");
  });

  it("raises safe incompatibility metadata from an unsuccessful preflight", async () => {
    const fetchResponse = vi
      .fn<typeof fetch>()
      .mockResolvedValue(jsonResponse(JSON.parse(incompatibleFixture), 422));
    const client = createEvaluationCatalogClient({ fetch: fetchResponse });

    await expect(
      client.preflightDataset(
        { datasetRevisionId, corpusId, snapshotId },
        new AbortController().signal,
      ),
    ).rejects.toEqual(
      expect.objectContaining({
        name: "EvaluationCatalogRequestError",
        code: "snapshot_incompatible",
        missingRequirements: [
          expect.objectContaining({ sourceAlias: "official-source" }),
        ],
      }),
    );
  });
});

function requestUrl(input: RequestInfo | URL): string {
  if (typeof input === "string") return input;
  return input instanceof URL ? input.href : input.url;
}

function jsonResponse(body: unknown, status = 200): Response {
  return new Response(JSON.stringify(body), {
    status,
    headers: { "Content-Type": "application/json" },
  });
}
