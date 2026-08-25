from __future__ import annotations

from uuid import UUID

import psycopg

from norvii_ingestion.publication.postgres.repository import (
    _repository_error_detail,
    retrieval_chunk_id_for_document,
)


def test_retrieval_chunk_identity_is_unique_per_document_version() -> None:
    logical_chunk_id = UUID("00000000-0000-4000-8000-000000000001")
    first_document_id = UUID("10000000-0000-4000-8000-000000000001")
    second_document_id = UUID("20000000-0000-4000-8000-000000000001")

    first_identity = retrieval_chunk_id_for_document(first_document_id, logical_chunk_id)

    assert first_identity == retrieval_chunk_id_for_document(first_document_id, logical_chunk_id)
    assert first_identity != retrieval_chunk_id_for_document(second_document_id, logical_chunk_id)


def test_repository_error_detail_contains_only_the_sqlstate() -> None:
    error = psycopg.errors.UniqueViolation("duplicate key with source content")

    assert _repository_error_detail(error) == "repository_sqlstate_23505"
