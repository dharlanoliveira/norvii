export type EvaluationQueryLanguage = "en" | "pt";
export type EvaluationAssetLanguage = "en" | "pt-BR";
export type EvaluationReviewDecision = "pending" | "approved" | "rejected";
export type EvaluationPublicationState =
  "draft" | "available" | "superseded" | "withdrawn";
export type EvaluationAuthorityRole =
  "statute" | "regulation" | "guidance" | "procedure";
export type EvaluationDocumentType =
  | "law"
  | "decree-law"
  | "statute"
  | "resolution"
  | "current-regulation-web-edition"
  | "official-guidance-page"
  | "guidance-pdf"
  | "official-procedure-page";

export interface EvaluationDatasetRevision {
  readonly id: string;
  readonly corpusId: string;
  readonly datasetKey: string;
  readonly semanticRevision: string;
  readonly jurisdiction: string;
  readonly manifestSha256: string;
  readonly jsonlSha256: string;
  readonly contentSha256: string;
  readonly declaredSnapshotDate: string;
  readonly queryLanguages: readonly EvaluationQueryLanguage[];
  readonly authoritativeEvidenceLanguage: EvaluationAssetLanguage;
}

export interface EvaluationDatasetReview {
  readonly decision: EvaluationReviewDecision;
  readonly publicationState: EvaluationPublicationState;
  readonly reviewedAt: string;
}

export interface EvaluationDatasetCatalogEntry {
  readonly revision: EvaluationDatasetRevision;
  readonly available: boolean;
  readonly review: EvaluationDatasetReview | null;
}

export interface EvaluationDatasetSource {
  readonly id: string;
  readonly sourceAlias: string;
  readonly title: string;
  readonly officialUrl: string;
  readonly issuingAuthority: string;
  readonly documentType: EvaluationDocumentType;
  readonly authorityRole: EvaluationAuthorityRole;
  readonly corpusSourceId: string | null;
  readonly bound: boolean;
}

export interface EvaluationDatasetStarter {
  readonly id: string;
  readonly caseId: string;
  readonly rank: number;
  readonly queryLanguage: EvaluationQueryLanguage;
  readonly reviewEligible: boolean;
}

export interface EvaluationDatasetDetail extends EvaluationDatasetCatalogEntry {
  readonly sources: readonly EvaluationDatasetSource[];
  readonly starters: readonly EvaluationDatasetStarter[];
}

export interface EvaluationMissingRequirement {
  readonly sourceAlias: string;
  readonly locator?: string | undefined;
  readonly reason: string;
}

export interface EvaluationDatasetPreflightRequest {
  readonly datasetRevisionId: string;
  readonly corpusId: string;
  readonly snapshotId: string;
}

export interface EvaluationDatasetPreflightResponse extends EvaluationDatasetPreflightRequest {
  readonly compatible: true;
  readonly missingRequirements: readonly [];
}

export type EvaluationCatalogErrorCode =
  | "maintainer_authorization_required"
  | "invalid_input"
  | "not_found"
  | "unavailable"
  | "dataset_not_available"
  | "corpus_mismatch"
  | "snapshot_incompatible"
  | "locator_unresolved"
  | "invalid_configuration"
  | "payload_too_large"
  | "internal_error";

export interface EvaluationCatalogErrorEnvelope {
  readonly error: {
    readonly code: EvaluationCatalogErrorCode;
    readonly message: string;
    readonly requestId: string;
    readonly missingRequirements?:
      readonly EvaluationMissingRequirement[] | undefined;
  };
}

const reviewDecisions = new Set<EvaluationReviewDecision>([
  "pending",
  "approved",
  "rejected",
]);
const publicationStates = new Set<EvaluationPublicationState>([
  "draft",
  "available",
  "superseded",
  "withdrawn",
]);
const authorityRoles = new Set<EvaluationAuthorityRole>([
  "statute",
  "regulation",
  "guidance",
  "procedure",
]);
const documentTypes = new Set<EvaluationDocumentType>([
  "law",
  "decree-law",
  "statute",
  "resolution",
  "current-regulation-web-edition",
  "official-guidance-page",
  "guidance-pdf",
  "official-procedure-page",
]);
const catalogErrorCodes = new Set<EvaluationCatalogErrorCode>([
  "maintainer_authorization_required",
  "invalid_input",
  "not_found",
  "unavailable",
  "dataset_not_available",
  "corpus_mismatch",
  "snapshot_incompatible",
  "locator_unresolved",
  "invalid_configuration",
  "payload_too_large",
  "internal_error",
]);

