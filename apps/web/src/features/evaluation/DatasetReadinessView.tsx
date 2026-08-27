import { useEffect, useRef, useState } from "react";
import { useTranslation } from "react-i18next";

import type {
  EvaluationDatasetCatalogEntry,
  EvaluationDatasetDetail,
  EvaluationDatasetPreflightRequest,
  EvaluationDatasetPreflightResponse,
  EvaluationMissingRequirement,
} from "../../api/evaluationCatalog";
import {
  EvaluationCatalogRequestError,
  type EvaluationCatalogClient,
} from "../../api/evaluationCatalogClient";
import "./evaluation.css";

export interface EvaluationSnapshotOption {
  readonly corpusId: string;
  readonly snapshotId: string;
  readonly snapshotManifestSha256: string;
  readonly label: string;
}

interface DatasetReadinessViewProps {
  readonly client: EvaluationCatalogClient;
  readonly snapshotOptions: readonly EvaluationSnapshotOption[];
  readonly onStartRun?:
    ((selection: EvaluationDatasetPreflightResponse) => void) | undefined;
}

type CatalogState =
  | { readonly status: "loading" }
  | {
      readonly status: "ready";
      readonly datasets: readonly EvaluationDatasetCatalogEntry[];
    }
  | { readonly status: "failed" };

type DetailState =
  | { readonly status: "idle" }
  | { readonly status: "loading" }
  | { readonly status: "ready"; readonly detail: EvaluationDatasetDetail }
  | { readonly status: "failed" };

type PreflightState =
  | { readonly status: "idle" }
  | { readonly status: "checking" }
  | {
      readonly status: "compatible";
      readonly selection: EvaluationDatasetPreflightResponse;
    }
  | {
      readonly status: "incompatible";
      readonly requirements: readonly EvaluationMissingRequirement[];
    }
  | { readonly status: "failed" };

export function DatasetReadinessView({
  client,
  snapshotOptions,
  onStartRun,
}: DatasetReadinessViewProps) {
  const catalogState = useEvaluationDatasetCatalog(client);
  const datasetDetail = useSelectedDatasetDetail(client);
  const preflight = useEvaluationDatasetPreflight(client);
  const [snapshotSelection, setSnapshotSelection] = useState("");
  const selectDatasetRevision = (revisionId: string): void => {
    preflight.reset();
    datasetDetail.select(revisionId);
    setSnapshotSelection("");
  };

  const selectSnapshot = (selectedSnapshotValue: string): void => {
    preflight.reset();
    setSnapshotSelection(selectedSnapshotValue);
  };

  return (
    <section
      className="evaluation-readiness"
      aria-labelledby="evaluation-heading"
    >
      <DatasetReadinessHeader />
      <DatasetCatalogControl
        state={catalogState}
        selectedRevisionId={datasetDetail.revisionId}
        onSelect={selectDatasetRevision}
      />
      <DatasetDetailContent state={datasetDetail.state} />
      {datasetDetail.state.status === "ready" ? (
        <DatasetSnapshotSelection
          detail={datasetDetail.state.detail}
          snapshotOptions={snapshotOptions}
          selection={snapshotSelection}
          preflightState={preflight.state}
          onSelect={selectSnapshot}
          onCheck={preflight.check}
          onStartRun={onStartRun}
        />
      ) : null}
    </section>
  );
}

function useEvaluationDatasetCatalog(
  client: EvaluationCatalogClient,
): CatalogState {
  const [state, setState] = useState<CatalogState>({ status: "loading" });

  useEffect(() => {
    const controller = new AbortController();
    void client
      .listDatasets(controller.signal)
      .then((datasets) => {
        if (!controller.signal.aborted) setState({ status: "ready", datasets });
      })
      .catch(() => {
        if (!controller.signal.aborted) setState({ status: "failed" });
      });
    return () => controller.abort();
  }, [client]);

  return state;
}

function useSelectedDatasetDetail(client: EvaluationCatalogClient) {
  const [revisionId, setRevisionId] = useState("");
  const [state, setState] = useState<DetailState>({ status: "idle" });

  useEffect(() => {
    if (revisionId === "") return undefined;

    const controller = new AbortController();
    void client
      .getDataset(revisionId, controller.signal)
      .then((detail) => {
        if (!controller.signal.aborted) setState({ status: "ready", detail });
      })
      .catch(() => {
        if (!controller.signal.aborted) setState({ status: "failed" });
      });
    return () => controller.abort();
  }, [client, revisionId]);

  const select = (selectedRevisionId: string): void => {
    setRevisionId(selectedRevisionId);
    setState(
      selectedRevisionId === "" ? { status: "idle" } : { status: "loading" },
    );
  };

  return { revisionId, state, select };
}

