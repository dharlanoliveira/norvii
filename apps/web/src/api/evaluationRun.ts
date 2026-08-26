export type EvaluationRunState =
  "queued" | "running" | "completed" | "completed_with_failures" | "failed";
export type EvaluationRunCaseState =
  "pending" | "leased" | "completed" | "abstained" | "failed" | "cancelled";
export type EvaluationMetricState =
  "scored" | "not_scored" | "not_applicable" | "needs_human_review";
export type EvaluationExpectedOutcome = "answer" | "abstain";
export type EvaluationEvidenceKind = "expected" | "retrieved" | "cited";
export type EvaluationComparisonState = "comparable" | "non_comparable";
export type EvaluationMetricName =
  | "retrieval_coverage"
  | "citation_coverage"
  | "citation_validity"
  | "citation_scope_validity"
  | "expected_abstention_outcome"
  | "execution_outcome"
  | "latency_milliseconds"
  | "input_tokens"
  | "output_tokens"
  | "semantic_claim_support";
export type EvaluationFailureCode =
  | "provider_unavailable"
  | "execution_retryable"
  | "invalid_execution_result"
  | "frozen_identity_unavailable";
export type EvaluationComparisonDifferenceField =
  | "dataset_revision_id"
  | "dataset_content_sha256"
  | "corpus_id"
  | "snapshot_id"
  | "snapshot_manifest_sha256"
  | "ordered_case_set_sha256"
  | "scoring_policy_version";

export interface EvaluationMetric {
  readonly name: EvaluationMetricName;
  readonly state: EvaluationMetricState;
  readonly value: number | null;
  readonly numerator: number | null;
  readonly denominator: number | null;
  readonly rationale: string;
  readonly scorerVersion: string;
}

export interface EvaluationRunCaseSummary {
  readonly id: string;
  readonly datasetCaseId: string;
  readonly position: number;
  readonly state: EvaluationRunCaseState;
  readonly attemptCount: number;
  readonly finishedAt: string | null;
  readonly failureCode: EvaluationFailureCode | null;
}

export interface EvaluationRunSummary {
  readonly id: string;
  readonly datasetRevision: {
    readonly id: string;
    readonly contentSha256: string;
  };
  readonly corpusId: string;
  readonly snapshotId: string;
  readonly snapshotManifestSha256: string;
  readonly orderedCaseSetSha256: string;
  readonly configuration: {
    readonly strategy: "vector" | "hybrid";
    readonly fingerprint: string;
  };
  readonly scoringPolicyVersion: string;
  readonly agentBuild: string;
  readonly chatModelIdentity: string;
  readonly embeddingModelIdentity: string;
  readonly initiatedBy: string;
  readonly state: EvaluationRunState;
  readonly createdAt: string;
  readonly startedAt: string | null;
  readonly completedAt: string | null;
  readonly aggregate: {
    readonly total: number;
    readonly eligible: number;
    readonly scored: number;
    readonly failed: number;
    readonly cancelled: number;
    readonly notApplicable: number;
    readonly metrics: readonly EvaluationMetric[];
  };
  readonly cases: readonly EvaluationRunCaseSummary[];
}

export interface EvaluationEvidence {
  readonly kind: EvaluationEvidenceKind;
  readonly position: number;
  readonly markerPosition: number | null;
  readonly corpusId: string;
  readonly snapshotId: string;
  readonly sourceId: string;
  readonly sourceRevisionId: string;
  readonly documentId: string;
  readonly legalUnitId: string;
  readonly canonicalLocator: string;
  readonly displayLocator: string;
  readonly startOffset: number | null;
  readonly endOffset: number | null;
  readonly contentSha256: string;
}

export interface EvaluationRunCase extends EvaluationRunCaseSummary {
  readonly runId: string;
  readonly corpusId: string;
  readonly snapshotId: string;
  readonly datasetRevisionId: string;
  readonly question: string;
  readonly referenceAnswer: string;
  readonly expectedOutcome: EvaluationExpectedOutcome;
  readonly expectedEvidence: readonly EvaluationEvidence[];
  readonly actualEvidence: readonly EvaluationEvidence[];
  readonly answer: string;
  readonly graphGroundingState: string;
  readonly latencyMilliseconds: number | null;
  readonly inputTokens: number | null;
  readonly outputTokens: number | null;
  readonly metrics: readonly EvaluationMetric[];
}

interface EvaluationComparisonExperiment {
  readonly retrievalStrategy: "vector" | "hybrid";
  readonly retrievalConfigurationFingerprint: string;
  readonly agentBuild: string;
  readonly chatModelIdentity: string;
  readonly embeddingModelIdentity: string;
}

