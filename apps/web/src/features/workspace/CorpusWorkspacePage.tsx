import { ArrowLeft, Languages } from "lucide-react";
import { useState } from "react";
import { useTranslation } from "react-i18next";
import { Link, useParams } from "react-router-dom";

import type { Citation } from "../../research/domain/models";
import type { ResearchCatalog } from "../../research/domain/researchCatalog";
import { ResearchChat } from "./ResearchChat";
import { SourceTree } from "./SourceTree";
import { SourceViewer } from "./SourceViewer";
import { UnknownCorpusPage } from "./UnknownCorpusPage";
import { useWorkspaceController } from "./useWorkspaceController";
import { WorkspaceModeSelector } from "./WorkspaceModeSelector";
import "./workspace.css";

interface CorpusWorkspacePageProps {
  readonly catalog: ResearchCatalog;
}

export function CorpusWorkspacePage({ catalog }: CorpusWorkspacePageProps) {
  const { corpusId = "" } = useParams();
  const corpus = catalog.findCorpus(corpusId);

  if (!corpus) {
    return <UnknownCorpusPage />;
  }

  return (
    <ResolvedCorpusWorkspace
      key={corpus.id}
      catalog={catalog}
      corpus={corpus}
    />
  );
}

interface ResolvedCorpusWorkspaceProps {
  readonly catalog: ResearchCatalog;
  readonly corpus: NonNullable<ReturnType<ResearchCatalog["findCorpus"]>>;
}

function ResolvedCorpusWorkspace({
  catalog,
  corpus,
}: ResolvedCorpusWorkspaceProps) {
  const { t } = useTranslation();
  const workspace = useWorkspaceController(corpus);
  const [citationUnavailable, setCitationUnavailable] = useState(false);
  const handleOpenCitation = (citation: Citation): void => {
    const resolved = catalog.resolveCitation(corpus.id, citation);
    if (!resolved) {
      setCitationUnavailable(true);
      return;
    }
    setCitationUnavailable(false);
    workspace.openSourceLocation(resolved.source.id, resolved.location.id);
  };

  return (
    <section className="workspace-page" aria-labelledby="workspace-title">
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
          <SourceTree
            corpus={corpus}
            selectedSourceId={workspace.selectedSource?.id}
            onSelect={workspace.selectSource}
          />
        </aside>
        <section className="workspace-primary">
          <header className="workspace-primary__toolbar">
            <WorkspaceModeSelector
              mode={workspace.mode}
              onChange={workspace.setMode}
            />
          </header>
          {citationUnavailable ? (
            <p
              className="citation-warning"
              role="alert"
              aria-label={t("errors.unavailableCitation")}
            >
              {t("errors.unavailableCitation")}
            </p>
          ) : null}
          <div
            id="chat-panel"
            role="tabpanel"
            aria-label={t("workspace.chat")}
            hidden={workspace.mode !== "chat"}
          >
            <ResearchChat corpus={corpus} onOpenCitation={handleOpenCitation} />
          </div>
          <div
            id="source-panel"
            role="tabpanel"
            aria-label={t("workspace.source")}
            hidden={workspace.mode !== "source"}
          >
            <SourceViewer
              source={workspace.selectedSource}
              activeLocationId={workspace.activeLocationId}
              onLocationChange={workspace.changeLocation}
            />
          </div>
        </section>
      </div>
    </section>
  );
}
