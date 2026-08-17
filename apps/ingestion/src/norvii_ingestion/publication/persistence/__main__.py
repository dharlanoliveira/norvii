"""Command-line entry point for ingestion persistence verification."""

from __future__ import annotations

import os
import sys

from norvii_ingestion.publication.persistence.config import (
    ConfigurationError,
    EnvironmentConfigurationLoader,
)
from norvii_ingestion.publication.persistence.errors import PersistenceError
from norvii_ingestion.publication.persistence.neo4j import Neo4jStore
from norvii_ingestion.publication.persistence.postgres import PostgresStore
from norvii_ingestion.publication.persistence.verifier import (
    PersistenceVerificationError,
    PersistenceVerifier,
)


def main() -> int:
    """Verify both stores and return a shell-compatible result."""
    postgres: PostgresStore | None = None
    neo4j: Neo4jStore | None = None
    try:
        configuration = EnvironmentConfigurationLoader(os.environ).load()
        postgres = PostgresStore.connect(configuration.postgres, configuration.timeout_seconds)
        neo4j = Neo4jStore.connect(configuration.neo4j, configuration.timeout_seconds)
        results = PersistenceVerifier([postgres, neo4j]).verify()
    except (ConfigurationError, PersistenceError, PersistenceVerificationError) as error:
        if postgres is not None and neo4j is None:
            postgres.close()
        print(f"Persistence verification failed: {error}", file=sys.stderr)
        return 1

    for result in results:
        print(f"{result.store} connectivity verified.")
    print("Persistence verification succeeded.")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