interface EvaluationComparisonBase {
  readonly comparisonState: EvaluationComparisonState;
  readonly leftRunId: string;
  readonly rightRunId: string;
  readonly left: EvaluationComparisonExperiment;
  readonly right: EvaluationComparisonExperiment;
}

export interface ComparableEvaluationComparison extends EvaluationComparisonBase {
  readonly comparisonState: "comparable";
  readonly differences: readonly [];
  readonly totals: {
    readonly leftCases: number;
    readonly rightCases: number;
    readonly pairedCases: number;
    readonly leftUnpaired: number;
    readonly rightUnpaired: number;
    readonly failedOrCancelled: number;
    readonly leftFailed: number;
    readonly leftCancelled: number;
    readonly rightFailed: number;
    readonly rightCancelled: number;
  };
  readonly metrics: readonly {
    readonly name: EvaluationMetricName;
    readonly state: EvaluationMetricState;
    readonly pairedCases: number;
    readonly leftNumerator: number;
    readonly leftDenominator: number;
    readonly rightNumerator: number;
    readonly rightDenominator: number;
    readonly leftValue: number | null;
    readonly rightValue: number | null;
    readonly delta: number | null;
  }[];
}

export interface NonComparableEvaluationComparison extends EvaluationComparisonBase {
  readonly comparisonState: "non_comparable";
  readonly differences: readonly {
    readonly field: EvaluationComparisonDifferenceField;
  }[];
  readonly totals: null;
  readonly metrics: readonly [];
}

export type EvaluationComparison =
  ComparableEvaluationComparison | NonComparableEvaluationComparison;

