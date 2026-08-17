import { CircleOff } from "lucide-react";
import { useTranslation } from "react-i18next";
import { Link } from "react-router-dom";

import { EmptyState } from "../../components/EmptyState";

export function UnknownCorpusPage() {
  const { t } = useTranslation();

  return (
    <EmptyState
      kicker={t("errors.unknownCorpusKicker")}
      title={t("errors.unknownCorpusTitle")}
      body={t("errors.unknownCorpusBody")}
      icon={<CircleOff size={24} strokeWidth={1.6} />}
      action={
        <Link className="recovery-link" to="/">
          {t("errors.returnToCatalog")}
        </Link>
      }
    />
  );
}
