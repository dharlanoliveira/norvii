from __future__ import annotations

from uuid import UUID

import psycopg

from norvii_ingestion.domain.models import Sha256
from norvii_ingestion.publication.postgres.repository import (
    _document_scoped_semantic_extraction,
    _repository_error_detail,
    retrieval_chunk_id_for_document,
)
from norvii_ingestion.semantic import SemanticEntity, SemanticExtraction, SemanticRelationship


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
                entity_type="location",
                label="document",
                normalized_label="document",
            ),
            SemanticEntity(
                id=second_entity_id,
                evidence_unit_id=UUID("70000000-0000-4000-8000-000000000001"),
                entity_type="location",
                label="Article 1",
                normalized_label="article 1",
            ),
        ),
        relationships=(
            SemanticRelationship(
                id=UUID("80000000-0000-4000-8000-000000000001"),
                subject_entity_id=first_entity_id,
                object_entity_id=second_entity_id,
                evidence_unit_id=UUID("70000000-0000-4000-8000-000000000001"),
                relationship_type="contains",
            ),
        ),
    )

    first = _document_scoped_semantic_extraction(first_document_id, extraction)
    second = _document_scoped_semantic_extraction(second_document_id, extraction)

    assert first.id != second.id
    assert {entity.id for entity in first.entities}.isdisjoint(
        {entity.id for entity in second.entities}
    )
    assert first.relationships[0].subject_entity_id in {entity.id for entity in first.entities}
    assert first.relationships[0].object_entity_id in {entity.id for entity in first.entities}


def test_repository_error_detail_contains_only_the_sqlstate() -> None:
    error = psycopg.errors.UniqueViolation("duplicate key with source content")

    assert _repository_error_detail(error) == "repository_sqlstate_23505"
