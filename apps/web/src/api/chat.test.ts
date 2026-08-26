import { describe, expect, it, vi } from "vitest";

import { createHttpChatProvider, parseChatEvent } from "./chat";

describe("HTTP grounded chat provider", () => {
  it("parses ordered SSE events and sends the active interface language", async () => {
    const calls: RequestInit[] = [];
    const fetchResponse = (
      _input: RequestInfo | URL,
      init?: RequestInit,
    ): Promise<Response> => {
      calls.push(init ?? {});
      const body =
        'event: started\ndata: {"type":"started","requestId":"00000000-0000-0000-0000-000000000001","corpusId":"00000000-0000-0000-0000-000000000002"}\n\n' +
        'event: delta\ndata: {"type":"delta","requestId":"00000000-0000-0000-0000-000000000001","text":"Answer"}\n\n' +
        'event: completed\ndata: {"type":"completed","requestId":"00000000-0000-0000-0000-000000000001","answer":"Answer [1].","references":[],"telemetry":{"outcome":"completed","evidenceCount":0,"durationMilliseconds":1}}\n\n';
      return Promise.resolve(
        new Response(body, {
          status: 200,
          headers: { "Content-Type": "text/event-stream" },
        }),
      );
    };
    const provider = createHttpChatProvider({ fetch: fetchResponse });
    const events: string[] = [];

    await provider.streamQuestion(
      "00000000-0000-0000-0000-000000000002",
      "What applies?",
      "pt",
      "vector",
      new AbortController().signal,
      (event) => events.push(event.type),
    );

    expect(events).toEqual(["started", "delta", "completed"]);
    expect(calls[0]?.method).toBe("POST");
    expect(calls[0]?.headers).toEqual({ "Content-Type": "application/json" });
    const requestBody = calls[0]?.body;
    expect(requestBody).toBeTypeOf("string");
    expect(JSON.parse(requestBody as string)).toEqual({
      question: "What applies?",
      interfaceLanguage: "pt",
      strategy: "vector",
    });
  });

  it("parses each supported event variant", () => {
    const telemetry = {
      outcome: "completed",
      evidenceCount: 1,
      durationMilliseconds: 12,
    };
    const reference = {
      id: "reference-1",
      corpusId: "corpus-1",
      sourceId: "source-1",
      documentId: "document-1",
      unitLocator: "Article 1",
      startOffset: 0,
      endOffset: 10,
      excerpt: "An excerpt",
      rank: 1,
    };

    expect(
      parseChatEvent({
        type: "started",
        requestId: "request-1",
        corpusId: "corpus-1",
      }),
    ).toMatchObject({ type: "started" });
    expect(
      parseChatEvent({
        type: "evidence",
        requestId: "request-1",
        references: [reference],
      }),
    ).toMatchObject({ type: "evidence", references: [reference] });
    expect(
      parseChatEvent({ type: "delta", requestId: "request-1", text: "part" }),
    ).toEqual({ type: "delta", requestId: "request-1", text: "part" });
    expect(
      parseChatEvent({
        type: "completed",
        requestId: "request-1",
        answer: "answer",
        references: [reference],
        telemetry,
      }),
    ).toMatchObject({ type: "completed", answer: "answer" });
    expect(
      parseChatEvent({
        type: "abstained",
        requestId: "request-1",
        reason: "No evidence",
        telemetry,
      }),
    ).toMatchObject({ type: "abstained", reason: "No evidence" });
    expect(
      parseChatEvent({ type: "cancelled", requestId: "request-1", telemetry }),
    ).toMatchObject({ type: "cancelled" });
    expect(
      parseChatEvent({
        type: "error",
        requestId: "request-1",
        code: "provider_error",
        message: "Provider failed",
        telemetry,
      }),
    ).toMatchObject({ type: "error", message: "Provider failed" });
  });

  it("rejects malformed events and invalid stream responses", async () => {
    expect(() => parseChatEvent(null)).toThrow("must have a type");
    expect(() => parseChatEvent({ type: "delta" })).toThrow(
      "must have a request ID",
    );
    expect(() =>
      parseChatEvent({ type: "unknown", requestId: "request-1" }),
    ).toThrow("unsupported");
    expect(() =>
      parseChatEvent({ type: "delta", requestId: "request-1", text: 1 }),
    ).toThrow("chat delta must be a string");
    expect(() =>
      parseChatEvent({
        type: "completed",
        requestId: "request-1",
        answer: "answer",
        references: "invalid",
        telemetry: {},
      }),
    ).toThrow("references must be an array");

    const failingFetch = vi
      .fn<typeof fetch>()
      .mockResolvedValue(new Response(null, { status: 503 }));
    await expect(
      createHttpChatProvider({ fetch: failingFetch }).streamQuestion(
        "corpus-1",
        "question",
        "en",
        "vector",
        new AbortController().signal,
        () => undefined,
      ),
    ).rejects.toMatchObject({ status: 503 });

    const emptyStreamFetch = vi
      .fn<typeof fetch>()
      .mockResolvedValue(new Response(null, { status: 200 }));
    await expect(
      createHttpChatProvider({ fetch: emptyStreamFetch }).streamQuestion(
        "corpus-1",
        "question",
        "en",
        "vector",
        new AbortController().signal,
        () => undefined,
      ),
    ).rejects.toMatchObject({ status: 502 });
  });

  it("preserves nullable inspection measurements and rejects negative values", () => {
    const event = parseChatEvent({
      type: "completed",
      requestId: "request-1",
      answer: "Answer [1].",
      references: [],
      telemetry: {
        outcome: "completed",
        evidenceCount: 0,
        durationMilliseconds: 1,
      },
      inspection: {
        outcome: "completed",
        retrieval: {
          strategy: "vector",
          topK: 8,
          returnedCount: 0,
          embeddingModel: null,
        },
        stages: [
          {
            name: "planning",
            state: "skipped",
            evidenceCount: 0,
            durationMilliseconds: 4,
            reasonCode: "not_relevant",
            inputTokens: 10,
            outputTokens: 2,
          },
        ],
        measurements: {
          retrievalMilliseconds: null,
          generationMilliseconds: 1,
          totalMilliseconds: 1,
          inputTokens: null,
          outputTokens: null,
        },
        evidence: [],
      },
    });
    expect(
      event.type === "completed" && event.inspection?.measurements.inputTokens,
    ).toBeNull();
    expect(
      event.type === "completed" && event.inspection?.stages?.[0],
    ).toMatchObject({ name: "planning", state: "skipped" });
    expect(() =>
      parseChatEvent({
        type: "completed",
        requestId: "request-1",
        answer: "Answer [1].",
        references: [],
        telemetry: {
          outcome: "completed",
          evidenceCount: 0,
          durationMilliseconds: 1,
        },
        inspection: {
          outcome: "completed",
          measurements: {
            retrievalMilliseconds: -1,
            generationMilliseconds: null,
            totalMilliseconds: null,
            inputTokens: null,
            outputTokens: null,
          },
        },
      }),
    ).toThrow("retrieval duration must not be negative");
  });

  it("validates graph-backed evidence and normative assertion provenance", () => {
    const event = parseChatEvent({
      type: "completed",
      requestId: "request-1",
      answer: "The authority must issue guidance. [1]",
      references: [
        {
          id: "reference-1",
          corpusId: "corpus-1",
          sourceId: "source-1",
          documentId: "document-1",
          unitLocator: "article-55",
          startOffset: 0,
          endOffset: 10,
          excerpt: "Authority duties.",
          rank: 1,
          contribution: "vector_and_graph",
        },
      ],
      telemetry: {
        outcome: "completed",
        evidenceCount: 1,
        durationMilliseconds: 12,
      },
      inspection: {
        outcome: "completed",
        measurements: {
          retrievalMilliseconds: 5,
          generationMilliseconds: 7,
          totalMilliseconds: 12,
          inputTokens: 4,
          outputTokens: 6,
        },
        assertionPath: [
          {
            assertionId: "assertion-1",
            predicate: "imposes_duty_on",
            subjectLabel: "data protection authority",
            objectLabel: "issue guidance",
            establishingLocator: "article-55",
            evidenceLocator: "article-55",
            hierarchyContext: ["chapter-9", "article-55"],
            qualifier: null,
          },
        ],
        scopeLocator: "chapter-9",
      },
    });

    if (event.type !== "completed")
      throw new Error("Expected completion event.");
    expect(event.references[0]?.contribution).toBe("vector_and_graph");
    expect(event.inspection?.assertionPath).toEqual([
      {
        assertionId: "assertion-1",
        predicate: "imposes_duty_on",
        subjectLabel: "data protection authority",
        objectLabel: "issue guidance",
        establishingLocator: "article-55",
        evidenceLocator: "article-55",
        hierarchyContext: ["chapter-9", "article-55"],
        qualifier: null,
      },
    ]);
    expect(event.inspection?.scopeLocator).toBe("chapter-9");
  });

  it("rejects an assertion predicate outside the normative vocabulary", () => {
    expect(() =>
      parseChatEvent({
        type: "completed",
        requestId: "request-1",
        answer: "Answer [1].",
        references: [],
        telemetry: {
          outcome: "completed",
          evidenceCount: 0,
          durationMilliseconds: 1,
        },
        inspection: {
          outcome: "completed",
          measurements: {
            retrievalMilliseconds: null,
            generationMilliseconds: null,
            totalMilliseconds: null,
            inputTokens: null,
            outputTokens: null,
          },
          assertionPath: [
            {
              assertionId: "assertion-1",
              predicate: "requires",
              subjectLabel: "Authority",
              objectLabel: "Controller",
              establishingLocator: "article-1",
              evidenceLocator: "article-1",
              hierarchyContext: ["article-1"],
              qualifier: null,
            },
          ],
        },
      }),
    ).toThrow("Normative assertion predicate is invalid.");
  });
});
