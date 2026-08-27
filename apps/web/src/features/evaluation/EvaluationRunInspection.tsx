import { useEffect, useState } from "react";
import { useTranslation } from "react-i18next";

import type {
  EvaluationComparison,
  EvaluationEvidence,
  EvaluationMetric,
  EvaluationRunCaseState,
  EvaluationRunCase,
  EvaluationRunState,
  EvaluationRunSummary,
} from "../../api/evaluationRun";
import type { EvaluationRunClient } from "../../api/evaluationRunClient";
import "./evaluation.css";

type RunState =
  | { readonly status: "loading"; readonly runId: string }
  | {
      readonly status: "ready";
      readonly runId: string;
      readonly run: EvaluationRunSummary;
    }
  | { readonly status: "failed"; readonly runId: string };

type CaseState =
  | { readonly status: "idle" }
  | {
      readonly status: "loading";
      readonly runId: string;
      readonly caseId: string;
    }
  | {
      readonly status: "ready";
      readonly runId: string;
      readonly caseId: string;
      readonly result: EvaluationRunCase;
    }
  | {
      readonly status: "failed";
      readonly runId: string;
      readonly caseId: string;
    };

type ComparisonState =
  | { readonly status: "loading" }
  | { readonly status: "ready"; readonly comparison: EvaluationComparison }
  | { readonly status: "failed" };

interface EvaluationRunInspectionProps {
  readonly client: EvaluationRunClient;
  readonly runId: string;
}

interface EvaluationComparisonViewProps {
  readonly client: EvaluationRunClient;
  readonly leftRunId: string;
  readonly rightRunId: string;
}

export function EvaluationRunInspection({
  client,
  runId,
}: EvaluationRunInspectionProps) {
  return (
    <EvaluationRunInspectionContent key={runId} client={client} runId={runId} />
  );
}

function EvaluationRunInspectionContent({
  client,
  runId,
}: EvaluationRunInspectionProps) {
  const { t } = useTranslation();
  const [selectedCase, setSelectedCase] = useState<{
    readonly runId: string;
    readonly caseId: string;
  } | null>(null);
  const runState = useEvaluationRun(client, runId);
  const showingRun =
    runState.status === "ready" &&
    runState.runId === runId &&
    runState.run.id === runId;
  const activeCaseId =
    showingRun &&
    selectedCase?.runId === runId &&
    runState.run.cases.some((item) => item.id === selectedCase.caseId)
      ? selectedCase.caseId
      : null;
  const caseState = useEvaluationCase(client, runId, activeCaseId);

  return (
    <section
      className="evaluation-inspection"
      aria-labelledby="evaluation-run-heading"
    >
      <header className="evaluation-inspection__header">
        <p className="kicker">{t("evaluation.run.kicker")}</p>
        <h1 id="evaluation-run-heading">{t("evaluation.run.title")}</h1>
        <p>{t("evaluation.run.introduction")}</p>
      </header>
      <p className="evaluation-readiness__notice" role="note">
        {t("evaluation.notice")}
      </p>
      {!showingRun && runState.status !== "failed" ? (
        <output>{t("evaluation.run.loading")}</output>
      ) : null}
      {runState.status === "failed" ? (
        <p role="alert">{t("evaluation.run.loadFailed")}</p>
      ) : null}
      {showingRun ? (
        <RunDetails
          caseState={caseState}
          onCaseSelect={(caseId) => setSelectedCase({ runId, caseId })}
          run={runState.run}
          selectedCaseId={activeCaseId}
        />
      ) : null}
    </section>
  );
}

function useEvaluationRun(
  client: EvaluationRunClient,
  runId: string,
): RunState {
  const [state, setState] = useState<RunState>({ status: "loading", runId });

  useEffect(() => {
    const controller = new AbortController();
    void client
      .getRun(runId, controller.signal)
      .then((run) => {
        if (!controller.signal.aborted) {
          setState({ status: "ready", runId, run });
        }
      })
      .catch(() => {
        if (!controller.signal.aborted) {
          setState({ status: "failed", runId });
        }
      });
    return () => controller.abort();
  }, [client, runId]);

  return state.runId === runId ? state : { status: "loading", runId };
}

