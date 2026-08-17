import { LibraryBig } from "lucide-react";
import { useTranslation } from "react-i18next";

import { EmptyState } from "../../components/EmptyState";
import type { Source } from "../../research/domain/models";
import { ExternalSourceViewer } from "./ExternalSourceViewer";
import { PdfSourceViewer } from "./PdfSourceViewer";

interface SourceViewerProps {
  readonly source: Source | undefined;
  readonly activeLocationId: string | undefined;
  readonly onLocationChange: (locationId: string) => void;
}

export function SourceViewer({
  source,
  activeLocationId,
  onLocationChange,
}: SourceViewerProps) {
  const { t } = useTranslation();

  if (!source) {
    return (
      <EmptyState
        kicker={t("viewer.noSourceKicker")}
        title={t("viewer.noSourceTitle")}
        body={t("viewer.noSourceBody")}
        icon={<LibraryBig size={24} strokeWidth={1.6} />}
      />
    );
  }

  if (source.kind === "pdf") {
    return (
      <PdfSourceViewer
        source={source}
        activeLocationId={activeLocationId}
        onLocationChange={onLocationChange}
      />
    );
  }

  return <ExternalSourceViewer source={source} />;
}
