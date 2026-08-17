import { ArrowLeft, Globe2, ShieldCheck } from "lucide-react";
import { useTranslation } from "react-i18next";
import { Link, useParams } from "react-router-dom";

import { findCorpus } from "../../fixtures/legal-content/corpora";
import { ResearchChat } from "./ResearchChat";
import { SourceTree } from "./SourceTree";
import { SourceViewer } from "./SourceViewer";
import { UnknownCorpusPage } from "./UnknownCorpusPage";
import { useWorkspaceController } from "./useWorkspaceController";
import { WorkspaceModeSelector } from "./WorkspaceModeSelector";
import "./workspace.css";

export function CorpusWorkspacePage() {
  const { corpusId } = useParams();
  const corpus = corpusId ? findCorpus(corpusId) : undefined;

  if (!corpus) {
    return <UnknownCorpusPage />;
  }

  return <CorpusWorkspace corpus={corpus} />;
}

function CorpusWorkspace({
  corpus,
}: {
  readonly corpus: NonNullable<ReturnType<typeof findCorpus>>;
}) {
  const { t } = useTranslation();
  const controller = useWorkspaceController(corpus);
  const languageLabel =
    corpus.language === "pt" ? t("language.portuguese") : t("language.english");

  return (
    <div className="workspace-page reveal">
      <div className="workspace-topline">
        <Link to="/" className="back-link">
          <ArrowLeft aria-hidden="true" size={15} />
          {t("navigation.backToCorpora")}
        </Link>
        <div className="corpus-boundary">
          <ShieldCheck aria-hidden="true" size={15} />
          <span>{t("workspace.corpusLabel")}</span>
          <strong>{corpus.label}</strong>
        </div>
      </div>

      <header className="workspace-titlebar">
        <div>
          <p className="eyebrow">{t(corpus.eyebrowKey)}</p>
          <h1>{corpus.label}</h1>
        </div>
        <div className="workspace-language">
          <Globe2 aria-hidden="true" size={16} />
          <span>{t("language.contentLanguage")}</span>
          <strong>{languageLabel}</strong>
        </div>
      </header>

      <div className="workspace-frame">
        <aside className="source-sidebar">
          <header className="source-sidebar-header">
            <span>01</span>
            <div>
              <h2>{t("workspace.sourcesLabel")}</h2>
              <p>{t("workspace.sourcesHint")}</p>
            </div>
          </header>
          <SourceTree
            corpus={corpus}
            selectedSourceId={controller.viewer.sourceId}
            onSelectSource={controller.selectSource}
          />
          <div className="sidebar-footnote">
            <span>{String(corpus.sources.length).padStart(2, "0")}</span>
            {t("catalog.sourceCount", { count: corpus.sources.length })}
          </div>
        </aside>

        <section className="research-surface">
          <header className="research-toolbar">
            <WorkspaceModeSelector
              mode={controller.mode}
              onChange={controller.selectMode}
            />
            {controller.selectedSource ? (
              <div className="active-source-label">
                <span>{t("workspace.sourceSelected")}</span>
                <strong>{controller.selectedSource.shortTitle}</strong>
              </div>
            ) : null}
          </header>
          <div
            className={
              controller.mode === "chat"
                ? "surface-panel active"
                : "surface-panel"
            }
            role="tabpanel"
            aria-hidden={controller.mode !== "chat"}
          >
            <ResearchChat
              key={corpus.id}
              corpus={corpus}
              onOpenCitation={controller.openCitation}
            />
          </div>
          <div
            className={
              controller.mode === "source"
                ? "surface-panel active"
                : "surface-panel"
            }
            role="tabpanel"
            aria-hidden={controller.mode !== "source"}
          >
            <SourceViewer
              source={controller.selectedSource}
              sectionId={controller.viewer.sectionId}
              onSelectSection={controller.selectSection}
            />
          </div>
          <div className="legal-disclaimer">
            <ShieldCheck aria-hidden="true" size={16} />
            <span>
              <strong>{t("disclaimer.title")}.</strong> {t("disclaimer.body")}
            </span>
          </div>
        </section>
      </div>
    </div>
  );
}
