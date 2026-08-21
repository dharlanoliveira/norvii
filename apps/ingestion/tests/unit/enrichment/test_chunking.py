from __future__ import annotations

from hashlib import sha256
from itertools import pairwise
from uuid import uuid4

from norvii_ingestion.domain.artifacts import DocumentArtifact, DocumentUnit, UnitKind
from norvii_ingestion.domain.models import Sha256
from norvii_ingestion.enrichment.chunking import LegalChunker


def test_chunker_keeps_nested_article_context_and_bounded_offsets() -> None:
    text = "Article 1. The controller shall protect data. I - Keep records. II - Explain decisions."
    root_id = uuid4()
    article_id = uuid4()
    item_id = uuid4()
    artifact = DocumentArtifact(
        text=text,
        text_sha256=Sha256(sha256(text.encode()).hexdigest()),
        units=(
            DocumentUnit(
                root_id,
                None,
                UnitKind.DOCUMENT,
                0,
                None,
                None,
                0,
                len(text),
                None,
                None,
                "document",
                Sha256(sha256(text.encode()).hexdigest()),
            ),
            DocumentUnit(
                article_id,
                root_id,
                UnitKind.ARTICLE,
                0,
                "Article 1",
                "Article 1",
                0,
                len(text),
                None,
                None,
                "article-1",
                Sha256(sha256(text.encode()).hexdigest()),
            ),
            DocumentUnit(
                item_id,
                article_id,
                UnitKind.ITEM,
                0,
                "I",
                "Keep records",
                50,
                63,
                None,
                None,
                "article-1.item-1",
                Sha256(sha256(text[50:63].encode()).hexdigest()),
            ),
        ),
    )
    artifact.validate()

    chunks = LegalChunker(max_characters=24).chunk(artifact)

    assert chunks
    assert all(len(chunk.text) <= 24 for chunk in chunks)
    assert all(chunk.context_locator == "article-1" for chunk in chunks)
    assert all(
        chunk.text_sha256 == Sha256(sha256(chunk.text.encode()).hexdigest()) for chunk in chunks
    )
    assert chunks[0].start_offset == 0
    assert chunks[-1].end_offset == len(text)
    assert all(left.end_offset == right.start_offset for left, right in pairwise(chunks))
