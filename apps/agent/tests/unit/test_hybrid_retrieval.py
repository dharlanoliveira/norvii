from __future__ import annotations

import logging
from uuid import UUID

import pytest

from norvii_agent.graph import Evidence, RetrievalInspection, StrategyUnavailableError
from norvii_agent.retrieval import HybridRetriever, StrategyRetriever
from norvii_agent.retrieval.planning import (
    GraphCapabilityCatalog,
    GraphRetrievalPlan,
)


class FixedVectorRetriever:
    def __init__(self, evidence: tuple[Evidence, ...]) -> None:
        self._evidence = evidence
        self.last_retrieval = RetrievalInspection("vector", 8, len(evidence), "embedding")

    def search(
        self, corpus_id: UUID, snapshot_id: UUID, question: str, strategy: str = "vector"
    ) -> tuple[Evidence, ...]:
        assert corpus_id.int != 0
        assert snapshot_id.int != 0
        assert question
        assert strategy == "vector"
        return self._evidence


class FixedGraphRetriever:
    def __init__(
        self,
        catalog: GraphCapabilityCatalog | None,
        evidence: tuple[Evidence, ...] = (),
        *,
        unavailable: bool = False,
    ) -> None:
        self._catalog = catalog
        self._evidence = evidence
        self._unavailable = unavailable
        self.last_assertion_path = ()
        self.last_scope_locator = None

    def capabilities(self, _corpus_id: UUID, _snapshot_id: UUID) -> GraphCapabilityCatalog | None:
        if self._unavailable:
            raise StrategyUnavailableError("graph unavailable")
        return self._catalog

    def search_plan(
        self, _corpus_id: UUID, _snapshot_id: UUID, plan: GraphRetrievalPlan
    ) -> tuple[Evidence, ...]:
        assert plan.use_graph
        return self._evidence


class FixedPlanner:
    def __init__(self, plan: GraphRetrievalPlan) -> None:
        self._plan = plan
        self.calls = 0

    def plan(self, _question: str, _catalog: GraphCapabilityCatalog) -> GraphRetrievalPlan:
        self.calls += 1
        return self._plan


def test_hybrid_keeps_vector_evidence_when_question_is_outside_graph_scope(
    caplog: pytest.LogCaptureFixture,
) -> None:
    planner = FixedPlanner(
        GraphRetrievalPlan(use_graph=False, decision_reason="outside_graph_scope")
    )
    hybrid = HybridRetriever(
        FixedVectorRetriever((_evidence("vector", "article-1"),)),
        FixedGraphRetriever(
            GraphCapabilityCatalog(("authority",), ("assigns_responsibility_to",), ("authority",))
        ),
        planner,
    )

    with caplog.at_level(logging.INFO, logger="norvii_agent.retrieval.hybrid"):
        result = hybrid.search(_corpus_id(), _snapshot_id(), "What is the purpose?")

    assert [item.id for item in result] == ["vector"]
    assert planner.calls == 1
    assert [(stage.name, stage.state) for stage in hybrid.last_stages] == [
        ("vector", "completed"),
        ("planning", "skipped"),
        ("graph", "skipped"),
    ]
    assert '"decision_reason": "outside_graph_scope"' in caplog.text
    assert '"use_graph": false' in caplog.text


def test_hybrid_keeps_vector_evidence_when_graph_is_unavailable() -> None:
    hybrid = HybridRetriever(
        FixedVectorRetriever((_evidence("vector", "article-1"),)),
        FixedGraphRetriever(None, unavailable=True),
        FixedPlanner(
            GraphRetrievalPlan(
                use_graph=True,
                predicates=("assigns_responsibility_to",),
                entity_labels=("authority",),
            )
        ),
    )

    result = hybrid.search(_corpus_id(), _snapshot_id(), "Who governs the authority?")

    assert [item.id for item in result] == ["vector"]
    assert [(stage.name, stage.state) for stage in hybrid.last_stages] == [
        ("vector", "completed"),
        ("planning", "unavailable"),
        ("graph", "unavailable"),
    ]


