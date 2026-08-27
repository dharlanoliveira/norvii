from __future__ import annotations

import json
from collections.abc import Callable
from http.client import HTTPConnection
from threading import Thread
from uuid import UUID

from norvii_agent.evaluation import (
    EvaluationExecutor,
    EvaluationGeneration,
    ExecutionIdentity,
    FrozenRetrievalConfiguration,
)
from norvii_agent.graph import (
    AnswerInspection,
    AssertionPathStep,
    Evidence,
    ExecutionMeasurements,
    GroundedChatRequest,
    GroundedChatResult,
    RetrievalInspection,
    RetrievalStage,
)
from norvii_agent.transport.server import AgentHTTPServer, _inspection, _telemetry


class EmptyEvidenceGraph:
    def run(
        self,
        request: GroundedChatRequest,
        emit: Callable[[str], None],
    ) -> GroundedChatResult:
        assert request.question
        del emit
        return GroundedChatResult(
            status="abstained",
            answer="",
            evidence=(),
            reason="insufficient_evidence",
        )


class FailingGraph:
    def run(
        self,
        request: GroundedChatRequest,
        emit: Callable[[str], None],
    ) -> GroundedChatResult:
        del request, emit
        raise RuntimeError("provider rejected diagnostic-token and private-request-content")


class FixedSnapshotRetriever:
    def __init__(self) -> None:
        self.requests: list[tuple[UUID, UUID, str, str]] = []

    def search(
        self, corpus_id: UUID, snapshot_id: UUID, question: str, strategy: str = "vector"
    ) -> tuple[Evidence, ...]:
        self.requests.append((corpus_id, snapshot_id, question, strategy))
        return (
            Evidence(
                id="evidence-1",
                corpus_id=corpus_id,
                source_id=UUID("30000000-0000-4000-8000-000000000001"),
                document_id=UUID("40000000-0000-4000-8000-000000000001"),
                unit_locator="Article 1",
                start_offset=0,
                end_offset=20,
                excerpt="The fixed snapshot applies.",
                rank=1,
                document_version_id=UUID("40000000-0000-4000-8000-000000000001"),
                source_revision_id=UUID("50000000-0000-4000-8000-000000000001"),
                snapshot_id=snapshot_id,
                unit_id=UUID("60000000-0000-4000-8000-000000000001"),
                canonical_locator="article:1",
                content_sha256="a" * 64,
            ),
        )


class FixedEvaluationModel:
    def generate(
        self, question: str, evidence: tuple[Evidence, ...], interface_language: str
    ) -> EvaluationGeneration:
        assert question == "Which fixed rule applies?"
        assert evidence[0].snapshot_id == UUID("20000000-0000-4000-8000-000000000001")
        assert interface_language == "en"
        return EvaluationGeneration(
            answer="The fixed snapshot applies [1].",
            outcome="completed",
            model_identity="fixed-evaluation-model",
            input_tokens=3,
            output_tokens=5,
        )


def test_chat_stream_reads_only_the_declared_request_body_length() -> None:
    server = AgentHTTPServer(("127.0.0.1", 0), EmptyEvidenceGraph)
    server.daemon_threads = True
    thread = Thread(target=server.serve_forever, daemon=True)
    thread.start()
    connection = HTTPConnection("127.0.0.1", int(server.server_address[1]), timeout=1)

    try:
        request_body = json.dumps(
            {
                "question": "What applies?",
                "interfaceLanguage": "en",
                "snapshotId": "50000000-0000-4000-8000-000000000001",
            }
        )
        connection.request(
            "POST",
            "/v1/corpora/10000000-0000-4000-8000-000000000001/chat/stream",
            body=request_body,
            headers={"Content-Type": "application/json"},
        )

        response = connection.getresponse()

        assert response.status == 200
        assert response.readline() == b"event: started\n"
    finally:
        connection.close()
        server.shutdown()
        server.server_close()
        thread.join()