function useEvaluationCase(
  client: EvaluationRunClient,
  runId: string,
  caseId: string | null,
): CaseState {
  const [state, setState] = useState<CaseState>({ status: "idle" });

  useEffect(() => {
    if (caseId === null) return undefined;
    const controller = new AbortController();
    void client
      .getCase(runId, caseId, controller.signal)
      .then((result) => {
        if (!controller.signal.aborted) {
          setState({ status: "ready", runId, caseId, result });
        }
      })
      .catch(() => {
        if (!controller.signal.aborted) {
          setState({ status: "failed", runId, caseId });
        }
      });
    return () => controller.abort();
  }, [caseId, client, runId]);

  if (caseId === null) return { status: "idle" };
  if (
    state.status === "idle" ||
    state.runId !== runId ||
    state.caseId !== caseId
  ) {
    return { status: "loading", runId, caseId };
  }
  return state;
}

export function EvaluationComparisonView({
  client,
  leftRunId,
  rightRunId,
}: EvaluationComparisonViewProps) {
  return (
    <EvaluationComparisonViewContent
      key={`${leftRunId}:${rightRunId}`}
      client={client}
      leftRunId={leftRunId}
      rightRunId={rightRunId}
    />
  );
}

function EvaluationComparisonViewContent({
  client,
  leftRunId,
  rightRunId,
}: EvaluationComparisonViewProps) {
  const { t } = useTranslation();
  const [state, setState] = useState<ComparisonState>({ status: "loading" });

  useEffect(() => {
    const controller = new AbortController();
    void client
      .compareRuns(leftRunId, rightRunId, controller.signal)
      .then((comparison) => {
        if (!controller.signal.aborted) {
          setState({ status: "ready", comparison });
        }
      })
      .catch(() => {
        if (!controller.signal.aborted) setState({ status: "failed" });
      });
    return () => controller.abort();
  }, [client, leftRunId, rightRunId]);

  const showingComparison =
    state.status === "ready" &&
    state.comparison.leftRunId === leftRunId &&
    state.comparison.rightRunId === rightRunId;

  return (
    <section
      className="evaluation-inspection"
      aria-labelledby="evaluation-comparison-heading"
    >
      <header className="evaluation-inspection__header">
        <p className="kicker">{t("evaluation.comparison.kicker")}</p>
        <h1 id="evaluation-comparison-heading">
          {t("evaluation.comparison.title")}
        </h1>
        <p>{t("evaluation.comparison.introduction")}</p>
      </header>
      <p className="evaluation-readiness__notice" role="note">
        {t("evaluation.notice")}
      </p>
      {!showingComparison && state.status !== "failed" ? (
        <output>{t("evaluation.comparison.loading")}</output>
      ) : null}
      {state.status === "failed" ? (
        <p role="alert">{t("evaluation.comparison.loadFailed")}</p>
      ) : null}
      {showingComparison ? (
        <ComparisonDetails comparison={state.comparison} />
      ) : null}
    </section>
  );
}

