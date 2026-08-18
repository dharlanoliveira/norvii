import { describe, expect, it, vi } from "vitest";

import { createHttpResearchProvider } from "./researchProvider";

const corpusId = "10000000-0000-4000-8000-000000000002";
const sourceId = "20000000-0000-4000-8000-000000000002";

describe("HTTP research provider", () => {
  it("maps every corpus and source operation to the versioned API", async () => {
    const fetchResponse = vi
      .fn<typeof fetch>()
      .mockImplementation((input, init) => {
        const url = requestUrl(input);
        if (url.endsWith("/document"))
          return Promise.resolve(jsonResponse(documentResponse()));
        if (url.endsWith("/sources") && init?.method === "GET") {
          return Promise.resolve(jsonResponse([sourceResponse()]));
        }
        if (url.endsWith("/sources/pdf"))
          return Promise.resolve(jsonResponse(sourceResponse("pdf")));
        if (url.endsWith("/sources/url"))
          return Promise.resolve(jsonResponse(sourceResponse()));
        if (url.includes("/sources/"))
          return Promise.resolve(jsonResponse(sourceResponse()));
        if (url.endsWith("/corpora") || url.includes("includeDisabled")) {
          return Promise.resolve(
            init?.method === "POST"
              ? jsonResponse(corpusResponse())
              : jsonResponse([corpusResponse()]),
          );
        }
        return Promise.resolve(jsonResponse(corpusResponse()));
      });
    const provider = createHttpResearchProvider({
      baseUrl: "https://api.example.test/v1",
      fetch: fetchResponse,
    });
    const signal = new AbortController().signal;

    await expect(provider.listCorpora(signal)).resolves.toHaveLength(1);
    await expect(provider.listCorpora(signal, true)).resolves.toHaveLength(1);
    await expect(provider.getCorpus(corpusId, signal)).resolves.toMatchObject({
      id: corpusId,
    });
    await expect(provider.listSources(corpusId, signal)).resolves.toHaveLength(
      1,
    );
    await expect(
      provider.getDocument(corpusId, sourceId, signal),
    ).resolves.toMatchObject({ id: documentResponse().id });
    await expect(
      provider.createCorpus(corpusDraft(), signal),
    ).resolves.toMatchObject({ id: corpusId });
    await expect(
      provider.updateCorpus(corpusId, { ...corpusDraft(), version: 1 }, signal),
    ).resolves.toMatchObject({ version: 1 });
    await expect(
      provider.createUrlSource(
        corpusId,
        { title: "Official law", url: "https://example.org/law" },
        signal,
      ),
    ).resolves.toMatchObject({ kind: "url" });
    await expect(
      provider.createPdfSource(
        corpusId,
        "Official PDF",
        new File(["%PDF"], "law.pdf", { type: "application/pdf" }),
        signal,
      ),
    ).resolves.toMatchObject({ kind: "pdf" });
    await expect(
      provider.retrySource(corpusId, sourceId, 2, signal),
    ).resolves.toMatchObject({ id: sourceId });
    await expect(
      provider.reprocessSource(corpusId, sourceId, 2, signal),
    ).resolves.toMatchObject({ id: sourceId });
    await expect(
      provider.disableCorpus(corpusId, 1, signal),
    ).resolves.toMatchObject({
      id: corpusId,
    });
    await expect(
      provider.enableCorpus(corpusId, 1, signal),
    ).resolves.toMatchObject({
      id: corpusId,
    });

    const requests = fetchResponse.mock.calls.map(([input]) =>
      requestUrl(input),
    );
    expect(requests).toContain(
      "https://api.example.test/v1/corpora?includeDisabled=true",
    );
    expect(requests).toContain(
      `https://api.example.test/v1/corpora/${corpusId}/sources/${sourceId}/reprocess`,
    );
  });

  it("raises the public API error with safe metadata", async () => {
    const fetchResponse = vi.fn<typeof fetch>().mockResolvedValue(
      jsonResponse(
        {
          error: {
            code: "stale_state",
            message: "Reload and retry.",
            fields: { version: "stale" },
            requestId: "40000000-0000-4000-8000-000000000001",
          },
        },
        409,
      ),
    );
    const provider = createHttpResearchProvider({ fetch: fetchResponse });

    const request = provider.disableCorpus(
      corpusId,
      1,
      new AbortController().signal,
    );

    await expect(request).rejects.toEqual(
      expect.objectContaining({
        name: "ResearchRequestError",
        code: "stale_state",
        requestId: "40000000-0000-4000-8000-000000000001",
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

function corpusDraft() {
  return {
    name: "EU Privacy Law",
    description: "Official materials.",
    language: "en" as const,
    jurisdiction: "European Union",
  };
}

function corpusResponse() {
  return {
    id: corpusId,
    ...corpusDraft(),
    status: "enabled",
    sourceCount: 1,
    version: 1,
    createdAt: "2026-08-17T12:00:00Z",
    updatedAt: "2026-08-17T12:00:00Z",
  };
}

function sourceResponse(kind: "url" | "pdf" = "url") {
  return {
    id: sourceId,
    corpusId,
    title: "Official law",
    kind,
    processingStatus: "ready",
    failureCategory: null,
    latestReadyDocumentId: "50000000-0000-4000-8000-000000000001",
    version: 2,
    createdAt: "2026-08-17T12:00:00Z",
    updatedAt: "2026-08-17T12:01:00Z",
    origin: {
      kind,
      submittedUrl: kind === "url" ? "https://example.org/law" : null,
      normalizedUrl: kind === "url" ? "https://example.org/law" : null,
      originalFilename: kind === "pdf" ? "law.pdf" : null,
      mediaType: kind === "pdf" ? "application/pdf" : "text/html",
      byteSize: 2048,
      sha256: "b".repeat(64),
      finalUrl: kind === "url" ? "https://example.org/law" : null,
      capturedAt: "2026-08-17T12:00:30Z",
      extractedContentSha256: "a".repeat(64),
    },
    latestAttempt: null,
    attempts: [],
  };
}

function documentResponse() {
  return {
    id: "50000000-0000-4000-8000-000000000001",
    sourceRevisionId: "40000000-0000-4000-8000-000000000001",
    pipelineVersion: "corpus-ingestion-v1",
    text: "Article 1\nPurpose.",
    textSha256: "a".repeat(64),
    createdAt: "2026-08-17T12:01:00Z",
    units: [],
    provenance: {
      contentSha256: "b".repeat(64),
      capturedAt: "2026-08-17T12:00:30Z",
      mediaType: "text/html",
      byteSize: 2048,
      finalUrl: "https://example.org/law",
      extractedContentSha256: "a".repeat(64),
    },
  };
}
