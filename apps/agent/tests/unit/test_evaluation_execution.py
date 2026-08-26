"""Tests for the fixed-snapshot evaluation execution boundary."""

from __future__ import annotations

from collections.abc import Callable
from uuid import UUID

import pytest

from norvii_agent.evaluation import (
    EvaluationContractError,
    EvaluationExecutor,
    EvaluationGeneration,
    EvaluationRequest,
    EvidenceScopeError,
    ExecutionIdentity,
    FrozenIdentityUnavailableError,
    FrozenRetrievalConfiguration,
)
from norvii_agent.graph import Evidence

CORPUS_ID = UUID("10000000-0000-4000-8000-000000000001")
HISTORICAL_SNAPSHOT_ID = UUID("20000000-0000-4000-8000-000000000001")
ACTIVE_SNAPSHOT_ID = UUID("20000000-0000-4000-8000-000000000002")
EXECUTION_IDENTITY = ExecutionIdentity(
    "agent-build-test", "synthetic-evaluation-model", "synthetic-embedding-model"
)
RUNNABLE_CONFIGURATION = FrozenRetrievalConfiguration("vector", "f" * 64)


class SnapshotRetriever:
    def __init__(self, evidence_by_snapshot: dict[UUID, tuple[Evidence, ...]]) -> None:
        self._evidence_by_snapshot = evidence_by_snapshot
        self.active_snapshot_id = ACTIVE_SNAPSHOT_ID
        self.requests: list[tuple[UUID, UUID, str, str]] = []

    def search(
        self, corpus_id: UUID, snapshot_id: UUID, question: str, strategy: str = "vector"
    ) -> tuple[Evidence, ...]:
        self.requests.append((corpus_id, snapshot_id, question, strategy))
        return self._evidence_by_snapshot[snapshot_id]


class NonStreamingModel:
    def __init__(self, on_generate: Callable[[], None] | None = None) -> None:
        self.calls: list[tuple[str, tuple[Evidence, ...], str]] = []
        self._on_generate = on_generate

    def generate(
        self, question: str, evidence: tuple[Evidence, ...], interface_language: str
    ) -> EvaluationGeneration:
        if self._on_generate is not None:
            self._on_generate()
        self.calls.append((question, evidence, interface_language))
        return EvaluationGeneration(
            answer="The historical rule applies [1].",
            outcome="completed",
            model_identity="synthetic-evaluation-model",
            input_tokens=13,
            output_tokens=7,
        )


def test_evaluation_keeps_historical_snapshot_after_active_release_changes() -> None:
    historical_evidence = _evidence("historical-evidence", HISTORICAL_SNAPSHOT_ID)
    retriever = SnapshotRetriever(
        {
            HISTORICAL_SNAPSHOT_ID: (historical_evidence,),
            ACTIVE_SNAPSHOT_ID: (_evidence("active-evidence", ACTIVE_SNAPSHOT_ID),),
        }
    )
    model = NonStreamingModel(
        on_generate=lambda: setattr(
            retriever,
            "active_snapshot_id",
            UUID("20000000-0000-4000-8000-000000000003"),
        )
    )
    executor = _executor(retriever, model)

    result = executor.execute(_request(HISTORICAL_SNAPSHOT_ID))

    assert retriever.active_snapshot_id == UUID("20000000-0000-4000-8000-000000000003")
    assert [item.id for item in result.retrieved_evidence] == ["historical-evidence"]
    assert retriever.requests == [
        (CORPUS_ID, HISTORICAL_SNAPSHOT_ID, "Which historical rule applies?", "vector")
    ]
    assert model.calls == [
        (
            "Which historical rule applies?",
            (historical_evidence,),
            "en",
        )
    ]
    assert result.citation_marker_inputs[0].marker_position == 1
    assert result.citation_marker_inputs[0].evidence == historical_evidence
    assert result.model_identity == "synthetic-evaluation-model"
    assert result.agent_build_identity == "agent-build-test"
    assert result.embedding_model_identity == "synthetic-embedding-model"
    assert result.telemetry.input_tokens == 13
    assert result.telemetry.output_tokens == 7


def test_evaluation_preserves_retrieval_order_for_one_based_citation_markers() -> None:
    first_evidence = _evidence("first", HISTORICAL_SNAPSHOT_ID)
    second_evidence = _evidence("second", HISTORICAL_SNAPSHOT_ID)
    third_evidence = _evidence("third", HISTORICAL_SNAPSHOT_ID)
    model = NonStreamingModel()
    executor = _executor(
        SnapshotRetriever(
            {HISTORICAL_SNAPSHOT_ID: (second_evidence, first_evidence, third_evidence)}
        ),
        model,
    )

    result = executor.execute(_request(HISTORICAL_SNAPSHOT_ID))

    assert result.retrieved_evidence == (second_evidence, first_evidence, third_evidence)
    assert [(item.marker_position, item.evidence) for item in result.citation_marker_inputs] == [
        (1, second_evidence),
        (2, first_evidence),
        (3, third_evidence),
    ]
    assert model.calls[0][1] == (second_evidence, first_evidence, third_evidence)


