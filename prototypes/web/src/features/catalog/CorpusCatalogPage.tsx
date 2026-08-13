import { useTranslation } from "react-i18next";

import { corpora } from "../../fixtures/legal-content/corpora";
import { CorpusCard } from "./CorpusCard";
import "./catalog.css";

export function CorpusCatalogPage() {
  const { t } = useTranslation();

  return (
    <div className="catalog-page">
      <section className="catalog-hero reveal">
        <div>
          <p className="eyebrow">{t("catalog.kicker")}</p>
          <h1>{t("catalog.title")}</h1>
        </div>
        <div className="catalog-introduction">
          <p>{t("catalog.introduction")}</p>
          <span>{t("catalog.principle")}</span>
        </div>
      </section>
      <section className="corpus-grid" aria-label={t("navigation.corpora")}>
        {corpora.map((corpus, index) => (
          <CorpusCard corpus={corpus} index={index} key={corpus.id} />
        ))}
      </section>
      <footer className="catalog-footer">
        <span>Norvii / 001</span>
        <span>{t("status.localData")}</span>
      </footer>
    </div>
  );
}
