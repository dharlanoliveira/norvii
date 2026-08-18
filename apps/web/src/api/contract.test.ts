import { describe, expect, it } from "vitest";

import {
  parseCorpusList,
  parseCorpusResponse,
  parseDocumentResponse,
  parseErrorEnvelope,
  parseSourceList,
} from "./contract";

describe("corpus ingestion HTTP contract", () => {
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
    acquiredByteCount: 2048,
    normalizedCharacterCount: 1024,
    unitCount: 1,
    durationMilliseconds: 1000,
  };
}
