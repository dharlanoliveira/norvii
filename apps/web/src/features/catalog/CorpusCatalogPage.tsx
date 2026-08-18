import { Plus, Scale } from "lucide-react";
import { useEffect, useState } from "react";
import { useTranslation } from "react-i18next";
import { Link } from "react-router-dom";

import type { CorpusResponse } from "../../api/contract";
import type { ResearchProvider } from "../../research/domain/authoritative";
import { CorpusCard } from "./CorpusCard";
import "./catalog.css";

interface CorpusCatalogPageProps {
  readonly provider: ResearchProvider;
}

type CatalogState =
  | { readonly status: "loading" }
  | { readonly status: "ready"; readonly corpora: readonly CorpusResponse[] }
  | { readonly status: "failed" };

export function CorpusCatalogPage({ provider }: CorpusCatalogPageProps) {
  const { t } = useTranslation();
  const [state, setState] = useState<CatalogState>({ status: "loading" });
  const [mutationFailed, setMutationFailed] = useState(false);

  useEffect(() => {
    const controller = new AbortController();
    void provider
      .listCorpora(controller.signal, true)
      .then((corpora) => setState({ status: "ready", corpora }))
      .catch(() => {
        if (!controller.signal.aborted) setState({ status: "failed" });
      });
    return () => controller.abort();
  }, [provider]);

  const toggleStatus = (corpus: CorpusResponse): void => {
    if (
      corpus.status === "enabled" &&
      !globalThis.confirm(t("catalog.confirmDisable", { name: corpus.name }))
    ) {
      return;
    }
    const controller = new AbortController();
    setMutationFailed(false);
    const mutation =
      corpus.status === "enabled"
        ? provider.disableCorpus(corpus.id, corpus.version, controller.signal)
        : provider.enableCorpus(corpus.id, corpus.version, controller.signal);
    void mutation
      .then((updated) => {
        setState((current) =>
          current.status === "ready"
            ? {
                status: "ready",
                corpora: current.corpora.map((item) =>
                  item.id === updated.id ? updated : item,
                ),
              }
            : current,
        );
      })
      .catch(handleMutationFailure);
  };

  const handleMutationFailure = (): void => {
    setMutationFailed(true);
  };

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
      <div className="catalog-toolbar">
        <Link className="catalog-toolbar__create" to="/corpora/new">
          <Plus aria-hidden="true" size={17} />
          {t("catalog.createCorpus")}
        </Link>
      </div>
      {mutationFailed ? (
        <p role="alert">{t("catalog.mutationFailed")}</p>
      ) : null}
      {state.status === "loading" ? (
        <p role="status">{t("catalog.loading")}</p>
      ) : null}
      {state.status === "failed" ? (
        <p role="alert">{t("catalog.loadFailed")}</p>
      ) : null}
      {state.status === "ready" && state.corpora.length === 0 ? (
        <p>{t("catalog.empty")}</p>
      ) : null}
      {state.status === "ready" ? (
        <div className="corpus-grid">
          {state.corpora.map((corpus, index) => (
            <CorpusCard
              key={corpus.id}
              corpus={corpus}
              index={index}
              onToggleStatus={toggleStatus}
            />
          ))}
        </div>
      ) : null}
    </section>
  );
}
