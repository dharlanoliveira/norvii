import { useCallback, useMemo, useState } from "react";

import type {
  CitationFixture,
  CorpusFixture,
  CorpusSource,
  ViewerState,
  WorkspaceMode,
} from "../../fixtures/models";

export interface WorkspaceController {
  readonly mode: WorkspaceMode;
  readonly viewer: ViewerState;
  readonly selectedSource: CorpusSource | null;
  readonly selectMode: (mode: WorkspaceMode) => void;
  readonly selectSource: (sourceId: string) => void;
  readonly selectSection: (sectionId: string) => void;
  readonly openCitation: (citation: CitationFixture) => void;
}

export function useWorkspaceController(
  corpus: CorpusFixture,
): WorkspaceController {
  const [mode, setMode] = useState<WorkspaceMode>("chat");
  const [viewer, setViewer] = useState<ViewerState>({
    sourceId: null,
    sectionId: null,
  });

  const selectedSource = useMemo(
    () =>
      corpus.sources.find((source) => source.id === viewer.sourceId) ?? null,
    [corpus.sources, viewer.sourceId],
  );

  const selectSource = useCallback(
    (sourceId: string) => {
      const source = corpus.sources.find(
        (candidate) => candidate.id === sourceId,
      );
      if (!source) {
        return;
      }
      setViewer({ sourceId, sectionId: source.sections[0]?.id ?? null });
      setMode("source");
    },
    [corpus.sources],
  );

  const selectSection = useCallback((sectionId: string) => {
    setViewer((current) => ({ ...current, sectionId }));
  }, []);

  const openCitation = useCallback(
    (citation: CitationFixture) => {
      const source = corpus.sources.find(
        (candidate) => candidate.id === citation.sourceId,
      );
      if (
        !source ||
        !source.sections.some((section) => section.id === citation.sectionId)
      ) {
        return;
      }
      setViewer({ sourceId: citation.sourceId, sectionId: citation.sectionId });
      setMode("source");
    },
    [corpus.sources],
  );

  return {
    mode,
    viewer,
    selectedSource,
    selectMode: setMode,
    selectSource,
    selectSection,
    openCitation,
  };
}
