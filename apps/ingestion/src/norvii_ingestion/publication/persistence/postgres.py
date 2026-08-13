"""Canonical PostgreSQL connectivity adapter."""

from __future__ import annotations

from typing import TYPE_CHECKING, ClassVar

import psycopg

from norvii_ingestion.publication.persistence.errors import PersistenceConnectionError

if TYPE_CHECKING:
    from norvii_ingestion.publication.persistence.config import PostgresConfiguration


class PostgresStore:
    """Verify canonical storage through the production psycopg driver."""

    name: ClassVar[str] = "PostgreSQL"

    def __init__(self, connection: psycopg.Connection[tuple[object, ...]]) -> None:
        self._connection = connection

    @classmethod
    def connect(cls, configuration: PostgresConfiguration, timeout_seconds: int) -> PostgresStore:
        """Open an authenticated connection with bounded connect and statement timeouts."""
        try:
            connection = psycopg.connect(
                host=configuration.host,
                port=configuration.port,
                dbname=configuration.database,
                user=configuration.user,
                password=configuration.password,
                connect_timeout=timeout_seconds,
                options=f"-c statement_timeout={timeout_seconds * 1000}",
                autocommit=True,
            )
        except psycopg.Error as error:
            raise PersistenceConnectionError("Connect to PostgreSQL failed.") from error
        return cls(connection)

    def verify(self) -> None:
        """Execute a constant read-only SQL query."""
        try:
            with self._connection.cursor() as cursor:
                cursor.execute("SELECT 1")
                row = cursor.fetchone()
        except psycopg.Error as error:
            raise PersistenceConnectionError(
                "Execute PostgreSQL readiness query failed."
            ) from error
        if row != (1,):
            raise PersistenceConnectionError(
                "PostgreSQL readiness query returned an unexpected value."
            )

    def close(self) -> None:
        """Release the canonical-store connection."""
        self._connection.close()
