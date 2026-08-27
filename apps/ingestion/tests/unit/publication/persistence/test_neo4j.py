from __future__ import annotations

from dataclasses import dataclass
from typing import TYPE_CHECKING, Self, cast
from uuid import UUID, uuid4

from norvii_ingestion.publication.persistence.neo4j import (
    _CREATE_CONTAINS_RELATIONSHIPS,
    _DELETE_PROJECTION_NODE_BATCHES,
    _DELETE_SUPERSEDED_RELEASES,
    _MARK_RELEASE_READY,
    _SUPERSEDED_RELEASE_IDS,
    _UPSERT_LEGAL_UNITS,
    _WRITE_BATCH_SIZE,
    GraphReleaseProjection,
    Neo4jStore,
)

if TYPE_CHECKING:
    from neo4j import Driver


@dataclass(frozen=True, slots=True)
class QueryCall:
    """One Cypher statement issued inside the recorded managed transaction."""

    query: str
    parameters: dict[str, object]


class RecordingResult:
    """Expose only the Neo4j result operations used by the projection adapter."""

    def __init__(self, record: dict[str, object]) -> None:
        self._record = record

    def consume(self) -> None:
        return None

    def single(self, *, strict: bool) -> dict[str, object]:
        assert strict
        return self._record


class RecordingTransaction:
    """Record one atomic projection replacement and provide deterministic query results."""

    def __init__(
        self,
        superseded_release_ids: tuple[str, ...] = (),
        deleted_counts: dict[str, list[int]] | None = None,
    ) -> None:
        self.calls: list[QueryCall] = []
        self._superseded_release_ids = superseded_release_ids
        self._deleted_counts = deleted_counts or {}

    def run(self, query: str, parameters: dict[str, object] | None = None) -> RecordingResult:
        parameters = parameters or {}
        self.calls.append(QueryCall(query, parameters))
        if query == _SUPERSEDED_RELEASE_IDS:
            return RecordingResult({"release_ids": list(self._superseded_release_ids)})
        if query in _DELETE_PROJECTION_NODE_BATCHES:
            counts = self._deleted_counts.setdefault(query, [0])
            return RecordingResult({"deleted": counts.pop(0) if counts else 0})
        return RecordingResult({})


class RecordingSession:
    """Invoke the managed transaction callback once without a live Neo4j service."""

    def __init__(self, transaction: RecordingTransaction) -> None:
        self.transaction = transaction
        self.execute_write_count = 0

    def __enter__(self) -> Self:
        return self

    def __exit__(self, *_: object) -> None:
        return None

    def execute_write(self, callback: object, release: GraphReleaseProjection) -> None:
        self.execute_write_count += 1
        assert callable(callback)
        callback(self.transaction, release)


class RecordingDriver:
    """Expose the driver session seam used by graph-release replacement."""

    def __init__(self, transaction: RecordingTransaction | None = None) -> None:
        self.session_instance = RecordingSession(transaction or RecordingTransaction())
        self.database: str | None = None

    def session(self, *, database: str) -> RecordingSession:
        self.database = database
        return self.session_instance

    def close(self) -> None:
        return None


@dataclass(frozen=True, slots=True)
class GraphReleaseSeed:
    """Release identity used by the in-memory Cypher replacement fixture."""

    release_id: str
    corpus_id: str
    snapshot_id: str


@dataclass(frozen=True, slots=True)
class GraphNodeSeed:
    """Derived node identity used by the in-memory Cypher replacement fixture."""

    release_id: str
    node_type: str
    node_id: str


class SemanticProjectionTransaction(RecordingTransaction):
    """Apply the observable replacement lifecycle to an isolated graph fixture."""

    def __init__(
        self, releases: tuple[GraphReleaseSeed, ...], nodes: tuple[GraphNodeSeed, ...]
    ) -> None:
        self.releases = list(releases)
        self.nodes = list(nodes)
        super().__init__()

    def run(self, query: str, parameters: dict[str, object] | None = None) -> RecordingResult:
        parameters = parameters or {}
        if query == _SUPERSEDED_RELEASE_IDS:
            return self._record_superseded_release_ids(parameters)
        result = super().run(query, parameters)
        if query == _MARK_RELEASE_READY:
            self._replace_release(parameters)
        return result

    def _record_superseded_release_ids(self, parameters: dict[str, object]) -> RecordingResult:
        self.calls.append(QueryCall(_SUPERSEDED_RELEASE_IDS, parameters))
        corpus_id = _parameter_string(parameters, "corpus_id")
        snapshot_id = _parameter_string(parameters, "snapshot_id")
        release_ids = [
            release.release_id
            for release in self.releases
            if release.corpus_id == corpus_id and release.snapshot_id == snapshot_id
        ]
        return RecordingResult({"release_ids": release_ids})

    def _replace_release(self, parameters: dict[str, object]) -> None:
        corpus_id = _parameter_string(parameters, "corpus_id")
        snapshot_id = _parameter_string(parameters, "snapshot_id")
        retired_release_ids = {
            release.release_id
            for release in self.releases
            if release.corpus_id == corpus_id and release.snapshot_id == snapshot_id
        }
        self.releases = [
            release for release in self.releases if release.release_id not in retired_release_ids
        ]
        self.nodes = [node for node in self.nodes if node.release_id not in retired_release_ids]

        replacement_id = _parameter_string(parameters, "release_id")
        self.releases.append(GraphReleaseSeed(replacement_id, corpus_id, snapshot_id))
        self.nodes.extend(
            GraphNodeSeed(replacement_id, "legal_unit", _item_string(item, "id"))
            for item in _parameter_items(parameters, "legal_units")
        )
        self.nodes.extend(
            GraphNodeSeed(replacement_id, "legal_entity", _item_string(item, "id"))
            for item in _parameter_items(parameters, "entities")
        )
        self.nodes.extend(
            GraphNodeSeed(replacement_id, "normative_assertion", _item_string(item, "id"))
            for item in _parameter_items(parameters, "assertions")
        )


