import {
  ArrowLeft,
  ExternalLink,
  FileText,
  FileUp,
  Languages,
  Link2,
} from "lucide-react";
import { useEffect, useState } from "react";
import { useTranslation } from "react-i18next";
import { Link, useParams } from "react-router-dom";

import type {
  CorpusResponse,
  DocumentResponse,
  SourceResponse,
} from "../../api/contract";
import type { ResearchProvider } from "../../research/domain/authoritative";
import { UrlSourceForm } from "../source-management/UrlSourceForm";
import { PdfSourceForm } from "../source-management/PdfSourceForm";
import { SourceStatus } from "../source-management/SourceStatus";
import { LegalDocumentReader } from "./LegalDocumentReader";
import { ResearchChat } from "./ResearchChat";
import { SourceSelectionPrompt } from "./SourceSelectionPrompt";
import {
  WorkspaceModeSelector,
  type WorkspaceMode,
} from "./WorkspaceModeSelector";
import "./workspace.css";

interface CorpusWorkspacePageProps {
  readonly provider: ResearchProvider;
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
  | { readonly status: "ready"; readonly document: DocumentResponse }
  | { readonly status: "failed" };

export function CorpusWorkspacePage({ provider }: CorpusWorkspacePageProps) {
  const { corpusId = "" } = useParams();
  return (
    <LoadedCorpusWorkspace
      key={corpusId}
      provider={provider}
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
}: LoadedCorpusWorkspaceProps) {
  const { t } = useTranslation();
  const [state, setState] = useState<WorkspaceState>({ status: "loading" });
  const [mode, setMode] = useState<WorkspaceMode>("chat");
  const [selectedSourceId, setSelectedSourceId] = useState<string>();
  const [documentState, setDocumentState] = useState<DocumentState>({
    status: "idle",
  });
  const [showUrlForm, setShowUrlForm] = useState(false);
  const [showPdfForm, setShowPdfForm] = useState(false);
  const [pollSources, setPollSources] = useState(false);
  const [selectedUnitId, setSelectedUnitId] = useState<string>();

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

  const selectSource = (source: SourceResponse): void => {
    setSelectedSourceId(source.id);
    setMode("source");
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
          document.units.find((unit) => unit.kind !== "document")?.id ??
            document.units[0]?.id,
        );
      })
      .catch(() => {
        if (!controller.signal.aborted) setDocumentState({ status: "failed" });
      });
  };

  if (state.status === "loading")
    return <output>{t("workspace.loading")}</output>;
  if (state.status === "failed")
    return <p role="alert">{t("workspace.loadFailed")}</p>;

  const selectedSource = state.sources.find(
    (source) => source.id === selectedSourceId,
  );
  return (
    <section className="workspace-page" aria-labelledby="workspace-title">
      <header className="workspace-heading">
        <div>
          <Link className="workspace-heading__back" to="/">
            <ArrowLeft aria-hidden="true" size={15} />
            {t("workspace.backToCatalog")}
          </Link>
          <p className="kicker">{t("workspace.activeCorpus")}</p>
          <h1 id="workspace-title">{state.corpus.name}</h1>
        </div>
        <div className="workspace-heading__meta">
          <Languages aria-hidden="true" size={15} />
          <span>{t(`language.${state.corpus.language}`)}</span>
          <span>{state.corpus.jurisdiction}</span>
        </div>
      </header>
      <div className="workspace-frame">
        <aside
          className="source-library"
          aria-labelledby="source-library-title"
        >
          <header>
            <p>{t("workspace.library")}</p>
            <span id="source-library-title">
              {t("workspace.libraryDescription")}
            </span>
          </header>
          <div className="source-library__actions">
            <button
              type="button"
              aria-controls="url-source-form"
              aria-expanded={showUrlForm}
              onClick={() => {
                setShowPdfForm(false);
                setShowUrlForm((visible) => !visible);
              }}
            >
              <Link2 aria-hidden="true" size={15} />
              {t("sourceManagement.addUrl")}
            </button>
            <button
              type="button"
              aria-controls="pdf-source-form"
              aria-expanded={showPdfForm}
              onClick={() => {
                setShowUrlForm(false);
                setShowPdfForm((visible) => !visible);
              }}
            >
              <FileUp aria-hidden="true" size={15} />
              {t("sourceManagement.addPdf")}
            </button>
          </div>
          {showUrlForm ? (
            <UrlSourceForm
              onSubmit={async (draft, signal) => {
                const created = await provider.createUrlSource(
                  corpusId,
                  draft,
                  signal,
                );
                setState((current) =>
                  current.status === "ready"
                    ? { ...current, sources: [...current.sources, created] }
                    : current,
                );
                setPollSources(true);
                setShowUrlForm(false);
              }}
            />
          ) : null}
          {showPdfForm ? (
            <PdfSourceForm
              onSubmit={async (title, file, signal) => {
                const created = await provider.createPdfSource(
                  corpusId,
                  title,
                  file,
                  signal,
                );
                setState((current) =>
                  current.status === "ready"
                    ? { ...current, sources: [...current.sources, created] }
                    : current,
                );
                setPollSources(true);
                setShowPdfForm(false);
              }}
            />
          ) : null}
          {state.sources.length === 0 ? (
            <output>{t("workspace.emptySources")}</output>
          ) : null}
          <div className="source-tree" role="tree" aria-label={t("tree.label")}>
            {state.sources.map((source) => {
              const statusLabel = t(`sourceStatus.${source.processingStatus}`);
              return (
                <button
                  type="button"
                  role="treeitem"
                  aria-selected={source.id === selectedSourceId}
                  aria-label={`${source.title} (${statusLabel})`}
                  className="source-tree__source"
                  key={source.id}
                  onClick={() => selectSource(source)}
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
            })}
          </div>
        </aside>
        <section className="workspace-primary">
          <header className="workspace-primary__toolbar">
            <WorkspaceModeSelector mode={mode} onChange={setMode} />
          </header>
          <div id="chat-panel" role="tabpanel" hidden={mode !== "chat"}>
            <ResearchChat />
          </div>
          <div
            id="source-panel"
            role="tabpanel"
            hidden={mode !== "source"}
            className={!selectedSource ? "source-panel--empty" : undefined}
          >
            {!selectedSource ? <SourceSelectionPrompt /> : null}
            {selectedSource ? (
              <SourceStatus
                source={selectedSource}
                onRetry={async (source, signal) => {
                  const updated = await provider.retrySource(
                    corpusId,
                    source.id,
                    source.version,
                    signal,
                  );
                  replaceSource(updated);
                }}
                onReprocess={async (source, signal) => {
                  const updated = await provider.reprocessSource(
                    corpusId,
                    source.id,
                    source.version,
                    signal,
                  );
                  replaceSource(updated);
                }}
              />
            ) : null}
            {selectedSource && selectedSource.processingStatus !== "ready" ? (
              <output>
                {t(`sourceStatus.${selectedSource.processingStatus}`)}
              </output>
            ) : null}
            {documentState.status === "loading" ? (
              <output>{t("viewer.loading")}</output>
            ) : null}
            {documentState.status === "failed" ? (
              <p role="alert">{t("viewer.loadFailed")}</p>
            ) : null}
            {documentState.status === "ready" ? (
              <div className="source-document">
                <LegalDocumentReader
                  document={documentState.document}
                  selectedUnitId={selectedUnitId}
                  onSelect={setSelectedUnitId}
                />
              </div>
            ) : null}
          </div>
        </section>
      </div>
    </section>
  );

  function replaceSource(updated: SourceResponse): void {
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
  }
}

function replaceWorkspaceSources(
  state: WorkspaceState,
  sources: readonly SourceResponse[],
): WorkspaceState {
  return state.status === "ready" ? { ...state, sources } : state;
}

function isActiveSource(source: SourceResponse): boolean {
  return (
    source.processingStatus === "pending" ||
    source.processingStatus === "processing"
  );
}