function useEvaluationDatasetPreflight(client: EvaluationCatalogClient) {
  const [state, setState] = useState<PreflightState>({ status: "idle" });
  const controllerReference = useRef<AbortController | null>(null);

  useEffect(() => () => controllerReference.current?.abort(), []);

  const reset = (): void => {
    controllerReference.current?.abort();
    setState({ status: "idle" });
  };

  const check = (request: EvaluationDatasetPreflightRequest): void => {
    controllerReference.current?.abort();
    const controller = new AbortController();
    controllerReference.current = controller;
    setState({ status: "checking" });
    void client
      .preflightDataset(request, controller.signal)
      .then((selection) => {
        if (!controller.signal.aborted)
          setState({ status: "compatible", selection });
      })
      .catch((error: unknown) => {
        if (controller.signal.aborted) return;
        setState(preflightFailureState(error));
      });
  };

  return { state, reset, check };
}

function DatasetReadinessHeader() {
  const { t } = useTranslation();

  return (
    <>
      <header className="evaluation-readiness__header">
        <p className="kicker">{t("evaluation.kicker")}</p>
        <h1 id="evaluation-heading">{t("evaluation.title")}</h1>
        <p>{t("evaluation.introduction")}</p>
      </header>
      <p className="evaluation-readiness__notice" role="note">
        {t("evaluation.notice")}
      </p>
    </>
  );
}

function DatasetCatalogControl({
  state,
  selectedRevisionId,
  onSelect,
}: {
  readonly state: CatalogState;
  readonly selectedRevisionId: string;
  readonly onSelect: (revisionId: string) => void;
}) {
  const { t } = useTranslation();
  if (state.status === "loading")
    return <output>{t("evaluation.loading")}</output>;
  if (state.status === "failed")
    return <p role="alert">{t("evaluation.catalogFailed")}</p>;

  return (
    <label className="evaluation-readiness__control">
      <span>{t("evaluation.datasetRevision")}</span>
      <select
        value={selectedRevisionId}
        onChange={(event) => onSelect(event.target.value)}
      >
        <option value="">{t("evaluation.selectDataset")}</option>
        {state.datasets.map((dataset) => (
          <option key={dataset.revision.id} value={dataset.revision.id}>
            {dataset.revision.datasetKey} - {dataset.revision.semanticRevision}
            {dataset.available ? "" : ` (${t("evaluation.unavailable")})`}
          </option>
        ))}
      </select>
    </label>
  );
}

function DatasetDetailContent({ state }: { readonly state: DetailState }) {
  const { t } = useTranslation();
  if (state.status === "loading")
    return <output>{t("evaluation.detailLoading")}</output>;
  if (state.status === "failed")
    return <p role="alert">{t("evaluation.detailFailed")}</p>;
  if (state.status === "ready")
    return <DatasetReadinessDetail detail={state.detail} />;
  return null;
}

function DatasetSnapshotSelection({
  detail,
  snapshotOptions,
  selection,
  preflightState,
  onSelect,
  onCheck,
  onStartRun,
}: {
  readonly detail: EvaluationDatasetDetail;
  readonly snapshotOptions: readonly EvaluationSnapshotOption[];
  readonly selection: string;
  readonly preflightState: PreflightState;
  readonly onSelect: (selection: string) => void;
  readonly onCheck: (request: EvaluationDatasetPreflightRequest) => void;
  readonly onStartRun: DatasetReadinessViewProps["onStartRun"];
}) {
  const { t } = useTranslation();
  const options = snapshotOptions.filter(
    (option) => option.corpusId === detail.revision.corpusId,
  );
  const selectedSnapshot = options.find(
    (option) => snapshotOptionValue(option) === selection,
  );

  return (
    <section
      className="evaluation-readiness__selection"
      aria-labelledby="evaluation-snapshot-selection"
    >
      <h2 id="evaluation-snapshot-selection">
        {t("evaluation.snapshotSelection")}
      </h2>
      <SnapshotOptionControl
        options={options}
        selection={selection}
        onSelect={onSelect}
      />
      {selectedSnapshot === undefined ? null : (
        <SelectedSnapshotIdentity snapshot={selectedSnapshot} />
      )}
      <DatasetPreflightControls
        detail={detail}
        selectedSnapshot={selectedSnapshot}
        state={preflightState}
        onCheck={onCheck}
        onStartRun={onStartRun}
      />
    </section>
  );
}

