from __future__ import annotations

import hashlib
from dataclasses import dataclass
from uuid import UUID, uuid4

import pytest

from norvii_ingestion.domain.artifacts import (
    DocumentArtifact,
    DocumentUnit,
    PublicationCommand,
    UnitKind,
)
from norvii_ingestion.domain.models import Sha256


def test_document_artifact_accepts_complete_ordered_hierarchy() -> None:
    text = "Article 1\nFirst rule.\nArticle 2\nSecond rule."
    root_id = uuid4()
    artifact = DocumentArtifact(
        text=text,
        text_sha256=_hash(text),
        units=(
            _unit(text, UnitSpec(root_id, None, UnitKind.DOCUMENT, 0, 0, len(text), "document")),
            _unit(text, UnitSpec(uuid4(), root_id, UnitKind.ARTICLE, 0, 0, 22, "article-1")),
            _unit(
                text, UnitSpec(uuid4(), root_id, UnitKind.ARTICLE, 1, 22, len(text), "article-2")
            ),
        ),
    )

    artifact.validate()


def test_document_artifact_rejects_overlapping_peer_units() -> None:
    text = "abcdefghij"
    root_id = uuid4()
    artifact = DocumentArtifact(
        text=text,
        text_sha256=_hash(text),
        units=(
            _unit(text, UnitSpec(root_id, None, UnitKind.DOCUMENT, 0, 0, 10, "document")),
            _unit(text, UnitSpec(uuid4(), root_id, UnitKind.BLOCK, 0, 0, 6, "block-1")),
            _unit(text, UnitSpec(uuid4(), root_id, UnitKind.BLOCK, 1, 5, 10, "block-2")),
        ),
    )

    with pytest.raises(ValueError, match="overlap"):
        artifact.validate()


def test_publication_command_rejects_incorrect_text_hash() -> None:
    artifact = DocumentArtifact(
        text="legal text",
        text_sha256=Sha256("a" * 64),
        units=(
            _unit(
                "legal text",
                UnitSpec(uuid4(), None, UnitKind.DOCUMENT, 0, 0, 10, "document"),
            ),
        ),
    )

    with pytest.raises(ValueError, match="text hash"):
        PublicationCommand(
            work_id=uuid4(),
            lease_token=uuid4(),
            pipeline_version="corpus-ingestion-v1",
            origin_sha256=_hash("origin"),
            artifact=artifact,
        ).validate()


@dataclass(frozen=True)
class UnitSpec:
    id: UUID
    parent_id: UUID | None
    kind: UnitKind
    ordinal: int
    start: int
    end: int
    locator: str


def _unit(text: str, spec: UnitSpec) -> DocumentUnit:
    return DocumentUnit(
        id=spec.id,
        parent_id=spec.parent_id,
        kind=spec.kind,
        ordinal=spec.ordinal,
        marker=None,
        label=None,
        start_offset=spec.start,
        end_offset=spec.end,
        start_page=None,
        end_page=None,
        locator=spec.locator,
        content_sha256=_hash(text[spec.start : spec.end]),
    )


def _hash(value: str) -> Sha256:
    return Sha256(hashlib.sha256(value.encode()).hexdigest())
