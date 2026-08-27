from __future__ import annotations

from uuid import UUID, uuid4

import psycopg

from norvii_ingestion.domain.artifacts import (
    DocumentArtifact,
    DocumentUnit,
    PublicationCommand,
    UnitKind,
)
from norvii_ingestion.domain.models import Sha256
from norvii_ingestion.publication.postgres.repository import (
    PostgresWorkRepository,
    _document_scoped_semantic_extraction,
    _repository_error_detail,
    retrieval_chunk_id_for_document,
)
from norvii_ingestion.semantic import SemanticAssertion, SemanticEntity, SemanticExtraction


def test_retrieval_chunk_identity_is_unique_per_document_version() -> None:
    logical_chunk_id = UUID("00000000-0000-4000-8000-000000000001")
    first_document_id = UUID("10000000-0000-4000-8000-000000000001")
    second_document_id = UUID("20000000-0000-4000-8000-000000000001")

    first_identity = retrieval_chunk_id_for_document(first_document_id, logical_chunk_id)

    assert first_identity == retrieval_chunk_id_for_document(first_document_id, logical_chunk_id)
    assert first_identity != retrieval_chunk_id_for_document(second_document_id, logical_chunk_id)


def test_semantic_artifact_identity_is_unique_per_document_version() -> None:
    first_document_id = UUID("10000000-0000-4000-8000-000000000001")
    second_document_id = UUID("20000000-0000-4000-8000-000000000001")
    first_entity_id = UUID("30000000-0000-4000-8000-000000000001")
    second_entity_id = UUID("40000000-0000-4000-8000-000000000001")
    extraction = SemanticExtraction(
        id=UUID("50000000-0000-4000-8000-000000000001"),
        extraction_version="test-v1",
        model_identifier="test-model",
        input_sha256=Sha256("0" * 64),
        input_tokens=1,
        output_tokens=1,
        duration_milliseconds=1,
        entities=(
            SemanticEntity(
                id=first_entity_id,
                evidence_unit_id=UUID("60000000-0000-4000-8000-000000000001"),
                entity_type="concept",
                label="Term",
                normalized_label="term",
            ),
            SemanticEntity(
                id=second_entity_id,
                evidence_unit_id=UUID("70000000-0000-4000-8000-000000000001"),
                entity_type="concept",
                label="Definition",
                normalized_label="definition",
            ),
        ),
        assertions=(
            SemanticAssertion(
                id=UUID("80000000-0000-4000-8000-000000000001"),
                subject_entity_id=first_entity_id,
                object_entity_id=second_entity_id,
                establishing_unit_id=UUID("70000000-0000-4000-8000-000000000001"),
                evidence_unit_id=UUID("70000000-0000-4000-8000-000000000001"),
                predicate="defines",
            ),
        ),
    )

    first = _document_scoped_semantic_extraction(first_document_id, extraction)
    second = _document_scoped_semantic_extraction(second_document_id, extraction)

    assert first.id != second.id
    assert {entity.id for entity in first.entities}.isdisjoint(
        {entity.id for entity in second.entities}
    )
    assert first.assertions[0].subject_entity_id in {entity.id for entity in first.entities}
    assert first.assertions[0].object_entity_id in {entity.id for entity in first.entities}


def test_repository_error_detail_contains_only_the_sqlstate() -> None:
    error = psycopg.errors.UniqueViolation("duplicate key with source content")

    assert _repository_error_detail(error) == "repository_sqlstate_23505"


def test_unit_publication_persists_the_immutable_canonical_legal_locator() -> None:
    text = "Article 19\nA clear statement."
    root_id = uuid4()
    article_id = uuid4()
    artifact = DocumentArtifact(
        text=text,
        text_sha256=Sha256.from_bytes(text.encode()),
        units=(
            DocumentUnit(
                id=root_id,
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
            DocumentUnit(
                id=article_id,
                parent_id=root_id,
                kind=UnitKind.ARTICLE,
                ordinal=0,
                marker="Article 19",
                label="Article 19",
                start_offset=0,
                end_offset=len(text),
                start_page=None,
                end_page=None,
                locator="article-1",
                content_sha256=Sha256.from_bytes(text.encode()),
                canonical_locator="article:19",
            ),
        ),
    )
    command = PublicationCommand(
        work_id=uuid4(),
        lease_token=uuid4(),
        pipeline_version="test-pipeline",
        origin_sha256=Sha256("0" * 64),
        artifact=artifact,
    )
    cursor = RecordingCursor()

    PostgresWorkRepository._insert_units(cursor, uuid4(), command)  # noqa: SLF001

    assert "canonical_locator" in cursor.query
    assert cursor.parameters[1][12] == "article:19"


class RecordingCursor:
    def __init__(self) -> None:
        self.query = ""
        self.parameters: list[tuple[object, ...]] = []

    def executemany(self, query: str, parameters: list[tuple[object, ...]]) -> None:
        self.query = query
        self.parameters = parameters
