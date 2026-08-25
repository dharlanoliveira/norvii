from __future__ import annotations

from collections.abc import Callable
from uuid import UUID

from norvii_agent.graph import Evidence, GroundedChatGraph, GroundedChatRequest, RetrievalStage


class FakeRetriever:
    def __init__(self, evidence: tuple[Evidence, ...]) -> None:
        self.evidence = evidence
        self.last_stages = (
            RetrievalStage("vector", "completed" if evidence else "no_evidence", len(evidence), 1),
        )

    def search(
        self, corpus_id: UUID, snapshot_id: UUID, question: str, strategy: str = "vector"
    ) -> tuple[Evidence, ...]:
        assert corpus_id
        assert snapshot_id
        assert question
        assert strategy == "vector"
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
        GroundedChatRequest(
            UUID("10000000-0000-4000-8000-000000000001"),
            "What applies?",
            snapshot_id=UUID("50000000-0000-4000-8000-000000000001"),
        ),
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
    assert result.inspection.stages[0].state == "completed"


def test_graph_generates_scope_limited_response_when_retrieval_is_empty() -> None:
    model = FakeModel("[NORVII_SCOPE_LIMITED]\nHello. Ask me about documents in this corpus.")
    graph = GroundedChatGraph(FakeRetriever(()), model)

    result = graph.run(
        GroundedChatRequest(
            UUID("10000000-0000-4000-8000-000000000001"),
            "Unknown?",
            snapshot_id=UUID("50000000-0000-4000-8000-000000000001"),
        ),
        lambda _: None,
    )

    assert result.status == "completed"
    assert result.answer == "Hello. Ask me about documents in this corpus."
    assert result.evidence == ()
    assert model.received_evidence == ()
    assert result.inspection is not None
    assert result.inspection.stages[0].state == "no_evidence"


def test_graph_preserves_all_retrieval_evidence_for_generation() -> None:
    evidence_items = tuple(
        Evidence(
            f"evidence-{index}",
            UUID("10000000-0000-4000-8000-000000000001"),
            UUID("20000000-0000-4000-8000-000000000001"),
            UUID("30000000-0000-4000-8000-000000000001"),
            f"article-{index}",
            0,
            20,
            f"The rule {index} applies.",
            index,
        )
        for index in range(1, 12)
    )
    model = FakeModel("[NORVII_GROUNDED]\nThe rules apply [1].")
    graph = GroundedChatGraph(FakeRetriever(evidence_items), model)

    result = graph.run(
        GroundedChatRequest(
            UUID("10000000-0000-4000-8000-000000000001"),
            "What applies?",
            snapshot_id=UUID("50000000-0000-4000-8000-000000000001"),
        ),
        lambda _: None,
    )

    assert model.received_evidence == evidence_items
    assert result.evidence == evidence_items
