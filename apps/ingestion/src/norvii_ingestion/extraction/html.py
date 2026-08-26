"""Structured HTML text extraction with deterministic legal or block units."""

from __future__ import annotations

import re
import xml.etree.ElementTree as ET
from collections import Counter
from dataclasses import dataclass, replace
from uuid import NAMESPACE_URL, UUID, uuid5

import trafilatura

from norvii_ingestion.domain.artifacts import DocumentArtifact, DocumentUnit, UnitKind
from norvii_ingestion.domain.models import Sha256
from norvii_ingestion.extraction.legal_locator import canonical_legal_locator

_LEGAL_MARKER = re.compile(
    r"^(?P<marker>"
    r"(?:\d+\s+U\.S\.C\.\s+\u00a7\s*\d+(?:\.\d+)*(?:\([A-Za-z0-9]+\))*"
    r"|\d+\s+CFR\s+\u00a7\s*\d+(?:\.\d+)*(?:\([A-Za-z0-9]+\))*)"
    r"|(?:Title|T\u00edtulo|Chapter|Cap\u00edtulo|Section|Se\u00e7\u00e3o)\s+[^\n]+"
    r"|(?:Article|Artigo)\s+\d+(?:[\u00ba\u00b0o])?(?:-[A-Za-z]+)?[.]?"
    r"|Art\.\s*\d+(?:[\u00ba\u00b0o])?(?:-[A-Za-z]+)?[.]?"
    r"|Recital\s+\d+"
    r"|\u00a7\s*\d+(?:[\u00bao])?(?:-[A-Za-z]+)?"
    r"|(?:\(\d+\)|\d+\.)\s+"
    r"|[IVXLCDM]+\s*[-\u2013\u2014]\s+"
    r"|(?:\([a-z]\)|[a-z]\))\s+"
    r")",
    re.IGNORECASE | re.MULTILINE,
)


@dataclass(frozen=True, slots=True)
class _UnitSpec:
    id: UUID
    parent_id: UUID | None
    kind: UnitKind
    ordinal: int
    start: int
    end: int
    locator: str
    marker: str | None = None
    canonical_locator: str | None = None


@dataclass(frozen=True, slots=True)
class _LegalNode:
    id: UUID
    parent_id: UUID
    kind: UnitKind
    ordinal: int
    start: int
    level: int
    locator: str
    marker: str
    canonical_locator: str | None


class ExtractionError(ValueError):
    """Report HTML content that cannot yield a valid document artifact."""


