"""Service-backed isolation coverage for Vector and planned Hybrid retrieval."""

from __future__ import annotations

from collections.abc import Iterator
from dataclasses import dataclass
from datetime import UTC, datetime
from uuid import UUID, uuid4

import psycopg
import pytest
from neo4j import Driver, GraphDatabase, RoutingControl

from norvii_agent.config import AgentConfig
from norvii_agent.retrieval import HybridRetriever, Neo4jGraphRetriever, PostgresRetriever
from norvii_agent.retrieval.planning import (
    GraphCapabilityCatalog,
    GraphRetrievalPlan,
)

pytestmark = pytest.mark.integration


class FixedEmbeddingProvider:
    """Return a deterministic vector compatible with the integration schema."""

    def embed(self, texts: tuple[str, ...]) -> tuple[tuple[float, ...], ...]:
        assert texts == ("Who governs the authority?",)
        return (tuple(0.0 for _ in range(1536)),)


class SupportingGraphPlanner:
    """Select the seeded relationship without invoking a configured model provider."""

    def plan(self, _question: str, catalog: GraphCapabilityCatalog) -> GraphRetrievalPlan:
        assert set(catalog.relationship_types) == {"governs"}
        assert set(catalog.entity_labels) == {"authority", "obligation"}
        return GraphRetrievalPlan(
            use_graph=True,
            relationship_types=("governs",),
            entity_labels=("authority",),
        )


@pytest.fixture
def configuration() -> AgentConfig:
    """Load the local service endpoints only for explicitly selected integration tests."""
    return AgentConfig.from_environment()


@pytest.fixture
def isolation_fixture(configuration: AgentConfig) -> Iterator[RetrievalIsolationFixture]:
    """Seed independent PostgreSQL snapshots and matching Neo4j graph releases."""
    fixture = RetrievalIsolationFixture()
    postgres = psycopg.connect(
        host=configuration.postgres_host,
        port=configuration.postgres_port,
        dbname=configuration.postgres_database,
        user=configuration.postgres_user,
        password=configuration.postgres_password,
        connect_timeout=5,
    )
    graph = GraphDatabase.driver(
        configuration.neo4j_uri,
        auth=(configuration.neo4j_user, configuration.neo4j_password),
        connection_timeout=5.0,
        connection_acquisition_timeout=5.0,
        max_transaction_retry_time=0.0,
        max_connection_pool_size=2,
    )
    try:
        fixture.seed_postgres(postgres)
        postgres.commit()
        fixture.seed_neo4j(graph, configuration.neo4j_database)
        yield fixture
    finally:
        fixture.delete_neo4j(graph, configuration.neo4j_database)
        fixture.delete_postgres(postgres)
        graph.close()
        postgres.close()


def test_vector_retrieval_excludes_foreign_corpus_and_snapshot_documents(
    configuration: AgentConfig, isolation_fixture: RetrievalIsolationFixture
) -> None:
    """Vector retrieval reads only the document membership of the declared snapshot."""
    evidence = PostgresRetriever(configuration, FixedEmbeddingProvider()).search(
        isolation_fixture.target.corpus_id,
        isolation_fixture.target.snapshot_id,
        "Who governs the authority?",
    )

    assert [item.document_id for item in evidence] == [isolation_fixture.target.document_id]
    assert {item.corpus_id for item in evidence} == {isolation_fixture.target.corpus_id}
    assert {item.snapshot_id for item in evidence} == {isolation_fixture.target.snapshot_id}
    assert isolation_fixture.foreign_snapshot.document_id not in {
        item.document_id for item in evidence
    }
    assert isolation_fixture.foreign_corpus.document_id not in {
        item.document_id for item in evidence
    }


