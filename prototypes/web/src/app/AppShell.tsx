import { Link, Outlet } from "react-router-dom";
import { useTranslation } from "react-i18next";

import { LanguageControl } from "../components/LanguageControl";

export function AppShell() {
  const { t } = useTranslation();

  return (
    <div className="app-shell">
      <header className="site-header">
        <Link className="brand-link" to="/" aria-label="Norvii">
          <span className="brand-mark" aria-hidden="true">
            N
          </span>
          <span>
            <span className="brand-name">Norvii</span>
            <span className="brand-descriptor">{t("brand.descriptor")}</span>
          </span>
        </Link>
        <div className="header-actions">
          <span className="prototype-status">
            <span className="status-dot" aria-hidden="true" />
            {t("status.prototype")}
          </span>
          <LanguageControl />
        </div>
      </header>
      <main className="page-main">
        <Outlet />
      </main>
    </div>
  );
}
