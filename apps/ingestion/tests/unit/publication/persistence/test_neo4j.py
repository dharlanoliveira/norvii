from __future__ import annotations

from typing import TYPE_CHECKING, cast
from uuid import uuid4

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
    assert driver.arguments["parameters_"]["legal_units"] == release.legal_units
    assert driver.arguments["parameters_"]["assertions"] == release.assertions
    assert "NorviiGraphLegalUnit" in _REPLACE_RELEASE
    assert "NorviiGraphLegalEntity" in _REPLACE_RELEASE
    assert "NorviiGraphNormativeAssertion" in _REPLACE_RELEASE
    assert "[:CONTAINS" in _REPLACE_RELEASE
    assert "[:ESTABLISHES" in _REPLACE_RELEASE
    assert "[:SUBJECT" in _REPLACE_RELEASE
    assert "[:OBJECT" in _REPLACE_RELEASE
    assert "LEGAL_RELATIONSHIP" not in _REPLACE_RELEASE
