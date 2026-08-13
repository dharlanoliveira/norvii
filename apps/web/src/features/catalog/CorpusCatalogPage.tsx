import { Scale } from "lucide-react";
import { useTranslation } from "react-i18next";

import type { ResearchCatalog } from "../../research/domain/researchCatalog";
import { CorpusCard } from "./CorpusCard";
import "./catalog.css";

interface CorpusCatalogPageProps {
  readonly catalog: ResearchCatalog;
}

export function CorpusCatalogPage({ catalog }: CorpusCatalogPageProps) {
  const { t } = useTranslation();
  const corpora = catalog.listCorpora();

  return (
    <section className="catalog-page" aria-labelledby="catalog-heading">
      <header className="catalog-hero reveal">
        <div>
          <p className="kicker">{t("catalog.kicker")}</p>
          <h1 id="catalog-heading">{t("catalog.title")}</h1>
        </div>
        <div className="catalog-hero__note">
          <Scale aria-hidden="true" size={22} strokeWidth={1.5} />
          <p>{t("catalog.introduction")}</p>
        </div>
      </header>
      <div className="corpus-grid">
        {corpora.map((corpus, index) => (
          <CorpusCard key={corpus.id} corpus={corpus} index={index} />
        ))}
      </div>
    </section>
  );
}
