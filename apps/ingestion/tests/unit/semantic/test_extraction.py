from __future__ import annotations

import hashlib
import json
from io import BytesIO
from typing import TYPE_CHECKING, Self
from urllib.error import HTTPError
from uuid import uuid4

import pytest

from norvii_ingestion.domain.artifacts import DocumentArtifact, DocumentUnit, UnitKind
from norvii_ingestion.domain.models import Sha256
from norvii_ingestion.semantic.extraction import (
    ExtractionProviderError,
    OpenAICompatibleSemanticExtractor,
    _remaining_timeout_seconds,
    _validated_batch,
)

if TYPE_CHECKING:
    from urllib.request import Request


class RecordingDiagnosticLogger:
    def __init__(self) -> None:
        self.events: list[tuple[str, dict[str, object]]] = []

    def failure(self, event: str, **fields: object) -> None:
        self.events.append((event, fields))


def test_validated_batch_creates_evidence_backed_atomic_assertions() -> None:
    unit = _article_unit()

    entities, assertions, usage = _validated_batch(
        _batch_content(
            unit,
            entities=(
                {"type": "obligation", "label": "General rules"},
                {"type": "actor", "label": "Union, States, Federal District, and Municipalities"},
            ),
            assertions=(
                {
                    "predicate": "must_be_observed_by",
                    "subject": "General rules",
                    "object": "Union, States, Federal District, and Municipalities",
                },
            ),
        ),
        (unit,),
    )

    assert {entity.label for entity in entities} == {
        "General rules",
        "Union",
        "States",
        "Federal District",
        "Municipalities",
    }
    assert len(assertions) == 4
    assert {assertion.evidence_unit_id for assertion in assertions} == {unit.id}
    assert {assertion.establishing_unit_id for assertion in assertions} == {unit.id}
    assert usage == (11, 7)


def test_validated_batch_deduplicates_assertions_that_only_differ_by_qualifier() -> None:
    unit = _article_unit()
    _entities, assertions, _usage = _validated_batch(
        _batch_content(
            unit,
            entities=(
                {"type": "concept", "label": "Term"},
                {"type": "concept", "label": "Definition"},
            ),
            assertions=(
                {
                    "predicate": "defines",
                    "subject": "Term",
                    "object": "Definition",
                    "qualifier": "first qualifier",
                },
                {
                    "predicate": "defines",
                    "subject": "Term",
                    "object": "Definition",
                    "qualifier": "second qualifier",
                },
            ),
        ),
        (unit,),
    )

    assert len(assertions) == 1
    assert assertions[0].qualifier == "first qualifier"


def test_validated_batch_discards_incomplete_assertions() -> None:
    unit = _article_unit()

    _entities, assertions, _usage = _validated_batch(
        {
            "content": {
                "entities": [
                    {"unitId": str(unit.id), "type": "concept", "label": "Term"},
                    {"unitId": str(unit.id), "type": "concept", "label": "Definition"},
                ],
                "assertions": [
                    {
                        "predicate": "defines",
                        "subject": "Term",
                        "object": "Definition",
                        "establishingUnitId": str(unit.id),
                    },
                    {
                        "predicate": "defines",
                        "subject": "Unknown",
                        "object": "Definition",
                        "establishingUnitId": str(unit.id),
                        "evidenceUnitId": str(unit.id),
                    },
                ],
            }
        },
        (unit,),
    )

    assert assertions == ()


def test_validated_batch_uses_the_normative_predicate_taxonomy() -> None:
    unit = _article_unit()

    _entities, assertions, _usage = _validated_batch(
        _batch_content(
            unit,
            entities=(
                {"type": "obligation", "label": "General rules"},
                {"type": "actor", "label": "Public bodies"},
            ),
            assertions=(
                {
                    "predicate": "must_be_observed_by",
                    "subject": "General rules",
                    "object": "Public bodies",
                },
                {
                    "predicate": "requires",
                    "subject": "General rules",
                    "object": "Public bodies",
                },
            ),
        ),
        (unit,),
    )

    assert [assertion.predicate for assertion in assertions] == ["must_be_observed_by"]


def test_extractor_exposes_a_safe_http_failure_diagnostic(monkeypatch: pytest.MonkeyPatch) -> None:
    unit = _article_unit()
    artifact = _artifact_for(unit)

    def fail_request(*_args: object, **_kwargs: object) -> None:
        raise HTTPError("https://example.test", 429, "rate limited", {}, BytesIO())

    monkeypatch.setattr("urllib.request.urlopen", fail_request)
    extractor = OpenAICompatibleSemanticExtractor(
        endpoint="https://example.test/v1/chat/completions",
        api_key="test-key",
        model="test-model",
    )

    with pytest.raises(ExtractionProviderError) as raised:
        extractor.extract(artifact)

    assert raised.value.detail == "provider_http_status_429"


