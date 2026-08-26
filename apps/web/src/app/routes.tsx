import { lazy, Suspense } from "react";
import { useTranslation } from "react-i18next";
import { Route, Routes, useParams, useSearchParams } from "react-router-dom";

import { createHttpResearchProvider } from "../api/researchProvider";
import { createEvaluationCatalogClient } from "../api/evaluationCatalogClient";
import { createEvaluationRunClient } from "../api/evaluationRunClient";
import { CorpusCatalogPage } from "../features/catalog/CorpusCatalogPage";
import { CorpusFormPage } from "../features/catalog/CorpusFormPage";
import { EvaluationReadinessPage } from "../features/evaluation/EvaluationReadinessPage";
import {
  EvaluationComparisonView,
  EvaluationRunInspection,
} from "../features/evaluation/EvaluationRunInspection";
import { UnknownCorpusPage } from "../features/workspace/UnknownCorpusPage";
import type { EvaluationDatasetPreflightResponse } from "../api/evaluationCatalog";
import type { EvaluationCatalogClient } from "../api/evaluationCatalogClient";
import type { EvaluationRunClient } from "../api/evaluationRunClient";
import type { ResearchProvider } from "../research/domain/authoritative";
import { AppShell } from "./AppShell";

const CorpusWorkspacePage = lazy(async () => {
  const module = await import("../features/workspace/CorpusWorkspacePage");
  return { default: module.CorpusWorkspacePage };
});

const defaultProvider = createHttpResearchProvider();
const defaultEvaluationClient = createEvaluationCatalogClient();
const defaultEvaluationRunClient = createEvaluationRunClient();

interface AppRoutesProps {
  readonly provider?: ResearchProvider;
  readonly evaluationClient?: EvaluationCatalogClient;
  readonly evaluationRunClient?: EvaluationRunClient;
  readonly onEvaluationPreflightSuccess?:
    ((selection: EvaluationDatasetPreflightResponse) => void) | undefined;
}

export function AppRoutes({
  provider = defaultProvider,
  evaluationClient = defaultEvaluationClient,
  evaluationRunClient = defaultEvaluationRunClient,
  onEvaluationPreflightSuccess,
}: AppRoutesProps) {
  return (
    <Routes>
      <Route element={<AppShell />}>
        <Route index element={<CorpusCatalogPage provider={provider} />} />
        <Route
          path="corpora/new"
          element={<CorpusFormPage provider={provider} />}
        />
        <Route
          path="corpora/:corpusId/edit"
          element={<CorpusFormPage provider={provider} />}
        />
        <Route
          path="corpora/:corpusId"
          element={
            <Suspense fallback={null}>
              <CorpusWorkspacePage provider={provider} />
            </Suspense>
          }
        />
        <Route
          path="evaluations"
          element={
            <EvaluationReadinessPage
              client={evaluationClient}
              onStartRun={onEvaluationPreflightSuccess}
              provider={provider}
            />
          }
        />
        <Route
          path="evaluations/compare"
          element={<EvaluationComparisonRoute client={evaluationRunClient} />}
        />
        <Route
          path="evaluations/:runId"
          element={<EvaluationRunRoute client={evaluationRunClient} />}
        />
        <Route path="*" element={<UnknownCorpusPage />} />
      </Route>
    </Routes>
  );
}

function EvaluationRunRoute({
  client,
}: {
  readonly client: EvaluationRunClient;
}) {
  const { runId } = useParams();
  if (runId === undefined) return null;
  return <EvaluationRunInspection client={client} runId={runId} />;
}

function EvaluationComparisonRoute({
  client,
}: {
  readonly client: EvaluationRunClient;
}) {
  const [params] = useSearchParams();
  const leftRunId = params.get("left");
  const rightRunId = params.get("right");
  if (leftRunId === null || rightRunId === null) {
    return <EvaluationComparisonMissingIdentities />;
  }
  return (
    <EvaluationComparisonView
      client={client}
      leftRunId={leftRunId}
      rightRunId={rightRunId}
    />
  );
}

function EvaluationComparisonMissingIdentities() {
  const { t } = useTranslation();
  return <p role="alert">{t("evaluation.comparison.identitiesRequired")}</p>;
}
