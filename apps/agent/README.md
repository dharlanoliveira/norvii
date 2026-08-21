# Norvii Python Agent

The agent is the internal online LangGraph runtime for Feature 005. The Go API is
the public facade; this module owns bounded retrieval, model generation, citation
validation, and abstention.

## Local development

```bash
uv sync --locked --all-groups
make ci
uv run norvii-agent
```

The service listens on `NORVII_AGENT_HOST` and `NORVII_AGENT_PORT` and exposes a
loopback-only health endpoint at `/healthz`. It reads PostgreSQL through the local
environment contract and uses the optional `NORVII_CHAT_*` OpenAI-compatible provider
settings for generation. `NORVII_CHAT_REASONING_EFFORT` defaults to `medium` and is
forwarded to compatible reasoning models.

Corpus retrieval uses the optional `NORVII_EMBEDDING_*` OpenAI-compatible embeddings
provider. The current PostgreSQL vector schema requires 1536 dimensions and defaults
to `text-embedding-3-small`. A blank `NORVII_EMBEDDING_API_KEY` reuses
`NORVII_CHAT_API_KEY`; credentials are never committed. The agent retrieves only
`ready` embedded chunks from a source's latest ready document version, so sources
must be reprocessed after enabling embeddings.
