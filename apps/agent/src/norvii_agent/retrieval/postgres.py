"""Corpus-scoped retrieval for the LangGraph runtime."""

from __future__ import annotations

import math
from typing import TYPE_CHECKING
from uuid import UUID

import psycopg

from norvii_agent.graph import Evidence, RetrievalInspection

if TYPE_CHECKING:
    from norvii_agent.config import AgentConfig
    from norvii_agent.providers import EmbeddingProvider

_DISTANCE_INDEX = 8
_SOURCE_REVISION_INDEX = 9
_PIPELINE_VERSION_INDEX = 10
_SOURCE_TITLE_INDEX = 11


class PostgresRetriever:
    """Retrieve only latest published chunks for an enabled corpus."""

    def __init__(self, configuration: AgentConfig, embeddings: EmbeddingProvider) -> None:
        self._configuration = configuration
        self._embeddings = embeddings
        self.last_retrieval: RetrievalInspection | None = None

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
                WITH ranked_chunks AS (
                    SELECT c.id, c.corpus_id, c.source_id, c.document_id,
                           c.context_locator, c.start_offset, c.end_offset, c.content,
                           c.embedding <=> %s::vector AS cosine_distance,
                           d.source_revision_id, d.pipeline_version, s.title, c.ordinal
                    FROM retrieval_chunks c
                    JOIN document_versions d ON d.id = c.document_id
                    JOIN corpora co ON co.id = c.corpus_id AND co.status = 'enabled'
                    JOIN sources s ON s.corpus_id = c.corpus_id
                     AND s.id = c.source_id AND s.latest_ready_document_id = c.document_id
                    WHERE c.corpus_id = %s
                      AND d.publication_status = 'published'
                      AND c.enrichment_status = 'ready'
                      AND c.embedding IS NOT NULL
                )
                SELECT id, corpus_id, source_id, document_id,
                       context_locator, start_offset, end_offset, content,
                       cosine_distance, source_revision_id, pipeline_version, title
                FROM ranked_chunks
                ORDER BY cosine_distance, ordinal, id
                LIMIT 8
                """,
                (vector_literal, corpus_id),
            )
            rows = cursor.fetchall()
        self.last_retrieval = RetrievalInspection(
            strategy="vector",
            top_k=8,
            returned_count=len(rows),
            embedding_model=self._configuration.embedding_model or None,
        )
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
                document_version_id=row[3],
                cosine_distance=_finite_distance(
                    row[_DISTANCE_INDEX] if len(row) > _DISTANCE_INDEX else None
                ),
                source_revision_id=(
                    row[_SOURCE_REVISION_INDEX] if len(row) > _SOURCE_REVISION_INDEX else None
                ),
                pipeline_version=(
                    row[_PIPELINE_VERSION_INDEX] if len(row) > _PIPELINE_VERSION_INDEX else None
                ),
                source_title=(row[_SOURCE_TITLE_INDEX] if len(row) > _SOURCE_TITLE_INDEX else None),
            )
            for index, row in enumerate(rows)
        )


def _finite_distance(value: object) -> float | None:
    """Return only finite non-negative distance values from pgvector."""
    if isinstance(value, bool) or not isinstance(value, (int, float)):
        return None
    distance = float(value)
    return distance if distance >= 0 and math.isfinite(distance) else None
