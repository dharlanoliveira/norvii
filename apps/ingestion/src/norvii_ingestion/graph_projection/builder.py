"""Build a snapshot-scoped normative-assertion graph from canonical artifacts."""

from __future__ import annotations

import hashlib
from dataclasses import dataclass
from datetime import UTC, datetime
from typing import TYPE_CHECKING, cast
from uuid import UUID, uuid5

import psycopg

from norvii_ingestion.publication.persistence.errors import PersistenceConnectionError
from norvii_ingestion.publication.persistence.neo4j import GraphReleaseProjection

if TYPE_CHECKING:
    from collections.abc import Sequence

    from norvii_ingestion.publication.persistence.neo4j import Neo4jStore

_RELEASE_NAMESPACE = UUID("3ee8ec57-0d32-4eb8-b742-ae97c8077aa7")


class GraphProjectionBuildError(RuntimeError):
    """Report a safe graph-build failure without exposing corpus contents."""


@dataclass(frozen=True, slots=True)
class GraphReleaseSummary:
    """Inspectable result of one explicit snapshot graph build."""

    release_id: UUID
    corpus_id: UUID
    snapshot_id: UUID
    manifest_sha256: str
    legal_unit_count: int
    entity_count: int
    assertion_count: int
    reused: bool


@dataclass(frozen=True, slots=True)
class _LegalUnitProjection:
    id: UUID
    document_id: UUID
    parent_id: UUID | None
    kind: str
    locator: str

    def neo4j(self) -> dict[str, object]:
        return {
            "id": str(self.id),
            "document_id": str(self.document_id),
            "parent_id": None if self.parent_id is None else str(self.parent_id),
            "kind": self.kind,
            "locator": self.locator,
        }


@dataclass(frozen=True, slots=True)
class _EntityProjection:
    id: UUID
    label: str
    normalized_label: str
    entity_type: str

    def neo4j(self) -> dict[str, object]:
        return {
            "id": str(self.id),
            "label": self.label,
            "normalized_label": self.normalized_label,
            "entity_type": self.entity_type,
        }


@dataclass(frozen=True, slots=True)
class _AssertionProjection:
    id: UUID
    subject_entity_id: UUID
    object_entity_id: UUID
    establishing_unit_id: UUID
    evidence_unit_id: UUID
    source_id: UUID
    document_id: UUID
    source_revision_id: UUID
    pipeline_version: str
    source_title: str
    establishing_locator: str
    evidence_locator: str
    start_offset: int
    end_offset: int
    excerpt: str
    predicate: str
    qualifier: str | None

    def neo4j(self) -> dict[str, object]:
        return {
            "id": str(self.id),
            "subject_entity_id": str(self.subject_entity_id),
            "object_entity_id": str(self.object_entity_id),
            "establishing_unit_id": str(self.establishing_unit_id),
            "evidence_unit_id": str(self.evidence_unit_id),
            "evidence_id": str(self.id),
            "source_id": str(self.source_id),
            "document_id": str(self.document_id),
            "source_revision_id": str(self.source_revision_id),
            "pipeline_version": self.pipeline_version,
            "source_title": self.source_title,
            "establishing_locator": self.establishing_locator,
            "evidence_locator": self.evidence_locator,
            "start_offset": self.start_offset,
            "end_offset": self.end_offset,
            "excerpt": self.excerpt,
            "predicate": self.predicate,
            "qualifier": self.qualifier,
        }