function RunDetails({
  run,
  selectedCaseId,
  onCaseSelect,
  caseState,
}: {
  readonly run: EvaluationRunSummary;
  readonly selectedCaseId: string | null;
  readonly onCaseSelect: (caseId: string) => void;
  readonly caseState: CaseState;
}) {
  const { t } = useTranslation();
  const caseMatchesSelection =
    caseState.status !== "idle" &&
    caseState.runId === run.id &&
    caseState.caseId === selectedCaseId;
  return (
    <>
      <section
        className="evaluation-inspection__panel"
        aria-labelledby="evaluation-run-identity"
      >
        <h2 id="evaluation-run-identity">
          {t("evaluation.run.immutableIdentity")}
        </h2>
        <dl className="evaluation-readiness__identity">
          <IdentityRow label={t("evaluation.run.id")} value={run.id} />
          <IdentityRow
            label={t("evaluation.run.datasetRevisionId")}
            value={run.datasetRevision.id}
          />
          <IdentityRow
            label={t("evaluation.run.datasetContentSha256")}
            value={run.datasetRevision.contentSha256}
          />
          <IdentityRow
            label={t("evaluation.run.corpusId")}
            value={run.corpusId}
          />
          <IdentityRow
            label={t("evaluation.run.snapshotId")}
            value={run.snapshotId}
          />
          <IdentityRow
            label={t("evaluation.run.snapshotManifestSha256")}
            value={run.snapshotManifestSha256}
          />
          <IdentityRow
            label={t("evaluation.run.orderedCaseSetSha256")}
            value={run.orderedCaseSetSha256}
          />
          <IdentityRow
            label={t("evaluation.run.configuration")}
            value={`${formatRetrievalStrategy(t, run.configuration.strategy)} / ${run.configuration.fingerprint}`}
          />
          <IdentityRow
            label={t("evaluation.run.scoringPolicy")}
            value={run.scoringPolicyVersion}
          />
          <IdentityRow
            label={t("evaluation.run.initiatedBy")}
            value={run.initiatedBy}
          />
          <IdentityRow
            label={t("evaluation.run.agentBuild")}
            value={run.agentBuild}
          />
          <IdentityRow
            label={t("evaluation.run.chatModel")}
            value={run.chatModelIdentity}
          />
          <IdentityRow
            label={t("evaluation.run.embeddingModel")}
            value={run.embeddingModelIdentity}
          />
          <IdentityRow
            label={t("evaluation.run.state")}
            value={formatRunState(t, run.state)}
          />
          <IdentityRow
            label={t("evaluation.run.createdAt")}
            value={run.createdAt}
          />
          <IdentityRow
            label={t("evaluation.run.startedAt")}
            value={run.startedAt ?? t("evaluation.run.notRecorded")}
          />
          <IdentityRow
            label={t("evaluation.run.completedAt")}
            value={run.completedAt ?? t("evaluation.run.notRecorded")}
          />
        </dl>
      </section>
      <section
        className="evaluation-inspection__panel"
        aria-labelledby="evaluation-run-aggregate"
      >
        <h2 id="evaluation-run-aggregate">{t("evaluation.run.aggregate")}</h2>
        <dl className="evaluation-inspection__counts">
          <IdentityRow
            label={t("evaluation.run.total")}
            value={String(run.aggregate.total)}
          />
          <IdentityRow
            label={t("evaluation.run.eligible")}
            value={String(run.aggregate.eligible)}
          />
          <IdentityRow
            label={t("evaluation.run.scored")}
            value={String(run.aggregate.scored)}
          />
          <IdentityRow
            label={t("evaluation.run.failed")}
            value={String(run.aggregate.failed)}
          />
          <IdentityRow
            label={t("evaluation.run.cancelled")}
            value={String(run.aggregate.cancelled)}
          />
          <IdentityRow
            label={t("evaluation.run.notApplicable")}
            value={String(run.aggregate.notApplicable)}
          />
        </dl>
        <MetricList metrics={run.aggregate.metrics} />
      </section>
      <section
        className="evaluation-inspection__panel"
        aria-labelledby="evaluation-run-cases"
      >
        <h2 id="evaluation-run-cases">{t("evaluation.run.cases")}</h2>
        <ol className="evaluation-inspection__cases">
          {run.cases.map((item) => (
            <li key={item.id}>
              <button
                aria-pressed={selectedCaseId === item.id}
                type="button"
                onClick={() => onCaseSelect(item.id)}
              >
                {t("evaluation.run.caseLabel", { position: item.position })}:{" "}
                {formatCaseState(t, item.state)}
              </button>
              {item.failureCode !== null ? (
                <span>
                  {t("evaluation.run.failure", {
                    code: formatFailureCode(t, item.failureCode),
                  })}
                </span>
              ) : null}
            </li>
          ))}
        </ol>
        {caseMatchesSelection && caseState.status === "loading" ? (
          <output>{t("evaluation.run.caseLoading")}</output>
        ) : null}
        {caseMatchesSelection && caseState.status === "failed" ? (
          <p role="alert">{t("evaluation.run.caseLoadFailed")}</p>
        ) : null}
        {caseMatchesSelection && caseState.status === "ready" ? (
          <CaseDetails result={caseState.result} />
        ) : null}
      </section>
    </>
  );
}

