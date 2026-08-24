import { useTranslation } from "react-i18next";

interface SourceSelectionPromptProps {
  readonly sourceTitle?: string | undefined;
  readonly onOpenSource?: (() => void) | undefined;
  readonly onAddSource: () => void;
}

export function SourceSelectionPrompt({
  sourceTitle,
  onOpenSource,
  onAddSource,
}: SourceSelectionPromptProps) {
  const { t } = useTranslation();
  const canOpenSource = sourceTitle !== undefined && onOpenSource !== undefined;

  return (
    <section
      className="source-selection-prompt"
      aria-labelledby="source-selection-title"
    >
      <div className="source-selection-prompt__copy">
        <h2 id="source-selection-title">{t("viewer.noSourceTitle")}</h2>
        <p>{t("viewer.noSourceBody")}</p>
        <button
          className="source-selection-prompt__action"
          type="button"
          onClick={canOpenSource ? onOpenSource : onAddSource}
        >
          {canOpenSource
            ? t("viewer.openSource", { source: sourceTitle })
            : t("viewer.addFirstSource")}
        </button>
      </div>
    </section>
  );
}
