import { useEffect, useRef, useState } from "react";
import { useTranslation } from "react-i18next";

import type {
  EvaluationDatasetCatalogEntry,
  EvaluationDatasetDetail,
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
  const { t } = useTranslation();
  const [catalogState, setCatalogState] = useState<CatalogState>({
    status: "loading",
  });
  const [detailState, setDetailState] = useState<DetailState>({
    status: "idle",
  });
  const [preflightState, setPreflightState] = useState<PreflightState>({
    status: "idle",
  });
  const [datasetRevisionId, setDatasetRevisionId] = useState("");
  const [snapshotSelection, setSnapshotSelection] = useState("");
  const preflightController = useRef<AbortController | null>(null);

  useEffect(() => {
    const controller = new AbortController();
    void client
      .listDatasets(controller.signal)
      .then((datasets) => {
        if (!controller.signal.aborted) {
          setCatalogState({ status: "ready", datasets });
        }
      })
      .catch(() => {
        if (!controller.signal.aborted) setCatalogState({ status: "failed" });
      });
    return () => controller.abort();
  }, [client]);

  useEffect(() => {
    if (datasetRevisionId === "") return undefined;

    const controller = new AbortController();
    void client
      .getDataset(datasetRevisionId, controller.signal)
      .then((detail) => {
        if (!controller.signal.aborted) {
          setDetailState({ status: "ready", detail });
        }
      })
      .catch(() => {
        if (!controller.signal.aborted) setDetailState({ status: "failed" });
      });
    return () => controller.abort();
  }, [client, datasetRevisionId]);

  useEffect(() => {
    return () => preflightController.current?.abort();
  }, []);

  const detail = detailState.status === "ready" ? detailState.detail : null;
  const corpusSnapshotOptions = snapshotOptions.filter(
    (option) => option.corpusId === detail?.revision.corpusId,
  );
  const selectedSnapshot = corpusSnapshotOptions.find(
    (option) => snapshotOptionValue(option) === snapshotSelection,
  );
  const canCheckCompatibility =
    detail?.available === true && selectedSnapshot !== undefined;
  const canStartRun =
    onStartRun !== undefined &&
    detail !== null &&
    selectedSnapshot !== undefined &&
    preflightState.status === "compatible" &&
    preflightState.selection.datasetRevisionId === detail.revision.id &&
    preflightState.selection.corpusId === selectedSnapshot.corpusId &&
    preflightState.selection.snapshotId === selectedSnapshot.snapshotId;

  const checkCompatibility = (): void => {
    if (!canCheckCompatibility) return;

    preflightController.current?.abort();
    const controller = new AbortController();
    preflightController.current = controller;
    setPreflightState({ status: "checking" });
    void client
      .preflightDataset(
        {
          datasetRevisionId: detail.revision.id,
          corpusId: selectedSnapshot.corpusId,
          snapshotId: selectedSnapshot.snapshotId,
        },
        controller.signal,
      )
      .then((selection) => {
        if (!controller.signal.aborted) {
          setPreflightState({ status: "compatible", selection });
        }
      })
      .catch((error: unknown) => {
        if (controller.signal.aborted) return;
        if (isCompatibilityError(error)) {
          setPreflightState({
            status: "incompatible",
            requirements: error.missingRequirements ?? [],
          });
          return;
        }
        setPreflightState({ status: "failed" });
      });
  };

  const selectDatasetRevision = (selectedRevisionId: string): void => {
    preflightController.current?.abort();
    setDatasetRevisionId(selectedRevisionId);
    setSnapshotSelection("");
    setPreflightState({ status: "idle" });
    setDetailState(
      selectedRevisionId === "" ? { status: "idle" } : { status: "loading" },
    );
  };

  const selectSnapshot = (selectedSnapshotValue: string): void => {
    preflightController.current?.abort();
    setSnapshotSelection(selectedSnapshotValue);
    setPreflightState({ status: "idle" });
  };

  return (
    <section
      className="evaluation-readiness"
      aria-labelledby="evaluation-heading"
    >
      <header className="evaluation-readiness__header">
        <p className="kicker">{t("evaluation.kicker")}</p>
        <h1 id="evaluation-heading">{t("evaluation.title")}</h1>
        <p>{t("evaluation.introduction")}</p>
      </header>

      <p className="evaluation-readiness__notice" role="note">
        {t("evaluation.notice")}
      </p>

      {catalogState.status === "loading" ? (
        <output>{t("evaluation.loading")}</output>
      ) : null}
      {catalogState.status === "failed" ? (
        <p role="alert">{t("evaluation.catalogFailed")}</p>
      ) : null}
      {catalogState.status === "ready" ? (
        <label className="evaluation-readiness__control">
          <span>{t("evaluation.datasetRevision")}</span>
          <select
            value={datasetRevisionId}
            onChange={(event) => selectDatasetRevision(event.target.value)}
          >
            <option value="">{t("evaluation.selectDataset")}</option>
            {catalogState.datasets.map((dataset) => (
              <option key={dataset.revision.id} value={dataset.revision.id}>
                {dataset.revision.datasetKey} -{" "}
                {dataset.revision.semanticRevision}
                {dataset.available ? "" : ` (${t("evaluation.unavailable")})`}
              </option>
            ))}
          </select>
        </label>
      ) : null}

      {detailState.status === "loading" ? (
        <output>{t("evaluation.detailLoading")}</output>
      ) : null}
      {detailState.status === "failed" ? (
        <p role="alert">{t("evaluation.detailFailed")}</p>
      ) : null}
      {detail !== null ? <DatasetReadinessDetail detail={detail} /> : null}

      {detail !== null ? (
        <section
          className="evaluation-readiness__selection"
          aria-labelledby="evaluation-snapshot-selection"
        >
          <h2 id="evaluation-snapshot-selection">
            {t("evaluation.snapshotSelection")}
          </h2>
          <label className="evaluation-readiness__control">
            <span>{t("evaluation.snapshot")}</span>
            <select
              value={snapshotSelection}
              onChange={(event) => selectSnapshot(event.target.value)}
            >
              <option value="">{t("evaluation.selectSnapshot")}</option>
              {corpusSnapshotOptions.map((option) => (
                <option
                  key={snapshotOptionValue(option)}
                  value={snapshotOptionValue(option)}
                >
                  {option.label}
                </option>
              ))}
            </select>
          </label>
          {selectedSnapshot !== undefined ? (
            <SelectedSnapshotIdentity snapshot={selectedSnapshot} />
          ) : null}
          <button
            type="button"
            disabled={
              !canCheckCompatibility || preflightState.status === "checking"
            }
            onClick={checkCompatibility}
          >
            {preflightState.status === "checking"
              ? t("evaluation.checkingCompatibility")
              : t("evaluation.checkCompatibility")}
          </button>
          <PreflightResult state={preflightState} />
          {onStartRun !== undefined ? (
            <button
              type="button"
              disabled={!canStartRun}
              onClick={() => {
                if (preflightState.status === "compatible") {
                  onStartRun(preflightState.selection);
                }
              }}
            >
              {t("evaluation.startRun")}
            </button>
          ) : null}
        </section>
      ) : null}
    </section>
  );
}

function DatasetReadinessDetail({
  detail,
}: {
  readonly detail: EvaluationDatasetDetail;
}) {
  const { t } = useTranslation();
  const review = detail.review;

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
        <IdentityRow
          label={t("evaluation.review")}
          value={
            review === null
              ? t("evaluation.reviewUnavailable")
              : `${t(`evaluation.reviewDecision.${review.decision}`)} - ${t(`evaluation.publicationState.${review.publicationState}`)} - ${review.reviewedAt}`
          }
        />
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
    return <p role="status">{t("evaluation.compatible")}</p>;
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
