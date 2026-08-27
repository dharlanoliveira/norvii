from __future__ import annotations

from dataclasses import replace
from typing import TYPE_CHECKING, Self, cast
from uuid import uuid4

import pytest

from norvii_ingestion.graph_projection.builder import (
    GraphProjectionBuildError,
    GraphReleaseBuilder,
    _AssertionProjection,
    _EntityProjection,
    _LegalUnitProjection,
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


def test_builder_projects_assertion_topology_with_exact_provenance(
    monkeypatch: pytest.MonkeyPatch,
) -> None:
    corpus_id = uuid4()
    snapshot_id = uuid4()
    legal_units, entities, assertions = _projection_inputs()
    graph = RecordingGraphStore()
    builder = GraphReleaseBuilder(
        cast("psycopg.Connection[tuple[object, ...]]", object()),
        cast("Neo4jStore", graph),
    )

    monkeypatch.setattr(
        builder,
        "_load_snapshot_projection",
        lambda _corpus_id, _snapshot_id: (legal_units, entities, assertions),
    )
    monkeypatch.setattr(builder, "_ready_release", lambda _summary: False)
    monkeypatch.setattr(builder, "_record_building", lambda _summary: None)
    monkeypatch.setattr(builder, "_record_ready", lambda *_arguments: None)

    summary = builder.build(corpus_id, snapshot_id)

    release = graph.releases[0]
    assertion = release.assertions[0]
    assert summary.legal_unit_count == 2
    assert summary.entity_count == 2
    assert summary.assertion_count == 1
    assert release.build_version == "legal-assertion-graph-v2"
    assert release.legal_units[1]["parent_id"] == str(legal_units[0].id)
    assert assertion["establishing_unit_id"] == str(legal_units[1].id)
    assert assertion["evidence_unit_id"] == str(legal_units[1].id)
    assert assertion["predicate"] == "imposes_duty_on"
    assert assertion["evidence_locator"] == "article-1"


def test_builder_commits_snapshot_reads_before_recording_a_graph_release() -> None:
    corpus_id = uuid4()
    snapshot_id = uuid4()
    legal_units, entities, assertions = _projection_inputs()
    connection = RecordingReadConnection(
        [
            tuple(
                (
                    unit.id,
                    unit.document_id,
                    unit.parent_id,
                    unit.kind,
                    unit.locator,
                    unit.canonical_locator,
                    unit.content_sha256,
                )
                for unit in legal_units
            ),
            tuple(
                (entity.id, entity.label, entity.normalized_label, entity.entity_type)
                for entity in entities
            ),
            tuple(
                (
                    assertion.id,
                    assertion.subject_entity_id,
                    assertion.object_entity_id,
                    assertion.establishing_unit_id,
                    assertion.evidence_unit_id,
                    assertion.source_id,
                    assertion.document_id,
                    assertion.source_revision_id,
                    assertion.pipeline_version,
                    assertion.source_title,
                    assertion.establishing_locator,
                    assertion.evidence_locator,
                    assertion.start_offset,
                    assertion.end_offset,
                    assertion.excerpt,
                    assertion.predicate,
                    assertion.qualifier,
                    assertion.evidence_canonical_locator,
                    assertion.evidence_content_sha256,
                )
                for assertion in assertions
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


def test_builder_allows_a_structural_snapshot_without_assertions(
    monkeypatch: pytest.MonkeyPatch,
) -> None:
    corpus_id = uuid4()
    snapshot_id = uuid4()
    legal_units, _, _ = _projection_inputs()
    graph = RecordingGraphStore()
    builder = GraphReleaseBuilder(
        cast("psycopg.Connection[tuple[object, ...]]", object()),
        cast("Neo4jStore", graph),
    )
    monkeypatch.setattr(
        builder,
        "_load_snapshot_projection",
        lambda _corpus_id, _snapshot_id: (legal_units, (), ()),
    )
    monkeypatch.setattr(builder, "_ready_release", lambda _summary: False)
    monkeypatch.setattr(builder, "_record_building", lambda _summary: None)
    monkeypatch.setattr(builder, "_record_ready", lambda *_arguments: None)

    summary = builder.build(corpus_id, snapshot_id)

    assert summary.assertion_count == 0
    assert graph.releases[0].entities == ()
    assert graph.releases[0].assertions == ()


def test_builder_rejects_an_assertion_with_an_unprojected_establishing_unit(
    monkeypatch: pytest.MonkeyPatch,
) -> None:
    corpus_id = uuid4()
    snapshot_id = uuid4()
    legal_units, entities, assertions = _projection_inputs()
    invalid = replace(assertions[0], establishing_unit_id=uuid4())
    builder = GraphReleaseBuilder(
        cast("psycopg.Connection[tuple[object, ...]]", object()),
        cast("Neo4jStore", RecordingGraphStore()),
    )
    monkeypatch.setattr(
        builder,
        "_load_snapshot_projection",
        lambda _corpus_id, _snapshot_id: (legal_units, entities, (invalid,)),
    )
    monkeypatch.setattr(builder, "_ready_release", lambda _summary: False)
    monkeypatch.setattr(builder, "_record_building", lambda _summary: None)

    with pytest.raises(GraphProjectionBuildError, match="inputs are inconsistent"):
        builder.build(corpus_id, snapshot_id)


def test_builder_reuses_the_same_manifest_across_three_snapshot_builds(
    monkeypatch: pytest.MonkeyPatch,
) -> None:
    corpus_id = uuid4()
    snapshot_id = uuid4()
    legal_units, entities, assertions = _projection_inputs()
    graph = RecordingGraphStore()
    builder = GraphReleaseBuilder(
        cast("psycopg.Connection[tuple[object, ...]]", object()),
        cast("Neo4jStore", graph),
    )
    ready: set[object] = set()
    monkeypatch.setattr(
        builder,
        "_load_snapshot_projection",
        lambda _corpus, _snapshot: (legal_units, entities, assertions),
    )
    monkeypatch.setattr(builder, "_ready_release", lambda summary: summary.release_id in ready)
    monkeypatch.setattr(builder, "_record_building", lambda _summary: None)
    monkeypatch.setattr(
        builder,
        "_record_ready",
        lambda summary, *_projection: ready.add(summary.release_id),
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


def _projection_inputs() -> tuple[
    tuple[_LegalUnitProjection, ...],
    tuple[_EntityProjection, ...],
    tuple[_AssertionProjection, ...],
]:
    document_id = uuid4()
    root = _LegalUnitProjection(
        uuid4(), document_id, None, "document", "law", "document:law", "a" * 64
    )
    article = _LegalUnitProjection(
        uuid4(), document_id, root.id, "article", "article-1", "article:1", "b" * 64
    )
    subject = _EntityProjection(uuid4(), "The norm", "the norm", "concept")
    object_ = _EntityProjection(uuid4(), "Controller", "controller", "actor")
    assertion = _AssertionProjection(
        id=uuid4(),
        subject_entity_id=subject.id,
        object_entity_id=object_.id,
        establishing_unit_id=article.id,
        evidence_unit_id=article.id,
        source_id=uuid4(),
        document_id=document_id,
        source_revision_id=uuid4(),
        pipeline_version="test-pipeline",
        source_title="Official source",
        establishing_locator="article-1",
        evidence_locator="article-1",
        evidence_canonical_locator="article:1",
        evidence_content_sha256="b" * 64,
        start_offset=0,
        end_offset=10,
        excerpt="Legal text",
        predicate="imposes_duty_on",
        qualifier=None,
    )
    return (root, article), (subject, object_), (assertion,)
