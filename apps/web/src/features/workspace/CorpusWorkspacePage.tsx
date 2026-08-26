import {
  ArrowLeft,
  ExternalLink,
  FileText,
  FileUp,
  Languages,
  Link2,
} from "lucide-react";
import { type RefObject, useEffect, useRef, useState } from "react";
import { useTranslation } from "react-i18next";
import { Link, useParams } from "react-router-dom";

import type {
  ActiveSnapshotResponse,
  CorpusOpeningSuggestion,
  CorpusOpeningSuggestionResponse,
  CorpusResponse,
  DocumentResponse,
  GraphReleaseResponse,
  SnapshotResponse,
  SourceResponse,
} from "../../api/contract";
import type { ChatProvider, ChatReference } from "../../api/chat";
import type {
  ResearchProvider,
  UrlSourceDraft,
} from "../../research/domain/authoritative";
import { UrlSourceForm } from "../source-management/UrlSourceForm";
import { PdfSourceForm } from "../source-management/PdfSourceForm";
import { SourceStatus } from "../source-management/SourceStatus";
import { LegalDocumentReader } from "./LegalDocumentReader";
import { resolveVisibleUnitId, type CitedRange } from "./citationLocation";
import { ResearchChat } from "./ResearchChat";
import { SourceSelectionPrompt } from "./SourceSelectionPrompt";
import {
  WorkspaceModeSelector,
  type WorkspaceMode,
} from "./WorkspaceModeSelector";
import "./workspace.css";

interface CorpusWorkspacePageProps {
  readonly provider: ResearchProvider;
  readonly chatProvider?: ChatProvider | undefined;
}

type WorkspaceState =
  | { readonly status: "loading" }
  | {
      readonly status: "ready";
      readonly corpus: CorpusResponse;
      readonly sources: readonly SourceResponse[];
    }
  | { readonly status: "failed" };

type DocumentState =
  | { readonly status: "idle" }
  | { readonly status: "loading" }
  | {
      readonly status: "ready";
      readonly document: DocumentResponse;
      readonly citedRange?: CitedRange | undefined;
    }
  | { readonly status: "failed" };

interface OpeningSuggestionIdentity {
  readonly corpusId: string;
  readonly interfaceLanguage: "en" | "pt";
  readonly snapshotId: string;
  readonly snapshotManifestSha256: string;
  readonly snapshotReleaseVersion: number;
}

interface OpeningSuggestionState {
  readonly identity?: OpeningSuggestionIdentity | undefined;
  readonly suggestions: readonly CorpusOpeningSuggestion[];
}

interface CitationTarget {
  readonly citedRange: CitedRange;
  readonly documentVersionId: string;
  readonly source: SourceResponse;
}

interface ResolvedCitationDocument {
  readonly citedRange: CitedRange;
  readonly document: DocumentResponse;
  readonly selectedUnitId: string;
}

export function CorpusWorkspacePage({
  provider,
  chatProvider,
}: CorpusWorkspacePageProps) {
  const { corpusId = "" } = useParams();
  return (
    <LoadedCorpusWorkspace
      key={corpusId}
      provider={provider}
      chatProvider={chatProvider}
      corpusId={corpusId}
    />
  );
}

interface LoadedCorpusWorkspaceProps extends CorpusWorkspacePageProps {
  readonly corpusId: string;
}

