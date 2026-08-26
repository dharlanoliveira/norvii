from __future__ import annotations

import asyncio
from uuid import UUID

from norvii_agent.mcp.server import build_server


class ResearchRepositoryStub:
    """Provide deterministic MCP tool results without persistence I/O."""

    def list_corpora(self) -> list[dict[str, object]]:
        return [{"id": "corpus"}]

    def active_snapshot(self, corpus_id: UUID) -> str:
        return str(corpus_id)

    def list_documents(self, corpus_id: UUID) -> dict[str, object]:
        return {"outcome": "completed", "snapshot_id": str(corpus_id)}

    def search(self, corpus_id: UUID, query: str) -> dict[str, object]:
        return {"outcome": "completed", "snapshot_id": str(corpus_id), "query": query}

    def article(self, corpus_id: UUID, _document_id: UUID, locator: str) -> dict[str, object]:
        return {"outcome": "completed", "snapshot_id": str(corpus_id), "locator": locator}

    def document_metadata(self, corpus_id: UUID, _document_id: UUID) -> dict[str, object]:
        return {"outcome": "completed", "snapshot_id": str(corpus_id)}

    def related(self, corpus_id: UUID, query: str) -> dict[str, object]:
        return {"outcome": "unavailable", "snapshot_id": str(corpus_id), "query": query}

    def compare(self, corpus_id: UUID, _first_id: UUID, _second_id: UUID) -> dict[str, object]:
        return {"outcome": "completed", "snapshot_id": str(corpus_id)}


def test_server_discovers_all_tools_and_reusable_prompts() -> None:
    server = build_server(ResearchRepositoryStub())

    tool_names = asyncio.run(_tool_names(server))
    prompt_names = asyncio.run(_prompt_names(server))

    assert tool_names == {
        "compare_provisions",
        "find_related_articles",
        "get_article",
        "get_document_metadata",
        "list_corpora",
        "list_documents",
        "search_documents",
        "traverse_legal_graph",
    }
    assert prompt_names == {
        "citation_support_verification",
        "evidence_grounded_research",
        "provision_comparison",
    }


def test_invalid_tool_identifiers_are_returned_as_structured_input_errors() -> None:
    result = asyncio.run(
        build_server(ResearchRepositoryStub()).call_tool(
            "list_documents", {"corpus_id": "not-a-uuid"}
        )
    )

    assert result.structured_content == {"outcome": "invalid_input"}
    assert not result.is_error


def test_workflow_prompt_binds_the_requested_corpus_to_its_active_snapshot() -> None:
    corpus_id = "10000000-0000-4000-8000-000000000001"
    result = asyncio.run(
        build_server(ResearchRepositoryStub()).get_prompt(
            "evidence_grounded_research", {"corpus_id": corpus_id}
        )
    )

    assert result.messages[0].content.text is not None
    assert corpus_id in result.messages[0].content.text
    assert "immutable snapshot" in result.messages[0].content.text


async def _tool_names(server: object) -> set[str]:
    return {tool.name for tool in await server.list_tools()}  # type: ignore[union-attr]


async def _prompt_names(server: object) -> set[str]:
    return {prompt.name for prompt in await server.list_prompts()}  # type: ignore[union-attr]
