import { ArrowLeft, SearchX } from "lucide-react";
import { useTranslation } from "react-i18next";
import { Link } from "react-router-dom";

import { EmptyState } from "../../components/EmptyState";

export function UnknownCorpusPage() {
  const { t } = useTranslation();
  return (
    <div className="unknown-corpus-page">
      <EmptyState
        eyebrow={t("errors.unknownCorpusKicker")}
        title={t("errors.unknownCorpusTitle")}
        body={t("errors.unknownCorpusBody")}
        icon={<SearchX size={24} />}
        action={
          <Link className="recovery-link" to="/">
            <ArrowLeft aria-hidden="true" size={17} />
            {t("errors.returnToCatalog")}
          </Link>
        }
      />
    </div>
  );
}