export function parseEvaluationDatasetCatalog(
  value: unknown,
): readonly EvaluationDatasetCatalogEntry[] {
  if (!Array.isArray(value)) {
    throw new TypeError("Evaluation dataset catalog must be an array.");
  }
  return value.map((entry, index) =>
    parseCatalogEntry(
      entry,
      `Evaluation dataset catalog entry at index ${String(index)}`,
    ),
  );
}

export function parseEvaluationDatasetDetail(
  value: unknown,
): EvaluationDatasetDetail {
  const detail = exactRecord(value, "Evaluation dataset detail", [
    "revision",
    "available",
    "review",
    "sources",
    "starters",
  ]);
  const entry = parseCatalogProjection(
    detail.revision,
    detail.available,
    detail.review,
  );
  if (!Array.isArray(detail.sources)) {
    throw new TypeError("Evaluation dataset sources must be an array.");
  }
  if (!Array.isArray(detail.starters)) {
    throw new TypeError("Evaluation dataset starters must be an array.");
  }
  const sources = detail.sources.map((source, index) =>
    parseSource(source, index),
  );
  const starters = detail.starters.map((starter, index) =>
    parseStarter(starter, index),
  );
  validateUnique(
    sources,
    (source) => source.id,
    "Evaluation dataset source IDs",
  );
  validateUnique(
    sources,
    (source) => source.sourceAlias,
    "Evaluation dataset source aliases",
  );
  validateUnique(
    starters,
    (starter) => starter.id,
    "Evaluation dataset starter IDs",
  );
  validateUnique(
    starters,
    (starter) => starter.caseId,
    "Evaluation dataset starter case IDs",
  );
  validateUnique(
    starters,
    (starter) => `${String(starter.rank)}:${starter.queryLanguage}`,
    "Evaluation dataset starter rank-language pairs",
  );
  return { ...entry, sources, starters };
}

export function parseEvaluationDatasetPreflightResponse(
  value: unknown,
): EvaluationDatasetPreflightResponse {
  const response = exactRecord(value, "Evaluation dataset preflight response", [
    "datasetRevisionId",
    "corpusId",
    "snapshotId",
    "compatible",
    "missingRequirements",
  ]);
  if (response.compatible !== true) {
    throw new Error(
      "Evaluation dataset preflight response must be compatible.",
    );
  }
  if (!Array.isArray(response.missingRequirements)) {
    throw new TypeError(
      "Evaluation dataset preflight missing requirements must be an array.",
    );
  }
  if (response.missingRequirements.length !== 0) {
    throw new Error(
      "Compatible evaluation dataset preflight cannot have missing requirements.",
    );
  }
  return {
    datasetRevisionId: uuidValue(
      response.datasetRevisionId,
      "Evaluation dataset preflight revision ID",
    ),
    corpusId: uuidValue(
      response.corpusId,
      "Evaluation dataset preflight corpus ID",
    ),
    snapshotId: uuidValue(
      response.snapshotId,
      "Evaluation dataset preflight snapshot ID",
    ),
    compatible: true,
    missingRequirements: [],
  };
}

export function parseEvaluationCatalogErrorEnvelope(
  value: unknown,
): EvaluationCatalogErrorEnvelope {
  const envelope = exactRecord(value, "Evaluation catalog error response", [
    "error",
  ]);
  const error = allowedRecord(envelope.error, "Evaluation catalog error", [
    "code",
    "message",
    "requestId",
    "missingRequirements",
  ]);
  const code = stringValue(error.code, "Evaluation catalog error code");
  if (!catalogErrorCodes.has(code as EvaluationCatalogErrorCode)) {
    throw new Error("Evaluation catalog error code is unsupported.");
  }
  const missingRequirements =
    error.missingRequirements === undefined
      ? undefined
      : parseMissingRequirements(error.missingRequirements);
  return {
    error: {
      code: code as EvaluationCatalogErrorCode,
      message: nonBlankString(
        error.message,
        "Evaluation catalog error message",
      ),
      requestId: uuidValue(
        error.requestId,
        "Evaluation catalog error request ID",
      ),
      ...(missingRequirements === undefined ? {} : { missingRequirements }),
    },
  };
}

export function assertEvaluationDatasetPreflightRequest(
  request: EvaluationDatasetPreflightRequest,
): void {
  const value = exactRecord(request, "Evaluation dataset preflight request", [
    "datasetRevisionId",
    "corpusId",
    "snapshotId",
  ]);
  uuidValue(
    value.datasetRevisionId,
    "Evaluation dataset preflight revision ID",
  );
  uuidValue(value.corpusId, "Evaluation dataset preflight corpus ID");
  uuidValue(value.snapshotId, "Evaluation dataset preflight snapshot ID");
}

