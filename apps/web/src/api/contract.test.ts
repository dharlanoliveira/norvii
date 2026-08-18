import { describe, expect, it } from "vitest";

import { parseCorpusList, parseErrorEnvelope } from "./contract";

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
});
