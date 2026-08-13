import os

import pytest

from norvii_ingestion.publication.persistence.config import EnvironmentConfigurationLoader
from norvii_ingestion.publication.persistence.neo4j import Neo4jStore
from norvii_ingestion.publication.persistence.postgres import PostgresStore
from norvii_ingestion.publication.persistence.verifier import PersistenceVerifier


@pytest.mark.integration
def test_python_runtime_verifies_both_stores_without_creating_product_artifacts() -> None:
    configuration = EnvironmentConfigurationLoader(os.environ).load()
    verifier = PersistenceVerifier(
        [
            PostgresStore.connect(configuration.postgres, configuration.timeout_seconds),
            Neo4jStore.connect(configuration.neo4j, configuration.timeout_seconds),
        ]
    )

    results = verifier.verify()

    assert [result.store for result in results] == ["PostgreSQL", "Neo4j"]
