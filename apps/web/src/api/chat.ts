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
  readonly documentVersionId?: string | undefined;
  readonly sourceRevisionId?: string | undefined;
  readonly pipelineVersion?: string | undefined;
  readonly sourceTitle?: string | undefined;
  readonly cosineDistance?: number | null | undefined;
}

export interface ChatInspection {
  readonly outcome: "completed" | "abstained" | "cancelled" | "failed";
  readonly retrieval?:
    | {
        readonly strategy: "vector";
        readonly topK: number;
        readonly returnedCount: number;
        readonly embeddingModel: string | null;
      }
    | undefined;
  readonly measurements: {
    readonly retrievalMilliseconds: number | null;
    readonly generationMilliseconds: number | null;
    readonly totalMilliseconds: number | null;
    readonly inputTokens: number | null;
    readonly outputTokens: number | null;
  };
  readonly evidence?: readonly ChatReference[] | undefined;
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
      readonly inspection?: ChatInspection | undefined;
    }
  | {
      readonly type: "abstained";
      readonly requestId: string;
      readonly reason: string;
      readonly telemetry: ChatTelemetry;
      readonly inspection?: ChatInspection | undefined;
    }
  | {
      readonly type: "cancelled";
      readonly requestId: string;
      readonly telemetry: ChatTelemetry;
      readonly inspection?: ChatInspection | undefined;
    }
  | {
      readonly type: "error";
      readonly requestId: string;
      readonly code: string;
      readonly message: string;
      readonly telemetry: ChatTelemetry;
      readonly inspection?: ChatInspection | undefined;
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
        inspection: inspectionValue(value.inspection),
      };
    case "abstained":
      return {
        type: "abstained",
        requestId: value.requestId,
        reason: stringValue(value.reason, "abstention reason"),
        telemetry: telemetryValue(value.telemetry),
        inspection: inspectionValue(value.inspection),
      };
    case "cancelled":
      return {
        type: "cancelled",
        requestId: value.requestId,
        telemetry: telemetryValue(value.telemetry),
        inspection: inspectionValue(value.inspection),
      };
    case "error":
      return {
        type: "error",
        requestId: value.requestId,
        code: stringValue(value.code, "chat error code"),
        message: stringValue(value.message, "chat error message"),
        telemetry: telemetryValue(value.telemetry),
        inspection: inspectionValue(value.inspection),
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
    const documentVersionId = optionalStringValue(
      item.documentVersionId,
      "reference document version ID",
    );
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
      documentVersionId,
      sourceRevisionId: optionalStringValue(
        item.sourceRevisionId,
        "reference source revision ID",
      ),
      pipelineVersion: optionalStringValue(
        item.pipelineVersion,
        "reference pipeline version",
      ),
      sourceTitle: optionalStringValue(
        item.sourceTitle,
        "reference source title",
      ),
      cosineDistance: nullableNonNegativeNumberValue(
        item.cosineDistance,
        "reference cosine distance",
      ),
    };
  });
}

function inspectionValue(value: unknown): ChatInspection | undefined {
  if (value === undefined) return undefined;
  if (!isRecord(value))
    throw new TypeError("Chat inspection must be an object.");
  const outcome = stringValue(value.outcome, "inspection outcome");
  if (!isInspectionOutcome(outcome)) {
    throw new TypeError("Chat inspection outcome is invalid.");
  }
  return {
    outcome,
    retrieval: retrievalValue(value.retrieval),
    measurements: measurementsValue(value.measurements),
    evidence:
      value.evidence === undefined
        ? undefined
        : referencesValue(value.evidence),
  };
}

function retrievalValue(value: unknown): ChatInspection["retrieval"] {
  if (value === undefined) return undefined;
  if (!isRecord(value))
    throw new TypeError("Chat retrieval inspection must be an object.");
  if (value.strategy !== "vector") {
    throw new TypeError("Chat retrieval strategy is invalid.");
  }
  return {
    strategy: "vector",
    topK: nonNegativeNumberValue(value.topK, "retrieval top K"),
    returnedCount: nonNegativeNumberValue(
      value.returnedCount,
      "retrieval returned count",
    ),
    embeddingModel: nullableStringValue(
      value.embeddingModel,
      "embedding model",
    ),
  };
}

function measurementsValue(value: unknown): ChatInspection["measurements"] {
  if (!isRecord(value))
    throw new TypeError("Chat measurements must be an object.");
  return {
    retrievalMilliseconds: measurementValue(
      value.retrievalMilliseconds,
      "retrieval duration",
    ),
    generationMilliseconds: measurementValue(
      value.generationMilliseconds,
      "generation duration",
    ),
    totalMilliseconds: measurementValue(
      value.totalMilliseconds,
      "total duration",
    ),
    inputTokens: measurementValue(value.inputTokens, "input tokens"),
    outputTokens: measurementValue(value.outputTokens, "output tokens"),
  };
}

function isInspectionOutcome(
  value: string,
): value is ChatInspection["outcome"] {
  return (
    value === "completed" ||
    value === "abstained" ||
    value === "cancelled" ||
    value === "failed"
  );
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

function nonNegativeNumberValue(value: unknown, label: string): number {
  const number = numberValue(value, label);
  if (number < 0) throw new TypeError(`${label} must not be negative.`);
  return number;
}

function nullableNonNegativeNumberValue(
  value: unknown,
  label: string,
): number | null | undefined {
  if (value === undefined) return undefined;
  if (value === null) return null;
  return nonNegativeNumberValue(value, label);
}

function measurementValue(value: unknown, label: string): number | null {
  if (value === null) return null;
  return nonNegativeNumberValue(value, label);
}

function optionalStringValue(
  value: unknown,
  label: string,
): string | undefined {
  if (value === undefined) return undefined;
  return stringValue(value, label);
}

function nullableStringValue(value: unknown, label: string): string | null {
  if (value === null) return null;
  return stringValue(value, label);
}