function LoadedCorpusWorkspace({
  provider,
  corpusId,
  chatProvider,
}: LoadedCorpusWorkspaceProps) {
  const { t, i18n } = useTranslation();
  const {
    registerCreatedSource,
    replaceCorpus,
    replaceSource,
    replaceSources,
    state,
  } = useCorpusData(provider, corpusId);
  const sourceForms = useSourceForms(provider, corpusId, registerCreatedSource);
  const [mode, setMode] = useState<WorkspaceMode>("chat");
  const sourceViewer = useSourceViewer(provider, corpusId, () => {
    setMode("source");
  });
  const interfaceLanguage: "en" | "pt" = i18n.resolvedLanguage?.startsWith("pt")
    ? "pt"
    : "en";
  const openingSuggestions = useCorpusOpeningSuggestions(
    provider,
    corpusId,
    state.status === "ready" ? state.corpus.activeSnapshot : undefined,
    interfaceLanguage,
  );

  if (state.status === "loading")
    return <output>{t("workspace.loading")}</output>;
  if (state.status === "failed")
    return <p role="alert">{t("workspace.loadFailed")}</p>;

  const selectedSource = state.sources.find(
    (source) => source.id === sourceViewer.selectedSourceId,
  );
  const availableSource = state.sources[0];
  return (
    <section className="workspace-page" aria-labelledby="workspace-title">
      <WorkspaceHeader corpus={state.corpus} />
      <div className="workspace-frame">
        <SourceLibrary
          addSourceRef={sourceForms.addSourceRef}
          onCreatePdfSource={sourceForms.createPdfSource}
          onCreateUrlSource={sourceForms.createUrlSource}
          onSelectSource={sourceViewer.selectSource}
          onTogglePdfForm={sourceForms.togglePdfForm}
          onToggleUrlForm={sourceForms.toggleUrlForm}
          selectedSourceId={sourceViewer.selectedSourceId}
          showPdfForm={sourceForms.showPdfForm}
          showUrlForm={sourceForms.showUrlForm}
          sources={state.sources}
        />
        <WorkspacePrimary
          activeSnapshot={state.corpus.activeSnapshot}
          availableSource={availableSource}
          chatProvider={chatProvider}
          citationUnavailable={sourceViewer.citationUnavailable}
          corpusId={corpusId}
          documentState={sourceViewer.documentState}
          mode={mode}
          onAddSource={sourceForms.showAddSourceForm}
          onLoadSnapshotHistory={(signal) =>
            provider.listSnapshots(corpusId, signal)
          }
          onLoadGraphRelease={(snapshotId, signal) =>
            provider.getGraphRelease(corpusId, snapshotId, signal)
          }
          onModeChange={setMode}
          onReferenceSelect={(reference) =>
            sourceViewer.selectReference(state.sources, reference)
          }
          onSelectSource={sourceViewer.selectSource}
          onSelectUnit={sourceViewer.selectUnit}
          openingSuggestions={openingSuggestions}
          onReprocess={async (source, signal) => {
            const updated = await provider.reprocessSource(
              corpusId,
              source.id,
              source.version,
              signal,
            );
            replaceSource(updated);
          }}
          onPublish={async (source, signal) => {
            const activeSnapshot = state.corpus.activeSnapshot;
            if (
              activeSnapshot === null ||
              activeSnapshot === undefined ||
              source.latestReadyDocumentId === null
            ) {
              return;
            }
            await provider.publishSnapshot(
              corpusId,
              source.id,
              source.latestReadyDocumentId,
              activeSnapshot.releaseVersion,
              signal,
            );
            const [corpus, sources] = await Promise.all([
              provider.getCorpus(corpusId, signal),
              provider.listSources(corpusId, signal),
            ]);
            replaceCorpus(corpus);
            replaceSources(sources);
          }}
          onRetry={async (source, signal) => {
            const updated = await provider.retrySource(
              corpusId,
              source.id,
              source.version,
              signal,
            );
            replaceSource(updated);
          }}
          selectedSource={selectedSource}
          selectedUnitId={sourceViewer.selectedUnitId}
        />
      </div>
    </section>
  );
}

function useCorpusOpeningSuggestions(
  provider: ResearchProvider,
  corpusId: string,
  activeSnapshot: ActiveSnapshotResponse | null | undefined,
  interfaceLanguage: "en" | "pt",
): readonly CorpusOpeningSuggestion[] {
  const [state, setState] = useState<OpeningSuggestionState>({
    suggestions: [],
  });
  const identity = openingSuggestionIdentity(
    corpusId,
    activeSnapshot,
    interfaceLanguage,
  );

  useEffect(() => {
    const requestIdentity = openingSuggestionIdentity(
      corpusId,
      activeSnapshot,
      interfaceLanguage,
    );
    if (requestIdentity === undefined) {
      return;
    }

    const controller = new AbortController();
    void provider
      .getCorpusOpeningSuggestions(
        requestIdentity.corpusId,
        requestIdentity.interfaceLanguage,
        controller.signal,
      )
      .then((response) => {
        if (
          !controller.signal.aborted &&
          openingSuggestionsMatchWorkspace(response, requestIdentity)
        ) {
          setState({
            identity: requestIdentity,
            suggestions: response.suggestions,
          });
        }
      })
      .catch(() => {
        // An unavailable suggestion projection intentionally leaves the empty chat unadorned.
      });

    return () => controller.abort();
  }, [activeSnapshot, corpusId, interfaceLanguage, provider]);

  return openingSuggestionIdentitiesMatch(state.identity, identity)
    ? state.suggestions
    : [];
}

