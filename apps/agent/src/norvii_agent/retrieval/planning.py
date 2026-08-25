"""Bounded planning contracts for optional graph retrieval."""

from __future__ import annotations

from dataclasses import dataclass
from typing import Protocol


@dataclass(frozen=True, slots=True)
class GraphCapabilityCatalog:
    """Snapshot-scoped graph capabilities that may be considered by the planner."""

    entity_types: tuple[str, ...]
    relationship_types: tuple[str, ...]
    entity_labels: tuple[str, ...]


@dataclass(frozen=True, slots=True)
class GraphRetrievalPlan:
    """Validated, bounded graph lookup selected for one question."""

    use_graph: bool
    relationship_types: tuple[str, ...] = ()
    entity_labels: tuple[str, ...] = ()
    input_tokens: int | None = None
    output_tokens: int | None = None


class GraphPlanner(Protocol):
    """Plan whether a ready graph can contribute to a bounded question."""

    def plan(self, question: str, catalog: GraphCapabilityCatalog) -> GraphRetrievalPlan:
        """Return a constrained graph retrieval decision for one question."""
        ...
