from __future__ import annotations

import pytest

from norvii_ingestion.domain.artifacts import UnitKind
from norvii_ingestion.extraction.html import ExtractionError, HtmlExtractor


@pytest.mark.parametrize(
    ("html", "expected_kinds"),
    [
        (
            """
            <main><h1>Title I General rules</h1>
            <p>This title establishes the complete general legal framework and its safeguards.</p>
            <h2>Chapter I Scope</h2>
            <p>This chapter explains the scope of the official regulation in detail.</p>
            <h3>Section 1 Principles</h3>
            <p>This section defines the principles governing every covered legal operation.</p>
            <p>Article 1 Purpose and application</p>
            <p>1. This Regulation protects people and establishes legal safeguards.</p>
            <p>(a) personal data receives protection under the official framework.</p></main>
            """,
            {
                UnitKind.TITLE,
                UnitKind.CHAPTER,
                UnitKind.SECTION,
                UnitKind.ARTICLE,
                UnitKind.PARAGRAPH,
                UnitKind.ITEM,
            },
        ),
        (
            (
                "<main><h1>T\u00edtulo I Regras gerais</h1>"
                "<p>Este t\u00edtulo estabelece o marco jur\u00eddico geral e todas "
                "as suas garantias.</p><h2>Cap\u00edtulo I Escopo</h2>"
                "<p>Este cap\u00edtulo explica detalhadamente o escopo da "
                "legisla\u00e7\u00e3o oficial.</p><h3>Se\u00e7\u00e3o I Princ\u00edpios</h3>"
                "<p>Esta se\u00e7\u00e3o define os princ\u00edpios aplic\u00e1veis a cada "
                "opera\u00e7\u00e3o jur\u00eddica.</p>"
                "<p>Art. 1\u00ba Esta Lei disp\u00f5e sobre dados pessoais.</p>"
                "<p>\u00a7 1\u00ba Aplicam-se as garantias previstas nesta "
                "legisla\u00e7\u00e3o.</p>"
                "<p>a) dado pessoal recebe a prote\u00e7\u00e3o definida nesta norma "
                "oficial.</p></main>"
            ),
            {
                UnitKind.TITLE,
                UnitKind.CHAPTER,
                UnitKind.SECTION,
                UnitKind.ARTICLE,
                UnitKind.PARAGRAPH,
                UnitKind.ITEM,
            },
        ),
    ],
)
def test_extractor_preserves_complete_bilingual_legal_structure(
    html: str, expected_kinds: set[UnitKind]
) -> None:
    artifact = HtmlExtractor().extract(html.encode())

    artifact.validate()
    assert artifact.units[0].kind is UnitKind.DOCUMENT
    assert expected_kinds.issubset({unit.kind for unit in artifact.units})
    assert artifact.units[0].end_offset == len(artifact.text)


def test_extractor_uses_ordered_blocks_when_legal_markers_are_absent() -> None:
    artifact = HtmlExtractor().extract(
        b"<main><p>First official block contains the complete legal introduction.</p>"
        b"<p>Second official block contains the complete legal conclusion.</p></main>"
    )

    artifact.validate()
    assert [unit.kind for unit in artifact.units[1:]] == [UnitKind.BLOCK, UnitKind.BLOCK]


def test_extractor_supports_legacy_windows_1252_legal_html() -> None:
    html = (
        "<main><h1>Lei de Prote\u00e7\u00e3o de Dados</h1>"
        "<p>Art. 1\u00ba Esta Lei protege informa\u00e7\u00f5es pessoais e "
        "preserva direitos fundamentais, inclusive liberdade, privacidade e "
        "o livre desenvolvimento da personalidade da pessoa natural.</p></main>"
    )

    artifact = HtmlExtractor().extract(html.encode("windows-1252"))

    assert "informa\u00e7\u00f5es" in artifact.text
    assert any(unit.kind is UnitKind.ARTICLE for unit in artifact.units)


def test_extractor_recognizes_ascii_portuguese_article_ordinals() -> None:
    artifact = HtmlExtractor().extract(
        b"<main><p>Art. 1o This Law establishes general rules for personal-data "
        b"protection.</p><p>Art. 2o This Law protects fundamental rights.</p></main>"
    )

    articles = [unit for unit in artifact.units if unit.kind is UnitKind.ARTICLE]

    assert len(articles) == 2
    assert articles[0].start_offset == 0
    assert articles[0].end_offset == articles[1].start_offset


