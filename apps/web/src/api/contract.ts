export type CorpusLanguage = "en" | "pt";
export type CorpusStatus = "enabled" | "disabled";

export interface CorpusResponse {
  readonly id: string;
  readonly name: string;
  readonly description: string;
  readonly language: CorpusLanguage;
  readonly jurisdiction: string;
  readonly status: CorpusStatus;
  readonly sourceCount: number;
  readonly version: number;
  readonly createdAt: string;
  readonly updatedAt: string;
}

export type SourceKind = "url" | "pdf";
export type ProcessingStatus = "pending" | "processing" | "ready" | "failed";

export interface SourceResponse {
  readonly id: string;
  readonly corpusId: string;
  readonly title: string;
  readonly kind: SourceKind;
  readonly processingStatus: ProcessingStatus;
  readonly failureCategory: string | null;
  readonly latestReadyDocumentId: string | null;
  readonly version: number;
  readonly createdAt: string;
  readonly updatedAt: string;
  readonly origin: SourceOriginResponse;
  readonly latestAttempt: ProcessingAttemptResponse | null;
  readonly attempts: readonly ProcessingAttemptResponse[];
}

export interface SourceOriginResponse {
  readonly kind: SourceKind;
  readonly submittedUrl: string | null;
  readonly normalizedUrl: string | null;
  readonly originalFilename: string | null;
  readonly mediaType: string | null;
  readonly byteSize: number | null;
  readonly sha256: string | null;
  readonly finalUrl: string | null;
  readonly capturedAt: string | null;
  readonly extractedContentSha256: string | null;
}

export interface ProcessingAttemptResponse {
  readonly number: number;
  readonly pipelineVersion: string;
  readonly status: "processing" | "succeeded" | "failed";
  readonly startedAt: string;
  readonly finishedAt: string | null;
  readonly failureCategory: string | null;
  readonly acquiredByteCount: number | null;
  readonly normalizedCharacterCount: number | null;
  readonly unitCount: number | null;
  readonly durationMilliseconds: number | null;
}

export interface DocumentUnitResponse {
  readonly id: string;
  readonly parentId: string | null;
  readonly kind: string;
  readonly ordinal: number;
  readonly marker: string | null;
  readonly label: string | null;
  readonly startOffset: number;
  readonly endOffset: number;
  readonly startPage: number | null;
  readonly endPage: number | null;
  readonly locator: string;
  readonly contentSha256: string;
}

export interface DocumentResponse {
  readonly id: string;
  readonly sourceRevisionId: string;
  readonly pipelineVersion: string;
  readonly text: string;
  readonly textSha256: string;
  readonly createdAt: string;
  readonly units: readonly DocumentUnitResponse[];
  readonly provenance: RevisionProvenanceResponse;
}

export interface RevisionProvenanceResponse {
  readonly contentSha256: string;
  readonly capturedAt: string;
  readonly mediaType: string;
  readonly byteSize: number;
  readonly finalUrl: string | null;
  readonly extractedContentSha256: string | null;
}

export type PublicErrorCode =
  | "invalid_input"
  | "payload_too_large"
  | "unsafe_url"
  | "unsupported_content"
  | "duplicate_source"
  | "stale_state"
  | "not_found"
  | "unavailable"
  | "acquisition_failed"
  | "extraction_failed"
  | "publication_failed"
  | "internal_error";

export interface ErrorEnvelope {
  readonly error: {
    readonly code: PublicErrorCode;
    readonly message: string;
    readonly fields?: Readonly<Record<string, string>>;
    readonly requestId: string;
  };
}

const errorCodes = new Set<PublicErrorCode>([
  "invalid_input",
  "payload_too_large",
  "unsafe_url",
  "unsupported_content",
  "duplicate_source",
  "stale_state",
  "not_found",
  "unavailable",
  "acquisition_failed",
  "extraction_failed",
  "publication_failed",
  "internal_error",
]);

