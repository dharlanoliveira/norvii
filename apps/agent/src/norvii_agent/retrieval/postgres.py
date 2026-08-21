"""Corpus-scoped retrieval for the LangGraph runtime."""

from __future__ import annotations

from typing import TYPE_CHECKING
from uuid import UUID

import psycopg

from norvii_agent.graph import Evidence

if TYPE_CHECKING:
    from norvii_agent.config import AgentConfig
    from norvii_agent.providers import EmbeddingProvider


class PostgresRetriever:
    """Retrieve only latest published chunks for an enabled corpus."""

    def __init__(self, configuration: AgentConfig, embeddings: EmbeddingProvider) -> None:
        self._configuration = configuration
        self._embeddings = embeddings

    def search(self, corpus_id: UUID, question: str) -> tuple[Evidence, ...]:
        """Return the nearest ready vectors within the active corpus boundary."""
        vectors = self._embeddings.embed((question,))
        if len(vectors) != 1:
            raise ValueError("embedding provider must return one question vector")
        vector_literal = "[" + ",".join(str(value) for value in vectors[0]) + "]"
        with (
            psycopg.connect(
                host=self._configuration.postgres_host,
                port=self._configuration.postgres_port,
                dbname=self._configuration.postgres_database,
                user=self._configuration.postgres_user,
                password=self._configuration.postgres_password,
                connect_timeout=5,
            ) as connection,
            connection.cursor() as cursor,
        ):
            cursor.execute(
                """
                SELECT c.id, c.corpus_id, c.source_id, c.document_id,
                       c.context_locator, c.start_offset, c.end_offset, c.content
                FROM retrieval_chunks c
                JOIN document_versions d ON d.id = c.document_id
                JOIN corpora co ON co.id = c.corpus_id AND co.status = 'enabled'
                JOIN sources s ON s.corpus_id = c.corpus_id
                 AND s.id = c.source_id AND s.latest_ready_document_id = c.document_id
                WHERE c.corpus_id = %s
                  AND d.publication_status = 'published'
                  AND c.enrichment_status = 'ready'
                  AND c.embedding IS NOT NULL
                ORDER BY c.embedding <=> %s::vector, c.ordinal, c.id
                LIMIT 8
                """,
                (corpus_id, vector_literal),
            )
            rows = cursor.fetchall()
        return tuple(
            Evidence(
                id=str(row[0]),
                corpus_id=row[1],
                source_id=row[2],
                document_id=row[3],
                unit_locator=row[4],
                start_offset=row[5],
                end_offset=row[6],
                excerpt=row[7],
                rank=index + 1,
            )
            for index, row in enumerate(rows)
        )
