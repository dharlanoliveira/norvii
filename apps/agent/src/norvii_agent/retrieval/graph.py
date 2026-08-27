"""Neo4j retrieval of snapshot-scoped, evidence-backed normative assertions."""

from __future__ import annotations

import json
import logging
from dataclasses import dataclass
from typing import TYPE_CHECKING, Any
from uuid import UUID

from neo4j import Driver, GraphDatabase, RoutingControl
from neo4j.exceptions import Neo4jError

from norvii_agent.graph import (
    AssertionPathStep,
    Evidence,
    RetrievalInspection,
    StrategyUnavailableError,
)
from norvii_agent.retrieval.planning import (
    NORMATIVE_PREDICATES,
    GraphCapabilityCatalog,
    GraphPredicateCapability,
    GraphRetrievalPlan,
)

if TYPE_CHECKING:
    from norvii_agent.config import AgentConfig

_LOGGER = logging.getLogger(__name__)


class GraphRetrievalUnavailableError(StrategyUnavailableError):
    """Signal that graph retrieval cannot safely complete for this snapshot."""


@dataclass(slots=True)
class Neo4jGraphRetriever:
    """Retrieve only ready graph assertions belonging to one immutable snapshot."""

    configuration: AgentConfig
    driver: Driver | None = None
    last_retrieval: RetrievalInspection | None = None
    last_assertion_path: tuple[AssertionPathStep, ...] = ()
    last_scope_locator: str | None = None

    def search(
        self, corpus_id: UUID, snapshot_id: UUID, question: str, strategy: str = "graph"
    ) -> tuple[Evidence, ...]:
        """Find bounded graph-supported locations without a vector fallback."""
        if strategy != "graph":
            raise ValueError("Neo4jGraphRetriever only supports graph retrieval")
        tokens = _query_tokens(question)
        if not tokens:
            self.last_retrieval = RetrievalInspection("graph", 8, 0, None)
            self.last_assertion_path = ()
            self.last_scope_locator = None
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
        self.last_assertion_path = tuple(_assertion_path(record.data()) for record in records)
        self.last_scope_locator = None
        return evidence

    def capabilities(self, corpus_id: UUID, snapshot_id: UUID) -> GraphCapabilityCatalog | None:
        """Return only the ready snapshot schema that can safely guide planning."""
        driver = self._driver()
        try:
            result = driver.execute_query(
                _GRAPH_CAPABILITIES,
                corpus_id=str(corpus_id),
                snapshot_id=str(snapshot_id),
                database_=self.configuration.neo4j_database,
                routing_=RoutingControl.READ,
            )
        except (Neo4jError, OSError, ValueError) as error:
            raise GraphRetrievalUnavailableError("graph capability query failed") from error
        if not result.records:
            return None
        row = result.records[0].data()
        predicates = tuple(
            predicate
            for predicate in _string_values(row.get("predicates"))
            if predicate in NORMATIVE_PREDICATES
        )
        entity_types = _string_values(row.get("entity_types"))
        entity_labels = _string_values(row.get("entity_labels"), limit=128)
        predicate_capabilities = _predicate_capabilities(row.get("predicate_capabilities"))
        scope_locators = _string_values(row.get("scope_locators"), limit=128)
        return (
            GraphCapabilityCatalog(
                entity_types,
                predicates,
                entity_labels,
                predicate_capabilities,
                scope_locators,
            )
            if predicates and entity_labels and predicate_capabilities
            else None
        )

    def search_plan(
        self,
        corpus_id: UUID,
        snapshot_id: UUID,
        plan: GraphRetrievalPlan,
    ) -> tuple[Evidence, ...]:
        """Run a constrained normative-assertion lookup selected by the planner."""
        if not plan.use_graph:
            return ()
        parameters: dict[str, Any] = {
            "corpus_id": str(corpus_id),
            "snapshot_id": str(snapshot_id),
            "predicates": list(plan.predicates),
            "entity_labels": list(plan.entity_labels),
            "scope_locator": plan.scope_locator,
        }
        _LOGGER.info(
            "Planned graph Cypher executed:\n%s\nparameters=%s",
            _PLANNED_GRAPH_SEARCH.strip(),
            json.dumps(parameters, sort_keys=True),
        )
        driver = self._driver()
        try:
            result = driver.execute_query(
                _PLANNED_GRAPH_SEARCH,
                parameters_=parameters,
                database_=self.configuration.neo4j_database,
                routing_=RoutingControl.READ,
            )
        except (Neo4jError, OSError, ValueError) as error:
            raise GraphRetrievalUnavailableError("planned graph projection query failed") from error
        records = result.records
        _LOGGER.info("Planned assertion Cypher returned %d evidence locations", len(records))
        evidence = tuple(
            _evidence(record.data(), index + 1, corpus_id, snapshot_id)
            for index, record in enumerate(records)
        )
        self.last_retrieval = RetrievalInspection("graph", 8, len(evidence), None)
        self.last_assertion_path = tuple(_assertion_path(record.data()) for record in records)
        self.last_scope_locator = plan.scope_locator
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


