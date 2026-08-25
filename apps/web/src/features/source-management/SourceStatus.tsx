import {
  ChevronDown,
  ExternalLink,
  FileText,
  History,
  Info,
  RefreshCw,
} from "lucide-react";
import { useState, type ReactNode } from "react";
import { useTranslation } from "react-i18next";

import type {
  ActiveSnapshotResponse,
  GraphReleaseResponse,
  SnapshotResponse,
  ProcessingAttemptResponse,
  SourceResponse,
} from "../../api/contract";
import { ResearchRequestError } from "../../api/researchProvider";
import { SnapshotHistory } from "./SnapshotHistory";

type SourceStatusAction = (
  source: SourceResponse,
  signal: AbortSignal,
) => Promise<void>;

type ActionOutcome = "idle" | "saving" | "failed" | "stale";
type PublicationState = "active" | "candidate" | undefined;

interface SourceStatusProps {
  readonly source: SourceResponse;
  readonly activeSnapshot?: ActiveSnapshotResponse | null | undefined;
  readonly onLoadSnapshotHistory: (
    signal: AbortSignal,
  ) => Promise<readonly SnapshotResponse[]>;
  readonly onLoadGraphRelease: (
    snapshotId: string,
    signal: AbortSignal,
  ) => Promise<GraphReleaseResponse>;
  readonly onPublish: SourceStatusAction;
  readonly onRetry: SourceStatusAction;
  readonly onReprocess: SourceStatusAction;
}

interface SourceLifecycleActionsProps {
  readonly source: SourceResponse;
  readonly activeSnapshot?: ActiveSnapshotResponse | null | undefined;
  readonly outcome: ActionOutcome;
  readonly publicationState: PublicationState;
  readonly canPublish: boolean;
  readonly onLoadSnapshotHistory: (
    signal: AbortSignal,
  ) => Promise<readonly SnapshotResponse[]>;
  readonly onLoadGraphRelease: (
    snapshotId: string,
    signal: AbortSignal,
  ) => Promise<GraphReleaseResponse>;
  readonly onRun: (action: SourceStatusAction) => void;
  readonly onPublish: SourceStatusAction;
  readonly onRetry: SourceStatusAction;
  readonly onReprocess: SourceStatusAction;
}

const ATTEMPT_HISTORY_LIMIT = 3;

export function SourceStatus({
  source,
  activeSnapshot,
  onLoadSnapshotHistory,
  onLoadGraphRelease,
  onPublish,
  onRetry,
  onReprocess,
}: SourceStatusProps) {
  const { t } = useTranslation();
  const [outcome, setOutcome] = useState<ActionOutcome>("idle");
  const publicationState = publicationStateFor(source);
  const canPublish = canPublishSource(source, activeSnapshot);

  const run = (action: SourceStatusAction): void => {
    const controller = new AbortController();
    setOutcome("saving");
    void action(source, controller.signal)
      .then(() => setOutcome("idle"))
      .catch((error: unknown) => setOutcome(actionOutcomeFor(error)));
  };

  return (
    <section
      className="source-status"
      aria-label={t("sourceManagement.lifecycle.label")}
    >
      <header className="source-status__bar">
        <SourceIdentity source={source} publicationState={publicationState} />
        <SourceLifecycleActions
          activeSnapshot={activeSnapshot}
          canPublish={canPublish}
          onLoadGraphRelease={onLoadGraphRelease}
          onLoadSnapshotHistory={onLoadSnapshotHistory}
          onPublish={onPublish}
          onReprocess={onReprocess}
          onRetry={onRetry}
          onRun={run}
          outcome={outcome}
          publicationState={publicationState}
          source={source}
        />
      </header>
      <SourceStatusNotices outcome={outcome} source={source} />
    </section>
  );
}

function SourceIdentity({
  source,
  publicationState,
}: {
  readonly source: SourceResponse;
  readonly publicationState: PublicationState;
}) {
  const { t } = useTranslation();
  return (
    <div className="source-status__identity">
      <h2>{source.title}</h2>
      <span
        className="source-status__badge"
        data-status={source.processingStatus}
      >
        {t(`sourceStatus.${source.processingStatus}`)}
      </span>
      {publicationState === undefined ? null : (
        <span className="source-status__publication-state">
          {t(`sourceManagement.snapshot.${publicationState}`)}
        </span>
      )}
    </div>
  );
}

