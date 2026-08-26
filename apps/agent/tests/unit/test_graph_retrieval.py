from __future__ import annotations

import logging
from types import SimpleNamespace
from uuid import UUID

import pytest

from norvii_agent.config import AgentConfig
from norvii_agent.retrieval.graph import (
    _PLANNED_GRAPH_SEARCH,
    GraphRetrievalUnavailableError,
    Neo4jGraphRetriever,
    _assertion_path,
)
from norvii_agent.retrieval.planning import GraphPredicateCapability, GraphRetrievalPlan


class FakeRecord:
    def __init__(self, values: dict[str, object]) -> None:
        self._values = values

    def data(self) -> dict[str, object]:
        return self._values


class FakeDriver:
    def __init__(self, entity_labels: list[str] | None = None) -> None:
        self.calls: list[tuple[str, dict[str, object]]] = []
        self._entity_labels = entity_labels or ["national authority", "data subject"]

    def execute_query(self, query: str, **kwargs: object) -> SimpleNamespace:
        self.calls.append((query, kwargs))
        if "predicate_capabilities" in query:
            return SimpleNamespace(
                records=[
                    FakeRecord(
                        {
                            "entity_types": ["authority", "right"],
                            "predicates": ["assigns_responsibility_to", "applies_to"],
                            "entity_labels": self._entity_labels,
                            "predicate_capabilities": [
                                {
                                    "predicate": "assigns_responsibility_to",
                                    "entity_labels": ["national authority"],
                                },
                                {
                                    "predicate": "applies_to",
                                    "entity_labels": ["data subject"],
                                },
                            ],
                            "scope_locators": ["chapter-1", "article-2"],
                        }
                    )
                ]
            )
        return SimpleNamespace(records=[FakeRecord(_assertion_row())])


def test_graph_capabilities_are_snapshot_scoped_and_content_free() -> None:
    driver = FakeDriver()
    retriever = Neo4jGraphRetriever(_configuration(), driver=driver)  # type: ignore[arg-type]

    catalog = retriever.capabilities(_corpus_id(), _snapshot_id())

    assert catalog is not None
    assert catalog.entity_types == ("authority", "right")
    assert catalog.predicates == ("assigns_responsibility_to", "applies_to")
    assert catalog.entity_labels == ("national authority", "data subject")
    assert catalog.predicate_capabilities == (
        GraphPredicateCapability("assigns_responsibility_to", ("national authority",)),
        GraphPredicateCapability("applies_to", ("data subject",)),
    )
    assert catalog.scope_locators == ("chapter-1", "article-2")
    _, parameters = driver.calls[0]
    assert parameters["corpus_id"] == str(_corpus_id())
    assert parameters["snapshot_id"] == str(_snapshot_id())


def test_graph_capabilities_keep_the_bounded_canonical_catalogs() -> None:
    entity_labels = [f"entity-{index}" for index in range(127)] + ["national authority"]
    retriever = Neo4jGraphRetriever(_configuration(), driver=FakeDriver(entity_labels))  # type: ignore[arg-type]

    catalog = retriever.capabilities(_corpus_id(), _snapshot_id())

    assert catalog is not None
    assert len(catalog.entity_labels) == 128
    assert catalog.entity_labels[-1] == "national authority"


def test_graph_plan_logs_parameterized_assertion_query_and_provenance(
    caplog: pytest.LogCaptureFixture,
) -> None:
    driver = FakeDriver()
    retriever = Neo4jGraphRetriever(_configuration(), driver=driver)  # type: ignore[arg-type]
    plan = GraphRetrievalPlan(
        use_graph=True,
        predicates=("assigns_responsibility_to",),
        entity_labels=("national authority",),
        scope_locator="chapter-1",
    )

    with caplog.at_level(logging.INFO, logger="norvii_agent.retrieval.graph"):
        evidence = retriever.search_plan(_corpus_id(), _snapshot_id(), plan)

    assert [item.unit_locator for item in evidence] == ["article-2-item-1"]
    _, parameters = driver.calls[0]
    query_parameters = parameters["parameters_"]
    assert isinstance(query_parameters, dict)
    assert query_parameters["predicates"] == ["assigns_responsibility_to"]
    assert query_parameters["entity_labels"] == ["national authority"]
    assert query_parameters["scope_locator"] == "chapter-1"
    assert "MATCH (release:NorviiGraphRelease" in caplog.text
    assert '"predicates": ["assigns_responsibility_to"]' in caplog.text
    assert '"scope_locator": "chapter-1"' in caplog.text
    assert "Planned assertion Cypher returned 1 evidence locations" in caplog.text
    assert retriever.last_scope_locator == "chapter-1"
    assert retriever.last_assertion_path[0].assertion_id == "assertion-1"
    assert retriever.last_assertion_path[0].establishing_locator == "article-2"
    assert retriever.last_assertion_path[0].evidence_locator == "article-2-item-1"
    assert retriever.last_assertion_path[0].hierarchy_context == ("chapter-1", "article-2")


def test_planned_query_is_bounded_to_descendants_and_active_release() -> None:
    assert "[:CONTAINS*0..6]" in _planned_query()
    assert "($scope_locator IS NULL OR establishing IN scoped_units)" in _planned_query()
    assert "MATCH (release)<-[:IN_GRAPH_RELEASE]-(assertion" in _planned_query()
    assert "LIMIT 8" in _planned_query()


def test_assertion_path_rejects_missing_hierarchy_provenance() -> None:
    row = _assertion_row()
    row["hierarchy_context"] = []

    with pytest.raises(GraphRetrievalUnavailableError, match="hierarchy context is invalid"):
        _assertion_path(row)


def _planned_query() -> str:
    return _PLANNED_GRAPH_SEARCH


def _configuration() -> AgentConfig:
    return AgentConfig(
        host="127.0.0.1",
        port=8090,
        postgres_host="localhost",
        postgres_port=5432,
        postgres_database="norvii",
        postgres_user="norvii",
        postgres_password="",
        chat_base_url="",
        chat_api_key="",
        chat_model="model",
        chat_reasoning_effort="medium",
        chat_timeout_seconds=1,
    )


def _assertion_row() -> dict[str, object]:
    return {
        "assertion_id": "assertion-1",
        "evidence_id": "evidence-1",
        "source_id": "20000000-0000-4000-8000-000000000001",
        "document_id": "30000000-0000-4000-8000-000000000001",
        "source_revision_id": "40000000-0000-4000-8000-000000000001",
        "pipeline_version": "corpus-ingestion-v2",
        "source_title": "Official text",
        "evidence_locator": "article-2-item-1",
        "start_offset": 0,
        "end_offset": 10,
        "excerpt": "Evidence excerpt.",
        "predicate": "assigns_responsibility_to",
        "qualifier": "when applicable",
        "subject_label": "National authority",
        "object_label": "Data controller",
        "establishing_locator": "article-2",
        "hierarchy_context": ["chapter-1", "article-2"],
    }


def _corpus_id() -> UUID:
    return UUID("10000000-0000-4000-8000-000000000001")


def _snapshot_id() -> UUID:
    return UUID("50000000-0000-4000-8000-000000000001")
