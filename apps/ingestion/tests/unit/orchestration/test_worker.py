from __future__ import annotations

from datetime import UTC, datetime, timedelta
from uuid import uuid4

from norvii_ingestion.config import WorkerConfig
from norvii_ingestion.domain.models import (
    IngestionWork,
    SourceKind,
    WorkClaim,
    WorkReason,
)
from norvii_ingestion.orchestration.worker import Worker


class FakeStopSignal:
    def __init__(self, waits_before_stop: int) -> None:
        self._waits_before_stop = waits_before_stop
        self.waits: list[float] = []

    def is_set(self) -> bool:
        return self._waits_before_stop <= 0

    def wait(self, timeout: float) -> bool:
        self.waits.append(timeout)
        self._waits_before_stop -= 1
        return self.is_set()


class FakeWorkSource:
    def __init__(self, work: IngestionWork | None) -> None:
        self._work = work
        self.lease_durations: list[timedelta] = []

    def claim(self, worker_id: str, lease_duration: timedelta) -> IngestionWork | None:
        assert worker_id == "worker-1"
        self.lease_durations.append(lease_duration)
        work, self._work = self._work, None
        return work


class RecordingProcessor:
    def __init__(self, failure: Exception | None = None) -> None:
        self.failure = failure
        self.works: list[IngestionWork] = []

    def process(self, work: IngestionWork) -> None:
        self.works.append(work)
        if self.failure is not None:
            raise self.failure


class RecordingLogger:
    def __init__(self) -> None:
        self.events: list[tuple[str, dict[str, object]]] = []

    def info(self, event: str, **fields: object) -> None:
        self.events.append((event, fields))

    def failure(self, event: str, **fields: object) -> None:
        self.events.append((event, fields))


def test_worker_polls_with_bounded_lease_and_honors_stop_signal() -> None:
    work_source = FakeWorkSource(None)
    stop = FakeStopSignal(waits_before_stop=1)
    worker = Worker(
        config=_config(),
        worker_id="worker-1",
        work_source=work_source,
        processor=RecordingProcessor(),
        logger=RecordingLogger(),
    )

    worker.run(stop)

    assert work_source.lease_durations == [timedelta(seconds=1_800)]
    assert stop.waits == [1.0]


def test_worker_processes_claim_without_logging_sensitive_exception_text() -> None:
    work = _work()
    logger = RecordingLogger()
    processor = RecordingProcessor(RuntimeError("password=secret private document"))
    worker = Worker(
        config=_config(),
        worker_id="worker-1",
        work_source=FakeWorkSource(work),
        processor=processor,
        logger=logger,
    )

    worker.run(FakeStopSignal(waits_before_stop=1))

    assert processor.works == [work]
    rendered_events = repr(logger.events)
    assert "secret" not in rendered_events
    assert "private document" not in rendered_events
    assert str(work.claim.work_id) in rendered_events
    started = next(fields for event, fields in logger.events if event == "ingestion_claim_started")
    failed = next(fields for event, fields in logger.events if event == "ingestion_claim_failed")
    assert started["corpus_id"] == str(work.claim.corpus_id)
    assert started["pipeline_version"] == "corpus-ingestion-v3"
    assert failed["state"] == "failed"
    assert isinstance(failed["duration_ms"], int)


def test_worker_does_not_claim_after_cancellation() -> None:
    work_source = FakeWorkSource(_work())
    worker = Worker(
        config=_config(),
        worker_id="worker-1",
        work_source=work_source,
        processor=RecordingProcessor(),
        logger=RecordingLogger(),
    )

    worker.run(FakeStopSignal(waits_before_stop=0))

    assert work_source.lease_durations == []


def _config() -> WorkerConfig:
    return WorkerConfig.from_environment({})


def _work() -> IngestionWork:
    return IngestionWork(
        claim=WorkClaim(
            work_id=uuid4(),
            corpus_id=uuid4(),
            source_id=uuid4(),
            source_kind=SourceKind.URL,
            reason=WorkReason.INITIAL,
            lease_token=uuid4(),
            lease_expires_at=datetime.now(UTC) + timedelta(minutes=2),
        ),
        attempt_id=uuid4(),
        corpus_language="en",
        url="https://example.com/legal",
        pdf_content=None,
    )
