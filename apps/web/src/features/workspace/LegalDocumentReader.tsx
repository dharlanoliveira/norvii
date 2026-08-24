import { BookOpenText } from "lucide-react";
import type { ReactNode, RefObject } from "react";
import { useLayoutEffect, useMemo, useRef } from "react";
import { useTranslation } from "react-i18next";

import type {
  DocumentResponse,
  DocumentUnitResponse,
} from "../../api/contract";
import type { CitedRange } from "./citationLocation";

interface LegalDocumentReaderProps {
  readonly document: DocumentResponse;
  readonly selectedUnitId: string | undefined;
  readonly onSelect: (unitId: string) => void;
  readonly citedRange?: CitedRange | undefined;
}

export function LegalDocumentReader({
  document,
  selectedUnitId,
  onSelect,
  citedRange,
}: LegalDocumentReaderProps) {
  const { t } = useTranslation();
  const selectedRef = useRef<HTMLElement>(null);
  const citationRef = useRef<HTMLElement>(null);
  const locations = useMemo(() => readingLocations(document), [document]);
  const visibleUnits = locations.length > 0 ? locations : document.units;
  const selectedId = visibleUnits.some((unit) => unit.id === selectedUnitId)
    ? selectedUnitId
    : visibleUnits[0]?.id;
  useLayoutEffect(() => {
    const target =
      citedRange === undefined ? selectedRef.current : citationRef.current;
    target?.scrollIntoView({
      block: citedRange === undefined ? "start" : "center",
    });
  }, [citedRange, selectedId]);

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
                  <p key={`${unit.id}-${String(index)}`}>
                    {highlightedText(paragraph, citedRange, citationRef)}
                  </p>
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

interface UnitParagraph {
  readonly text: string;
  readonly startOffset: number;
  readonly endOffset: number;
}

function unitParagraphs(
  documentText: string,
  unit: DocumentUnitResponse,
): readonly UnitParagraph[] {
  const rawContent = documentText.slice(unit.startOffset, unit.endOffset);
  const leadingWhitespace = rawContent.length - rawContent.trimStart().length;
  let content = rawContent.trim();
  let startOffset = unit.startOffset + leadingWhitespace;
  const marker = displayMarker(unit);
  if (marker && content.startsWith(marker)) {
    const afterMarker = content.slice(marker.length);
    content = afterMarker.trimStart();
    startOffset += marker.length;
    startOffset += afterMarker.length - content.length;
  }
  const paragraphs = [...content.matchAll(/[^\n]+/gu)]
    .map((match) => {
      const line = match[0];
      const text = line.trim();
      const leading = line.length - line.trimStart().length;
      const paragraphStart = startOffset + match.index + leading;
      return {
        text,
        startOffset: paragraphStart,
        endOffset: paragraphStart + text.length,
      };
    })
    .filter((paragraph) => paragraph.text !== "");
  return paragraphs.length > 0
    ? paragraphs
    : [{ text: content, startOffset, endOffset: startOffset + content.length }];
}

function highlightedText(
  paragraph: UnitParagraph,
  citedRange: CitedRange | undefined,
  citationRef: RefObject<HTMLElement | null>,
): ReactNode {
  if (
    citedRange === undefined ||
    citedRange.startOffset >= paragraph.endOffset ||
    citedRange.endOffset <= paragraph.startOffset
  ) {
    return paragraph.text;
  }
  const start = Math.max(0, citedRange.startOffset - paragraph.startOffset);
  const end = Math.min(
    paragraph.text.length,
    citedRange.endOffset - paragraph.startOffset,
  );
  return (
    <>
      {paragraph.text.slice(0, start)}
      <mark className="legal-citation-highlight" ref={citationRef}>
        {paragraph.text.slice(start, end)}
      </mark>
      {paragraph.text.slice(end)}
    </>
  );
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
    legalMarker(marker, /^(?:Article|Artigo)\s+\d+[A-Z]?/iu) ??
    legalMarker(marker, /^Art\.\s*\d+(?:-[A-Z])?/iu) ??
    marker
  );
}

function legalMarker(marker: string, pattern: RegExp): string | null {
  const match = pattern.exec(marker)?.[0];
  if (match === undefined) return null;
  const suffix = marker.at(match.length);
  return suffix === "." || suffix === "\u00ba" ? match + suffix : match;
}