def test_replace_release_projects_nodes_and_fixed_assertion_edges() -> None:
    driver = RecordingDriver()
    store = Neo4jStore(cast("Driver", driver), "neo4j")
    release = _graph_release_projection(uuid4(), uuid4())

    store.replace_release(release)

    calls = driver.session_instance.transaction.calls
    assert driver.database == "neo4j"
    assert driver.session_instance.execute_write_count == 1
    assert calls[0].query == _SUPERSEDED_RELEASE_IDS
    assert calls[-1].query == _MARK_RELEASE_READY
    assert _UPSERT_LEGAL_UNITS in [call.query for call in calls]
    assert _CREATE_CONTAINS_RELATIONSHIPS in [call.query for call in calls]
    assert all(call.parameters["release_id"] == str(release.release_id) for call in calls[1:])


def test_replace_release_writes_large_projection_collections_in_bounded_batches() -> None:
    driver = RecordingDriver()
    store = Neo4jStore(cast("Driver", driver), "neo4j")
    release = _graph_release_projection(uuid4(), uuid4(), legal_unit_count=_WRITE_BATCH_SIZE + 1)

    store.replace_release(release)

    calls = driver.session_instance.transaction.calls
    unit_upserts = [call for call in calls if call.query == _UPSERT_LEGAL_UNITS]
    contains_writes = [call for call in calls if call.query == _CREATE_CONTAINS_RELATIONSHIPS]
    assert len(unit_upserts) == 2
    assert len(contains_writes) == 2
    assert [len(_parameter_items(call.parameters, "legal_units")) for call in unit_upserts] == [
        _WRITE_BATCH_SIZE,
        1,
    ]
    assert all(
        len(_parameter_items(call.parameters, "legal_units")) <= _WRITE_BATCH_SIZE
        for call in contains_writes
    )


def test_replace_release_deletes_superseded_projection_nodes_in_bounded_batches() -> None:
    deleted_counts = {query: [_WRITE_BATCH_SIZE, 1, 0] for query in _DELETE_PROJECTION_NODE_BATCHES}
    superseded_release_ids = (str(uuid4()), str(uuid4()))
    transaction = RecordingTransaction(superseded_release_ids, deleted_counts)
    driver = RecordingDriver(transaction)
    store = Neo4jStore(cast("Driver", driver), "neo4j")

    store.replace_release(_graph_release_projection(uuid4(), uuid4()))

    calls = transaction.calls
    for query in _DELETE_PROJECTION_NODE_BATCHES:
        assert len([call for call in calls if call.query == query]) == 3
    release_deletion = next(call for call in calls if call.query == _DELETE_SUPERSEDED_RELEASES)
    assert release_deletion.parameters["release_ids"] == list(superseded_release_ids)


def test_replacement_queries_stay_labelled_scoped_and_bounded() -> None:
    projection_queries = (
        _SUPERSEDED_RELEASE_IDS,
        *_DELETE_PROJECTION_NODE_BATCHES,
        _UPSERT_LEGAL_UNITS,
        _CREATE_CONTAINS_RELATIONSHIPS,
    )

    assert "corpus_id: $corpus_id, snapshot_id: $snapshot_id" in _SUPERSEDED_RELEASE_IDS
    assert "{id: release_id}" in _DELETE_SUPERSEDED_RELEASES
    assert all(":NorviiGraph" in query for query in projection_queries)
    assert all("LIMIT $batch_size" in query for query in _DELETE_PROJECTION_NODE_BATCHES)
    assert "OPTIONAL MATCH (superseded_node)" not in "\n".join(projection_queries)


