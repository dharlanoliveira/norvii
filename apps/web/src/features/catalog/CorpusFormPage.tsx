import { ArrowLeft, Check, LibraryBig } from "lucide-react";
import { useEffect, useState, type SyntheticEvent } from "react";
import { useTranslation } from "react-i18next";
import { Link, useNavigate, useParams } from "react-router-dom";

import type { CorpusResponse } from "../../api/contract";
import { ResearchRequestError } from "../../api/researchProvider";
import type {
  CorpusDraft,
  ResearchProvider,
} from "../../research/domain/authoritative";
import "./catalog.css";

interface CorpusFormPageProps {
  readonly provider: ResearchProvider;
}

type FormPageState =
  | { readonly status: "ready"; readonly corpus?: CorpusResponse }
  | { readonly status: "loading" }
  | { readonly status: "saving"; readonly corpus?: CorpusResponse }
  | {
      readonly status: "failed";
      readonly reason: "load" | "save" | "stale";
      readonly corpus?: CorpusResponse;
    };

export function CorpusFormPage({ provider }: CorpusFormPageProps) {
  const { t } = useTranslation();
  const navigate = useNavigate();
  const { corpusId } = useParams<{ corpusId: string }>();
  const isEditing = corpusId !== undefined;
  const [state, setState] = useState<FormPageState>(
    isEditing ? { status: "loading" } : { status: "ready" },
  );
  const [dirty, setDirty] = useState(false);

  useEffect(() => {
    if (!corpusId) return;
    const controller = new AbortController();
    void provider
      .getCorpus(corpusId, controller.signal)
      .then((corpus) => setState({ status: "ready", corpus }))
      .catch(() => {
        if (!controller.signal.aborted)
          setState({ status: "failed", reason: "load" });
      });
    return () => controller.abort();
  }, [corpusId, provider]);

  useEffect(() => {
    if (!dirty) return;
    const warnBeforeUnload = (event: BeforeUnloadEvent): void => {
      event.preventDefault();
    };
    globalThis.addEventListener("beforeunload", warnBeforeUnload);
    return () =>
      globalThis.removeEventListener("beforeunload", warnBeforeUnload);
  }, [dirty]);

  const corpus = "corpus" in state ? state.corpus : undefined;
  const submit = (event: SyntheticEvent<HTMLFormElement>): void => {
    event.preventDefault();
    const data = new FormData(event.currentTarget);
    const draft: CorpusDraft = {
      name: formValue(data, "name"),
      description: formValue(data, "description"),
      language: data.get("language") === "pt" ? "pt" : "en",
      jurisdiction: formValue(data, "jurisdiction"),
    };
    const controller = new AbortController();
    setState(corpus ? { status: "saving", corpus } : { status: "saving" });
    const request =
      corpusId && corpus
        ? provider.updateCorpus(
            corpusId,
            { ...draft, version: corpus.version },
            controller.signal,
          )
        : provider.createCorpus(draft, controller.signal);
    void request
      .then(() => {
        setDirty(false);
        void navigate("/");
      })
      .catch((error: unknown) => {
        const reason =
          error instanceof ResearchRequestError && error.code === "stale_state"
            ? "stale"
            : "save";
        setState(
          corpus
            ? { status: "failed", reason, corpus }
            : { status: "failed", reason },
        );
      });
  };

  if (state.status === "loading") {
    return (
      <p className="corpus-form-status" role="status">
        {t("catalog.form.loading")}
      </p>
    );
  }
  if (state.status === "failed" && state.reason === "load") {
    return (
      <section
        className="corpus-form-status"
        aria-labelledby="corpus-form-load-error"
      >
        <h1 id="corpus-form-load-error">{t("catalog.form.loadFailed")}</h1>
        <Link className="recovery-link" to="/">
          {t("catalog.form.backToCatalog")}
        </Link>
      </section>
    );
  }

  const saving = state.status === "saving";
  const errorReason = state.status === "failed" ? state.reason : undefined;
  return (
    <section className="corpus-form-page" aria-labelledby="corpus-form-heading">
      <Link className="corpus-form-page__back" to="/">
        <ArrowLeft aria-hidden="true" size={16} />
        {t("catalog.form.backToCatalog")}
      </Link>
      <div className="corpus-form-layout">
        <header className="corpus-form-intro reveal">
          <span className="corpus-form-intro__mark" aria-hidden="true">
            <LibraryBig size={25} strokeWidth={1.5} />
          </span>
          <p className="kicker">
            {isEditing
              ? t("catalog.form.editKicker")
              : t("catalog.form.createKicker")}
          </p>
          <h1 id="corpus-form-heading">
            {isEditing
              ? t("catalog.form.editTitle")
              : t("catalog.form.createTitle")}
          </h1>
          <p>
            {isEditing
              ? t("catalog.form.editIntroduction")
              : t("catalog.form.createIntroduction")}
          </p>
        </header>
        <form
          className="corpus-form reveal"
          key={corpus?.id ?? "new"}
          onChange={() => setDirty(true)}
          onSubmit={submit}
        >
          <label className="corpus-form__field corpus-form__field--wide">
            <span>{t("catalog.form.name")}</span>
            <input
              autoComplete="off"
              name="name"
              defaultValue={corpus?.name}
              required
            />
          </label>
          <label className="corpus-form__field corpus-form__field--wide">
            <span>{t("catalog.form.description")}</span>
            <textarea
              autoComplete="off"
              name="description"
              defaultValue={corpus?.description}
              rows={5}
              required
            />
          </label>
          <label className="corpus-form__field">
            <span>{t("catalog.form.language")}</span>
            <select name="language" defaultValue={corpus?.language ?? "en"}>
              <option value="en">{t("language.en")}</option>
              <option value="pt">{t("language.pt")}</option>
            </select>
          </label>
          <label className="corpus-form__field">
            <span>{t("catalog.form.jurisdiction")}</span>
            <input
              autoComplete="off"
              name="jurisdiction"
              defaultValue={corpus?.jurisdiction}
              required
            />
          </label>
          {errorReason ? (
            <p className="corpus-form__error" role="alert">
              {errorReason === "stale"
                ? t("catalog.staleState")
                : t("catalog.mutationFailed")}
            </p>
          ) : null}
          <div className="corpus-form__actions">
            <Link className="corpus-form__cancel" to="/">
              {t("catalog.form.cancel")}
            </Link>
            <button
              className="corpus-form__submit"
              type="submit"
              disabled={saving}
            >
              <Check aria-hidden="true" size={17} />
              {saving
                ? t("catalog.form.saving")
                : isEditing
                  ? t("catalog.form.saveChanges")
                  : t("catalog.form.create")}
            </button>
          </div>
        </form>
      </div>
    </section>
  );
}

function formValue(data: FormData, field: string): string {
  const value = data.get(field);
  return typeof value === "string" ? value.trim() : "";
}
