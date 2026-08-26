"""MCP server composition for bounded Norvii legal research."""

from __future__ import annotations

import json
import logging
from collections.abc import Callable
from dataclasses import dataclass
from time import perf_counter
from typing import TYPE_CHECKING
from uuid import UUID

import psycopg
from mcp.server import MCPServer

if TYPE_CHECKING:
    from norvii_agent.mcp.research import MCPResearchRepository

_LOGGER = logging.getLogger(__name__)
_ToolOperation = Callable[[], dict[str, object]]


@dataclass(frozen=True, slots=True)
class InvocationMeasurement:
    """Content-safe fields recorded for one MCP operation."""

    kind: str
    name: str
    corpus_id: str | None
    snapshot_id: str | None
    strategy: str
    started_at: float
    result_items: int
    outcome: str


def build_server(repository: MCPResearchRepository) -> MCPServer:
    """Build the discoverable read-only Norvii MCP server."""
    server = MCPServer(
        "Norvii Research",
        version="1.0.0",
        instructions=(
            "Use only returned corpus evidence. This is a technical demonstration, "
            "not legal advice."
        ),
    )
    _register_catalog_tools(server, repository)
    _register_graph_tools(server, repository)
    _register_prompts(server, repository)
    return server


def _register_catalog_tools(server: MCPServer, repository: MCPResearchRepository) -> None:
    """Register corpus, document, and lexical evidence operations."""

    @server.tool(description="List enabled legal corpora.")
    def list_corpora() -> dict[str, object]:
        return _measure_tool(
            "list_corpora",
            None,
            "catalog",
            lambda: {"outcome": "completed", "corpora": repository.list_corpora()},
        )

    @server.tool(description="List documents in a corpus active published snapshot.")
    def list_documents(corpus_id: str) -> dict[str, object]:
        parsed_corpus_id = _uuid_or_none(corpus_id)
        return _measure_tool(
            "list_documents",
            corpus_id,
            "catalog",
            lambda: (
                repository.list_documents(parsed_corpus_id)
                if parsed_corpus_id is not None
                else _invalid_input()
            ),
        )

    @server.tool(description="Find up to eight evidence-backed legal locations in a corpus.")
    def search_documents(corpus_id: str, query: str) -> dict[str, object]:
        parsed_corpus_id = _uuid_or_none(corpus_id)
        return _measure_tool(
            "search_documents",
            corpus_id,
            "lexical",
            lambda: (
                repository.search(parsed_corpus_id, query)
                if parsed_corpus_id is not None
                else _invalid_input()
            ),
        )

    @server.tool(
        description="Return an exact immutable legal unit by document version and locator."
    )
    def get_article(
        corpus_id: str, document_version_id: str, unit_locator: str
    ) -> dict[str, object]:
        parsed_corpus_id = _uuid_or_none(corpus_id)
        parsed_document_id = _uuid_or_none(document_version_id)
        return _measure_tool(
            "get_article",
            corpus_id,
            "catalog",
            lambda: (
                repository.article(parsed_corpus_id, parsed_document_id, unit_locator)
                if parsed_corpus_id is not None and parsed_document_id is not None
                else _invalid_input()
            ),
        )

    @server.tool(description="Return metadata for one immutable document version.")
    def get_document_metadata(corpus_id: str, document_version_id: str) -> dict[str, object]:
        parsed_corpus_id = _uuid_or_none(corpus_id)
        parsed_document_id = _uuid_or_none(document_version_id)
        return _measure_tool(
            "get_document_metadata",
            corpus_id,
            "catalog",
            lambda: (
                repository.document_metadata(parsed_corpus_id, parsed_document_id)
                if parsed_corpus_id is not None and parsed_document_id is not None
                else _invalid_input()
            ),
        )


def _register_graph_tools(server: MCPServer, repository: MCPResearchRepository) -> None:
    """Register bounded relationship and comparison operations."""

    @server.tool(description="Find evidence-backed graph relationships matching a query.")
    def find_related_articles(corpus_id: str, query: str) -> dict[str, object]:
        parsed_corpus_id = _uuid_or_none(corpus_id)
        return _measure_tool(
            "find_related_articles",
            corpus_id,
            "graph",
            lambda: (
                repository.related(parsed_corpus_id, query)
                if parsed_corpus_id is not None
                else _invalid_input()
            ),
        )

    @server.tool(description="Traverse bounded, evidence-backed legal graph relationships.")
    def traverse_legal_graph(corpus_id: str, query: str) -> dict[str, object]:
        parsed_corpus_id = _uuid_or_none(corpus_id)
        return _measure_tool(
            "traverse_legal_graph",
            corpus_id,
            "graph",
            lambda: (
                repository.related(parsed_corpus_id, query)
                if parsed_corpus_id is not None
                else _invalid_input()
            ),
        )

    @server.tool(description="Return two cited provisions without generating a legal conclusion.")
    def compare_provisions(
        corpus_id: str, first_document_version_id: str, second_document_version_id: str
    ) -> dict[str, object]:
        parsed_corpus_id = _uuid_or_none(corpus_id)
        parsed_first_id = _uuid_or_none(first_document_version_id)
        parsed_second_id = _uuid_or_none(second_document_version_id)
        return _measure_tool(
            "compare_provisions",
            corpus_id,
            "comparison",
            lambda: (
                repository.compare(parsed_corpus_id, parsed_first_id, parsed_second_id)
                if parsed_corpus_id is not None
                and parsed_first_id is not None
                and parsed_second_id is not None
                else _invalid_input()
            ),
        )