function openingSuggestionIdentity(
  corpusId: string,
  activeSnapshot: ActiveSnapshotResponse | null | undefined,
  interfaceLanguage: "en" | "pt",
): OpeningSuggestionIdentity | undefined {
  if (activeSnapshot === null || activeSnapshot === undefined) {
    return undefined;
  }

  return {
    corpusId,
    interfaceLanguage,
    snapshotId: activeSnapshot.id,
    snapshotManifestSha256: activeSnapshot.manifestSha256,
    snapshotReleaseVersion: activeSnapshot.releaseVersion,
  };
}

function openingSuggestionsMatchWorkspace(
  response: CorpusOpeningSuggestionResponse,
  identity: OpeningSuggestionIdentity,
): boolean {
  return (
    response.corpusId === identity.corpusId &&
    response.interfaceLanguage === identity.interfaceLanguage &&
    response.activeSnapshotId === identity.snapshotId &&
    response.activeSnapshotManifestSha256 === identity.snapshotManifestSha256
  );
}

function openingSuggestionIdentitiesMatch(
  first: OpeningSuggestionIdentity | undefined,
  second: OpeningSuggestionIdentity | undefined,
): boolean {
  return (
    first !== undefined &&
    second !== undefined &&
    first.corpusId === second.corpusId &&
    first.interfaceLanguage === second.interfaceLanguage &&
    first.snapshotId === second.snapshotId &&
    first.snapshotManifestSha256 === second.snapshotManifestSha256 &&
    first.snapshotReleaseVersion === second.snapshotReleaseVersion
  );
}

function useCorpusData(provider: ResearchProvider, corpusId: string) {
  const [state, setState] = useState<WorkspaceState>({ status: "loading" });
  const [pollSources, setPollSources] = useState(false);

  useEffect(() => {
    const controller = new AbortController();
    void Promise.all([
      provider.getCorpus(corpusId, controller.signal),
      provider.listSources(corpusId, controller.signal),
    ])
      .then(([corpus, sources]) => {
        setState({ status: "ready", corpus, sources });
        setPollSources(sources.some(isActiveSource));
      })
      .catch(() => {
        if (!controller.signal.aborted) setState({ status: "failed" });
      });
    return () => controller.abort();
  }, [corpusId, provider]);

  useEffect(() => {
    if (!pollSources) return;
    let cancelled = false;
    let attempts = 0;
    let timer: ReturnType<typeof setTimeout> | undefined;
    const poll = (): void => {
      timer = setTimeout(async () => {
        const controller = new AbortController();
        try {
          const sources = await provider.listSources(
            corpusId,
            controller.signal,
          );
          if (cancelled) return;
          attempts += 1;
          setState((current) => replaceWorkspaceSources(current, sources));
          if (attempts < 90 && sources.some(isActiveSource)) poll();
          else setPollSources(false);
        } catch {
          setPollSources(false);
        }
      }, 1000);
    };
    poll();
    return () => {
      cancelled = true;
      if (timer !== undefined) clearTimeout(timer);
    };
  }, [corpusId, pollSources, provider]);

  const registerCreatedSource = (source: SourceResponse): void => {
    setState((current) =>
      current.status === "ready"
        ? { ...current, sources: [...current.sources, source] }
        : current,
    );
    setPollSources(true);
  };
  const replaceSource = (updated: SourceResponse): void => {
    setState((current) =>
      current.status === "ready"
        ? {
            ...current,
            sources: current.sources.map((source) =>
              source.id === updated.id ? updated : source,
            ),
          }
        : current,
    );
    setPollSources(isActiveSource(updated));
  };
  const replaceCorpus = (updated: CorpusResponse): void => {
    setState((current) =>
      current.status === "ready" ? { ...current, corpus: updated } : current,
    );
  };
  const replaceSources = (sources: readonly SourceResponse[]): void => {
    setState((current) => replaceWorkspaceSources(current, sources));
  };

  return {
    registerCreatedSource,
    replaceCorpus,
    replaceSource,
    replaceSources,
    state,
  };
}

