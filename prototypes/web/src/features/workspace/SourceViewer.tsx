import { LibraryBig } from "lucide-react";
import { useTranslation } from "react-i18next";

import { EmptyState } from "../../components/EmptyState";
import type { CorpusSource } from "../../fixtures/models";
import { ExternalSourceViewer } from "./ExternalSourceViewer";
import { PdfSourceViewer } from "./PdfSourceViewer";

interface SourceViewerProps {
  readonly source: CorpusSource | null;
  readonly sectionId: string | null;
  readonly onSelectSection: (sectionId: string) => void;
}

export function SourceViewer({
  source,
  sectionId,
  onSelectSection,
}: SourceViewerProps) {
  const { t } = useTranslation();

  if (!source) {
    return (
      <EmptyState
        eyebrow={t("viewer.emptyKicker")}
        title={t("viewer.emptyTitle")}
        body={t("viewer.emptyBody")}
        icon={<LibraryBig size={24} strokeWidth={1.5} />}
      />
    );
  }

  return source.kind === "pdf" ? (
    <PdfSourceViewer
      source={source}
      sectionId={sectionId}
      onSelectSection={onSelectSection}
    />
  ) : (
    <ExternalSourceViewer source={source} />
  );
}
