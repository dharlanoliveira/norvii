from __future__ import annotations

from dataclasses import dataclass
from datetime import UTC, datetime
from uuid import UUID, uuid4

import psycopg
import pytest

from norvii_agent.config import AgentConfig
from norvii_agent.retrieval import PostgresRetriever

pytestmark = pytest.mark.integration


class FixedEmbeddingProvider:
    def embed(self, texts: tuple[str, ...]) -> tuple[tuple[float, ...], ...]:
        assert texts == ("Which evidence is published?",)
        return (tuple(0.0 for _ in range(1536)),)


def test_retrieval_uses_only_the_declared_snapshot_membership() -> None:
    configuration = AgentConfig.from_environment()
    fixture = SnapshotFixture(uuid4(), uuid4())
    snapshot_id = uuid4()
    with psycopg.connect(
        host=configuration.postgres_host,
        port=configuration.postgres_port,
        dbname=configuration.postgres_database,
        user=configuration.postgres_user,
        password=configuration.postgres_password,
        connect_timeout=5,
    ) as connection:
        try:
            fixture.seed(connection)
            published_document_id, published_revision_id = fixture.insert_document(
                connection, "published", "a", "c"
            )
            excluded_document_id, _ = fixture.insert_document(connection, "candidate", "b", "d")
            fixture.insert_snapshot(
                connection,
                snapshot_id,
                published_document_id,
                published_revision_id,
            )
            connection.commit()

            evidence = PostgresRetriever(configuration, FixedEmbeddingProvider()).search(
                fixture.corpus_id, snapshot_id, "Which evidence is published?"
            )

            assert [item.document_id for item in evidence] == [published_document_id]
            assert evidence[0].snapshot_id == snapshot_id
            assert excluded_document_id not in [item.document_id for item in evidence]
        finally:
            fixture.delete(connection)


