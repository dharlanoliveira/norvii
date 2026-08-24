from __future__ import annotations

import hashlib
from uuid import uuid4

import pytest

from norvii_ingestion.domain.artifacts import DocumentArtifact, DocumentUnit, UnitKind
from norvii_ingestion.domain.models import Sha256
from norvii_ingestion.semantic.extraction import ExtractionProviderError, _validated_batch


def test_validated_batch_creates_evidence_backed_entities_and_relationships() -> None:
    unit = _article_unit()
    entities, relationships, usage = _validated_batch(
        {
            "content": {
                "entities": [
                    {"unitId": str(unit.id), "type": "actor", "label": "Controller"},
                    {"unitId": str(unit.id), "type": "right", "label": "Access"},
                ],
                "relationships": [
                    {
                        "unitId": str(unit.id),
                        "type": "grants",
                        "subject": "Controller",
                        "object": "Access",
                    }
                ],
            },
            "usage": {"prompt_tokens": 11, "completion_tokens": 7},
        },
        (unit,),
    )

    assert len(entities) == 2
    assert relationships[0].evidence_unit_id == unit.id
    assert usage == (11, 7)


def test_validated_batch_rejects_relationships_without_declared_entities() -> None:
    unit = _article_unit()

    with pytest.raises(ExtractionProviderError, match="not evidence-backed"):
        _validated_batch(
            {
                "content": {
                    "entities": [],
                    "relationships": [
                        {
                            "unitId": str(unit.id),
                            "type": "grants",
                            "subject": "Controller",
                            "object": "Access",
                        }
                    ],
                }
            },
            (unit,),
        )


def _article_unit() -> DocumentUnit:
    text = "Article 1 grants access."
    artifact = DocumentArtifact(
        text=text,
        text_sha256=Sha256.from_bytes(text.encode()),
        units=(
            DocumentUnit(
                id=uuid4(),
                parent_id=None,
                kind=UnitKind.DOCUMENT,
                ordinal=0,
                marker=None,
                label=None,
                start_offset=0,
                end_offset=len(text),
                start_page=None,
                end_page=None,
                locator="document",
                content_sha256=Sha256.from_bytes(text.encode()),
            ),
        ),
    )
    artifact.validate()
    return DocumentUnit(
        id=uuid4(),
        parent_id=artifact.units[0].id,
        kind=UnitKind.ARTICLE,
        ordinal=0,
        marker="1",
        label="Article 1",
        start_offset=0,
        end_offset=len(text),
        start_page=None,
        end_page=None,
        locator="Article 1",
        content_sha256=Sha256(hashlib.sha256(text.encode()).hexdigest()),
    )
