"""Validated ingestion worker configuration."""

from __future__ import annotations

from dataclasses import dataclass
from datetime import timedelta
from typing import TYPE_CHECKING

if TYPE_CHECKING:
    from collections.abc import Mapping


class ConfigurationError(ValueError):
    """Report an invalid ingestion worker setting."""


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

    @classmethod
    def from_environment(cls, environment: Mapping[str, str]) -> WorkerConfig:
        """Build configuration from an environment mapping with safe defaults."""
        poll_seconds = cls._positive_integer(environment, "NORVII_INGESTION_POLL_SECONDS", 1)
        lease_seconds = cls._positive_integer(environment, "NORVII_INGESTION_LEASE_SECONDS", 120)
        if lease_seconds <= poll_seconds:
            raise ConfigurationError("ingestion lease duration must exceed the polling interval")
        pipeline_version = environment.get(
            "NORVII_INGESTION_PIPELINE_VERSION", "corpus-ingestion-v2"
        ).strip()
        if not pipeline_version:
            raise ConfigurationError("NORVII_INGESTION_PIPELINE_VERSION must not be empty")
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
    def _integer(environment: Mapping[str, str], key: str, default: int) -> int:
        raw_value = environment.get(key)
        if raw_value is None:
            return default
        try:
            return int(raw_value)
        except ValueError as error:
            raise ConfigurationError(f"{key} must be an integer") from error


__all__ = ["ConfigurationError", "WorkerConfig"]
