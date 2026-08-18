import { BookOpenText } from "lucide-react";
import { useEffect, useMemo, useRef } from "react";
import { useTranslation } from "react-i18next";

import type {
  DocumentResponse,
  DocumentUnitResponse,
} from "../../api/contract";

interface LegalDocumentReaderProps {
  readonly document: DocumentResponse;
  readonly selectedUnitId: string | undefined;
  readonly onSelect: (unitId: string) => void;
}

export function LegalDocumentReader({
  document,
  selectedUnitId,
  onSelect,
}: LegalDocumentReaderProps) {
  const { t } = useTranslation();
  const selectedRef = useRef<HTMLElement>(null);
  const locations = useMemo(() => readingLocations(document), [document]);
  const visibleUnits = locations.length > 0 ? locations : document.units;
  const selectedId = visibleUnits.some((unit) => unit.id === selectedUnitId)
    ? selectedUnitId
    : visibleUnits[0]?.id;
  const previousSelection = useRef(selectedId);

  useEffect(() => {
    if (previousSelection.current === selectedId) {
      return;
    }
    previousSelection.current = selectedId;
    selectedRef.current?.scrollIntoView({ block: "start" });
  }, [selectedId]);

  return (
    <section className="legal-reader" aria-label={t("viewer.readerLabel")}>
      {visibleUnits.length > 1 ? (
        <nav
          className="legal-reader__navigator"
          aria-label={t("viewer.locations")}
        >
          <BookOpenText aria-hidden="true" size={17} />
          <label htmlFor="document-location">{t("viewer.locationLabel")}</label>
          <select
            id="document-location"
            value={selectedId}
            onChange={(event) => onSelect(event.target.value)}
          >
            {visibleUnits.map((unit) => (
              <option key={unit.id} value={unit.id}>
                {unitLabel(unit, t)}
              </option>
            ))}
          </select>
          <span>
            {t("viewer.locationCount", { count: visibleUnits.length })}
          </span>
        </nav>
      ) : null}
      <div className="legal-reader__paper">
        {visibleUnits.map((unit) => {
          const label = unitLabel(unit, t);
          const isSelected = unit.id === selectedId;
          const paragraphs = unitParagraphs(document.text, unit);
          return (
            <article
              ref={isSelected ? selectedRef : undefined}
              className="legal-unit"
              data-selected={isSelected || undefined}
              aria-label={label}
              key={unit.id}
            >
              <header className="legal-unit__header">
                <span>{locationKind(unit, t)}</span>
                <h2>{label}</h2>
              </header>
              <div className="legal-unit__body">
                {paragraphs.map((paragraph, index) => (
                  <p key={`${unit.id}-${String(index)}`}>{paragraph}</p>
                ))}
              </div>
            </article>
          );
        })}
      </div>
    </section>
  );
}

type Translator = ReturnType<typeof useTranslation>["t"];

function unitLabel(unit: DocumentUnitResponse, t: Translator): string {
  const marker = displayMarker(unit);
  return (
    marker ??
    unit.label ??
    (unit.kind === "title"
      ? t("viewer.unitKinds.title")
      : t("viewer.unitKinds.document"))
  );
}

function locationKind(unit: DocumentUnitResponse, t: Translator): string {
  const supportedKinds = new Set([
    "title",
    "chapter",
    "section",
    "article",
    "recital",
    "paragraph",
    "item",
    "block",
  ]);
  const kind = supportedKinds.has(unit.kind) ? unit.kind : "document";
  return t(`viewer.unitKinds.${kind}`);
}

function unitParagraphs(
  documentText: string,
  unit: DocumentUnitResponse,
): readonly string[] {
  let content = documentText.slice(unit.startOffset, unit.endOffset).trim();
  const marker = displayMarker(unit);
  if (marker && content.startsWith(marker)) {
    content = content.slice(marker.length).trimStart();
  }
  const paragraphs = content
    .split(/\n+/u)
    .map((paragraph) => paragraph.trim())
    .filter(Boolean);
  return paragraphs.length > 0 ? paragraphs : [content];
}

function readingLocations(
  document: DocumentResponse,
): readonly DocumentUnitResponse[] {
  const navigable = document.units.filter(
    (unit) =>
      unit.kind !== "document" &&
      unit.kind !== "paragraph" &&
      unit.kind !== "item",
  );
  if (navigable.length === 0) {
    return document.units;
  }
  return navigable.map((unit, index) => {
    const nextStart = navigable[index + 1]?.startOffset ?? document.text.length;
    return nextStart > unit.startOffset && nextStart !== unit.endOffset
      ? { ...unit, endOffset: nextStart }
      : unit;
  });
}

function displayMarker(unit: DocumentUnitResponse): string | null {
  const marker = unit.marker?.trim() ?? null;
  if (unit.kind !== "article" || marker === null) return marker;
  return (
    /^(?:Article|Artigo)\s+\d+[A-Za-z]?[.\u00ba\u00b0]?/iu.exec(marker)?.[0] ??
    /^Art\.\s*\d+(?:-[A-Za-z])?[\u00ba\u00b0]?/iu.exec(marker)?.[0] ??
    marker
  );
}