const uuidPattern =
  /^[0-9a-f]{8}-[0-9a-f]{4}-[1-5][0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$/i;
const sha256Pattern = /^[0-9a-f]{64}$/;
const runStates = new Set<EvaluationRunState>([
  "queued",
  "running",
  "completed",
  "completed_with_failures",
  "failed",
]);
const caseStates = new Set<EvaluationRunCaseState>([
  "pending",
  "leased",
  "completed",
  "abstained",
  "failed",
  "cancelled",
]);
const metricStates = new Set<EvaluationMetricState>([
  "scored",
  "not_scored",
  "not_applicable",
  "needs_human_review",
]);
const metricNames = new Set<EvaluationMetricName>([
  "retrieval_coverage",
  "citation_coverage",
  "citation_validity",
  "citation_scope_validity",
  "expected_abstention_outcome",
  "execution_outcome",
  "latency_milliseconds",
  "input_tokens",
  "output_tokens",
  "semantic_claim_support",
]);
const failureCodes = new Set<EvaluationFailureCode>([
  "provider_unavailable",
  "execution_retryable",
  "invalid_execution_result",
  "frozen_identity_unavailable",
]);
const comparisonDifferenceFields = new Set<EvaluationComparisonDifferenceField>(
  [
    "dataset_revision_id",
    "dataset_content_sha256",
    "corpus_id",
    "snapshot_id",
    "snapshot_manifest_sha256",
    "ordered_case_set_sha256",
    "scoring_policy_version",
  ],
);

export function assertEvaluationRunId(value: string): void {
  uuid(value, "Evaluation run ID");
}

export function assertEvaluationRunCaseId(value: string): void {
  uuid(value, "Evaluation run case ID");
}

export function parseEvaluationRunSummary(
  value: unknown,
): EvaluationRunSummary {
  const record = recordWithOptional(
    value,
    "Evaluation run summary",
    [
      "id",
      "datasetRevision",
      "corpusId",
      "snapshotId",
      "snapshotManifestSha256",
      "orderedCaseSetSha256",
      "configuration",
      "scoringPolicyVersion",
      "agentBuild",
      "chatModelIdentity",
      "embeddingModelIdentity",
      "initiatedBy",
      "state",
      "createdAt",
      "aggregate",
      "cases",
    ],
    ["startedAt", "completedAt"],
  );
  const aggregate = exactRecord(record.aggregate, "Evaluation run aggregate", [
    "total",
    "eligible",
    "scored",
    "failed",
    "cancelled",
    "notApplicable",
    "metrics",
  ]);
  return {
    id: uuid(record.id, "Evaluation run ID"),
    datasetRevision: parseRevision(record.datasetRevision),
    corpusId: uuid(record.corpusId, "Evaluation run corpus ID"),
    snapshotId: uuid(record.snapshotId, "Evaluation run snapshot ID"),
    snapshotManifestSha256: sha256(
      record.snapshotManifestSha256,
      "Evaluation run snapshot manifest hash",
    ),
    orderedCaseSetSha256: sha256(
      record.orderedCaseSetSha256,
      "Evaluation run ordered case set hash",
    ),
    configuration: parseConfiguration(
      record.configuration,
      "Evaluation run configuration",
    ),
    scoringPolicyVersion: nonBlank(
      record.scoringPolicyVersion,
      "Evaluation scoring policy version",
    ),
    agentBuild: nonBlank(record.agentBuild, "Evaluation agent build"),
    chatModelIdentity: nonBlank(
      record.chatModelIdentity,
      "Evaluation chat model identity",
    ),
    embeddingModelIdentity: nonBlank(
      record.embeddingModelIdentity,
      "Evaluation embedding model identity",
    ),
    initiatedBy: nonBlank(record.initiatedBy, "Evaluation initiator"),
    state: setValue(record.state, runStates, "Evaluation run state"),
    createdAt: dateTime(record.createdAt, "Evaluation run creation time"),
    startedAt: nullableDateTime(record.startedAt, "Evaluation run start time"),
    completedAt: nullableDateTime(
      record.completedAt,
      "Evaluation run completion time",
    ),
    aggregate: {
      total: nonNegativeInteger(aggregate.total, "Evaluation aggregate total"),
      eligible: nonNegativeInteger(
        aggregate.eligible,
        "Evaluation aggregate eligible",
      ),
      scored: nonNegativeInteger(
        aggregate.scored,
        "Evaluation aggregate scored",
      ),
      failed: nonNegativeInteger(
        aggregate.failed,
        "Evaluation aggregate failed",
      ),
      cancelled: nonNegativeInteger(
        aggregate.cancelled,
        "Evaluation aggregate cancelled",
      ),
      notApplicable: nonNegativeInteger(
        aggregate.notApplicable,
        "Evaluation aggregate not applicable",
      ),
      metrics: parseMetrics(
        array(aggregate.metrics, "Evaluation aggregate metrics"),
      ),
    },
    cases: parseCaseSummaries(array(record.cases, "Evaluation run cases")),
  };
}

export function parseEvaluationRunCase(value: unknown): EvaluationRunCase {
  const record = recordWithOptional(
    value,
    "Evaluation run case",
    [
      "id",
      "datasetCaseId",
      "position",
      "state",
      "attemptCount",
      "runId",
      "corpusId",
      "snapshotId",
      "datasetRevisionId",
      "question",
      "referenceAnswer",
      "expectedOutcome",
      "expectedEvidence",
      "actualEvidence",
      "metrics",
    ],
    [
      "finishedAt",
      "failureCode",
      "answer",
      "graphGroundingState",
      "latencyMilliseconds",
      "inputTokens",
      "outputTokens",
    ],
  );
  const summary = parseCaseSummary({
    id: record.id,
    datasetCaseId: record.datasetCaseId,
    position: record.position,
    state: record.state,
    attemptCount: record.attemptCount,
    finishedAt: record.finishedAt,
    failureCode: record.failureCode,
  });
  const runId = uuid(record.runId, "Evaluation case run ID");
  const corpusId = uuid(record.corpusId, "Evaluation case corpus ID");
  const snapshotId = uuid(record.snapshotId, "Evaluation case snapshot ID");
  const expectedEvidence = array(
    record.expectedEvidence,
    "Evaluation expected evidence",
  ).map((item) => parseEvidence(item, "expected"));
  const actualEvidence = array(
    record.actualEvidence,
    "Evaluation actual evidence",
  ).map((item) => parseEvidence(item, undefined));
  validateEvidenceOwnership(expectedEvidence, corpusId, snapshotId);
  validateEvidenceOwnership(actualEvidence, corpusId, snapshotId);
  validateEvidenceOrdering(expectedEvidence, "Evaluation expected evidence");
  validateEvidenceOrdering(actualEvidence, "Evaluation actual evidence");
  return {
    ...summary,
    runId,
    corpusId,
    snapshotId,
    datasetRevisionId: uuid(
      record.datasetRevisionId,
      "Evaluation case dataset revision ID",
    ),
    question: nonBlank(record.question, "Evaluation case question"),
    referenceAnswer: nonBlank(
      record.referenceAnswer,
      "Evaluation case reference answer",
    ),
    expectedOutcome: expectedOutcome(record.expectedOutcome),
    expectedEvidence,
    actualEvidence,
    answer: optionalString(record.answer, "Evaluation case answer"),
    graphGroundingState: optionalString(
      record.graphGroundingState,
      "Evaluation graph grounding state",
    ),
    latencyMilliseconds: nullableNonNegativeInteger(
      record.latencyMilliseconds,
      "Evaluation latency",
    ),
    inputTokens: nullableNonNegativeInteger(
      record.inputTokens,
      "Evaluation input tokens",
    ),
    outputTokens: nullableNonNegativeInteger(
      record.outputTokens,
      "Evaluation output tokens",
    ),
    metrics: parseMetrics(array(record.metrics, "Evaluation case metrics")),
  };
}

export function parseEvaluationComparison(
  value: unknown,
): EvaluationComparison {
  const record = recordWithOptional(
    value,
    "Evaluation comparison",
    [
      "comparisonState",
      "leftRunId",
      "rightRunId",
      "left",
      "right",
      "differences",
      "metrics",
    ],
    ["totals"],
  );
  const base = {
    comparisonState: comparisonState(record.comparisonState),
    leftRunId: uuid(record.leftRunId, "Evaluation comparison left run ID"),
    rightRunId: uuid(record.rightRunId, "Evaluation comparison right run ID"),
    left: parseExperiment(record.left, "left"),
    right: parseExperiment(record.right, "right"),
  } as const;
  const differences = array(
    record.differences,
    "Evaluation comparison differences",
  ).map((item) => ({
    field: comparisonDifferenceField(
      exactRecord(item, "Evaluation comparison difference", ["field"]).field,
      "Evaluation comparison difference field",
    ),
  }));
  if (base.comparisonState === "non_comparable") {
    if (
      (record.totals !== undefined && record.totals !== null) ||
      !Array.isArray(record.metrics) ||
      record.metrics.length !== 0 ||
      differences.length === 0
    )
      throw new Error(
        "Non-comparable evaluation response must not include totals or metric deltas.",
      );
    return {
      ...base,
      comparisonState: "non_comparable",
      differences,
      totals: null,
      metrics: [],
    };
  }
  if (differences.length !== 0)
    throw new Error(
      "Comparable evaluation response cannot include identity differences.",
    );
  const totals = exactRecord(record.totals, "Evaluation comparison totals", [
    "leftCases",
    "rightCases",
    "pairedCases",
    "leftUnpaired",
    "rightUnpaired",
    "failedOrCancelled",
    "leftFailed",
    "leftCancelled",
    "rightFailed",
    "rightCancelled",
  ]);
  return {
    ...base,
    comparisonState: "comparable",
    differences: [],
    totals: {
      leftCases: nonNegativeInteger(
        totals.leftCases,
        "Evaluation comparison left cases",
      ),
      rightCases: nonNegativeInteger(
        totals.rightCases,
        "Evaluation comparison right cases",
      ),
      pairedCases: nonNegativeInteger(
        totals.pairedCases,
        "Evaluation comparison paired cases",
      ),
      leftUnpaired: nonNegativeInteger(
        totals.leftUnpaired,
        "Evaluation comparison left unpaired",
      ),
      rightUnpaired: nonNegativeInteger(
        totals.rightUnpaired,
        "Evaluation comparison right unpaired",
      ),
      failedOrCancelled: nonNegativeInteger(
        totals.failedOrCancelled,
        "Evaluation comparison failed or cancelled",
      ),
      leftFailed: nonNegativeInteger(
        totals.leftFailed,
        "Evaluation comparison left failed",
      ),
      leftCancelled: nonNegativeInteger(
        totals.leftCancelled,
        "Evaluation comparison left cancelled",
      ),
      rightFailed: nonNegativeInteger(
        totals.rightFailed,
        "Evaluation comparison right failed",
      ),
      rightCancelled: nonNegativeInteger(
        totals.rightCancelled,
        "Evaluation comparison right cancelled",
      ),
    },
    metrics: parseComparisonMetrics(
      array(record.metrics, "Evaluation comparison metrics"),
    ),
  };
}

function parseRevision(value: unknown) {
  const record = exactRecord(value, "Evaluation run dataset revision", [
    "id",
    "contentSha256",
  ]);
  return {
    id: uuid(record.id, "Evaluation run dataset revision ID"),
    contentSha256: sha256(
      record.contentSha256,
      "Evaluation run dataset content hash",
    ),
  };
}
function parseConfiguration(value: unknown, label: string) {
  const record = exactRecord(value, label, ["strategy", "fingerprint"]);
  return {
    strategy: retrievalStrategy(record.strategy, `${label} strategy`),
    fingerprint: sha256(record.fingerprint, `${label} fingerprint`),
  };
}
function parseCaseSummary(value: unknown): EvaluationRunCaseSummary {
  const record = recordWithOptional(
    value,
    "Evaluation run case summary",
    ["id", "datasetCaseId", "position", "state", "attemptCount"],
    ["finishedAt", "failureCode"],
  );
  const state = setValue(record.state, caseStates, "Evaluation case state");
  const failureCode = nullableFailureCode(
    record.failureCode,
    "Evaluation case failure code",
  );
  if (
    (state === "failed" || state === "cancelled") !==
    (failureCode !== null)
  ) {
    throw new Error(
      "Evaluation case failure codes must match failed or cancelled states.",
    );
  }
  return {
    id: uuid(record.id, "Evaluation run case ID"),
    datasetCaseId: uuid(record.datasetCaseId, "Evaluation dataset case ID"),
    position: positiveInteger(record.position, "Evaluation case position"),
    state,
    attemptCount: nonNegativeInteger(
      record.attemptCount,
      "Evaluation case attempt count",
    ),
    finishedAt: nullableDateTime(
      record.finishedAt,
      "Evaluation case completion time",
    ),
    failureCode,
  };
}
function parseCaseSummaries(
  values: readonly unknown[],
): readonly EvaluationRunCaseSummary[] {
  const summaries = values.map((item) => parseCaseSummary(item));
  validateUnique(summaries, (summary) => summary.id, "Evaluation run case IDs");
  validateUnique(
    summaries,
    (summary) => String(summary.position),
    "Evaluation run case positions",
  );
  return summaries;
}
function parseMetrics(values: readonly unknown[]): readonly EvaluationMetric[] {
  const metrics = values.map((item) => parseMetric(item));
  validateUnique(metrics, (metric) => metric.name, "Evaluation metric names");
  return metrics;
}
function parseMetric(value: unknown): EvaluationMetric {
  const record = recordWithOptional(
    value,
    "Evaluation metric",
    ["name", "state", "rationale", "scorerVersion"],
    ["value", "numerator", "denominator"],
  );
  const state = setValue(record.state, metricStates, "Evaluation metric state");
  const parsed = {
    name: metricName(record.name, "Evaluation metric name"),
    state,
    value: nullableFiniteNumber(record.value, "Evaluation metric value"),
    numerator: nullableNonNegativeInteger(
      record.numerator,
      "Evaluation metric numerator",
    ),
    denominator: nullableNonNegativeInteger(
      record.denominator,
      "Evaluation metric denominator",
    ),
    rationale: nonBlank(record.rationale, "Evaluation metric rationale"),
    scorerVersion: nonBlank(
      record.scorerVersion,
      "Evaluation metric scorer version",
    ),
  };
  validateMetricArithmetic(parsed, "Evaluation metric");
  return parsed;
}
function validateMetricArithmetic(
  metric: Pick<
    EvaluationMetric,
    "state" | "value" | "numerator" | "denominator"
  >,
  label: string,
): void {
  if (metric.state !== "scored") {
    if (
      metric.value !== null ||
      metric.numerator !== null ||
      metric.denominator !== null
    ) {
      throw new Error(`${label} without a score cannot include arithmetic.`);
    }
    return;
  }
  if (
    metric.value === null ||
    metric.numerator === null ||
    metric.denominator === null ||
    metric.denominator === 0
  ) {
    throw new Error(`${label} requires value and non-zero arithmetic.`);
  }
  const calculatedValue = metric.numerator / metric.denominator;
  if (!numbersMatch(metric.value, calculatedValue)) {
    throw new Error(`${label} value does not match its arithmetic.`);
  }
}
function parseEvidence(
  value: unknown,
  expectedKind: "expected" | undefined,
): EvaluationEvidence {
  const record = recordWithOptional(
    value,
    "Evaluation evidence",
    [
      "kind",
      "position",
      "corpusId",
      "snapshotId",
      "sourceId",
      "sourceRevisionId",
      "documentId",
      "legalUnitId",
      "canonicalLocator",
      "contentSha256",
    ],
    ["markerPosition", "displayLocator", "startOffset", "endOffset"],
  );
  const kind = evidenceKind(record.kind);
  if (expectedKind !== undefined && kind !== expectedKind)
    throw new Error("Expected evidence must use the expected evidence kind.");
  if (expectedKind === undefined && kind === "expected")
    throw new Error("Actual evidence cannot use the expected evidence kind.");
  const markerPosition = nullablePositiveInteger(
    record.markerPosition,
    "Evaluation evidence marker position",
  );
  const startOffset = nullableNonNegativeInteger(
    record.startOffset,
    "Evaluation evidence start offset",
  );
  const endOffset = nullableNonNegativeInteger(
    record.endOffset,
    "Evaluation evidence end offset",
  );
  if (kind === "cited" && markerPosition === null) {
    throw new Error("Cited evaluation evidence requires a marker position.");
  }
  if (kind !== "cited" && markerPosition !== null) {
    throw new Error(
      "Only cited evaluation evidence can include a marker position.",
    );
  }
  if (kind === "expected" && (startOffset !== null || endOffset !== null)) {
    throw new Error(
      "Expected evaluation evidence cannot include source offsets.",
    );
  }
  if (kind !== "expected") {
    if (
      startOffset === null ||
      endOffset === null ||
      endOffset <= startOffset
    ) {
      throw new Error(
        "Actual evaluation evidence requires an increasing source offset range.",
      );
    }
  }
  return {
    kind,
    position: positiveInteger(record.position, "Evaluation evidence position"),
    markerPosition,
    corpusId: uuid(record.corpusId, "Evaluation evidence corpus ID"),
    snapshotId: uuid(record.snapshotId, "Evaluation evidence snapshot ID"),
    sourceId: uuid(record.sourceId, "Evaluation evidence source ID"),
    sourceRevisionId: uuid(
      record.sourceRevisionId,
      "Evaluation evidence source revision ID",
    ),
    documentId: uuid(record.documentId, "Evaluation evidence document ID"),
    legalUnitId: uuid(record.legalUnitId, "Evaluation evidence legal unit ID"),
    canonicalLocator: nonBlank(
      record.canonicalLocator,
      "Evaluation evidence canonical locator",
    ),
    displayLocator: optionalString(
      record.displayLocator,
      "Evaluation evidence display locator",
    ),
    startOffset,
    endOffset,
    contentSha256: sha256(
      record.contentSha256,
      "Evaluation evidence content hash",
    ),
  };
}
function parseExperiment(
  value: unknown,
  side: string,
): EvaluationComparisonExperiment {
  const record = exactRecord(
    value,
    `Evaluation comparison ${side} experiment`,
    [
      "retrievalStrategy",
      "retrievalConfigurationFingerprint",
      "agentBuild",
      "chatModelIdentity",
      "embeddingModelIdentity",
    ],
  );
  const strategy = string(
    record.retrievalStrategy,
    `Evaluation comparison ${side} strategy`,
  );
  if (strategy !== "vector" && strategy !== "hybrid")
    throw new Error("Evaluation comparison strategy is unsupported.");
  return {
    retrievalStrategy: strategy,
    retrievalConfigurationFingerprint: sha256(
      record.retrievalConfigurationFingerprint,
      "Evaluation comparison configuration fingerprint",
    ),
    agentBuild: nonBlank(
      record.agentBuild,
      "Evaluation comparison agent build",
    ),
    chatModelIdentity: nonBlank(
      record.chatModelIdentity,
      "Evaluation comparison chat model identity",
    ),
    embeddingModelIdentity: nonBlank(
      record.embeddingModelIdentity,
      "Evaluation comparison embedding model identity",
    ),
  };
}
function parseComparisonMetric(value: unknown) {
  const record = recordWithOptional(
    value,
    "Evaluation comparison metric",
    [
      "name",
      "state",
      "pairedCases",
      "leftNumerator",
      "leftDenominator",
      "rightNumerator",
      "rightDenominator",
    ],
    ["leftValue", "rightValue", "delta"],
  );
  const parsed = {
    name: metricName(record.name, "Evaluation comparison metric name"),
    state: setValue(
      record.state,
      metricStates,
      "Evaluation comparison metric state",
    ),
    pairedCases: nonNegativeInteger(
      record.pairedCases,
      "Evaluation comparison paired cases",
    ),
    leftNumerator: nonNegativeInteger(
      record.leftNumerator,
      "Evaluation comparison left numerator",
    ),
    leftDenominator: nonNegativeInteger(
      record.leftDenominator,
      "Evaluation comparison left denominator",
    ),
    rightNumerator: nonNegativeInteger(
      record.rightNumerator,
      "Evaluation comparison right numerator",
    ),
    rightDenominator: nonNegativeInteger(
      record.rightDenominator,
      "Evaluation comparison right denominator",
    ),
    leftValue: nullableFiniteNumber(
      record.leftValue,
      "Evaluation comparison left value",
    ),
    rightValue: nullableFiniteNumber(
      record.rightValue,
      "Evaluation comparison right value",
    ),
    delta: nullableFiniteNumber(record.delta, "Evaluation comparison delta"),
  };
  validateComparisonMetricArithmetic(parsed);
  return parsed;
}
function parseComparisonMetrics(values: readonly unknown[]) {
  const metrics = values.map((item) => parseComparisonMetric(item));
  validateUnique(
    metrics,
    (metric) => metric.name,
    "Evaluation comparison metric names",
  );
  return metrics;
}
function validateComparisonMetricArithmetic(
  metric: ReturnType<typeof parseComparisonMetric>,
): void {
  const values = [metric.leftValue, metric.rightValue, metric.delta];
  if (metric.state !== "scored") {
    if (
      metric.pairedCases !== 0 ||
      metric.leftNumerator !== 0 ||
      metric.leftDenominator !== 0 ||
      metric.rightNumerator !== 0 ||
      metric.rightDenominator !== 0 ||
      values.some((value) => value !== null)
    ) {
      throw new Error(
        "Unscored evaluation comparison metrics cannot include paired arithmetic.",
      );
    }
    return;
  }
  if (
    metric.pairedCases === 0 ||
    metric.leftDenominator === 0 ||
    metric.rightDenominator === 0 ||
    metric.leftValue === null ||
    metric.rightValue === null ||
    metric.delta === null
  ) {
    throw new Error(
      "Scored evaluation comparison metrics require paired arithmetic.",
    );
  }
  const leftValue = metric.leftNumerator / metric.leftDenominator;
  const rightValue = metric.rightNumerator / metric.rightDenominator;
  if (
    !numbersMatch(metric.leftValue, leftValue) ||
    !numbersMatch(metric.rightValue, rightValue) ||
    !numbersMatch(metric.delta, rightValue - leftValue)
  ) {
    throw new Error(
      "Evaluation comparison metric values do not match their arithmetic.",
    );
  }
}
function validateEvidenceOwnership(
  evidence: readonly EvaluationEvidence[],
  corpusId: string,
  snapshotId: string,
): void {
  for (const item of evidence) {
    if (item.corpusId !== corpusId || item.snapshotId !== snapshotId) {
      throw new Error(
        "Evaluation evidence must belong to the case run corpus and snapshot.",
      );
    }
  }
}
function validateEvidenceOrdering(
  evidence: readonly EvaluationEvidence[],
  label: string,
): void {
  let previousMarkerPosition = 0;
  for (const [index, item] of evidence.entries()) {
    if (item.position !== index + 1) {
      throw new Error(`${label} positions must be contiguous.`);
    }
    if (item.kind === "cited") {
      if (
        item.markerPosition === null ||
        item.markerPosition <= previousMarkerPosition
      ) {
        throw new Error(`${label} citation markers must be strictly ordered.`);
      }
      previousMarkerPosition = item.markerPosition;
    }
  }
}
function validateUnique<T>(
  values: readonly T[],
  identifier: (value: T) => string,
  label: string,
): void {
  const seen = new Set<string>();
  for (const value of values) {
    const id = identifier(value);
    if (seen.has(id)) throw new Error(`${label} must be unique.`);
    seen.add(id);
  }
}
function numbersMatch(left: number, right: number): boolean {
  return (
    Math.abs(left - right) <=
    Number.EPSILON * Math.max(1, Math.abs(left), Math.abs(right))
  );
}
function exactRecord(
  value: unknown,
  label: string,
  fields: readonly string[],
): Record<string, unknown> {
  if (value === null || typeof value !== "object" || Array.isArray(value))
    throw new TypeError(`${label} must be an object.`);
  const record = value as Record<string, unknown>;
  for (const key of Object.keys(record))
    if (!fields.includes(key))
      throw new Error(`${label} contains unsupported field ${key}.`);
  for (const field of fields)
    if (!(field in record))
      throw new Error(`${label} is missing required field ${field}.`);
  return record;
}
function recordWithOptional(
  value: unknown,
  label: string,
  required: readonly string[],
  optional: readonly string[],
): Record<string, unknown> {
  const fields = [...required, ...optional];
  if (value === null || typeof value !== "object" || Array.isArray(value))
    throw new TypeError(`${label} must be an object.`);
  const record = value as Record<string, unknown>;
  for (const key of Object.keys(record))
    if (!fields.includes(key))
      throw new Error(`${label} contains unsupported field ${key}.`);
  for (const field of required)
    if (!(field in record))
      throw new Error(`${label} is missing required field ${field}.`);
  return record;
}
function array(value: unknown, label: string): readonly unknown[] {
  if (!Array.isArray(value)) throw new TypeError(`${label} must be an array.`);
  return value;
}
function string(value: unknown, label: string): string {
  if (typeof value !== "string")
    throw new TypeError(`${label} must be a string.`);
  return value;
}
function nonBlank(value: unknown, label: string): string {
  const result = string(value, label);
  if (result.trim() === "") throw new Error(`${label} must not be blank.`);
  return result;
}
function nullableFailureCode(
  value: unknown,
  label: string,
): EvaluationFailureCode | null {
  if (value === undefined || value === null) return null;
  const result = nonBlank(value, label) as EvaluationFailureCode;
  if (!failureCodes.has(result)) throw new Error(`${label} is unsupported.`);
  return result;
}
function optionalString(value: unknown, label: string): string {
  return value === undefined ? "" : string(value, label);
}
function uuid(value: unknown, label: string): string {
  const result = string(value, label);
  if (!uuidPattern.test(result)) throw new Error(`${label} must be a UUID.`);
  return result;
}
function sha256(value: unknown, label: string): string {
  const result = string(value, label);
  if (!sha256Pattern.test(result))
    throw new Error(`${label} must be a SHA-256 hash.`);
  return result;
}
function dateTime(value: unknown, label: string): string {
  const result = nonBlank(value, label);
  if (Number.isNaN(Date.parse(result)))
    throw new Error(`${label} must be an ISO date-time.`);
  return result;
}
function nullableDateTime(value: unknown, label: string): string | null {
  return value === undefined || value === null ? null : dateTime(value, label);
}
function integer(value: unknown, label: string): number {
  if (typeof value !== "number" || !Number.isSafeInteger(value))
    throw new TypeError(`${label} must be a safe integer.`);
  return value;
}
function nonNegativeInteger(value: unknown, label: string): number {
  const result = integer(value, label);
  if (result < 0) throw new Error(`${label} must not be negative.`);
  return result;
}
function positiveInteger(value: unknown, label: string): number {
  const result = nonNegativeInteger(value, label);
  if (result === 0) throw new Error(`${label} must be positive.`);
  return result;
}
function nullableNonNegativeInteger(
  value: unknown,
  label: string,
): number | null {
  return value === undefined || value === null
    ? null
    : nonNegativeInteger(value, label);
}
function nullablePositiveInteger(value: unknown, label: string): number | null {
  return value === undefined || value === null
    ? null
    : positiveInteger(value, label);
}
function nullableFiniteNumber(value: unknown, label: string): number | null {
  if (value === undefined || value === null) return null;
  if (typeof value !== "number" || !Number.isFinite(value))
    throw new TypeError(`${label} must be a finite number.`);
  return value;
}
function setValue<T extends string>(
  value: unknown,
  values: ReadonlySet<T>,
  label: string,
): T {
  const result = string(value, label) as T;
  if (!values.has(result)) throw new Error(`${label} is unsupported.`);
  return result;
}
function metricName(value: unknown, label: string): EvaluationMetricName {
  const result = nonBlank(value, label) as EvaluationMetricName;
  if (!metricNames.has(result)) throw new Error(`${label} is unsupported.`);
  return result;
}
function comparisonDifferenceField(
  value: unknown,
  label: string,
): EvaluationComparisonDifferenceField {
  const result = nonBlank(value, label) as EvaluationComparisonDifferenceField;
  if (!comparisonDifferenceFields.has(result)) {
    throw new Error(`${label} is unsupported.`);
  }
  return result;
}
function expectedOutcome(value: unknown): EvaluationExpectedOutcome {
  const result = string(value, "Evaluation expected outcome");
  if (result !== "answer" && result !== "abstain")
    throw new Error("Evaluation expected outcome is unsupported.");
  return result;
}
function evidenceKind(value: unknown): EvaluationEvidenceKind {
  const result = string(value, "Evaluation evidence kind");
  if (result !== "expected" && result !== "retrieved" && result !== "cited")
    throw new Error("Evaluation evidence kind is unsupported.");
  return result;
}
function comparisonState(value: unknown): EvaluationComparisonState {
  const result = string(value, "Evaluation comparison state");
  if (result !== "comparable" && result !== "non_comparable")
    throw new Error("Evaluation comparison state is unsupported.");
  return result;
}
function retrievalStrategy(value: unknown, label: string): "vector" | "hybrid" {
  const result = string(value, label);
  if (result !== "vector" && result !== "hybrid") {
    throw new Error(`${label} is unsupported.`);
  }
  return result;
}
