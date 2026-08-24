import type {
  DocumentResponse,
  DocumentUnitResponse,
} from "../../api/contract";

export interface CitedRange {
  readonly startOffset: number;
  readonly endOffset: number;
}

export function resolveVisibleUnitId(
  document: DocumentResponse,
  unitLocator: string,
  citedRange: CitedRange,
): string | undefined {
  if (
    citedRange.startOffset < 0 ||
    citedRange.endOffset <= citedRange.startOffset ||
    citedRange.endOffset > document.text.length
  ) {
    return undefined;
  }

  const target = document.units.find((unit) => unit.locator === unitLocator);
  if (
    target === undefined ||
    citedRange.startOffset < target.startOffset ||
    citedRange.endOffset > target.endOffset
  ) {
    return undefined;
  }

  const unitsById = new Map(document.units.map((unit) => [unit.id, unit]));
  let current: DocumentUnitResponse | undefined = target;
  while (current !== undefined) {
    if (
      current.kind !== "document" &&
      current.kind !== "paragraph" &&
      current.kind !== "item"
    ) {
      return current.id;
    }
    current =
      current.parentId === null ? undefined : unitsById.get(current.parentId);
  }

  return undefined;
}
