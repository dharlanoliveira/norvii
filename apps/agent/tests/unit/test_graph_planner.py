from __future__ import annotations

import json
from io import BytesIO
from types import SimpleNamespace
from typing import TYPE_CHECKING, Self
from urllib.request import Request

from norvii_agent.providers import OpenAICompatibleGraphPlanner
from norvii_agent.retrieval.planning import GraphCapabilityCatalog, GraphPredicateCapability

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
                                    "decision_reason": "relationship_required",
                                    "predicates": ["assigns_responsibility_to", "invented"],
                                    "entity_labels": ["autoridade nacional", "invented"],
                                    "scope_locator": "chapter-1",
                                }
                            )
                        }
                    }
                ],
                "usage": {"prompt_tokens": 12, "completion_tokens": 4},
            }
        ).encode()
    )
    requests: list[Request] = []

    def respond(http_request: Request, *_args: object, **_kwargs: object) -> FakeResponse:
        requests.append(http_request)
        return response

    monkeypatch.setattr("norvii_agent.providers.planning.request.urlopen", respond)
    planner = OpenAICompatibleGraphPlanner("https://provider.test/chat", "", "model", 1)

    plan = planner.plan(
        "Which authority is responsible in chapter 1?",
        GraphCapabilityCatalog(
            ("authority",),
            ("assigns_responsibility_to", "applies_to"),
            ("autoridade nacional",),
            (GraphPredicateCapability("assigns_responsibility_to", ("autoridade nacional",)),),
            ("chapter-1",),
        ),
    )

    assert plan.use_graph is True
    assert plan.decision_reason == "relationship_required"
    assert plan.predicates == ("assigns_responsibility_to",)
    assert plan.entity_labels == ("autoridade nacional",)
    assert plan.scope_locator == "chapter-1"
    assert plan.input_tokens == 12
    assert plan.output_tokens == 4
    assert planner.last_usage.input_tokens == 12
    assert planner.last_usage.output_tokens == 4
    assert len(requests) == 1
    assert requests[0].data is not None
    request_body = json.loads(requests[0].data)
    policy = request_body["messages"][0]["content"]
    assert "Vector retrieval has already searched" in policy
    assert "If a relationship is not necessary" in policy
    assert "decision_reason" in policy
    assert "predicate_capabilities" in request_body["messages"][1]["content"]
    assert "scope_locators" in request_body["messages"][1]["content"]


def test_graph_planner_rejects_a_plan_without_declared_entity_labels(
    monkeypatch: pytest.MonkeyPatch,
) -> None:
    response = FakeResponse(
        b'{"choices":[{"message":{"content":"{\\"use_graph\\":true,\\"decision_reason\\":\\"relationship_required\\",\\"predicates\\":[\\"assigns_responsibility_to\\"],\\"entity_labels\\":[]}"}}]}'
    )
    monkeypatch.setattr(
        "norvii_agent.providers.planning.request.urlopen", lambda *_args, **_kwargs: response
    )

    plan = OpenAICompatibleGraphPlanner("https://provider.test/chat", "", "model", 1).plan(
        "Who is responsible?",
        GraphCapabilityCatalog(
            ("authority",),
            ("assigns_responsibility_to",),
            ("autoridade nacional",),
            (GraphPredicateCapability("assigns_responsibility_to", ("autoridade nacional",)),),
        ),
    )

    assert plan.use_graph is False
    assert plan.decision_reason == "uncertain"


def test_graph_planner_rejects_unpaired_predicate_and_entity_filters(
    monkeypatch: pytest.MonkeyPatch,
) -> None:
    response = FakeResponse(
        b'{"choices":[{"message":{"content":"{\\"use_graph\\":true,\\"decision_reason\\":\\"relationship_required\\",\\"predicates\\":[\\"must_be_observed_by\\"],\\"entity_labels\\":[\\"document\\"]}"}}]}'
    )
    monkeypatch.setattr(
        "norvii_agent.providers.planning.request.urlopen", lambda *_args, **_kwargs: response
    )

    plan = OpenAICompatibleGraphPlanner("https://provider.test/chat", "", "model", 1).plan(
        "Which public bodies must follow the general rules?",
        GraphCapabilityCatalog(
            ("document", "public_body"),
            ("must_be_observed_by",),
            ("document", "public body"),
            (GraphPredicateCapability("must_be_observed_by", ("public body",)),),
        ),
    )

    assert plan.use_graph is False
    assert plan.decision_reason == "uncertain"


def test_graph_planner_discards_an_invented_scope_locator(
    monkeypatch: pytest.MonkeyPatch,
) -> None:
    response = FakeResponse(
        b'{"choices":[{"message":{"content":"{\\"use_graph\\":true,\\"decision_reason\\":\\"relationship_required\\",\\"predicates\\":[\\"imposes_duty_on\\"],\\"entity_labels\\":[\\"controller\\"],\\"scope_locator\\":\\"invented-chapter\\"}"}}]}'
    )
    monkeypatch.setattr(
        "norvii_agent.providers.planning.request.urlopen", lambda *_args, **_kwargs: response
    )

    plan = OpenAICompatibleGraphPlanner("https://provider.test/chat", "", "model", 1).plan(
        "Who has a duty?",
        GraphCapabilityCatalog(
            ("actor",),
            ("imposes_duty_on",),
            ("controller",),
            (GraphPredicateCapability("imposes_duty_on", ("controller",)),),
            ("chapter-1",),
        ),
    )

    assert plan.use_graph is True
    assert plan.scope_locator is None


def test_graph_planner_records_direct_evidence_as_sufficient(
    monkeypatch: pytest.MonkeyPatch,
) -> None:
    response = FakeResponse(
        b'{"choices":[{"message":{"content":"{\\"use_graph\\":false,\\"decision_reason\\":\\"direct_evidence_sufficient\\",\\"predicates\\":[],\\"entity_labels\\":[]}"}}]}'
    )
    monkeypatch.setattr(
        "norvii_agent.providers.planning.request.urlopen", lambda *_args, **_kwargs: response
    )

    plan = OpenAICompatibleGraphPlanner("https://provider.test/chat", "", "model", 1).plan(
        "What does article 4 establish?",
        GraphCapabilityCatalog(("article",), ("applies_to",), ("article",)),
    )

    assert plan.use_graph is False
    assert plan.decision_reason == "direct_evidence_sufficient"
