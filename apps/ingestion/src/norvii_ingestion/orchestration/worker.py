"""Cancellation-aware polling shell for bounded ingestion attempts."""

from __future__ import annotations

import time
from typing import TYPE_CHECKING, Protocol

if TYPE_CHECKING:
    from datetime import timedelta

    from norvii_ingestion.config import WorkerConfig
    from norvii_ingestion.domain.models import IngestionWork


class StopSignal(Protocol):
    """Expose cooperative cancellation without coupling the worker to threads."""

    def is_set(self) -> bool:
        """Return whether shutdown has been requested."""
        ...

    def wait(self, timeout: float) -> bool:
        """Wait up to timeout seconds and return whether shutdown was requested."""
        ...


class WorkSource(Protocol):
    """Claim one oldest eligible work item with a bounded lease."""

    def claim(self, worker_id: str, lease_duration: timedelta) -> IngestionWork | None:
        """Return one leased claim or none when the queue is empty."""
        ...


class ClaimProcessor(Protocol):
    """Execute one claimed ingestion attempt."""

    def process(self, work: IngestionWork) -> None:
        """Acquire, extract, and publish one claim or categorize its failure."""
        ...


class EventLogger(Protocol):
    """Record structured events without source content or secrets."""

    def info(self, event: str, **fields: object) -> None:
        """Record a safe informational event."""
        ...

    def failure(self, event: str, **fields: object) -> None:
        """Record a safe failure event."""
        ...


class Worker:
    """Poll, process one claim at a time, and stop cooperatively."""

    def __init__(
        self,
        *,
        config: WorkerConfig,
        worker_id: str,
        work_source: WorkSource,
        processor: ClaimProcessor,
        logger: EventLogger,
    ) -> None:
        if not worker_id.strip():
            raise ValueError("worker identifier must not be empty")
        self._config = config
        self._worker_id = worker_id
        self._work_source = work_source
        self._processor = processor
        self._logger = logger

    def run(self, stop: StopSignal) -> None:
        """Run until cancellation while keeping every poll and lease bounded."""
        self._logger.info("ingestion_worker_started", worker_id=self._worker_id)
        while not stop.is_set():
            work = self._work_source.claim(self._worker_id, self._config.lease_duration)
            if work is None:
                stop.wait(self._config.poll_interval.total_seconds())
                continue
            started = time.monotonic()
            fields = {
                "worker_id": self._worker_id,
                "work_id": str(work.claim.work_id),
                "corpus_id": str(work.claim.corpus_id),
                "source_id": str(work.claim.source_id),
                "source_kind": work.claim.source_kind.value,
                "reason": work.claim.reason.value,
                "pipeline_version": self._config.pipeline_version,
            }
            self._logger.info("ingestion_claim_started", **fields, state="processing")
            try:
                self._processor.process(work)
                self._logger.info(
                    "ingestion_claim_completed",
                    **fields,
                    state="terminal",
                    duration_ms=int((time.monotonic() - started) * 1000),
                )
            except Exception as error:  # noqa: BLE001 - process boundary must survive one claim
                self._logger.failure(
                    "ingestion_claim_failed",
                    **fields,
                    state="failed",
                    duration_ms=int((time.monotonic() - started) * 1000),
                    error_type=type(error).__name__,
                )
        self._logger.info("ingestion_worker_stopped", worker_id=self._worker_id)
