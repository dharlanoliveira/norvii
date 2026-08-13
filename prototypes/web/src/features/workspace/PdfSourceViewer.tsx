import { ChevronLeft, ChevronRight, FileText } from "lucide-react";
import { useTranslation } from "react-i18next";

import type { CorpusSource } from "../../fixtures/models";

interface PdfSourceViewerProps {
  readonly source: CorpusSource;
  readonly sectionId: string | null;
  readonly onSelectSection: (sectionId: string) => void;
}

export function PdfSourceViewer({
  source,
  sectionId,
  onSelectSection,
}: PdfSourceViewerProps) {
  const { t } = useTranslation();
  const sectionIndex = Math.max(
    0,
    source.sections.findIndex((section) => section.id === sectionId),
  );
  const section = source.sections[sectionIndex];

  if (!section) {
    return null;
  }

  return (
    <article className="document-viewer" aria-label={t("viewer.documentLabel")}>
      <header className="viewer-header">
        <div className="viewer-type">
          <FileText aria-hidden="true" size={16} />
          {t("workspace.pdfType")}
        </div>
        <h2>{source.title}</h2>
        <div className="viewer-metadata">
          <span>
            {t("viewer.publishedBy")} <strong>{source.publisher}</strong>
          </span>
          <span>{source.publishedLabel}</span>
        </div>
      </header>
      <div className="document-page">
        <div className="citation-margin" aria-hidden="true">
          <span>{section.marker}</span>
        </div>
        <div className="document-copy" id={section.id} tabIndex={-1}>
          <span className="section-marker">{section.marker}</span>
          <h3>{section.heading}</h3>
          {section.paragraphs.map((paragraph) => (
            <p key={paragraph}>{paragraph}</p>
          ))}
        </div>
      </div>
      <footer className="viewer-footer">
        <span>
          {t("viewer.location")}: <strong>{section.marker}</strong>
        </span>
        <div className="viewer-pagination">
          <button
            type="button"
            aria-label={t("viewer.previousSection")}
            disabled={sectionIndex === 0}
            onClick={() => {
              const previous = source.sections[sectionIndex - 1];
              if (previous) onSelectSection(previous.id);
            }}
          >
            <ChevronLeft aria-hidden="true" size={17} />
          </button>
          <span>
            {sectionIndex + 1} / {source.sections.length}
          </span>
          <button
            type="button"
            aria-label={t("viewer.nextSection")}
            disabled={sectionIndex === source.sections.length - 1}
            onClick={() => {
              const next = source.sections[sectionIndex + 1];
              if (next) onSelectSection(next.id);
            }}
          >
            <ChevronRight aria-hidden="true" size={17} />
          </button>
        </div>
      </footer>
    </article>
  );
}
