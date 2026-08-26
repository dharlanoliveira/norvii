import { describe, expect, it } from "vitest";

import emptyNoActiveSnapshotFixture from "../../../../contracts/corpus-opening-suggestions/v1/fixtures/suggestions-response-empty-no-active-snapshot.json?raw";
import emptyStaleProjectionFixture from "../../../../contracts/corpus-opening-suggestions/v1/fixtures/suggestions-response-empty-stale-projection.json?raw";
import englishSuggestionsFixture from "../../../../contracts/corpus-opening-suggestions/v1/fixtures/suggestions-response-en.json?raw";
import evaluationLeakageFixture from "../../../../contracts/corpus-opening-suggestions/v1/fixtures/suggestions-response-invalid-evaluation-leakage.json?raw";
import invalidRankOrderFixture from "../../../../contracts/corpus-opening-suggestions/v1/fixtures/suggestions-response-invalid-rank-order.json?raw";
import portugueseSuggestionsFixture from "../../../../contracts/corpus-opening-suggestions/v1/fixtures/suggestions-response-pt.json?raw";

import {
  parseCorpusList,
  parseCorpusOpeningSuggestionResponse,
  parseCorpusResponse,
  parseDocumentResponse,
  parseErrorEnvelope,
  parseGraphReleaseResponse,
  parseSnapshotPublicationResponse,
  parseSourceList,
} from "./contract";

