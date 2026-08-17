"""Graph projection connectivity adapter."""

from __future__ import annotations

from typing import TYPE_CHECKING, ClassVar

from neo4j import Driver, EagerResult, GraphDatabase, RoutingControl
from neo4j.exceptions import Neo4jError

from norvii_ingestion.publication.persistence.errors import PersistenceConnectionError

if TYPE_CHECKING:
    from norvii_ingestion.publication.persistence.config import Neo4jConfiguration


class Neo4jStore:
    """Verify graph projection storage through the production Bolt driver."""

    name: ClassVar[str] = "Neo4j"

    def __init__(self, driver: Driver, database: str) -> None:
        self._driver = driver
        self._database = database

    @classmethod
    def connect(cls, configuration: Neo4jConfiguration, timeout_seconds: int) -> Neo4jStore:
        """Construct an authenticated driver with bounded acquisition and retry behavior."""
        try:
            driver = GraphDatabase.driver(
                configuration.uri,
                auth=(configuration.user, configuration.password),
                connection_timeout=float(timeout_seconds),
                connection_acquisition_timeout=float(timeout_seconds),
                max_transaction_retry_time=0.0,
                max_connection_pool_size=2,
            )
        except (Neo4jError, ValueError) as error:
            raise PersistenceConnectionError("Create Neo4j driver failed.") from error
        return cls(driver, configuration.database)

    def verify(self) -> None:
        """Execute a constant read-only Cypher query over Bolt."""
        try:
            result: EagerResult = self._driver.execute_query(
                "RETURN 1 AS ready",
                routing_=RoutingControl.READ,
                database_=self._database,
            )
        except (Neo4jError, OSError, ValueError) as error:
            raise PersistenceConnectionError("Execute Neo4j readiness query failed.") from error
        if len(result.records) != 1:
            raise PersistenceConnectionError(
                "Neo4j readiness query returned an unexpected record count."
            )

    def close(self) -> None:
        """Release the graph driver and connection pool."""
        self._driver.close()
