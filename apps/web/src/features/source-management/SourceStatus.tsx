import {
  ChevronDown,
  ExternalLink,
  FileText,
  History,
  Info,
  RefreshCw,
} from "lucide-react";
import { useState } from "react";
import { useTranslation } from "react-i18next";

import type {
  ActiveSnapshotResponse,
  GraphReleaseResponse,
  SnapshotResponse,
  SourceResponse,
} from "../../api/contract";
import { ResearchRequestError } from "../../api/researchProvider";
import { SnapshotHistory } from "./SnapshotHistory";

type SourceStatusAction = (
  source: SourceResponse,
  signal: AbortSignal,
) => Promise<void>;

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
  const [outcome, setOutcome] = useState<
    "idle" | "saving" | "failed" | "stale"
  >("idle");
  const visibleAttempts = source.attempts.slice(0, ATTEMPT_HISTORY_LIMIT);

  const run = (action: SourceStatusAction): void => {
    const controller = new AbortController();
    setOutcome("saving");
    void action(source, controller.signal)
      .then(() => setOutcome("idle"))
      .catch((error: unknown) =>
        setOutcome(
          error instanceof ResearchRequestError && error.code === "stale_state"
            ? "stale"
            : "failed",
        ),
      );
  };
  const canPublish =
    source.processingStatus === "ready" &&
    source.latestReadyDocumentId !== null &&
    activeSnapshot !== null &&
    activeSnapshot !== undefined &&
    source.latestReadyDocumentId !== source.activeSnapshotDocumentId;
  const publicationState =
    source.processingStatus !== "ready" || source.latestReadyDocumentId === null
      ? undefined
      : source.latestReadyDocumentId === source.activeSnapshotDocumentId
        ? "active"
        : "candidate";

  return (
    <section
      className="source-status"
      aria-label={t("sourceManagement.lifecycle.label")}
    >
      <header className="source-status__bar">
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
        <div
          className="source-status__actions"
          role="toolbar"
          aria-label={t("viewer.sourceActions")}
        >
          <details className="source-status__details">
            <summary>
              <Info aria-hidden="true" size={15} />
              {t("viewer.metadata")}
              <ChevronDown aria-hidden="true" size={14} />
            </summary>
            <div className="source-status__details-panel">
              <dl className="source-status__metadata">
                {source.origin.submittedUrl ? (
                  <div>
                    <dt>{t("sourceManagement.lifecycle.submittedUrl")}</dt>
                    <dd>{source.origin.submittedUrl}</dd>
                  </div>
                ) : null}
                {source.origin.originalFilename ? (
                  <div>
                    <dt>{t("sourceManagement.lifecycle.originalFilename")}</dt>
                    <dd>{source.origin.originalFilename}</dd>
                  </div>
                ) : null}
                {source.origin.mediaType ? (
                  <div>
                    <dt>{t("viewer.mediaType")}</dt>
                    <dd>{source.origin.mediaType}</dd>
                  </div>
                ) : null}
                {source.origin.byteSize !== null ? (
                  <div>
                    <dt>{t("viewer.byteSize")}</dt>
                    <dd>{source.origin.byteSize.toLocaleString()}</dd>
                  </div>
                ) : null}
                {source.origin.sha256 ? (
                  <div>
                    <dt>{t("viewer.originHash")}</dt>
                    <dd>{source.origin.sha256}</dd>
                  </div>
                ) : null}
              </dl>
              {visibleAttempts.length > 0 ? (
                <section
                  className="source-status__history"
                  aria-label={t("sourceManagement.lifecycle.attemptHistory")}
                >
                  <h3>
                    <History aria-hidden="true" size={16} />
                    {t("sourceManagement.lifecycle.attemptHistory")}
                  </h3>
                  <div className="source-status__attempt-list">
                    {visibleAttempts.map((attempt) => (
                      <article
                        className="source-status__attempt"
                        key={`${attempt.startedAt}-${String(attempt.number)}`}
                      >
                        <header>
                          <h4>
                            {t("sourceManagement.lifecycle.attemptNumber", {
                              number: attempt.number,
                            })}
                          </h4>
                          <span data-status={attempt.status}>
                            {t(
                              `sourceManagement.lifecycle.attemptStates.${attempt.status}`,
                            )}
                          </span>
                        </header>
                        <dl>
                          <div>
                            <dt>
                              {t("sourceManagement.lifecycle.pipelineVersion")}
                            </dt>
                            <dd>{attempt.pipelineVersion}</dd>
                          </div>
                          <div>
                            <dt>{t("sourceManagement.lifecycle.startedAt")}</dt>
                            <dd>
                              <time dateTime={attempt.startedAt}>
                                {new Date(attempt.startedAt).toLocaleString()}
                              </time>
                            </dd>
                          </div>
                          {attempt.finishedAt ? (
                            <div>
                              <dt>
                                {t("sourceManagement.lifecycle.finishedAt")}
                              </dt>
                              <dd>
                                <time dateTime={attempt.finishedAt}>
                                  {new Date(
                                    attempt.finishedAt,
                                  ).toLocaleString()}
                                </time>
                              </dd>
                            </div>
                          ) : null}
                          {attempt.failureCategory ? (
                            <div>
                              <dt>
                                {t(
                                  "sourceManagement.lifecycle.failureCategory",
                                )}
                              </dt>
                              <dd>{attempt.failureCategory}</dd>
                            </div>
                          ) : null}
                          {attempt.failureDetail ? (
                            <div>
                              <dt>
                                {t(
                                  "sourceManagement.lifecycle.failureDiagnostic",
                                )}
                              </dt>
                              <dd>{attempt.failureDetail}</dd>
                            </div>
                          ) : null}
                          {attempt.acquiredByteCount !== null ? (
                            <div>
                              <dt>
                                {t("sourceManagement.lifecycle.acquiredBytes")}
                              </dt>
                              <dd>
                                {attempt.acquiredByteCount.toLocaleString()}
                              </dd>
                            </div>
                          ) : null}
                          {attempt.normalizedCharacterCount !== null ? (
                            <div>
                              <dt>
                                {t(
                                  "sourceManagement.lifecycle.normalizedCharacters",
                                )}
                              </dt>
                              <dd>
                                {attempt.normalizedCharacterCount.toLocaleString()}
                              </dd>
                            </div>
                          ) : null}
                          {attempt.unitCount !== null ? (
                            <div>
                              <dt>
                                {t("sourceManagement.lifecycle.unitCount")}
                              </dt>
                              <dd>{attempt.unitCount.toLocaleString()}</dd>
                            </div>
                          ) : null}
                          {attempt.durationMilliseconds !== null ? (
                            <div>
                              <dt>
                                {t("sourceManagement.lifecycle.duration")}
                              </dt>
                              <dd>
                                {attempt.durationMilliseconds.toLocaleString()}
                              </dd>
                            </div>
                          ) : null}
                        </dl>
                      </article>
                    ))}
                  </div>
                </section>
              ) : null}
            </div>
          </details>
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
              disabled={outcome === "saving"}
              onClick={() => run(onRetry)}
            >
              <RefreshCw aria-hidden="true" size={14} />
              {t("sourceManagement.lifecycle.retry")}
            </button>
          ) : null}
          {source.processingStatus === "ready" &&
          publicationState !== "active" ? (
            <button
              className="source-status__action"
              type="button"
              disabled={outcome === "saving" || !canPublish}
              title={
                canPublish
                  ? undefined
                  : t("sourceManagement.snapshot.publishUnavailable")
              }
              onClick={() => run(onPublish)}
            >
              {t("sourceManagement.snapshot.publish")}
            </button>
          ) : null}
          {source.processingStatus === "ready" ? (
            <button
              className="source-status__action"
              type="button"
              disabled={outcome === "saving"}
              onClick={() => {
                if (
                  globalThis.confirm(
                    t("sourceManagement.lifecycle.confirmReprocess", {
                      title: source.title,
                    }),
                  )
                ) {
                  run(onReprocess);
                }
              }}
            >
              <RefreshCw aria-hidden="true" size={14} />
              {t("sourceManagement.lifecycle.reprocess")}
            </button>
          ) : null}
          {source.kind === "pdf" ? (
            <a
              className="source-status__action source-status__action--primary"
              href={`/api/v1/corpora/${encodeURIComponent(source.corpusId)}/sources/${encodeURIComponent(source.id)}/origin/pdf`}
              download
            >
              <FileText aria-hidden="true" size={14} />
              {t("viewer.downloadPdf")}
            </a>
          ) : (
            <a
              className="source-status__action source-status__action--primary"
              href={`/api/v1/corpora/${encodeURIComponent(source.corpusId)}/sources/${encodeURIComponent(source.id)}/origin`}
              target="_blank"
              rel="noreferrer noopener"
            >
              <ExternalLink aria-hidden="true" size={14} />
              {t("viewer.openOfficial")}
            </a>
          )}
        </div>
      </header>
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
    </section>
  );
}
