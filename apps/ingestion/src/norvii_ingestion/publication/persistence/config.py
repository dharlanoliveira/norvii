"""Validated configuration for local persistence adapters."""

from collections.abc import Mapping
from dataclasses import dataclass
from urllib.parse import urlparse


class ConfigurationError(ValueError):
    """Indicate that the persistence environment contract is incomplete or invalid."""


@dataclass(frozen=True, slots=True)
class PostgresConfiguration:
    """Contain discrete PostgreSQL connection fields without a credential-bearing URL."""

    host: str
    port: int
    database: str
    user: str
    password: str


@dataclass(frozen=True, slots=True)
class Neo4jConfiguration:
    """Contain Bolt connection settings for the graph projection."""

    uri: str
    user: str
    password: str
    database: str


@dataclass(frozen=True, slots=True)
class PersistenceConfiguration:
    """Contain validated settings for one ingestion process."""

    postgres: PostgresConfiguration
    neo4j: Neo4jConfiguration
    timeout_seconds: int
    snapshot_api_base_url: str


class EnvironmentConfigurationLoader:
    """Load the version 1 local persistence contract from an injected environment."""

    def __init__(self, environment: Mapping[str, str]) -> None:
        self._environment = environment

    def load(self) -> PersistenceConfiguration:
        """Return validated immutable configuration or raise a secret-safe error."""
        postgres_port = self._required_integer("NORVII_POSTGRES_PORT", minimum=1, maximum=65535)
        timeout_seconds = self._required_integer(
            "NORVII_PERSISTENCE_TIMEOUT_SECONDS",
            minimum=1,
            maximum=10,
        )
        neo4j_uri = self._required_neo4j_uri()
        snapshot_api_base_url = self._snapshot_api_base_url()

        return PersistenceConfiguration(
            postgres=PostgresConfiguration(
                host=self._required_value("NORVII_POSTGRES_HOST"),
                port=postgres_port,
                database=self._required_value("NORVII_POSTGRES_DATABASE"),
                user=self._required_value("NORVII_POSTGRES_USER"),
                password=self._required_value("NORVII_POSTGRES_PASSWORD"),
            ),
            neo4j=Neo4jConfiguration(
                uri=neo4j_uri,
                user=self._required_value("NORVII_NEO4J_USER"),
                password=self._required_value("NORVII_NEO4J_PASSWORD"),
                database=self._required_value("NORVII_NEO4J_DATABASE"),
            ),
            timeout_seconds=timeout_seconds,
            snapshot_api_base_url=snapshot_api_base_url,
        )

    def _required_value(self, name: str) -> str:
        value = self._environment.get(name, "")
        if not value.strip():
            raise ConfigurationError(f"Configuration variable {name} is required.")
        return value

    def _required_integer(self, name: str, *, minimum: int, maximum: int) -> int:
        value = self._required_value(name)
        message = (
            f"Configuration variable {name} must be an integer from {minimum} through {maximum}."
        )
        try:
            parsed = int(value)
        except ValueError as error:
            raise ConfigurationError(message) from error
        if parsed < minimum or parsed > maximum:
            raise ConfigurationError(message)
        return parsed

    def _required_neo4j_uri(self) -> str:
        uri = self._required_value("NORVII_NEO4J_URI")
        parsed = urlparse(uri)
        if parsed.scheme != "neo4j" or not parsed.hostname or parsed.username or parsed.password:
            raise ConfigurationError(
                "Configuration variable NORVII_NEO4J_URI must be a credential-free neo4j URI."
            )
        return uri

    def _snapshot_api_base_url(self) -> str:
        value = self._environment.get("NORVII_INGESTION_API_BASE_URL", "http://127.0.0.1:8080")
        parsed = urlparse(value)
        is_credential_free_http = (
            parsed.scheme in {"http", "https"}
            and parsed.hostname is not None
            and parsed.username is None
            and parsed.password is None
        )
        if not is_credential_free_http:
            raise ConfigurationError(
                "NORVII_INGESTION_API_BASE_URL must be a credential-free HTTP URL."
            )
        return value.rstrip("/")