def test_extractor_retries_one_invalid_completion(monkeypatch: pytest.MonkeyPatch) -> None:
    unit = _article_unit()
    artifact = _artifact_for(unit)
    responses = iter(
        (
            _ProviderResponse(
                json.dumps({"choices": [{"message": {"content": "not valid JSON"}}]}).encode()
            ),
            _ProviderResponse(
                json.dumps(
                    {
                        "choices": [{"message": {"content": '{"entities":[],"assertions":[]}'}}],
                        "usage": {"prompt_tokens": 1, "completion_tokens": 1},
                    }
                ).encode()
            ),
        )
    )
    monkeypatch.setattr("urllib.request.urlopen", lambda *_args, **_kwargs: next(responses))
    extractor = OpenAICompatibleSemanticExtractor(
        endpoint="https://example.test/v1/chat/completions",
        api_key="test-key",
        model="test-model",
    )

    extraction = extractor.extract(artifact)

    assert extraction.entities == ()
    assert extraction.assertions == ()


def test_extractor_skips_a_unit_after_two_invalid_completions(
    monkeypatch: pytest.MonkeyPatch,
) -> None:
    unit = _article_unit()
    artifact = _artifact_for(unit)
    monkeypatch.setattr(
        "urllib.request.urlopen",
        lambda *_args, **_kwargs: _ProviderResponse(
            json.dumps({"choices": [{"message": {"content": "not valid JSON"}}]}).encode()
        ),
    )
    extraction = OpenAICompatibleSemanticExtractor(
        endpoint="https://example.test/v1/chat/completions",
        api_key="test-key",
        model="test-model",
    ).extract(artifact)

    assert extraction.entities == ()
    assert extraction.assertions == ()


def test_extractor_logs_safe_invalid_completion_diagnostics(
    monkeypatch: pytest.MonkeyPatch,
) -> None:
    unit = _article_unit()
    artifact = _artifact_for(unit)
    logger = RecordingDiagnosticLogger()
    invalid_completion = "not valid JSON"
    monkeypatch.setattr(
        "urllib.request.urlopen",
        lambda *_args, **_kwargs: _ProviderResponse(
            json.dumps({"choices": [{"message": {"content": invalid_completion}}]}).encode(),
            headers={"content-type": "application/json", "x-request-id": "provider-123"},
        ),
    )

    OpenAICompatibleSemanticExtractor(
        endpoint="https://example.test/v1/chat/completions",
        api_key="test-key",
        model="test-model",
        diagnostic_logger=logger,
    ).extract(artifact)

    assert len(logger.events) == 2
    event, fields = logger.events[0]
    assert event == "semantic_provider_response_invalid"
    assert fields["diagnostic_code"] == "completion_content_invalid_json"
    assert fields["provider_request_id"] == "provider-123"
    assert fields["response_content_type"] == "application/json"
    assert fields["response_byte_count"] > 0
    assert fields["completion_byte_count"] == len(invalid_completion.encode())
    assert fields["provider_response_attempt"] == 1
    assert fields["unit_count"] == 1
    assert invalid_completion not in repr(logger.events)


def test_extractor_bounds_each_provider_request_and_requires_atomic_entities(
    monkeypatch: pytest.MonkeyPatch,
) -> None:
    artifact = _article_artifact(9)
    requests: list[dict[str, object]] = []

    def respond(request: Request, *_args: object, **_kwargs: object) -> _ProviderResponse:
        body = json.loads(request.data)
        requests.append(body)
        return _ProviderResponse(
            json.dumps(
                {
                    "choices": [{"message": {"content": '{"entities":[],"assertions":[]}'}}],
                    "usage": {"prompt_tokens": 1, "completion_tokens": 1},
                }
            ).encode()
        )

    monkeypatch.setattr("urllib.request.urlopen", respond)
    extraction = OpenAICompatibleSemanticExtractor(
        endpoint="https://example.test/v1/chat/completions",
        api_key="test-key",
        model="test-model",
    ).extract(artifact)

    assert len(requests) == 8
    assert all(_request_unit_count(request) == 1 for request in requests)
    assert all(request["max_completion_tokens"] == 1_600 for request in requests)
    assert extraction.entities == ()
    assert extraction.assertions == ()
    system_message = requests[0]["messages"][0]
    assert isinstance(system_message, dict)
    assert "establishingUnitId" in system_message["content"]
    assert "comma-separated aggregate entity" in system_message["content"]