export function assertEvaluationDatasetRevisionId(value: string): void {
  uuidValue(value, "Evaluation dataset revision ID");
}

function parseCatalogEntry(
  value: unknown,
  label: string,
): EvaluationDatasetCatalogEntry {
  const entry = exactRecord(value, label, ["revision", "available", "review"]);
  return parseCatalogProjection(entry.revision, entry.available, entry.review);
}

function parseCatalogProjection(
  revisionValue: unknown,
  availabilityValue: unknown,
  reviewValue: unknown,
): EvaluationDatasetCatalogEntry {
  const review = parseReview(reviewValue);
  const available = booleanValue(
    availabilityValue,
    "Evaluation dataset availability",
  );
  if (
    available !==
    (review?.decision === "approved" && review.publicationState === "available")
  ) {
    throw new Error(
      "Evaluation dataset availability must match the immutable review state.",
    );
  }
  return {
    revision: parseRevision(revisionValue),
    available,
    review,
  };
}

function parseRevision(value: unknown): EvaluationDatasetRevision {
  const revision = exactRecord(value, "Evaluation dataset revision", [
    "id",
    "corpusId",
    "datasetKey",
    "semanticRevision",
    "jurisdiction",
    "manifestSha256",
    "jsonlSha256",
    "contentSha256",
    "declaredSnapshotDate",
    "queryLanguages",
    "authoritativeEvidenceLanguage",
  ]);
  if (!Array.isArray(revision.queryLanguages)) {
    throw new TypeError("Evaluation dataset query languages must be an array.");
  }
  if (
    revision.queryLanguages.length === 0 ||
    revision.queryLanguages.length > 2
  ) {
    throw new Error(
      "Evaluation dataset query languages must contain one or two values.",
    );
  }
  const queryLanguages = revision.queryLanguages.map((language, index) =>
    queryLanguage(
      language,
      `Evaluation dataset query language at index ${String(index)}`,
    ),
  );
  validateUnique(
    queryLanguages,
    (language) => language,
    "Evaluation dataset query languages",
  );
  return {
    id: uuidValue(revision.id, "Evaluation dataset revision ID"),
    corpusId: uuidValue(revision.corpusId, "Evaluation dataset corpus ID"),
    datasetKey: safeIdentifier(revision.datasetKey, "Evaluation dataset key"),
    semanticRevision: nonBlankString(
      revision.semanticRevision,
      "Evaluation dataset semantic revision",
    ),
    jurisdiction: nonBlankString(
      revision.jurisdiction,
      "Evaluation dataset jurisdiction",
    ),
    manifestSha256: sha256Value(
      revision.manifestSha256,
      "Evaluation dataset manifest hash",
    ),
    jsonlSha256: sha256Value(
      revision.jsonlSha256,
      "Evaluation dataset JSONL hash",
    ),
    contentSha256: sha256Value(
      revision.contentSha256,
      "Evaluation dataset content hash",
    ),
    declaredSnapshotDate: dateValue(
      revision.declaredSnapshotDate,
      "Evaluation dataset declared snapshot date",
    ),
    queryLanguages,
    authoritativeEvidenceLanguage: assetLanguage(
      revision.authoritativeEvidenceLanguage,
      "Evaluation dataset authoritative evidence language",
    ),
  };
}

function parseReview(value: unknown): EvaluationDatasetReview | null {
  if (value === null) return null;
  const review = exactRecord(value, "Evaluation dataset review", [
    "decision",
    "publicationState",
    "reviewedAt",
  ]);
  const decision = stringValue(
    review.decision,
    "Evaluation dataset review decision",
  );
  const publicationState = stringValue(
    review.publicationState,
    "Evaluation dataset publication state",
  );
  if (!reviewDecisions.has(decision as EvaluationReviewDecision)) {
    throw new Error("Evaluation dataset review decision is unsupported.");
  }
  if (!publicationStates.has(publicationState as EvaluationPublicationState)) {
    throw new Error("Evaluation dataset publication state is unsupported.");
  }
  if (publicationState === "available" && decision !== "approved") {
    throw new Error(
      "Available evaluation dataset publications must be approved.",
    );
  }
  return {
    decision: decision as EvaluationReviewDecision,
    publicationState: publicationState as EvaluationPublicationState,
    reviewedAt: dateTimeValue(
      review.reviewedAt,
      "Evaluation dataset review time",
    ),
  };
}

