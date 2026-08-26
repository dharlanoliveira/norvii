"""Validated online agent configuration."""

from __future__ import annotations

import os
from dataclasses import dataclass

_REASONING_EFFORTS = frozenset({"none", "low", "medium", "high", "xhigh", "max"})
_EMBEDDING_DIMENSIONS = 1536
_EXECUTION_IDENTITY_TRIM_CHARACTERS = " \t\n\v\f\r\x1c\x1d\x1e\x1f"


@dataclass(frozen=True, slots=True)
class AgentConfig:
    """Bound network, persistence, and model settings."""

    host: str
    port: int
    postgres_host: str
    postgres_port: int
    postgres_database: str
    postgres_user: str
    postgres_password: str
    chat_base_url: str
    chat_api_key: str
    chat_model: str
    chat_reasoning_effort: str
    chat_timeout_seconds: float
    mcp_host: str = "127.0.0.1"
    mcp_port: int = 8091
    embedding_base_url: str = ""
    embedding_api_key: str = ""
    embedding_model: str = "text-embedding-3-small"
    embedding_dimensions: int = _EMBEDDING_DIMENSIONS
    embedding_timeout_seconds: float = 30.0
    neo4j_uri: str = ""
    neo4j_user: str = ""
    neo4j_password: str = ""
    neo4j_database: str = "neo4j"
    evaluation_retrieval_strategy: str = "vector"
    evaluation_retrieval_fingerprint: str = (
        "4a24773ff594172e714cb08099af9525839b5c16c0ec09da62bfae7612102523"
    )
    evaluation_agent_build: str = "norvii-agent-v1"

    @classmethod
    def from_environment(cls) -> AgentConfig:
        """Load bounded settings from the process environment."""
        return cls(
            host=os.environ.get("NORVII_AGENT_HOST", "127.0.0.1"),
            port=_positive_int("NORVII_AGENT_PORT", 8090),
            postgres_host=os.environ.get("NORVII_POSTGRES_HOST", "localhost"),
            postgres_port=_positive_int("NORVII_POSTGRES_PORT", 5432),
            postgres_database=os.environ.get("NORVII_POSTGRES_DATABASE", "norvii"),
            postgres_user=os.environ.get("NORVII_POSTGRES_USER", "norvii"),
            postgres_password=os.environ.get("NORVII_POSTGRES_PASSWORD", ""),
            chat_base_url=os.environ.get("NORVII_CHAT_BASE_URL", "").strip(),
            chat_api_key=os.environ.get("NORVII_CHAT_API_KEY", ""),
            chat_model=_normalized_execution_identity_value(
                "NORVII_CHAT_MODEL", "gpt-4o-mini"
            ),
            chat_reasoning_effort=_reasoning_effort("NORVII_CHAT_REASONING_EFFORT", "medium"),
            chat_timeout_seconds=_positive_float("NORVII_CHAT_TIMEOUT_SECONDS", 30.0),
            mcp_host=os.environ.get("NORVII_MCP_HOST", "127.0.0.1"),
            mcp_port=_positive_int("NORVII_MCP_PORT", 8091),
            embedding_base_url=os.environ.get("NORVII_EMBEDDING_BASE_URL", "").strip(),
            embedding_api_key=(
                os.environ.get("NORVII_EMBEDDING_API_KEY", "").strip()
                or os.environ.get("NORVII_CHAT_API_KEY", "")
            ),
            embedding_model=_normalized_execution_identity_value(
                "NORVII_EMBEDDING_MODEL", "text-embedding-3-small"
            ),
            embedding_dimensions=_embedding_dimensions(),
            embedding_timeout_seconds=_positive_float("NORVII_EMBEDDING_TIMEOUT_SECONDS", 30.0),
            neo4j_uri=os.environ.get("NORVII_NEO4J_URI", "").strip(),
            neo4j_user=os.environ.get("NORVII_NEO4J_USER", "").strip(),
            neo4j_password=os.environ.get("NORVII_NEO4J_PASSWORD", ""),
            neo4j_database=os.environ.get("NORVII_NEO4J_DATABASE", "neo4j").strip(),
            evaluation_retrieval_strategy=os.environ.get(
                "NORVII_EVALUATION_RETRIEVAL_STRATEGY", "vector"
            ).strip(),
            evaluation_retrieval_fingerprint=os.environ.get(
                "NORVII_EVALUATION_RETRIEVAL_FINGERPRINT",
                "4a24773ff594172e714cb08099af9525839b5c16c0ec09da62bfae7612102523",
            ).strip(),
            evaluation_agent_build=_normalized_execution_identity_value(
                "NORVII_EVALUATION_AGENT_BUILD", "norvii-agent-v1"
            ),
        )


def _positive_int(key: str, fallback: int) -> int:
    value = int(os.environ.get(key, str(fallback)))
    if value < 1:
        raise ValueError(f"{key} must be positive")
    return value


def _normalized_execution_identity_value(key: str, fallback: str) -> str:
    """Normalize a runtime identity before it can select or persist an evaluation run."""
    value = os.environ.get(key, fallback).strip(_EXECUTION_IDENTITY_TRIM_CHARACTERS)
    if not value:
        raise ValueError(f"{key} must not be empty")
    return value


def _positive_float(key: str, fallback: float) -> float:
    value = float(os.environ.get(key, str(fallback)))
    if value <= 0:
        raise ValueError(f"{key} must be positive")
    return value


def _reasoning_effort(key: str, fallback: str) -> str:
    value = os.environ.get(key, fallback).strip().lower()
    if value not in _REASONING_EFFORTS:
        raise ValueError(f"{key} must be one of {', '.join(sorted(_REASONING_EFFORTS))}")
    return value


def _embedding_dimensions() -> int:
    dimensions = _positive_int("NORVII_EMBEDDING_DIMENSIONS", _EMBEDDING_DIMENSIONS)
    if dimensions != _EMBEDDING_DIMENSIONS:
        raise ValueError(
            f"NORVII_EMBEDDING_DIMENSIONS must be {_EMBEDDING_DIMENSIONS} for the current schema"
        )
    return dimensions