def test_remaining_timeout_uses_a_single_document_budget(monkeypatch: pytest.MonkeyPatch) -> None:
    monkeypatch.setattr("norvii_ingestion.semantic.extraction.time.perf_counter", lambda: 10.2)

    assert _remaining_timeout_seconds(0.0, 180) == 170
    assert _remaining_timeout_seconds(0.0, 10) is None


def test_extractor_caps_each_provider_request_within_document_budget(
    monkeypatch: pytest.MonkeyPatch,
) -> None:
    unit = _article_unit()
    artifact = _artifact_for(unit)
    timeouts: list[int] = []

    def respond(_request: Request, *, timeout: int) -> _ProviderResponse:
        timeouts.append(timeout)
        return _ProviderResponse(
            json.dumps(
                {
                    "choices": [{"message": {"content": '{"entities":[],"assertions":[]}'}}],
                }
            ).encode()
        )

    monkeypatch.setattr("urllib.request.urlopen", respond)
    OpenAICompatibleSemanticExtractor(
        endpoint="https://example.test/v1/chat/completions",
        api_key="test-key",
        model="test-model",
        timeout_seconds=240,
    ).extract(artifact)

    assert timeouts == [30]


def _batch_content(
    unit: DocumentUnit,
    *,
    entities: tuple[dict[str, str], ...],
    assertions: tuple[dict[str, str], ...],
) -> dict[str, object]:
    return {
        "content": {
            "entities": [{"unitId": str(unit.id), **entity} for entity in entities],
            "assertions": [
                {
                    "establishingUnitId": str(unit.id),
                    "evidenceUnitId": str(unit.id),
                    **assertion,
                }
                for assertion in assertions
            ],
        },
        "usage": {"prompt_tokens": 11, "completion_tokens": 7},
    }


def _article_unit() -> DocumentUnit:
    text = "Article 1 grants access."
    root_id = uuid4()
    return DocumentUnit(
        id=uuid4(),
        parent_id=root_id,
        kind=UnitKind.ARTICLE,
        ordinal=0,
        marker="1",
        label="Article 1",
        start_offset=0,
        end_offset=len(text),
        start_page=None,
        end_page=None,
        locator="Article 1",
        content_sha256=Sha256(hashlib.sha256(text.encode()).hexdigest()),
    )


def _artifact_for(unit: DocumentUnit) -> DocumentArtifact:
    text = "Article 1 grants access."
    root = DocumentUnit(
        id=unit.parent_id or uuid4(),
        parent_id=None,
        kind=UnitKind.DOCUMENT,
        ordinal=0,
        marker=None,
        label=None,
        start_offset=0,
        end_offset=len(text),
        start_page=None,
        end_page=None,
        locator="document",
        content_sha256=Sha256.from_bytes(text.encode()),
    )
    artifact = DocumentArtifact(
        text=text,
        text_sha256=Sha256.from_bytes(text.encode()),
        units=(root, unit),
    )
    artifact.validate()
    return artifact


def _article_artifact(article_count: int) -> DocumentArtifact:
    text = "\n".join(f"Article {ordinal + 1}." for ordinal in range(article_count))
    root_id = uuid4()
    root = DocumentUnit(
        id=root_id,
        parent_id=None,
        kind=UnitKind.DOCUMENT,
        ordinal=0,
        marker=None,
        label=None,
        start_offset=0,
        end_offset=len(text),
        start_page=None,
        end_page=None,
        locator="document",
        content_sha256=Sha256.from_bytes(text.encode()),
    )
    units = [root]
    offset = 0
    for ordinal, line in enumerate(text.splitlines()):
        end_offset = offset + len(line)
        units.append(
            DocumentUnit(
                id=uuid4(),
                parent_id=root_id,
                kind=UnitKind.ARTICLE,
                ordinal=ordinal,
                marker=str(ordinal + 1),
                label=line,
                start_offset=offset,
                end_offset=end_offset,
                start_page=None,
                end_page=None,
                locator=line,
                content_sha256=Sha256.from_bytes(line.encode()),
            )
        )
        offset = end_offset + 1
    artifact = DocumentArtifact(
        text=text,
        text_sha256=Sha256.from_bytes(text.encode()),
        units=tuple(units),
    )
    artifact.validate()
    return artifact


def _request_unit_count(request: dict[str, object]) -> int:
    messages = request["messages"]
    assert isinstance(messages, list)
    user_message = messages[1]
    assert isinstance(user_message, dict)
    content = json.loads(user_message["content"])
    return len(content["units"])


class _ProviderResponse:
    def __init__(self, payload: bytes, headers: dict[str, str] | None = None) -> None:
        self._payload = payload
        self.headers = headers or {}

    def __enter__(self) -> Self:
        return self

    def __exit__(self, *_args: object) -> None:
        return None

    def read(self) -> bytes:
        return self._payload
