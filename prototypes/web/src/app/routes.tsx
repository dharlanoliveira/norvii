import { Route, Routes } from "react-router-dom";

import { CorpusCatalogPage } from "../features/catalog/CorpusCatalogPage";
import { CorpusWorkspacePage } from "../features/workspace/CorpusWorkspacePage";
import { AppShell } from "./AppShell";

export function AppRoutes() {
  return (
    <Routes>
      <Route element={<AppShell />}>
        <Route index element={<CorpusCatalogPage />} />
        <Route path="corpora/:corpusId" element={<CorpusWorkspacePage />} />
      </Route>
    </Routes>
  );
}