function SnapshotOptionControl({
  options,
  selection,
  onSelect,
}: {
  readonly options: readonly EvaluationSnapshotOption[];
  readonly selection: string;
  readonly onSelect: (selection: string) => void;
}) {
  const { t } = useTranslation();

  return (
    <label className="evaluation-readiness__control">
      <span>{t("evaluation.snapshot")}</span>
      <select
        value={selection}
        onChange={(event) => onSelect(event.target.value)}
      >
        <option value="">{t("evaluation.selectSnapshot")}</option>
        {options.map((option) => (
          <option
            key={snapshotOptionValue(option)}
            value={snapshotOptionValue(option)}
          >
            {option.label}
          </option>
        ))}
      </select>
    </label>
  );
}

function DatasetPreflightControls({
  detail,
  selectedSnapshot,
  state,
  onCheck,
  onStartRun,
}: {
  readonly detail: EvaluationDatasetDetail;
  readonly selectedSnapshot: EvaluationSnapshotOption | undefined;
  readonly state: PreflightState;
  readonly onCheck: (request: EvaluationDatasetPreflightRequest) => void;
  readonly onStartRun: DatasetReadinessViewProps["onStartRun"];
}) {
  const { t } = useTranslation();
  const canCheck = detail.available && selectedSnapshot !== undefined;
  const canStartRun = canStartEvaluationRun(
    onStartRun,
    detail,
    selectedSnapshot,
    state,
  );
  const check = (): void => {
    if (selectedSnapshot === undefined) return;
    onCheck({
      datasetRevisionId: detail.revision.id,
      corpusId: selectedSnapshot.corpusId,
      snapshotId: selectedSnapshot.snapshotId,
    });
  };

  return (
    <>
      <button
        type="button"
        disabled={!canCheck || state.status === "checking"}
        onClick={check}
      >
        {state.status === "checking"
          ? t("evaluation.checkingCompatibility")
          : t("evaluation.checkCompatibility")}
      </button>
      <PreflightResult state={state} />
      {onStartRun === undefined ? null : (
        <button
          type="button"
          disabled={!canStartRun}
          onClick={() => {
            if (state.status === "compatible") onStartRun(state.selection);
          }}
        >
          {t("evaluation.startRun")}
        </button>
      )}
    </>
  );
}

function DatasetReadinessDetail({
  detail,
}: {
  readonly detail: EvaluationDatasetDetail;
}) {
  const { t } = useTranslation();
  const review = detail.review;
  const reviewValue =
    review === null
      ? t("evaluation.reviewUnavailable")
      : formatReviewValue(
          t(`evaluation.reviewDecision.${review.decision}`),
          t(`evaluation.publicationState.${review.publicationState}`),
          review.reviewedAt,
        );

  return (
    <section
      className="evaluation-readiness__detail"
      aria-labelledby="evaluation-identity"
    >
      <h2 id="evaluation-identity">{t("evaluation.immutableIdentity")}</h2>
      <dl className="evaluation-readiness__identity">
        <IdentityRow
          label={t("evaluation.datasetRevisionId")}
          value={detail.revision.id}
        />
        <IdentityRow
          label={t("evaluation.corpusId")}
          value={detail.revision.corpusId}
        />
        <IdentityRow
          label={t("evaluation.datasetKey")}
          value={detail.revision.datasetKey}
        />
        <IdentityRow
          label={t("evaluation.semanticRevision")}
          value={detail.revision.semanticRevision}
        />
        <IdentityRow
          label={t("evaluation.jurisdiction")}
          value={detail.revision.jurisdiction}
        />
        <IdentityRow
          label={t("evaluation.declaredSnapshotDate")}
          value={detail.revision.declaredSnapshotDate}
        />
        <IdentityRow
          label={t("evaluation.manifestSha256")}
          value={detail.revision.manifestSha256}
        />
        <IdentityRow
          label={t("evaluation.jsonlSha256")}
          value={detail.revision.jsonlSha256}
        />
        <IdentityRow
          label={t("evaluation.contentSha256")}
          value={detail.revision.contentSha256}
        />
        <IdentityRow
          label={t("evaluation.queryLanguages")}
          value={detail.revision.queryLanguages.join(", ")}
        />
        <IdentityRow
          label={t("evaluation.evidenceLanguage")}
          value={detail.revision.authoritativeEvidenceLanguage}
        />
        <IdentityRow
          label={t("evaluation.availability")}
          value={
            detail.available
              ? t("evaluation.available")
              : t("evaluation.unavailable")
          }
        />
        <IdentityRow label={t("evaluation.review")} value={reviewValue} />
      </dl>

      <h3>{t("evaluation.sourceRequirements")}</h3>
      <ul className="evaluation-readiness__sources">
        {detail.sources.map((source) => (
          <li key={source.id}>
            <a href={source.officialUrl} rel="noreferrer" target="_blank">
              {source.title}
            </a>
            <span>{source.sourceAlias}</span>
            <span>{source.issuingAuthority}</span>
            <span>{source.documentType}</span>
            <span>{source.authorityRole}</span>
            <span>
              {source.bound ? t("evaluation.bound") : t("evaluation.unbound")}
            </span>
          </li>
        ))}
      </ul>
    </section>
  );
}