@pytest.mark.parametrize("token_count", [-1, True, False, 1.5, "1"])
@pytest.mark.parametrize("token_field", ["input_tokens", "output_tokens"])
def test_evaluation_generation_rejects_invalid_token_counts(
    token_count: object, token_field: str
) -> None:
    generation = {
        "answer": "A bounded answer.",
        "outcome": "completed",
        "model_identity": "synthetic-evaluation-model",
        "input_tokens": 13,
        "output_tokens": 7,
    }
    generation[token_field] = token_count

    with pytest.raises(EvaluationContractError, match="non-negative integer"):
        EvaluationGeneration(**generation)


@pytest.mark.parametrize(
    ("identifier", "evidence_corpus_id", "evidence_snapshot_id"),
    [
        (
            "other-corpus",
            UUID("10000000-0000-4000-8000-000000000099"),
            HISTORICAL_SNAPSHOT_ID,
        ),
        ("other-snapshot", CORPUS_ID, ACTIVE_SNAPSHOT_ID),
    ],
)
def test_evaluation_rejects_cross_boundary_retrieved_evidence(
    identifier: str,
    evidence_corpus_id: UUID,
    evidence_snapshot_id: UUID,
) -> None:
    model = NonStreamingModel()
    invalid_evidence = _evidence(
        identifier,
        evidence_snapshot_id,
        corpus_id=evidence_corpus_id,
    )
    executor = _executor(SnapshotRetriever({HISTORICAL_SNAPSHOT_ID: (invalid_evidence,)}), model)

    with pytest.raises(EvidenceScopeError, match="outside the evaluation corpus snapshot"):
        executor.execute(_request(HISTORICAL_SNAPSHOT_ID))

    assert model.calls == []


def test_evaluation_uses_only_the_non_streaming_model_port() -> None:
    model = NonStreamingModel()
    executor = _executor(
        SnapshotRetriever({HISTORICAL_SNAPSHOT_ID: (_evidence("fixed", HISTORICAL_SNAPSHOT_ID),)}),
        model,
    )

    result = executor.execute(_request(HISTORICAL_SNAPSHOT_ID))

    assert result.outcome == "completed"
    assert model.calls == [
        (
            "Which historical rule applies?",
            (_evidence("fixed", HISTORICAL_SNAPSHOT_ID),),
            "en",
        )
    ]


def _request(snapshot_id: UUID) -> EvaluationRequest:
    return EvaluationRequest(
        corpus_id=CORPUS_ID,
        snapshot_id=snapshot_id,
        question="Which historical rule applies?",
        interface_language="en",
        retrieval_configuration=RUNNABLE_CONFIGURATION,
        execution_identity=EXECUTION_IDENTITY,
    )


def _executor(retriever: SnapshotRetriever, model: NonStreamingModel) -> EvaluationExecutor:
    return EvaluationExecutor(retriever, model, EXECUTION_IDENTITY, RUNNABLE_CONFIGURATION)


def test_evaluation_rejects_an_unavailable_frozen_identity_before_retrieval() -> None:
    retriever = SnapshotRetriever(
        {HISTORICAL_SNAPSHOT_ID: (_evidence("fixed", HISTORICAL_SNAPSHOT_ID),)}
    )
    request = EvaluationRequest(
        corpus_id=CORPUS_ID,
        snapshot_id=HISTORICAL_SNAPSHOT_ID,
        question="Which historical rule applies?",
        interface_language="en",
        retrieval_configuration=FrozenRetrievalConfiguration("vector", "a" * 64),
        execution_identity=EXECUTION_IDENTITY,
    )

    with pytest.raises(FrozenIdentityUnavailableError, match="frozen retrieval configuration"):
        _executor(retriever, NonStreamingModel()).execute(request)

    assert retriever.requests == []


def _evidence(identifier: str, snapshot_id: UUID, *, corpus_id: UUID = CORPUS_ID) -> Evidence:
    return Evidence(
        identifier,
        corpus_id,
        UUID("30000000-0000-4000-8000-000000000001"),
        UUID("40000000-0000-4000-8000-000000000001"),
        "article:1",
        0,
        20,
        "Synthetic evidence.",
        1,
        snapshot_id=snapshot_id,
    )
