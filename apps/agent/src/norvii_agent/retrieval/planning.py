"""Bounded planning contracts for optional graph retrieval."""

from __future__ import annotations

from dataclasses import dataclass
from typing import Protocol

NORMATIVE_PREDICATES = frozenset(
    {
        "defines",
        "applies_to",
        "must_be_observed_by",
        "imposes_duty_on",
        "grants",
        "protects",
        "assigns_responsibility_to",
        "conditions",
    }
)


@dataclass(frozen=True, slots=True)
class GraphPredicateCapability:
    """One assertion predicate and the canonical entities it actually connects."""

    predicate: str
    entity_labels: tuple[str, ...]


@dataclass(frozen=True, slots=True)
class GraphCapabilityCatalog:
    """Snapshot-scoped graph capabilities that may be considered by the planner."""

    entity_types: tuple[str, ...]
    predicates: tuple[str, ...]
    entity_labels: tuple[str, ...]
    predicate_capabilities: tuple[GraphPredicateCapability, ...] = ()
    scope_locators: tuple[str, ...] = ()


@dataclass(frozen=True, slots=True)
class GraphRetrievalPlan:
    """Validated, bounded graph lookup selected for one question."""

    use_graph: bool
    decision_reason: str = "uncertain"
    predicates: tuple[str, ...] = ()
    entity_labels: tuple[str, ...] = ()
    scope_locator: str | None = None
    input_tokens: int | None = None
    output_tokens: int | None = None


class GraphPlanner(Protocol):
    """Plan whether a ready graph can contribute to a bounded question."""

    def plan(self, question: str, catalog: GraphCapabilityCatalog) -> GraphRetrievalPlan:
        """Return a constrained graph retrieval decision for one question."""
        ...
