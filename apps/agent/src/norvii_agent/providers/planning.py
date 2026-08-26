"""OpenAI-compatible bounded graph planner."""

from __future__ import annotations

import json
from dataclasses import dataclass, field, replace
from typing import cast
from urllib import error, request
from urllib.parse import urlparse

from norvii_agent.graph import ModelUsage
from norvii_agent.providers.chat import ProviderUnavailableError
from norvii_agent.retrieval.planning import GraphCapabilityCatalog, GraphRetrievalPlan

_MAX_ENTITY_LABELS = 8
_MAX_PREDICATES = 8
_MAX_SCOPE_LOCATOR_LENGTH = 160
_MAX_TERM_LENGTH = 80
_DECISION_REASONS = frozenset({"relationship_required", "outside_graph_scope", "uncertain"})
_PLANNING_POLICY = (
    "Do not assess whether vector evidence is sufficient. Decide only whether the graph "
    "can add grounded legal structure to the answer. Set use_graph to true when a "
    "catalog-supported assertion, entity connection, or legal hierarchy scope can materially "
    "improve the answer, including an authority's responsibility, a right's holder, an "
    "obligation's subject, an application condition, a legal definition, or a connection "
    "between named legal concepts. Set use_graph to false only when the question is outside "
    "the available graph structure."
)


@dataclass(slots=True)
class OpenAICompatibleGraphPlanner:
    """Ask the configured model for a strictly bounded graph-use decision."""

    base_url: str
    api_key: str
    model: str
    timeout_seconds: float
    reasoning_effort: str = "medium"
    last_usage: ModelUsage = field(
        default_factory=lambda: ModelUsage(None, None), init=False, repr=False, compare=False
    )

    def plan(self, question: str, catalog: GraphCapabilityCatalog) -> GraphRetrievalPlan:
        """Return a validated decision without allowing model-authored graph queries."""
        self.last_usage = ModelUsage(None, None)
        if not self.base_url or urlparse(self.base_url).scheme not in {"http", "https"}:
            raise ProviderUnavailableError("graph planner provider is not configured")
        payload = json.dumps(
            {
                "model": self.model,
                "reasoning_effort": self.reasoning_effort,
                "stream": False,
                "response_format": {"type": "json_object"},
                "messages": [
                    {
                        "role": "system",
                        "content": (
                            "Decide whether graph relationships can add grounded evidence to a "
                            "legal research question. Return JSON only with use_graph (boolean), "
                            "decision_reason (one of relationship_required, "
                            "outside_graph_scope, or uncertain), predicates "
                            "(array), entity_labels (array), and scope_locator (string or null). "
                            "Use only catalog predicates, entity labels, and scope locators. "
                            "Predicate capabilities define the only valid predicate "
                            "and entity-label combinations. "
                            f"{_PLANNING_POLICY} "
                            "Use relationship_required only when use_graph is true. When "
                            "use_graph is false, use outside_graph_scope or uncertain and return "
                            "empty arrays and null scope_locator. Select scope_locator only "
                            "when the question explicitly restricts a legal location. "
                            "Entity labels are canonical graph values and can be in a different "
                            "language than the question. "
                            "Never write Cypher, "
                            "never infer schema, and never answer the question."
                        ),
                    },
                    {
                        "role": "user",
                        "content": json.dumps(
                            {
                                "question": question,
                                "graph_capabilities": {
                                    "entity_types": catalog.entity_types,
                                    "predicates": catalog.predicates,
                                    "entity_labels": catalog.entity_labels,
                                    "predicate_capabilities": [
                                        {
                                            "predicate": capability.predicate,
                                            "entity_labels": capability.entity_labels,
                                        }
                                        for capability in catalog.predicate_capabilities
                                    ],
                                    "scope_locators": catalog.scope_locators,
                                },
                            }
                        ),
                    },
                ],
            }
        ).encode()
        headers = {"Content-Type": "application/json"}
        if self.api_key:
            headers["Authorization"] = f"Bearer {self.api_key}"
        http_request = request.Request(  # noqa: S310 - scheme validated above
            self.base_url, data=payload, headers=headers, method="POST"
        )
        try:
            with request.urlopen(http_request, timeout=self.timeout_seconds) as response:  # noqa: S310
                decoded = _payload(response.read())
        except (error.URLError, TimeoutError, json.JSONDecodeError) as exc:
            raise ProviderUnavailableError("graph planner provider request failed") from exc
        self.last_usage = _usage(decoded.get("usage"))
        return replace(
            _plan(decoded, catalog),
            input_tokens=self.last_usage.input_tokens,
            output_tokens=self.last_usage.output_tokens,
        )


