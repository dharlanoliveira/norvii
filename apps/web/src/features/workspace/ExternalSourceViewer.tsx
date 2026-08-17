import { ArrowUpRight, ExternalLink, EyeOff } from "lucide-react";
import { useTranslation } from "react-i18next";

import type { ExternalSource } from "../../research/domain/models";

interface ExternalSourceViewerProps {
  readonly source: ExternalSource;
}

export function ExternalSourceViewer({ source }: ExternalSourceViewerProps) {
  const { t } = useTranslation();

  return (
    <article className="external-source" aria-labelledby="source-title">
      <header className="external-source__header">
        <span className="source-document__icon" aria-hidden="true">
          <ExternalLink size={19} />
        </span>
        <div>
          <p className="kicker">{t("viewer.externalKicker")}</p>
          <h2 id="source-title">{source.title}</h2>
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
      </dl>
      <section className="external-source__preview">
        {source.preview.status === "available" ? (
          <>
            <p className="external-source__label">{t("viewer.preview")}</p>
            <p>{source.preview.summary}</p>
          </>
        ) : (
          <div className="external-source__unavailable">
            <EyeOff aria-hidden="true" size={20} />
            <div>
              <p className="external-source__label">
                {t("viewer.previewUnavailable")}
              </p>
              <p>{source.preview.reason}</p>
            </div>
          </div>
        )}
      </section>
      <div className="external-source__action-row">
        <p>{t("viewer.opensNewContext")}</p>
        <a
          className="external-source__action"
          href={source.url}
          target="_blank"
          rel="noopener noreferrer"
        >
          {t("viewer.openOfficial")}
          <ArrowUpRight aria-hidden="true" size={18} />
        </a>
      </div>
    </article>
  );
}
