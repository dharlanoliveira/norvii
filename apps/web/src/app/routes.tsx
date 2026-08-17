import { lazy, Suspense } from "react";
import { Route, Routes } from "react-router-dom";

import { CorpusCatalogPage } from "../features/catalog/CorpusCatalogPage";
import { UnknownCorpusPage } from "../features/workspace/UnknownCorpusPage";
import { createDemonstrationCatalog } from "../research/demonstration/createDemonstrationCatalog";
import { AppShell } from "./AppShell";

const CorpusWorkspacePage = lazy(async () => {
  const module = await import("../features/workspace/CorpusWorkspacePage");
  return { default: module.CorpusWorkspacePage };
});

const catalog = createDemonstrationCatalog();

export function AppRoutes() {
  return (
    <Routes>
      <Route element={<AppShell />}>
        <Route index element={<CorpusCatalogPage catalog={catalog} />} />
        <Route
          path="corpora/:corpusId"
          element={
            <Suspense fallback={null}>
              <CorpusWorkspacePage catalog={catalog} />
            </Suspense>
          }
        />
        <Route path="*" element={<UnknownCorpusPage />} />
      </Route>
    </Routes>
  );
}
