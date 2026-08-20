# Quickstart: Grounded RAG Chat

This guide is the acceptance target for Feature 005. It uses deterministic fake providers in
automated tests and optional configured providers for local demonstration.

## Prerequisites

- Docker Compose v2 with the existing PostgreSQL/pgvector and Neo4j services
- Go 1.26, Python 3.13 with uv 0.11, and Node.js 24 with npm
- Feature 004 initialized with at least one ready document
- Optional OpenAI-compatible embedding and chat provider configuration in `infra/.env.example`

## Start the environment

```bash
make bootstrap
```

If no model credentials are configured, the catalog and document viewer remain usable and chat
shows the localized unavailable-provider state. Automated tests never require live credentials.

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
make -C apps/ingestion ci
make -C apps/web ci
python .github/scripts/validate_contracts.py
python .github/scripts/validate_repository_language.py
git diff --check
```

Feature-specific integration tests use deterministic provider fakes and isolated PostgreSQL data.
No CI job may call a live model or embedding provider.