class GraphReleaseBuilder:
    """Own explicit idempotent builds of one immutable published snapshot projection."""

    def __init__(
        self,
        connection: psycopg.Connection[tuple[object, ...]],
        neo4j: Neo4jStore,
        build_version: str = "legal-assertion-graph-v1",
    ) -> None:
        if not build_version.strip():
            raise ValueError("graph build version is required")
        self._connection = connection
        self._neo4j = neo4j
        self._build_version = build_version

    def build(self, corpus_id: UUID, snapshot_id: UUID) -> GraphReleaseSummary:
        """Project only canonical legal units and supported assertions in one snapshot."""
        legal_units, entities, assertions = self._load_snapshot_projection(corpus_id, snapshot_id)
        _validate_projection_inputs(legal_units, entities, assertions)
        if not legal_units:
            raise GraphProjectionBuildError("No legal units exist for this snapshot.")
        manifest = _manifest(legal_units, entities, assertions)
        release_id = uuid5(_RELEASE_NAMESPACE, f"{snapshot_id}:{self._build_version}:{manifest}")
        summary = GraphReleaseSummary(
            release_id=release_id,
            corpus_id=corpus_id,
            snapshot_id=snapshot_id,
            manifest_sha256=manifest,
            legal_unit_count=len(legal_units),
            entity_count=len(entities),
            assertion_count=len(assertions),
            reused=False,
        )
        if self._ready_release(summary):
            return GraphReleaseSummary(
                release_id=summary.release_id,
                corpus_id=summary.corpus_id,
                snapshot_id=summary.snapshot_id,
                manifest_sha256=summary.manifest_sha256,
                legal_unit_count=summary.legal_unit_count,
                entity_count=summary.entity_count,
                assertion_count=summary.assertion_count,
                reused=True,
            )
        self._record_building(summary)
        try:
            self._neo4j.replace_release(
                GraphReleaseProjection(
                    release_id=summary.release_id,
                    corpus_id=corpus_id,
                    snapshot_id=snapshot_id,
                    manifest_sha256=manifest,
                    legal_units=tuple(unit.neo4j() for unit in legal_units),
                    entities=tuple(entity.neo4j() for entity in entities),
                    assertions=tuple(assertion.neo4j() for assertion in assertions),
                )
            )
            self._record_ready(summary, legal_units, entities, assertions)
        except (GraphProjectionBuildError, PersistenceConnectionError, psycopg.Error) as error:
            self._record_failed(summary)
            if isinstance(error, GraphProjectionBuildError):
                raise
            raise GraphProjectionBuildError("Graph projection build failed.") from error
        return summary

    def _load_snapshot_projection(
        self, corpus_id: UUID, snapshot_id: UUID
    ) -> tuple[
        tuple[_LegalUnitProjection, ...],
        tuple[_EntityProjection, ...],
        tuple[_AssertionProjection, ...],
    ]:
        try:
            with self._connection.transaction(), self._connection.cursor() as cursor:
                cursor.execute(_LEGAL_UNIT_QUERY, (corpus_id, snapshot_id))
                legal_units = tuple(_legal_unit(row) for row in cursor.fetchall())
                cursor.execute(_ENTITY_QUERY, (corpus_id, snapshot_id))
                entities = tuple(_entity(row) for row in cursor.fetchall())
                cursor.execute(_ASSERTION_QUERY, (corpus_id, snapshot_id))
                assertions = tuple(_assertion(row) for row in cursor.fetchall())
        except psycopg.Error as error:
            raise GraphProjectionBuildError("Read graph projection inputs failed.") from error
        return legal_units, entities, assertions

    def _ready_release(self, summary: GraphReleaseSummary) -> bool:
        with self._connection.transaction(), self._connection.cursor() as cursor:
            cursor.execute(
                """
                SELECT status FROM graph_releases
                WHERE id = %s AND corpus_id = %s AND snapshot_id = %s
                """,
                (summary.release_id, summary.corpus_id, summary.snapshot_id),
            )
            row = cursor.fetchone()
        return row is not None and cast("str", row[0]) == "ready"

    def _record_building(self, summary: GraphReleaseSummary) -> None:
        now = datetime.now(UTC)
        with self._connection.transaction(), self._connection.cursor() as cursor:
            cursor.execute(
                """
                INSERT INTO graph_releases (
                    id, corpus_id, snapshot_id, manifest_sha256, build_version, status,
                    entity_count, relationship_count, created_at
                ) VALUES (%s, %s, %s, %s, %s, 'building', %s, %s, %s)
                ON CONFLICT (id) DO UPDATE
                SET status = 'building', failure_category = NULL, completed_at = NULL,
                    entity_count = EXCLUDED.entity_count,
                    relationship_count = EXCLUDED.relationship_count
                """,
                (
                    summary.release_id,
                    summary.corpus_id,
                    summary.snapshot_id,
                    summary.manifest_sha256,
                    self._build_version,
                    summary.entity_count,
                    summary.assertion_count,
                    now,
                ),
            )

    def _record_ready(
        self,
        summary: GraphReleaseSummary,
        legal_units: Sequence[_LegalUnitProjection],
        entities: Sequence[_EntityProjection],
        assertions: Sequence[_AssertionProjection],
    ) -> None:
        now = datetime.now(UTC)
        with self._connection.transaction(), self._connection.cursor() as cursor:
            cursor.execute(
                "DELETE FROM graph_release_legal_units WHERE graph_release_id = %s",
                (summary.release_id,),
            )
            cursor.execute(
                "DELETE FROM graph_release_entities WHERE graph_release_id = %s",
                (summary.release_id,),
            )
            cursor.execute(
                "DELETE FROM graph_release_assertions WHERE graph_release_id = %s",
                (summary.release_id,),
            )
            cursor.executemany(
                """
                INSERT INTO graph_release_legal_units (graph_release_id, document_id, legal_unit_id)
                VALUES (%s, %s, %s)
                """,
                [(summary.release_id, unit.document_id, unit.id) for unit in legal_units],
            )
            cursor.executemany(
                """
                INSERT INTO graph_release_entities (graph_release_id, semantic_entity_id)
                VALUES (%s, %s)
                """,
                [(summary.release_id, entity.id) for entity in entities],
            )
            cursor.executemany(
                """
                INSERT INTO graph_release_assertions (graph_release_id, normative_assertion_id)
                VALUES (%s, %s)
                """,
                [(summary.release_id, assertion.id) for assertion in assertions],
            )
            cursor.execute(
                """
                UPDATE graph_releases
                SET status = 'ready', failure_category = NULL, completed_at = %s
                WHERE id = %s AND status = 'building'
                """,
                (now, summary.release_id),
            )
            if cursor.rowcount != 1:
                raise GraphProjectionBuildError("Graph release state changed during projection.")

    def _record_failed(self, summary: GraphReleaseSummary) -> None:
        try:
            with self._connection.transaction(), self._connection.cursor() as cursor:
                cursor.execute(
                    """
                    UPDATE graph_releases
                    SET status = 'failed', failure_category = 'projection_failed', completed_at = %s
                    WHERE id = %s AND status = 'building'
                    """,
                    (datetime.now(UTC), summary.release_id),
                )
        except psycopg.Error:
            return