_READY_PROVENANCE_COMPLETE_RELEASE = """
CALL () {
  MATCH (candidate:NorviiGraphRelease {
    corpus_id: $corpus_id,
    snapshot_id: $snapshot_id,
    status: 'ready',
    build_version: 'legal-assertion-graph-v2'
  })
  WHERE NOT EXISTS {
    MATCH (candidate)<-[:IN_GRAPH_RELEASE]-(candidate_assertion:NorviiGraphNormativeAssertion)
    WHERE candidate_assertion.evidence_unit_id IS NULL
       OR candidate_assertion.evidence_canonical_locator IS NULL
       OR candidate_assertion.evidence_content_sha256 IS NULL
  }
  WITH candidate
  ORDER BY candidate.id
  LIMIT 1
  RETURN candidate AS release
}
"""

_GRAPH_SEARCH = (
    _READY_PROVENANCE_COMPLETE_RELEASE
    + """
MATCH (release)<-[:IN_GRAPH_RELEASE]-(assertion:NorviiGraphNormativeAssertion)
MATCH (release)<-[:IN_GRAPH_RELEASE]-(subject:NorviiGraphLegalEntity)<-[:SUBJECT]-(assertion)
MATCH (release)<-[:IN_GRAPH_RELEASE]-(object:NorviiGraphLegalEntity)<-[:OBJECT]-(assertion)
MATCH (release)<-[:IN_GRAPH_RELEASE]-(establishing:NorviiGraphLegalUnit)
      -[:ESTABLISHES]->(assertion)
WHERE any(token IN $tokens WHERE
  subject.normalized_label CONTAINS token OR object.normalized_label CONTAINS token
)
RETURN assertion.normative_assertion_id AS assertion_id,
       assertion.evidence_id AS evidence_id,
       assertion.source_id AS source_id,
       assertion.document_id AS document_id,
       assertion.source_revision_id AS source_revision_id,
       assertion.pipeline_version AS pipeline_version,
       assertion.source_title AS source_title,
       assertion.evidence_locator AS evidence_locator,
       assertion.evidence_canonical_locator AS evidence_canonical_locator,
       assertion.evidence_content_sha256 AS evidence_content_sha256,
       assertion.evidence_unit_id AS evidence_unit_id,
       assertion.start_offset AS start_offset,
       assertion.end_offset AS end_offset,
       assertion.excerpt AS excerpt,
       assertion.predicate AS predicate,
       assertion.qualifier AS qualifier,
       subject.label AS subject_label,
       object.label AS object_label,
       establishing.locator AS establishing_locator,
       [establishing.locator] AS hierarchy_context
ORDER BY assertion.evidence_locator, assertion.normative_assertion_id
LIMIT 8
"""
)

_GRAPH_CAPABILITIES = (
    _READY_PROVENANCE_COMPLETE_RELEASE
    + """
CALL (release) {
  OPTIONAL MATCH (release)<-[:IN_GRAPH_RELEASE]-(entity:NorviiGraphLegalEntity)
  RETURN collect(DISTINCT entity.entity_type)[..32] AS entity_types,
         collect(DISTINCT entity.normalized_label)[..128] AS entity_labels
}
CALL (release) {
  OPTIONAL MATCH (release)<-[:IN_GRAPH_RELEASE]-(unit:NorviiGraphLegalUnit)
  RETURN collect(DISTINCT unit.locator)[..128] AS scope_locators
}
CALL (release) {
  OPTIONAL MATCH (release)<-[:IN_GRAPH_RELEASE]-(assertion:NorviiGraphNormativeAssertion)
  OPTIONAL MATCH (assertion)-[:SUBJECT]->(subject:NorviiGraphLegalEntity)
  OPTIONAL MATCH (assertion)-[:OBJECT]->(object:NorviiGraphLegalEntity)
  WITH assertion.predicate AS predicate,
       collect(DISTINCT subject.normalized_label) + collect(DISTINCT object.normalized_label)
         AS connected_labels
  WHERE predicate IS NOT NULL
  RETURN collect({
    predicate: predicate,
    entity_labels: connected_labels[..128]
  })[..32] AS predicate_capabilities
}
RETURN entity_types,
       [capability IN predicate_capabilities | capability.predicate] AS predicates,
       entity_labels,
       predicate_capabilities,
       scope_locators
"""
)

