import { MessageSquareLock } from "lucide-react";
import { useTranslation } from "react-i18next";

export function ResearchChat() {
  const { t } = useTranslation();
  return (
    <section className="research-chat" aria-label={t("chat.regionLabel")}>
      <div className="chat-empty">
        <MessageSquareLock aria-hidden="true" size={24} />
        <p className="kicker">{t("chat.kicker")}</p>
        <h2>{t("chat.unavailableTitle")}</h2>
        <p>{t("chat.unavailable")}</p>
      </div>
    </section>
  );
}