def _validate_projection_inputs(
    legal_units: Sequence[_LegalUnitProjection],
    entities: Sequence[_EntityProjection],
    assertions: Sequence[_AssertionProjection],
) -> None:
    legal_unit_ids = {unit.id for unit in legal_units}
    entity_ids = {entity.id for entity in entities}
    if len(legal_unit_ids) != len(legal_units) or len(entity_ids) != len(entities):
        raise GraphProjectionBuildError("Graph projection inputs are inconsistent.")
    if any(
        unit.parent_id is not None and unit.parent_id not in legal_unit_ids for unit in legal_units
    ) or any(
        assertion.subject_entity_id not in entity_ids
        or assertion.object_entity_id not in entity_ids
        or assertion.establishing_unit_id not in legal_unit_ids
        or assertion.evidence_unit_id not in legal_unit_ids
        for assertion in assertions
    ):
        raise GraphProjectionBuildError("Graph projection inputs are inconsistent.")


def _legal_unit(row: tuple[object, ...]) -> _LegalUnitProjection:
    return _LegalUnitProjection(
        id=cast("UUID", row[0]),
        document_id=cast("UUID", row[1]),
        parent_id=cast("UUID | None", row[2]),
        kind=cast("str", row[3]),
        locator=cast("str", row[4]),
    )


def _entity(row: tuple[object, ...]) -> _EntityProjection:
    return _EntityProjection(
        id=cast("UUID", row[0]),
        label=cast("str", row[1]),
        normalized_label=cast("str", row[2]),
        entity_type=cast("str", row[3]),
    )