def test_hybrid_retrieval_excludes_foreign_postgres_and_neo4j_evidence(
    configuration: AgentConfig, isolation_fixture: RetrievalIsolationFixture
) -> None:
    """Hybrid composition keeps both service-backed contributions inside one snapshot boundary."""
    hybrid = HybridRetriever(
        PostgresRetriever(configuration, FixedEmbeddingProvider()),
        Neo4jGraphRetriever(configuration),
        SupportingGraphPlanner(),
    )

    evidence = hybrid.search(
        isolation_fixture.target.corpus_id,
        isolation_fixture.target.snapshot_id,
        "Who governs the authority?",
    )

    assert [item.document_id for item in evidence] == [isolation_fixture.target.document_id]
    assert [item.contribution for item in evidence] == ["vector_and_graph"]
    assert {item.corpus_id for item in evidence} == {isolation_fixture.target.corpus_id}
    assert {item.snapshot_id for item in evidence} == {isolation_fixture.target.snapshot_id}
    assert [(stage.name, stage.state, stage.evidence_count) for stage in hybrid.last_stages] == [
        ("vector", "completed", 1),
        ("planning", "completed", 0),
        ("graph", "completed", 1),
    ]
    returned_documents = {item.document_id for item in evidence}
    assert isolation_fixture.foreign_snapshot.document_id not in returned_documents
    assert isolation_fixture.foreign_corpus.document_id not in returned_documents
    assert [path.evidence_id for path in hybrid.last_graph_path] == [
        str(isolation_fixture.target.graph_relationship_id)
    ]


@dataclass(frozen=True, slots=True)
class SnapshotDocument:
    """One document intentionally placed in exactly one immutable snapshot."""

    corpus_id: UUID
    snapshot_id: UUID
    source_id: UUID
    source_revision_id: UUID
    document_id: UUID
    unit_id: UUID
    chunk_id: UUID
    graph_release_id: UUID
    graph_relationship_id: UUID
    label: str


