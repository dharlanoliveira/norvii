"""Small dependency-light HTTP server for the LangGraph service."""

from __future__ import annotations

import json
import re
from http.server import BaseHTTPRequestHandler, ThreadingHTTPServer
from typing import TYPE_CHECKING, Literal, cast
from uuid import UUID, uuid4

from norvii_agent.evaluation import (
    EvaluationContractError,
    EvaluationExecutor,
    EvaluationRequest,
    EvaluationResult,
    ExecutionIdentity,
    FrozenIdentityUnavailableError,
    FrozenRetrievalConfiguration,
)
from norvii_agent.graph import GroundedChatRequest

if TYPE_CHECKING:
    from collections.abc import Callable

    from norvii_agent.graph import AnswerInspection, Evidence, GroundedChatGraph

_CORPUS_PATH = re.compile(r"^/v1/corpora/(?P<corpus>[0-9a-f-]+)/chat/stream$")
_EVALUATION_PATH = "/v1/evaluations/execute"
_MAX_REQUEST_BODY_BYTES = 64 * 1024
_MAX_TELEMETRY_EVIDENCE_COUNT = 8
_SHA256 = re.compile(r"^[0-9a-f]{64}$")


class InvalidRequestBodyError(ValueError):
    """Raised when an internal request body cannot be read safely."""


class ClientDisconnectedError(ConnectionError):
    """Raised when an SSE client disconnects before the stream is complete."""


class AgentHTTPServer(ThreadingHTTPServer):
    """HTTP server carrying independent chat and fixed-snapshot evaluation factories."""

    def __init__(
        self,
        address: tuple[str, int],
        graph_factory: Callable[[], GroundedChatGraph],
        evaluation_executor_factory: Callable[[], EvaluationExecutor] | None = None,
    ) -> None:
        self.graph_factory = graph_factory
        self.evaluation_executor_factory = evaluation_executor_factory
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
        """Route an internal request to its isolated transport contract."""
        if self.path == _EVALUATION_PATH:
            self._execute_evaluation()
            return
        match = _CORPUS_PATH.fullmatch(self.path)
        if match is None:
            self.send_error(404)
            return
        try:
            corpus_id = UUID(match.group("corpus"))
            payload = self._read_json_payload()
            question = str(payload["question"]).strip()
            snapshot_id = UUID(str(payload["snapshotId"]))
            interface_language = str(payload.get("interfaceLanguage", "en")).strip().lower()
            strategy = str(payload.get("strategy", "vector")).strip().lower()
            if interface_language not in {"en", "pt"}:
                self.send_error(400, "invalid interface language")
                return
            if strategy not in {"vector", "hybrid"}:
                self.send_error(400, "invalid retrieval strategy")
                return
            if not question:
                self._invalid_question()
        except InvalidRequestBodyError:
            self.send_error(400, "invalid request body")
            return
        except (ValueError, KeyError, TypeError, json.JSONDecodeError):
            self.send_error(400, "invalid question")
            return

        self._stream_graph(corpus_id, snapshot_id, question, interface_language, strategy)

    def _execute_evaluation(self) -> None:
        """Execute one strict JSON evaluation request without invoking the SSE chat handler."""
        if self.server.evaluation_executor_factory is None:
            self._evaluation_error(404, "not_found")
            return
        try:
            request = _evaluation_request(self._read_evaluation_payload())
            result = self.server.evaluation_executor_factory().execute(request)
            payload = _evaluation_result_payload(result)
        except FrozenIdentityUnavailableError:
            self._evaluation_error(409, "frozen_identity_unavailable")
            return
        except (EvaluationContractError, InvalidRequestBodyError, ValueError, TypeError):
            self._evaluation_error(400, "invalid_request")
            return
        except Exception:  # noqa: BLE001 - never expose provider or retrieval diagnostics
            self._evaluation_error(502, "evaluation_unavailable")
            return
        encoded = json.dumps(payload, separators=(",", ":")).encode()
        self.send_response(200)
        self.send_header("Content-Type", "application/json")
        self.send_header("Content-Length", str(len(encoded)))
        self.end_headers()
        self.wfile.write(encoded)

    def _read_evaluation_payload(self) -> dict[str, object]:
        """Read the dedicated JSON media type before strict request validation."""
        if self.headers.get_content_type() != "application/json":
            raise InvalidRequestBodyError("evaluation request content type is invalid")
        return self._read_json_payload()

    def _evaluation_error(self, status: int, code: str) -> None:
        """Return a bounded content-free JSON error for the private evaluation transport."""
        encoded = json.dumps({"code": code}, separators=(",", ":")).encode()
        self.send_response(status)
        self.send_header("Content-Type", "application/json")
        self.send_header("Content-Length", str(len(encoded)))
        self.end_headers()
        self.wfile.write(encoded)

    def _stream_graph(
        self,
        corpus_id: UUID,
        snapshot_id: UUID,
        question: str,
        interface_language: str,
        strategy: str,
    ) -> None:
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
                GroundedChatRequest(corpus_id, question, interface_language, snapshot_id, strategy),
                deltas.append,
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
                        "inspection": _inspection(result.inspection),
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
        "snapshotId": _uuid_or_none(evidence.snapshot_id),
        "sourceId": str(evidence.source_id),
        "documentId": str(evidence.document_id),
        "documentVersionId": str(evidence.document_version_id or evidence.document_id),
        "sourceRevisionId": _uuid_or_none(evidence.source_revision_id),
        "pipelineVersion": evidence.pipeline_version,
        "sourceTitle": evidence.source_title,
        "unitLocator": evidence.unit_locator,
        "startOffset": evidence.start_offset,
        "endOffset": evidence.end_offset,
        "excerpt": evidence.excerpt,
        "rank": evidence.rank,
        "cosineDistance": evidence.cosine_distance,
        "contribution": evidence.contribution,
    }


