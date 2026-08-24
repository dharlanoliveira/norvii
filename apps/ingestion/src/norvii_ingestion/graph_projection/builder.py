"""Build a snapshot-scoped derived graph from PostgreSQL canonical artifacts."""

from __future__ import annotations

import hashlib
from dataclasses import dataclass, replace
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
    entity_count: int
    relationship_count: int
    reused: bool


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
class _RelationshipProjection:
    id: UUID
    subject_entity_id: UUID
    object_entity_id: UUID
    evidence_unit_id: UUID
    source_id: UUID
    document_id: UUID
    source_revision_id: UUID
    pipeline_version: str
    source_title: str
    evidence_locator: str
    start_offset: int
    end_offset: int
    excerpt: str
    relationship_type: str

    def neo4j(self) -> dict[str, object]:
        return {
            "id": str(self.id),
            "subject_entity_id": str(self.subject_entity_id),
            "object_entity_id": str(self.object_entity_id),
            "evidence_id": str(self.id),
            "source_id": str(self.source_id),
            "document_id": str(self.document_id),
            "source_revision_id": str(self.source_revision_id),
            "pipeline_version": self.pipeline_version,
            "source_title": self.source_title,
            "evidence_locator": self.evidence_locator,
            "start_offset": self.start_offset,
            "end_offset": self.end_offset,
            "excerpt": self.excerpt,
            "relationship_type": self.relationship_type,
        }


