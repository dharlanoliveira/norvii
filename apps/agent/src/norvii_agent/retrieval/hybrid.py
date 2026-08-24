"""Deterministic composition of vector and graph evidence."""

from __future__ import annotations

from dataclasses import replace
from typing import TYPE_CHECKING

from norvii_agent.graph import Evidence, GraphPathStep, RetrievalInspection

if TYPE_CHECKING:
    from uuid import UUID

    from norvii_agent.graph import RetrievalPort


class HybridRetriever:
    """Combine independently retrieved graph and vector evidence without hidden fallback."""

    def __init__(self, vector: RetrievalPort, graph: RetrievalPort) -> None:
        self._vector = vector
        self._graph = graph
        self.last_retrieval: RetrievalInspection | None = None
        self.last_graph_path: tuple[GraphPathStep, ...] = ()

    def search(
        self, corpus_id: UUID, snapshot_id: UUID, question: str, strategy: str = "hybrid"
    ) -> tuple[Evidence, ...]:
        """Return deduplicated evidence only when both declared contributions are available."""
        if strategy != "hybrid":
            raise ValueError("HybridRetriever only supports hybrid retrieval")
        graph_evidence = tuple(self._graph.search(corpus_id, snapshot_id, question, "graph"))
        vector_evidence = tuple(self._vector.search(corpus_id, snapshot_id, question, "vector"))
        if not graph_evidence:
            self.last_retrieval = RetrievalInspection("hybrid", 8, 0, None)
            self.last_graph_path = ()
            return ()
        self.last_graph_path = tuple(getattr(self._graph, "last_graph_path", ()))
        evidence = _deduplicated(graph_evidence, vector_evidence)
        vector_metadata = getattr(self._vector, "last_retrieval", None)
        embedding_model = (
            vector_metadata.embedding_model
            if isinstance(vector_metadata, RetrievalInspection)
            else None
        )
        self.last_retrieval = RetrievalInspection("hybrid", 8, len(evidence), embedding_model)
        return evidence


class StrategyRetriever:
    """Select a declared retrieval strategy without changing its evidence identity."""

    def __init__(self, vector: RetrievalPort, graph: RetrievalPort, hybrid: RetrievalPort) -> None:
        self._strategies = {"vector": vector, "graph": graph, "hybrid": hybrid}
        self.last_retrieval: RetrievalInspection | None = None
        self.last_graph_path: tuple[GraphPathStep, ...] = ()

    def search(
        self, corpus_id: UUID, snapshot_id: UUID, question: str, strategy: str = "vector"
    ) -> tuple[Evidence, ...]:
        """Delegate only to the selected declared retrieval strategy."""
        selected = self._strategies.get(strategy)
        if selected is None:
            raise ValueError("retrieval strategy is unsupported")
        evidence = tuple(selected.search(corpus_id, snapshot_id, question, strategy))
        self.last_retrieval = getattr(selected, "last_retrieval", None)
        self.last_graph_path = tuple(getattr(selected, "last_graph_path", ()))
        return evidence


def _deduplicated(
    graph_evidence: tuple[Evidence, ...], vector_evidence: tuple[Evidence, ...]
) -> tuple[Evidence, ...]:
    locations: dict[tuple[object, ...], Evidence] = {}
    for evidence in (*graph_evidence, *vector_evidence):
        key = (
            evidence.document_id,
            evidence.unit_locator,
            evidence.start_offset,
            evidence.end_offset,
        )
        locations.setdefault(key, evidence)
    return tuple(
        evidence if evidence.rank == index else replace(evidence, rank=index)
        for index, evidence in enumerate(locations.values(), start=1)
    )[:8]