def _assertion(row: tuple[object, ...]) -> _AssertionProjection:
    return _AssertionProjection(
        id=cast("UUID", row[0]),
        subject_entity_id=cast("UUID", row[1]),
        object_entity_id=cast("UUID", row[2]),
        establishing_unit_id=cast("UUID", row[3]),
        evidence_unit_id=cast("UUID", row[4]),
        source_id=cast("UUID", row[5]),
        document_id=cast("UUID", row[6]),
        source_revision_id=cast("UUID", row[7]),
        pipeline_version=cast("str", row[8]),
        source_title=cast("str", row[9]),
        establishing_locator=cast("str", row[10]),
        evidence_locator=cast("str", row[11]),
        start_offset=cast("int", row[12]),
        end_offset=cast("int", row[13]),
        excerpt=cast("str", row[14]),
        predicate=cast("str", row[15]),
        qualifier=cast("str | None", row[16]),
    )


def _manifest(
    legal_units: Sequence[_LegalUnitProjection],
    entities: Sequence[_EntityProjection],
    assertions: Sequence[_AssertionProjection],
) -> str:
    values = [
        *(f"legal_unit:{unit.id}" for unit in legal_units),
        *(f"entity:{entity.id}" for entity in entities),
        *(f"assertion:{assertion.id}" for assertion in assertions),
    ]
    return hashlib.sha256("\n".join(sorted(values)).encode("utf-8")).hexdigest()


_LEGAL_UNIT_QUERY = """
SELECT unit.id, unit.document_id, unit.parent_id, unit.kind, unit.locator
FROM corpus_snapshot_documents snapshot_document
JOIN document_units unit
  ON unit.document_id = snapshot_document.document_id
WHERE snapshot_document.corpus_id = %s AND snapshot_document.snapshot_id = %s
ORDER BY unit.document_id, unit.id
"""

_ENTITY_QUERY = """
SELECT DISTINCT entity.id, entity.label, entity.normalized_label, entity.entity_type
FROM corpus_snapshot_documents snapshot_document
JOIN semantic_extraction_runs extraction
  ON extraction.document_id = snapshot_document.document_id
 AND extraction.corpus_id = snapshot_document.corpus_id
 AND extraction.source_id = snapshot_document.source_id
 AND extraction.status = 'ready'
JOIN normative_assertions assertion
  ON assertion.extraction_run_id = extraction.id
 AND assertion.validation_status = 'supported'
JOIN semantic_entities entity
  ON (entity.id = assertion.subject_entity_id OR entity.id = assertion.object_entity_id)
 AND entity.validation_status = 'supported'
WHERE snapshot_document.corpus_id = %s AND snapshot_document.snapshot_id = %s
ORDER BY entity.id
"""

_ASSERTION_QUERY = """
SELECT assertion.id, assertion.subject_entity_id, assertion.object_entity_id,
       assertion.establishing_unit_id, assertion.evidence_unit_id, assertion.source_id,
       assertion.document_id, document.source_revision_id, document.pipeline_version,
       source.title, establishing_unit.locator, evidence_unit.locator,
       evidence_unit.start_offset, evidence_unit.end_offset,
       left(
         substring(
           document.text_content FROM evidence_unit.start_offset + 1
           FOR evidence_unit.end_offset - evidence_unit.start_offset
         ),
         1200
       ),
       assertion.predicate, assertion.qualifier
FROM corpus_snapshot_documents snapshot_document
JOIN semantic_extraction_runs extraction
  ON extraction.document_id = snapshot_document.document_id
 AND extraction.corpus_id = snapshot_document.corpus_id
 AND extraction.source_id = snapshot_document.source_id
 AND extraction.status = 'ready'
JOIN normative_assertions assertion
  ON assertion.extraction_run_id = extraction.id
 AND assertion.validation_status = 'supported'
JOIN document_versions document
  ON document.id = assertion.document_id
 AND document.corpus_id = snapshot_document.corpus_id
 AND document.source_id = snapshot_document.source_id
JOIN sources source
  ON source.id = assertion.source_id
 AND source.corpus_id = snapshot_document.corpus_id
JOIN document_units establishing_unit
  ON establishing_unit.document_id = assertion.document_id
 AND establishing_unit.id = assertion.establishing_unit_id
JOIN document_units evidence_unit
  ON evidence_unit.document_id = assertion.document_id
 AND evidence_unit.id = assertion.evidence_unit_id
WHERE snapshot_document.corpus_id = %s AND snapshot_document.snapshot_id = %s
ORDER BY assertion.id
"""