function useSourceForms(
  provider: ResearchProvider,
  corpusId: string,
  registerCreatedSource: (source: SourceResponse) => void,
) {
  const [showUrlForm, setShowUrlForm] = useState(false);
  const [showPdfForm, setShowPdfForm] = useState(false);
  const addSourceRef = useRef<HTMLButtonElement>(null);

  const showAddSourceForm = (): void => {
    setShowPdfForm(false);
    setShowUrlForm(true);
    requestAnimationFrame(() => addSourceRef.current?.focus());
  };
  const toggleUrlForm = (): void => {
    setShowPdfForm(false);
    setShowUrlForm((visible) => !visible);
  };
  const togglePdfForm = (): void => {
    setShowUrlForm(false);
    setShowPdfForm((visible) => !visible);
  };
  const createUrlSource = async (
    draft: UrlSourceDraft,
    signal: AbortSignal,
  ): Promise<void> => {
    const source = await provider.createUrlSource(corpusId, draft, signal);
    registerCreatedSource(source);
    setShowUrlForm(false);
  };
  const createPdfSource = async (
    title: string,
    file: File,
    signal: AbortSignal,
  ): Promise<void> => {
    const source = await provider.createPdfSource(
      corpusId,
      title,
      file,
      signal,
    );
    registerCreatedSource(source);
    setShowPdfForm(false);
  };

  return {
    addSourceRef,
    createPdfSource,
    createUrlSource,
    showAddSourceForm,
    showPdfForm,
    showUrlForm,
    togglePdfForm,
    toggleUrlForm,
  };
}

function useSourceViewer(
  provider: ResearchProvider,
  corpusId: string,
  activateSourceMode: () => void,
) {
  const [selectedSourceId, setSelectedSourceId] = useState<string>();
  const [documentState, setDocumentState] = useState<DocumentState>({
    status: "idle",
  });
  const [selectedUnitId, setSelectedUnitId] = useState<string>();
  const [citationUnavailable, setCitationUnavailable] = useState(false);

  const selectSource = (source: SourceResponse, unitLocator?: string): void => {
    setCitationUnavailable(false);
    setSelectedSourceId(source.id);
    activateSourceMode();
    if (source.processingStatus !== "ready") {
      setDocumentState({ status: "idle" });
      return;
    }
    const controller = new AbortController();
    setDocumentState({ status: "loading" });
    void provider
      .getDocument(corpusId, source.id, controller.signal)
      .then((document) => {
        setDocumentState({ status: "ready", document });
        setSelectedUnitId(
          document.units.find((unit) => unit.locator === unitLocator)?.id ??
            document.units.find((unit) => unit.kind !== "document")?.id ??
            document.units[0]?.id,
        );
      })
      .catch(() => {
        if (!controller.signal.aborted) setDocumentState({ status: "failed" });
      });
  };
  const selectReference = (
    sources: readonly SourceResponse[],
    reference: ChatReference,
  ): void => {
    const target = resolveCitationTarget(corpusId, sources, reference);
    if (target === undefined) {
      setCitationUnavailable(true);
      return;
    }
    setSelectedSourceId(target.source.id);
    activateSourceMode();
    setCitationUnavailable(false);
    setDocumentState({ status: "loading" });
    const controller = new AbortController();
    void provider
      .getDocumentVersion(
        corpusId,
        target.source.id,
        target.documentVersionId,
        controller.signal,
      )
      .then((document) => {
        const resolvedDocument = resolveCitationDocument(
          document,
          reference.unitLocator,
          target.citedRange,
        );
        if (
          resolvedDocument === undefined ||
          document.id !== target.documentVersionId
        ) {
          setCitationUnavailable(true);
          setDocumentState({ status: "idle" });
          return;
        }
        setSelectedUnitId(resolvedDocument.selectedUnitId);
        setDocumentState({
          status: "ready",
          document: resolvedDocument.document,
          citedRange: resolvedDocument.citedRange,
        });
      })
      .catch(() => {
        if (!controller.signal.aborted) {
          setCitationUnavailable(true);
          setDocumentState({ status: "idle" });
        }
      });
  };
  const selectUnit = (unitId: string): void => {
    setSelectedUnitId(unitId);
    setDocumentState((current) =>
      current.status === "ready" && current.citedRange !== undefined
        ? { ...current, citedRange: undefined }
        : current,
    );
  };

  return {
    citationUnavailable,
    documentState,
    selectedSourceId,
    selectedUnitId,
    selectReference,
    selectSource,
    selectUnit,
  };
}

