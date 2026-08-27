"""Service-backed Neo4j graph-release replacement coverage."""

from __future__ import annotations

import os
from uuid import UUID, uuid4

import pytest
from neo4j import Driver, GraphDatabase, RoutingControl

from norvii_ingestion.publication.persistence.config import EnvironmentConfigurationLoader
from norvii_ingestion.publication.persistence.neo4j import GraphReleaseProjection, Neo4jStore


@pytest.mark.integration
def test_replace_release_retires_only_target_snapshot_releases_and_nodes() -> None:
    """Replacement removes v1/v2 target data while retaining foreign graph releases."""
    configuration = EnvironmentConfigurationLoader(os.environ).load()
    corpus_id = uuid4()
    snapshot_id = uuid4()
    target_v1_release_id = str(uuid4())
    target_v2_release_id = str(uuid4())
    foreign_snapshot_release_id = str(uuid4())
    foreign_corpus_release_id = str(uuid4())
    replacement = _replacement_projection(corpus_id, snapshot_id)
    release_ids = [
        target_v1_release_id,
        target_v2_release_id,
        foreign_snapshot_release_id,
        foreign_corpus_release_id,
        str(replacement.release_id),
    ]
    driver = GraphDatabase.driver(
        configuration.neo4j.uri,
        auth=(configuration.neo4j.user, configuration.neo4j.password),
    )
    store = Neo4jStore(driver, configuration.neo4j.database)
    try:
        _seed_releases(
            driver,
            configuration.neo4j.database,
            releases=(
                {
                    "id": target_v1_release_id,
                    "corpus_id": str(corpus_id),
                    "snapshot_id": str(snapshot_id),
                    "build_version": "legal-assertion-graph-v1",
                },
                {
                    "id": target_v2_release_id,
                    "corpus_id": str(corpus_id),
                    "snapshot_id": str(snapshot_id),
                    "build_version": "legal-assertion-graph-v2",
                },
                {
                    "id": foreign_snapshot_release_id,
                    "corpus_id": str(corpus_id),
                    "snapshot_id": str(uuid4()),
                    "build_version": "legal-assertion-graph-v2",
                },
                {
                    "id": foreign_corpus_release_id,
                    "corpus_id": str(uuid4()),
                    "snapshot_id": str(snapshot_id),
                    "build_version": "legal-assertion-graph-v2",
                },
            ),
            nodes=(
                {"release_id": target_v1_release_id, "id": "target-v1-unit"},
                {"release_id": target_v2_release_id, "id": "target-v2-unit"},
                {"release_id": foreign_snapshot_release_id, "id": "foreign-snapshot-unit"},
                {"release_id": foreign_corpus_release_id, "id": "foreign-corpus-unit"},
            ),
        )

        store.replace_release(replacement)

        present_release_ids = _string_values(
            driver.execute_query(
                """
                MATCH (release:NorviiGraphRelease)
                WHERE release.id IN $release_ids
                RETURN collect(release.id) AS release_ids
                """,
                release_ids=release_ids,
                database_=configuration.neo4j.database,
                routing_=RoutingControl.READ,
            )
            .records[0]
            .data()["release_ids"]
        )
        node_release_ids = _string_values(
            driver.execute_query(
                """
                MATCH (node)
                WHERE node.release_id IN $release_ids
                  AND (
                    node:NorviiGraphLegalUnit
                    OR node:NorviiGraphLegalEntity
                    OR node:NorviiGraphNormativeAssertion
                  )
                RETURN collect(DISTINCT node.release_id) AS release_ids
                """,
                release_ids=release_ids,
                database_=configuration.neo4j.database,
                routing_=RoutingControl.READ,
            )
            .records[0]
            .data()["release_ids"]
        )

        assert target_v1_release_id not in present_release_ids
        assert target_v2_release_id not in present_release_ids
        assert target_v1_release_id not in node_release_ids
        assert target_v2_release_id not in node_release_ids
        assert str(replacement.release_id) in present_release_ids
        assert str(replacement.release_id) in node_release_ids
        assert foreign_snapshot_release_id in present_release_ids
        assert foreign_corpus_release_id in present_release_ids
        assert foreign_snapshot_release_id in node_release_ids
        assert foreign_corpus_release_id in node_release_ids
    finally:
        _delete_seeded_releases(driver, configuration.neo4j.database, release_ids)
        store.close()