function SourceLifecycleActions({
  source,
  activeSnapshot,
  outcome,
  publicationState,
  canPublish,
  onLoadSnapshotHistory,
  onLoadGraphRelease,
  onRun,
  onPublish,
  onRetry,
  onReprocess,
}: SourceLifecycleActionsProps) {
  const { t } = useTranslation();
  const isSaving = outcome === "saving";
  return (
    <div
      className="source-status__actions"
      role="toolbar"
      aria-label={t("viewer.sourceActions")}
    >
      <SourceDetails source={source} />
      <SnapshotHistory
        key={activeSnapshot?.id ?? "no-active-snapshot"}
        activeSnapshot={activeSnapshot}
        loadGraphRelease={onLoadGraphRelease}
        loadSnapshots={onLoadSnapshotHistory}
      />
      {source.processingStatus === "failed" ? (
        <button
          className="source-status__action"
          type="button"
          disabled={isSaving}
          onClick={() => onRun(onRetry)}
        >
          <RefreshCw aria-hidden="true" size={14} />
          {t("sourceManagement.lifecycle.retry")}
        </button>
      ) : null}
      {source.processingStatus === "ready" && publicationState !== "active" ? (
        <button
          className="source-status__action"
          type="button"
          disabled={isSaving || !canPublish}
          title={
            canPublish
              ? undefined
              : t("sourceManagement.snapshot.publishUnavailable")
          }
          onClick={() => onRun(onPublish)}
        >
          {t("sourceManagement.snapshot.publish")}
        </button>
      ) : null}
      {source.processingStatus === "ready" ? (
        <ReprocessSourceAction
          disabled={isSaving}
          source={source}
          onReprocess={onReprocess}
          onRun={onRun}
        />
      ) : null}
      <SourceOriginAction source={source} />
    </div>
  );
}

function SourceDetails({ source }: { readonly source: SourceResponse }) {
  const { t } = useTranslation();
  return (
    <details className="source-status__details">
      <summary>
        <Info aria-hidden="true" size={15} />
        {t("viewer.metadata")}
        <ChevronDown aria-hidden="true" size={14} />
      </summary>
      <div className="source-status__details-panel">
        <SourceMetadata source={source} />
        <AttemptHistory
          attempts={source.attempts.slice(0, ATTEMPT_HISTORY_LIMIT)}
        />
      </div>
    </details>
  );
}

function SourceMetadata({ source }: { readonly source: SourceResponse }) {
  const { t } = useTranslation();
  const entries = sourceMetadata(source, t);
  return (
    <dl className="source-status__metadata">
      {entries.map(({ label, value }) => (
        <div key={label}>
          <dt>{label}</dt>
          <dd>{value}</dd>
        </div>
      ))}
    </dl>
  );
}

function AttemptHistory({
  attempts,
}: {
  readonly attempts: readonly ProcessingAttemptResponse[];
}) {
  const { t } = useTranslation();
  if (attempts.length === 0) return null;
  return (
    <section
      className="source-status__history"
      aria-label={t("sourceManagement.lifecycle.attemptHistory")}
    >
      <h3>
        <History aria-hidden="true" size={16} />
        {t("sourceManagement.lifecycle.attemptHistory")}
      </h3>
      <div className="source-status__attempt-list">
        {attempts.map((attempt) => (
          <AttemptHistoryItem
            attempt={attempt}
            key={`${attempt.startedAt}-${String(attempt.number)}`}
          />
        ))}
      </div>
    </section>
  );
}

function AttemptHistoryItem({
  attempt,
}: {
  readonly attempt: ProcessingAttemptResponse;
}) {
  const { t } = useTranslation();
  return (
    <article className="source-status__attempt">
      <header>
        <h4>
          {t("sourceManagement.lifecycle.attemptNumber", {
            number: attempt.number,
          })}
        </h4>
        <span data-status={attempt.status}>
          {t(`sourceManagement.lifecycle.attemptStates.${attempt.status}`)}
        </span>
      </header>
      <dl>
        <AttemptDetail
          label={t("sourceManagement.lifecycle.pipelineVersion")}
          value={attempt.pipelineVersion}
        />
        <AttemptDetail
          label={t("sourceManagement.lifecycle.startedAt")}
          value={
            <time dateTime={attempt.startedAt}>
              {formatDate(attempt.startedAt)}
            </time>
          }
        />
        {attempt.finishedAt ? (
          <AttemptDetail
            label={t("sourceManagement.lifecycle.finishedAt")}
            value={
              <time dateTime={attempt.finishedAt}>
                {formatDate(attempt.finishedAt)}
              </time>
            }
          />
        ) : null}
        {attempt.failureCategory ? (
          <AttemptDetail
            label={t("sourceManagement.lifecycle.failureCategory")}
            value={attempt.failureCategory}
          />
        ) : null}
        {attempt.failureDetail ? (
          <AttemptDetail
            label={t("sourceManagement.lifecycle.failureDiagnostic")}
            value={attempt.failureDetail}
          />
        ) : null}
        {attempt.acquiredByteCount !== null ? (
          <AttemptDetail
            label={t("sourceManagement.lifecycle.acquiredBytes")}
            value={attempt.acquiredByteCount.toLocaleString()}
          />
        ) : null}
        {attempt.normalizedCharacterCount !== null ? (
          <AttemptDetail
            label={t("sourceManagement.lifecycle.normalizedCharacters")}
            value={attempt.normalizedCharacterCount.toLocaleString()}
          />
        ) : null}
        {attempt.unitCount !== null ? (
          <AttemptDetail
            label={t("sourceManagement.lifecycle.unitCount")}
            value={attempt.unitCount.toLocaleString()}
          />
        ) : null}
        {attempt.durationMilliseconds !== null ? (
          <AttemptDetail
            label={t("sourceManagement.lifecycle.duration")}
            value={attempt.durationMilliseconds.toLocaleString()}
          />
        ) : null}
      </dl>
    </article>
  );
}