function WorkspaceHeader({ corpus }: { readonly corpus: CorpusResponse }) {
  const { t } = useTranslation();
  return (
    <header className="workspace-heading">
      <div>
        <Link className="workspace-heading__back" to="/">
          <ArrowLeft aria-hidden="true" size={15} />
          {t("workspace.backToCatalog")}
        </Link>
        <p className="kicker">{t("workspace.activeCorpus")}</p>
        <h1 id="workspace-title">{corpus.name}</h1>
      </div>
      <div className="workspace-heading__meta">
        <Languages aria-hidden="true" size={15} />
        <span>{t(`language.${corpus.language}`)}</span>
        <span>{corpus.jurisdiction}</span>
      </div>
    </header>
  );
}

interface SourceLibraryProps {
  readonly addSourceRef: RefObject<HTMLButtonElement | null>;
  readonly onCreatePdfSource: (
    title: string,
    file: File,
    signal: AbortSignal,
  ) => Promise<void>;
  readonly onCreateUrlSource: (
    draft: UrlSourceDraft,
    signal: AbortSignal,
  ) => Promise<void>;
  readonly onSelectSource: (
    source: SourceResponse,
    unitLocator?: string,
  ) => void;
  readonly onTogglePdfForm: () => void;
  readonly onToggleUrlForm: () => void;
  readonly selectedSourceId?: string | undefined;
  readonly showPdfForm: boolean;
  readonly showUrlForm: boolean;
  readonly sources: readonly SourceResponse[];
}

function SourceLibrary({
  addSourceRef,
  onCreatePdfSource,
  onCreateUrlSource,
  onSelectSource,
  onTogglePdfForm,
  onToggleUrlForm,
  selectedSourceId,
  showPdfForm,
  showUrlForm,
  sources,
}: SourceLibraryProps) {
  const { t } = useTranslation();
  return (
    <aside className="source-library" aria-labelledby="source-library-title">
      <header>
        <p>{t("workspace.library")}</p>
        <span id="source-library-title">
          {t("workspace.libraryDescription")}
        </span>
      </header>
      <div className="source-library__actions">
        <button
          type="button"
          ref={addSourceRef}
          aria-controls="url-source-form"
          aria-expanded={showUrlForm}
          onClick={onToggleUrlForm}
        >
          <Link2 aria-hidden="true" size={15} />
          {t("sourceManagement.addUrl")}
        </button>
        <button
          type="button"
          aria-controls="pdf-source-form"
          aria-expanded={showPdfForm}
          onClick={onTogglePdfForm}
        >
          <FileUp aria-hidden="true" size={15} />
          {t("sourceManagement.addPdf")}
        </button>
      </div>
      {showUrlForm ? <UrlSourceForm onSubmit={onCreateUrlSource} /> : null}
      {showPdfForm ? <PdfSourceForm onSubmit={onCreatePdfSource} /> : null}
      {sources.length === 0 ? (
        <output>{t("workspace.emptySources")}</output>
      ) : null}
      <div className="source-tree" role="tree" aria-label={t("tree.label")}>
        {sources.map((source) => (
          <SourceTreeItem
            key={source.id}
            onSelect={onSelectSource}
            selected={source.id === selectedSourceId}
            source={source}
          />
        ))}
      </div>
    </aside>
  );
}

function SourceTreeItem({
  onSelect,
  selected,
  source,
}: {
  readonly onSelect: (source: SourceResponse) => void;
  readonly selected: boolean;
  readonly source: SourceResponse;
}) {
  const { t } = useTranslation();
  const statusLabel = t(`sourceStatus.${source.processingStatus}`);
  return (
    <button
      type="button"
      role="treeitem"
      aria-selected={selected}
      aria-label={`${source.title} (${statusLabel})`}
      className="source-tree__source"
      onClick={() => onSelect(source)}
    >
      {source.kind === "pdf" ? (
        <FileText aria-hidden="true" size={15} />
      ) : (
        <ExternalLink aria-hidden="true" size={15} />
      )}
      <span>{source.title}</span>
      <small>{statusLabel}</small>
    </button>
  );
}

