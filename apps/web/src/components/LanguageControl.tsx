import { Languages } from "lucide-react";
import { useEffect } from "react";
import { useTranslation } from "react-i18next";

type InterfaceLanguage = "en" | "pt";

export function LanguageControl() {
  const { i18n, t } = useTranslation();
  const language: InterfaceLanguage =
    i18n.resolvedLanguage === "pt" ? "pt" : "en";

  useEffect(() => {
    document.documentElement.lang = language;
  }, [language]);

  return (
    <label className="language-control">
      <Languages aria-hidden="true" size={15} strokeWidth={1.8} />
      <span className="visually-hidden">{t("language.label")}</span>
      <select
        aria-label={t("language.label")}
        value={language}
        onChange={(event) => void i18n.changeLanguage(event.target.value)}
      >
        <option value="en">EN</option>
        <option value="pt">PT</option>
      </select>
    </label>
  );
}
