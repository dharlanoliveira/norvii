import {
  ArrowUpRight,
  BookOpenText,
  MapPin,
  MoreHorizontal,
} from "lucide-react";
import { useTranslation } from "react-i18next";
import { Link } from "react-router-dom";

import type { CorpusResponse } from "../../api/contract";

interface CorpusCardProps {
  readonly corpus: CorpusResponse;
  readonly index: number;
  readonly onToggleStatus?: (corpus: CorpusResponse) => void;
}

export function CorpusCard({ corpus, index, onToggleStatus }: CorpusCardProps) {
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
          <span>{t(`catalog.status.${corpus.status}`)}</span>
        </div>
        <h2>{corpus.name}</h2>
        <p className="corpus-card__summary">{corpus.description}</p>
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
          <div>
            <dt>{t("catalog.activeSnapshot")}</dt>
            <dd>
              {corpus.activeSnapshot === null ||
              corpus.activeSnapshot === undefined
                ? t("catalog.snapshotUnavailable")
                : corpus.activeSnapshot.id.slice(0, 8)}
            </dd>
          </div>
        </dl>
        <div className="corpus-card__actions">
          <Link
            className="corpus-card__action"
            to={`/corpora/${corpus.id}`}
            aria-label={`${t("catalog.openCorpus")} ${corpus.name}`}
          >
            <span>{t("catalog.openCorpus")}</span>
            <ArrowUpRight aria-hidden="true" size={18} />
          </Link>
          <details className="corpus-card__menu">
            <summary aria-label={`${t("catalog.manageCorpus")} ${corpus.name}`}>
              <MoreHorizontal aria-hidden="true" size={19} />
            </summary>
            <div className="corpus-card__menu-panel">
              <Link
                to={`/corpora/${corpus.id}/edit`}
                aria-label={`${t("catalog.editCorpus")} ${corpus.name}`}
              >
                {t("catalog.editCorpus")}
              </Link>
              {onToggleStatus ? (
                <button type="button" onClick={() => onToggleStatus(corpus)}>
                  {corpus.status === "enabled"
                    ? t("catalog.disableCorpus")
                    : t("catalog.enableCorpus")}
                </button>
              ) : null}
            </div>
          </details>
        </div>
      </div>
    </article>
  );
}
