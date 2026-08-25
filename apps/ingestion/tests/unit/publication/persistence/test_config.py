from collections.abc import Mapping

import pytest

from norvii_ingestion.publication.persistence.config import (
    ConfigurationError,
    EnvironmentConfigurationLoader,
)


def test_loader_accepts_complete_environment() -> None:
    configuration = EnvironmentConfigurationLoader(valid_environment()).load()

    assert configuration.postgres.port == 5432
    assert configuration.neo4j.uri == "neo4j://localhost:7687"
    assert configuration.timeout_seconds == 5
    assert configuration.snapshot_api_base_url == "http://127.0.0.1:8080"


@pytest.mark.parametrize(
    ("variable", "value"),
    [
        ("NORVII_POSTGRES_HOST", ""),
        ("NORVII_POSTGRES_PORT", "70000"),
        ("NORVII_NEO4J_URI", "http://localhost:7687"),
        ("NORVII_PERSISTENCE_TIMEOUT_SECONDS", "11"),
        ("NORVII_INGESTION_API_BASE_URL", "ftp://localhost:8080"),
    ],
)
def test_loader_rejects_invalid_environment_without_disclosing_secrets(
    variable: str,
    value: str,
) -> None:
    postgres_secret = "postgres-secret-value"
    neo4j_secret = "neo4j-secret-value"
    environment = valid_environment() | {
        "NORVII_POSTGRES_PASSWORD": postgres_secret,
        "NORVII_NEO4J_PASSWORD": neo4j_secret,
        variable: value,
    }
    loader = EnvironmentConfigurationLoader(environment)

    with pytest.raises(ConfigurationError) as error:
        loader.load()

    assert variable in str(error.value)
    assert postgres_secret not in str(error.value)
    assert neo4j_secret not in str(error.value)


def valid_environment() -> Mapping[str, str]:
    return {
        "NORVII_POSTGRES_HOST": "localhost",
        "NORVII_POSTGRES_PORT": "5432",
        "NORVII_POSTGRES_DATABASE": "norvii",
        "NORVII_POSTGRES_USER": "norvii",
        "NORVII_POSTGRES_PASSWORD": "local-postgres-secret",
        "NORVII_NEO4J_URI": "neo4j://localhost:7687",
        "NORVII_NEO4J_USER": "neo4j",
        "NORVII_NEO4J_PASSWORD": "local-neo4j-secret",
        "NORVII_NEO4J_DATABASE": "neo4j",
        "NORVII_PERSISTENCE_TIMEOUT_SECONDS": "5",
    }
