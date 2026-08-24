from __future__ import annotations

from uuid import UUID

import pytest

from norvii_agent.graph import Evidence, RetrievalInspection
from norvii_agent.retrieval import HybridRetriever, StrategyRetriever


class FixedRetriever:
    def __init__(self, evidence: tuple[Evidence, ...], strategy: str) -> None:
        self._evidence = evidence
        self._strategy = strategy
        self.last_retrieval = RetrievalInspection(strategy, 8, len(evidence), "embedding")
        self.last_graph_path = ()

    def search(
        self, corpus_id: UUID, snapshot_id: UUID, question: str, strategy: str = "vector"
    ) -> tuple[Evidence, ...]:
        assert corpus_id.int != 0
        assert snapshot_id.int != 0
        assert question
        assert strategy == self._strategy
        return self._evidence


def test_hybrid_retrieval_deduplicates_shared_immutable_locations() -> None:
    graph_evidence = _evidence("graph", "article-1", 1)
    vector_evidence = _evidence("vector", "article-1", 1)
    hybrid = HybridRetriever(
        FixedRetriever((vector_evidence,), "vector"),
        FixedRetriever((graph_evidence,), "graph"),
    )

    result = hybrid.search(_corpus_id(), _snapshot_id(), "Which right applies?")

    assert len(result) == 1
    assert result[0].id == "graph"
    assert result[0].rank == 1
    assert hybrid.last_retrieval == RetrievalInspection("hybrid", 8, 1, "embedding")


def test_strategy_retriever_rejects_an_undeclared_strategy() -> None:
    vector = FixedRetriever((), "vector")
    strategies = StrategyRetriever(vector, FixedRetriever((), "graph"), vector)

    with pytest.raises(ValueError, match="retrieval strategy is unsupported"):
        strategies.search(_corpus_id(), _snapshot_id(), "Question", "unknown")


def _evidence(identifier: str, locator: str, rank: int) -> Evidence:
    return Evidence(
        identifier,
        _corpus_id(),
        UUID("20000000-0000-4000-8000-000000000001"),
        UUID("30000000-0000-4000-8000-000000000001"),
        locator,
        0,
        10,
        "Evidence excerpt.",
        rank,
        snapshot_id=_snapshot_id(),
    )


def _corpus_id() -> UUID:
    return UUID("10000000-0000-4000-8000-000000000001")


def _snapshot_id() -> UUID:
    return UUID("50000000-0000-4000-8000-000000000001")
