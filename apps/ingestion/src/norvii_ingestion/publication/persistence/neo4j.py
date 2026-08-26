"""Graph projection connectivity adapter."""

from __future__ import annotations

from dataclasses import dataclass
from itertools import batched
from typing import TYPE_CHECKING, Any, ClassVar, cast

from neo4j import Driver, EagerResult, GraphDatabase, ManagedTransaction, RoutingControl
from neo4j.exceptions import Neo4jError

from norvii_ingestion.publication.persistence.errors import PersistenceConnectionError

if TYPE_CHECKING:
    from uuid import UUID

    from norvii_ingestion.publication.persistence.config import Neo4jConfiguration


_WRITE_BATCH_SIZE = 250


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
        """Atomically replace one snapshot projection with bounded Cypher writes."""
        try:
            with self._driver.session(database=self._database) as session:
                session.execute_write(_replace_release_transaction, release)
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
    build_version: str
    legal_units: tuple[dict[str, object], ...]
    entities: tuple[dict[str, object], ...]
    assertions: tuple[dict[str, object], ...]


def _replace_release_transaction(
    transaction: ManagedTransaction, release: GraphReleaseProjection
) -> None:
    """Replace one release inside the managed transaction supplied by the driver."""
    parameters = _release_parameters(release)
    superseded_release_ids = _superseded_release_ids(transaction, parameters)
    _delete_superseded_projection(transaction, superseded_release_ids)
    _run(transaction, _CREATE_RELEASE, parameters)
    _write_projection_batches(transaction, parameters, "legal_units", _UPSERT_LEGAL_UNITS)
    _write_projection_batches(
        transaction, parameters, "legal_units", _CREATE_CONTAINS_RELATIONSHIPS
    )
    _write_projection_batches(transaction, parameters, "entities", _UPSERT_LEGAL_ENTITIES)
    _write_projection_batches(transaction, parameters, "assertions", _UPSERT_ASSERTIONS)
    _run(transaction, _MARK_RELEASE_READY, parameters)


def _release_parameters(release: GraphReleaseProjection) -> dict[str, object]:
    """Map an immutable release projection to the Cypher parameter contract."""
    return {
        "release_id": str(release.release_id),
        "corpus_id": str(release.corpus_id),
        "snapshot_id": str(release.snapshot_id),
        "manifest_sha256": release.manifest_sha256,
        "build_version": release.build_version,
        "legal_units": release.legal_units,
        "entities": release.entities,
        "assertions": release.assertions,
    }


def _superseded_release_ids(
    transaction: ManagedTransaction, parameters: dict[str, object]
) -> tuple[str, ...]:
    """Return only the release identifiers scoped to the replaced corpus snapshot."""
    result = transaction.run(_SUPERSEDED_RELEASE_IDS, cast("dict[str, Any]", parameters))
    record = result.single(strict=True)
    value = record["release_ids"]
    if not isinstance(value, list) or not all(isinstance(identifier, str) for identifier in value):
        raise ValueError("Neo4j returned invalid superseded graph release identifiers.")
    return tuple(value)


def _delete_superseded_projection(
    transaction: ManagedTransaction, release_ids: tuple[str, ...]
) -> None:
    """Delete all known projection node labels for retired v1/v2 releases in batches."""
    if not release_ids:
        return
    parameters = {"release_ids": list(release_ids), "batch_size": _WRITE_BATCH_SIZE}
    for query in _DELETE_PROJECTION_NODE_BATCHES:
        _delete_batches(transaction, query, parameters)
    _run(transaction, _DELETE_SUPERSEDED_RELEASES, parameters)


def _delete_batches(
    transaction: ManagedTransaction, query: str, parameters: dict[str, object]
) -> None:
    """Delete one labelled projection class until no scoped nodes remain."""
    deleted_count = _deleted_count(transaction, query, parameters)
    while deleted_count:
        deleted_count = _deleted_count(transaction, query, parameters)


def _deleted_count(
    transaction: ManagedTransaction, query: str, parameters: dict[str, object]
) -> int:
    """Execute one bounded deletion query and validate its affected-node count."""
    result = transaction.run(query, cast("dict[str, Any]", parameters))
    record = result.single(strict=True)
    value = record["deleted"]
    if not isinstance(value, int) or value < 0:
        raise ValueError("Neo4j returned an invalid graph deletion count.")
    return value


def _write_projection_batches(
    transaction: ManagedTransaction,
    release_parameters: dict[str, object],
    item_name: str,
    query: str,
) -> None:
    """Write a single projection collection in bounded, release-scoped batches."""
    items = _projection_items(release_parameters, item_name)
    for item_batch in batched(items, _WRITE_BATCH_SIZE, strict=False):
        parameters = release_parameters | {item_name: list(item_batch)}
        _run(transaction, query, parameters)


def _projection_items(
    parameters: dict[str, object], item_name: str
) -> tuple[dict[str, object], ...]:
    """Validate one pre-built projection collection before it reaches Cypher."""
    items = parameters[item_name]
    if not isinstance(items, tuple) or not all(isinstance(item, dict) for item in items):
        raise ValueError(f"Neo4j graph projection parameter {item_name} is invalid.")
    return items


