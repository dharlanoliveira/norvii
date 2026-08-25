import { ChevronDown, History } from "lucide-react";
import { useState } from "react";
import { useTranslation } from "react-i18next";

import type {
  ActiveSnapshotResponse,
  GraphReleaseResponse,
  SnapshotResponse,
} from "../../api/contract";

interface SnapshotHistoryProps {
  readonly activeSnapshot?: ActiveSnapshotResponse | null | undefined;
  readonly loadSnapshots: (
    signal: AbortSignal,
  ) => Promise<readonly SnapshotResponse[]>;
  readonly loadGraphRelease: (
    snapshotId: string,
    signal: AbortSignal,
  ) => Promise<GraphReleaseResponse>;
}

type SnapshotHistoryState =
  | { readonly status: "idle" }
  | { readonly status: "loading" }
  | {
      readonly status: "ready";
      readonly snapshots: readonly SnapshotResponse[];
    }
  | { readonly status: "failed" };

/**
 * Defers manifest retrieval until a maintainer asks to inspect release provenance.
 * The workspace remains focused on reading while preserving a direct audit path.
 */
export function SnapshotHistory({
  activeSnapshot,
  loadGraphRelease,
  loadSnapshots,
}: SnapshotHistoryProps) {
  const { t } = useTranslation();
  const [state, setState] = useState<SnapshotHistoryState>({ status: "idle" });
  const [graphRelease, setGraphRelease] = useState<
    | { readonly status: "idle" }
    | { readonly status: "ready"; readonly release: GraphReleaseResponse }
    | { readonly status: "unavailable" }
  >({ status: "idle" });

  const load = (): void => {
    if (state.status !== "idle") return;
    const controller = new AbortController();
    setState({ status: "loading" });
    void loadSnapshots(controller.signal)
      .then((snapshots) => setState({ status: "ready", snapshots }))
      .catch(() => setState({ status: "failed" }));
  };

  const loadActiveGraphRelease = (): void => {
    if (
      activeSnapshot === null ||
      activeSnapshot === undefined ||
      graphRelease.status !== "idle"
    ) {
      return;
    }
    const controller = new AbortController();
    void loadGraphRelease(activeSnapshot.id, controller.signal)
      .then((release) => setGraphRelease({ status: "ready", release }))
      .catch(() => setGraphRelease({ status: "unavailable" }));
  };

  return (
    <details
      className="snapshot-history"
      onToggle={(event) => {
        if (event.currentTarget.open) load();
        if (event.currentTarget.open) loadActiveGraphRelease();
      }}
    >
      <summary>
        <History aria-hidden="true" size={14} />
        {t("sourceManagement.snapshot.history")}
        <ChevronDown aria-hidden="true" size={14} />
      </summary>
      <div className="snapshot-history__content">
        <GraphReleaseSummary state={graphRelease} />
        {state.status === "loading" ? (
          <output>{t("sourceManagement.snapshot.loading")}</output>
        ) : null}
        {state.status === "failed" ? (
          <p role="alert">{t("sourceManagement.snapshot.loadFailed")}</p>
        ) : null}
        {state.status === "ready" && state.snapshots.length === 0 ? (
          <p>{t("sourceManagement.snapshot.empty")}</p>
        ) : null}
        {state.status === "ready" ? (
          <ol aria-label={t("sourceManagement.snapshot.history")}>
            {state.snapshots.map((snapshot) => (
              <SnapshotManifest
                active={snapshot.id === activeSnapshot?.id}
                key={snapshot.id}
                snapshot={snapshot}
              />
            ))}
          </ol>
        ) : null}
      </div>
    </details>
  );
}

function GraphReleaseSummary({
  state,
}: {
  readonly state:
    | { readonly status: "idle" }
    | { readonly status: "ready"; readonly release: GraphReleaseResponse }
    | { readonly status: "unavailable" };
}) {
  const { t } = useTranslation();
  if (state.status === "idle") return null;
  if (state.status === "unavailable") {
    return (
      <p>
        {t("sourceManagement.snapshot.graphUnavailable")}{" "}
        {t("sourceManagement.snapshot.graphGuidance")}
      </p>
    );
  }
  const { release } = state;
  return (
    <section
      className="snapshot-history__graph"
      aria-label={t("sourceManagement.snapshot.graph")}
    >
      <strong>{t("sourceManagement.snapshot.graph")}</strong>
      <span data-status={release.status}>
        {t(`sourceManagement.snapshot.graphStates.${release.status}`)}
      </span>
      <span>
        {t("sourceManagement.snapshot.graphCounts", {
          entities: release.entityCount,
          relationships: release.relationshipCount,
        })}
      </span>
      {release.failureCategory ? <code>{release.failureCategory}</code> : null}
    </section>
  );
}

function SnapshotManifest({
  active,
  snapshot,
}: {
  readonly active: boolean;
  readonly snapshot: SnapshotResponse;
}) {
  const { t } = useTranslation();
  return (
    <li className="snapshot-history__release">
      <header>
        <div>
          <strong>{t("sourceManagement.snapshot.release")}</strong>
          {active ? <span>{t("sourceManagement.snapshot.active")}</span> : null}
        </div>
        <time dateTime={snapshot.createdAt}>
          {new Date(snapshot.createdAt).toLocaleString()}
        </time>
      </header>
      <dl>
        <div>
          <dt>{t("sourceManagement.snapshot.identity")}</dt>
          <dd>
            <code>{snapshot.id}</code>
          </dd>
        </div>
        <div>
          <dt>{t("sourceManagement.snapshot.manifest")}</dt>
          <dd>
            <code>{snapshot.manifestSha256}</code>
          </dd>
        </div>
      </dl>
      <ol className="snapshot-history__members">
        {snapshot.members.map((member) => (
          <li key={member.sourceId}>
            <a href={member.officialOrigin} rel="noreferrer" target="_blank">
              {t("sourceManagement.snapshot.officialOrigin")}
            </a>
            <dl>
              <div>
                <dt>{t("sourceManagement.snapshot.capturedAt")}</dt>
                <dd>
                  <time dateTime={member.capturedAt}>
                    {new Date(member.capturedAt).toLocaleString()}
                  </time>
                </dd>
              </div>
              <div>
                <dt>{t("sourceManagement.snapshot.sourceRevision")}</dt>
                <dd>
                  <code>{member.sourceRevisionId}</code>
                </dd>
              </div>
              <div>
                <dt>{t("sourceManagement.snapshot.document")}</dt>
                <dd>
                  <code>{member.documentId}</code>
                </dd>
              </div>
              <div>
                <dt>{t("sourceManagement.snapshot.contentIdentity")}</dt>
                <dd>
                  <code>{member.contentSha256}</code>
                </dd>
              </div>
            </dl>
          </li>
        ))}
      </ol>
    </li>
  );
}
