import { ArrowUpRight, BookOpenText, MapPin } from "lucide-react";
import { useTranslation } from "react-i18next";
import { Link } from "react-router-dom";

import type { CorpusSummary } from "../../research/domain/models";

interface CorpusCardProps {
  readonly corpus: CorpusSummary;
  readonly index: number;
}

export function CorpusCard({ corpus, index }: CorpusCardProps) {
  const { t } = useTranslation();

  return (
    <article className="corpus-card reveal">
      <div className="corpus-card__rail" aria-hidden="true">
        <span>{String(index + 1).padStart(2, "0")}</span>
      </div>
      <div className="corpus-card__body">
        <div className="corpus-card__topline">
          <span className="corpus-card__type">
            {t("catalog.collectionLabel")}
          </span>
          <span className="corpus-card__language">
            {t(`language.${corpus.language}`)}
          </span>
        </div>
        <h2>{corpus.name}</h2>
        <p className="corpus-card__summary">{corpus.summary}</p>
        <dl className="corpus-card__facts">
          <div>
            <dt>
              <MapPin aria-hidden="true" size={14} />
              {t("catalog.jurisdiction")}
            </dt>
            <dd>{corpus.jurisdiction}</dd>
          </div>
          <div>
            <dt>
              <BookOpenText aria-hidden="true" size={14} />
              {t("catalog.sources")}
            </dt>
            <dd>{t("catalog.sourceCount", { count: corpus.sourceCount })}</dd>
          </div>
        </dl>
        <Link
          className="corpus-card__action"
          to={`/corpora/${corpus.id}`}
          aria-label={`${t("catalog.openCorpus")} ${corpus.name}`}
        >
          <span>{t("catalog.openCorpus")}</span>
          <ArrowUpRight aria-hidden="true" size={18} />
        </Link>
      </div>
    </article>
  );
}