def _payload(body: bytes) -> dict[str, object]:
    """Decode a provider response without retaining provider-specific payloads."""
    decoded = cast("dict[str, object]", json.loads(body))
    try:
        choices = cast("list[dict[str, object]]", decoded["choices"])
        message = cast("dict[str, object]", choices[0]["message"])
        content = cast("str", message["content"])
    except (KeyError, IndexError, TypeError) as exc:
        raise ProviderUnavailableError("graph planner response shape is invalid") from exc
    try:
        decision = cast("dict[str, object]", json.loads(content))
    except (TypeError, json.JSONDecodeError) as exc:
        raise ProviderUnavailableError("graph planner response is not JSON") from exc
    return {"decision": decision, "usage": decoded.get("usage")}


def _plan(payload: dict[str, object], catalog: GraphCapabilityCatalog) -> GraphRetrievalPlan:
    """Validate every model-produced field against bounded local policy."""
    decision = payload.get("decision")
    if not isinstance(decision, dict):
        raise ProviderUnavailableError("graph planner decision is invalid")
    use_graph = decision.get("use_graph")
    if not isinstance(use_graph, bool):
        raise ProviderUnavailableError("graph planner use_graph is invalid")
    decision_reason = _decision_reason(decision.get("decision_reason"), use_graph=use_graph)
    if not use_graph:
        return GraphRetrievalPlan(use_graph=False, decision_reason=decision_reason)
    if decision_reason != "relationship_required":
        return GraphRetrievalPlan(use_graph=False, decision_reason="uncertain")
    labels_by_predicate = {
        capability.predicate: set(capability.entity_labels)
        for capability in catalog.predicate_capabilities
    }
    predicates = tuple(
        item
        for item in _bounded_strings(decision.get("predicates"), _MAX_PREDICATES)
        if item in labels_by_predicate
    )
    predicates, entity_labels = _supported_plan_filters(
        predicates,
        _bounded_strings(decision.get("entity_labels"), _MAX_ENTITY_LABELS),
        labels_by_predicate,
    )
    scope_locator = _scope_locator(decision.get("scope_locator"), catalog.scope_locators)
    if not predicates or not entity_labels:
        return GraphRetrievalPlan(use_graph=False, decision_reason="uncertain")
    return GraphRetrievalPlan(
        use_graph=True,
        decision_reason=decision_reason,
        predicates=predicates,
        entity_labels=entity_labels,
        scope_locator=scope_locator,
    )


def _supported_plan_filters(
    predicates: tuple[str, ...],
    entity_labels: tuple[str, ...],
    labels_by_predicate: dict[str, set[str]],
) -> tuple[tuple[str, ...], tuple[str, ...]]:
    """Keep only filters that can match a published normative assertion."""
    if not predicates:
        return (), ()
    supported_labels = set().union(*(labels_by_predicate[predicate] for predicate in predicates))
    selected_labels = tuple(label for label in entity_labels if label in supported_labels)
    selected_predicates = tuple(
        predicate
        for predicate in predicates
        if any(label in labels_by_predicate[predicate] for label in selected_labels)
    )
    return selected_predicates, selected_labels


def _scope_locator(value: object, available_locators: tuple[str, ...]) -> str | None:
    """Accept one published hierarchy scope and reject model-invented locations."""
    if not isinstance(value, str):
        return None
    normalized = value.strip().lower()
    if not normalized or len(normalized) > _MAX_SCOPE_LOCATOR_LENGTH:
        return None
    return normalized if normalized in available_locators else None


def _decision_reason(value: object, *, use_graph: bool) -> str:
    """Normalize an unsafe or incomplete provider explanation to the safe decision."""
    if not isinstance(value, str) or value not in _DECISION_REASONS:
        return "uncertain"
    if use_graph and value != "relationship_required":
        return "uncertain"
    if not use_graph and value == "relationship_required":
        return "uncertain"
    return value


def _bounded_strings(value: object, limit: int) -> tuple[str, ...]:
    """Accept compact unique strings only at the provider boundary."""
    if not isinstance(value, list):
        return ()
    result: list[str] = []
    for item in value:
        if not isinstance(item, str):
            continue
        normalized = item.strip().lower()
        if not normalized or len(normalized) > _MAX_TERM_LENGTH or normalized in result:
            continue
        result.append(normalized)
        if len(result) == limit:
            break
    return tuple(result)


def _usage(value: object) -> ModelUsage:
    """Read provider-reported usage without estimating costs."""
    if not isinstance(value, dict):
        return ModelUsage(None, None)
    return ModelUsage(
        _token_count(value.get("prompt_tokens")),
        _token_count(value.get("completion_tokens")),
    )


def _token_count(value: object) -> int | None:
    return value if isinstance(value, int) and not isinstance(value, bool) and value >= 0 else None