def test_replace_release_retires_only_the_target_snapshot_graph() -> None:
    corpus_id = uuid4()
    snapshot_id = uuid4()
    legacy_v1_id = str(uuid4())
    legacy_v2_id = str(uuid4())
    foreign_snapshot_release_id = str(uuid4())
    foreign_corpus_release_id = str(uuid4())
    foreign_snapshot_id = str(uuid4())
    foreign_corpus_id = str(uuid4())
    transaction = SemanticProjectionTransaction(
        releases=(
            GraphReleaseSeed(legacy_v1_id, str(corpus_id), str(snapshot_id)),
            GraphReleaseSeed(legacy_v2_id, str(corpus_id), str(snapshot_id)),
            GraphReleaseSeed(foreign_snapshot_release_id, str(corpus_id), foreign_snapshot_id),
            GraphReleaseSeed(foreign_corpus_release_id, foreign_corpus_id, str(snapshot_id)),
        ),
        nodes=(
            GraphNodeSeed(legacy_v1_id, "legal_unit", "legacy-v1-unit"),
            GraphNodeSeed(legacy_v2_id, "normative_assertion", "legacy-v2-assertion"),
            GraphNodeSeed(foreign_snapshot_release_id, "legal_entity", "foreign-snapshot-entity"),
            GraphNodeSeed(foreign_corpus_release_id, "legal_unit", "foreign-corpus-unit"),
        ),
    )
    replacement = _graph_release_projection(corpus_id, snapshot_id)
    driver = RecordingDriver(transaction)
    store = Neo4jStore(cast("Driver", driver), "neo4j")

    store.replace_release(replacement)

    retained_release_ids = {release.release_id for release in transaction.releases}
    retained_node_ids = {node.node_id for node in transaction.nodes}
    assert legacy_v1_id not in retained_release_ids
    assert legacy_v2_id not in retained_release_ids
    assert "legacy-v1-unit" not in retained_node_ids
    assert "legacy-v2-assertion" not in retained_node_ids
    assert str(replacement.release_id) in retained_release_ids
    assert retained_node_ids >= {
        "replacement-unit-0",
        "replacement-subject",
        "replacement-object",
        "replacement-assertion",
    }
    assert foreign_snapshot_release_id in retained_release_ids
    assert foreign_corpus_release_id in retained_release_ids
    assert {"foreign-snapshot-entity", "foreign-corpus-unit"} <= retained_node_ids


def _graph_release_projection(
    corpus_id: UUID, snapshot_id: UUID, *, legal_unit_count: int = 2
) -> GraphReleaseProjection:
    parent_unit_id = "replacement-unit-0"
    legal_units = tuple(
        {
            "id": f"replacement-unit-{position}",
            "document_id": str(uuid4()),
            "parent_id": None if position == 0 else parent_unit_id,
            "kind": "document" if position == 0 else "article",
            "locator": f"article-{position}",
            "canonical_locator": f"article:{position}",
            "content_sha256": "b" * 64,
        }
        for position in range(legal_unit_count)
    )
    return GraphReleaseProjection(
        release_id=uuid4(),
        corpus_id=corpus_id,
        snapshot_id=snapshot_id,
        manifest_sha256="b" * 64,
        build_version="legal-assertion-graph-v2",
        legal_units=legal_units,
        entities=(
            {
                "id": "replacement-subject",
                "label": "Subject",
                "normalized_label": "subject",
                "entity_type": "actor",
            },
            {
                "id": "replacement-object",
                "label": "Object",
                "normalized_label": "object",
                "entity_type": "concept",
            },
        ),
        assertions=(
            {
                "id": "replacement-assertion",
                "subject_entity_id": "replacement-subject",
                "object_entity_id": "replacement-object",
                "establishing_unit_id": parent_unit_id,
                "evidence_unit_id": parent_unit_id,
                "evidence_id": "replacement-assertion",
                "source_id": str(uuid4()),
                "document_id": str(uuid4()),
                "source_revision_id": str(uuid4()),
                "pipeline_version": "test-pipeline",
                "source_title": "Official source",
                "establishing_locator": "article-1",
                "evidence_locator": "article-1",
                "evidence_canonical_locator": "article:1",
                "evidence_content_sha256": "b" * 64,
                "start_offset": 0,
                "end_offset": 10,
                "excerpt": "Replacement legal text",
                "predicate": "imposes_duty_on",
                "qualifier": None,
            },
        ),
    )


def _parameter_items(parameters: dict[str, object], name: str) -> tuple[dict[str, object], ...]:
    value = parameters[name]
    if not isinstance(value, (list, tuple)) or not all(isinstance(item, dict) for item in value):
        raise TypeError(f"{name} must contain graph projection dictionaries.")
    return tuple(value)


def _parameter_string(parameters: dict[str, object], name: str) -> str:
    value = parameters[name]
    if not isinstance(value, str):
        raise TypeError(f"{name} must be a string.")
    return value


def _item_string(item: dict[str, object], name: str) -> str:
    value = item[name]
    if not isinstance(value, str):
        raise TypeError(f"{name} must be a string.")
    return value
