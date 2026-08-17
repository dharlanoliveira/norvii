import { ChevronLeft, ChevronRight, FileText } from "lucide-react";
import { useTranslation } from "react-i18next";

import type { PdfSource } from "../../research/domain/models";

interface PdfSourceViewerProps {
  readonly source: PdfSource;
  readonly activeLocationId: string | undefined;
  readonly onLocationChange: (locationId: string) => void;
}

export function PdfSourceViewer({
  source,
  activeLocationId,
  onLocationChange,
}: PdfSourceViewerProps) {
  const { t } = useTranslation();
  const activeIndex = Math.max(
    source.locations.findIndex((location) => location.id === activeLocationId),
    0,
  );
  const location = source.locations[activeIndex];

  if (!location) return null;

  const previous = source.locations[activeIndex - 1];
  const next = source.locations[activeIndex + 1];

  return (
    <article className="source-document" aria-labelledby="source-title">
      <header className="source-document__header">
        <div className="source-document__identity">
          <span className="source-document__icon" aria-hidden="true">
            <FileText size={19} />
          </span>
          <div>
            <p className="kicker">{t("tree.pdfType")}</p>
            <h2 id="source-title">{source.title}</h2>
          </div>
        </div>
        <div className="source-navigation">
          <button
            type="button"
            aria-label={t("viewer.previous")}
            disabled={!previous}
            onClick={() => previous && onLocationChange(previous.id)}
          >
            <ChevronLeft aria-hidden="true" size={18} />
          </button>
          <span>
            {location.page
              ? t("viewer.page", {
                  page: location.page,
                  count: source.pageCount,
                })
              : location.label}
          </span>
          <button
            type="button"
            aria-label={t("viewer.next")}
            disabled={!next}
            onClick={() => next && onLocationChange(next.id)}
          >
            <ChevronRight aria-hidden="true" size={18} />
          </button>
        </div>
      </header>
      <dl className="source-metadata" aria-label={t("viewer.metadata")}>
        <div>
          <dt>{t("viewer.authority")}</dt>
          <dd>{source.authority}</dd>
        </div>
        <div>
          <dt>{t("viewer.reference")}</dt>
          <dd>{source.officialReference}</dd>
        </div>
        <div>
          <dt>{t("viewer.currentLocation")}</dt>
          <dd>{location.label}</dd>
        </div>
      </dl>
      <div className="source-document__paper">
        <span aria-hidden="true">
          {String(activeIndex + 1).padStart(2, "0")}
        </span>
        <div>
          <p className="source-document__location">{location.label}</p>
          <p className="source-document__content">{location.content}</p>
        </div>
      </div>
    </article>
  );
}