export function parseCorpusList(value: unknown): readonly CorpusResponse[] {
  if (!Array.isArray(value)) {
    throw new TypeError("Corpus list response must be an array.");
  }
  return value.map((item, index) => parseCorpus(item, index));
}

export function parseCorpusResponse(value: unknown): CorpusResponse {
  return parseCorpus(value, 0);
}

export function parseSourceList(value: unknown): readonly SourceResponse[] {
  if (!Array.isArray(value)) {
    throw new TypeError("Source list response must be an array.");
  }
  return value.map((item, index) => parseSource(item, index));
}

export function parseDocumentResponse(value: unknown): DocumentResponse {
  const document = record(value, "Document response");
  const rawUnits = document.units;
  if (!Array.isArray(rawUnits)) {
    throw new TypeError("Document units must be an array.");
  }
  return {
    id: uuidValue(document.id, "document ID"),
    sourceRevisionId: uuidValue(
      document.sourceRevisionId,
      "source revision ID",
    ),
    pipelineVersion: stringValue(document.pipelineVersion, "pipeline version"),
    text: stringValue(document.text, "document text"),
    textSha256: sha256Value(document.textSha256, "document text hash"),
    createdAt: dateTimeValue(document.createdAt, "document creation time"),
    units: rawUnits.map((unit, index) => parseDocumentUnit(unit, index)),
    provenance: parseProvenance(document.provenance),
  };
}

export function parseErrorEnvelope(value: unknown): ErrorEnvelope {
  const envelope = record(value, "Error response");
  const error = record(envelope.error, "Error response error");
  const code = stringValue(error.code, "error code");
  if (!errorCodes.has(code as PublicErrorCode)) {
    throw new Error("Error response contains an unsupported error code.");
  }
  const fields = optionalStringRecord(error.fields, "error fields");
  return {
    error: {
      code: code as PublicErrorCode,
      message: stringValue(error.message, "error message"),
      requestId: uuidValue(error.requestId, "request ID"),
      ...(fields === undefined ? {} : { fields }),
    },
  };
}

function parseCorpus(value: unknown, index: number): CorpusResponse {
  const item = record(value, `Corpus at index ${String(index)}`);
  const language = stringValue(item.language, "corpus language");
  const status = stringValue(item.status, "corpus status");
  if (language !== "en" && language !== "pt") {
    throw new Error("Corpus language must be en or pt.");
  }
  if (status !== "enabled" && status !== "disabled") {
    throw new Error("Corpus status must be enabled or disabled.");
  }
  return {
    id: uuidValue(item.id, "corpus ID"),
    name: stringValue(item.name, "corpus name"),
    description: stringValue(item.description, "corpus description"),
    language,
    jurisdiction: stringValue(item.jurisdiction, "corpus jurisdiction"),
    status,
    sourceCount: nonnegativeInteger(item.sourceCount, "corpus source count"),
    version: positiveInteger(item.version, "corpus version"),
    createdAt: dateTimeValue(item.createdAt, "corpus creation time"),
    updatedAt: dateTimeValue(item.updatedAt, "corpus update time"),
  };
}

function parseSource(value: unknown, index: number): SourceResponse {
  const source = record(value, `Source at index ${String(index)}`);
  const kind = stringValue(source.kind, "source kind");
  const processingStatus = stringValue(
    source.processingStatus,
    "source processing status",
  );
  if (kind !== "url" && kind !== "pdf") {
    throw new Error("Source kind must be url or pdf.");
  }
  if (
    !["pending", "processing", "ready", "failed"].includes(processingStatus)
  ) {
    throw new Error("Source processing status is unsupported.");
  }
  return {
    id: uuidValue(source.id, "source ID"),
    corpusId: uuidValue(source.corpusId, "source corpus ID"),
    title: stringValue(source.title, "source title"),
    kind,
    processingStatus: processingStatus as ProcessingStatus,
    failureCategory: nullableString(
      source.failureCategory,
      "source failure category",
    ),
    latestReadyDocumentId: nullableUUID(
      source.latestReadyDocumentId,
      "latest document ID",
    ),
    version: positiveInteger(source.version, "source version"),
    createdAt: dateTimeValue(source.createdAt, "source creation time"),
    updatedAt: dateTimeValue(source.updatedAt, "source update time"),
    origin: parseOrigin(source.origin, kind),
    latestAttempt: parseAttempt(source.latestAttempt),
    attempts: parseAttempts(source.attempts),
  };
}