def test_hybrid_deduplicates_shared_immutable_locations_with_both_contributions() -> None:
    hybrid = HybridRetriever(
        FixedVectorRetriever((_evidence("vector", "article-1"),)),
        FixedGraphRetriever(
            GraphCapabilityCatalog(("authority",), ("assigns_responsibility_to",), ("authority",)),
            (_evidence("graph", "article-1"),),
        ),
        FixedPlanner(
            GraphRetrievalPlan(
                use_graph=True,
                predicates=("assigns_responsibility_to",),
                entity_labels=("authority",),
            )
        ),
    )

    result = hybrid.search(_corpus_id(), _snapshot_id(), "Who governs the authority?")

    assert len(result) == 1
    assert result[0].contribution == "vector_and_graph"
    assert hybrid.last_retrieval == RetrievalInspection("hybrid", 8, 1, "embedding")


def test_hybrid_records_no_assertion_evidence_without_losing_vector_results() -> None:
    hybrid = HybridRetriever(
        FixedVectorRetriever((_evidence("vector", "article-1"),)),
        FixedGraphRetriever(
            GraphCapabilityCatalog(("actor",), ("imposes_duty_on",), ("controller",))
        ),
        FixedPlanner(
            GraphRetrievalPlan(
                use_graph=True,
                predicates=("imposes_duty_on",),
                entity_labels=("controller",),
                scope_locator="chapter-1",
            )
        ),
    )

    result = hybrid.search(_corpus_id(), _snapshot_id(), "Who has the duty?")

    assert [item.id for item in result] == ["vector"]
    assert hybrid.last_stages[-1].reason_code == "no_assertion_evidence"


def test_hybrid_preserves_distinct_evidence_from_vector_and_graph() -> None:
    vector_evidence = tuple(
        _evidence(f"vector-{index}", f"article-{index}") for index in range(1, 9)
    )
    graph_evidence = tuple(
        _evidence(f"graph-{index}", f"graph-article-{index}") for index in range(1, 4)
    )
    hybrid = HybridRetriever(
        FixedVectorRetriever(vector_evidence),
        FixedGraphRetriever(
            GraphCapabilityCatalog(("authority",), ("must_be_observed_by",), ("authority",)),
            graph_evidence,
        ),
        FixedPlanner(
            GraphRetrievalPlan(
                use_graph=True,
                predicates=("must_be_observed_by",),
                entity_labels=("authority",),
            )
        ),
    )

    result = hybrid.search(_corpus_id(), _snapshot_id(), "What does the authority require?")

    assert [item.id for item in result] == [
        *(f"vector-{index}" for index in range(1, 9)),
        *(f"graph-{index}" for index in range(1, 4)),
    ]
    assert [item.contribution for item in result] == ["vector"] * 8 + ["graph"] * 3


def test_strategy_retriever_rejects_a_removed_graph_strategy() -> None:
    vector = FixedVectorRetriever(())
    strategies = StrategyRetriever(vector, vector)

    with pytest.raises(ValueError, match="retrieval strategy is unsupported"):
        strategies.search(_corpus_id(), _snapshot_id(), "Question", "graph")


def _evidence(identifier: str, locator: str) -> Evidence:
    return Evidence(
        identifier,
        _corpus_id(),
        UUID("20000000-0000-4000-8000-000000000001"),
        UUID("30000000-0000-4000-8000-000000000001"),
        locator,
        0,
        10,
        "Evidence excerpt.",
        1,
        snapshot_id=_snapshot_id(),
    )


def _corpus_id() -> UUID:
    return UUID("10000000-0000-4000-8000-000000000001")


def _snapshot_id() -> UUID:
    return UUID("50000000-0000-4000-8000-000000000001")
