"""Fixed-snapshot, non-streaming execution contract for evaluation cases."""

from __future__ import annotations

import re
from dataclasses import dataclass
from time import perf_counter
from typing import TYPE_CHECKING, Literal, Protocol

if TYPE_CHECKING:
    from uuid import UUID

    from norvii_agent.graph import Evidence, RetrievalPort

EvaluationOutcome = Literal["completed", "abstained"]
GraphGroundingStatus = Literal["not_requested", "not_used", "grounded"]


class EvaluationContractError(ValueError):
    """Raised when an evaluation request or model result violates its contract."""


class EvidenceScopeError(EvaluationContractError):
    """Raised when retrieved evidence is outside the requested immutable snapshot."""


class FrozenIdentityUnavailableError(EvaluationContractError):
    """Raised when this worker cannot execute a persisted evaluation identity exactly."""


_SHA256 = re.compile(r"^[0-9a-f]{64}$")


@dataclass(frozen=True, slots=True)
class FrozenRetrievalConfiguration:
    """Validated retrieval settings persisted with one immutable evaluation run."""

    strategy: Literal["vector", "hybrid"]
    fingerprint: str

    def __post_init__(self) -> None:
        """Reject an unidentifiable configuration at the execution boundary."""
        if self.strategy not in {"vector", "hybrid"}:
            raise EvaluationContractError("retrieval strategy is unsupported")
        if _SHA256.fullmatch(self.fingerprint) is None:
            raise EvaluationContractError("retrieval configuration fingerprint is invalid")


@dataclass(frozen=True, slots=True)
class ExecutionIdentity:
    """Immutable evaluator identities persisted with one evaluation run."""

    agent_build: str
    chat_model_identity: str
    embedding_model_identity: str

    def __post_init__(self) -> None:
        """Reject incomplete identities before they can select a runtime configuration."""
        if not all(
            value.strip()
            for value in (
                self.agent_build,
                self.chat_model_identity,
                self.embedding_model_identity,
            )
        ):
            raise EvaluationContractError("evaluation execution identity is required")


@dataclass(frozen=True, slots=True)
class EvaluationRequest:
    """One case execution against a caller-selected immutable corpus snapshot."""

    corpus_id: UUID
    snapshot_id: UUID
    question: str
    interface_language: Literal["en", "pt"]
    retrieval_configuration: FrozenRetrievalConfiguration
    execution_identity: ExecutionIdentity

    def __post_init__(self) -> None:
        """Reject a blank case question before retrieval or model execution."""
        if self.interface_language not in {"en", "pt"}:
            raise EvaluationContractError("evaluation interface language is unsupported")
        if not self.question.strip():
            raise EvaluationContractError("evaluation question is required")


@dataclass(frozen=True, slots=True)
class EvaluationGeneration:
    """Safe, non-streaming model completion supplied to the evaluator."""

    answer: str
    outcome: EvaluationOutcome
    model_identity: str
    input_tokens: int | None = None
    output_tokens: int | None = None

    def __post_init__(self) -> None:
        """Ensure terminal generation data is safe to persist with a run case."""
        if self.outcome not in {"completed", "abstained"}:
            raise EvaluationContractError("evaluation outcome is unsupported")
        if not self.model_identity.strip():
            raise EvaluationContractError("model identity is required")
        if self.outcome == "completed" and not self.answer.strip():
            raise EvaluationContractError("completed evaluation answer is required")
        _validate_token_count(self.input_tokens)
        _validate_token_count(self.output_tokens)


class EvaluationModelPort(Protocol):
    """Generate one evaluation result without public chat stream callbacks."""

    def generate(
        self,
        question: str,
        evidence: tuple[Evidence, ...],
        interface_language: Literal["en", "pt"],
    ) -> EvaluationGeneration:
        """Return a terminal generation without exposing provider payloads."""
        ...


@dataclass(frozen=True, slots=True)
class CitationMarkerInput:
    """One marker position paired with immutable retrieved evidence."""

    marker_position: int
    evidence: Evidence


@dataclass(frozen=True, slots=True)
class GraphGrounding:
    """Safe graph provenance state for an evaluation result."""

    status: GraphGroundingStatus


@dataclass(frozen=True, slots=True)
class EvaluationTelemetry:
    """Nullable duration and token measurements owned by the execution boundary."""

    retrieval_milliseconds: int | None
    generation_milliseconds: int | None
    total_milliseconds: int | None
    input_tokens: int | None
    output_tokens: int | None


