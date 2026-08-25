"""Planned hybrid retrieval with vector evidence as the reliable baseline."""

from __future__ import annotations

from dataclasses import replace
from time import perf_counter
from typing import TYPE_CHECKING, Protocol

from norvii_agent.graph import (
    Evidence,
    GraphPathStep,
    RetrievalInspection,
    RetrievalStage,
    StrategyUnavailableError,
)

if TYPE_CHECKING:
    from uuid import UUID

    from norvii_agent.graph import RetrievalPort
    from norvii_agent.retrieval.planning import (
        GraphCapabilityCatalog,
        GraphPlanner,
        GraphRetrievalPlan,
    )


class PlannedGraphRetriever(Protocol):
    """Expose only bounded graph operations needed by hybrid retrieval."""

    last_graph_path: tuple[GraphPathStep, ...]

    def capabilities(self, corpus_id: UUID, snapshot_id: UUID) -> GraphCapabilityCatalog | None:
        """Return graph capabilities for a ready snapshot, if any."""
        ...

    def search_plan(
        self, corpus_id: UUID, snapshot_id: UUID, plan: GraphRetrievalPlan
    ) -> tuple[Evidence, ...]:
        """Execute one validated graph lookup."""
        ...


class HybridRetriever:
    """Always retrieve vector evidence and conditionally enrich it through graph planning."""

    def __init__(
        self, vector: RetrievalPort, graph: PlannedGraphRetriever, planner: GraphPlanner
    ) -> None:
        self._vector = vector
        self._graph = graph
        self._planner = planner
        self.last_retrieval: RetrievalInspection | None = None
        self.last_graph_path: tuple[GraphPathStep, ...] = ()
        self.last_stages: tuple[RetrievalStage, ...] = ()

    def search(
        self, corpus_id: UUID, snapshot_id: UUID, question: str, strategy: str = "hybrid"
    ) -> tuple[Evidence, ...]:
        """Return vector evidence plus any safe, planned graph contribution."""
        if strategy != "hybrid":
            raise ValueError("HybridRetriever only supports hybrid retrieval")
        self.last_graph_path = ()
        vector_evidence, vector_stage = self._vector_evidence(corpus_id, snapshot_id, question)
        stages: list[RetrievalStage] = [vector_stage]
        graph_evidence = self._graph_evidence(corpus_id, snapshot_id, question, stages)
        evidence = _deduplicated(vector_evidence, graph_evidence)
        vector_metadata = getattr(self._vector, "last_retrieval", None)
        embedding_model = (
            vector_metadata.embedding_model
            if isinstance(vector_metadata, RetrievalInspection)
            else None
        )
        self.last_retrieval = RetrievalInspection("hybrid", 8, len(evidence), embedding_model)
        self.last_stages = tuple(stages)
        return evidence

    def _vector_evidence(
        self, corpus_id: UUID, snapshot_id: UUID, question: str
    ) -> tuple[tuple[Evidence, ...], RetrievalStage]:
        started = perf_counter()
        evidence = tuple(
            replace(item, contribution="vector")
            for item in self._vector.search(corpus_id, snapshot_id, question, "vector")
        )
        return evidence, RetrievalStage(
            "vector",
            "completed" if evidence else "no_evidence",
            len(evidence),
            _elapsed_milliseconds(started),
        )

    def _graph_evidence(
        self,
        corpus_id: UUID,
        snapshot_id: UUID,
        question: str,
        stages: list[RetrievalStage],
    ) -> tuple[Evidence, ...]:
        try:
            catalog = self._graph.capabilities(corpus_id, snapshot_id)
        except StrategyUnavailableError:
            stages.extend(
                (
                    RetrievalStage("planning", "unavailable", 0, None, "graph_unavailable"),
                    RetrievalStage("graph", "unavailable", 0, None, "graph_unavailable"),
                )
            )
            return ()
        if catalog is None:
            stages.extend(
                (
                    RetrievalStage("planning", "skipped", 0, None, "graph_release_unavailable"),
                    RetrievalStage("graph", "skipped", 0, None, "graph_release_unavailable"),
                )
            )
            return ()
        started = perf_counter()
        try:
            plan = self._planner.plan(question, catalog)
        except StrategyUnavailableError:
            stages.extend(
                (
                    RetrievalStage(
                        "planning",
                        "failed",
                        0,
                        _elapsed_milliseconds(started),
                        "planner_unavailable",
                    ),
                    RetrievalStage("graph", "skipped", 0, None, "planner_unavailable"),
                )
            )
            return ()
        stages.append(
            RetrievalStage(
                "planning",
                "completed" if plan.use_graph else "skipped",
                0,
                _elapsed_milliseconds(started),
                None if plan.use_graph else "not_relevant",
                plan.input_tokens,
                plan.output_tokens,
            )
        )
        if not plan.use_graph:
            stages.append(RetrievalStage("graph", "skipped", 0, None, "not_relevant"))
            return ()
        started = perf_counter()
        try:
            evidence = tuple(
                replace(item, contribution="graph")
                for item in self._graph.search_plan(corpus_id, snapshot_id, plan)
            )
        except StrategyUnavailableError:
            stages.append(
                RetrievalStage(
                    "graph", "unavailable", 0, _elapsed_milliseconds(started), "graph_unavailable"
                )
            )
            return ()
        self.last_graph_path = tuple(getattr(self._graph, "last_graph_path", ()))
        stages.append(
            RetrievalStage(
                "graph",
                "completed" if evidence else "no_evidence",
                len(evidence),
                _elapsed_milliseconds(started),
            )
        )
        return evidence