@dataclass(frozen=True)
class SnapshotFixture:
    corpus_id: UUID
    source_id: UUID

    def seed(self, connection: psycopg.Connection[tuple[object, ...]]) -> None:
        with connection.cursor() as cursor:
            cursor.execute(
                """
                INSERT INTO corpora (id, name, description, language, jurisdiction)
                VALUES (%s, 'Agent snapshot fixture',
                        'Agent retrieval integration fixture.', 'en', 'Test')
                """,
                (self.corpus_id,),
            )
            cursor.execute(
                """
                INSERT INTO sources (id, corpus_id, title, kind, processing_status)
                VALUES (%s, %s, 'Agent snapshot source', 'url', 'ready')
                """,
                (self.source_id, self.corpus_id),
            )
            cursor.execute(
                """
                INSERT INTO url_origins (source_id, corpus_id, submitted_url, normalized_url)
                VALUES (%s, %s, 'https://example.org/agent-snapshot',
                        'https://example.org/agent-snapshot')
                """,
                (self.source_id, self.corpus_id),
            )

    def insert_document(
        self,
        connection: psycopg.Connection[tuple[object, ...]],
        label: str,
        content_hash_character: str,
        text_hash_character: str,
    ) -> tuple[UUID, UUID]:
        work_id, attempt_id, revision_id, document_id, unit_id, chunk_id = (
            uuid4(),
            uuid4(),
            uuid4(),
            uuid4(),
            uuid4(),
            uuid4(),
        )
        captured_at = datetime(2026, 8, 24, 16, 0, tzinfo=UTC)
        content_hash = content_hash_character * 64
        text_hash = text_hash_character * 64
        text = f"{label} snapshot evidence."
        with connection.cursor() as cursor:
            cursor.execute(
                """
                INSERT INTO ingestion_work (id, source_id, corpus_id, reason, status)
                VALUES (%s, %s, %s, 'reprocess', 'succeeded')
                """,
                (work_id, self.source_id, self.corpus_id),
            )
            cursor.execute(
                """
                INSERT INTO processing_attempts (
                    id, work_id, source_id, corpus_id, attempt_number, pipeline_version, status,
                    lease_token, worker_id, started_at, finished_at
                ) VALUES (%s, %s, %s, %s, 1, 'agent-snapshot-test', 'succeeded', %s,
                          'integration-test', %s, %s)
                """,
                (
                    attempt_id,
                    work_id,
                    self.source_id,
                    self.corpus_id,
                    uuid4(),
                    captured_at,
                    captured_at,
                ),
            )
            cursor.execute(
                """
                INSERT INTO source_revisions (
                    id, source_id, corpus_id, attempt_id, content_sha256, captured_at, media_type,
                    byte_size, pipeline_version, final_url, extracted_content_sha256
                ) VALUES (%s, %s, %s, %s, %s, %s, 'text/html', 120, 'agent-snapshot-test',
                          'https://example.org/agent-snapshot', %s)
                """,
                (
                    revision_id,
                    self.source_id,
                    self.corpus_id,
                    attempt_id,
                    content_hash,
                    captured_at,
                    content_hash,
                ),
            )
            cursor.execute(
                """
                INSERT INTO document_versions (
                    id, source_revision_id, source_id, corpus_id, pipeline_version, text_content,
                    text_sha256, published_at
                ) VALUES (%s, %s, %s, %s, 'agent-snapshot-test', %s, %s, %s)
                """,
                (
                    document_id,
                    revision_id,
                    self.source_id,
                    self.corpus_id,
                    text,
                    text_hash,
                    captured_at,
                ),
            )
            cursor.execute(
                """
                INSERT INTO document_units (
                    id, document_id, kind, ordinal, start_offset, end_offset, locator,
                    content_sha256
                ) VALUES (%s, %s, 'article', 1, 0, 28, 'article-1', %s)
                """,
                (unit_id, document_id, text_hash),
            )
            cursor.execute(
                """
                INSERT INTO retrieval_chunks (
                    id, corpus_id, source_id, document_id, unit_id, ordinal, start_offset,
                    end_offset, content, content_sha256, context_locator, embedding,
                    embedding_model, enrichment_status
                ) VALUES (%s, %s, %s, %s, %s, 1, 0, 28, %s, %s, 'article-1', %s::vector,
                          'agent-snapshot-test', 'ready')
                """,
                (
                    chunk_id,
                    self.corpus_id,
                    self.source_id,
                    document_id,
                    unit_id,
                    text,
                    text_hash,
                    _zero_vector(),
                ),
            )
        return document_id, revision_id

    def insert_snapshot(
        self,
        connection: psycopg.Connection[tuple[object, ...]],
        snapshot_id: UUID,
        document_id: UUID,
        revision_id: UUID,
    ) -> None:
        captured_at = datetime(2026, 8, 24, 16, 0, tzinfo=UTC)
        with connection.cursor() as cursor:
            cursor.execute(
                """
                INSERT INTO corpus_snapshots (
                    id, corpus_id, manifest_sha256, created_by, created_at
                )
                VALUES (%s, %s, %s, 'integration-test', %s)
                """,
                (snapshot_id, self.corpus_id, "e" * 64, captured_at),
            )
            cursor.execute(
                """
                INSERT INTO corpus_snapshot_documents (
                    snapshot_id, corpus_id, source_id, source_revision_id, document_id,
                    official_origin, captured_at, content_sha256
                ) VALUES (%s, %s, %s, %s, %s, 'https://example.org/agent-snapshot', %s, %s)
                """,
                (
                    snapshot_id,
                    self.corpus_id,
                    self.source_id,
                    revision_id,
                    document_id,
                    captured_at,
                    "a" * 64,
                ),
            )

    def delete(self, connection: psycopg.Connection[tuple[object, ...]]) -> None:
        connection.rollback()
        with connection.transaction(), connection.cursor() as cursor:
            cursor.execute(
                "DELETE FROM corpus_snapshot_documents WHERE corpus_id = %s",
                (self.corpus_id,),
            )
            cursor.execute("DELETE FROM corpus_snapshots WHERE corpus_id = %s", (self.corpus_id,))
            cursor.execute("DELETE FROM retrieval_chunks WHERE corpus_id = %s", (self.corpus_id,))
            cursor.execute(
                """
                DELETE FROM document_units
                WHERE document_id IN (
                    SELECT id FROM document_versions WHERE corpus_id = %s
                )
                """,
                (self.corpus_id,),
            )
            cursor.execute(
                "UPDATE sources SET latest_ready_document_id = NULL WHERE corpus_id = %s",
                (self.corpus_id,),
            )
            cursor.execute("DELETE FROM document_versions WHERE corpus_id = %s", (self.corpus_id,))
            cursor.execute("DELETE FROM source_revisions WHERE corpus_id = %s", (self.corpus_id,))
            cursor.execute(
                "DELETE FROM processing_attempts WHERE corpus_id = %s", (self.corpus_id,)
            )
            cursor.execute("DELETE FROM ingestion_work WHERE corpus_id = %s", (self.corpus_id,))
            cursor.execute("DELETE FROM url_origins WHERE corpus_id = %s", (self.corpus_id,))
            cursor.execute("DELETE FROM sources WHERE corpus_id = %s", (self.corpus_id,))
            cursor.execute("DELETE FROM corpora WHERE id = %s", (self.corpus_id,))


def _zero_vector() -> str:
    return "[" + ",".join("0" for _ in range(1536)) + "]"
