"""Deterministic text-PDF extraction with page and legal units."""

from __future__ import annotations

import re
from dataclasses import dataclass
from io import BytesIO
from typing import TYPE_CHECKING, Protocol, cast
from uuid import NAMESPACE_URL, UUID, uuid5

from pypdf import PdfReader
from pypdf.errors import PdfReadError

from norvii_ingestion.domain.artifacts import DocumentArtifact, DocumentUnit, UnitKind
from norvii_ingestion.domain.models import Sha256
from norvii_ingestion.extraction.legal_locator import canonical_legal_locator

if TYPE_CHECKING:
    from collections.abc import Callable

_LEGAL_MARKER = re.compile(
    r"^(?P<marker>"
    r"(?:\d+\s+U\.S\.C\.\s+\u00a7\s*\d+(?:\.\d+)*(?:\([A-Za-z0-9]+\))*"
    r"|\d+\s+CFR\s+\u00a7\s*\d+(?:\.\d+)*(?:\([A-Za-z0-9]+\))*)"
    r"|(?:Article|Artigo|Art\.)\s+[^\n]+"
    r"|(?:Section|Se\u00e7\u00e3o)\s+[^\n]+"
    r")",
    re.IGNORECASE | re.MULTILINE,
)


class PdfExtractionError(ValueError):
    """Report an invalid, encrypted, image-only, or empty PDF."""


class PdfPage(Protocol):
    """Expose page text without coupling tests to pypdf internals."""

    def extract_text(self, *, extraction_mode: str) -> str | None:
        """Extract page text."""
        ...


class PdfDocument(Protocol):
    """Expose the bounded reader state required by extraction."""

    @property
    def is_encrypted(self) -> bool:
        """Return whether credentials are required to read content."""
        ...

    @property
    def pages(self) -> list[PdfPage]:
        """Return pages in source order."""
        ...


@dataclass(frozen=True, slots=True)
class _Span:
    start: int
    end: int
    page: int
    text: str


@dataclass(frozen=True, slots=True)
class _PageContext:
    id: UUID
    number: int
    ordinal: int
    start: int
    end: int


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
    start_page: int | None = None
    end_page: int | None = None
    canonical_locator: str | None = None


class PdfExtractor:
    """Extract complete normalized text from a text-based PDF."""

    def __init__(self, reader_factory: Callable[[BytesIO], PdfDocument] | None = None) -> None:
        self._reader_factory = reader_factory or _read_pdf

    def extract(self, content: bytes) -> DocumentArtifact:
        """Return one complete document with page-first stable hierarchy."""
        try:
            reader = self._reader_factory(BytesIO(content))
        except (PdfReadError, OSError, ValueError) as error:
            raise PdfExtractionError("PDF content is invalid.") from error
        if reader.is_encrypted:
            raise PdfExtractionError("Encrypted PDFs are not supported.")
        spans = self._page_spans(reader)
        if not spans:
            raise PdfExtractionError("PDF does not contain extractable text.")
        text = "\n\n".join(span.text for span in spans)
        text_hash = Sha256.from_bytes(text.encode())
        root_id = self._unit_id(text_hash, "document")
        units: list[DocumentUnit] = [
            self._unit(
                text,
                _UnitSpec(root_id, None, UnitKind.DOCUMENT, 0, 0, len(text), "document"),
            )
        ]
        offset = 0
        for ordinal, span in enumerate(spans):
            start = offset
            end = start + len(span.text)
            page_id = self._unit_id(text_hash, f"page-{span.page}")
            page = _PageContext(page_id, span.page, ordinal, start, end)
            units.append(
                self._unit(
                    text,
                    _UnitSpec(
                        page.id,
                        root_id,
                        UnitKind.PAGE,
                        page.ordinal,
                        page.start,
                        page.end,
                        f"page-{page.number}",
                        start_page=page.number,
                        end_page=page.number,
                    ),
                )
            )
            units.extend(self._legal_units(text, text_hash, page))
            offset = end + 2
        artifact = DocumentArtifact(text=text, text_sha256=text_hash, units=tuple(units))
        artifact.validate()
        return artifact

    @staticmethod
    def _page_spans(reader: PdfDocument) -> list[_Span]:
        spans: list[_Span] = []
        for page_number, page in enumerate(reader.pages, start=1):
            try:
                raw_text = page.extract_text(extraction_mode="layout")
            except (KeyError, TypeError, ValueError) as error:
                raise PdfExtractionError("PDF page text extraction failed.") from error
            text = "\n".join(
                line.strip()
                for line in (raw_text or "").replace("\r\n", "\n").split("\n")
                if line.strip()
            )
            if text:
                spans.append(_Span(0, len(text), page_number, text))
        return spans

    def _legal_units(
        self,
        text: str,
        text_hash: Sha256,
        page: _PageContext,
    ) -> list[DocumentUnit]:
        page_text = text[page.start : page.end]
        matches = list(_LEGAL_MARKER.finditer(page_text))
        units: list[DocumentUnit] = []
        for ordinal, match in enumerate(matches):
            unit_start = page.start + match.start()
            unit_end = page.start + (
                matches[ordinal + 1].start() if ordinal + 1 < len(matches) else len(page_text)
            )
            marker = match.group("marker").strip()
            kind = (
                UnitKind.SECTION
                if marker.casefold().startswith(("section ", "se\u00e7\u00e3o "))
                or "u.s.c." in marker.casefold()
                or "cfr" in marker.casefold()
                else UnitKind.ARTICLE
            )
            locator = f"page-{page.number}-{kind.value}-{ordinal + 1}"
            units.append(
                self._unit(
                    text,
                    _UnitSpec(
                        self._unit_id(text_hash, locator),
                        page.id,
                        kind,
                        ordinal,
                        unit_start,
                        unit_end,
                        locator,
                        marker=marker,
                        start_page=page.number,
                        end_page=page.number,
                        canonical_locator=canonical_legal_locator(kind.value, marker, None),
                    ),
                )
            )
        return units

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
            start_page=spec.start_page,
            end_page=spec.end_page,
            locator=spec.locator,
            content_sha256=Sha256.from_bytes(text[spec.start : spec.end].encode()),
            canonical_locator=spec.canonical_locator,
        )

    @staticmethod
    def _unit_id(text_hash: Sha256, locator: str) -> UUID:
        return uuid5(NAMESPACE_URL, f"norvii:{text_hash}:{locator}")


def _read_pdf(content: BytesIO) -> PdfDocument:
    return cast("PdfDocument", PdfReader(content))
