import { useState, type SyntheticEvent } from "react";
import { useTranslation } from "react-i18next";

import { ResearchRequestError } from "../../api/researchProvider";

const MAX_PDF_BYTES = 10 * 1024 * 1024;

interface PdfSourceFormProps {
  readonly onSubmit: (
    title: string,
    file: File,
    signal: AbortSignal,
  ) => Promise<void>;
}

export function PdfSourceForm({ onSubmit }: PdfSourceFormProps) {
  const { t } = useTranslation();
  const [status, setStatus] = useState<
    "idle" | "uploading" | "queued" | "failed" | "duplicate"
  >("idle");
  const [selectedFile, setSelectedFile] = useState<File>();

  const submit = (event: SyntheticEvent<HTMLFormElement>): void => {
    event.preventDefault();
    const form = event.currentTarget;
    const data = new FormData(form);
    const title = data.get("title");
    if (
      typeof title !== "string" ||
      selectedFile === undefined ||
      !isValidPdf(selectedFile)
    ) {
      setStatus("failed");
      return;
    }
    const controller = new AbortController();
    setStatus("uploading");
    void onSubmit(title.trim(), selectedFile, controller.signal)
      .then(() => {
        setStatus("queued");
        setSelectedFile(undefined);
        form.reset();
      })
      .catch((error: unknown) =>
        setStatus(
          error instanceof ResearchRequestError &&
            error.code === "duplicate_source"
            ? "duplicate"
            : "failed",
        ),
      );
  };

  return (
    <form
      aria-labelledby="pdf-source-form-title"
      className="source-form"
      id="pdf-source-form"
      onSubmit={submit}
    >
      <h2 id="pdf-source-form-title">{t("sourceManagement.pdf.title")}</h2>
      <label>
        <span>{t("sourceManagement.titleLabel")}</span>
        <input autoComplete="off" name="title" required maxLength={200} />
      </label>
      <label>
        <span>{t("sourceManagement.pdf.fileLabel")}</span>
        <input
          name="file"
          type="file"
          accept="application/pdf,.pdf"
          aria-required="true"
          onChange={(event) => setSelectedFile(event.currentTarget.files?.[0])}
        />
      </label>
      <button
        className="source-form__submit"
        type="submit"
        disabled={status === "uploading"}
      >
        {t("sourceManagement.pdf.submit")}
      </button>
      {status === "uploading" ? (
        <progress
          className="source-form__progress"
          aria-label={t("sourceManagement.pdf.progress")}
        />
      ) : null}
      {status === "queued" ? (
        <p className="source-form__message" role="status">
          {t("sourceManagement.queued")}
        </p>
      ) : null}
      {status === "failed" ? (
        <p
          className="source-form__message source-form__message--error"
          role="alert"
        >
          {t("sourceManagement.pdf.failed")}
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
    </form>
  );
}

function isValidPdf(file: File): boolean {
  return (
    file.size > 0 &&
    file.size <= MAX_PDF_BYTES &&
    (file.type === "application/pdf" ||
      file.name.toLowerCase().endsWith(".pdf"))
  );
}