function parseSource(value: unknown, index: number): EvaluationDatasetSource {
  const source = exactRecord(
    value,
    `Evaluation dataset source at index ${String(index)}`,
    [
      "id",
      "sourceAlias",
      "title",
      "officialUrl",
      "issuingAuthority",
      "documentType",
      "authorityRole",
      "corpusSourceId",
      "bound",
    ],
  );
  const documentType = stringValue(
    source.documentType,
    "Evaluation dataset source document type",
  );
  const authorityRole = stringValue(
    source.authorityRole,
    "Evaluation dataset source authority role",
  );
  if (!documentTypes.has(documentType as EvaluationDocumentType)) {
    throw new Error("Evaluation dataset source document type is unsupported.");
  }
  if (!authorityRoles.has(authorityRole as EvaluationAuthorityRole)) {
    throw new Error("Evaluation dataset source authority role is unsupported.");
  }
  const corpusSourceId = nullableUuidValue(
    source.corpusSourceId,
    "Evaluation dataset source corpus source ID",
  );
  const bound = booleanValue(
    source.bound,
    "Evaluation dataset source bound state",
  );
  if (bound !== (corpusSourceId !== null)) {
    throw new Error(
      "Evaluation dataset source bound state must match its corpus source identity.",
    );
  }
  return {
    id: uuidValue(source.id, "Evaluation dataset source ID"),
    sourceAlias: safeIdentifier(
      source.sourceAlias,
      "Evaluation dataset source alias",
    ),
    title: nonBlankString(source.title, "Evaluation dataset source title"),
    officialUrl: httpsUrlValue(
      source.officialUrl,
      "Evaluation dataset source official URL",
    ),
    issuingAuthority: nonBlankString(
      source.issuingAuthority,
      "Evaluation dataset source issuing authority",
    ),
    documentType: documentType as EvaluationDocumentType,
    authorityRole: authorityRole as EvaluationAuthorityRole,
    corpusSourceId,
    bound,
  };
}

function parseStarter(value: unknown, index: number): EvaluationDatasetStarter {
  const starter = exactRecord(
    value,
    `Evaluation dataset starter at index ${String(index)}`,
    ["id", "caseId", "rank", "queryLanguage", "reviewEligible"],
  );
  const reviewEligible = booleanValue(
    starter.reviewEligible,
    "Evaluation dataset starter review eligibility",
  );
  if (!reviewEligible) {
    throw new Error("Evaluation dataset starters must be review eligible.");
  }
  return {
    id: uuidValue(starter.id, "Evaluation dataset starter ID"),
    caseId: uuidValue(starter.caseId, "Evaluation dataset starter case ID"),
    rank: boundedPositiveInteger(
      starter.rank,
      "Evaluation dataset starter rank",
      5,
    ),
    queryLanguage: queryLanguage(
      starter.queryLanguage,
      "Evaluation dataset starter query language",
    ),
    reviewEligible,
  };
}

function parseMissingRequirements(
  value: unknown,
): readonly EvaluationMissingRequirement[] {
  if (!Array.isArray(value)) {
    throw new TypeError("Evaluation missing requirements must be an array.");
  }
  if (value.length > 32) {
    throw new Error("Evaluation missing requirements cannot exceed 32 items.");
  }
  return value.map((item, index) => {
    const requirement = allowedRecord(
      item,
      `Evaluation missing requirement at index ${String(index)}`,
      ["sourceAlias", "locator", "reason"],
    );
    const locator =
      requirement.locator === undefined
        ? undefined
        : nonBlankString(
            requirement.locator,
            `Evaluation missing requirement locator at index ${String(index)}`,
          );
    return {
      sourceAlias: safeIdentifier(
        requirement.sourceAlias,
        `Evaluation missing requirement source alias at index ${String(index)}`,
      ),
      ...(locator === undefined ? {} : { locator }),
      reason: nonBlankString(
        requirement.reason,
        `Evaluation missing requirement reason at index ${String(index)}`,
      ),
    };
  });
}

function exactRecord(
  value: unknown,
  label: string,
  fields: readonly string[],
): Record<string, unknown> {
  const object = allowedRecord(value, label, fields);
  for (const field of fields) {
    if (!Object.hasOwn(object, field)) {
      throw new Error(`${label} must contain ${field}.`);
    }
  }
  return object;
}

