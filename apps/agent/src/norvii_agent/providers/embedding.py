"""OpenAI-compatible query embedding adapter."""

from __future__ import annotations

import json
import math
from dataclasses import dataclass
from numbers import Real
from typing import Protocol
from urllib import error, request
from urllib.parse import urlparse


class EmbeddingProvider(Protocol):
    """Generate configured-dimension vectors in input order."""

    def embed(self, texts: tuple[str, ...]) -> tuple[tuple[float, ...], ...]:
        """Embed the provided non-empty text values."""
        ...


class EmbeddingProviderError(RuntimeError):
    """Signal a safely reportable embedding provider failure."""


@dataclass(frozen=True, slots=True)
class OpenAICompatibleEmbeddingProvider:
    """Call an OpenAI-compatible embeddings endpoint without logging content."""

    base_url: str
    api_key: str
    model: str
    dimensions: int
    timeout_seconds: float

    def embed(self, texts: tuple[str, ...]) -> tuple[tuple[float, ...], ...]:
        """Return one validated vector per input text in input order."""
        if not texts:
            return ()
        self._validate_configuration()
        if any(not text.strip() for text in texts):
            raise EmbeddingProviderError("embedding input must not be empty")
        payload = json.dumps({"model": self.model, "input": list(texts)}).encode()
        headers = {"Content-Type": "application/json"}
        if self.api_key:
            headers["Authorization"] = f"Bearer {self.api_key}"
        http_request = request.Request(  # noqa: S310 - scheme is validated above
            self.base_url, data=payload, headers=headers, method="POST"
        )
        try:
            with request.urlopen(http_request, timeout=self.timeout_seconds) as response:  # noqa: S310
                decoded = json.loads(response.read())
        except (error.URLError, TimeoutError, json.JSONDecodeError) as exc:
            raise EmbeddingProviderError("embedding provider request failed") from exc
        return self._decode_vectors(decoded, len(texts))

    def _validate_configuration(self) -> None:
        if not self.base_url:
            raise EmbeddingProviderError("embedding provider is not configured")
        if urlparse(self.base_url).scheme not in {"http", "https"}:
            raise EmbeddingProviderError("embedding provider endpoint scheme is unsupported")
        if not self.model.strip():
            raise EmbeddingProviderError("embedding provider model is not configured")
        if self.dimensions < 1:
            raise EmbeddingProviderError("embedding provider dimension must be positive")
        if self.timeout_seconds <= 0:
            raise EmbeddingProviderError("embedding provider timeout must be positive")

    def _decode_vectors(
        self, decoded: object, expected_count: int
    ) -> tuple[tuple[float, ...], ...]:
        if not isinstance(decoded, dict):
            raise EmbeddingProviderError("embedding provider response shape is invalid")
        rows = decoded.get("data")
        if not isinstance(rows, list):
            raise EmbeddingProviderError("embedding provider response shape is invalid")
        indexed_rows: dict[int, list[object]] = {}
        for row in rows:
            if not isinstance(row, dict):
                raise EmbeddingProviderError("embedding provider response shape is invalid")
            index = row.get("index")
            vector = row.get("embedding")
            if (
                isinstance(index, bool)
                or not isinstance(index, int)
                or not isinstance(vector, list)
            ):
                raise EmbeddingProviderError("embedding provider response shape is invalid")
            indexed_rows[index] = vector
        if set(indexed_rows) != set(range(expected_count)):
            raise EmbeddingProviderError("embedding provider response indexes are invalid")
        return tuple(self._validate_vector(indexed_rows[index]) for index in range(expected_count))

    def _validate_vector(self, vector: list[object]) -> tuple[float, ...]:
        if len(vector) != self.dimensions:
            raise EmbeddingProviderError("embedding provider vector dimension is invalid")
        values = tuple(float(value) for value in vector if isinstance(value, Real))
        if len(values) != self.dimensions or not all(math.isfinite(value) for value in values):
            raise EmbeddingProviderError("embedding provider vector values are invalid")
        return values