function CaseDetails({ result }: { readonly result: EvaluationRunCase }) {
  const { t } = useTranslation();
  return (
    <section
      className="evaluation-inspection__case"
      aria-labelledby="evaluation-case-evidence"
    >
      <h3 id="evaluation-case-evidence">
        {t("evaluation.run.expectedActual")}
      </h3>
      <dl className="evaluation-readiness__identity">
        <IdentityRow
          label={t("evaluation.run.datasetCaseId")}
          value={result.datasetCaseId}
        />
        <IdentityRow
          label={t("evaluation.run.caseState")}
          value={formatCaseState(t, result.state)}
        />
        <IdentityRow
          label={t("evaluation.run.expectedOutcome")}
          value={formatExpectedOutcome(t, result.expectedOutcome)}
        />
        <IdentityRow
          label={t("evaluation.run.latency")}
          value={
            result.latencyMilliseconds === null
              ? t("evaluation.run.notRecorded")
              : String(result.latencyMilliseconds)
          }
        />
        <IdentityRow
          label={t("evaluation.run.inputTokens")}
          value={
            result.inputTokens === null
              ? t("evaluation.run.notRecorded")
              : String(result.inputTokens)
          }
        />
        <IdentityRow
          label={t("evaluation.run.outputTokens")}
          value={
            result.outputTokens === null
              ? t("evaluation.run.notRecorded")
              : String(result.outputTokens)
          }
        />
      </dl>
      <h4>{t("evaluation.run.question")}</h4>
      <p>{result.question}</p>
      <h4>{t("evaluation.run.referenceAnswer")}</h4>
      <p>{result.referenceAnswer}</p>
      <h4>{t("evaluation.run.generatedAnswer")}</h4>
      <p>{result.answer || t("evaluation.run.noAnswer")}</p>
      <EvidenceList
        evidence={result.expectedEvidence}
        heading={t("evaluation.run.expectedEvidence")}
      />
      <EvidenceList
        evidence={result.actualEvidence}
        heading={t("evaluation.run.actualEvidence")}
      />
      <MetricList metrics={result.metrics} />
    </section>
  );
}

