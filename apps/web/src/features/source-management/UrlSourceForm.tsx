import { useState, type SyntheticEvent } from "react";
import { useTranslation } from "react-i18next";

import { ResearchRequestError } from "../../api/researchProvider";
import type { UrlSourceDraft } from "../../research/domain/authoritative";

interface UrlSourceFormProps {
  readonly onSubmit: (
    draft: UrlSourceDraft,
    signal: AbortSignal,
  ) => Promise<void>;
}

export function UrlSourceForm({ onSubmit }: UrlSourceFormProps) {
  const { t } = useTranslation();
  const [status, setStatus] = useState<
    "idle" | "saving" | "saved" | "failed" | "duplicate"
  >("idle");

  const submit = (event: SyntheticEvent<HTMLFormElement>): void => {
    event.preventDefault();
    const form = event.currentTarget;
    if (!form.reportValidity()) return;
    const data = new FormData(form);
    const controller = new AbortController();
    setStatus("saving");
    void onSubmit(
      { title: requiredValue(data, "title"), url: requiredValue(data, "url") },
      controller.signal,
    )
      .then(() => {
        setStatus("saved");
        form.reset();
      })
      .catch((error: unknown) => {
        setStatus(
          error instanceof ResearchRequestError &&
            error.code === "duplicate_source"
            ? "duplicate"
            : "failed",
        );
      });
  };

  return (
    <form
      aria-labelledby="url-source-form-title"
      className="source-form"
      id="url-source-form"
      onSubmit={submit}
    >
      <h2 id="url-source-form-title">{t("sourceManagement.url.title")}</h2>
      <label>
        <span>{t("sourceManagement.titleLabel")}</span>
        <input autoComplete="off" name="title" required maxLength={200} />
      </label>
      <label>
        <span>{t("sourceManagement.url.urlLabel")}</span>
        <input
          autoComplete="off"
          name="url"
          type="url"
          spellCheck={false}
          required
          pattern="https://.*"
          placeholder="https://example.org/official-law"
        />
      </label>
      <button
        className="source-form__submit"
        type="submit"
        disabled={status === "saving"}
      >
        {status === "saving"
          ? t("sourceManagement.saving")
          : t("sourceManagement.url.submit")}
      </button>
      {status === "saved" ? (
        <p className="source-form__message" role="status">
          {t("sourceManagement.queued")}
        </p>
      ) : null}
      {status === "duplicate" ? (
        <p
          className="source-form__message source-form__message--error"
          role="alert"
        >
          {t("sourceManagement.duplicate")}
        </p>
      ) : null}
      {status === "failed" ? (
        <p
          className="source-form__message source-form__message--error"
          role="alert"
        >
          {t("sourceManagement.failed")}
        </p>
      ) : null}
    </form>
  );
}

function requiredValue(data: FormData, key: string): string {
  const value = data.get(key);
  return typeof value === "string" ? value.trim() : "";
}
