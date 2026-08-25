# Quickstart: Grounded RAG Chat

This guide is the acceptance target for Feature 005. It uses deterministic fake providers in
automated tests and optional configured providers for local demonstration.

## Prerequisites

- Docker Compose v2 with the existing PostgreSQL/pgvector and Neo4j services
- Go 1.26, Python 3.13 with uv 0.11, and Node.js 24 with npm
- The online `apps/agent/` Python LangGraph service started by `make bootstrap`
- Feature 004 initialized with at least one ready document
- Optional OpenAI-compatible embedding and chat provider configuration in `infra/.env.example`

`NORVII_AGENT_BASE_URL` points the Go facade to the internal agent (the local default is
`http://127.0.0.1:8090`). `NORVII_CHAT_BASE_URL` must point to the provider's
OpenAI-compatible chat-completions endpoint when model-backed answers are enabled.
`NORVII_EMBEDDING_BASE_URL` must point to its embeddings endpoint. The default model is
`text-embedding-3-small`, and the current PostgreSQL vector schema requires
`NORVII_EMBEDDING_DIMENSIONS=1536`. An empty `NORVII_EMBEDDING_API_KEY` reuses the
ignored `NORVII_CHAT_API_KEY` for a compatible provider.

## Start the environment

```bash
make bootstrap
```

If no chat endpoint is configured, the catalog and document viewer remain usable and the agent
emits a localized provider-unavailable failure through the Go facade. Automated tests never
require live credentials.

## Backfill sources published before embeddings

1. Configure the embeddings endpoint and credentials in the ignored `infra/.env`.
2. Start the managed environment with `make bootstrap`.
3. In a ready corpus, choose **Reprocess source** for every source that predates
   `corpus-ingestion-v3`.
4. Wait until each source returns to the ready state. The worker publishes a new immutable
   document version, its addressable units, and ready 1536-dimensional retrieval chunks in one
   transaction.

If embedding acquisition, response validation, or persistence fails, the processing attempt is
marked as failed and the prior ready document remains the source's latest evidence. Retry only
after correcting the provider configuration or transient provider condition.

## Validate the grounded answer journey

1. Open an enabled corpus containing a ready document.
2. Ask a deterministic question whose answer appears in that document.
3. Observe pending, evidence, streaming, and completed states.
4. Confirm that the answer contains numbered evidence references.
5. Open a reference and confirm that the source and document unit are selected in the source view.
6. Switch the interface to Portuguese and repeat the state journey; legal excerpts remain in the
   source language.

## Validate abstention and safety

- Ask a question absent from the active corpus and confirm an insufficient-evidence abstention.
- Ask while all sources are pending or failed and confirm no answer provider call is made.
- Include an instruction-like sentence in a legal fixture and confirm it is rendered only as
  evidence, never treated as system guidance.
- Cancel a streaming response and confirm no terminal completed state is shown.
- Simulate provider timeout, malformed stream, and unresolved citation; confirm localized safe
  outcomes without internal details or unrelated source content.

## Automated quality gates

```bash
make -C apps/api ci
make -C apps/agent ci
make -C apps/ingestion ci
make -C apps/web ci
python .github/scripts/validate_contracts.py
python .github/scripts/validate_repository_language.py
git diff --check
```

Feature-specific integration tests use deterministic provider fakes and isolated PostgreSQL data.
No CI job may call a live model or embedding provider.

## Deterministic service-backed acceptance evidence

The browser journey below uses the public HTTP/SSE boundary with deterministic service fixtures;
it does not require a configured provider or credentials. It verifies active corpus and snapshot
identity, immutable citation navigation, foreign-corpus citation rejection, abstention, and
cancellation terminal rendering.

```bash
npm --prefix apps/web run test:e2e -- grounded-rag-chat.spec.ts
```

On 2026-08-25, this command passed twice: 4 browser scenarios in 4.49 seconds on each run.
The runs used no live provider.