def _register_prompts(server: MCPServer, repository: MCPResearchRepository) -> None:
    """Register reusable evidence-grounded research workflows."""

    @server.prompt(description="Guide evidence-grounded legal research using Norvii tools.")
    def evidence_grounded_research(corpus_id: str) -> str:
        return _workflow_prompt(
            repository,
            "evidence_grounded_research",
            corpus_id,
            "Search the corpus, cite each material claim, and abstain if evidence is missing.",
        )

    @server.prompt(description="Guide a cited comparison of two legal provisions.")
    def provision_comparison(corpus_id: str) -> str:
        return _workflow_prompt(
            repository,
            "provision_comparison",
            corpus_id,
            "Retrieve both provisions and distinguish source text from interpretation.",
        )

    @server.prompt(description="Guide verification that claims have immutable citation support.")
    def citation_support_verification(corpus_id: str) -> str:
        return _workflow_prompt(
            repository,
            "citation_support_verification",
            corpus_id,
            "Verify every material claim has an immutable evidence reference; identify gaps.",
        )


def _uuid_or_none(value: str) -> UUID | None:
    try:
        return UUID(value)
    except ValueError:
        return None


def _invalid_input() -> dict[str, object]:
    return {"outcome": "invalid_input"}


def _measure_tool(
    name: str, corpus_id: str | None, strategy: str, operation: _ToolOperation
) -> dict[str, object]:
    started_at = perf_counter()
    try:
        result = operation()
    except psycopg.Error:
        result = {"outcome": "unavailable"}
    _record_invocation(
        InvocationMeasurement(
            kind="tool",
            name=name,
            corpus_id=_valid_corpus_id(corpus_id),
            snapshot_id=_string_or_none(result.get("snapshot_id")),
            strategy=strategy,
            started_at=started_at,
            result_items=_result_usage(result),
            outcome=_string_or_none(result.get("outcome")) or "unavailable",
        )
    )
    return result


def _workflow_prompt(
    repository: MCPResearchRepository, name: str, corpus_id: str, instruction: str
) -> str:
    started_at = perf_counter()
    parsed_corpus_id = _uuid_or_none(corpus_id)
    snapshot_id: str | None = None
    if parsed_corpus_id is not None:
        try:
            snapshot_id = repository.active_snapshot(parsed_corpus_id)
        except psycopg.Error:
            snapshot_id = None
    outcome = "completed" if snapshot_id is not None else "unavailable"
    _record_invocation(
        InvocationMeasurement(
            kind="skill",
            name=name,
            corpus_id=_valid_corpus_id(corpus_id),
            snapshot_id=snapshot_id,
            strategy="workflow",
            started_at=started_at,
            result_items=0,
            outcome=outcome,
        )
    )
    if snapshot_id is None:
        return (
            "No active published snapshot is available for the selected corpus. "
            "Abstain and report the unavailable evidence boundary."
        )
    return _prompt(corpus_id, snapshot_id, instruction)


def _prompt(corpus_id: str, snapshot_id: str, instruction: str) -> str:
    return (
        f"Research only corpus {corpus_id} at immutable snapshot {snapshot_id}. {instruction} "
        "This is a technical demonstration and not legal advice."
    )


def _record_invocation(measurement: InvocationMeasurement) -> None:
    """Write content-safe invocation measurements to stderr through logging."""
    _LOGGER.info(
        "%s",
        json.dumps(
            {
                "event": "mcp_invocation",
                "kind": measurement.kind,
                "name": measurement.name,
                "corpus_id": measurement.corpus_id,
                "snapshot_id": measurement.snapshot_id,
                "strategy": measurement.strategy,
                "latency_ms": round((perf_counter() - measurement.started_at) * 1000, 3),
                "result_items": measurement.result_items,
                "outcome": measurement.outcome,
            },
            sort_keys=True,
        ),
    )


def _valid_corpus_id(value: str | None) -> str | None:
    parsed_value = _uuid_or_none(value) if value is not None else None
    return str(parsed_value) if parsed_value is not None else None


def _result_usage(result: dict[str, object]) -> int:
    for key in ("corpora", "documents", "evidence", "relationships"):
        value = result.get(key)
        if isinstance(value, list):
            return len(value)
    return sum(1 for key in ("document", "article", "first", "second") if key in result)


def _string_or_none(value: object | None) -> str | None:
    return value if isinstance(value, str) else None
