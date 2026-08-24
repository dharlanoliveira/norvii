"""Neo4j retrieval of snapshot-scoped, evidence-backed legal relationships."""

from __future__ import annotations

from dataclasses import dataclass
from typing import TYPE_CHECKING
from uuid import UUID

from neo4j import Driver, GraphDatabase, RoutingControl
from neo4j.exceptions import Neo4jError

from norvii_agent.graph import (
    Evidence,
    GraphPathStep,
    RetrievalInspection,
    StrategyUnavailableError,
)

if TYPE_CHECKING:
    from norvii_agent.config import AgentConfig


class GraphRetrievalUnavailableError(StrategyUnavailableError):
    """Signal that graph retrieval cannot safely complete for this snapshot."""


@dataclass(slots=True)
class Neo4jGraphRetriever:
    """Retrieve only ready graph relationships belonging to one immutable snapshot."""

    configuration: AgentConfig
    driver: Driver | None = None
    last_retrieval: RetrievalInspection | None = None
    last_graph_path: tuple[GraphPathStep, ...] = ()

    def search(
        self, corpus_id: UUID, snapshot_id: UUID, question: str, strategy: str = "graph"
    ) -> tuple[Evidence, ...]:
        """Find bounded graph-supported locations without a vector fallback."""
        if strategy != "graph":
            raise ValueError("Neo4jGraphRetriever only supports graph retrieval")
        tokens = _query_tokens(question)
        if not tokens:
            self.last_retrieval = RetrievalInspection("graph", 8, 0, None)
            self.last_graph_path = ()
            return ()
        driver = self._driver()
        try:
            result = driver.execute_query(
                _GRAPH_SEARCH,
                corpus_id=str(corpus_id),
                snapshot_id=str(snapshot_id),
                tokens=tokens,
                database_=self.configuration.neo4j_database,
                routing_=RoutingControl.READ,
            )
        except (Neo4jError, OSError, ValueError) as error:
            raise GraphRetrievalUnavailableError("graph projection query failed") from error
        records = result.records
        evidence = tuple(
            _evidence(record.data(), index + 1, corpus_id, snapshot_id)
            for index, record in enumerate(records)
        )
        self.last_retrieval = RetrievalInspection("graph", 8, len(evidence), None)
        self.last_graph_path = tuple(_path_step(record.data()) for record in records)
        return evidence

    def close(self) -> None:
        """Release the optional driver owned by this adapter."""
        if self.driver is not None:
            self.driver.close()
            self.driver = None

    def _driver(self) -> Driver:
        if self.driver is not None:
            return self.driver
        if not all(
            (
                self.configuration.neo4j_uri,
                self.configuration.neo4j_user,
                self.configuration.neo4j_password,
            )
        ):
            raise GraphRetrievalUnavailableError("graph projection is not configured")
        try:
            self.driver = GraphDatabase.driver(
                self.configuration.neo4j_uri,
                auth=(self.configuration.neo4j_user, self.configuration.neo4j_password),
                connection_timeout=5.0,
                connection_acquisition_timeout=5.0,
                max_transaction_retry_time=0.0,
                max_connection_pool_size=2,
            )
        except (Neo4jError, ValueError) as error:
            raise GraphRetrievalUnavailableError("graph projection connection failed") from error
        return self.driver


_GRAPH_SEARCH = """
MATCH (release:NorviiGraphRelease {
  corpus_id: $corpus_id, snapshot_id: $snapshot_id, status: 'ready'
})
MATCH (release)<-[:IN_GRAPH_RELEASE]-(subject:NorviiGraphEntity)
MATCH (subject)-[relationship:LEGAL_RELATIONSHIP {release_id: release.id}]->
      (object:NorviiGraphEntity)
WHERE any(token IN $tokens WHERE
  subject.normalized_label CONTAINS token OR object.normalized_label CONTAINS token
)
RETURN relationship.evidence_id AS evidence_id,
       relationship.source_id AS source_id,
       relationship.document_id AS document_id,
       relationship.source_revision_id AS source_revision_id,
       relationship.pipeline_version AS pipeline_version,
       relationship.source_title AS source_title,
       relationship.evidence_locator AS evidence_locator,
       relationship.start_offset AS start_offset,
       relationship.end_offset AS end_offset,
       relationship.excerpt AS excerpt,
       relationship.relationship_type AS relationship_type,
       subject.label AS subject_label,
       object.label AS object_label
ORDER BY relationship.evidence_locator, relationship.evidence_id
LIMIT 8
"""


_MIN_QUERY_TOKEN_LENGTH = 3
_MAX_QUERY_TOKEN_COUNT = 12


def _query_tokens(question: str) -> tuple[str, ...]:
    """Keep graph matching bounded and deterministic for POC questions."""
    return tuple(
        token for token in question.lower().split() if len(token) >= _MIN_QUERY_TOKEN_LENGTH
    )[:_MAX_QUERY_TOKEN_COUNT]


def _evidence(row: dict[str, object], rank: int, corpus_id: UUID, snapshot_id: UUID) -> Evidence:
    return Evidence(
        id=str(row["evidence_id"]),
        corpus_id=corpus_id,
        source_id=UUID(str(row["source_id"])),
        document_id=UUID(str(row["document_id"])),
        unit_locator=str(row["evidence_locator"]),
        start_offset=_integer_value(row["start_offset"], "start offset"),
        end_offset=_integer_value(row["end_offset"], "end offset"),
        excerpt=str(row["excerpt"]),
        rank=rank,
        document_version_id=UUID(str(row["document_id"])),
        source_revision_id=UUID(str(row["source_revision_id"])),
        pipeline_version=str(row["pipeline_version"]),
        source_title=str(row["source_title"]),
        snapshot_id=snapshot_id,
    )


def _path_step(row: dict[str, object]) -> GraphPathStep:
    return GraphPathStep(
        relationship_type=str(row["relationship_type"]),
        subject_label=str(row["subject_label"]),
        object_label=str(row["object_label"]),
        evidence_id=str(row["evidence_id"]),
        evidence_locator=str(row["evidence_locator"]),
    )


def _integer_value(value: object, field: str) -> int:
    """Validate numeric graph properties received from the driver boundary."""
    if isinstance(value, bool) or not isinstance(value, int):
        raise GraphRetrievalUnavailableError(f"graph {field} is invalid")
    return value