def _evaluation_request(payload: dict[str, object]) -> EvaluationRequest:
    """Decode the versioned fixed-snapshot request without accepting chat fields."""
    _require_exact_keys(
        payload,
        {
            "corpusId",
            "snapshotId",
            "question",
            "interfaceLanguage",
            "retrievalConfiguration",
            "executionIdentity",
        },
    )
    configuration = _required_object(payload, "retrievalConfiguration")
    _require_exact_keys(configuration, {"strategy", "fingerprint"})
    execution_identity = _required_object(payload, "executionIdentity")
    _require_exact_keys(
        execution_identity,
        {"agentBuild", "chatModelIdentity", "embeddingModelIdentity"},
    )
    corpus_id = UUID(_required_text(payload, "corpusId"))
    snapshot_id = UUID(_required_text(payload, "snapshotId"))
    question = _required_text(payload, "question")
    interface_language = _required_text(payload, "interfaceLanguage")
    strategy = _required_text(configuration, "strategy")
    fingerprint = _required_text(configuration, "fingerprint")
    if interface_language not in {"en", "pt"}:
        raise EvaluationContractError("evaluation interface language is unsupported")
    return EvaluationRequest(
        corpus_id=corpus_id,
        snapshot_id=snapshot_id,
        question=question,
        interface_language=cast("Literal['en', 'pt']", interface_language),
        retrieval_configuration=FrozenRetrievalConfiguration(
            strategy=cast("Literal['vector', 'hybrid']", strategy), fingerprint=fingerprint
        ),
        execution_identity=ExecutionIdentity(
            agent_build=_required_text(execution_identity, "agentBuild"),
            chat_model_identity=_required_text(execution_identity, "chatModelIdentity"),
            embedding_model_identity=_required_text(execution_identity, "embeddingModelIdentity"),
        ),
    )


