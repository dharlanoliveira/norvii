import { BookOpenText, MessageCircleMore } from "lucide-react";
import { useTranslation } from "react-i18next";

import type { WorkspaceMode } from "../../fixtures/models";

interface WorkspaceModeSelectorProps {
  readonly mode: WorkspaceMode;
  readonly onChange: (mode: WorkspaceMode) => void;
}

export function WorkspaceModeSelector({
  mode,
  onChange,
}: WorkspaceModeSelectorProps) {
  const { t } = useTranslation();

  return (
    <div className="mode-selector" role="tablist" aria-label={t("modes.label")}>
      <button
        type="button"
        role="tab"
        aria-selected={mode === "chat"}
        className={mode === "chat" ? "mode-button active" : "mode-button"}
        onClick={() => onChange("chat")}
      >
        <MessageCircleMore aria-hidden="true" size={16} />
        {t("modes.chat")}
      </button>
      <button
        type="button"
        role="tab"
        aria-selected={mode === "source"}
        className={mode === "source" ? "mode-button active" : "mode-button"}
        onClick={() => onChange("source")}
      >
        <BookOpenText aria-hidden="true" size={16} />
        {t("modes.source")}
      </button>
    </div>
  );
}
