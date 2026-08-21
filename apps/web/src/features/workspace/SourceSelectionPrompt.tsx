import { FileText } from "lucide-react";
import { useTranslation } from "react-i18next";

export function SourceSelectionPrompt() {
  const { t } = useTranslation();

  return (
    <section
      className="source-selection-prompt"
      aria-labelledby="source-selection-title"
    >
      <div className="source-selection-prompt__document" aria-hidden="true">
        <span className="source-selection-prompt__sheet source-selection-prompt__sheet--back" />
        <span className="source-selection-prompt__sheet source-selection-prompt__sheet--front">
          <FileText size={28} strokeWidth={1.5} />
          <span>01</span>
        </span>
      </div>
      <div className="source-selection-prompt__copy">
        <p className="kicker">{t("viewer.noSourceKicker")}</p>
        <h2 id="source-selection-title">{t("viewer.noSourceTitle")}</h2>
        <p>{t("viewer.noSourceBody")}</p>
      </div>
    </section>
  );
}