_PLANNED_GRAPH_SEARCH = (
    _READY_PROVENANCE_COMPLETE_RELEASE
    + """
OPTIONAL MATCH (release)<-[:IN_GRAPH_RELEASE]-(scope:NorviiGraphLegalUnit {
  locator: $scope_locator
})
WITH release, head(collect(scope)) AS scope
OPTIONAL MATCH hierarchy_path=(scope)-[:CONTAINS*0..6]->(scoped_unit:NorviiGraphLegalUnit)
WITH release, scope, collect(scoped_unit) AS scoped_units,
     collect([node IN nodes(hierarchy_path) | node.locator]) AS hierarchy_paths
MATCH (release)<-[:IN_GRAPH_RELEASE]-(assertion:NorviiGraphNormativeAssertion)
MATCH (release)<-[:IN_GRAPH_RELEASE]-(subject:NorviiGraphLegalEntity)<-[:SUBJECT]-(assertion)
MATCH (release)<-[:IN_GRAPH_RELEASE]-(object:NorviiGraphLegalEntity)<-[:OBJECT]-(assertion)
MATCH (release)<-[:IN_GRAPH_RELEASE]-(establishing:NorviiGraphLegalUnit)
      -[:ESTABLISHES]->(assertion)
WITH assertion, subject, object, establishing, scope, scoped_units, hierarchy_paths
WHERE assertion.predicate IN $predicates
  AND (subject.normalized_label IN $entity_labels OR object.normalized_label IN $entity_labels)
  AND ($scope_locator IS NULL OR establishing IN scoped_units)
RETURN assertion.normative_assertion_id AS assertion_id,
       assertion.evidence_id AS evidence_id,
       assertion.source_id AS source_id,
       assertion.document_id AS document_id,
       assertion.source_revision_id AS source_revision_id,
       assertion.pipeline_version AS pipeline_version,
       assertion.source_title AS source_title,
       assertion.evidence_locator AS evidence_locator,
       assertion.evidence_canonical_locator AS evidence_canonical_locator,
       assertion.evidence_content_sha256 AS evidence_content_sha256,
       assertion.evidence_unit_id AS evidence_unit_id,
       assertion.start_offset AS start_offset,
       assertion.end_offset AS end_offset,
       assertion.excerpt AS excerpt,
       assertion.predicate AS predicate,
       assertion.qualifier AS qualifier,
       subject.label AS subject_label,
       object.label AS object_label,
       establishing.locator AS establishing_locator,
       CASE WHEN scope IS NULL THEN [establishing.locator]
            ELSE head([path IN hierarchy_paths WHERE last(path) = establishing.locator])
       END AS hierarchy_context
ORDER BY assertion.evidence_locator, assertion.normative_assertion_id
LIMIT 8
"""
)


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
        unit_id=UUID(str(row["evidence_unit_id"])),
        canonical_locator=_required_string(
            row.get("evidence_canonical_locator"), "evidence canonical locator"
        ),
        content_sha256=_required_string(
            row.get("evidence_content_sha256"), "evidence content hash"
        ),
    )


def _assertion_path(row: dict[str, object]) -> AssertionPathStep:
    """Decode the bounded provenance values returned by the assertion query."""
    hierarchy_context = _string_values(row.get("hierarchy_context"), limit=7)
    if not hierarchy_context:
        raise GraphRetrievalUnavailableError("graph assertion hierarchy context is invalid")
    establishing_locator = _required_string(
        row.get("establishing_locator"), "assertion establishing locator"
    )
    if hierarchy_context[-1] != establishing_locator:
        raise GraphRetrievalUnavailableError("graph assertion hierarchy context is unresolved")
    return AssertionPathStep(
        assertion_id=_required_string(row.get("assertion_id"), "assertion identifier"),
        predicate=_required_string(row.get("predicate"), "assertion predicate"),
        subject_label=_required_string(row.get("subject_label"), "assertion subject label"),
        object_label=_required_string(row.get("object_label"), "assertion object label"),
        establishing_locator=establishing_locator,
        evidence_locator=_required_string(
            row.get("evidence_locator"), "assertion evidence locator"
        ),
        hierarchy_context=hierarchy_context,
        qualifier=_optional_string(row.get("qualifier")),
    )


def _integer_value(value: object, field: str) -> int:
    """Validate numeric graph properties received from the driver boundary."""
    if isinstance(value, bool) or not isinstance(value, int):
        raise GraphRetrievalUnavailableError(f"graph {field} is invalid")
    return value


def _string_values(value: object, *, limit: int = 32) -> tuple[str, ...]:
    """Return bounded graph-schema labels without accepting malformed driver values."""
    if not isinstance(value, list):
        return ()
    return tuple(item for item in value if isinstance(item, str) and item)[:limit]


def _predicate_capabilities(value: object) -> tuple[GraphPredicateCapability, ...]:
    """Read only bounded predicate-to-entity combinations from Neo4j."""
    if not isinstance(value, list):
        return ()
    capabilities: list[GraphPredicateCapability] = []
    for item in value:
        if not isinstance(item, dict):
            continue
        predicate = item.get("predicate")
        entity_labels = _string_values(item.get("entity_labels"), limit=128)
        if (
            not isinstance(predicate, str)
            or predicate not in NORMATIVE_PREDICATES
            or not entity_labels
        ):
            continue
        capabilities.append(GraphPredicateCapability(predicate, entity_labels))
    return tuple(capabilities)


def _optional_string(value: object) -> str | None:
    """Return an optional non-empty qualifier from the driver boundary."""
    return value if isinstance(value, str) and value else None


def _required_string(value: object, field: str) -> str:
    """Reject malformed mandatory assertion provenance at the driver boundary."""
    if not isinstance(value, str) or not value:
        raise GraphRetrievalUnavailableError(f"graph {field} is invalid")
    return value
