import { BookOpenText, MessagesSquare } from "lucide-react";
import { useRef, type KeyboardEvent } from "react";
import { useTranslation } from "react-i18next";

export type WorkspaceMode = "chat" | "source";

interface WorkspaceModeSelectorProps {
  readonly mode: WorkspaceMode;
  readonly onChange: (mode: WorkspaceMode) => void;
}

export function WorkspaceModeSelector({
  mode,
  onChange,
}: WorkspaceModeSelectorProps) {
  const { t } = useTranslation();
  const chatRef = useRef<HTMLButtonElement>(null);
  const sourceRef = useRef<HTMLButtonElement>(null);

  const handleKeyDown = (
    event: KeyboardEvent<HTMLButtonElement>,
    nextMode: WorkspaceMode,
  ): void => {
    if (event.key !== "ArrowLeft" && event.key !== "ArrowRight") return;
    event.preventDefault();
    onChange(nextMode);
    (nextMode === "chat" ? chatRef : sourceRef).current?.focus();
  };

  return (
    <div
      className="mode-selector"
      role="tablist"
      aria-label={t("workspace.modeLabel")}
    >
      <button
        ref={chatRef}
        type="button"
        role="tab"
        aria-selected={mode === "chat"}
        aria-controls="chat-panel"
        tabIndex={mode === "chat" ? 0 : -1}
        onClick={() => onChange("chat")}
        onKeyDown={(event) => handleKeyDown(event, "source")}
      >
        <MessagesSquare aria-hidden="true" size={15} />
        {t("workspace.chat")}
      </button>
      <button
        ref={sourceRef}
        type="button"
        role="tab"
        aria-selected={mode === "source"}
        aria-controls="source-panel"
        tabIndex={mode === "source" ? 0 : -1}
        onClick={() => onChange("source")}
        onKeyDown={(event) => handleKeyDown(event, "chat")}
      >
        <BookOpenText aria-hidden="true" size={15} />
        {t("workspace.source")}
      </button>
    </div>
  );
}