def _evaluation_result_payload(result: EvaluationResult) -> dict[str, object]:
    """Serialize complete immutable evaluation provenance as terminal JSON only."""
    answer = result.answer
    outcome = result.outcome
    retrieved_evidence = result.retrieved_evidence
    citation_marker_inputs = result.citation_marker_inputs
    graph_grounding = result.graph_grounding
    telemetry = result.telemetry
    model_identity = result.model_identity
    agent_build_identity = result.agent_build_identity
    embedding_model_identity = result.embedding_model_identity
    if (
        not isinstance(answer, str)
        or outcome not in {"completed", "abstained"}
        or not isinstance(retrieved_evidence, tuple)
        or not isinstance(citation_marker_inputs, tuple)
        or not isinstance(model_identity, str)
        or not model_identity.strip()
        or not isinstance(agent_build_identity, str)
        or not agent_build_identity.strip()
        or not isinstance(embedding_model_identity, str)
        or not embedding_model_identity.strip()
    ):
        raise EvaluationContractError("evaluation result is invalid")
    graph_status = _required_attribute(graph_grounding, "status")
    if graph_status not in {"not_requested", "not_used", "grounded"}:
        raise EvaluationContractError("evaluation graph grounding is invalid")
    return {
        "answer": answer,
        "outcome": outcome,
        "retrievedEvidence": [
            _evaluation_evidence_payload(item, index)
            for index, item in enumerate(retrieved_evidence, start=1)
        ],
        "citationMarkerInputs": [
            _citation_marker_payload(item, index)
            for index, item in enumerate(citation_marker_inputs, start=1)
        ],
        "graphGrounding": {"status": graph_status},
        "modelIdentity": model_identity,
        "agentBuildIdentity": agent_build_identity,
        "embeddingModelIdentity": embedding_model_identity,
        "telemetry": {
            "retrievalMilliseconds": _measurement_payload(telemetry, "retrieval_milliseconds"),
            "generationMilliseconds": _measurement_payload(telemetry, "generation_milliseconds"),
            "totalMilliseconds": _measurement_payload(telemetry, "total_milliseconds"),
            "inputTokens": _measurement_payload(telemetry, "input_tokens"),
            "outputTokens": _measurement_payload(telemetry, "output_tokens"),
        },
    }


def _evaluation_evidence_payload(evidence: object, expected_rank: int) -> dict[str, object]:
    """Require the full Go evidence provenance contract before writing a response."""
    rank = _required_attribute(evidence, "rank")
    corpus_id = _required_attribute(evidence, "corpus_id")
    snapshot_id = _required_attribute(evidence, "snapshot_id")
    source_id = _required_attribute(evidence, "source_id")
    source_revision_id = _required_attribute(evidence, "source_revision_id")
    document_id = _required_attribute(evidence, "document_id")
    unit_id = _required_attribute(evidence, "unit_id")
    canonical_locator = _required_attribute(evidence, "canonical_locator")
    start_offset = _required_attribute(evidence, "start_offset")
    end_offset = _required_attribute(evidence, "end_offset")
    content_sha256 = _required_attribute(evidence, "content_sha256")
    if (
        isinstance(rank, bool)
        or not isinstance(rank, int)
        or rank != expected_rank
        or not all(
            isinstance(value, UUID)
            for value in (
                corpus_id,
                snapshot_id,
                source_id,
                source_revision_id,
                document_id,
                unit_id,
            )
        )
        or not isinstance(canonical_locator, str)
        or not canonical_locator.strip()
        or not isinstance(start_offset, int)
        or isinstance(start_offset, bool)
        or not isinstance(end_offset, int)
        or isinstance(end_offset, bool)
        or start_offset < 0
        or end_offset <= start_offset
        or not isinstance(content_sha256, str)
        or _SHA256.fullmatch(content_sha256) is None
    ):
        raise EvaluationContractError("evaluation evidence provenance is invalid")
    return {
        "rank": rank,
        "corpusId": str(corpus_id),
        "snapshotId": str(snapshot_id),
        "sourceId": str(source_id),
        "sourceRevisionId": str(source_revision_id),
        "documentId": str(document_id),
        "unitId": str(unit_id),
        "canonicalLocator": canonical_locator,
        "startOffset": start_offset,
        "endOffset": end_offset,
        "contentSha256": content_sha256,
    }


def _citation_marker_payload(marker: object, expected_position: int) -> dict[str, int]:
    """Serialize the one-based marker/evidence order supplied by the executor."""
    marker_position = _required_attribute(marker, "marker_position")
    evidence = _required_attribute(marker, "evidence")
    evidence_rank = _required_attribute(evidence, "rank")
    if marker_position != expected_position or evidence_rank != expected_position:
        raise EvaluationContractError("evaluation citation marker order is invalid")
    return {"markerPosition": marker_position, "evidenceRank": evidence_rank}


