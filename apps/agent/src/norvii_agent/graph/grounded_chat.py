"""LangGraph workflow for corpus-scoped grounded answers."""

from __future__ import annotations

from collections.abc import Callable
from dataclasses import dataclass
from time import perf_counter
from typing import TYPE_CHECKING, Protocol, TypedDict, cast

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
    document_version_id: UUID | None = None
    source_revision_id: UUID | None = None
    pipeline_version: str | None = None
    source_title: str | None = None
    cosine_distance: float | None = None


@dataclass(frozen=True, slots=True)
class RetrievalInspection:
    """Content-free facts about one corpus-scoped retrieval operation."""

    strategy: str
    top_k: int
    returned_count: int
    embedding_model: str | None


@dataclass(frozen=True, slots=True)
class ExecutionMeasurements:
    """Measured execution values owned by the agent or its provider."""

    retrieval_milliseconds: int | None
    generation_milliseconds: int | None
    total_milliseconds: int | None = None
    input_tokens: int | None = None
    output_tokens: int | None = None


@dataclass(frozen=True, slots=True)
class AnswerInspection:
    """Ephemeral inspection details for one completed grounded answer."""

    outcome: str
    retrieval: RetrievalInspection
    measurements: ExecutionMeasurements
    evidence: tuple[Evidence, ...]


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
class ModelUsage:
    """Provider-reported token usage, when the provider returns it."""

    input_tokens: int | None
    output_tokens: int | None


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
    inspection: AnswerInspection | None = None


class _State(TypedDict, total=False):
    request: GroundedChatRequest
    evidence: tuple[Evidence, ...]
    answer: str
    status: str
    reason: str | None
    emit: Callable[[str], None]
    retrieval_inspection: RetrievalInspection
    retrieval_milliseconds: int | None
    generation_milliseconds: int | None
    input_tokens: int | None
    output_tokens: int | None
    response_mode: str


class GroundedChatGraph:
    """Compile and invoke the grounded workflow through LangGraph nodes and edges."""

    def __init__(self, retriever: RetrievalPort, model: ChatModelPort) -> None:
        self._retriever = retriever
        self._model = model
        builder = StateGraph(_State)
        builder.add_node("retrieve", self._retrieve)
        builder.add_node("generate", self._generate)
        builder.add_node("validate", self._validate)
        builder.add_edge(START, "retrieve")
        builder.add_edge("retrieve", "generate")
        builder.add_edge("generate", "validate")
        builder.add_edge("validate", END)
        self._compiled = builder.compile()

    def run(
        self,
        request: GroundedChatRequest,
        emit: Callable[[str], None],
    ) -> GroundedChatResult:
        """Run one graph and return its terminal state."""
        state = cast("_State", self._compiled.invoke({"request": request, "emit": emit}))
        return GroundedChatResult(
            status=state.get("status", "error"),
            answer=state.get("answer", ""),
            evidence=state.get("evidence", ()),
            reason=state.get("reason"),
            inspection=self._inspection(state),
        )

    def _retrieve(self, state: _State) -> _State:
        request = state["request"]
        started = perf_counter()
        evidence = tuple(self._retriever.search(request.corpus_id, request.question))[:8]
        elapsed = _elapsed_milliseconds(started)
        metadata = getattr(self._retriever, "last_retrieval", None)
        if not isinstance(metadata, RetrievalInspection):
            metadata = RetrievalInspection("vector", 8, len(evidence), None)
        return {
            "evidence": evidence,
            "retrieval_inspection": RetrievalInspection(
                metadata.strategy, metadata.top_k, len(evidence), metadata.embedding_model
            ),
            "retrieval_milliseconds": elapsed,
        }

    def _generate(self, state: _State) -> _State:
        request = state["request"]
        started = perf_counter()
        model_deltas: list[str] = []
        generated = self._model.generate(
            request.question,
            state.get("evidence", ()),
            request.interface_language,
            model_deltas.append,
        )
        answer, response_mode = _response_mode(generated)
        state["emit"](answer)
        usage = getattr(self._model, "last_usage", None)
        return {
            "answer": answer,
            "response_mode": response_mode,
            "generation_milliseconds": _elapsed_milliseconds(started),
            "input_tokens": usage.input_tokens if isinstance(usage, ModelUsage) else None,
            "output_tokens": usage.output_tokens if isinstance(usage, ModelUsage) else None,
        }

    @staticmethod
    def _validate(state: _State) -> _State:
        answer = state.get("answer", "")
        if state.get("response_mode") == "scope_limited":
            return {"status": "completed", "evidence": ()}
        evidence_count = len(state.get("evidence", ()))
        if not any(f"[{index}]" in answer for index in range(1, evidence_count + 1)):
            return {"status": "abstained", "reason": "grounding_validation_failed"}
        return {"status": "completed"}

    @staticmethod
    def _inspection(state: _State) -> AnswerInspection | None:
        if state.get("status") != "completed":
            return None
        evidence = tuple(state.get("evidence", ()))
        retrieval = state.get(
            "retrieval_inspection", RetrievalInspection("vector", 8, len(evidence), None)
        )
        return AnswerInspection(
            outcome="completed",
            retrieval=retrieval,
            measurements=ExecutionMeasurements(
                retrieval_milliseconds=state.get("retrieval_milliseconds"),
                generation_milliseconds=state.get("generation_milliseconds"),
                input_tokens=state.get("input_tokens"),
                output_tokens=state.get("output_tokens"),
            ),
            evidence=evidence,
        )


def _elapsed_milliseconds(started: float) -> int:
    """Convert a monotonic duration to a non-negative integer measurement."""
    return max(0, round((perf_counter() - started) * 1000))


def _response_mode(answer: str) -> tuple[str, str]:
    """Remove the model-only response mode and fail closed when it is absent."""
    marker, separator, content = answer.partition("\n")
    if separator and marker == "[NORVII_SCOPE_LIMITED]":
        return content.strip(), "scope_limited"
    if separator and marker == "[NORVII_GROUNDED]":
        return content.strip(), "grounded"
    return answer, "grounded"
