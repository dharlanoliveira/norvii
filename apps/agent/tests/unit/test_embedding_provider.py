from __future__ import annotations

import json
from io import BytesIO
from types import SimpleNamespace
from typing import Self

import pytest

from norvii_agent.providers.embedding import (
    EmbeddingProviderError,
    OpenAICompatibleEmbeddingProvider,
)


class FakeResponse:
    def __init__(self, body: bytes) -> None:
        self.headers = SimpleNamespace(get=lambda *_args, **_kwargs: "application/json")
        self._body = BytesIO(body)

    def __enter__(self) -> Self:
        return self

    def __exit__(self, *_args: object) -> None:
        return None

    def read(self) -> bytes:
        return self._body.read()


def test_provider_preserves_response_order_and_configured_model(
    monkeypatch: pytest.MonkeyPatch,
) -> None:
    captured_request: dict[str, object] = {}

    def open_request(http_request: object, **_kwargs: object) -> FakeResponse:
        captured_request["body"] = json.loads(http_request.data)
        return FakeResponse(
            b'{"data":[{"index":1,"embedding":[0.3,0.4]},{"index":0,"embedding":[0.1,0.2]}]}'
        )

    monkeypatch.setattr("norvii_agent.providers.embedding.request.urlopen", open_request)

    embeddings = OpenAICompatibleEmbeddingProvider(
        "https://provider.test/embeddings", "secret", "embedding-model", 2, 1
    ).embed(("first", "second"))

    assert embeddings == ((0.1, 0.2), (0.3, 0.4))
    assert captured_request["body"] == {
        "model": "embedding-model",
        "input": ["first", "second"],
    }


def test_provider_rejects_an_embedding_with_the_wrong_dimension(
    monkeypatch: pytest.MonkeyPatch,
) -> None:
    monkeypatch.setattr(
        "norvii_agent.providers.embedding.request.urlopen",
        lambda *_args, **_kwargs: FakeResponse(b'{"data":[{"index":0,"embedding":[0.1]}]}'),
    )

    provider = OpenAICompatibleEmbeddingProvider(
        "https://provider.test/embeddings", "secret", "embedding-model", 2, 1
    )

    with pytest.raises(EmbeddingProviderError, match="dimension"):
        provider.embed(("first",))
