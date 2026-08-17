import { Link, Outlet } from "react-router-dom";
import { useTranslation } from "react-i18next";

import { LanguageControl } from "../components/LanguageControl";

export function AppShell() {
  const { t } = useTranslation();

  return (
    <div className="app-shell">
      <a className="skip-link" href="#main-content">
        {t("app.skipToContent")}
      </a>
      <header className="site-header">
        <Link className="brand" to="/" aria-label="Norvii">
          <span className="brand__seal" aria-hidden="true">
            N
          </span>
          <span>
            <span className="brand__name">Norvii</span>
            <span className="brand__tagline">{t("app.brandTagline")}</span>
          </span>
        </Link>
        <div className="header-actions">
          <span className="demonstration-status">
            <span aria-hidden="true" />
            {t("app.demonstration")}
          </span>
          <LanguageControl />
        </div>
      </header>
      <main className="page-main" id="main-content">
        <Outlet />
      </main>
      <footer className="app-disclaimer">
        <strong>{t("app.demonstration")}</strong>
        <span>{t("app.notLegalAdvice")}</span>
      </footer>
    </div>
  );
}