function AttemptDetail({
  label,
  value,
}: {
  readonly label: string;
  readonly value: ReactNode;
}) {
  return (
    <div>
      <dt>{label}</dt>
      <dd>{value}</dd>
    </div>
  );
}

function ReprocessSourceAction({
  source,
  disabled,
  onReprocess,
  onRun,
}: {
  readonly source: SourceResponse;
  readonly disabled: boolean;
  readonly onReprocess: SourceStatusAction;
  readonly onRun: (action: SourceStatusAction) => void;
}) {
  const { t } = useTranslation();
  const confirmReprocess = (): void => {
    const confirmed = globalThis.confirm(
      t("sourceManagement.lifecycle.confirmReprocess", { title: source.title }),
    );
    if (confirmed) onRun(onReprocess);
  };
  return (
    <button
      className="source-status__action"
      type="button"
      disabled={disabled}
      onClick={confirmReprocess}
    >
      <RefreshCw aria-hidden="true" size={14} />
      {t("sourceManagement.lifecycle.reprocess")}
    </button>
  );
}

function SourceOriginAction({ source }: { readonly source: SourceResponse }) {
  const { t } = useTranslation();
  if (source.kind === "pdf") {
    return (
      <a
        className="source-status__action source-status__action--primary"
        href={sourceOriginPath(source, "/pdf")}
        download
      >
        <FileText aria-hidden="true" size={14} />
        {t("viewer.downloadPdf")}
      </a>
    );
  }
  return (
    <a
      className="source-status__action source-status__action--primary"
      href={sourceOriginPath(source)}
      target="_blank"
      rel="noreferrer noopener"
    >
      <ExternalLink aria-hidden="true" size={14} />
      {t("viewer.openOfficial")}
    </a>
  );
}

function SourceStatusNotices({
  source,
  outcome,
}: {
  readonly source: SourceResponse;
  readonly outcome: ActionOutcome;
}) {
  const { t } = useTranslation();
  return (
    <>
      {source.failureCategory ? (
        <p className="source-status__notice">
          {t("sourceManagement.lifecycle.safeFailure")}
          <code>{source.failureCategory}</code>
        </p>
      ) : null}
      {outcome === "stale" ? (
        <p className="source-status__notice" role="alert">
          {t("sourceManagement.lifecycle.stale")}
        </p>
      ) : null}
      {outcome === "failed" ? (
        <p className="source-status__notice" role="alert">
          {t("sourceManagement.lifecycle.failed")}
        </p>
      ) : null}
    </>
  );
}

function actionOutcomeFor(error: unknown): ActionOutcome {
  if (error instanceof ResearchRequestError && error.code === "stale_state") {
    return "stale";
  }
  return "failed";
}

function canPublishSource(
  source: SourceResponse,
  activeSnapshot: ActiveSnapshotResponse | null | undefined,
): boolean {
  return (
    source.processingStatus === "ready" &&
    source.latestReadyDocumentId !== null &&
    activeSnapshot !== null &&
    activeSnapshot !== undefined &&
    source.latestReadyDocumentId !== source.activeSnapshotDocumentId
  );
}

function publicationStateFor(source: SourceResponse): PublicationState {
  if (
    source.processingStatus !== "ready" ||
    source.latestReadyDocumentId === null
  ) {
    return undefined;
  }
  return source.latestReadyDocumentId === source.activeSnapshotDocumentId
    ? "active"
    : "candidate";
}

function sourceMetadata(
  source: SourceResponse,
  t: ReturnType<typeof useTranslation>["t"],
): readonly { readonly label: string; readonly value: string }[] {
  const entries = [
    [t("sourceManagement.lifecycle.submittedUrl"), source.origin.submittedUrl],
    [
      t("sourceManagement.lifecycle.originalFilename"),
      source.origin.originalFilename,
    ],
    [t("viewer.mediaType"), source.origin.mediaType],
    [
      t("viewer.byteSize"),
      source.origin.byteSize === null
        ? null
        : source.origin.byteSize.toLocaleString(),
    ],
    [t("viewer.originHash"), source.origin.sha256],
  ] as const;
  return entries.flatMap(([label, value]) => (value ? [{ label, value }] : []));
}

function sourceOriginPath(source: SourceResponse, suffix = ""): string {
  return `/api/v1/corpora/${encodeURIComponent(source.corpusId)}/sources/${encodeURIComponent(source.id)}/origin${suffix}`;
}

function formatDate(value: string): string {
  return new Date(value).toLocaleString();
}
