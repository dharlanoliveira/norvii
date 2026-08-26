"""Graph projection connectivity adapter."""

from __future__ import annotations

from dataclasses import dataclass
from typing import TYPE_CHECKING, ClassVar

from neo4j import Driver, EagerResult, GraphDatabase, RoutingControl
from neo4j.exceptions import Neo4jError

from norvii_ingestion.publication.persistence.errors import PersistenceConnectionError

if TYPE_CHECKING:
    from uuid import UUID

    from norvii_ingestion.publication.persistence.config import Neo4jConfiguration


class Neo4jStore:
    """Verify graph projection storage through the production Bolt driver."""

    name: ClassVar[str] = "Neo4j"

    def __init__(self, driver: Driver, database: str) -> None:
        self._driver = driver
        self._database = database

    @classmethod
    def connect(cls, configuration: Neo4jConfiguration, timeout_seconds: int) -> Neo4jStore:
        """Construct an authenticated driver with bounded acquisition and retry behavior."""
        try:
            driver = GraphDatabase.driver(
                configuration.uri,
                auth=(configuration.user, configuration.password),
                connection_timeout=float(timeout_seconds),
                connection_acquisition_timeout=float(timeout_seconds),
                max_transaction_retry_time=0.0,
                max_connection_pool_size=2,
            )
        except (Neo4jError, ValueError) as error:
            raise PersistenceConnectionError("Create Neo4j driver failed.") from error
        return cls(driver, configuration.database)

    def verify(self) -> None:
        """Execute a constant read-only Cypher query over Bolt."""
        try:
            result: EagerResult = self._driver.execute_query(
                "RETURN 1 AS ready",
                routing_=RoutingControl.READ,
                database_=self._database,
            )
        except (Neo4jError, OSError, ValueError) as error:
            raise PersistenceConnectionError("Execute Neo4j readiness query failed.") from error
        if len(result.records) != 1:
            raise PersistenceConnectionError(
                "Neo4j readiness query returned an unexpected record count."
            )

    def replace_release(self, release: GraphReleaseProjection) -> None:
        """Replace one derived graph projection without touching other releases."""
        parameters = {
            "release_id": str(release.release_id),
            "corpus_id": str(release.corpus_id),
            "snapshot_id": str(release.snapshot_id),
            "manifest_sha256": release.manifest_sha256,
            "legal_units": release.legal_units,
            "entities": release.entities,
            "assertions": release.assertions,
        }
        try:
            self._driver.execute_query(
                _REPLACE_RELEASE,
                parameters_=parameters,
                database_=self._database,
                routing_=RoutingControl.WRITE,
            )
        except (Neo4jError, OSError, ValueError) as error:
            raise PersistenceConnectionError("Replace graph release projection failed.") from error

    def close(self) -> None:
        """Release the graph driver and connection pool."""
        self._driver.close()


@dataclass(frozen=True, slots=True)
class GraphReleaseProjection:
    """One fully materialized derived graph release ready for Neo4j replacement."""

    release_id: UUID
    corpus_id: UUID
    snapshot_id: UUID
    manifest_sha256: str
    legal_units: tuple[dict[str, object], ...]
    entities: tuple[dict[str, object], ...]
    assertions: tuple[dict[str, object], ...]


_REPLACE_RELEASE = """
OPTIONAL MATCH (old:NorviiGraphRelease {id: $release_id})
WITH collect(old) AS old_releases, $release_id AS release_id, $corpus_id AS corpus_id,
     $snapshot_id AS snapshot_id, $manifest_sha256 AS manifest_sha256,
     $legal_units AS legal_units, $entities AS entities, $assertions AS assertions
FOREACH (old_release IN old_releases | DETACH DELETE old_release)
WITH release_id, corpus_id, snapshot_id, manifest_sha256, legal_units, entities, assertions
OPTIONAL MATCH (old_node {release_id: release_id})
WHERE old_node:NorviiGraphLegalUnit
   OR old_node:NorviiGraphLegalEntity
   OR old_node:NorviiGraphNormativeAssertion
WITH collect(old_node) AS old_nodes, release_id, corpus_id, snapshot_id, manifest_sha256,
     legal_units, entities, assertions
FOREACH (old_node IN old_nodes | DETACH DELETE old_node)
WITH release_id, corpus_id, snapshot_id, manifest_sha256, legal_units, entities, assertions
MERGE (release:NorviiGraphRelease {id: release_id})
SET release.corpus_id = corpus_id, release.snapshot_id = snapshot_id,
    release.manifest_sha256 = manifest_sha256, release.status = 'ready'
WITH release, legal_units, entities, assertions
UNWIND legal_units AS legal_unit
MERGE (unit:NorviiGraphLegalUnit {release_id: release.id, legal_unit_id: legal_unit.id})
SET unit.document_id = legal_unit.document_id, unit.locator = legal_unit.locator,
    unit.kind = legal_unit.kind
MERGE (unit)-[:IN_GRAPH_RELEASE]->(release)
WITH release, legal_units, entities, assertions
UNWIND legal_units AS child_unit
OPTIONAL MATCH (parent:NorviiGraphLegalUnit {
  release_id: release.id, legal_unit_id: child_unit.parent_id
})
MATCH (child:NorviiGraphLegalUnit {
  release_id: release.id, legal_unit_id: child_unit.id
})
FOREACH (_ IN CASE WHEN child_unit.parent_id IS NULL THEN [] ELSE [1] END |
  MERGE (parent)-[:CONTAINS {release_id: release.id}]->(child)
)
WITH DISTINCT release, entities, assertions
UNWIND entities AS entity
MERGE (entity_node:NorviiGraphLegalEntity {release_id: release.id, semantic_entity_id: entity.id})
SET entity_node.label = entity.label, entity_node.normalized_label = entity.normalized_label,
    entity_node.entity_type = entity.entity_type
MERGE (entity_node)-[:IN_GRAPH_RELEASE]->(release)
WITH release, assertions
UNWIND assertions AS assertion
MATCH (establishing_unit:NorviiGraphLegalUnit {
  release_id: release.id, legal_unit_id: assertion.establishing_unit_id
})
MATCH (subject:NorviiGraphLegalEntity {
  release_id: release.id, semantic_entity_id: assertion.subject_entity_id
})
MATCH (object:NorviiGraphLegalEntity {
  release_id: release.id, semantic_entity_id: assertion.object_entity_id
})
MERGE (assertion_node:NorviiGraphNormativeAssertion {
  release_id: release.id, normative_assertion_id: assertion.id
})
SET assertion_node.predicate = assertion.predicate,
    assertion_node.qualifier = assertion.qualifier,
    assertion_node.evidence_id = assertion.evidence_id,
    assertion_node.source_id = assertion.source_id,
    assertion_node.document_id = assertion.document_id,
    assertion_node.source_revision_id = assertion.source_revision_id,
    assertion_node.pipeline_version = assertion.pipeline_version,
    assertion_node.source_title = assertion.source_title,
    assertion_node.establishing_locator = assertion.establishing_locator,
    assertion_node.evidence_locator = assertion.evidence_locator,
    assertion_node.start_offset = assertion.start_offset,
    assertion_node.end_offset = assertion.end_offset,
    assertion_node.excerpt = assertion.excerpt
MERGE (assertion_node)-[:IN_GRAPH_RELEASE]->(release)
MERGE (establishing_unit)-[:ESTABLISHES {release_id: release.id}]->(assertion_node)
MERGE (assertion_node)-[:SUBJECT {release_id: release.id}]->(subject)
MERGE (assertion_node)-[:OBJECT {release_id: release.id}]->(object)
"""