def test_terminal_inspection_serializes_measurements_and_evidence_metadata() -> None:
    inspection = AnswerInspection(
        outcome="completed",
        retrieval=RetrievalInspection("vector", 8, 1, "text-embedding-3-small"),
        measurements=ExecutionMeasurements(12, 34, None, 10, None),
        evidence=(),
        stages=(RetrievalStage("vector", "completed", 1, 12),),
        assertion_path=(
            AssertionPathStep(
                "assertion-1",
                "imposes_duty_on",
                "Controller",
                "Data protection officer",
                "article-41",
                "article-41-item-2",
                ("chapter-9", "article-41"),
                None,
            ),
        ),
        scope_locator="chapter-9",
    )

    payload = _inspection(inspection)

    assert payload["outcome"] == "completed"
    assert payload["retrieval"] == {
        "strategy": "vector",
        "topK": 8,
        "returnedCount": 1,
        "embeddingModel": "text-embedding-3-small",
    }
    assert payload["measurements"] == {
        "retrievalMilliseconds": 12,
        "generationMilliseconds": 34,
        "totalMilliseconds": None,
        "inputTokens": 10,
        "outputTokens": None,
    }
    assert payload["evidence"] == []
    assert payload["assertionPath"] == [
        {
            "assertionId": "assertion-1",
            "predicate": "imposes_duty_on",
            "subjectLabel": "Controller",
            "objectLabel": "Data protection officer",
            "establishingLocator": "article-41",
            "evidenceLocator": "article-41-item-2",
            "hierarchyContext": ["chapter-9", "article-41"],
            "qualifier": None,
        }
    ]
    assert payload["scopeLocator"] == "chapter-9"
    assert payload["stages"] == [
        {
            "name": "vector",
            "state": "completed",
            "evidenceCount": 1,
            "durationMilliseconds": 12,
            "reasonCode": None,
            "inputTokens": None,
            "outputTokens": None,
        }
    ]


def test_terminal_telemetry_bounds_the_evidence_counter() -> None:
    assert _telemetry("completed", 99) == {
        "outcome": "completed",
        "evidenceCount": 8,
        "durationMilliseconds": 0,
    }
    assert _telemetry("failed", -1)["evidenceCount"] == 0


def test_provider_diagnostics_and_request_content_are_not_exposed_in_terminal_error() -> None:
    server = AgentHTTPServer(("127.0.0.1", 0), FailingGraph)
    server.daemon_threads = True
    thread = Thread(target=server.serve_forever, daemon=True)
    thread.start()
    connection = HTTPConnection("127.0.0.1", int(server.server_address[1]), timeout=1)

    try:
        request_body = json.dumps(
            {
                "question": "private-request-content",
                "interfaceLanguage": "en",
                "snapshotId": "50000000-0000-4000-8000-000000000001",
            }
        )
        connection.request(
            "POST",
            "/v1/corpora/10000000-0000-4000-8000-000000000001/chat/stream",
            body=request_body,
            headers={"Content-Type": "application/json"},
        )

        response = connection.getresponse()
        body = response.read().decode()

        assert response.status == 200
        assert "event: error" in body
        assert '"code": "generation_failed"' in body
        assert "diagnostic-token" not in body
        assert "private-request-content" not in body
    finally:
        connection.close()
        server.shutdown()
        server.server_close()
        thread.join()