describe("corpus ingestion HTTP contract", () => {
  it("validates rank-ordered English and Portuguese opening suggestions", () => {
    const english = parseCorpusOpeningSuggestionResponse(
      openingSuggestionResponse("en"),
    );
    const portuguese = parseCorpusOpeningSuggestionResponse(
      openingSuggestionResponse("pt"),
    );

    expect(english.interfaceLanguage).toBe("en");
    expect(portuguese.interfaceLanguage).toBe("pt");
    expect(english.suggestions.map((suggestion) => suggestion.rank)).toEqual([
      1, 2, 3, 4, 5,
    ]);
  });

  it("validates an empty opening-suggestion response without a snapshot", () => {
    const response = parseCorpusOpeningSuggestionResponse(
      fixturePayload(emptyNoActiveSnapshotFixture),
    );
    const staleProjection = parseCorpusOpeningSuggestionResponse(
      fixturePayload(emptyStaleProjectionFixture),
    );

    expect(response.suggestions).toEqual([]);
    expect(response.activeSnapshotId).toBeNull();
    expect(staleProjection.suggestions).toEqual([]);
    expect(staleProjection.activeSnapshotId).not.toBeNull();
  });

  it("rejects malformed and evaluation-leaking opening suggestions", () => {
    expect(() =>
      parseCorpusOpeningSuggestionResponse({
        ...openingSuggestionResponse("en"),
        unexpected: true,
      }),
    ).toThrow("unsupported field");
    expect(() =>
      parseCorpusOpeningSuggestionResponse(
        fixturePayload(evaluationLeakageFixture),
      ),
    ).toThrow("unsupported field");
    expect(() =>
      parseCorpusOpeningSuggestionResponse(
        fixturePayload(invalidRankOrderFixture),
      ),
    ).toThrow("ascending order");
    expect(() =>
      parseCorpusOpeningSuggestionResponse({
        ...openingSuggestionResponse("en"),
        suggestions: Array.from({ length: 6 }, (_, index) => ({
          caseId: `synthetic-${String(index + 1)}`,
          rank: index + 1,
          question: "Synthetic question?",
        })),
      }),
    ).toThrow("more than five");
  });

  it("rejects invalid opening-suggestion values at the response boundary", () => {
    const invalidResponses = [
      {
        ...openingSuggestionResponse("en"),
        corpusId: "not-a-uuid",
      },
      {
        ...openingSuggestionResponse("en"),
        activeSnapshotManifestSha256: "not-a-hash",
      },
      {
        ...openingSuggestionResponse("en"),
        interfaceLanguage: "fr",
      },
      {
        ...openingSuggestionResponse("en"),
        suggestions: [
          { caseId: "synthetic-question", rank: 0, question: "Question?" },
        ],
      },
      {
        ...openingSuggestionResponse("en"),
        suggestions: [
          { caseId: "synthetic-question", rank: 1, question: "   " },
        ],
      },
      {
        ...openingSuggestionResponse("en"),
        activeSnapshotId: null,
      },
      {
        ...openingSuggestionResponse("en"),
        activeSnapshotId: undefined,
      },
      {
        ...openingSuggestionResponse("en"),
        activeSnapshotManifestSha256: undefined,
      },
    ];

    for (const response of invalidResponses) {
      expect(() => parseCorpusOpeningSuggestionResponse(response)).toThrow();
    }
    expect(() =>
      parseCorpusOpeningSuggestionResponse({
        corpusId: "10000000-0000-4000-8000-000000000002",
        activeSnapshotId: null,
        activeSnapshotManifestSha256: null,
        interfaceLanguage: "en",
      }),
    ).toThrow("must contain suggestions");
    expect(() =>
      parseCorpusOpeningSuggestionResponse({
        ...openingSuggestionResponse("en"),
        suggestions: [{ caseId: "synthetic-question", rank: 1 }],
      }),
    ).toThrow("must contain question");
  });
  it("validates authoritative corpus list responses", () => {
    const corpora = parseCorpusList([
      {
        id: "10000000-0000-4000-8000-000000000002",
        name: "European Union General Data Protection Regulation",
        description: "Official European Union data-protection regulation.",
        language: "en",
        jurisdiction: "European Union",
        status: "enabled",
        sourceCount: 1,
        version: 1,
        createdAt: "2026-08-17T12:00:00Z",
        updatedAt: "2026-08-17T12:00:00Z",
      },
    ]);

    expect(corpora[0]?.language).toBe("en");
  });

  it("rejects unknown error codes", () => {
    expect(() =>
      parseErrorEnvelope({
        error: {
          code: "database_stack_trace",
          message: "unsafe",
          requestId: "40000000-0000-4000-8000-000000000001",
        },
      }),
    ).toThrow("error code");
  });

  it("validates an immutable snapshot publication response", () => {
    const publication = parseSnapshotPublicationResponse({
      snapshot: {
        id: "70000000-0000-4000-8000-000000000001",
        corpusId: "10000000-0000-4000-8000-000000000002",
        manifestSha256: "a".repeat(64),
        createdBy: "local-maintainer",
        createdAt: "2026-08-24T12:00:00Z",
        members: [
          {
            sourceId: "20000000-0000-4000-8000-000000000002",
            sourceRevisionId: "40000000-0000-4000-8000-000000000001",
            documentId: "50000000-0000-4000-8000-000000000001",
            officialOrigin: "https://example.org/law",
            capturedAt: "2026-08-24T11:00:00Z",
            contentSha256: "b".repeat(64),
          },
        ],
      },
      release: {
        id: "70000000-0000-4000-8000-000000000001",
        manifestSha256: "a".repeat(64),
        createdAt: "2026-08-24T12:00:00Z",
        activatedAt: "2026-08-24T12:00:00Z",
        releaseVersion: 2,
      },
      published: true,
    });

    expect(publication.snapshot.members).toHaveLength(1);
    expect(publication.release.releaseVersion).toBe(2);
  });

  it("validates a snapshot-scoped graph release response", () => {
    const release = parseGraphReleaseResponse({
      id: "80000000-0000-4000-8000-000000000001",
      corpusId: "10000000-0000-4000-8000-000000000002",
      snapshotId: "70000000-0000-4000-8000-000000000001",
      manifestSha256: "a".repeat(64),
      buildVersion: "legal-graph-v1",
      status: "ready",
      entityCount: 8,
      relationshipCount: 4,
      createdAt: "2026-08-24T12:00:00Z",
      completedAt: "2026-08-24T12:01:00Z",
    });

    expect(release.status).toBe("ready");
    expect(release.snapshotId).toBe("70000000-0000-4000-8000-000000000001");
  });

  it("validates populated source, attempt, document, and provenance values", () => {
    const source = parseSourceList([
      {
        id: "20000000-0000-4000-8000-000000000002",
        corpusId: "10000000-0000-4000-8000-000000000002",
        title: "Official PDF",
        kind: "pdf",
        processingStatus: "failed",
        failureCategory: "extraction_failed",
        latestReadyDocumentId: "50000000-0000-4000-8000-000000000001",
        version: 2,
        createdAt: "2026-08-17T12:00:00Z",
        updatedAt: "2026-08-17T12:01:00Z",
        origin: {
          kind: "pdf",
          submittedUrl: null,
          normalizedUrl: null,
          originalFilename: "law.pdf",
          mediaType: "application/pdf",
          byteSize: 2048,
          sha256: "b".repeat(64),
          finalUrl: null,
          capturedAt: "2026-08-17T12:00:30Z",
          extractedContentSha256: "a".repeat(64),
        },
        latestAttempt: processingAttempt(),
        attempts: [processingAttempt()],
      },
    ]);
    const document = parseDocumentResponse({
      id: "50000000-0000-4000-8000-000000000001",
      sourceRevisionId: "40000000-0000-4000-8000-000000000001",
      pipelineVersion: "corpus-ingestion-v1",
      text: "Article 1",
      textSha256: "a".repeat(64),
      createdAt: "2026-08-17T12:01:00Z",
      units: [
        {
          id: "60000000-0000-4000-8000-000000000001",
          parentId: "60000000-0000-4000-8000-000000000000",
          kind: "article",
          ordinal: 1,
          marker: "Article 1",
          label: "Purpose",
          startOffset: 0,
          endOffset: 9,
          startPage: 1,
          endPage: 2,
          locator: "article-1",
          contentSha256: "c".repeat(64),
        },
      ],
      provenance: {
        contentSha256: "b".repeat(64),
        capturedAt: "2026-08-17T12:00:30Z",
        mediaType: "application/pdf",
        byteSize: 2048,
        finalUrl: null,
        extractedContentSha256: "a".repeat(64),
      },
    });

    expect(source[0]?.latestAttempt?.durationMilliseconds).toBe(1000);
    expect(document.units[0]?.startPage).toBe(1);
  });

  it("normalizes document units into source reading order", () => {
    const document = parseDocumentResponse({
      id: "50000000-0000-4000-8000-000000000001",
      sourceRevisionId: "40000000-0000-4000-8000-000000000001",
      pipelineVersion: "corpus-ingestion-v1",
      text: "A complete legal document",
      textSha256: "a".repeat(64),
      createdAt: "2026-08-17T12:01:00Z",
      units: [
        documentUnit(
          "60000000-0000-4000-8000-000000000004",
          "article-2",
          50,
          100,
          1,
        ),
        documentUnit("60000000-0000-4000-8000-000000000002", "title", 0, 20, 0),
        documentUnit(
          "60000000-0000-4000-8000-000000000003",
          "article-1",
          20,
          50,
          0,
        ),
        documentUnit(
          "60000000-0000-4000-8000-000000000001",
          "document",
          0,
          100,
          0,
        ),
      ],
      provenance: {
        contentSha256: "b".repeat(64),
        capturedAt: "2026-08-17T12:00:30Z",
        mediaType: "text/html",
        byteSize: 2048,
        finalUrl: "https://example.org/law",
        extractedContentSha256: "a".repeat(64),
      },
    });

    expect(document.units.map((unit) => unit.locator)).toEqual([
      "document",
      "title",
      "article-1",
      "article-2",
    ]);
  });

  it("uses type errors for malformed response shapes", () => {
    expect(() => parseCorpusList({})).toThrow(TypeError);
    expect(() => parseSourceList({})).toThrow(TypeError);
    expect(() =>
      parseDocumentResponse({
        id: "50000000-0000-4000-8000-000000000001",
        units: {},
      }),
    ).toThrow(TypeError);
    expect(() =>
      parseCorpusResponse({
        id: "10000000-0000-4000-8000-000000000002",
        name: "Privacy",
        description: "Official materials.",
        language: "en",
        jurisdiction: "European Union",
        status: "enabled",
        sourceCount: 1,
        version: 1,
        createdAt: "not-a-date",
        updatedAt: "2026-08-17T12:00:00Z",
      }),
    ).toThrow(TypeError);
  });
});

function processingAttempt() {
  return {
    number: 1,
    pipelineVersion: "corpus-ingestion-v1",
    status: "failed",
    startedAt: "2026-08-17T12:00:00Z",
    finishedAt: "2026-08-17T12:00:01Z",
    failureCategory: "extraction_failed",
    failureDetail: "provider_response_invalid",
    acquiredByteCount: 2048,
    normalizedCharacterCount: 1024,
    unitCount: 1,
    durationMilliseconds: 1000,
  };
}

function documentUnit(
  id: string,
  locator: string,
  startOffset: number,
  endOffset: number,
  ordinal: number,
) {
  return {
    id,
    parentId: null,
    kind: locator,
    ordinal,
    marker: null,
    label: null,
    startOffset,
    endOffset,
    startPage: null,
    endPage: null,
    locator,
    contentSha256: "c".repeat(64),
  };
}

function openingSuggestionResponse(interfaceLanguage: "en" | "pt") {
  const fixture =
    interfaceLanguage === "en"
      ? englishSuggestionsFixture
      : portugueseSuggestionsFixture;
  return parseCorpusOpeningSuggestionResponse(fixturePayload(fixture));
}

function fixturePayload(fixture: string): unknown {
  return JSON.parse(fixture) as unknown;
}
