"""Explicit command for rebuilding a derived graph release from a published snapshot."""

from __future__ import annotations

import argparse
import os
import sys
from uuid import UUID

from norvii_ingestion.graph_projection import GraphProjectionBuildError, GraphReleaseBuilder
from norvii_ingestion.publication.persistence.config import (
    ConfigurationError,
    EnvironmentConfigurationLoader,
)
from norvii_ingestion.publication.persistence.neo4j import Neo4jStore
from norvii_ingestion.publication.postgres.repository import PostgresWorkRepository


def main() -> int:
    """Build one named snapshot graph without changing canonical source artifacts."""
    parser = argparse.ArgumentParser(description="Build a Norvii graph release for a snapshot.")
    parser.add_argument("--corpus-id", type=UUID, required=True)
    parser.add_argument("--snapshot-id", type=UUID, required=True)
    arguments = parser.parse_args()
    repository: PostgresWorkRepository | None = None
    graph: Neo4jStore | None = None
    try:
        configuration = EnvironmentConfigurationLoader(os.environ).load()
        repository = PostgresWorkRepository.connect(
            configuration.postgres, configuration.timeout_seconds
        )
        graph = Neo4jStore.connect(configuration.neo4j, configuration.timeout_seconds)
        summary = GraphReleaseBuilder(repository.connection, graph).build(
            arguments.corpus_id, arguments.snapshot_id
        )
    except (ConfigurationError, GraphProjectionBuildError, ValueError) as error:
        print(f"Graph release build failed: {error}", file=sys.stderr)
        return 1
    finally:
        if graph is not None:
            graph.close()
        if repository is not None:
            repository.close()
    state = "reused" if summary.reused else "built"
    print(
        f"Graph release {state}: {summary.release_id} "
        f"({summary.entity_count} entities, {summary.relationship_count} relationships)."
    )
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