function parseOrigin(
  value: unknown,
  sourceKind: SourceKind,
): SourceOriginResponse {
  const origin = record(value, "source origin");
  const kind = stringValue(origin.kind, "source origin kind");
  if (kind !== sourceKind)
    throw new Error("Source origin kind must match its source.");
  return {
    kind,
    submittedUrl: nullableString(origin.submittedUrl, "submitted URL"),
    normalizedUrl: nullableString(origin.normalizedUrl, "normalized URL"),
    originalFilename: nullableString(
      origin.originalFilename,
      "original filename",
    ),
    mediaType: nullableString(origin.mediaType, "origin media type"),
    byteSize: nullableNonnegativeInteger(origin.byteSize, "origin byte size"),
    sha256: nullableSha256(origin.sha256, "origin hash"),
    finalUrl: nullableString(origin.finalUrl, "captured final URL"),
    capturedAt: nullableDateTime(origin.capturedAt, "capture time"),
    extractedContentSha256: nullableSha256(
      origin.extractedContentSha256,
      "extracted content hash",
    ),
  };
}

function parseAttempt(value: unknown): ProcessingAttemptResponse | null {
  if (value === null || value === undefined) return null;
  const attempt = record(value, "latest processing attempt");
  const status = stringValue(attempt.status, "attempt status");
  if (
    status !== "processing" &&
    status !== "succeeded" &&
    status !== "failed"
  ) {
    throw new Error("Processing attempt status is unsupported.");
  }
  return {
    number: positiveInteger(attempt.number, "attempt number"),
    pipelineVersion: stringValue(
      attempt.pipelineVersion,
      "attempt pipeline version",
    ),
    status,
    startedAt: dateTimeValue(attempt.startedAt, "attempt start time"),
    finishedAt: nullableDateTime(attempt.finishedAt, "attempt finish time"),
    failureCategory: nullableString(
      attempt.failureCategory,
      "attempt failure category",
    ),
    acquiredByteCount: nullableNonnegativeInteger(
      attempt.acquiredByteCount,
      "acquired bytes",
    ),
    normalizedCharacterCount: nullableNonnegativeInteger(
      attempt.normalizedCharacterCount,
      "normalized characters",
    ),
    unitCount: nullableNonnegativeInteger(attempt.unitCount, "unit count"),
    durationMilliseconds: nullableNonnegativeInteger(
      attempt.durationMilliseconds,
      "attempt duration",
    ),
  };
}

function parseAttempts(value: unknown): readonly ProcessingAttemptResponse[] {
  if (!Array.isArray(value))
    throw new Error("Processing attempt history must be an array.");
  return value.map((attempt) => {
    const parsed = parseAttempt(attempt);
    if (parsed === null)
      throw new Error("Processing attempt history cannot contain null.");
    return parsed;
  });
}

function parseProvenance(value: unknown): RevisionProvenanceResponse {
  const provenance = record(value, "document provenance");
  return {
    contentSha256: sha256Value(
      provenance.contentSha256,
      "revision content hash",
    ),
    capturedAt: dateTimeValue(provenance.capturedAt, "revision capture time"),
    mediaType: stringValue(provenance.mediaType, "revision media type"),
    byteSize: positiveInteger(provenance.byteSize, "revision byte size"),
    finalUrl: nullableString(provenance.finalUrl, "revision final URL"),
    extractedContentSha256: nullableSha256(
      provenance.extractedContentSha256,
      "revision extracted content hash",
    ),
  };
}

