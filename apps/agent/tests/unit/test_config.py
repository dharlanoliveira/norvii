from __future__ import annotations

import pytest

from norvii_agent.config import AgentConfig


def test_agent_configuration_uses_the_vector_schema_dimension_by_default(
    monkeypatch: pytest.MonkeyPatch,
) -> None:
    monkeypatch.delenv("NORVII_EMBEDDING_DIMENSIONS", raising=False)

    configuration = AgentConfig.from_environment()

    assert configuration.embedding_model == "text-embedding-3-small"
    assert configuration.embedding_dimensions == 1536


def test_agent_configuration_rejects_a_nonpositive_embedding_dimension(
    monkeypatch: pytest.MonkeyPatch,
) -> None:
    monkeypatch.setenv("NORVII_EMBEDDING_DIMENSIONS", "0")

    with pytest.raises(ValueError, match="NORVII_EMBEDDING_DIMENSIONS must be positive"):
        AgentConfig.from_environment()


def test_agent_configuration_rejects_a_dimension_that_does_not_match_the_schema(
    monkeypatch: pytest.MonkeyPatch,
) -> None:
    monkeypatch.setenv("NORVII_EMBEDDING_DIMENSIONS", "1024")

    with pytest.raises(ValueError, match="NORVII_EMBEDDING_DIMENSIONS must be 1536"):
        AgentConfig.from_environment()


def test_agent_configuration_reuses_the_chat_key_when_embedding_key_is_blank(
    monkeypatch: pytest.MonkeyPatch,
) -> None:
    monkeypatch.setenv("NORVII_CHAT_API_KEY", "chat-key")
    monkeypatch.setenv("NORVII_EMBEDDING_API_KEY", "")

    configuration = AgentConfig.from_environment()

    assert configuration.embedding_api_key == "chat-key"


def test_agent_configuration_normalizes_evaluation_execution_identity(
    monkeypatch: pytest.MonkeyPatch,
) -> None:
    monkeypatch.setenv("NORVII_EVALUATION_AGENT_BUILD", "\x1crelease-2026-08-26\x1f")
    monkeypatch.setenv("NORVII_CHAT_MODEL", "\x1dchat-model-test\x1e")
    monkeypatch.setenv("NORVII_EMBEDDING_MODEL", "\x1etext-embedding-3-small\x1d")

    configuration = AgentConfig.from_environment()

    assert configuration.evaluation_agent_build == "release-2026-08-26"
    assert configuration.chat_model == "chat-model-test"
    assert configuration.embedding_model == "text-embedding-3-small"


@pytest.mark.parametrize(
    "key",
    [
        "NORVII_EVALUATION_AGENT_BUILD",
        "NORVII_CHAT_MODEL",
        "NORVII_EMBEDDING_MODEL",
    ],
)
def test_agent_configuration_rejects_blank_evaluation_execution_identity(
    monkeypatch: pytest.MonkeyPatch, key: str
) -> None:
    monkeypatch.setenv(key, " \t ")

    with pytest.raises(ValueError, match=f"{key} must not be empty"):
        AgentConfig.from_environment()
