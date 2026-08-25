from __future__ import annotations

from typing import TYPE_CHECKING, Self
from uuid import UUID

from norvii_agent.config import AgentConfig
from norvii_agent.retrieval import PostgresRetriever

if TYPE_CHECKING:
    import pytest


class FakeEmbeddingProvider:
    def __init__(self) -> None:
        self.texts: tuple[str, ...] | None = None

    def embed(self, texts: tuple[str, ...]) -> tuple[tuple[float, ...], ...]:
        self.texts = texts
        return ((0.1, 0.2),)


class FakeCursor:
    def __init__(self, rows: list[tuple[object, ...]]) -> None:
        self._rows = rows
        self.query = ""
        self.parameters: tuple[object, ...] = ()

    def __enter__(self) -> Self:
        return self

    def __exit__(self, *_args: object) -> None:
        return None

    def execute(self, query: str, parameters: tuple[object, ...]) -> None:
        self.query = query
        self.parameters = parameters

    def fetchall(self) -> list[tuple[object, ...]]:
        return self._rows


class FakeConnection:
    def __init__(self, cursor: FakeCursor) -> None:
        self._cursor = cursor

    def __enter__(self) -> Self:
        return self

    def __exit__(self, *_args: object) -> None:
        return None

    def cursor(self) -> FakeCursor:
        return self._cursor


def configuration() -> AgentConfig:
    return AgentConfig(
        "127.0.0.1",
        8090,
        "localhost",
        5432,
        "norvii",
        "norvii",
        "password",
        "",
        "",
        "",
        "medium",
        30,
    )


def test_search_uses_one_question_vector_and_only_ready_latest_chunks(
    monkeypatch: pytest.MonkeyPatch,
) -> None:
    corpus_id = UUID("10000000-0000-4000-8000-000000000001")
    snapshot_id = UUID("50000000-0000-4000-8000-000000000001")
    source_id = UUID("20000000-0000-4000-8000-000000000001")
    document_id = UUID("30000000-0000-4000-8000-000000000001")
    cursor = FakeCursor(
        [
            (
                "chunk-1",
                corpus_id,
                source_id,
                document_id,
                "Article 1",
                0,
                40,
                "The regulation protects personal data.",
                0.1834,
                UUID("40000000-0000-4000-8000-000000000001"),
                "corpus-ingestion-v1",
                "Official law",
            )
        ]
    )
    monkeypatch.setattr(
        "norvii_agent.retrieval.postgres.psycopg.connect",
        lambda **_kwargs: FakeConnection(cursor),
    )
    embeddings = FakeEmbeddingProvider()

    evidence = PostgresRetriever(configuration(), embeddings).search(
        corpus_id, snapshot_id, "What does the regulation protect?"
    )

    assert embeddings.texts == ("What does the regulation protect?",)
    assert len(evidence) == 1
    assert evidence[0].rank == 1
    assert evidence[0].document_version_id == document_id
    assert evidence[0].source_revision_id == UUID("40000000-0000-4000-8000-000000000001")
    assert evidence[0].pipeline_version == "corpus-ingestion-v1"
    assert evidence[0].source_title == "Official law"
    assert evidence[0].cosine_distance == 0.1834
    assert "c.embedding <=> %s::vector" in cursor.query
    assert "s.title, c.ordinal" in cursor.query
    assert "ORDER BY cosine_distance, ordinal, id" in cursor.query
    assert "c.enrichment_status = 'ready'" in cursor.query
    assert "sd.snapshot_id = %s" in cursor.query
    assert "corpus_snapshot_documents sd" in cursor.query
    assert "to_tsvector" not in cursor.query
    assert cursor.parameters == ("[0.1,0.2]", corpus_id, snapshot_id)
    assert evidence[0].snapshot_id == snapshot_id
