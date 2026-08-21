from __future__ import annotations

import os
from dataclasses import replace
from datetime import UTC, datetime, timedelta
from typing import TYPE_CHECKING
from uuid import UUID, uuid4

import pytest

from norvii_ingestion.domain.artifacts import PublicationCommand
from norvii_ingestion.domain.models import (
    FailureCategory,
    IngestionWork,
    OriginCapture,
    SafeFailure,
    Sha256,
)
from norvii_ingestion.enrichment.chunking import LegalChunker
from norvii_ingestion.extraction.html import HtmlExtractor
from norvii_ingestion.publication.persistence.config import EnvironmentConfigurationLoader
from norvii_ingestion.publication.postgres.repository import PostgresWorkRepository

if TYPE_CHECKING:
    import psycopg

pytestmark = pytest.mark.integration


def test_expired_lease_recovers_and_reprocessing_is_idempotent() -> None:
    configuration = EnvironmentConfigurationLoader(os.environ).load()
    repository = PostgresWorkRepository.connect(
        configuration.postgres, configuration.timeout_seconds
    )
    corpus_id, source_id, work_id = uuid4(), uuid4(), uuid4()
    now = datetime(2020, 1, 2, tzinfo=UTC)
    try:
        _insert_expired_claim(repository.connection, corpus_id, source_id, work_id)
        first = repository.claim("recovery-worker", timedelta(minutes=2), now)
        assert first is not None
        assert first.claim.work_id == work_id
        renewed = repository.renew(first, timedelta(minutes=3), now + timedelta(seconds=10))
        assert renewed == now + timedelta(minutes=3, seconds=10)

        first_document = _publish(repository, first, _html("First"), now + timedelta(seconds=20))
        same_work = _queue(repository.connection, corpus_id, source_id, now + timedelta(minutes=4))
        same = repository.claim("recovery-worker", timedelta(minutes=2), now + timedelta(minutes=4))
        assert same is not None
        assert same.claim.work_id == same_work
        same_document = _publish(
            repository, same, _html("First"), now + timedelta(minutes=4, seconds=10)
        )
        assert same_document == first_document

        changed_work = _queue(
            repository.connection, corpus_id, source_id, now + timedelta(minutes=7)
        )
        changed = repository.claim(
            "recovery-worker", timedelta(minutes=2), now + timedelta(minutes=7)
        )
        assert changed is not None
        assert changed.claim.work_id == changed_work
        changed_document = _publish(
            repository, changed, _html("Changed"), now + timedelta(minutes=7, seconds=10)
        )
        assert changed_document != first_document

        failed_work = _queue(
            repository.connection, corpus_id, source_id, now + timedelta(minutes=10)
        )
        failed = repository.claim(
            "recovery-worker", timedelta(minutes=2), now + timedelta(minutes=10)
        )
        assert failed is not None
        assert failed.claim.work_id == failed_work
        repository.fail(
            failed,
            SafeFailure(FailureCategory.EXTRACTION_FAILED),
            now + timedelta(minutes=10, seconds=10),
        )

        with repository.connection.cursor() as cursor:
            cursor.execute(
                """
                SELECT processing_status, latest_ready_document_id,
                       (SELECT count(*) FROM source_revisions WHERE source_id = %s),
                       (SELECT count(*) FROM document_versions WHERE source_id = %s),
                       (SELECT count(*) FROM processing_attempts WHERE source_id = %s),
                       (SELECT bool_and(duration_milliseconds > 0)
                        FROM processing_attempts WHERE source_id = %s)
                FROM sources WHERE id = %s
                """,
                (source_id, source_id, source_id, source_id, source_id),
            )
            row = cursor.fetchone()
        assert row == ("failed", changed_document, 2, 2, 5, True)
    finally:
        _cleanup(repository.connection, corpus_id)
        repository.close()


