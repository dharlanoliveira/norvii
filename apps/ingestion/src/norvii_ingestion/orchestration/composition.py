"""Production composition adapters for the ingestion worker shell."""

from __future__ import annotations

import json
from typing import TYPE_CHECKING

if TYPE_CHECKING:
    import logging
    from collections.abc import Callable
    from datetime import datetime, timedelta

    from norvii_ingestion.domain.models import IngestionWork
    from norvii_ingestion.publication.postgres.repository import PostgresWorkRepository


class PostgresWorkSource:
    """Adapt the timestamped PostgreSQL claim operation to the worker polling port."""

    def __init__(
        self,
        repository: PostgresWorkRepository,
        clock: Callable[[], datetime],
    ) -> None:
        self._repository = repository
        self._clock = clock

    def claim(self, worker_id: str, lease_duration: timedelta) -> IngestionWork | None:
        """Claim one work item using the current UTC time."""
        return self._repository.claim(worker_id, lease_duration, self._clock())


class StructuredEventLogger:
    """Write allowlisted structured worker events through standard logging."""

    _ALLOWED_FIELDS = frozenset(
        {
            "worker_id",
            "work_id",
            "corpus_id",
            "source_id",
            "source_kind",
            "reason",
            "state",
            "duration_ms",
            "pipeline_version",
            "failure_category",
            "byte_count",
            "character_count",
            "unit_count",
            "error_type",
        }
    )

    def __init__(self, logger: logging.Logger) -> None:
        self._logger = logger

    def info(self, event: str, **fields: object) -> None:
        """Write a safe informational event."""
        self._logger.info("%s %s", event, self._safe_fields(fields))

    def failure(self, event: str, **fields: object) -> None:
        """Write a safe failure event without exception text."""
        self._logger.error("%s %s", event, self._safe_fields(fields))

    def _safe_fields(self, fields: dict[str, object]) -> str:
        safe = {key: str(value) for key, value in fields.items() if key in self._ALLOWED_FIELDS}
        return json.dumps(safe, sort_keys=True)
