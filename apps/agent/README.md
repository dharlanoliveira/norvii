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

## Fixed-snapshot evaluation boundary

The same process also exposes the private, non-streaming
`POST /v1/evaluations/execute` transport for the managed Go evaluation worker. It is
not a public command or a browser API. The Go API supplies one persisted corpus,
snapshot, question, interface language, frozen retrieval configuration, and execution
identity. Malformed or incomplete request data is rejected with the bounded
`invalid_request` outcome. A well-formed request whose requested execution or
retrieval identity does not match the running configuration is rejected with
`frozen_identity_unavailable` instead.

Set `NORVII_EVALUATION_RETRIEVAL_STRATEGY` (`vector` or `hybrid`),
`NORVII_EVALUATION_RETRIEVAL_FINGERPRINT` (a lowercase SHA-256), and
`NORVII_EVALUATION_AGENT_BUILD` consistently with the API. The execution identity
also includes the configured `NORVII_CHAT_MODEL` and `NORVII_EMBEDDING_MODEL`. A
fixed-snapshot evaluation never resolves the active snapshot and never changes the
public chat SSE flow. Transport errors return safe codes (`invalid_request` for an
invalid request, `frozen_identity_unavailable` for runtime frozen-identity drift,
or `evaluation_unavailable` for another execution failure) without provider payloads,
prompts, credentials, or source text.

## MCP research server

Feature 010 adds a separate MCP entry point. In a local MCP client configuration,
use stdio:

```bash
uv run norvii-mcp --transport stdio
```

For Docker, the `mcp` Compose profile runs the same server over Streamable HTTP at
`/mcp`. `make bootstrap` and `make local-start` include this profile and manage its
log at `.log/mcp.log`; see the
[feature quickstart](../../specs/010-mcp-research-tools/quickstart.md). The server
exposes bounded, read-only corpus research tools and reusable prompts. It always
resolves the active published snapshot and must not be treated as legal advice.