def _run(transaction: ManagedTransaction, query: str, parameters: dict[str, object]) -> None:
    """Consume one write result before issuing the next statement in the transaction."""
    transaction.run(query, cast("dict[str, Any]", parameters)).consume()


_SUPERSEDED_RELEASE_IDS = """
MATCH (release:NorviiGraphRelease {
  corpus_id: $corpus_id, snapshot_id: $snapshot_id
})
RETURN collect(release.id) AS release_ids
"""

_DELETE_PROJECTION_NODE_BATCHES = (
    """
MATCH (node:NorviiGraphLegalUnit)
WHERE node.release_id IN $release_ids
WITH node LIMIT $batch_size
DETACH DELETE node
RETURN count(node) AS deleted
""",
    """
MATCH (node:NorviiGraphLegalEntity)
WHERE node.release_id IN $release_ids
WITH node LIMIT $batch_size
DETACH DELETE node
RETURN count(node) AS deleted
""",
    """
MATCH (node:NorviiGraphNormativeAssertion)
WHERE node.release_id IN $release_ids
WITH node LIMIT $batch_size
DETACH DELETE node
RETURN count(node) AS deleted
""",
)

_DELETE_SUPERSEDED_RELEASES = """
UNWIND $release_ids AS release_id
MATCH (release:NorviiGraphRelease {id: release_id})
DETACH DELETE release
"""

_CREATE_RELEASE = """
MERGE (release:NorviiGraphRelease {id: $release_id})
SET release.corpus_id = $corpus_id,
    release.snapshot_id = $snapshot_id,
    release.manifest_sha256 = $manifest_sha256,
    release.build_version = $build_version,
    release.status = 'building'
"""

_UPSERT_LEGAL_UNITS = """
MATCH (release:NorviiGraphRelease {id: $release_id})
UNWIND $legal_units AS legal_unit
MERGE (unit:NorviiGraphLegalUnit {
  release_id: $release_id, legal_unit_id: legal_unit.id
})
SET unit.document_id = legal_unit.document_id,
    unit.locator = legal_unit.locator,
    unit.canonical_locator = legal_unit.canonical_locator,
    unit.content_sha256 = legal_unit.content_sha256,
    unit.kind = legal_unit.kind
MERGE (unit)-[:IN_GRAPH_RELEASE]->(release)
"""

_CREATE_CONTAINS_RELATIONSHIPS = """
UNWIND $legal_units AS child_unit
WITH child_unit
WHERE child_unit.parent_id IS NOT NULL
MATCH (parent:NorviiGraphLegalUnit {
  release_id: $release_id, legal_unit_id: child_unit.parent_id
})
MATCH (child:NorviiGraphLegalUnit {
  release_id: $release_id, legal_unit_id: child_unit.id
})
MERGE (parent)-[:CONTAINS {release_id: $release_id}]->(child)
"""

_UPSERT_LEGAL_ENTITIES = """
MATCH (release:NorviiGraphRelease {id: $release_id})
UNWIND $entities AS entity
MERGE (entity_node:NorviiGraphLegalEntity {
  release_id: $release_id, semantic_entity_id: entity.id
})
SET entity_node.label = entity.label,
    entity_node.normalized_label = entity.normalized_label,
    entity_node.entity_type = entity.entity_type
MERGE (entity_node)-[:IN_GRAPH_RELEASE]->(release)
"""

_UPSERT_ASSERTIONS = """
UNWIND $assertions AS assertion
MATCH (release:NorviiGraphRelease {id: $release_id})
MATCH (establishing_unit:NorviiGraphLegalUnit {
  release_id: $release_id, legal_unit_id: assertion.establishing_unit_id
})
MATCH (subject:NorviiGraphLegalEntity {
  release_id: $release_id, semantic_entity_id: assertion.subject_entity_id
})
MATCH (object:NorviiGraphLegalEntity {
  release_id: $release_id, semantic_entity_id: assertion.object_entity_id
})
MERGE (assertion_node:NorviiGraphNormativeAssertion {
  release_id: $release_id, normative_assertion_id: assertion.id
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
    assertion_node.evidence_canonical_locator = assertion.evidence_canonical_locator,
    assertion_node.evidence_content_sha256 = assertion.evidence_content_sha256,
    assertion_node.evidence_unit_id = assertion.evidence_unit_id,
    assertion_node.start_offset = assertion.start_offset,
    assertion_node.end_offset = assertion.end_offset,
    assertion_node.excerpt = assertion.excerpt
MERGE (assertion_node)-[:IN_GRAPH_RELEASE]->(release)
MERGE (establishing_unit)-[:ESTABLISHES {release_id: $release_id}]->(assertion_node)
MERGE (assertion_node)-[:SUBJECT {release_id: $release_id}]->(subject)
MERGE (assertion_node)-[:OBJECT {release_id: $release_id}]->(object)
"""

_MARK_RELEASE_READY = """
MATCH (release:NorviiGraphRelease {id: $release_id})
SET release.status = 'ready'
"""
