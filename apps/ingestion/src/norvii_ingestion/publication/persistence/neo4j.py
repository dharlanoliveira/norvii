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
            "entities": release.entities,
            "relationships": release.relationships,
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
    entities: tuple[dict[str, object], ...]
    relationships: tuple[dict[str, object], ...]


_REPLACE_RELEASE = """
OPTIONAL MATCH (old:NorviiGraphRelease {id: $release_id})
WITH collect(old) AS old_releases, $release_id AS release_id, $corpus_id AS corpus_id,
     $snapshot_id AS snapshot_id, $manifest_sha256 AS manifest_sha256,
     $entities AS entities, $relationships AS relationships
FOREACH (old_release IN old_releases | DETACH DELETE old_release)
WITH release_id, corpus_id, snapshot_id, manifest_sha256, entities, relationships
OPTIONAL MATCH (old_entity:NorviiGraphEntity {release_id: release_id})
WITH collect(old_entity) AS old_entities, release_id, corpus_id, snapshot_id, manifest_sha256,
     entities, relationships
FOREACH (old_entity IN old_entities | DETACH DELETE old_entity)
WITH release_id, corpus_id, snapshot_id, manifest_sha256, entities, relationships
MERGE (release:NorviiGraphRelease {id: release_id})
SET release.corpus_id = corpus_id, release.snapshot_id = snapshot_id,
    release.manifest_sha256 = manifest_sha256, release.status = 'ready'
WITH release, entities, relationships
UNWIND entities AS entity
MERGE (node:NorviiGraphEntity {release_id: release.id, semantic_entity_id: entity.id})
SET node.label = entity.label, node.normalized_label = entity.normalized_label,
    node.entity_type = entity.entity_type
MERGE (node)-[:IN_GRAPH_RELEASE]->(release)
WITH release, relationships
UNWIND relationships AS relationship
MATCH (subject:NorviiGraphEntity {
  release_id: release.id, semantic_entity_id: relationship.subject_entity_id
})
MATCH (object:NorviiGraphEntity {
  release_id: release.id, semantic_entity_id: relationship.object_entity_id
})
MERGE (subject)-[edge:LEGAL_RELATIONSHIP {release_id: release.id, id: relationship.id}]->(object)
SET edge.evidence_id = relationship.evidence_id,
    edge.source_id = relationship.source_id,
    edge.document_id = relationship.document_id,
    edge.source_revision_id = relationship.source_revision_id,
    edge.pipeline_version = relationship.pipeline_version,
    edge.source_title = relationship.source_title,
    edge.evidence_locator = relationship.evidence_locator,
    edge.start_offset = relationship.start_offset,
    edge.end_offset = relationship.end_offset,
    edge.excerpt = relationship.excerpt,
    edge.relationship_type = relationship.relationship_type
"""
