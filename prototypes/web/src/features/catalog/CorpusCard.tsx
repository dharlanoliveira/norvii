import { ArrowUpRight, BookOpenText, Globe2 } from "lucide-react";
import { useTranslation } from "react-i18next";
import { Link } from "react-router-dom";

import type { CorpusFixture } from "../../fixtures/models";

interface CorpusCardProps {
  readonly corpus: CorpusFixture;
  readonly index: number;
}

export function CorpusCard({ corpus, index }: CorpusCardProps) {
  const { t } = useTranslation();
  const languageLabel =
    corpus.language === "pt" ? t("language.portuguese") : t("language.english");

  return (
    <article
      className="corpus-card reveal"
      style={{ animationDelay: `${String(index * 90)}ms` }}
    >
      <div className="card-number" aria-hidden="true">
        0{index + 1}
      </div>
      <div className="corpus-card-heading">
        <p className="corpus-card-eyebrow">{t(corpus.eyebrowKey)}</p>
        <h2>{corpus.label}</h2>
      </div>
      <p className="corpus-description">{t(corpus.descriptionKey)}</p>
      <dl className="corpus-facts">
        <div>
          <dt>
            <Globe2 aria-hidden="true" size={15} />
            {t("language.contentLanguage")}
          </dt>
          <dd>{languageLabel}</dd>
        </div>
        <div>
          <dt>
            <BookOpenText aria-hidden="true" size={15} />
            {t("catalog.collectionLabel")}
          </dt>
          <dd>
            {corpus.jurisdiction} /{" "}
            {t("catalog.sourceCount", { count: corpus.sources.length })}
          </dd>
        </div>
      </dl>
      <Link className="corpus-open-link" to={`/corpora/${corpus.id}`}>
        <span>{t("catalog.openCorpus")}</span>
        <ArrowUpRight aria-hidden="true" size={20} strokeWidth={1.6} />
      </Link>
    </article>
  );
}
