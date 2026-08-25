from __future__ import annotations

import json
from io import BytesIO
from types import SimpleNamespace
from typing import TYPE_CHECKING, Self

from norvii_agent.providers import OpenAICompatibleGraphPlanner
from norvii_agent.retrieval.planning import GraphCapabilityCatalog

if TYPE_CHECKING:
    import pytest


class FakeResponse:
    def __init__(self, body: bytes) -> None:
        self._body = BytesIO(body)
        self.headers = SimpleNamespace()

    def __enter__(self) -> Self:
        return self

    def __exit__(self, *_args: object) -> None:
        return None

    def read(self) -> bytes:
        return self._body.read()


def test_graph_planner_keeps_only_declared_capabilities(monkeypatch: pytest.MonkeyPatch) -> None:
    response = FakeResponse(
        json.dumps(
            {
                "choices": [
                    {
                        "message": {
                            "content": json.dumps(
                                {
                                    "use_graph": True,
                                    "relationship_types": ["governs", "invented"],
                                    "entity_labels": ["autoridade nacional", "invented"],
                                }
                            )
                        }
                    }
                ],
                "usage": {"prompt_tokens": 12, "completion_tokens": 4},
            }
        ).encode()
    )
    monkeypatch.setattr(
        "norvii_agent.providers.planning.request.urlopen", lambda *_args, **_kwargs: response
    )
    planner = OpenAICompatibleGraphPlanner("https://provider.test/chat", "", "model", 1)

    plan = planner.plan(
        "Who governs personal data?",
        GraphCapabilityCatalog(
            ("authority",),
            ("governs", "applies_to"),
            ("autoridade nacional",),
        ),
    )

    assert plan.use_graph is True
    assert plan.relationship_types == ("governs",)
    assert plan.entity_labels == ("autoridade nacional",)
    assert plan.input_tokens == 12
    assert plan.output_tokens == 4
    assert planner.last_usage.input_tokens == 12
    assert planner.last_usage.output_tokens == 4


def test_graph_planner_rejects_a_plan_without_declared_entity_labels(
    monkeypatch: pytest.MonkeyPatch,
) -> None:
    response = FakeResponse(
        b'{"choices":[{"message":{"content":"{\\"use_graph\\":true,\\"relationship_types\\":[\\"governs\\"],\\"entity_labels\\":[]}"}}]}'
    )
    monkeypatch.setattr(
        "norvii_agent.providers.planning.request.urlopen", lambda *_args, **_kwargs: response
    )

    plan = OpenAICompatibleGraphPlanner("https://provider.test/chat", "", "model", 1).plan(
        "Who governs personal data?",
        GraphCapabilityCatalog(("authority",), ("governs",), ("autoridade nacional",)),
    )

    assert plan.use_graph is False
