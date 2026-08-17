import { ExternalLink, Globe2, TriangleAlert } from "lucide-react";
import { useTranslation } from "react-i18next";

import type { CorpusSource } from "../../fixtures/models";

interface ExternalSourceViewerProps {
  readonly source: CorpusSource;
}

export function ExternalSourceViewer({ source }: ExternalSourceViewerProps) {
  const { t } = useTranslation();

  return (
    <article className="external-viewer" aria-label={t("viewer.externalLabel")}>
      <header className="viewer-header">
        <div className="viewer-type">
          <Globe2 aria-hidden="true" size={16} />
          {t("workspace.externalType")}
        </div>
        <h2>{source.title}</h2>
        <div className="viewer-metadata">
          <span>
            {t("viewer.publishedBy")} <strong>{source.publisher}</strong>
          </span>
          <span>{source.publishedLabel}</span>
        </div>
      </header>
      {source.status === "unavailable" ? (
        <div className="unavailable-preview">
          <TriangleAlert aria-hidden="true" size={24} />
          <div>
            <h3>{t("viewer.unavailableTitle")}</h3>
            <p>{t("viewer.unavailableBody")}</p>
          </div>
        </div>
      ) : (
        <div className="external-preview">
          <p className="eyebrow">{t("viewer.previewHeading")}</p>
          {source.sections.map((section) => (
            <section key={section.id} id={section.id}>
              <span>{section.marker}</span>
              <h3>{section.heading}</h3>
              {section.paragraphs.map((paragraph) => (
                <p key={paragraph}>{paragraph}</p>
              ))}
            </section>
          ))}
        </div>
      )}
      {source.externalUrl ? (
        <footer className="external-footer">
          <p>{t("viewer.externalNotice")}</p>
          <a href={source.externalUrl} target="_blank" rel="noreferrer">
            {t("viewer.openOriginal")}
            <ExternalLink aria-hidden="true" size={16} />
          </a>
        </footer>
      ) : null}
    </article>
  );
}
