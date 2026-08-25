"""Validated ingestion worker configuration."""

from __future__ import annotations

from dataclasses import dataclass
from datetime import timedelta
from typing import TYPE_CHECKING

if TYPE_CHECKING:
    from collections.abc import Mapping


class ConfigurationError(ValueError):
    """Report an invalid ingestion worker setting."""


_EMBEDDING_DIMENSIONS = 1536


@dataclass(frozen=True, slots=True)
class WorkerConfig:
    """Bound polling, leases, acquisition, and pipeline identity."""

    poll_interval: timedelta
    lease_duration: timedelta
    max_source_bytes: int
    max_redirects: int
    connect_timeout: timedelta
    read_timeout: timedelta
    pipeline_version: str
    embedding_endpoint: str = "https://api.openai.com/v1/embeddings"
    embedding_api_key: str = ""
    embedding_model: str = "text-embedding-3-small"
    embedding_dimensions: int = _EMBEDDING_DIMENSIONS
    embedding_timeout_seconds: int = 30
    embedding_batch_size: int = 32
    semantic_endpoint: str = "https://api.openai.com/v1/chat/completions"
    semantic_api_key: str = ""
    semantic_model: str = "gpt-5.6-luna"
    semantic_timeout_seconds: int = 30
    semantic_reasoning_effort: str = "medium"
    semantic_extraction_version: str = "legal-semantic-v1"

    @classmethod
    def from_environment(cls, environment: Mapping[str, str]) -> WorkerConfig:
        """Build configuration from an environment mapping with safe defaults."""
        poll_seconds = cls._positive_integer(environment, "NORVII_INGESTION_POLL_SECONDS", 1)
        lease_seconds = cls._positive_integer(environment, "NORVII_INGESTION_LEASE_SECONDS", 120)
        if lease_seconds <= poll_seconds:
            raise ConfigurationError("ingestion lease duration must exceed the polling interval")
        pipeline_version = environment.get(
            "NORVII_INGESTION_PIPELINE_VERSION", "corpus-ingestion-v3"
        ).strip()
        if not pipeline_version:
            raise ConfigurationError("NORVII_INGESTION_PIPELINE_VERSION must not be empty")
        semantic_timeout_seconds = cls._positive_integer(
            environment, "NORVII_SEMANTIC_TIMEOUT_SECONDS", 30
        )
        embedding_timeout_seconds = cls._positive_integer(
            environment, "NORVII_EMBEDDING_TIMEOUT_SECONDS", 30
        )
        if semantic_timeout_seconds + embedding_timeout_seconds >= lease_seconds:
            raise ConfigurationError(
                "semantic and embedding timeouts must fit within the ingestion lease duration"
            )
        return cls(
            poll_interval=timedelta(seconds=poll_seconds),
            lease_duration=timedelta(seconds=lease_seconds),
            max_source_bytes=cls._positive_integer(
                environment, "NORVII_INGESTION_MAX_SOURCE_BYTES", 10 * 1024 * 1024
            ),
            max_redirects=cls._nonnegative_integer(
                environment, "NORVII_INGESTION_MAX_REDIRECTS", 5
            ),
            connect_timeout=timedelta(
                seconds=cls._positive_integer(
                    environment, "NORVII_INGESTION_CONNECT_TIMEOUT_SECONDS", 5
                )
            ),
            read_timeout=timedelta(
                seconds=cls._positive_integer(
                    environment, "NORVII_INGESTION_READ_TIMEOUT_SECONDS", 15
                )
            ),
            pipeline_version=pipeline_version,
            embedding_endpoint=environment.get(
                "NORVII_EMBEDDING_BASE_URL", "https://api.openai.com/v1/embeddings"
            ).strip(),
            embedding_api_key=(
                environment.get("NORVII_EMBEDDING_API_KEY", "").strip()
                or environment.get("NORVII_CHAT_API_KEY", "").strip()
            ),
            embedding_model=environment.get(
                "NORVII_EMBEDDING_MODEL", "text-embedding-3-small"
            ).strip(),
            embedding_dimensions=cls._embedding_dimensions(environment),
            embedding_timeout_seconds=embedding_timeout_seconds,
            embedding_batch_size=cls._positive_integer(
                environment, "NORVII_EMBEDDING_BATCH_SIZE", 32
            ),
            semantic_endpoint=environment.get(
                "NORVII_SEMANTIC_BASE_URL", "https://api.openai.com/v1/chat/completions"
            ).strip(),
            semantic_api_key=(
                environment.get("NORVII_SEMANTIC_API_KEY", "").strip()
                or environment.get("NORVII_CHAT_API_KEY", "").strip()
            ),
            semantic_model=environment.get(
                "NORVII_SEMANTIC_MODEL", environment.get("NORVII_CHAT_MODEL", "gpt-5.6-luna")
            ).strip(),
            semantic_timeout_seconds=semantic_timeout_seconds,
            semantic_reasoning_effort=environment.get(
                "NORVII_SEMANTIC_REASONING_EFFORT", "medium"
            ).strip(),
            semantic_extraction_version=environment.get(
                "NORVII_SEMANTIC_EXTRACTION_VERSION", "legal-semantic-v1"
            ).strip(),
        )

    @staticmethod
    def _positive_integer(environment: Mapping[str, str], key: str, default: int) -> int:
        value = WorkerConfig._integer(environment, key, default)
        if value <= 0:
            raise ConfigurationError(f"{key} must be greater than zero")
        return value

    @staticmethod
    def _nonnegative_integer(environment: Mapping[str, str], key: str, default: int) -> int:
        value = WorkerConfig._integer(environment, key, default)
        if value < 0:
            raise ConfigurationError(f"{key} must not be negative")
        return value

    @staticmethod
    def _embedding_dimensions(environment: Mapping[str, str]) -> int:
        dimensions = WorkerConfig._positive_integer(
            environment, "NORVII_EMBEDDING_DIMENSIONS", _EMBEDDING_DIMENSIONS
        )
        if dimensions != _EMBEDDING_DIMENSIONS:
            raise ConfigurationError(
                "NORVII_EMBEDDING_DIMENSIONS must be "
                f"{_EMBEDDING_DIMENSIONS} for the current schema"
            )
        return dimensions

    @staticmethod
    def _integer(environment: Mapping[str, str], key: str, default: int) -> int:
        raw_value = environment.get(key)
        if raw_value is None:
            return default
        try:
            return int(raw_value)
        except ValueError as error:
            raise ConfigurationError(f"{key} must be an integer") from error


__all__ = ["ConfigurationError", "WorkerConfig"]
