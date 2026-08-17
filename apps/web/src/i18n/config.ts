import i18n from "i18next";
import { initReactI18next } from "react-i18next";

import { englishTranslation } from "./en/translation";
import { portugueseTranslation } from "./pt/translation";

void i18n.use(initReactI18next).init({
  resources: {
    en: { translation: englishTranslation },
    pt: { translation: portugueseTranslation },
  },
  lng: "en",
  fallbackLng: "en",
  supportedLngs: ["en", "pt"],
  interpolation: {
    escapeValue: false,
  },
  returnNull: false,
});

export { i18n };
