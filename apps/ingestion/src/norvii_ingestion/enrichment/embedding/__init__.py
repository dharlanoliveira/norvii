"""Provider-neutral embedding ports and bounded OpenAI-compatible adapter."""

from __future__ import annotations

import json
import urllib.request
from dataclasses import dataclass
from typing import TYPE_CHECKING, Protocol
from urllib.error import HTTPError, URLError
from urllib.request import Request

if TYPE_CHECKING:
    from collections.abc import Sequence

_MALFORMED_RESPONSE = "embedding provider response is malformed"


class EmbeddingProviderError(RuntimeError):
    """Report an unavailable or invalid embedding provider response."""

    def __init__(self, message: str, detail: str = "provider_response_invalid") -> None:
        super().__init__(message)
        self.detail = detail


class EmbeddingProvider(Protocol):
    """Generate vectors for a bounded sequence of texts."""

    def embed(self, texts: Sequence[str]) -> tuple[tuple[float, ...], ...]:
        """Return one vector per input text, preserving input order."""
        ...


@dataclass(frozen=True, slots=True)
class OpenAICompatibleEmbeddingProvider:
    """Call an OpenAI-compatible embeddings endpoint without an SDK dependency."""

    endpoint: str
    api_key: str
    model: str
    dimensions: int = 1536
    timeout_seconds: int = 30
    batch_size: int = 32

    def __post_init__(self) -> None:
        """Validate provider settings at construction time."""
        if not self.endpoint.strip() or not self.api_key.strip() or not self.model.strip():
            raise ValueError("embedding endpoint, API key, and model are required")
        if self.dimensions <= 0 or self.timeout_seconds <= 0 or self.batch_size <= 0:
            raise ValueError("embedding dimensions, timeout, and batch size must be positive")

    def embed(self, texts: Sequence[str]) -> tuple[tuple[float, ...], ...]:
        """Generate bounded batches and validate every returned vector strictly."""
        if any(not text.strip() for text in texts):
            raise ValueError("embedding text must not be empty")
        vectors: list[tuple[float, ...]] = []
        for start in range(0, len(texts), self.batch_size):
            vectors.extend(self._embed_batch(texts[start : start + self.batch_size]))
        return tuple(vectors)

    def _embed_batch(self, texts: Sequence[str]) -> tuple[tuple[float, ...], ...]:
        payload = json.dumps({"model": self.model, "input": list(texts)}).encode("utf-8")
        request = Request(  # noqa: S310
            self.endpoint,
            data=payload,
            headers={
                "Authorization": f"Bearer {self.api_key}",
                "Content-Type": "application/json",
            },
            method="POST",
        )
        try:
            with urllib.request.urlopen(request, timeout=self.timeout_seconds) as response:  # noqa: S310
                decoded = json.loads(response.read())
        except HTTPError as error:
            raise EmbeddingProviderError(
                "embedding provider request failed", f"provider_http_status_{error.code}"
            ) from error
        except URLError as error:
            detail = (
                "provider_timeout"
                if isinstance(error.reason, TimeoutError)
                else "provider_transport"
            )
            raise EmbeddingProviderError("embedding provider request failed", detail) from error
        except TimeoutError as error:
            raise EmbeddingProviderError(
                "embedding provider request failed", "provider_timeout"
            ) from error
        except json.JSONDecodeError as error:
            raise EmbeddingProviderError(
                "embedding provider response is malformed", "provider_response_invalid"
            ) from error
        return self._parse_vectors(decoded, len(texts))

    def _parse_vectors(self, payload: object, expected_count: int) -> tuple[tuple[float, ...], ...]:
        if not isinstance(payload, dict) or not isinstance(payload.get("data"), list):
            raise EmbeddingProviderError(_MALFORMED_RESPONSE, "provider_response_schema_invalid")
        entries = payload["data"]
        if len(entries) != expected_count:
            raise EmbeddingProviderError(
                "embedding provider returned an unexpected item count",
                "provider_response_schema_invalid",
            )
        vectors = [self._parse_entry(entry, expected_count) for entry in entries]
        indexed_vectors = dict(vectors)
        if set(indexed_vectors) != set(range(expected_count)):
            raise EmbeddingProviderError(_MALFORMED_RESPONSE, "provider_response_schema_invalid")
        ordered = [indexed_vectors[index] for index in range(expected_count)]
        return tuple(ordered)

    def _parse_entry(self, entry: object, expected_count: int) -> tuple[int, tuple[float, ...]]:
        if not isinstance(entry, dict):
            raise EmbeddingProviderError(_MALFORMED_RESPONSE, "provider_response_schema_invalid")
        index = entry.get("index")
        values = entry.get("embedding")
        if not isinstance(index, int) or not 0 <= index < expected_count:
            raise EmbeddingProviderError(
                "embedding provider returned an invalid index", "provider_response_schema_invalid"
            )
        if not isinstance(values, list):
            raise EmbeddingProviderError(_MALFORMED_RESPONSE, "provider_response_schema_invalid")
        if len(values) != self.dimensions or not all(
            isinstance(value, (int, float)) for value in values
        ):
            raise EmbeddingProviderError(
                "embedding vector dimensions are invalid", "provider_response_schema_invalid"
            )
        return index, tuple(float(value) for value in values)


__all__ = ["EmbeddingProvider", "EmbeddingProviderError", "OpenAICompatibleEmbeddingProvider"]