class StrategyRetriever:
    """Select a public retrieval strategy without changing its evidence identity."""

    def __init__(self, vector: RetrievalPort, hybrid: RetrievalPort) -> None:
        self._strategies = {"vector": vector, "hybrid": hybrid}
        self.last_retrieval: RetrievalInspection | None = None
        self.last_graph_path: tuple[GraphPathStep, ...] = ()
        self.last_stages: tuple[RetrievalStage, ...] = ()

    def search(
        self, corpus_id: UUID, snapshot_id: UUID, question: str, strategy: str = "vector"
    ) -> tuple[Evidence, ...]:
        """Delegate only to a declared public retrieval strategy."""
        selected = self._strategies.get(strategy)
        if selected is None:
            raise ValueError("retrieval strategy is unsupported")
        evidence = tuple(selected.search(corpus_id, snapshot_id, question, strategy))
        self.last_retrieval = getattr(selected, "last_retrieval", None)
        self.last_graph_path = tuple(getattr(selected, "last_graph_path", ()))
        stages = tuple(getattr(selected, "last_stages", ()))
        self.last_stages = stages or (
            RetrievalStage(
                "vector",
                "completed" if evidence else "no_evidence",
                len(evidence),
                None,
            ),
        )
        return evidence


def _deduplicated(
    vector_evidence: tuple[Evidence, ...], graph_evidence: tuple[Evidence, ...]
) -> tuple[Evidence, ...]:
    """Keep immutable locations once while preserving every retrieval contribution."""
    locations: dict[tuple[object, ...], Evidence] = {}
    for evidence in (*vector_evidence, *graph_evidence):
        key = (
            evidence.document_id,
            evidence.unit_locator,
            evidence.start_offset,
            evidence.end_offset,
        )
        existing = locations.get(key)
        if existing is None:
            locations[key] = evidence
            continue
        locations[key] = replace(existing, contribution="vector_and_graph")
    return tuple(
        evidence if evidence.rank == index else replace(evidence, rank=index)
        for index, evidence in enumerate(locations.values(), start=1)
    )


def _elapsed_milliseconds(started: float) -> int:
    """Return a monotonic duration suitable for safe inspection metadata."""
    return max(0, round((perf_counter() - started) * 1000))