function allowedRecord(
  value: unknown,
  label: string,
  fields: readonly string[],
): Record<string, unknown> {
  if (typeof value !== "object" || value === null || Array.isArray(value)) {
    throw new TypeError(`${label} must be an object.`);
  }
  const object = value as Record<string, unknown>;
  const allowed = new Set(fields);
  for (const key of Object.keys(object)) {
    if (!allowed.has(key)) {
      throw new Error(`${label} contains an unsupported field: ${key}.`);
    }
  }
  return object;
}

function stringValue(value: unknown, label: string): string {
  if (typeof value !== "string" || value.length === 0) {
    throw new TypeError(`${label} must be a non-empty string.`);
  }
  return value;
}

function nonBlankString(value: unknown, label: string): string {
  const text = stringValue(value, label);
  if (text.trim().length === 0) {
    throw new Error(`${label} must not be blank.`);
  }
  return text;
}

function safeIdentifier(value: unknown, label: string): string {
  const identifier = nonBlankString(value, label);
  if (!/^[a-z0-9][a-z0-9-]{0,127}$/u.test(identifier)) {
    throw new Error(`${label} must be a safe identifier.`);
  }
  return identifier;
}

function booleanValue(value: unknown, label: string): boolean {
  if (typeof value !== "boolean") {
    throw new TypeError(`${label} must be a boolean.`);
  }
  return value;
}

function uuidValue(value: unknown, label: string): string {
  const identifier = stringValue(value, label);
  if (
    !/^[0-9a-f]{8}-[0-9a-f]{4}-[1-5][0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$/iu.test(
      identifier,
    )
  ) {
    throw new Error(`${label} must be a UUID.`);
  }
  return identifier;
}

function nullableUuidValue(value: unknown, label: string): string | null {
  return value === null ? null : uuidValue(value, label);
}

function sha256Value(value: unknown, label: string): string {
  const hash = stringValue(value, label);
  if (!/^[0-9a-f]{64}$/u.test(hash)) {
    throw new Error(`${label} must be a lowercase SHA-256 hash.`);
  }
  return hash;
}

function dateValue(value: unknown, label: string): string {
  const date = stringValue(value, label);
  if (!isValidIsoDate(date)) {
    throw new Error(`${label} must be an ISO date.`);
  }
  return date;
}

function dateTimeValue(value: unknown, label: string): string {
  const dateTime = stringValue(value, label);
  const date = dateTime.slice(0, 10);
  if (
    !/^\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}(?:\.\d+)?(?:Z|[+-]\d{2}:\d{2})$/u.test(
      dateTime,
    ) ||
    !isValidIsoDate(date) ||
    Number.isNaN(Date.parse(dateTime))
  ) {
    throw new Error(`${label} must be an RFC 3339 timestamp.`);
  }
  return dateTime;
}

function isValidIsoDate(value: string): boolean {
  if (!/^\d{4}-\d{2}-\d{2}$/u.test(value)) return false;
  const parsed = new Date(`${value}T00:00:00Z`);
  return (
    !Number.isNaN(parsed.valueOf()) &&
    parsed.toISOString().slice(0, 10) === value
  );
}

function queryLanguage(value: unknown, label: string): EvaluationQueryLanguage {
  const language = stringValue(value, label);
  if (language !== "en" && language !== "pt") {
    throw new Error(`${label} must be en or pt.`);
  }
  return language;
}

function assetLanguage(value: unknown, label: string): EvaluationAssetLanguage {
  const language = stringValue(value, label);
  if (language !== "en" && language !== "pt-BR") {
    throw new Error(`${label} must be en or pt-BR.`);
  }
  return language;
}

function boundedPositiveInteger(
  value: unknown,
  label: string,
  maximum: number,
): number {
  if (
    typeof value !== "number" ||
    !Number.isInteger(value) ||
    value < 1 ||
    value > maximum
  ) {
    throw new Error(
      `${label} must be an integer between one and ${String(maximum)}.`,
    );
  }
  return value;
}

function httpsUrlValue(value: unknown, label: string): string {
  const rawUrl = stringValue(value, label);
  let url: URL;
  try {
    url = new URL(rawUrl);
  } catch {
    throw new Error(`${label} must be an HTTPS URL.`);
  }
  if (url.protocol !== "https:" || url.hostname.length === 0) {
    throw new Error(`${label} must be an HTTPS URL.`);
  }
  return rawUrl;
}

function validateUnique<T>(
  values: readonly T[],
  key: (value: T) => string,
  label: string,
): void {
  const seen = new Set<string>();
  for (const value of values) {
    const valueKey = key(value);
    if (seen.has(valueKey)) {
      throw new Error(`${label} must be unique.`);
    }
    seen.add(valueKey);
  }
}
