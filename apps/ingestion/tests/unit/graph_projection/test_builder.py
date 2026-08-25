from __future__ import annotations

from typing import TYPE_CHECKING, Self, cast
from uuid import uuid4

import pytest

from norvii_ingestion.graph_projection.builder import (
    GraphProjectionBuildError,
    GraphReleaseBuilder,
    _EntityProjection,
    _RelationshipProjection,
)

if TYPE_CHECKING:
    import psycopg

    from norvii_ingestion.publication.persistence.neo4j import (
        GraphReleaseProjection,
        Neo4jStore,
    )


class RecordingGraphStore:
    def __init__(self) -> None:
        self.releases: list[GraphReleaseProjection] = []

    def replace_release(self, release: GraphReleaseProjection) -> None:
        self.releases.append(release)


class _ReadTransaction:
    def __init__(self, connection: RecordingReadConnection) -> None:
        self._connection = connection

    def __enter__(self) -> None:
        return None

    def __exit__(self, *_arguments: object) -> None:
        self._connection.completed_transactions += 1


class _ReadCursor:
    def __init__(self, connection: RecordingReadConnection) -> None:
        self._connection = connection
        self._result: tuple[tuple[object, ...], ...] = ()
        self.rowcount = 1

    def __enter__(self) -> Self:
        return self

    def __exit__(self, *_arguments: object) -> None:
        return None

    def execute(self, query: str, _parameters: tuple[object, ...]) -> None:
        if query.lstrip().startswith("SELECT"):
            self._result = self._connection.select_results.pop(0)

    def executemany(self, _query: str, _parameters: list[tuple[object, ...]]) -> None:
        return None

    def fetchall(self) -> tuple[tuple[object, ...], ...]:
        return self._result

    def fetchone(self) -> tuple[object, ...] | None:
        return self._result[0] if self._result else None


class RecordingReadConnection:
    def __init__(self, results: list[tuple[tuple[object, ...], ...]]) -> None:
        self.select_results = results
        self.completed_transactions = 0

    def transaction(self) -> _ReadTransaction:
        return _ReadTransaction(self)

    def cursor(self) -> _ReadCursor:
        return _ReadCursor(self)


def test_builder_marks_a_release_failed_when_final_persistence_state_changes(
    monkeypatch: pytest.MonkeyPatch,
) -> None:
    corpus_id = uuid4()
    snapshot_id = uuid4()
    graph = RecordingGraphStore()
    builder = GraphReleaseBuilder(
        cast("psycopg.Connection[tuple[object, ...]]", object()),
        cast("Neo4jStore", graph),
    )
    entity = _EntityProjection(uuid4(), "Controller", "controller", "actor")
    relationship = _RelationshipProjection(
        id=uuid4(),
        subject_entity_id=entity.id,
        object_entity_id=entity.id,
        evidence_unit_id=uuid4(),
        source_id=uuid4(),
        document_id=uuid4(),
        source_revision_id=uuid4(),
        pipeline_version="test-pipeline",
        source_title="Official source",
        evidence_locator="Article 1",
        start_offset=0,
        end_offset=10,
        excerpt="Legal text",
        relationship_type="governs",
    )
    recorded_failures: list[object] = []

    monkeypatch.setattr(
        builder,
        "_load_snapshot_projection",
        lambda _corpus_id, _snapshot_id: ((entity,), (relationship,)),
    )
    monkeypatch.setattr(builder, "_ready_release", lambda _summary: False)
    monkeypatch.setattr(builder, "_record_building", lambda _summary: None)
    monkeypatch.setattr(builder, "_record_ready", _raise_state_change)
    monkeypatch.setattr(builder, "_record_failed", recorded_failures.append)

    with pytest.raises(GraphProjectionBuildError, match="state changed"):
        builder.build(corpus_id, snapshot_id)

    assert len(graph.releases) == 1
    assert len(recorded_failures) == 1


def test_builder_commits_snapshot_reads_before_recording_a_graph_release() -> None:
    corpus_id = uuid4()
    snapshot_id = uuid4()
    entity = _EntityProjection(uuid4(), "Controller", "controller", "actor")
    relationship = _RelationshipProjection(
        id=uuid4(),
        subject_entity_id=entity.id,
        object_entity_id=entity.id,
        evidence_unit_id=uuid4(),
        source_id=uuid4(),
        document_id=uuid4(),
        source_revision_id=uuid4(),
        pipeline_version="test-pipeline",
        source_title="Official source",
        evidence_locator="Article 1",
        start_offset=0,
        end_offset=10,
        excerpt="Legal text",
        relationship_type="governs",
    )
    connection = RecordingReadConnection(
        [
            ((entity.id, entity.label, entity.normalized_label, entity.entity_type),),
            (
                (
                    relationship.id,
                    relationship.subject_entity_id,
                    relationship.object_entity_id,
                    relationship.evidence_unit_id,
                    relationship.source_id,
                    relationship.document_id,
                    relationship.source_revision_id,
                    relationship.pipeline_version,
                    relationship.source_title,
                    relationship.evidence_locator,
                    relationship.start_offset,
                    relationship.end_offset,
                    relationship.excerpt,
                    relationship.relationship_type,
                ),
            ),
            (),
        ]
    )
    builder = GraphReleaseBuilder(
        cast("psycopg.Connection[tuple[object, ...]]", connection),
        cast("Neo4jStore", RecordingGraphStore()),
    )

    summary = builder.build(corpus_id, snapshot_id)

    assert summary.reused is False
    assert connection.completed_transactions == 4


def test_builder_reuses_the_same_manifest_across_three_snapshot_builds(
    monkeypatch: pytest.MonkeyPatch,
) -> None:
    corpus_id = uuid4()
    snapshot_id = uuid4()
    entity = _EntityProjection(uuid4(), "Controller", "controller", "actor")
    relationship = _RelationshipProjection(
        id=uuid4(),
        subject_entity_id=entity.id,
        object_entity_id=entity.id,
        evidence_unit_id=uuid4(),
        source_id=uuid4(),
        document_id=uuid4(),
        source_revision_id=uuid4(),
        pipeline_version="test-pipeline",
        source_title="Official source",
        evidence_locator="Article 1",
        start_offset=0,
        end_offset=10,
        excerpt="Legal text",
        relationship_type="governs",
    )
    graph = RecordingGraphStore()
    builder = GraphReleaseBuilder(
        cast("psycopg.Connection[tuple[object, ...]]", object()),
        cast("Neo4jStore", graph),
    )
    ready: set[object] = set()
    monkeypatch.setattr(
        builder,
        "_load_snapshot_projection",
        lambda _corpus, _snapshot: ((entity,), (relationship,)),
    )
    monkeypatch.setattr(builder, "_ready_release", lambda summary: summary.release_id in ready)
    monkeypatch.setattr(builder, "_record_building", lambda _summary: None)
    monkeypatch.setattr(
        builder,
        "_record_ready",
        lambda summary, _entities, _relationships: ready.add(summary.release_id),
    )

    first = builder.build(corpus_id, snapshot_id)
    second = builder.build(corpus_id, snapshot_id)
    third = builder.build(corpus_id, snapshot_id)

    assert first.reused is False
    assert second.reused is True
    assert third.reused is True
    assert first.release_id == second.release_id == third.release_id
    assert first.manifest_sha256 == second.manifest_sha256 == third.manifest_sha256
    assert len(graph.releases) == 1


def _raise_state_change(
    _summary: object,
    _entities: object,
    _relationships: object,
) -> None:
    raise GraphProjectionBuildError("Graph release state changed during projection.")
