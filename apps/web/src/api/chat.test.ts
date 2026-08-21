import { describe, expect, it } from "vitest";

import { createHttpChatProvider } from "./chat";

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
});
