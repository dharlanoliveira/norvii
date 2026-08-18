from __future__ import annotations

from datetime import timedelta

import pytest

from norvii_ingestion.config import ConfigurationError, WorkerConfig


def test_worker_config_uses_bounded_defaults() -> None:
    config = WorkerConfig.from_environment({})

    assert config.poll_interval == timedelta(seconds=1)
    assert config.lease_duration == timedelta(seconds=120)
    assert config.max_source_bytes == 10 * 1024 * 1024
    assert config.max_redirects == 5
    assert config.pipeline_version == "corpus-ingestion-v2"


def test_worker_config_rejects_lease_shorter_than_poll_interval() -> None:
    environment = {
        "NORVII_INGESTION_POLL_SECONDS": "10",
        "NORVII_INGESTION_LEASE_SECONDS": "5",
    }

    with pytest.raises(ConfigurationError, match="lease"):
        WorkerConfig.from_environment(environment)
