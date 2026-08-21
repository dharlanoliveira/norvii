import type { CorpusLanguage } from "./contract";

export interface ChatReference {
  readonly id: string;
  readonly corpusId: string;
  readonly sourceId: string;
  readonly documentId: string;
  readonly unitLocator: string;
  readonly startOffset: number;
  readonly endOffset: number;
  readonly excerpt: string;
  readonly rank: number;
}

export type ChatStreamEvent =
  | {
      readonly type: "started";
      readonly requestId: string;
      readonly corpusId: string;
    }
  | {
      readonly type: "evidence";
      readonly requestId: string;
      readonly references: readonly ChatReference[];
    }
  | {
      readonly type: "delta";
      readonly requestId: string;
      readonly text: string;
    }
  | {
      readonly type: "completed";
      readonly requestId: string;
      readonly answer: string;
      readonly references: readonly ChatReference[];
      readonly telemetry: ChatTelemetry;
    }
  | {
      readonly type: "abstained";
      readonly requestId: string;
      readonly reason: string;
      readonly telemetry: ChatTelemetry;
    }
  | {
      readonly type: "cancelled";
      readonly requestId: string;
      readonly telemetry: ChatTelemetry;
    }
  | {
      readonly type: "error";
      readonly requestId: string;
      readonly code: string;
      readonly message: string;
      readonly telemetry: ChatTelemetry;
    };

export interface ChatTelemetry {
  readonly outcome: string;
  readonly evidenceCount: number;
  readonly durationMilliseconds: number;
}

export interface ChatProvider {
  streamQuestion(
    corpusId: string,
    question: string,
    interfaceLanguage: CorpusLanguage,
    signal: AbortSignal,
    onEvent: (event: ChatStreamEvent) => void,
  ): Promise<void>;
}

interface HttpChatProviderOptions {
  readonly baseUrl?: string;
  readonly fetch?: typeof fetch;
}

export class ChatRequestError extends Error {
  constructor(
    readonly status: number,
    message: string,
  ) {
    super(message);
    this.name = "ChatRequestError";
  }
}

class HttpChatProvider implements ChatProvider {
  readonly #baseUrl: string;
  readonly #fetch: typeof fetch;

  constructor(options: HttpChatProviderOptions) {
    this.#baseUrl = options.baseUrl ?? "/api/v1";
    this.#fetch = options.fetch ?? globalThis.fetch.bind(globalThis);
  }

  async streamQuestion(
    corpusId: string,
    question: string,
    interfaceLanguage: CorpusLanguage,
    signal: AbortSignal,
    onEvent: (event: ChatStreamEvent) => void,
  ): Promise<void> {
    const response = await this.#fetch(
      `${this.#baseUrl}/corpora/${encodeURIComponent(corpusId)}/chat/stream`,
      {
        method: "POST",
        signal,
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ question, interfaceLanguage }),
      },
    );
    if (!response.ok) {
      throw new ChatRequestError(
        response.status,
        "The grounded chat request could not be started.",
      );
    }
    if (response.body === null) {
      throw new ChatRequestError(
        502,
        "The chat stream did not provide a body.",
      );
    }
    const reader = response.body.getReader();
    const decoder = new TextDecoder();
    let buffer = "";
    for (;;) {
      const chunk = await reader.read();
      buffer += decoder.decode(chunk.value, { stream: !chunk.done });
      const records = buffer.split("\n\n");
      buffer = records.pop() ?? "";
      for (const record of records) {
        const event = parseSseRecord(record);
        if (event !== undefined) onEvent(event);
      }
      if (chunk.done) break;
    }
    const trailing = parseSseRecord(buffer);
    if (trailing !== undefined) onEvent(trailing);
  }
}

export function createHttpChatProvider(
  options: HttpChatProviderOptions = {},
): ChatProvider {
  return new HttpChatProvider(options);
}

function parseSseRecord(record: string): ChatStreamEvent | undefined {
  const data = record
    .split("\n")
    .find((line) => line.startsWith("data:"))
    ?.slice("data:".length)
    .trim();
  if (data === undefined || data === "") return undefined;
  return parseChatEvent(JSON.parse(data) as unknown);
}

export function parseChatEvent(value: unknown): ChatStreamEvent {
  if (!isRecord(value) || typeof value.type !== "string") {
    throw new TypeError("Chat stream event must have a type.");
  }
  if (typeof value.requestId !== "string") {
    throw new TypeError("Chat stream event must have a request ID.");
  }
  switch (value.type) {
    case "started":
      return {
        type: "started",
        requestId: value.requestId,
        corpusId: stringValue(value.corpusId, "chat corpus ID"),
      };
    case "evidence":
      return {
        type: "evidence",
        requestId: value.requestId,
        references: referencesValue(value.references),
      };
    case "delta":
      return {
        type: "delta",
        requestId: value.requestId,
        text: stringValue(value.text, "chat delta"),
      };
    case "completed":
      return {
        type: "completed",
        requestId: value.requestId,
        answer: stringValue(value.answer, "chat answer"),
        references: referencesValue(value.references),
        telemetry: telemetryValue(value.telemetry),
      };
    case "abstained":
      return {
        type: "abstained",
        requestId: value.requestId,
        reason: stringValue(value.reason, "abstention reason"),
        telemetry: telemetryValue(value.telemetry),
      };
    case "cancelled":
      return {
        type: "cancelled",
        requestId: value.requestId,
        telemetry: telemetryValue(value.telemetry),
      };
    case "error":
      return {
        type: "error",
        requestId: value.requestId,
        code: stringValue(value.code, "chat error code"),
        message: stringValue(value.message, "chat error message"),
        telemetry: telemetryValue(value.telemetry),
      };
    default:
      throw new TypeError("Chat stream event type is unsupported.");
  }
}

function referencesValue(value: unknown): readonly ChatReference[] {
  if (!Array.isArray(value))
    throw new TypeError("Chat references must be an array.");
  return value.map((item) => {
    if (!isRecord(item))
      throw new TypeError("Chat reference must be an object.");
    return {
      id: stringValue(item.id, "reference ID"),
      corpusId: stringValue(item.corpusId, "reference corpus ID"),
      sourceId: stringValue(item.sourceId, "reference source ID"),
      documentId: stringValue(item.documentId, "reference document ID"),
      unitLocator: stringValue(item.unitLocator, "reference locator"),
      startOffset: numberValue(item.startOffset, "reference start offset"),
      endOffset: numberValue(item.endOffset, "reference end offset"),
      excerpt: stringValue(item.excerpt, "reference excerpt"),
      rank: numberValue(item.rank, "reference rank"),
    };
  });
}

function telemetryValue(value: unknown): ChatTelemetry {
  if (!isRecord(value))
    throw new TypeError("Chat telemetry must be an object.");
  return {
    outcome: stringValue(value.outcome, "chat outcome"),
    evidenceCount: numberValue(value.evidenceCount, "evidence count"),
    durationMilliseconds: numberValue(
      value.durationMilliseconds,
      "chat duration",
    ),
  };
}

function isRecord(value: unknown): value is Record<string, unknown> {
  return typeof value === "object" && value !== null;
}

function stringValue(value: unknown, label: string): string {
  if (typeof value !== "string" || value === "")
    throw new TypeError(`${label} must be a string.`);
  return value;
}

function numberValue(value: unknown, label: string): number {
  if (typeof value !== "number" || !Number.isFinite(value))
    throw new TypeError(`${label} must be a number.`);
  return value;
}
