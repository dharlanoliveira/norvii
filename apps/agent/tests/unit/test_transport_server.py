from __future__ import annotations

import json
from collections.abc import Callable
from http.client import HTTPConnection
from threading import Thread

from norvii_agent.graph import (
    AnswerInspection,
    ExecutionMeasurements,
    GroundedChatRequest,
    GroundedChatResult,
    RetrievalInspection,
    RetrievalStage,
)
from norvii_agent.transport.server import AgentHTTPServer, _inspection


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