def _seed_releases(
    driver: Driver,
    database: str,
    *,
    releases: tuple[dict[str, str], ...],
    nodes: tuple[dict[str, str], ...],
) -> None:
    driver.execute_query(
        """
        UNWIND $releases AS seed
        MERGE (release:NorviiGraphRelease {id: seed.id})
        SET release.corpus_id = seed.corpus_id,
            release.snapshot_id = seed.snapshot_id,
            release.build_version = seed.build_version,
            release.status = 'ready'
        """,
        releases=releases,
        database_=database,
        routing_=RoutingControl.WRITE,
    )
    driver.execute_query(
        """
        UNWIND $nodes AS seed
        MERGE (:NorviiGraphLegalUnit {release_id: seed.release_id, legal_unit_id: seed.id})
        """,
        nodes=nodes,
        database_=database,
        routing_=RoutingControl.WRITE,
    )


def _delete_seeded_releases(driver: Driver, database: str, release_ids: list[str]) -> None:
    driver.execute_query(
        "MATCH (node) WHERE node.release_id IN $release_ids DETACH DELETE node",
        release_ids=release_ids,
        database_=database,
        routing_=RoutingControl.WRITE,
    )
    driver.execute_query(
        "MATCH (release:NorviiGraphRelease) WHERE release.id IN $release_ids DETACH DELETE release",
        release_ids=release_ids,
        database_=database,
        routing_=RoutingControl.WRITE,
    )


def _replacement_projection(corpus_id: UUID, snapshot_id: UUID) -> GraphReleaseProjection:
    unit_id = str(uuid4())
    subject_id = str(uuid4())
    object_id = str(uuid4())
    assertion_id = str(uuid4())
    return GraphReleaseProjection(
        release_id=uuid4(),
        corpus_id=corpus_id,
        snapshot_id=snapshot_id,
        manifest_sha256="a" * 64,
        build_version="legal-assertion-graph-v2",
        legal_units=(
            {
                "id": unit_id,
                "document_id": str(uuid4()),
                "parent_id": None,
                "kind": "article",
                "locator": "article-1",
                "canonical_locator": "article:1",
                "content_sha256": "a" * 64,
            },
        ),
        entities=(
            {
                "id": subject_id,
                "label": "Subject",
                "normalized_label": "subject",
                "entity_type": "actor",
            },
            {
                "id": object_id,
                "label": "Object",
                "normalized_label": "object",
                "entity_type": "concept",
            },
        ),
        assertions=(
            {
                "id": assertion_id,
                "subject_entity_id": subject_id,
                "object_entity_id": object_id,
                "establishing_unit_id": unit_id,
                "evidence_unit_id": unit_id,
                "evidence_id": assertion_id,
                "source_id": str(uuid4()),
                "document_id": str(uuid4()),
                "source_revision_id": str(uuid4()),
                "pipeline_version": "test-pipeline",
                "source_title": "Official source",
                "establishing_locator": "article-1",
                "evidence_locator": "article-1",
                "evidence_canonical_locator": "article:1",
                "evidence_content_sha256": "a" * 64,
                "start_offset": 0,
                "end_offset": 10,
                "excerpt": "Replacement legal text.",
                "predicate": "imposes_duty_on",
                "qualifier": None,
            },
        ),
    )


def _string_values(value: object) -> set[str]:
    if not isinstance(value, list) or not all(isinstance(item, str) for item in value):
        raise TypeError("Neo4j replacement query returned invalid identifiers.")
    return set(value)