def test_evaluation_endpoint_is_strict_json_and_preserves_fixed_provenance() -> None:
    retriever = FixedSnapshotRetriever()
    server = AgentHTTPServer(
        ("127.0.0.1", 0),
        EmptyEvidenceGraph,
        lambda: _fixed_evaluation_executor(retriever),
    )
    server.daemon_threads = True
    thread = Thread(target=server.serve_forever, daemon=True)
    thread.start()
    connection = HTTPConnection("127.0.0.1", int(server.server_address[1]), timeout=1)

    try:
        request_body = json.dumps(
            {
                "corpusId": "10000000-0000-4000-8000-000000000001",
                "snapshotId": "20000000-0000-4000-8000-000000000001",
                "question": "Which fixed rule applies?",
                "interfaceLanguage": "en",
                "retrievalConfiguration": {
                    "strategy": "vector",
                    "fingerprint": "f" * 64,
                },
                "executionIdentity": {
                    "agentBuild": "agent-build-test",
                    "chatModelIdentity": "fixed-evaluation-model",
                    "embeddingModelIdentity": "fixed-embedding-model",
                },
            }
        )
        connection.request(
            "POST",
            "/v1/evaluations/execute",
            body=request_body,
            headers={"Content-Type": "application/json"},
        )

        response = connection.getresponse()
        payload = json.loads(response.read())

        assert response.status == 200
        assert response.getheader("Content-Type") == "application/json"
        assert payload["answer"] == "The fixed snapshot applies [1]."
        assert payload["citationMarkerInputs"] == [{"markerPosition": 1, "evidenceRank": 1}]
        assert payload["retrievedEvidence"] == [
            {
                "rank": 1,
                "corpusId": "10000000-0000-4000-8000-000000000001",
                "snapshotId": "20000000-0000-4000-8000-000000000001",
                "sourceId": "30000000-0000-4000-8000-000000000001",
                "sourceRevisionId": "50000000-0000-4000-8000-000000000001",
                "documentId": "40000000-0000-4000-8000-000000000001",
                "unitId": "60000000-0000-4000-8000-000000000001",
                "canonicalLocator": "article:1",
                "startOffset": 0,
                "endOffset": 20,
                "contentSha256": "a" * 64,
            }
        ]
        assert retriever.requests == [
            (
                UUID("10000000-0000-4000-8000-000000000001"),
                UUID("20000000-0000-4000-8000-000000000001"),
                "Which fixed rule applies?",
                "vector",
            )
        ]
    finally:
        connection.close()
        server.shutdown()
        server.server_close()
        thread.join()


def test_evaluation_endpoint_rejects_unknown_fixed_snapshot_fields() -> None:
    server = AgentHTTPServer(
        ("127.0.0.1", 0),
        EmptyEvidenceGraph,
        lambda: _fixed_evaluation_executor(FixedSnapshotRetriever()),
    )
    server.daemon_threads = True
    thread = Thread(target=server.serve_forever, daemon=True)
    thread.start()
    connection = HTTPConnection("127.0.0.1", int(server.server_address[1]), timeout=1)

    try:
        connection.request(
            "POST",
            "/v1/evaluations/execute",
            body=json.dumps({"question": "invalid", "stream": True}),
            headers={"Content-Type": "application/json"},
        )
        response = connection.getresponse()

        assert response.status == 400
        assert response.getheader("Content-Type") == "application/json"
        assert response.read() == b'{"code":"invalid_request"}'
    finally:
        connection.close()
        server.shutdown()
        server.server_close()
        thread.join()


def test_evaluation_endpoint_fails_closed_for_an_unavailable_frozen_configuration() -> None:
    retriever = FixedSnapshotRetriever()
    server = AgentHTTPServer(
        ("127.0.0.1", 0), EmptyEvidenceGraph, lambda: _fixed_evaluation_executor(retriever)
    )
    server.daemon_threads = True
    thread = Thread(target=server.serve_forever, daemon=True)
    thread.start()
    connection = HTTPConnection("127.0.0.1", int(server.server_address[1]), timeout=1)

    try:
        connection.request(
            "POST",
            "/v1/evaluations/execute",
            body=json.dumps(
                {
                    "corpusId": "10000000-0000-4000-8000-000000000001",
                    "snapshotId": "20000000-0000-4000-8000-000000000001",
                    "question": "Which fixed rule applies?",
                    "interfaceLanguage": "en",
                    "retrievalConfiguration": {"strategy": "vector", "fingerprint": "a" * 64},
                    "executionIdentity": {
                        "agentBuild": "agent-build-test",
                        "chatModelIdentity": "fixed-evaluation-model",
                        "embeddingModelIdentity": "fixed-embedding-model",
                    },
                }
            ),
            headers={"Content-Type": "application/json"},
        )
        response = connection.getresponse()

        assert response.status == 409
        assert response.read() == b'{"code":"frozen_identity_unavailable"}'
        assert retriever.requests == []
    finally:
        connection.close()
        server.shutdown()
        server.server_close()
        thread.join()


def _fixed_evaluation_executor(retriever: FixedSnapshotRetriever) -> EvaluationExecutor:
    return EvaluationExecutor(
        retriever,
        FixedEvaluationModel(),
        ExecutionIdentity("agent-build-test", "fixed-evaluation-model", "fixed-embedding-model"),
        FrozenRetrievalConfiguration("vector", "f" * 64),
    )
