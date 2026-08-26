from __future__ import annotations

from dataclasses import dataclass, field
from uuid import UUID

from norvii_agent.config import AgentConfig
from norvii_agent.mcp.research import MCPResearchRepository

_CORPUS_ID = UUID("10000000-0000-4000-8000-000000000001")
_DOCUMENT_ID = UUID("30000000-0000-4000-8000-000000000001")


@dataclass(frozen=True, slots=True)
class RecordingRepository(MCPResearchRepository):
    """Return scripted query results while exposing requested query boundaries."""

    responses: list[list[dict[str, object]]]
    calls: list[tuple[str, tuple[object, ...]]] = field(default_factory=list)

    def _query(
        self, statement: str, parameters: tuple[object, ...] = ()
    ) -> list[dict[str, object]]:
        self.calls.append((statement, parameters))
        return self.responses.pop(0)


def test_document_listing_binds_to_the_resolved_snapshot() -> None:
    repository = RecordingRepository(
        _configuration(), [[{"snapshot_id": "50000000-0000-4000-8000-000000000001"}], []]
    )

    result = repository.list_documents(_CORPUS_ID)

    assert result == {
        "outcome": "completed",
        "snapshot_id": "50000000-0000-4000-8000-000000000001",
        "documents": [],
    }
    assert repository.calls[1][1] == (
        _CORPUS_ID,
        "50000000-0000-4000-8000-000000000001",
    )


def test_empty_search_is_rejected_without_querying_corpus_content() -> None:
    repository = RecordingRepository(
        _configuration(), [[{"snapshot_id": "50000000-0000-4000-8000-000000000001"}]]
    )

    result = repository.search(_CORPUS_ID, " ")

    assert result == {"outcome": "invalid_input"}
    assert len(repository.calls) == 1


def test_article_not_found_never_substitutes_another_document() -> None:
    repository = RecordingRepository(
        _configuration(), [[{"snapshot_id": "50000000-0000-4000-8000-000000000001"}], []]
    )

    result = repository.article(_CORPUS_ID, _DOCUMENT_ID, "article-1")

    assert result == {"outcome": "not_found"}
    assert repository.calls[1][1] == (
        _CORPUS_ID,
        "50000000-0000-4000-8000-000000000001",
        _DOCUMENT_ID,
        "article-1",
    )


def test_ready_graph_with_no_matching_assertions_is_a_completed_empty_result() -> None:
    repository = RecordingRepository(
        _configuration(),
        [
            [{"snapshot_id": "50000000-0000-4000-8000-000000000001"}],
            [{"?column?": 1}],
            [],
        ],
    )

    result = repository.related(_CORPUS_ID, "nonexistent entity")

    assert result == {
        "outcome": "completed",
        "snapshot_id": "50000000-0000-4000-8000-000000000001",
        "assertions": [],
    }
    assert len(repository.calls) == 3
    assert "normative_assertions" in repository.calls[2][0]
    assert "graph_release_assertions" in repository.calls[2][0]
    assert "semantic_relationships" not in repository.calls[2][0]


def test_related_reports_unavailable_when_no_graph_release_is_ready() -> None:
    repository = RecordingRepository(
        _configuration(),
        [
            [{"snapshot_id": "50000000-0000-4000-8000-000000000001"}],
            [],
        ],
    )

    result = repository.related(_CORPUS_ID, "entity")

    assert result == {"outcome": "unavailable"}
    assert len(repository.calls) == 2


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
        chat_timeout_seconds=30.0,
    )
