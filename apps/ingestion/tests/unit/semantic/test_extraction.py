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


def test_validated_batch_creates_evidence_backed_entities_and_relationships() -> None:
    unit = _article_unit()
    entities, relationships, usage = _validated_batch(
        {
            "content": {
                "entities": [
                    {"unitId": str(unit.id), "type": "actor", "label": "Controller"},
                    {"unitId": str(unit.id), "type": "right", "label": "Access"},
                ],
                "relationships": [
                    {
                        "unitId": str(unit.id),
                        "type": "grants",
                        "subject": "Controller",
                        "object": "Access",
                    }
                ],
            },
            "usage": {"prompt_tokens": 11, "completion_tokens": 7},
        },
        (unit,),
    )

    assert len(entities) == 2
    assert relationships[0].evidence_unit_id == unit.id
    assert usage == (11, 7)


def test_validated_batch_deduplicates_relationships_that_only_differ_by_qualifier() -> None:
    unit = _article_unit()
    _entities, relationships, _usage = _validated_batch(
        {
            "content": {
                "entities": [
                    {"unitId": str(unit.id), "type": "actor", "label": "Controller"},
                    {"unitId": str(unit.id), "type": "right", "label": "Access"},
                ],
                "relationships": [
                    {
                        "unitId": str(unit.id),
                        "type": "grants",
                        "subject": "Controller",
                        "object": "Access",
                        "qualifier": "first qualifier",
                    },
                    {
                        "unitId": str(unit.id),
                        "type": "grants",
                        "subject": "Controller",
                        "object": "Access",
                        "qualifier": "second qualifier",
                    },
                ],
            }
        },
        (unit,),
    )

    assert len(relationships) == 1
    assert relationships[0].qualifier == "first qualifier"


def test_validated_batch_discards_relationships_without_declared_entities() -> None:
    unit = _article_unit()

    entities, relationships, _usage = _validated_batch(
        {
            "content": {
                "entities": [],
                "relationships": [
                    {
                        "unitId": str(unit.id),
                        "type": "grants",
                        "subject": "Controller",
                        "object": "Access",
                    }
                ],
            }
        },
        (unit,),
    )

    assert entities == ()
    assert relationships == ()


def test_validated_batch_discards_semantic_items_with_an_invalid_schema() -> None:
    unit = _article_unit()

    entities, relationships, _usage = _validated_batch(
        {
            "content": {
                "entities": [
                    {"unitId": "unrecognized", "type": "right", "label": "Access"},
                    {"unitId": str(unit.id), "type": "unsupported", "label": "Access"},
                    {"unitId": str(unit.id), "type": "right", "label": "Access"},
                ],
                "relationships": [
                    {
                        "unitId": str(unit.id),
                        "type": "unsupported",
                        "subject": "Access",
                        "object": "Access",
                    }
                ],
            }
        },
        (unit,),
    )

    assert [entity.label for entity in entities] == ["Access"]
    assert relationships == ()


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


def test_extractor_bounds_each_provider_request_to_one_legal_location(
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
                    "choices": [{"message": {"content": '{"entities":[],"relationships":[]}'}}],
                    "usage": {"prompt_tokens": 1, "completion_tokens": 1},
                }
            ).encode()
        )

    monkeypatch.setattr("urllib.request.urlopen", respond)
    extractor = OpenAICompatibleSemanticExtractor(
        endpoint="https://example.test/v1/chat/completions",
        api_key="test-key",
        model="test-model",
    )

    extraction = extractor.extract(artifact)

    assert len(requests) == 8
    assert all(_request_unit_count(request) == 1 for request in requests)
    assert extraction.input_tokens == 8
    assert extraction.output_tokens == 8


def test_remaining_timeout_uses_a_single_document_budget(monkeypatch: pytest.MonkeyPatch) -> None:
    monkeypatch.setattr("norvii_ingestion.semantic.extraction.time.perf_counter", lambda: 10.2)

    assert _remaining_timeout_seconds(0.0, 180) == 170
    assert _remaining_timeout_seconds(0.0, 10) is None


def _article_unit() -> DocumentUnit:
    text = "Article 1 grants access."
    artifact = DocumentArtifact(
        text=text,
        text_sha256=Sha256.from_bytes(text.encode()),
        units=(
            DocumentUnit(
                id=uuid4(),
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
            ),
        ),
    )
    artifact.validate()
    return DocumentUnit(
        id=uuid4(),
        parent_id=artifact.units[0].id,
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


def _article_artifact(count: int) -> DocumentArtifact:
    text = "\n".join(f"Article {index} grants access." for index in range(1, count + 1))
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
    for index, line in enumerate(text.splitlines(), start=1):
        units.append(
            DocumentUnit(
                id=uuid4(),
                parent_id=root_id,
                kind=UnitKind.ARTICLE,
                ordinal=index - 1,
                marker=str(index),
                label=f"Article {index}",
                start_offset=offset,
                end_offset=offset + len(line),
                start_page=None,
                end_page=None,
                locator=f"Article {index}",
                content_sha256=Sha256.from_bytes(line.encode()),
            )
        )
        offset += len(line) + 1
    artifact = DocumentArtifact(
        text=text,
        text_sha256=Sha256.from_bytes(text.encode()),
        units=tuple(units),
    )
    artifact.validate()
    return artifact


class _ProviderResponse:
    def __init__(self, body: bytes) -> None:
        self._body = body

    def __enter__(self) -> Self:
        return self

    def __exit__(self, *_args: object) -> None:
        return None

    def read(self) -> bytes:
        return self._body


def _request_unit_count(body: dict[str, object]) -> int:
    messages = body["messages"]
    assert isinstance(messages, list)
    user_message = messages[1]
    assert isinstance(user_message, dict)
    content = user_message["content"]
    assert isinstance(content, str)
    units = json.loads(content)["units"]
    assert isinstance(units, list)
    return len(units)
