import { lazy, Suspense } from "react";
import { Route, Routes } from "react-router-dom";

import { createHttpResearchProvider } from "../api/researchProvider";
import { CorpusCatalogPage } from "../features/catalog/CorpusCatalogPage";
import { CorpusFormPage } from "../features/catalog/CorpusFormPage";
import { UnknownCorpusPage } from "../features/workspace/UnknownCorpusPage";
import type { ResearchProvider } from "../research/domain/authoritative";
import { AppShell } from "./AppShell";

const CorpusWorkspacePage = lazy(async () => {
  const module = await import("../features/workspace/CorpusWorkspacePage");
  return { default: module.CorpusWorkspacePage };
});

const defaultProvider = createHttpResearchProvider();

interface AppRoutesProps {
  readonly provider?: ResearchProvider;
}

export function AppRoutes({ provider = defaultProvider }: AppRoutesProps) {
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
        <Route path="*" element={<UnknownCorpusPage />} />
      </Route>
    </Routes>
  );
}