interface WorkspacePrimaryProps {
  readonly activeSnapshot?: CorpusResponse["activeSnapshot"];
  readonly availableSource?: SourceResponse | undefined;
  readonly chatProvider?: ChatProvider | undefined;
  readonly citationUnavailable: boolean;
  readonly corpusId: string;
  readonly documentState: DocumentState;
  readonly mode: WorkspaceMode;
  readonly onAddSource: () => void;
  readonly onLoadSnapshotHistory: (
    signal: AbortSignal,
  ) => Promise<readonly SnapshotResponse[]>;
  readonly onLoadGraphRelease: (
    snapshotId: string,
    signal: AbortSignal,
  ) => Promise<GraphReleaseResponse>;
  readonly onModeChange: (mode: WorkspaceMode) => void;
  readonly onReferenceSelect: (reference: ChatReference) => void;
  readonly onReprocess: (
    source: SourceResponse,
    signal: AbortSignal,
  ) => Promise<void>;
  readonly onPublish: (
    source: SourceResponse,
    signal: AbortSignal,
  ) => Promise<void>;
  readonly onRetry: (
    source: SourceResponse,
    signal: AbortSignal,
  ) => Promise<void>;
  readonly onSelectSource: (source: SourceResponse) => void;
  readonly onSelectUnit: (unitId: string) => void;
  readonly openingSuggestions: readonly CorpusOpeningSuggestion[];
  readonly selectedSource?: SourceResponse | undefined;
  readonly selectedUnitId?: string | undefined;
}

function WorkspacePrimary({
  activeSnapshot,
  availableSource,
  chatProvider,
  citationUnavailable,
  corpusId,
  documentState,
  mode,
  onAddSource,
  onLoadSnapshotHistory,
  onLoadGraphRelease,
  onModeChange,
  onReferenceSelect,
  onReprocess,
  onPublish,
  onRetry,
  onSelectSource,
  onSelectUnit,
  openingSuggestions,
  selectedSource,
  selectedUnitId,
}: WorkspacePrimaryProps) {
  return (
    <section className="workspace-primary">
      <header className="workspace-primary__toolbar">
        <WorkspaceModeSelector mode={mode} onChange={onModeChange} />
      </header>
      <div id="chat-panel" role="tabpanel" hidden={mode !== "chat"}>
        <ResearchChat
          corpusId={corpusId}
          openingSuggestions={openingSuggestions}
          provider={chatProvider}
          onReferenceSelect={onReferenceSelect}
        />
      </div>
      <SourcePanel
        activeSnapshot={activeSnapshot}
        availableSource={availableSource}
        citationUnavailable={citationUnavailable}
        documentState={documentState}
        hidden={mode !== "source"}
        onAddSource={onAddSource}
        onLoadSnapshotHistory={onLoadSnapshotHistory}
        onLoadGraphRelease={onLoadGraphRelease}
        onReprocess={onReprocess}
        onPublish={onPublish}
        onRetry={onRetry}
        onSelectSource={onSelectSource}
        onSelectUnit={onSelectUnit}
        selectedSource={selectedSource}
        selectedUnitId={selectedUnitId}
      />
    </section>
  );
}

interface SourcePanelProps {
  readonly activeSnapshot?: CorpusResponse["activeSnapshot"];
  readonly availableSource?: SourceResponse | undefined;
  readonly citationUnavailable: boolean;
  readonly documentState: DocumentState;
  readonly hidden: boolean;
  readonly onAddSource: () => void;
  readonly onLoadSnapshotHistory: (
    signal: AbortSignal,
  ) => Promise<readonly SnapshotResponse[]>;
  readonly onLoadGraphRelease: (
    snapshotId: string,
    signal: AbortSignal,
  ) => Promise<GraphReleaseResponse>;
  readonly onReprocess: (
    source: SourceResponse,
    signal: AbortSignal,
  ) => Promise<void>;
  readonly onPublish: (
    source: SourceResponse,
    signal: AbortSignal,
  ) => Promise<void>;
  readonly onRetry: (
    source: SourceResponse,
    signal: AbortSignal,
  ) => Promise<void>;
  readonly onSelectSource: (source: SourceResponse) => void;
  readonly onSelectUnit: (unitId: string) => void;
  readonly selectedSource?: SourceResponse | undefined;
  readonly selectedUnitId?: string | undefined;
}