def _measurement_payload(telemetry: object, name: str) -> int | None:
    """Keep nullable telemetry bounded to non-negative integral values."""
    value = getattr(telemetry, name, None)
    if value is None:
        return None
    if isinstance(value, bool) or not isinstance(value, int) or value < 0:
        raise EvaluationContractError("evaluation telemetry is invalid")
    return value


def _require_exact_keys(payload: dict[str, object], expected: set[str]) -> None:
    """Reject unknown or missing JSON fields at the evaluation boundary."""
    if set(payload) != expected:
        raise InvalidRequestBodyError("evaluation request fields are invalid")


def _required_object(payload: dict[str, object], key: str) -> dict[str, object]:
    """Return one required object field without coercing client-controlled values."""
    value = payload.get(key)
    if not isinstance(value, dict):
        raise InvalidRequestBodyError("evaluation request object is invalid")
    return cast("dict[str, object]", value)


def _required_text(payload: dict[str, object], key: str) -> str:
    """Return a non-blank string without accepting coercible request values."""
    value = payload.get(key)
    if not isinstance(value, str) or not value.strip():
        raise InvalidRequestBodyError("evaluation request text is invalid")
    return value.strip()


def _required_attribute(value: object, name: str) -> object:
    """Read required immutable result attributes without silently substituting values."""
    result = getattr(value, name, None)
    if result is None:
        raise EvaluationContractError("evaluation result is incomplete")
    return result


def _inspection(inspection: AnswerInspection | None) -> dict[str, object]:
    """Serialize safe terminal inspection metadata without provider payloads."""
    if inspection is None:
        return {
            "outcome": "completed",
            "retrieval": {
                "strategy": "vector",
                "topK": 8,
                "returnedCount": 0,
                "embeddingModel": None,
            },
            "measurements": {
                "retrievalMilliseconds": None,
                "generationMilliseconds": None,
                "totalMilliseconds": None,
                "inputTokens": None,
                "outputTokens": None,
            },
            "evidence": [],
            "assertionPath": [],
            "scopeLocator": None,
            "stages": [],
        }
    measurements = inspection.measurements
    return {
        "outcome": inspection.outcome,
        "retrieval": {
            "strategy": inspection.retrieval.strategy,
            "topK": inspection.retrieval.top_k,
            "returnedCount": inspection.retrieval.returned_count,
            "embeddingModel": inspection.retrieval.embedding_model,
        },
        "measurements": {
            "retrievalMilliseconds": measurements.retrieval_milliseconds,
            "generationMilliseconds": measurements.generation_milliseconds,
            "totalMilliseconds": measurements.total_milliseconds,
            "inputTokens": measurements.input_tokens,
            "outputTokens": measurements.output_tokens,
        },
        "evidence": [_reference(item) for item in inspection.evidence],
        "assertionPath": [
            {
                "assertionId": step.assertion_id,
                "predicate": step.predicate,
                "subjectLabel": step.subject_label,
                "objectLabel": step.object_label,
                "establishingLocator": step.establishing_locator,
                "evidenceLocator": step.evidence_locator,
                "hierarchyContext": list(step.hierarchy_context),
                "qualifier": step.qualifier,
            }
            for step in inspection.assertion_path
        ],
        "scopeLocator": inspection.scope_locator,
        "stages": [
            {
                "name": stage.name,
                "state": stage.state,
                "evidenceCount": stage.evidence_count,
                "durationMilliseconds": stage.duration_milliseconds,
                "reasonCode": stage.reason_code,
                "inputTokens": stage.input_tokens,
                "outputTokens": stage.output_tokens,
            }
            for stage in inspection.stages
        ],
    }


def _uuid_or_none(value: object) -> str | None:
    return str(value) if value is not None else None


def _telemetry(outcome: str, evidence_count: int) -> dict[str, object]:
    """Return only bounded, content-free terminal measurements."""
    return {
        "outcome": outcome,
        "evidenceCount": min(max(evidence_count, 0), _MAX_TELEMETRY_EVIDENCE_COUNT),
        "durationMilliseconds": 0,
    }
