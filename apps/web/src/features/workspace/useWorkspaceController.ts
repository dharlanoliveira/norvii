import { useCallback, useMemo, useState } from "react";

import type { Corpus, Source } from "../../research/domain/models";

export type WorkspaceMode = "chat" | "source";

export interface WorkspaceController {
  readonly mode: WorkspaceMode;
  readonly selectedSource: Source | undefined;
  readonly activeLocationId: string | undefined;
  readonly setMode: (mode: WorkspaceMode) => void;
  readonly selectSource: (sourceId: string) => void;
  readonly changeLocation: (locationId: string) => void;
  readonly openSourceLocation: (sourceId: string, locationId: string) => void;
}

export function useWorkspaceController(corpus: Corpus): WorkspaceController {
  const [mode, setMode] = useState<WorkspaceMode>("chat");
  const [selectedSourceId, setSelectedSourceId] = useState<string>();
  const [locationsBySource, setLocationsBySource] = useState<
    Readonly<Record<string, string>>
  >({});
  const selectedSource = useMemo(
    () => corpus.sources.find((source) => source.id === selectedSourceId),
    [corpus.sources, selectedSourceId],
  );
  const activeLocationId = selectedSource
    ? (locationsBySource[selectedSource.id] ?? selectedSource.locations[0]?.id)
    : undefined;

  const selectSource = useCallback(
    (sourceId: string) => {
      const source = corpus.sources.find(
        (candidate) => candidate.id === sourceId,
      );
      if (!source) return;

      setSelectedSourceId(source.id);
      setLocationsBySource((current) => {
        const initialLocation = source.locations[0]?.id;
        return current[source.id] || !initialLocation
          ? current
          : { ...current, [source.id]: initialLocation };
      });
      setMode("source");
    },
    [corpus.sources],
  );

  const changeLocation = useCallback(
    (locationId: string) => {
      if (!selectedSource) return;
      if (
        !selectedSource.locations.some((location) => location.id === locationId)
      ) {
        return;
      }
      setLocationsBySource((current) => ({
        ...current,
        [selectedSource.id]: locationId,
      }));
    },
    [selectedSource],
  );

  const openSourceLocation = useCallback(
    (sourceId: string, locationId: string) => {
      const source = corpus.sources.find(
        (candidate) => candidate.id === sourceId,
      );
      if (!source?.locations.some((location) => location.id === locationId)) {
        return;
      }
      setSelectedSourceId(source.id);
      setLocationsBySource((current) => ({
        ...current,
        [source.id]: locationId,
      }));
      setMode("source");
    },
    [corpus.sources],
  );

  return {
    mode,
    selectedSource,
    activeLocationId,
    setMode,
    selectSource,
    changeLocation,
    openSourceLocation,
  };
}