function SourcePanel({
  activeSnapshot,
  availableSource,
  citationUnavailable,
  documentState,
  hidden,
  onAddSource,
  onLoadSnapshotHistory,
  onLoadGraphRelease,
  onReprocess,
  onPublish,
  onRetry,
  onSelectSource,
  onSelectUnit,
  selectedSource,
  selectedUnitId,
}: SourcePanelProps) {
  const { t } = useTranslation();
  return (
    <div
      id="source-panel"
      role="tabpanel"
      hidden={hidden}
      className={
        selectedSource === undefined ? "source-panel--empty" : undefined
      }
    >
      {selectedSource === undefined ? (
        <SourceSelectionPrompt
          sourceTitle={availableSource?.title}
          onOpenSource={
            availableSource === undefined
              ? undefined
              : () => onSelectSource(availableSource)
          }
          onAddSource={onAddSource}
        />
      ) : (
        <SelectedSourceContent
          activeSnapshot={activeSnapshot}
          citationUnavailable={citationUnavailable}
          documentState={documentState}
          onLoadSnapshotHistory={onLoadSnapshotHistory}
          onLoadGraphRelease={onLoadGraphRelease}
          onReprocess={onReprocess}
          onPublish={onPublish}
          onRetry={onRetry}
          onSelectUnit={onSelectUnit}
          selectedSource={selectedSource}
          selectedUnitId={selectedUnitId}
        />
      )}
      {selectedSource === undefined && citationUnavailable ? (
        <p role="alert">{t("errors.unavailableCitation")}</p>
      ) : null}
    </div>
  );
}

function SelectedSourceContent({
  activeSnapshot,
  citationUnavailable,
  documentState,
  onLoadSnapshotHistory,
  onLoadGraphRelease,
  onReprocess,
  onPublish,
  onRetry,
  onSelectUnit,
  selectedSource,
  selectedUnitId,
}: Omit<
  SourcePanelProps,
  "availableSource" | "hidden" | "onAddSource" | "onSelectSource"
> & {
  readonly selectedSource: SourceResponse;
}) {
  const { t } = useTranslation();
  return (
    <>
      <SourceStatus
        source={selectedSource}
        activeSnapshot={activeSnapshot}
        onLoadSnapshotHistory={onLoadSnapshotHistory}
        onLoadGraphRelease={onLoadGraphRelease}
        onPublish={onPublish}
        onRetry={onRetry}
        onReprocess={onReprocess}
      />
      {selectedSource.processingStatus !== "ready" ? (
        <output>{t(`sourceStatus.${selectedSource.processingStatus}`)}</output>
      ) : null}
      {documentState.status === "loading" ? (
        <output>{t("viewer.loading")}</output>
      ) : null}
      {documentState.status === "failed" ? (
        <p role="alert">{t("viewer.loadFailed")}</p>
      ) : null}
      {citationUnavailable ? (
        <p role="alert">{t("errors.unavailableCitation")}</p>
      ) : null}
      {documentState.status === "ready" ? (
        <div className="source-document">
          <LegalDocumentReader
            document={documentState.document}
            selectedUnitId={selectedUnitId}
            onSelect={onSelectUnit}
            citedRange={documentState.citedRange}
          />
        </div>
      ) : null}
    </>
  );
}

function replaceWorkspaceSources(
  state: WorkspaceState,
  sources: readonly SourceResponse[],
): WorkspaceState {
  return state.status === "ready" ? { ...state, sources } : state;
}

function resolveCitationTarget(
  corpusId: string,
  sources: readonly SourceResponse[],
  reference: ChatReference,
): CitationTarget | undefined {
  if (
    reference.corpusId !== corpusId ||
    reference.documentVersionId === undefined
  ) {
    return undefined;
  }
  const source = sources.find(
    (candidate) => candidate.id === reference.sourceId,
  );
  if (source === undefined) return undefined;
  return {
    source,
    documentVersionId: reference.documentVersionId,
    citedRange: {
      startOffset: reference.startOffset,
      endOffset: reference.endOffset,
    },
  };
}

function resolveCitationDocument(
  document: DocumentResponse,
  unitLocator: string,
  citedRange: CitedRange,
): ResolvedCitationDocument | undefined {
  const selectedUnitId = resolveVisibleUnitId(
    document,
    unitLocator,
    citedRange,
  );
  if (selectedUnitId === undefined) return undefined;
  return { document, citedRange, selectedUnitId };
}

function isActiveSource(source: SourceResponse): boolean {
  return (
    source.processingStatus === "pending" ||
    source.processingStatus === "processing"
  );
}