@dataclass(frozen=True, slots=True)
class EvaluationResult:
    """Terminal safe output for one evaluation case."""

    answer: str
    outcome: EvaluationOutcome
    retrieved_evidence: tuple[Evidence, ...]
    citation_marker_inputs: tuple[CitationMarkerInput, ...]
    graph_grounding: GraphGrounding
    model_identity: str
    agent_build_identity: str
    embedding_model_identity: str
    telemetry: EvaluationTelemetry


class EvaluationExecutor:
    """Execute one case through explicitly snapshot-scoped retrieval and a model port."""

    def __init__(
        self,
        retriever: RetrievalPort,
        model: EvaluationModelPort,
        execution_identity: ExecutionIdentity,
        runnable_retrieval_configuration: FrozenRetrievalConfiguration,
    ) -> None:
        self._retriever = retriever
        self._model = model
        self._execution_identity = execution_identity
        self._runnable_retrieval_configuration = runnable_retrieval_configuration

    def execute(self, request: EvaluationRequest) -> EvaluationResult:
        """Run one case without resolving or consulting a current active snapshot."""
        self._validate_runnable_identity(request)
        started = perf_counter()
        retrieval_started = perf_counter()
        evidence = tuple(
            self._retriever.search(
                request.corpus_id,
                request.snapshot_id,
                request.question,
                request.retrieval_configuration.strategy,
            )
        )
        _validate_evidence_scope(evidence, request)
        retrieval_milliseconds = _elapsed_milliseconds(retrieval_started)

        generation_started = perf_counter()
        generation = self._model.generate(
            request.question,
            evidence,
            request.interface_language,
        )
        if generation.model_identity != request.execution_identity.chat_model_identity:
            raise FrozenIdentityUnavailableError(
                "the configured chat model does not match the frozen identity"
            )
        generation_milliseconds = _elapsed_milliseconds(generation_started)
        return EvaluationResult(
            answer=generation.answer,
            outcome=generation.outcome,
            retrieved_evidence=evidence,
            citation_marker_inputs=tuple(
                CitationMarkerInput(position, item)
                for position, item in enumerate(evidence, start=1)
            ),
            graph_grounding=_graph_grounding(evidence, request.retrieval_configuration.strategy),
            model_identity=generation.model_identity,
            agent_build_identity=request.execution_identity.agent_build,
            embedding_model_identity=request.execution_identity.embedding_model_identity,
            telemetry=EvaluationTelemetry(
                retrieval_milliseconds=retrieval_milliseconds,
                generation_milliseconds=generation_milliseconds,
                total_milliseconds=_elapsed_milliseconds(started),
                input_tokens=generation.input_tokens,
                output_tokens=generation.output_tokens,
            ),
        )

    def _validate_runnable_identity(self, request: EvaluationRequest) -> None:
        """Fail closed unless the exact persisted runtime configuration is available."""
        if request.execution_identity != self._execution_identity:
            raise FrozenIdentityUnavailableError(
                "the worker cannot serve the frozen execution identity"
            )
        if request.retrieval_configuration != self._runnable_retrieval_configuration:
            raise FrozenIdentityUnavailableError(
                "the worker cannot serve the frozen retrieval configuration"
            )


def _validate_evidence_scope(evidence: tuple[Evidence, ...], request: EvaluationRequest) -> None:
    """Reject results that cannot be persisted under the requested corpus snapshot."""
    for item in evidence:
        if item.corpus_id != request.corpus_id or item.snapshot_id != request.snapshot_id:
            raise EvidenceScopeError("retrieved evidence is outside the evaluation corpus snapshot")


def _validate_token_count(token_count: int | None) -> None:
    """Reject telemetry values that cannot represent a non-negative token count."""
    if token_count is not None and (type(token_count) is not int or token_count < 0):
        raise EvaluationContractError("evaluation token count must be a non-negative integer")


def _graph_grounding(evidence: tuple[Evidence, ...], strategy: str) -> GraphGrounding:
    """Derive graph grounding only from the returned snapshot-scoped evidence."""
    if strategy != "hybrid":
        return GraphGrounding("not_requested")
    graph_contribution = {"graph", "vector_and_graph"}
    return GraphGrounding(
        "grounded"
        if any(item.contribution in graph_contribution for item in evidence)
        else "not_used"
    )


def _elapsed_milliseconds(started: float) -> int:
    """Return a non-negative monotonic elapsed duration."""
    return max(0, round((perf_counter() - started) * 1000))
