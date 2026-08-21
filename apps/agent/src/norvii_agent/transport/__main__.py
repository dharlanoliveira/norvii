"""Start the local Norvii LangGraph agent service."""

from __future__ import annotations

from norvii_agent.config import AgentConfig
from norvii_agent.graph import GroundedChatGraph
from norvii_agent.providers import (
    OpenAICompatibleChatModel,
    OpenAICompatibleEmbeddingProvider,
)
from norvii_agent.retrieval import PostgresRetriever
from norvii_agent.transport.server import AgentHTTPServer


def main() -> None:
    """Build provider adapters and serve the internal graph endpoint."""
    configuration = AgentConfig.from_environment()
    embeddings = OpenAICompatibleEmbeddingProvider(
        configuration.embedding_base_url,
        configuration.embedding_api_key,
        configuration.embedding_model,
        configuration.embedding_dimensions,
        configuration.embedding_timeout_seconds,
    )
    retriever = PostgresRetriever(configuration, embeddings)
    model = OpenAICompatibleChatModel(
        configuration.chat_base_url,
        configuration.chat_api_key,
        configuration.chat_model,
        configuration.chat_timeout_seconds,
        configuration.chat_reasoning_effort,
    )
    server = AgentHTTPServer(
        (configuration.host, configuration.port),
        lambda: GroundedChatGraph(retriever, model),
    )
    server.serve_forever()


if __name__ == "__main__":
    main()
