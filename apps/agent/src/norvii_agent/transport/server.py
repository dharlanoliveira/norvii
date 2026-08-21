"""Small dependency-light HTTP server for the LangGraph service."""

from __future__ import annotations

import json
import re
from http.server import BaseHTTPRequestHandler, ThreadingHTTPServer
from typing import TYPE_CHECKING
from uuid import UUID, uuid4

from norvii_agent.graph import GroundedChatRequest

if TYPE_CHECKING:
    from collections.abc import Callable

    from norvii_agent.graph import Evidence, GroundedChatGraph

_CORPUS_PATH = re.compile(r"^/v1/corpora/(?P<corpus>[0-9a-f-]+)/chat/stream$")
_MAX_REQUEST_BODY_BYTES = 64 * 1024


class InvalidRequestBodyError(ValueError):
    """Raised when an internal request body cannot be read safely."""


class ClientDisconnectedError(ConnectionError):
    """Raised when an SSE client disconnects before the stream is complete."""


class AgentHTTPServer(ThreadingHTTPServer):
    """HTTP server carrying one graph factory."""

    def __init__(
        self, address: tuple[str, int], graph_factory: Callable[[], GroundedChatGraph]
    ) -> None:
        self.graph_factory = graph_factory
        super().__init__(address, AgentRequestHandler)


class AgentRequestHandler(BaseHTTPRequestHandler):
    """Translate one internal request into the feature-local event stream."""

    server: AgentHTTPServer

    def do_GET(self) -> None:
        """Expose a content-free readiness endpoint."""
        if self.path != "/healthz":
            self.send_error(404)
            return
        self.send_response(200)
        self.send_header("Content-Type", "application/json")
        self.end_headers()
        self.wfile.write(b'{"status":"ok"}')

    def do_POST(self) -> None:
        """Handle one graph stream request."""
        match = _CORPUS_PATH.fullmatch(self.path)
        if match is None:
            self.send_error(404)
            return
        try:
            corpus_id = UUID(match.group("corpus"))
            payload = self._read_json_payload()
            question = str(payload["question"]).strip()
            interface_language = str(payload.get("interfaceLanguage", "en")).strip().lower()
            if interface_language not in {"en", "pt"}:
                self.send_error(400, "invalid interface language")
                return
            if not question:
                self._invalid_question()
        except InvalidRequestBodyError:
            self.send_error(400, "invalid request body")
            return
        except (ValueError, KeyError, TypeError, json.JSONDecodeError):
            self.send_error(400, "invalid question")
            return

        self._stream_graph(corpus_id, question, interface_language)

    def _stream_graph(self, corpus_id: UUID, question: str, interface_language: str) -> None:
        request_id = uuid4()
        self.send_response(200)
        self.send_header("Content-Type", "text/event-stream")
        self.send_header("Cache-Control", "no-cache")
        self.send_header("Connection", "close")
        self.end_headers()
        self.close_connection = True
        try:
            self._event(
                "started",
                {"type": "started", "requestId": str(request_id), "corpusId": str(corpus_id)},
            )
            deltas: list[str] = []
            result = self.server.graph_factory().run(
                GroundedChatRequest(corpus_id, question, interface_language), deltas.append
            )
            self._event(
                "evidence",
                {
                    "type": "evidence",
                    "requestId": str(request_id),
                    "references": [_reference(item) for item in result.evidence],
                },
            )
            for delta in deltas:
                self._event("delta", {"type": "delta", "requestId": str(request_id), "text": delta})
            if result.status == "completed":
                self._event(
                    "completed",
                    {
                        "type": "completed",
                        "requestId": str(request_id),
                        "answer": result.answer,
                        "references": [_reference(item) for item in result.evidence],
                        "telemetry": _telemetry("completed", len(result.evidence)),
                    },
                )
            else:
                self._event(
                    "abstained",
                    {
                        "type": "abstained",
                        "requestId": str(request_id),
                        "reason": result.reason or "insufficient_evidence",
                        "telemetry": _telemetry("abstained", len(result.evidence)),
                    },
                )
        except ClientDisconnectedError:
            return
        except Exception:  # noqa: BLE001 - convert provider failures to safe events
            try:
                self._event(
                    "error",
                    {
                        "type": "error",
                        "requestId": str(request_id),
                        "code": "generation_failed",
                        "message": "The grounded graph could not be completed.",
                        "telemetry": _telemetry("failed", 0),
                    },
                )
            except ClientDisconnectedError:
                return

    def _read_json_payload(self) -> dict[str, object]:
        content_length = self.headers.get("Content-Length")
        if content_length is None:
            raise InvalidRequestBodyError("missing content length")

        try:
            body_length = int(content_length)
        except ValueError as error:
            raise InvalidRequestBodyError("invalid content length") from error

        if body_length <= 0 or body_length > _MAX_REQUEST_BODY_BYTES:
            raise InvalidRequestBodyError("request body length is outside the allowed range")

        body = self.rfile.read(body_length)
        if len(body) != body_length:
            raise InvalidRequestBodyError("request body ended before the declared content length")

        payload = json.loads(body)
        if not isinstance(payload, dict):
            raise InvalidRequestBodyError("request body must be a JSON object")
        return payload

    def _invalid_question(self) -> None:
        raise ValueError("question must not be empty")

    def log_message(self, _format: str, *_args: object) -> None:
        """Suppress request payloads from default stderr logging."""

    def _event(self, name: str, payload: dict[str, object]) -> None:
        try:
            self.wfile.write(f"event: {name}\ndata: {json.dumps(payload)}\n\n".encode())
            self.wfile.flush()
        except (BrokenPipeError, ConnectionResetError) as error:
            raise ClientDisconnectedError from error


def _reference(evidence: Evidence) -> dict[str, object]:
    return {
        "id": evidence.id,
        "corpusId": str(evidence.corpus_id),
        "sourceId": str(evidence.source_id),
        "documentId": str(evidence.document_id),
        "unitLocator": evidence.unit_locator,
        "startOffset": evidence.start_offset,
        "endOffset": evidence.end_offset,
        "excerpt": evidence.excerpt,
        "rank": evidence.rank,
    }


def _telemetry(outcome: str, evidence_count: int) -> dict[str, object]:
    return {"outcome": outcome, "evidenceCount": evidence_count, "durationMilliseconds": 0}
