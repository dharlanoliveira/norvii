from __future__ import annotations

from dataclasses import dataclass
from typing import TYPE_CHECKING, cast
from uuid import UUID, uuid4

from norvii_ingestion.publication.persistence.neo4j import (
    _REPLACE_RELEASE,
    GraphReleaseProjection,
    Neo4jStore,
)

if TYPE_CHECKING:
    from neo4j import Driver


class RecordingDriver:
    def __init__(self) -> None:
        self.query: str | None = None
        self.arguments: dict[str, object] | None = None

    def execute_query(self, query: str, **arguments: object) -> None:
        self.query = query
        self.arguments = arguments

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


class SemanticProjectionDriver:
    """Execute the replacement contract against an isolated in-memory graph fixture."""

    def __init__(
        self, releases: tuple[GraphReleaseSeed, ...], nodes: tuple[GraphNodeSeed, ...]
    ) -> None:
        self.releases = list(releases)
        self.nodes = list(nodes)
        self.queries: list[str] = []

    def execute_query(self, query: str, **arguments: object) -> None:
        self.queries.append(query)
        assert query == _REPLACE_RELEASE
        parameters = arguments["parameters_"]
        assert isinstance(parameters, dict)
        self._replace_release(parameters)

    def close(self) -> None:
        return None

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
    release_id = uuid4()
    parent_unit_id = uuid4()
    child_unit_id = uuid4()
    subject_id = uuid4()
    object_id = uuid4()
    assertion_id = uuid4()
    driver = RecordingDriver()
    store = Neo4jStore(cast("Driver", driver), "neo4j")
    release = GraphReleaseProjection(
        release_id=release_id,
        corpus_id=uuid4(),
        snapshot_id=uuid4(),
        manifest_sha256="a" * 64,
        build_version="legal-assertion-graph-v2",
        legal_units=(
            {
                "id": str(parent_unit_id),
                "document_id": str(uuid4()),
                "parent_id": None,
                "kind": "document",
                "locator": "law",
            },
            {
                "id": str(child_unit_id),
                "document_id": str(uuid4()),
                "parent_id": str(parent_unit_id),
                "kind": "article",
                "locator": "article-1",
            },
        ),
        entities=(
            {
                "id": str(subject_id),
                "label": "Norm",
                "normalized_label": "norm",
                "entity_type": "concept",
            },
            {
                "id": str(object_id),
                "label": "Controller",
                "normalized_label": "controller",
                "entity_type": "actor",
            },
        ),
        assertions=(
            {
                "id": str(assertion_id),
                "subject_entity_id": str(subject_id),
                "object_entity_id": str(object_id),
                "establishing_unit_id": str(child_unit_id),
                "evidence_unit_id": str(child_unit_id),
                "evidence_id": str(assertion_id),
                "source_id": str(uuid4()),
                "document_id": str(uuid4()),
                "source_revision_id": str(uuid4()),
                "pipeline_version": "test-pipeline",
                "source_title": "Official source",
                "establishing_locator": "article-1",
                "evidence_locator": "article-1",
                "start_offset": 0,
                "end_offset": 10,
                "excerpt": "Legal text",
                "predicate": "imposes_duty_on",
                "qualifier": None,
            },
        ),
    )

    store.replace_release(release)

    assert driver.query == _REPLACE_RELEASE
    assert driver.arguments is not None
    query_parameters = driver.arguments["parameters_"]
    assert isinstance(query_parameters, dict)
    assert query_parameters["build_version"] == "legal-assertion-graph-v2"
    assert query_parameters["legal_units"] == release.legal_units
    assert query_parameters["assertions"] == release.assertions
    assert "NorviiGraphLegalUnit" in _REPLACE_RELEASE
    assert "NorviiGraphLegalEntity" in _REPLACE_RELEASE
    assert "NorviiGraphNormativeAssertion" in _REPLACE_RELEASE
    assert "[:CONTAINS" in _REPLACE_RELEASE
    assert "[:ESTABLISHES" in _REPLACE_RELEASE
    assert "[:SUBJECT" in _REPLACE_RELEASE
    assert "[:OBJECT" in _REPLACE_RELEASE
    assert "superseded_release_ids" in _REPLACE_RELEASE
    assert "corpus_id: $corpus_id, snapshot_id: $snapshot_id" in _REPLACE_RELEASE
    assert "release.build_version = build_version" in _REPLACE_RELEASE
    assert "LEGAL_RELATIONSHIP" not in _REPLACE_RELEASE


def test_replace_release_retires_only_the_target_snapshot_graph() -> None:
    corpus_id = uuid4()
    snapshot_id = uuid4()
    legacy_v1_id = str(uuid4())
    legacy_v2_id = str(uuid4())
    foreign_snapshot_release_id = str(uuid4())
    foreign_corpus_release_id = str(uuid4())
    foreign_snapshot_id = str(uuid4())
    foreign_corpus_id = str(uuid4())
    driver = SemanticProjectionDriver(
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
    store = Neo4jStore(cast("Driver", driver), "neo4j")

    store.replace_release(replacement)

    retained_release_ids = {release.release_id for release in driver.releases}
    retained_node_ids = {node.node_id for node in driver.nodes}
    assert legacy_v1_id not in retained_release_ids
    assert legacy_v2_id not in retained_release_ids
    assert "legacy-v1-unit" not in retained_node_ids
    assert "legacy-v2-assertion" not in retained_node_ids
    assert str(replacement.release_id) in retained_release_ids
    assert retained_node_ids >= {
        "replacement-unit",
        "replacement-subject",
        "replacement-object",
        "replacement-assertion",
    }
    assert foreign_snapshot_release_id in retained_release_ids
    assert foreign_corpus_release_id in retained_release_ids
    assert {"foreign-snapshot-entity", "foreign-corpus-unit"} <= retained_node_ids
    assert driver.queries == [_REPLACE_RELEASE]


def _graph_release_projection(corpus_id: UUID, snapshot_id: UUID) -> GraphReleaseProjection:
    return GraphReleaseProjection(
        release_id=uuid4(),
        corpus_id=corpus_id,
        snapshot_id=snapshot_id,
        manifest_sha256="b" * 64,
        build_version="legal-assertion-graph-v2",
        legal_units=(
            {
                "id": "replacement-unit",
                "document_id": str(uuid4()),
                "parent_id": None,
                "kind": "article",
                "locator": "article-1",
            },
        ),
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
                "establishing_unit_id": "replacement-unit",
                "evidence_unit_id": "replacement-unit",
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
    if not isinstance(value, tuple) or not all(isinstance(item, dict) for item in value):
        raise TypeError(f"{name} must contain graph projection dictionaries.")
    return value


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
