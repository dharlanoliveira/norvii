"""LangGraph workflow for corpus-scoped grounded answers."""

from __future__ import annotations

from collections.abc import Callable
from dataclasses import dataclass
from typing import TYPE_CHECKING, Protocol, TypedDict

from langgraph.graph import END, START, StateGraph

if TYPE_CHECKING:
    from collections.abc import Sequence
    from uuid import UUID


@dataclass(frozen=True, slots=True)
class Evidence:
    """Immutable evidence location returned by retrieval."""

    id: str
    corpus_id: UUID
    source_id: UUID
    document_id: UUID
    unit_locator: str
    start_offset: int
    end_offset: int
    excerpt: str
    rank: int


class RetrievalPort(Protocol):
    """Retrieve evidence inside one corpus boundary."""

    def search(self, corpus_id: UUID, question: str) -> Sequence[Evidence]:
        """Return evidence owned by the requested corpus."""
        ...


class ChatModelPort(Protocol):
    """Generate an answer and emit bounded text deltas."""

    def generate(
        self,
        question: str,
        evidence: Sequence[Evidence],
        interface_language: str,
        emit: Callable[[str], None],
    ) -> str:
        """Generate an answer from the provided evidence."""
        ...


@dataclass(frozen=True, slots=True)
class GroundedChatRequest:
    """One ephemeral graph invocation."""

    corpus_id: UUID
    question: str
    interface_language: str = "en"


@dataclass(frozen=True, slots=True)
class GroundedChatResult:
    """Validated terminal graph result."""

    status: str
    answer: str
    evidence: tuple[Evidence, ...]
    reason: str | None = None


class _State(TypedDict, total=False):
    request: GroundedChatRequest
    evidence: tuple[Evidence, ...]
    answer: str
    status: str
    reason: str | None
    emit: Callable[[str], None]


class GroundedChatGraph:
    """Compile and invoke the grounded workflow through LangGraph nodes and edges."""

    def __init__(self, retriever: RetrievalPort, model: ChatModelPort) -> None:
        self._retriever = retriever
        self._model = model
        builder = StateGraph(_State)
        builder.add_node("retrieve", self._retrieve)
        builder.add_node("decide", self._decide)
        builder.add_node("abstain", self._abstain)  # type: ignore[arg-type]
        builder.add_node("generate", self._generate)
        builder.add_node("validate", self._validate)
        builder.add_edge(START, "retrieve")
        builder.add_edge("retrieve", "decide")
        builder.add_conditional_edges(
            "decide", self._route, {"abstain": "abstain", "generate": "generate"}
        )
        builder.add_edge("abstain", END)
        builder.add_edge("generate", "validate")
        builder.add_edge("validate", END)
        self._compiled = builder.compile()

    def run(
        self,
        request: GroundedChatRequest,
        emit: Callable[[str], None],
    ) -> GroundedChatResult:
        """Run one graph and return its terminal state."""
        state = self._compiled.invoke({"request": request, "emit": emit})
        return GroundedChatResult(
            status=state.get("status", "error"),
            answer=state.get("answer", ""),
            evidence=state.get("evidence", ()),
            reason=state.get("reason"),
        )

    def _retrieve(self, state: _State) -> _State:
        request = state["request"]
        evidence = tuple(self._retriever.search(request.corpus_id, request.question))[:8]
        return {"evidence": evidence}

    @staticmethod
    def _decide(state: _State) -> _State:
        return {"status": "answerable" if state.get("evidence") else "abstained"}

    @staticmethod
    def _route(state: _State) -> str:
        return "generate" if state.get("status") == "answerable" else "abstain"

    def _abstain(self, _state: _State) -> _State:
        return {"status": "abstained", "reason": "insufficient_evidence"}

    def _generate(self, state: _State) -> _State:
        request = state["request"]
        answer = self._model.generate(
            request.question,
            state.get("evidence", ()),
            request.interface_language,
            state["emit"],
        )
        return {"answer": answer}

    @staticmethod
    def _validate(state: _State) -> _State:
        answer = state.get("answer", "")
        evidence_count = len(state.get("evidence", ()))
        if not any(f"[{index}]" in answer for index in range(1, evidence_count + 1)):
            return {"status": "abstained", "reason": "grounding_validation_failed"}
        return {"status": "completed"}