function PreflightResult({ state }: { readonly state: PreflightState }) {
  const { t } = useTranslation();
  if (state.status === "compatible") {
    return <output>{t("evaluation.compatible")}</output>;
  }
  if (state.status === "incompatible") {
    return (
      <div role="alert">
        <p>{t("evaluation.incompatible")}</p>
        {state.requirements.length > 0 ? (
          <ul>
            {state.requirements.map((requirement, index) => (
              <li key={`${requirement.sourceAlias}-${String(index)}`}>
                <strong>{requirement.sourceAlias}</strong>
                {requirement.locator === undefined
                  ? null
                  : ` - ${requirement.locator}`}
                {` - ${requirement.reason}`}
              </li>
            ))}
          </ul>
        ) : null}
      </div>
    );
  }
  if (state.status === "failed") {
    return <p role="alert">{t("evaluation.preflightFailed")}</p>;
  }
  return null;
}

function canStartEvaluationRun(
  onStartRun: DatasetReadinessViewProps["onStartRun"],
  detail: EvaluationDatasetDetail | null,
  selectedSnapshot: EvaluationSnapshotOption | undefined,
  preflightState: PreflightState,
): boolean {
  if (
    onStartRun === undefined ||
    detail === null ||
    selectedSnapshot === undefined ||
    preflightState.status !== "compatible"
  ) {
    return false;
  }

  return (
    preflightState.selection.datasetRevisionId === detail.revision.id &&
    preflightState.selection.corpusId === selectedSnapshot.corpusId &&
    preflightState.selection.snapshotId === selectedSnapshot.snapshotId
  );
}

function formatReviewValue(
  reviewDecision: string,
  publicationState: string,
  reviewedAt: string,
): string {
  return `${reviewDecision} - ${publicationState} - ${reviewedAt}`;
}

function SelectedSnapshotIdentity({
  snapshot,
}: {
  readonly snapshot: EvaluationSnapshotOption;
}) {
  const { t } = useTranslation();

  return (
    <section
      className="evaluation-readiness__snapshot-identity"
      aria-labelledby="evaluation-snapshot-identity"
    >
      <h3 id="evaluation-snapshot-identity">
        {t("evaluation.selectedSnapshotIdentity")}
      </h3>
      <dl className="evaluation-readiness__identity">
        <IdentityRow
          label={t("evaluation.selectedCorpusId")}
          value={snapshot.corpusId}
        />
        <IdentityRow
          label={t("evaluation.selectedSnapshotId")}
          value={snapshot.snapshotId}
        />
        <IdentityRow
          label={t("evaluation.selectedSnapshotManifestSha256")}
          value={snapshot.snapshotManifestSha256}
        />
      </dl>
    </section>
  );
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

function snapshotOptionValue(option: EvaluationSnapshotOption): string {
  return `${option.corpusId}:${option.snapshotId}`;
}

function isCompatibilityError(
  error: unknown,
): error is EvaluationCatalogRequestError {
  return (
    error instanceof EvaluationCatalogRequestError &&
    (error.code === "dataset_not_available" ||
      error.code === "corpus_mismatch" ||
      error.code === "snapshot_incompatible" ||
      error.code === "locator_unresolved")
  );
}

function preflightFailureState(error: unknown): PreflightState {
  if (isCompatibilityError(error)) {
    return {
      status: "incompatible",
      requirements: error.missingRequirements ?? [],
    };
  }
  return { status: "failed" };
}