def _insert_expired_claim(
    connection: psycopg.Connection[tuple[object, ...]],
    corpus_id: UUID,
    source_id: UUID,
    work_id: UUID,
) -> None:
    lease_token, attempt_id = uuid4(), uuid4()
    with connection.transaction(), connection.cursor() as cursor:
        cursor.execute(
            """
            INSERT INTO corpora (id, name, description, language, jurisdiction)
            VALUES (%s, 'Recovery integration', 'Controlled recovery corpus.', 'en', 'Test')
            """,
            (corpus_id,),
        )
        cursor.execute(
            """
            INSERT INTO sources (
                id, corpus_id, title, kind, processing_status, version, created_at, updated_at
            ) VALUES (%s, %s, 'Recovery URL', 'url', 'processing', 2, %s, %s)
            """,
            (
                source_id,
                corpus_id,
                datetime(2019, 1, 1, tzinfo=UTC),
                datetime(2019, 1, 1, tzinfo=UTC),
            ),
        )
        cursor.execute(
            """
            INSERT INTO url_origins (source_id, corpus_id, submitted_url, normalized_url)
            VALUES (%s, %s, 'https://example.org/recovery', 'https://example.org/recovery')
            """,
            (source_id, corpus_id),
        )
        cursor.execute(
            """
            INSERT INTO ingestion_work (
                id, source_id, corpus_id, reason, status, requested_at,
                lease_token, worker_id, lease_expires_at, created_at, updated_at
            ) VALUES (%s, %s, %s, 'initial', 'leased', %s, %s, 'crashed-worker', %s, %s, %s)
            """,
            (
                work_id,
                source_id,
                corpus_id,
                datetime(2019, 1, 1, tzinfo=UTC),
                lease_token,
                datetime(2019, 1, 2, tzinfo=UTC),
                datetime(2019, 1, 1, tzinfo=UTC),
                datetime(2019, 1, 1, tzinfo=UTC),
            ),
        )
        cursor.execute(
            """
            INSERT INTO processing_attempts (
                id, work_id, source_id, corpus_id, attempt_number, pipeline_version,
                status, lease_token, worker_id, started_at
            ) VALUES (%s, %s, %s, %s, 1, 'corpus-ingestion-v1',
                      'processing', %s, 'crashed-worker', %s)
            """,
            (
                attempt_id,
                work_id,
                source_id,
                corpus_id,
                lease_token,
                datetime(2019, 1, 1, tzinfo=UTC),
            ),
        )


def _queue(
    connection: psycopg.Connection[tuple[object, ...]],
    corpus_id: UUID,
    source_id: UUID,
    requested_at: datetime,
) -> UUID:
    work_id = uuid4()
    with connection.transaction(), connection.cursor() as cursor:
        cursor.execute(
            """
            UPDATE sources SET processing_status = 'pending', version = version + 1
            WHERE id = %s AND corpus_id = %s
            """,
            (source_id, corpus_id),
        )
        cursor.execute(
            """
            INSERT INTO ingestion_work (id, source_id, corpus_id, reason, requested_at)
            VALUES (%s, %s, %s, 'reprocess', %s)
            """,
            (work_id, source_id, corpus_id, requested_at),
        )
    return work_id


def _publish(
    repository: PostgresWorkRepository,
    work: IngestionWork,
    content: bytes,
    now: datetime,
) -> UUID:
    artifact = HtmlExtractor().extract(content)
    capture = OriginCapture(
        content_sha256=Sha256.from_bytes(content),
        captured_at=now,
        media_type="text/html",
        byte_size=len(content),
        final_url="https://example.org/recovery",
    )
    return repository.publish(
        work,
        capture,
        PublicationCommand(
            work_id=work.claim.work_id,
            lease_token=work.claim.lease_token,
            pipeline_version="corpus-ingestion-v1",
            origin_sha256=capture.content_sha256,
            artifact=artifact,
            retrieval_chunks=tuple(
                replace(chunk, embedding=(0.0,) * 1536, embedding_model="test-embedding")
                for chunk in LegalChunker().chunk(artifact)
            ),
        ),
        now,
    )


def _html(version: str) -> bytes:
    return (
        "<main><h1>Article 1 Recovery</h1><p>"
        + (f"{version} controlled legal content for immutable publication. " * 20)
        + "</p></main>"
    ).encode()


def _cleanup(connection: psycopg.Connection[tuple[object, ...]], corpus_id: UUID) -> None:
    connection.rollback()
    with connection.transaction(), connection.cursor() as cursor:
        cursor.execute("DELETE FROM retrieval_chunks WHERE corpus_id = %s", (corpus_id,))
        cursor.execute(
            """
            DELETE FROM document_units
            WHERE document_id IN (
                SELECT id FROM document_versions WHERE corpus_id = %s
            )
            """,
            (corpus_id,),
        )
        cursor.execute(
            "UPDATE sources SET latest_ready_document_id = NULL WHERE corpus_id = %s", (corpus_id,)
        )
        for table in (
            "document_versions",
            "source_revisions",
            "processing_attempts",
            "ingestion_work",
            "url_origins",
            "sources",
            "corpora",
        ):
            column = "id" if table == "corpora" else "corpus_id"
            cursor.execute(f"DELETE FROM {table} WHERE {column} = %s", (corpus_id,))  # noqa: S608
