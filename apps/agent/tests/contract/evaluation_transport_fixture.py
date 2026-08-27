"""Serve a deterministic Python evaluation transport fixture for the Go contract test."""

from __future__ import annotations

import signal
from collections.abc import Callable
from typing import Literal, cast
from uuid import UUID

from norvii_agent.config import AgentConfig
from norvii_agent.evaluation import (
    EvaluationExecutor,
    EvaluationGeneration,
    ExecutionIdentity,
    FrozenRetrievalConfiguration,
)
from norvii_agent.graph import Evidence, GroundedChatGraph, GroundedChatRequest, GroundedChatResult
from norvii_agent.transport.server import AgentHTTPServer


class _UnusedChatGraph:
    """Make any accidental public-chat routing fail the cross-language contract test."""

    def run(self, request: GroundedChatRequest, emit: Callable[[str], None]) -> GroundedChatResult:
        del request, emit
        raise AssertionError("the evaluation transport must not invoke the public chat graph")


class _FixedSnapshotRetriever:
    """Return one complete provenance record only for the expected immutable snapshot."""

    def search(
        self, corpus_id: UUID, snapshot_id: UUID, question: str, strategy: str = "vector"
    ) -> tuple[Evidence, ...]:
        if (
            corpus_id != UUID("10000000-0000-4000-8000-000000000001")
            or snapshot_id != UUID("20000000-0000-4000-8000-000000000001")
            or question != "Which fixed rule applies?"
            or strategy != "vector"
        ):
            raise ValueError("the fixed snapshot request is invalid")
        return (
            Evidence(
                id="evidence-1",
                corpus_id=corpus_id,
                source_id=UUID("30000000-0000-4000-8000-000000000001"),
                document_id=UUID("40000000-0000-4000-8000-000000000001"),
                unit_locator="Article 1",
                start_offset=0,
                end_offset=10,
                excerpt="The fixed snapshot applies.",
                rank=1,
                document_version_id=UUID("40000000-0000-4000-8000-000000000001"),
                source_revision_id=UUID("50000000-0000-4000-8000-000000000001"),
                snapshot_id=snapshot_id,
                unit_id=UUID("60000000-0000-4000-8000-000000000001"),
                canonical_locator="article:1/item:a",
                content_sha256="a" * 64,
            ),
        )


class _FixedEvaluationModel:
    """Provide a terminal non-streaming completion without a provider dependency."""

    def generate(
        self, question: str, evidence: tuple[Evidence, ...], interface_language: str
    ) -> EvaluationGeneration:
        if (
            question != "Which fixed rule applies?"
            or len(evidence) != 1
            or interface_language != "en"
        ):
            raise ValueError("the fixed evaluation generation request is invalid")
        return EvaluationGeneration(
            answer="The fixed snapshot applies [1].",
            outcome="completed",
            model_identity="test-model",
            input_tokens=7,
        )


def main() -> None:
    """Print the listening port, then serve until the parent contract test stops the process."""
    configuration = AgentConfig.from_environment()
    server = AgentHTTPServer(
        ("127.0.0.1", 0),
        cast("Callable[[], GroundedChatGraph]", _UnusedChatGraph),
        lambda: EvaluationExecutor(
            _FixedSnapshotRetriever(),
            _FixedEvaluationModel(),
            ExecutionIdentity(
                configuration.evaluation_agent_build,
                configuration.chat_model,
                configuration.embedding_model,
            ),
            FrozenRetrievalConfiguration(
                cast("Literal['vector', 'hybrid']", configuration.evaluation_retrieval_strategy),
                configuration.evaluation_retrieval_fingerprint,
            ),
        ),
    )
    signal.signal(signal.SIGTERM, _shutdown(server))
    print(server.server_address[1], flush=True)
    try:
        server.serve_forever()
    finally:
        server.server_close()


def _shutdown(server: AgentHTTPServer) -> Callable[[int, object], None]:
    """Create the signal callback without exposing process control to the test request path."""

    def handler(_signal: int, _frame: object) -> None:
        server.shutdown()

    return handler


if __name__ == "__main__":
    main()
