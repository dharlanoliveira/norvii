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
    assert config.pipeline_version == "corpus-ingestion-v3"
    assert config.embedding_model == "text-embedding-3-small"
    assert config.embedding_dimensions == 1536


def test_worker_config_rejects_lease_shorter_than_poll_interval() -> None:
    environment = {
        "NORVII_INGESTION_POLL_SECONDS": "10",
        "NORVII_INGESTION_LEASE_SECONDS": "5",
    }

    with pytest.raises(ConfigurationError, match="lease"):
        WorkerConfig.from_environment(environment)


def test_worker_config_rejects_provider_timeouts_that_can_expire_the_lease() -> None:
    environment = {
        "NORVII_INGESTION_LEASE_SECONDS": "120",
        "NORVII_SEMANTIC_TIMEOUT_SECONDS": "180",
    }

    with pytest.raises(ConfigurationError, match="semantic and embedding timeouts"):
        WorkerConfig.from_environment(environment)


def test_worker_config_reuses_chat_key_when_embedding_key_is_not_set() -> None:
    config = WorkerConfig.from_environment({"NORVII_CHAT_API_KEY": "local-key"})

    assert config.embedding_api_key == "local-key"


def test_worker_config_rejects_a_dimension_that_does_not_match_the_vector_schema() -> None:
    with pytest.raises(ConfigurationError, match="NORVII_EMBEDDING_DIMENSIONS must be 1536"):
        WorkerConfig.from_environment({"NORVII_EMBEDDING_DIMENSIONS": "1024"})