function ComparisonDetails({
  comparison,
}: {
  readonly comparison: EvaluationComparison;
}) {
  const { t } = useTranslation();
  return (
    <section
      className="evaluation-inspection__panel"
      aria-labelledby="evaluation-comparison-result"
    >
      <h2 id="evaluation-comparison-result">
        {t("evaluation.comparison.result")}
      </h2>
      <ExperimentIdentity
        experiment={comparison.left}
        heading={t("evaluation.comparison.left")}
      />
      <ExperimentIdentity
        experiment={comparison.right}
        heading={t("evaluation.comparison.right")}
      />
      {comparison.comparisonState === "non_comparable" ? (
        <div role="alert">
          <p>{t("evaluation.comparison.nonComparable")}</p>
          <ul>
            {comparison.differences.map((difference) => (
              <li key={difference.field}>
                {formatComparisonDifference(t, difference.field)}
              </li>
            ))}
          </ul>
        </div>
      ) : (
        <>
          <dl className="evaluation-inspection__counts">
            <IdentityRow
              label={t("evaluation.comparison.pairedCases")}
              value={String(comparison.totals.pairedCases)}
            />
            <IdentityRow
              label={t("evaluation.comparison.failedOrCancelled")}
              value={String(comparison.totals.failedOrCancelled)}
            />
            <IdentityRow
              label={t("evaluation.comparison.leftUnpaired")}
              value={String(comparison.totals.leftUnpaired)}
            />
            <IdentityRow
              label={t("evaluation.comparison.rightUnpaired")}
              value={String(comparison.totals.rightUnpaired)}
            />
          </dl>
          <table>
            <caption>{t("evaluation.comparison.metrics")}</caption>
            <thead>
              <tr>
                <th scope="col">{t("evaluation.comparison.metric")}</th>
                <th scope="col">{t("evaluation.comparison.left")}</th>
                <th scope="col">{t("evaluation.comparison.right")}</th>
                <th scope="col">{t("evaluation.comparison.delta")}</th>
              </tr>
            </thead>
            <tbody>
              {comparison.metrics.map((metric) => (
                <tr key={metric.name}>
                  <th scope="row">{formatMetricName(t, metric.name)}</th>
                  <td>
                    {formatMetricValue(
                      metric.leftValue,
                      metric.leftNumerator,
                      metric.leftDenominator,
                    )}
                  </td>
                  <td>
                    {formatMetricValue(
                      metric.rightValue,
                      metric.rightNumerator,
                      metric.rightDenominator,
                    )}
                  </td>
                  <td>
                    {metric.delta === null
                      ? t("evaluation.run.notRecorded")
                      : String(metric.delta)}
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </>
      )}
    </section>
  );
}

function ExperimentIdentity({
  heading,
  experiment,
}: {
  readonly heading: string;
  readonly experiment: EvaluationComparison["left"];
}) {
  const { t } = useTranslation();
  return (
    <section className="evaluation-inspection__experiment">
      <h3>{heading}</h3>
      <dl className="evaluation-readiness__identity">
        <IdentityRow
          label={t("evaluation.comparison.strategy")}
          value={formatRetrievalStrategy(t, experiment.retrievalStrategy)}
        />
        <IdentityRow
          label={t("evaluation.comparison.configurationFingerprint")}
          value={experiment.retrievalConfigurationFingerprint}
        />
        <IdentityRow
          label={t("evaluation.comparison.agentBuild")}
          value={experiment.agentBuild}
        />
        <IdentityRow
          label={t("evaluation.comparison.chatModel")}
          value={experiment.chatModelIdentity}
        />
        <IdentityRow
          label={t("evaluation.comparison.embeddingModel")}
          value={experiment.embeddingModelIdentity}
        />
      </dl>
    </section>
  );
}

function EvidenceList({
  evidence,
  heading,
}: {
  readonly evidence: readonly EvaluationEvidence[];
  readonly heading: string;
}) {
  const { t } = useTranslation();
  return (
    <section>
      <h4>{heading}</h4>
      {evidence.length === 0 ? (
        <p>{t("evaluation.run.noEvidence")}</p>
      ) : (
        <ol className="evaluation-inspection__evidence">
          {evidence.map((item) => (
            <li key={`${item.kind}-${String(item.position)}`}>
              <strong>{formatEvidenceKind(t, item.kind)}</strong>
              <span>{item.displayLocator || item.canonicalLocator}</span>
              <span>{item.contentSha256}</span>
            </li>
          ))}
        </ol>
      )}
    </section>
  );
}

function MetricList({
  metrics,
}: {
  readonly metrics: readonly EvaluationMetric[];
}) {
  const { t } = useTranslation();
  if (metrics.length === 0) return <p>{t("evaluation.run.noMetrics")}</p>;
  return (
    <ul className="evaluation-inspection__metrics">
      {metrics.map((metric) => (
        <li key={metric.name}>
          <strong>{formatMetricName(t, metric.name)}</strong>
          <span>
            {formatMetricValue(
              metric.value,
              metric.numerator,
              metric.denominator,
            )}
          </span>
          <span>{metric.rationale}</span>
        </li>
      ))}
    </ul>
  );
}

function formatMetricValue(
  value: number | null,
  numerator: number | null,
  denominator: number | null,
): string {
  if (value === null || numerator === null || denominator === null) return "--";
  return `${String(value)} (${String(numerator)}/${String(denominator)})`;
}

type Translate = (key: string) => string;

function formatRunState(
  translate: Translate,
  state: EvaluationRunState,
): string {
  return translate(`evaluation.run.states.${state}`);
}

function formatCaseState(
  translate: Translate,
  state: EvaluationRunCaseState,
): string {
  return translate(`evaluation.run.caseStates.${state}`);
}

function formatExpectedOutcome(
  translate: Translate,
  outcome: EvaluationRunCase["expectedOutcome"],
): string {
  return translate(`evaluation.run.expectedOutcomes.${outcome}`);
}

function formatEvidenceKind(
  translate: Translate,
  kind: EvaluationEvidence["kind"],
): string {
  return translate(`evaluation.run.evidenceKinds.${kind}`);
}

function formatFailureCode(
  translate: Translate,
  code: NonNullable<EvaluationRunCase["failureCode"]>,
): string {
  return translate(`evaluation.run.failureCodes.${code}`);
}

function formatMetricName(
  translate: Translate,
  name: EvaluationMetric["name"],
): string {
  return translate(`evaluation.run.metricNames.${name}`);
}

function formatRetrievalStrategy(
  translate: Translate,
  strategy: EvaluationRunSummary["configuration"]["strategy"],
): string {
  return translate(`evaluation.run.strategy.${strategy}`);
}

function formatComparisonDifference(
  translate: Translate,
  field: Extract<
    EvaluationComparison,
    { readonly comparisonState: "non_comparable" }
  >["differences"][number]["field"],
): string {
  return translate(`evaluation.comparison.differenceFields.${field}`);
}

function IdentityRow({
  label,
  value,
}: {
  readonly label: string;
  readonly value: string;
}) {
  return (
    <div>
      <dt>{label}</dt>
      <dd>{value}</dd>
    </div>
  );
}
