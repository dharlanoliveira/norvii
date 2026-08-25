from __future__ import annotations

import json
from io import BytesIO
from typing import Self
from urllib.error import HTTPError

import pytest

from norvii_ingestion.enrichment.embedding import (
    EmbeddingProviderError,
    OpenAICompatibleEmbeddingProvider,
)


class FakeResponse:
    def __init__(self, payload: dict[str, object]) -> None:
        self._payload = json.dumps(payload).encode()

    def read(self) -> bytes:
        return self._payload

    def __enter__(self) -> Self:
        return self

    def __exit__(self, *args: object) -> None:
        return None


def test_provider_returns_embeddings_in_input_order(monkeypatch: pytest.MonkeyPatch) -> None:
    captured: list[object] = []

    def open_url(request: object, timeout: float) -> FakeResponse:
        captured.append((request, timeout))
        return FakeResponse(
            {
                "data": [
                    {"index": 1, "embedding": [0.2, 0.3]},
                    {"index": 0, "embedding": [0.1, 0.4]},
                ]
            }
        )

    monkeypatch.setattr("urllib.request.urlopen", open_url)
    provider = OpenAICompatibleEmbeddingProvider(
        endpoint="https://api.example.test/v1/embeddings",
        api_key="test-key",
        model="test-embedding",
        dimensions=2,
        timeout_seconds=3,
        batch_size=8,
    )

    result = provider.embed(("first", "second"))

    assert result == ((0.1, 0.4), (0.2, 0.3))
    assert len(captured) == 1


def test_provider_rejects_wrong_embedding_dimensions(monkeypatch: pytest.MonkeyPatch) -> None:
    def open_url(_request: object, *, timeout: float) -> FakeResponse:
        del timeout
        return FakeResponse({"data": [{"index": 0, "embedding": [0.1]}]})

    monkeypatch.setattr("urllib.request.urlopen", open_url)
    provider = OpenAICompatibleEmbeddingProvider(
        endpoint="https://api.example.test/v1/embeddings",
        api_key="test-key",
        model="test-embedding",
        dimensions=2,
        timeout_seconds=3,
        batch_size=8,
    )

    with pytest.raises(EmbeddingProviderError, match="dimensions"):
        provider.embed(("first",))


def test_provider_exposes_a_safe_http_failure_diagnostic(monkeypatch: pytest.MonkeyPatch) -> None:
    def open_url(_request: object, *, timeout: float) -> FakeResponse:
        del timeout
        raise HTTPError("https://api.example.test", 429, "rate limited", {}, BytesIO())

    monkeypatch.setattr("urllib.request.urlopen", open_url)
    provider = OpenAICompatibleEmbeddingProvider(
        endpoint="https://api.example.test/v1/embeddings",
        api_key="test-key",
        model="test-embedding",
        dimensions=2,
    )

    with pytest.raises(EmbeddingProviderError) as raised:
        provider.embed(("first",))

    assert raised.value.detail == "provider_http_status_429"
