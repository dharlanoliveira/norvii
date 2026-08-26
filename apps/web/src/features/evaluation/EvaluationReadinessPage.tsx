import { useEffect, useState } from "react";
import { useTranslation } from "react-i18next";

import type { EvaluationDatasetPreflightResponse } from "../../api/evaluationCatalog";
import type { EvaluationCatalogClient } from "../../api/evaluationCatalogClient";
import type { ResearchProvider } from "../../research/domain/authoritative";
import {
  DatasetReadinessView,
  type EvaluationSnapshotOption,
} from "./DatasetReadinessView";

interface EvaluationReadinessPageProps {
  readonly client: EvaluationCatalogClient;
  readonly provider: ResearchProvider;
  readonly onStartRun?:
    ((selection: EvaluationDatasetPreflightResponse) => void) | undefined;
}

type SnapshotOptionsState =
  | { readonly status: "loading" }
  | {
      readonly status: "ready";
      readonly snapshotOptions: readonly EvaluationSnapshotOption[];
    }
  | { readonly status: "failed" };

export function EvaluationReadinessPage({
  client,
  provider,
  onStartRun,
}: EvaluationReadinessPageProps) {
  const { t } = useTranslation();
  const [snapshotOptionsState, setSnapshotOptionsState] =
    useState<SnapshotOptionsState>({ status: "loading" });

  useEffect(() => {
    const controller = new AbortController();
    void loadSnapshotOptions(provider, controller.signal)
      .then((snapshotOptions) => {
        if (!controller.signal.aborted) {
          setSnapshotOptionsState({ status: "ready", snapshotOptions });
        }
      })
      .catch(() => {
        if (!controller.signal.aborted) {
          setSnapshotOptionsState({ status: "failed" });
        }
      });
    return () => controller.abort();
  }, [provider]);

  const snapshotOptions =
    snapshotOptionsState.status === "ready"
      ? snapshotOptionsState.snapshotOptions
      : [];

  return (
    <>
      {snapshotOptionsState.status === "loading" ? (
        <output>{t("evaluation.snapshotLoading")}</output>
      ) : null}
      {snapshotOptionsState.status === "failed" ? (
        <p role="alert">{t("evaluation.snapshotFailed")}</p>
      ) : null}
      <DatasetReadinessView
        client={client}
        onStartRun={onStartRun}
        snapshotOptions={snapshotOptions}
      />
    </>
  );
}

async function loadSnapshotOptions(
  provider: ResearchProvider,
  signal: AbortSignal,
): Promise<readonly EvaluationSnapshotOption[]> {
  const corpora = await provider.listCorpora(signal, true);
  const snapshotsByCorpus = await Promise.all(
    corpora.map(async (corpus) => ({
      corpus,
      snapshots: await provider.listSnapshots(corpus.id, signal),
    })),
  );

  return snapshotsByCorpus.flatMap(({ corpus, snapshots }) =>
    snapshots.map((snapshot) => ({
      corpusId: corpus.id,
      snapshotId: snapshot.id,
      snapshotManifestSha256: snapshot.manifestSha256,
      label: `${corpus.name} - ${snapshot.id}`,
    })),
  );
}
