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
        new AbortController().signal,
        () => undefined,
      ),
    ).rejects.toMatchObject({ status: 502 });
  });
});