class GraphReleaseBuilder:
    """Own explicit idempotent builds of one immutable published snapshot projection."""

    def __init__(
        self,
        connection: psycopg.Connection[tuple[object, ...]],
        neo4j: Neo4jStore,
        build_version: str = "legal-graph-v1",
    ) -> None:
        if not build_version.strip():
            raise ValueError("graph build version is required")
        self._connection = connection
        self._neo4j = neo4j
        self._build_version = build_version

    def build(self, corpus_id: UUID, snapshot_id: UUID) -> GraphReleaseSummary:
        """Project only supported semantic records owned by the named published snapshot."""
        entities, relationships = self._load_snapshot_projection(corpus_id, snapshot_id)
        if not relationships:
            raise GraphProjectionBuildError(
                "No supported graph relationships exist for this snapshot."
            )
        manifest = _manifest(entities, relationships)
        release_id = uuid5(_RELEASE_NAMESPACE, f"{snapshot_id}:{self._build_version}:{manifest}")
        summary = GraphReleaseSummary(
            release_id=release_id,
            corpus_id=corpus_id,
            snapshot_id=snapshot_id,
            manifest_sha256=manifest,
            entity_count=len(entities),
            relationship_count=len(relationships),
            reused=False,
        )
        if self._ready_release(summary):
            return replace(summary, reused=True)
        self._record_building(summary)
        try:
            self._neo4j.replace_release(
                GraphReleaseProjection(
                    release_id=summary.release_id,
                    corpus_id=corpus_id,
                    snapshot_id=snapshot_id,
                    manifest_sha256=manifest,
                    entities=tuple(entity.neo4j() for entity in entities),
                    relationships=tuple(relationship.neo4j() for relationship in relationships),
                )
            )
            self._record_ready(summary, entities, relationships)
        except (GraphProjectionBuildError, PersistenceConnectionError, psycopg.Error) as error:
            self._record_failed(summary)
            if isinstance(error, GraphProjectionBuildError):
                raise
            raise GraphProjectionBuildError("Graph projection build failed.") from error
        return summary

    def _load_snapshot_projection(
        self, corpus_id: UUID, snapshot_id: UUID
    ) -> tuple[tuple[_EntityProjection, ...], tuple[_RelationshipProjection, ...]]:
        try:
            with self._connection.cursor() as cursor:
                cursor.execute(_ENTITY_QUERY, (corpus_id, snapshot_id))
                entities = tuple(_entity(row) for row in cursor.fetchall())
                cursor.execute(_RELATIONSHIP_QUERY, (corpus_id, snapshot_id))
                relationships = tuple(_relationship(row) for row in cursor.fetchall())
        except psycopg.Error as error:
            raise GraphProjectionBuildError("Read graph projection inputs failed.") from error
        entity_ids = {entity.id for entity in entities}
        if not entity_ids or any(
            relationship.subject_entity_id not in entity_ids
            or relationship.object_entity_id not in entity_ids
            for relationship in relationships
        ):
            raise GraphProjectionBuildError("Graph projection inputs are inconsistent.")
        return entities, relationships

    def _ready_release(self, summary: GraphReleaseSummary) -> bool:
        with self._connection.cursor() as cursor:
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
                    summary.relationship_count,
                    now,
                ),
            )

    def _record_ready(
        self,
        summary: GraphReleaseSummary,
        entities: Sequence[_EntityProjection],
        relationships: Sequence[_RelationshipProjection],
    ) -> None:
        now = datetime.now(UTC)
        with self._connection.transaction(), self._connection.cursor() as cursor:
            cursor.execute(
                "DELETE FROM graph_release_entities WHERE graph_release_id = %s",
                (summary.release_id,),
            )
            cursor.execute(
                "DELETE FROM graph_release_relationships WHERE graph_release_id = %s",
                (summary.release_id,),
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
                INSERT INTO graph_release_relationships (graph_release_id, semantic_relationship_id)
                VALUES (%s, %s)
                """,
                [(summary.release_id, relationship.id) for relationship in relationships],
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


def _entity(row: tuple[object, ...]) -> _EntityProjection:
    return _EntityProjection(
        id=cast("UUID", row[0]),
        label=cast("str", row[1]),
        normalized_label=cast("str", row[2]),
        entity_type=cast("str", row[3]),
    )


def _relationship(row: tuple[object, ...]) -> _RelationshipProjection:
    return _RelationshipProjection(
        id=cast("UUID", row[0]),
        subject_entity_id=cast("UUID", row[1]),
        object_entity_id=cast("UUID", row[2]),
        evidence_unit_id=cast("UUID", row[3]),
        source_id=cast("UUID", row[4]),
        document_id=cast("UUID", row[5]),
        source_revision_id=cast("UUID", row[6]),
        pipeline_version=cast("str", row[7]),
        source_title=cast("str", row[8]),
        evidence_locator=cast("str", row[9]),
        start_offset=cast("int", row[10]),
        end_offset=cast("int", row[11]),
        excerpt=cast("str", row[12]),
        relationship_type=cast("str", row[13]),
    )


def _manifest(
    entities: Sequence[_EntityProjection], relationships: Sequence[_RelationshipProjection]
) -> str:
    values = [
        *(f"entity:{entity.id}" for entity in entities),
        *(f"relationship:{item.id}" for item in relationships),
    ]
    return hashlib.sha256("\n".join(sorted(values)).encode("utf-8")).hexdigest()


_ENTITY_QUERY = """
SELECT DISTINCT entity.id, entity.label, entity.normalized_label, entity.entity_type
FROM corpus_snapshot_documents snapshot_document
JOIN semantic_extraction_runs extraction
  ON extraction.document_id = snapshot_document.document_id
 AND extraction.corpus_id = snapshot_document.corpus_id
 AND extraction.source_id = snapshot_document.source_id
 AND extraction.status = 'ready'
JOIN semantic_entities entity
  ON entity.extraction_run_id = extraction.id
 AND entity.validation_status = 'supported'
WHERE snapshot_document.corpus_id = %s AND snapshot_document.snapshot_id = %s
ORDER BY entity.id
"""

_RELATIONSHIP_QUERY = """
SELECT relationship.id, relationship.subject_entity_id, relationship.object_entity_id,
       relationship.evidence_unit_id, relationship.source_id, relationship.document_id,
       document.source_revision_id, document.pipeline_version, source.title, unit.locator,
       unit.start_offset, unit.end_offset,
       left(
         substring(
           document.text_content FROM unit.start_offset + 1
           FOR unit.end_offset - unit.start_offset
         ),
         1200
       ),
       relationship.relationship_type
FROM corpus_snapshot_documents snapshot_document
JOIN semantic_extraction_runs extraction
  ON extraction.document_id = snapshot_document.document_id
 AND extraction.corpus_id = snapshot_document.corpus_id
 AND extraction.source_id = snapshot_document.source_id
 AND extraction.status = 'ready'
JOIN semantic_relationships relationship
  ON relationship.extraction_run_id = extraction.id
 AND relationship.validation_status = 'supported'
JOIN document_versions document
  ON document.id = relationship.document_id
 AND document.corpus_id = snapshot_document.corpus_id
 AND document.source_id = snapshot_document.source_id
JOIN sources source
  ON source.id = relationship.source_id
 AND source.corpus_id = snapshot_document.corpus_id
JOIN document_units unit
  ON unit.document_id = relationship.document_id
 AND unit.id = relationship.evidence_unit_id
WHERE snapshot_document.corpus_id = %s AND snapshot_document.snapshot_id = %s
ORDER BY relationship.id
"""
