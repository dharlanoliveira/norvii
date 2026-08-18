from __future__ import annotations

from typing import TYPE_CHECKING

import pytest
from pypdf.errors import PdfReadError

from norvii_ingestion.domain.artifacts import UnitKind
from norvii_ingestion.extraction.pdf import PdfExtractionError, PdfExtractor

if TYPE_CHECKING:
    from io import BytesIO


class FakePage:
    def __init__(self, text: str | None) -> None:
        self._text = text

    def extract_text(self, *, extraction_mode: str) -> str | None:
        assert extraction_mode == "layout"
        return self._text


class FakeReader:
    def __init__(self, pages: list[FakePage], *, encrypted: bool = False) -> None:
        self.pages = pages
        self.is_encrypted = encrypted


def test_pdf_extractor_preserves_pages_and_legal_children() -> None:
    reader = FakeReader(
        [
            FakePage("Section I Scope\nArticle 1 First rule"),
            FakePage("Art. 2\u00ba Segunda regra"),
        ]
    )
    artifact = PdfExtractor(lambda _content: reader).extract(b"%PDF-generated-test")

    artifact.validate()
    assert artifact.text == "Section I Scope\nArticle 1 First rule\n\nArt. 2\u00ba Segunda regra"
    assert [unit.kind for unit in artifact.units].count(UnitKind.PAGE) == 2
    assert {UnitKind.SECTION, UnitKind.ARTICLE}.issubset({unit.kind for unit in artifact.units})
    assert {unit.start_page for unit in artifact.units if unit.kind is UnitKind.PAGE} == {1, 2}


@pytest.mark.parametrize(
    "reader",
    [
        FakeReader([], encrypted=True),
        FakeReader([FakePage(None), FakePage("   ")]),
    ],
)
def test_pdf_extractor_rejects_encrypted_and_image_only_documents(reader: FakeReader) -> None:
    extractor = PdfExtractor(lambda _content: reader)

    with pytest.raises(PdfExtractionError):
        extractor.extract(b"%PDF-invalid-for-policy")


def test_pdf_extractor_rejects_malformed_documents() -> None:
    def fail(_content: BytesIO) -> FakeReader:
        raise PdfReadError("private parser detail")

    extractor = PdfExtractor(fail)

    with pytest.raises(PdfExtractionError, match="invalid"):
        extractor.extract(b"not a PDF")