function parseDocumentUnit(
  value: unknown,
  index: number,
): DocumentUnitResponse {
  const unit = record(value, `Document unit at index ${String(index)}`);
  return {
    id: uuidValue(unit.id, "document unit ID"),
    parentId: nullableUUID(unit.parentId, "document unit parent ID"),
    kind: stringValue(unit.kind, "document unit kind"),
    ordinal: nonnegativeInteger(unit.ordinal, "document unit ordinal"),
    marker: nullableString(unit.marker, "document unit marker"),
    label: nullableString(unit.label, "document unit label"),
    startOffset: nonnegativeInteger(
      unit.startOffset,
      "document unit start offset",
    ),
    endOffset: nonnegativeInteger(unit.endOffset, "document unit end offset"),
    startPage: nullablePositiveInteger(
      unit.startPage,
      "document unit start page",
    ),
    endPage: nullablePositiveInteger(unit.endPage, "document unit end page"),
    locator: stringValue(unit.locator, "document unit locator"),
    contentSha256: sha256Value(
      unit.contentSha256,
      "document unit content hash",
    ),
  };
}

function record(value: unknown, label: string): Record<string, unknown> {
  if (typeof value !== "object" || value === null || Array.isArray(value)) {
    throw new Error(`${label} must be an object.`);
  }
  return value as Record<string, unknown>;
}

function stringValue(value: unknown, label: string): string {
  if (typeof value !== "string" || value.length === 0) {
    throw new Error(`${label} must be a non-empty string.`);
  }
  return value;
}

function uuidValue(value: unknown, label: string): string {
  const text = stringValue(value, label);
  if (
    !/^[0-9a-f]{8}-[0-9a-f]{4}-[1-5][0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$/iu.test(
      text,
    )
  ) {
    throw new Error(`${label} must be a UUID.`);
  }
  return text;
}

function nonnegativeInteger(value: unknown, label: string): number {
  if (typeof value !== "number" || !Number.isInteger(value) || value < 0) {
    throw new Error(`${label} must be a nonnegative integer.`);
  }
  return value;
}

function positiveInteger(value: unknown, label: string): number {
  const result = nonnegativeInteger(value, label);
  if (result === 0) {
    throw new Error(`${label} must be positive.`);
  }
  return result;
}

function nullablePositiveInteger(value: unknown, label: string): number | null {
  if (value === null || value === undefined) return null;
  return positiveInteger(value, label);
}

function nullableNonnegativeInteger(
  value: unknown,
  label: string,
): number | null {
  if (value === null || value === undefined) return null;
  return nonnegativeInteger(value, label);
}

function dateTimeValue(value: unknown, label: string): string {
  const text = stringValue(value, label);
  if (Number.isNaN(Date.parse(text))) {
    throw new TypeError(`${label} must be an RFC 3339 timestamp.`);
  }
  return text;
}

function nullableString(value: unknown, label: string): string | null {
  if (value === null || value === undefined) return null;
  return stringValue(value, label);
}

function nullableDateTime(value: unknown, label: string): string | null {
  if (value === null || value === undefined) return null;
  return dateTimeValue(value, label);
}

function nullableUUID(value: unknown, label: string): string | null {
  if (value === null || value === undefined) return null;
  return uuidValue(value, label);
}

function sha256Value(value: unknown, label: string): string {
  const text = stringValue(value, label);
  if (!/^[0-9a-f]{64}$/u.test(text)) {
    throw new Error(`${label} must be a SHA-256 hash.`);
  }
  return text;
}

function nullableSha256(value: unknown, label: string): string | null {
  if (value === null || value === undefined) return null;
  return sha256Value(value, label);
}

function optionalStringRecord(
  value: unknown,
  label: string,
): Readonly<Record<string, string>> | undefined {
  if (value === undefined) {
    return undefined;
  }
  const result: Record<string, string> = {};
  for (const [key, item] of Object.entries(record(value, label))) {
    result[key] = stringValue(item, `${label}.${key}`);
  }
  return result;
}
