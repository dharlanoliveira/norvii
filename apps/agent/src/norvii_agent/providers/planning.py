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
_MAX_RELATIONSHIP_TYPES = 8
_MAX_TERM_LENGTH = 80


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
                            "relationship_types (array), and entity_labels (array). "
                            "Use only relationship types and entity labels in the catalog. "
                            "Use graph only when relationships or entities can materially help; "
                            "otherwise return false and empty arrays. Entity labels are canonical "
                            "graph values and can be in a different language than the question. "
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
                                    "relationship_types": catalog.relationship_types,
                                    "entity_labels": catalog.entity_labels,
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
    if not use_graph:
        return GraphRetrievalPlan(use_graph=False)
    allowed = set(catalog.relationship_types)
    relationship_types = tuple(
        item
        for item in _bounded_strings(decision.get("relationship_types"), _MAX_RELATIONSHIP_TYPES)
        if item in allowed
    )
    allowed_labels = set(catalog.entity_labels)
    entity_labels = tuple(
        item
        for item in _bounded_strings(decision.get("entity_labels"), _MAX_ENTITY_LABELS)
        if item in allowed_labels
    )
    if not relationship_types or not entity_labels:
        return GraphRetrievalPlan(use_graph=False)
    return GraphRetrievalPlan(
        use_graph=True,
        relationship_types=relationship_types,
        entity_labels=entity_labels,
    )


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
