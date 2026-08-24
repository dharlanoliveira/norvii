from __future__ import annotations

from typing import TYPE_CHECKING, cast
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


def _raise_state_change(
    _summary: object,
    _entities: object,
    _relationships: object,
) -> None:
    raise GraphProjectionBuildError("Graph release state changed during projection.")