class RetrievalIsolationFixture:
    """Own the cross-service records used to prove corpus and snapshot isolation."""

    def __init__(self) -> None:
        target_corpus_id = uuid4()
        foreign_corpus_id = uuid4()
        self.target = self._document(target_corpus_id, "target")
        self.foreign_snapshot = self._document(target_corpus_id, "foreign-snapshot")
        self.foreign_corpus = self._document(foreign_corpus_id, "foreign-corpus")
        self._corpus_ids = (target_corpus_id, foreign_corpus_id)

    def seed_postgres(self, connection: psycopg.Connection[tuple[object, ...]]) -> None:
        """Create target and excluded vector evidence in their explicit snapshot memberships."""
        captured_at = datetime(2026, 8, 25, 12, 0, tzinfo=UTC)
        with connection.cursor() as cursor:
            for corpus_id, language in (
                (self.target.corpus_id, "en"),
                (self.foreign_corpus.corpus_id, "pt"),
            ):
                cursor.execute(
                    """
                    INSERT INTO corpora (id, name, description, language, jurisdiction)
                    VALUES (%s, 'Graph retrieval isolation fixture',
                            'Service-backed retrieval isolation fixture.', %s, 'Test')
                    """,
                    (corpus_id, language),
                )
            for document in self._documents:
                self._insert_document(cursor, document, captured_at)
                self._insert_snapshot(cursor, document, captured_at)

    def seed_neo4j(self, driver: Driver, database: str) -> None:
        """Create a ready graph release for every target and deliberately excluded snapshot."""
        for document in self._documents:
            driver.execute_query(
                _INSERT_GRAPH_RELEASE,
                release_id=str(document.graph_release_id),
                corpus_id=str(document.corpus_id),
                snapshot_id=str(document.snapshot_id),
                relationship_id=str(document.graph_relationship_id),
                source_id=str(document.source_id),
                document_id=str(document.document_id),
                source_revision_id=str(document.source_revision_id),
                evidence_locator="article-1",
                excerpt=f"{document.label} authority governs obligations.",
                database_=database,
                routing_=RoutingControl.WRITE,
            )

    def delete_neo4j(self, driver: Driver, database: str) -> None:
        """Delete only the graph releases and nodes created by this fixture."""
        driver.execute_query(
            _DELETE_GRAPH_RELEASES,
            release_ids=[str(document.graph_release_id) for document in self._documents],
            database_=database,
            routing_=RoutingControl.WRITE,
        )

    def delete_postgres(self, connection: psycopg.Connection[tuple[object, ...]]) -> None:
        """Remove only the relational records created by this fixture."""
        connection.rollback()
        with connection.transaction(), connection.cursor() as cursor:
            for corpus_id in self._corpus_ids:
                cursor.execute(
                    "DELETE FROM corpus_snapshot_documents WHERE corpus_id = %s", (corpus_id,)
                )
                cursor.execute("DELETE FROM corpus_snapshots WHERE corpus_id = %s", (corpus_id,))
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
                    "UPDATE sources SET latest_ready_document_id = NULL WHERE corpus_id = %s",
                    (corpus_id,),
                )
                cursor.execute("DELETE FROM document_versions WHERE corpus_id = %s", (corpus_id,))
                cursor.execute("DELETE FROM source_revisions WHERE corpus_id = %s", (corpus_id,))
                cursor.execute("DELETE FROM processing_attempts WHERE corpus_id = %s", (corpus_id,))
                cursor.execute("DELETE FROM ingestion_work WHERE corpus_id = %s", (corpus_id,))
                cursor.execute("DELETE FROM url_origins WHERE corpus_id = %s", (corpus_id,))
                cursor.execute("DELETE FROM sources WHERE corpus_id = %s", (corpus_id,))
                cursor.execute("DELETE FROM corpora WHERE id = %s", (corpus_id,))

    @property
    def _documents(self) -> tuple[SnapshotDocument, ...]:
        return (self.target, self.foreign_snapshot, self.foreign_corpus)

    @staticmethod
    def _document(corpus_id: UUID, label: str) -> SnapshotDocument:
        return SnapshotDocument(
            corpus_id=corpus_id,
            snapshot_id=uuid4(),
            source_id=uuid4(),
            source_revision_id=uuid4(),
            document_id=uuid4(),
            unit_id=uuid4(),
            chunk_id=uuid4(),
            graph_release_id=uuid4(),
            graph_relationship_id=uuid4(),
            label=label,
        )

    @staticmethod
    def _insert_document(
        cursor: psycopg.Cursor[tuple[object, ...]],
        document: SnapshotDocument,
        captured_at: datetime,
    ) -> None:
        work_id = uuid4()
        attempt_id = uuid4()
        text = f"{document.label} authority governs obligations."
        content_hash = uuid4().hex * 2
        text_hash = uuid4().hex * 2
        cursor.execute(
            """
            INSERT INTO sources (id, corpus_id, title, kind, processing_status)
            VALUES (%s, %s, %s, 'url', 'ready')
            """,
            (document.source_id, document.corpus_id, f"{document.label} source"),
        )
        cursor.execute(
            """
            INSERT INTO url_origins (source_id, corpus_id, submitted_url, normalized_url)
            VALUES (%s, %s, %s, %s)
            """,
            (
                document.source_id,
                document.corpus_id,
                f"https://example.org/{document.source_id}",
                f"https://example.org/{document.source_id}",
            ),
        )
        cursor.execute(
            """
            INSERT INTO ingestion_work (id, source_id, corpus_id, reason, status)
            VALUES (%s, %s, %s, 'reprocess', 'succeeded')
            """,
            (work_id, document.source_id, document.corpus_id),
        )
        cursor.execute(
            """
            INSERT INTO processing_attempts (
                id, work_id, source_id, corpus_id, attempt_number, pipeline_version, status,
                lease_token, worker_id, started_at, finished_at
            ) VALUES (%s, %s, %s, %s, 1, 'graph-retrieval-test', 'succeeded', %s,
                      'integration-test', %s, %s)
            """,
            (
                attempt_id,
                work_id,
                document.source_id,
                document.corpus_id,
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
            ) VALUES (%s, %s, %s, %s, %s, %s, 'text/html', 120, 'graph-retrieval-test', %s, %s)
            """,
            (
                document.source_revision_id,
                document.source_id,
                document.corpus_id,
                attempt_id,
                content_hash,
                captured_at,
                f"https://example.org/{document.source_id}",
                content_hash,
            ),
        )
        cursor.execute(
            """
            INSERT INTO document_versions (
                id, source_revision_id, source_id, corpus_id, pipeline_version, text_content,
                text_sha256, published_at
            ) VALUES (%s, %s, %s, %s, 'graph-retrieval-test', %s, %s, %s)
            """,
            (
                document.document_id,
                document.source_revision_id,
                document.source_id,
                document.corpus_id,
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
            ) VALUES (%s, %s, 'article', 1, 0, 40, 'article-1', %s)
            """,
            (document.unit_id, document.document_id, text_hash),
        )
        cursor.execute(
            """
            INSERT INTO retrieval_chunks (
                id, corpus_id, source_id, document_id, unit_id, ordinal, start_offset,
                end_offset, content, content_sha256, context_locator, embedding,
                embedding_model, enrichment_status
            ) VALUES (%s, %s, %s, %s, %s, 1, 0, 40, %s, %s, 'article-1', %s::vector,
                      'graph-retrieval-test', 'ready')
            """,
            (
                document.chunk_id,
                document.corpus_id,
                document.source_id,
                document.document_id,
                document.unit_id,
                text,
                text_hash,
                _zero_vector(),
            ),
        )

    @staticmethod
    def _insert_snapshot(
        cursor: psycopg.Cursor[tuple[object, ...]],
        document: SnapshotDocument,
        captured_at: datetime,
    ) -> None:
        cursor.execute(
            """
            INSERT INTO corpus_snapshots (id, corpus_id, manifest_sha256, created_by, created_at)
            VALUES (%s, %s, %s, 'integration-test', %s)
            """,
            (document.snapshot_id, document.corpus_id, uuid4().hex * 2, captured_at),
        )
        cursor.execute(
            """
            INSERT INTO corpus_snapshot_documents (
                snapshot_id, corpus_id, source_id, source_revision_id, document_id,
                official_origin, captured_at, content_sha256
            ) VALUES (%s, %s, %s, %s, %s, 'https://example.org/graph-retrieval', %s, %s)
            """,
            (
                document.snapshot_id,
                document.corpus_id,
                document.source_id,
                document.source_revision_id,
                document.document_id,
                captured_at,
                uuid4().hex * 2,
            ),
        )


