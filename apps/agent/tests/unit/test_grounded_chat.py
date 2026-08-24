from __future__ import annotations

from collections.abc import Callable
from uuid import UUID

from norvii_agent.graph import Evidence, GroundedChatGraph, GroundedChatRequest


class FakeRetriever:
    def __init__(self, evidence: tuple[Evidence, ...]) -> None:
        self.evidence = evidence

    def search(self, corpus_id: UUID, question: str) -> tuple[Evidence, ...]:
        assert corpus_id
        assert question
        return self.evidence


class FakeModel:
    def __init__(self, answer: str) -> None:
        self.answer = answer
        self.received_evidence: tuple[Evidence, ...] | None = None

    def generate(
        self,
        question: str,
        evidence: tuple[Evidence, ...],
        interface_language: str,
        emit: Callable[[str], None],
    ) -> str:
        assert question
        assert interface_language == "en"
        self.received_evidence = evidence
        emit(self.answer)
        return self.answer


def evidence() -> Evidence:
    return Evidence(
        "evidence-1",
        UUID("10000000-0000-4000-8000-000000000001"),
        UUID("20000000-0000-4000-8000-000000000001"),
        UUID("30000000-0000-4000-8000-000000000001"),
        "article-1",
        0,
        20,
        "The rule applies.",
        1,
    )


def test_graph_completes_grounded_answer_and_hides_its_mode_marker() -> None:
    deltas: list[str] = []
    graph = GroundedChatGraph(
        FakeRetriever((evidence(),)), FakeModel("[NORVII_GROUNDED]\nThe rule applies [1].")
    )

    result = graph.run(
        GroundedChatRequest(UUID("10000000-0000-4000-8000-000000000001"), "What applies?"),
        deltas.append,
    )

    assert result.status == "completed"
    assert result.evidence[0].unit_locator == "article-1"
    assert deltas == ["The rule applies [1]."]
    assert result.inspection is not None
    assert result.inspection.retrieval.strategy == "vector"
    assert result.inspection.retrieval.returned_count == 1
    assert result.inspection.measurements.retrieval_milliseconds is not None
    assert result.inspection.measurements.generation_milliseconds is not None
    assert result.inspection.evidence == result.evidence


def test_graph_generates_scope_limited_response_when_retrieval_is_empty() -> None:
    model = FakeModel("[NORVII_SCOPE_LIMITED]\nHello. Ask me about documents in this corpus.")
    graph = GroundedChatGraph(FakeRetriever(()), model)

    result = graph.run(
        GroundedChatRequest(UUID("10000000-0000-4000-8000-000000000001"), "Unknown?"),
        lambda _: None,
    )

    assert result.status == "completed"
    assert result.answer == "Hello. Ask me about documents in this corpus."
    assert result.evidence == ()
    assert model.received_evidence == ()
