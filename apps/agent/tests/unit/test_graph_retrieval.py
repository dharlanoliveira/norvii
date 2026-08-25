from __future__ import annotations

from types import SimpleNamespace
from uuid import UUID

from norvii_agent.config import AgentConfig
from norvii_agent.retrieval.graph import Neo4jGraphRetriever
from norvii_agent.retrieval.planning import GraphRetrievalPlan


class FakeRecord:
    def __init__(self, values: dict[str, object]) -> None:
        self._values = values

    def data(self) -> dict[str, object]:
        return self._values


class FakeDriver:
    def __init__(self, entity_labels: list[str] | None = None) -> None:
        self.calls: list[tuple[str, dict[str, object]]] = []
        self._entity_labels = entity_labels or ["autoridade nacional", "titular"]

    def execute_query(self, query: str, **kwargs: object) -> SimpleNamespace:
        self.calls.append((query, kwargs))
        if "entity_types" in query:
            return SimpleNamespace(
                records=[
                    FakeRecord(
                        {
                            "entity_types": ["authority", "right"],
                            "relationship_types": ["governs", "applies_to"],
                            "entity_labels": self._entity_labels,
                        }
                    )
                ]
            )
        return SimpleNamespace(records=[FakeRecord(_evidence_row())])


def test_graph_capabilities_are_snapshot_scoped_and_content_free() -> None:
    driver = FakeDriver()
    retriever = Neo4jGraphRetriever(_configuration(), driver=driver)  # type: ignore[arg-type]

    catalog = retriever.capabilities(_corpus_id(), _snapshot_id())

    assert catalog is not None
    assert catalog.entity_types == ("authority", "right")
    assert catalog.relationship_types == ("governs", "applies_to")
    assert catalog.entity_labels == ("autoridade nacional", "titular")
    _, parameters = driver.calls[0]
    assert parameters["corpus_id"] == str(_corpus_id())
    assert parameters["snapshot_id"] == str(_snapshot_id())


def test_graph_capabilities_keep_the_bounded_canonical_entity_catalog() -> None:
    entity_labels = [f"entity-{index}" for index in range(127)] + ["autoridade nacional"]
    retriever = Neo4jGraphRetriever(_configuration(), driver=FakeDriver(entity_labels))  # type: ignore[arg-type]

    catalog = retriever.capabilities(_corpus_id(), _snapshot_id())

    assert catalog is not None
    assert len(catalog.entity_labels) == 128
    assert catalog.entity_labels[-1] == "autoridade nacional"


def test_graph_plan_uses_parameterized_relationships_and_canonical_entity_labels() -> None:
    driver = FakeDriver()
    retriever = Neo4jGraphRetriever(_configuration(), driver=driver)  # type: ignore[arg-type]
    plan = GraphRetrievalPlan(
        use_graph=True,
        relationship_types=("governs",),
        entity_labels=("autoridade nacional",),
    )

    evidence = retriever.search_plan(_corpus_id(), _snapshot_id(), plan)

    assert [item.unit_locator for item in evidence] == ["article-1"]
    _, parameters = driver.calls[0]
    assert parameters["relationship_types"] == ["governs"]
    assert parameters["entity_labels"] == ["autoridade nacional"]


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


def _evidence_row() -> dict[str, object]:
    return {
        "evidence_id": "evidence-1",
        "source_id": "20000000-0000-4000-8000-000000000001",
        "document_id": "30000000-0000-4000-8000-000000000001",
        "source_revision_id": "40000000-0000-4000-8000-000000000001",
        "pipeline_version": "corpus-ingestion-v1",
        "source_title": "Official text",
        "evidence_locator": "article-1",
        "start_offset": 0,
        "end_offset": 10,
        "excerpt": "Evidence excerpt.",
        "relationship_type": "governs",
        "subject_label": "Authority",
        "object_label": "Data processing",
    }


def _corpus_id() -> UUID:
    return UUID("10000000-0000-4000-8000-000000000001")


def _snapshot_id() -> UUID:
    return UUID("50000000-0000-4000-8000-000000000001")