def test_extractor_preserves_brazilian_article_hierarchy() -> None:
    html = (
        "<main><p>CAP\u00cdTULO I DISPOSI\u00c7\u00d5ES PRELIMINARES</p>"
        "<p>Art. 3\u00ba Esta Lei aplica-se \u00e0s opera\u00e7\u00f5es de tratamento "
        "desde que:</p>"
        "<p>I - a opera\u00e7\u00e3o seja realizada no territ\u00f3rio nacional;</p>"
        "<p>II - os dados pessoais tenham sido coletados no territ\u00f3rio nacional.</p>"
        "<p>\u00a7 1\u00ba Aplicam-se as seguintes condi\u00e7\u00f5es complementares:</p>"
        "<p>I - o titular esteja no territ\u00f3rio nacional;</p>"
        "<p>a) no momento em que os dados pessoais forem coletados.</p>"
        "<p>Art. 4\u00ba Esta Lei n\u00e3o se aplica ao tratamento realizado para fins "
        "exclusivamente particulares e n\u00e3o econ\u00f4micos.</p></main>"
    )

    artifact = HtmlExtractor().extract(html.encode())
    first_article, second_article = [
        unit for unit in artifact.units if unit.kind is UnitKind.ARTICLE
    ]
    paragraphs = [unit for unit in artifact.units if unit.kind is UnitKind.PARAGRAPH]
    items = [unit for unit in artifact.units if unit.kind is UnitKind.ITEM]

    assert first_article.end_offset == second_article.start_offset
    assert paragraphs[0].parent_id == first_article.id
    assert [item.parent_id for item in items[:2]] == [first_article.id, first_article.id]
    assert items[2].parent_id == paragraphs[0].id
    assert items[3].parent_id == items[2].id


def test_extractor_derives_auditable_hierarchical_canonical_legal_locators() -> None:
    artifact = HtmlExtractor().extract(
        (
            "<main><p>Art. 3\u00ba This Law applies to processing operations.</p>"
            "<p>\u00a7 1\u00ba A supplementary condition applies.</p>"
            "<p>I - the data subject is in the national territory.</p>"
            "<p>a) at the time the personal data was collected.</p></main>"
        ).encode()
    )

    assert {
        unit.canonical_locator for unit in artifact.units if unit.canonical_locator is not None
    } == {
        "article:3",
        "article:3/paragraph:1",
        "article:3/paragraph:1/item:i",
        "article:3/paragraph:1/item:i/item:a",
    }
    assert artifact.resolve_canonical_legal_locator(" ARTICLE:3/PARAGRAPH:1 ").kind is (
        UnitKind.PARAGRAPH
    )


def test_extractor_preserves_suffixed_provisions_and_scopes_repeated_headings() -> None:
    artifact = HtmlExtractor().extract(
        (
            "<main><p>Title I First title.</p><p>Chapter I First chapter.</p>"
            "<p>Article 4 Main provision.</p><p>Article 4-A Supplementary provision.</p>"
            "<p>\u00a7 2\u00ba Main paragraph.</p><p>\u00a7 2\u00ba-B Supplementary paragraph.</p>"
            "<p>Title II Second title.</p><p>Chapter I Second chapter.</p>"
            "<p>Article 17-A A suffixed article.</p></main>"
        ).encode()
    )

    assert {
        unit.canonical_locator for unit in artifact.units if unit.canonical_locator is not None
    } == {
        "title:i",
        "title:i/chapter:i",
        "article:4",
        "article:4-a",
        "article:4-a/paragraph:2",
        "article:4-a/paragraph:2-b",
        "title:ii",
        "title:ii/chapter:i",
        "article:17-a",
    }


def test_extractor_omits_ambiguous_aliases_without_discarding_document_units() -> None:
    artifact = HtmlExtractor().extract(
        b"<main><p>Art. 17 First version.</p><p>I - First item.</p>"
        b"<p>Art. 17 Second version.</p><p>I - Second item.</p>"
        b"<p>Art. 18 Unambiguous provision.</p></main>"
    )

    article_markers = [unit for unit in artifact.units if unit.marker == "Art. 17"]

    assert len(article_markers) == 2
    assert all(unit.canonical_locator is None for unit in article_markers)
    assert all(
        unit.canonical_locator is None
        for unit in artifact.units
        if unit.parent_id in {article.id for article in article_markers}
    )
    assert artifact.resolve_canonical_legal_locator("article:18").marker == "Art. 18"


def test_extractor_maps_remaining_artifact_validation_errors_to_extraction_errors(
    monkeypatch: pytest.MonkeyPatch,
) -> None:
    def reject_artifact(_artifact: object) -> None:
        raise ValueError("test-only structural validation failure")

    monkeypatch.setattr(
        "norvii_ingestion.extraction.html.DocumentArtifact.validate", reject_artifact
    )
    extractor = HtmlExtractor()

    with pytest.raises(ExtractionError, match="structurally invalid"):
        extractor.extract(
            b"<main><p>First official block contains the complete legal introduction.</p>"
            b"<p>Second official block contains the complete legal conclusion.</p></main>"
        )


def test_extractor_rejects_empty_and_invalid_unicode_content() -> None:
    extractor = HtmlExtractor()

    with pytest.raises(ExtractionError):
        extractor.extract(b"<html><body></body></html>")
    with pytest.raises(ExtractionError):
        extractor.extract(b"\xff")