class HtmlExtractor:
    """Extract UTF-8 or legacy Windows-1252 HTML into an addressable hierarchy."""

    def extract(self, content: bytes) -> DocumentArtifact:
        """Return complete normalized text with legal or block fallback units."""
        html = self._decode_html(content)
        extracted = trafilatura.extract(
            html,
            include_comments=False,
            include_tables=True,
            include_formatting=True,
            output_format="xml",
            favor_recall=True,
            no_fallback=False,
        )
        if extracted is None:
            raise ExtractionError("HTML content does not contain extractable text.")
        text = self._normalize_xml(extracted)
        if not text:
            raise ExtractionError("HTML content does not contain extractable text.")
        text_hash = Sha256.from_bytes(text.encode("utf-8"))
        root_id = self._unit_id(text_hash, "document")
        children = self._legal_units(text, text_hash, root_id)
        if not children:
            children = self._block_units(text, text_hash, root_id)
        artifact = DocumentArtifact(
            text=text,
            text_sha256=text_hash,
            units=(
                self._unit(
                    text,
                    _UnitSpec(
                        root_id,
                        None,
                        UnitKind.DOCUMENT,
                        0,
                        0,
                        len(text),
                        "document",
                    ),
                ),
                *children,
            ),
        )
        try:
            artifact.validate()
        except ValueError as error:
            raise ExtractionError("Extracted HTML artifact is structurally invalid.") from error
        return artifact

    @staticmethod
    def _decode_html(content: bytes) -> str:
        """Decode modern HTML first and deterministically support legacy legal sites."""
        try:
            return content.decode("utf-8")
        except UnicodeDecodeError:
            return content.decode("windows-1252")

    @staticmethod
    def _normalize_xml(value: str) -> str:
        try:
            # Trafilatura produced this bounded XML from already acquired bytes.
            root = ET.fromstring(value)  # noqa: S314
        except ET.ParseError as error:
            raise ExtractionError("Extracted HTML structure is invalid.") from error
        blocks: list[str] = []
        for element in root.iter():
            tag = element.tag.rsplit("}", maxsplit=1)[-1]
            if tag not in {"head", "p", "item", "quote", "cell"}:
                continue
            block = " ".join("".join(element.itertext()).split())
            if block:
                blocks.append(block)
        return "\n".join(blocks)

    def _legal_units(self, text: str, text_hash: Sha256, root_id: UUID) -> tuple[DocumentUnit, ...]:
        matches = list(_LEGAL_MARKER.finditer(text))
        if not matches:
            return ()
        units: list[DocumentUnit] = []
        child_counts: dict[UUID, int] = {}
        if matches[0].start() > 0:
            units.append(
                self._unit(
                    text,
                    _UnitSpec(
                        self._unit_id(text_hash, "title"),
                        root_id,
                        UnitKind.TITLE,
                        0,
                        0,
                        matches[0].start(),
                        "title",
                    ),
                )
            )
            child_counts[root_id] = 1
        nodes: list[_LegalNode] = []
        active: list[_LegalNode] = []
        for index, match in enumerate(matches):
            marker = match.group("marker").strip()
            kind = self._marker_kind(marker)
            level = self._hierarchy_level(kind, marker)
            while active and active[-1].level >= level:
                active.pop()
            parent_id = active[-1].id if active else root_id
            ordinal = child_counts.get(parent_id, 0)
            child_counts[parent_id] = ordinal + 1
            locator = f"{kind.value}-{index + 1}"
            parent_locator = next(
                (node.canonical_locator for node in reversed(active) if node.canonical_locator),
                None,
            )
            node = _LegalNode(
                id=self._unit_id(text_hash, locator),
                parent_id=parent_id,
                kind=kind,
                ordinal=ordinal,
                start=match.start(),
                level=level,
                locator=locator,
                marker=marker,
                canonical_locator=canonical_legal_locator(kind.value, marker, parent_locator),
            )
            nodes.append(node)
            active.append(node)
        for index, node in enumerate(nodes):
            end = next(
                (
                    candidate.start
                    for candidate in nodes[index + 1 :]
                    if candidate.level <= node.level
                ),
                len(text),
            )
            units.append(
                self._unit(
                    text,
                    _UnitSpec(
                        node.id,
                        node.parent_id,
                        node.kind,
                        node.ordinal,
                        node.start,
                        end,
                        node.locator,
                        node.marker,
                        node.canonical_locator,
                    ),
                )
            )
        return self._omit_ambiguous_canonical_locators(units)

    @staticmethod
    def _omit_ambiguous_canonical_locators(
        units: list[DocumentUnit],
    ) -> tuple[DocumentUnit, ...]:
        aliases = [unit.canonical_locator for unit in units if unit.canonical_locator is not None]
        ambiguous_aliases = {locator for locator, count in Counter(aliases).items() if count > 1}
        if not ambiguous_aliases:
            return tuple(units)
        return tuple(
            replace(unit, canonical_locator=None)
            if _depends_on_ambiguous_alias(unit.canonical_locator, ambiguous_aliases)
            else unit
            for unit in units
        )

    @staticmethod
    def _marker_kind(marker: str) -> UnitKind:
        folded = marker.casefold()
        if "u.s.c." in folded or "cfr" in folded:
            return UnitKind.SECTION
        prefixes = (
            (("title ", "t\u00edtulo "), UnitKind.TITLE),
            (("chapter ", "cap\u00edtulo "), UnitKind.CHAPTER),
            (("section ", "se\u00e7\u00e3o "), UnitKind.SECTION),
            (("article ", "artigo ", "art. "), UnitKind.ARTICLE),
        )
        for candidates, kind in prefixes:
            if folded.startswith(candidates):
                return kind
        if folded.startswith("recital "):
            return UnitKind.RECITAL
        if (
            folded.startswith("\u00a7")
            or folded[0].isdigit()
            or (folded.startswith("(") and folded[1].isdigit())
        ):
            return UnitKind.PARAGRAPH
        return UnitKind.ITEM

    @staticmethod
    def _hierarchy_level(kind: UnitKind, marker: str) -> int:
        levels = {
            UnitKind.TITLE: 1,
            UnitKind.CHAPTER: 2,
            UnitKind.SECTION: 3,
            UnitKind.ARTICLE: 4,
            UnitKind.RECITAL: 4,
            UnitKind.PARAGRAPH: 5,
        }
        if kind is not UnitKind.ITEM:
            return levels[kind]
        return 6 if re.match(r"^[IVXLCDM]+\s*[-\u2013\u2014]", marker, re.IGNORECASE) else 7

    def _block_units(self, text: str, text_hash: Sha256, root_id: UUID) -> tuple[DocumentUnit, ...]:
        units: list[DocumentUnit] = []
        offset = 0
        for ordinal, block in enumerate(text.split("\n")):
            start = text.index(block, offset)
            end = start + len(block)
            locator = f"block-{ordinal + 1}"
            units.append(
                self._unit(
                    text,
                    _UnitSpec(
                        self._unit_id(text_hash, locator),
                        root_id,
                        UnitKind.BLOCK,
                        ordinal,
                        start,
                        end,
                        locator,
                    ),
                )
            )
            offset = end
        return tuple(units)

    @staticmethod
    def _unit(text: str, spec: _UnitSpec) -> DocumentUnit:
        return DocumentUnit(
            id=spec.id,
            parent_id=spec.parent_id,
            kind=spec.kind,
            ordinal=spec.ordinal,
            marker=spec.marker,
            label=spec.marker,
            start_offset=spec.start,
            end_offset=spec.end,
            start_page=None,
            end_page=None,
            locator=spec.locator,
            content_sha256=Sha256.from_bytes(text[spec.start : spec.end].encode("utf-8")),
            canonical_locator=spec.canonical_locator,
        )

    @staticmethod
    def _unit_id(text_hash: Sha256, locator: str) -> UUID:
        return uuid5(NAMESPACE_URL, f"norvii:{text_hash}:{locator}")


def _depends_on_ambiguous_alias(alias: str | None, ambiguous_aliases: set[str]) -> bool:
    if alias is None:
        return False
    return any(
        alias == candidate or alias.startswith(f"{candidate}/") for candidate in ambiguous_aliases
    )
