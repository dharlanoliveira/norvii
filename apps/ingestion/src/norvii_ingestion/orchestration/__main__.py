"""Ingestion worker production process entry point."""

from __future__ import annotations

import logging
import os
import signal
import socket
from datetime import UTC, datetime
from threading import Event
from typing import TYPE_CHECKING

from norvii_ingestion.acquisition.https import HttpsAcquirer
from norvii_ingestion.config import WorkerConfig
from norvii_ingestion.enrichment.embedding import OpenAICompatibleEmbeddingProvider
from norvii_ingestion.extraction.html import HtmlExtractor
from norvii_ingestion.extraction.pdf import PdfExtractor
from norvii_ingestion.orchestration.composition import (
    PostgresWorkSource,
    StructuredEventLogger,
)
from norvii_ingestion.orchestration.processor import ArtifactExtractors, IngestionProcessor
from norvii_ingestion.orchestration.worker import Worker
from norvii_ingestion.publication.persistence.config import (
    EnvironmentConfigurationLoader,
)
from norvii_ingestion.publication.postgres.repository import (
    PostgresWorkRepository,
    WorkRepositoryError,
)

_LOGGER = logging.getLogger(__name__)

if TYPE_CHECKING:
    from types import FrameType


def main() -> int:
    """Compose and run the PostgreSQL-backed worker until SIGINT or SIGTERM."""
    logging.basicConfig(level=logging.INFO, format="%(asctime)s %(levelname)s %(message)s")
    try:
        worker_config = WorkerConfig.from_environment(os.environ)
        persistence_config = EnvironmentConfigurationLoader(os.environ).load()
        repository = PostgresWorkRepository.connect(
            persistence_config.postgres,
            persistence_config.timeout_seconds,
        )
    except (ValueError, WorkRepositoryError):
        return _startup_failure()

    stop = Event()

    def request_stop(_signal_number: int, _frame: FrameType | None) -> None:
        stop.set()

    signal.signal(signal.SIGINT, request_stop)
    signal.signal(signal.SIGTERM, request_stop)
    worker_id = f"{socket.gethostname()}-{os.getpid()}"
    worker = Worker(
        config=worker_config,
        worker_id=worker_id,
        work_source=PostgresWorkSource(repository, _utc_now),
        processor=IngestionProcessor(
            repository=repository,
            acquirer=HttpsAcquirer(worker_config),
            extractors=ArtifactExtractors(html=HtmlExtractor(), pdf=PdfExtractor()),
            pipeline_version=worker_config.pipeline_version,
            clock=_utc_now,
            embedding_provider=OpenAICompatibleEmbeddingProvider(
                endpoint=worker_config.embedding_endpoint,
                api_key=worker_config.embedding_api_key,
                model=worker_config.embedding_model,
                dimensions=worker_config.embedding_dimensions,
                timeout_seconds=worker_config.embedding_timeout_seconds,
                batch_size=worker_config.embedding_batch_size,
            ),
            embedding_model=worker_config.embedding_model,
        ),
        logger=StructuredEventLogger(_LOGGER),
    )
    try:
        worker.run(stop)
    except WorkRepositoryError:
        return _storage_failure()
    finally:
        repository.close()
    return 0


def _utc_now() -> datetime:
    return datetime.now(UTC)


def _startup_failure() -> int:
    _LOGGER.error("Ingestion worker startup failed.")
    return 1


def _storage_failure() -> int:
    _LOGGER.error("Ingestion worker storage operation failed.")
    return 1


if __name__ == "__main__":
    raise SystemExit(main())