_INSERT_GRAPH_RELEASE = """
CREATE (release:NorviiGraphRelease {
  id: $release_id, corpus_id: $corpus_id, snapshot_id: $snapshot_id, status: 'ready'
})
CREATE (subject:NorviiGraphEntity {
  release_id: $release_id, semantic_entity_id: $release_id + '-authority', label: 'Authority',
  normalized_label: 'authority', entity_type: 'actor'
})-[:IN_GRAPH_RELEASE]->(release)
CREATE (object:NorviiGraphEntity {
  release_id: $release_id, semantic_entity_id: $release_id + '-obligation', label: 'Obligation',
  normalized_label: 'obligation', entity_type: 'obligation'
})-[:IN_GRAPH_RELEASE]->(release)
CREATE (subject)-[:LEGAL_RELATIONSHIP {
  release_id: $release_id, id: $relationship_id, evidence_id: $relationship_id,
  source_id: $source_id, document_id: $document_id, source_revision_id: $source_revision_id,
  pipeline_version: 'graph-retrieval-test', source_title: 'Graph retrieval source',
  evidence_locator: $evidence_locator, start_offset: 0, end_offset: 40, excerpt: $excerpt,
  relationship_type: 'governs'
}]->(object)
"""

_DELETE_GRAPH_RELEASES = """
MATCH (release:NorviiGraphRelease)
WHERE release.id IN $release_ids
OPTIONAL MATCH (entity:NorviiGraphEntity)-[:IN_GRAPH_RELEASE]->(release)
WITH collect(DISTINCT entity) + collect(DISTINCT release) AS nodes
UNWIND nodes AS node
DETACH DELETE node
"""


def _zero_vector() -> str:
    return "[" + ",".join("0" for _ in range(1536)) + "]"
