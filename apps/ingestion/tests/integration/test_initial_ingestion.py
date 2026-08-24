from __future__ import annotations

import os
from dataclasses import replace
from datetime import UTC, datetime, timedelta
from typing import TYPE_CHECKING

import pytest

from norvii_ingestion.acquisition.https import HttpsAcquirer
from norvii_ingestion.config import WorkerConfig
from norvii_ingestion.domain.artifacts import PublicationCommand
from norvii_ingestion.domain.models import OriginCapture, Sha256
from norvii_ingestion.enrichment.chunking import LegalChunker
from norvii_ingestion.extraction.html import HtmlExtractor
from norvii_ingestion.publication.persistence.config import EnvironmentConfigurationLoader
from norvii_ingestion.publication.postgres.repository import PostgresWorkRepository

if TYPE_CHECKING:
    import psycopg


@pytest.mark.integration
def test_oldest_initial_work_is_leased_extracted_and_published_atomically() -> None:
    configuration = EnvironmentConfigurationLoader(os.environ).load()
    repository = PostgresWorkRepository.connect(
        configuration.postgres, configuration.timeout_seconds
    )
    now = datetime(2026, 8, 17, 12, 0, tzinfo=UTC)
    try:
        _restore_initial_state(repository.connection)
        claimed = repository.claim("integration-worker", timedelta(seconds=120), now)

        assert claimed is not None
        assert str(claimed.claim.work_id) == "30000000-0000-4000-8000-000000000001"
        assert claimed.url is not None
        assert claimed.url.startswith("https://")
        assert claimed.claim.lease_expires_at == now + timedelta(seconds=120)

        html = b"""
            <html><body><main>
              <h1>Data protection law</h1>
              <h2>Article 1</h2><p>This law protects personal data.</p>
            </main></body></html>
        """
        acquisition = HttpsAcquirer(
            WorkerConfig.from_environment({}),
            resolver=lambda _host, _port: ("93.184.216.34",),
            connection_factory=lambda *_args: FakeHTTPSConnection(html),
        ).acquire(claimed.url)
        artifact = HtmlExtractor().extract(acquisition.content)
        capture = OriginCapture(
            content_sha256=Sha256.from_bytes(acquisition.content),
            captured_at=now,
            media_type=acquisition.media_type,
            byte_size=len(acquisition.content),
            final_url=acquisition.final_url,
        )
        command = PublicationCommand(
            work_id=claimed.claim.work_id,
            lease_token=claimed.claim.lease_token,
            pipeline_version="corpus-ingestion-v1",
            origin_sha256=capture.content_sha256,
            artifact=artifact,
            retrieval_chunks=tuple(
                replace(chunk, embedding=(0.0,) * 1536, embedding_model="test-embedding")
                for chunk in LegalChunker().chunk(artifact)
            ),
        )

        document_id = repository.publish(claimed, capture, command, now)

        with repository.connection.cursor() as cursor:
            cursor.execute(
                """
                SELECT s.processing_status, s.latest_ready_document_id,
                       w.status, a.status,
                       (SELECT count(*) FROM source_revisions WHERE source_id = s.id),
                       (SELECT count(*) FROM document_versions WHERE source_id = s.id),
                       (SELECT count(*) FROM document_units WHERE document_id = %s),
                       (SELECT count(*) FROM retrieval_chunks WHERE document_id = %s
                          AND enrichment_status = 'ready'
                          AND vector_dims(embedding) = 1536
                          AND embedding_model = 'test-embedding')
                FROM sources s
                JOIN ingestion_work w ON w.source_id = s.id
                JOIN processing_attempts a ON a.work_id = w.id
                WHERE s.id = %s
                """,
                (document_id, document_id, claimed.claim.source_id),
            )
            row = cursor.fetchone()
        assert row == (
            "ready",
            document_id,
            "succeeded",
            "succeeded",
            1,
            1,
            len(artifact.units),
            len(LegalChunker().chunk(artifact)),
        )
    finally:
        _restore_initial_state(repository.connection)
        repository.close()


def _restore_initial_state(connection: psycopg.Connection[tuple[object, ...]]) -> None:
    connection.rollback()
    with connection.transaction(), connection.cursor() as cursor:
        cursor.execute("DELETE FROM corpus_snapshot_releases")
        cursor.execute("DELETE FROM corpus_snapshot_documents")
        cursor.execute("DELETE FROM corpus_snapshots")
        cursor.execute("UPDATE sources SET latest_ready_document_id = NULL")
        cursor.execute("DELETE FROM retrieval_chunks")
        cursor.execute("DELETE FROM document_units")
        cursor.execute("DELETE FROM document_versions")
        cursor.execute("DELETE FROM source_revisions")
        cursor.execute("DELETE FROM processing_attempts")
        cursor.execute(
            """
            UPDATE sources
            SET processing_status = 'pending', latest_failure_category = NULL, version = 1
            WHERE seed_key IS NOT NULL
            """
        )
        cursor.execute(
            """
            UPDATE ingestion_work
            SET status = 'pending', lease_token = NULL, worker_id = NULL,
                lease_expires_at = NULL,
                requested_at = CASE
                    WHEN id = '30000000-0000-4000-8000-000000000001'
                        THEN '2026-08-17T11:58:00Z'::timestamptz
                    ELSE '2026-08-17T11:59:00Z'::timestamptz
                END
            WHERE reason = 'initial'
            """
        )


class FakeHTTPSResponse:
    status = 200

    def __init__(self, content: bytes) -> None:
        self._content = content

    def getheader(self, name: str) -> str | None:
        headers = {
            "Content-Type": "text/html; charset=utf-8",
            "Content-Length": str(len(self._content)),
        }
        return headers.get(name)

    def read(self, amount: int = -1) -> bytes:
        if amount < 0:
            return self._content
        content, self._content = self._content[:amount], self._content[amount:]
        return content


class FakeHTTPSConnection:
    def __init__(self, content: bytes) -> None:
        self._response = FakeHTTPSResponse(content)

    def request(self, method: str, target: str, *, headers: dict[str, str]) -> None:
        assert method == "GET"
        assert target.startswith("/")
        assert "Host" in headers

    def getresponse(self) -> FakeHTTPSResponse:
        return self._response

    def close(self) -> None:
        return None
