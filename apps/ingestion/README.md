# Norvii Ingestion

The Python worker claims leased work from PostgreSQL, safely acquires public HTTPS
content or stored PDF bytes, extracts complete normalized text and legal units, and
creates legal-aware retrieval chunks, generates embeddings, and atomically publishes
immutable document revisions. The module uses class-based acquisition, extraction,
orchestration, enrichment, and repository boundaries.

## Run locally

Use `make bootstrap` from the repository root for the managed worker. For focused
development with `infra/.env` configured, migrations applied, and PostgreSQL
running:

```bash
uv sync --directory apps/ingestion --locked --all-groups
python infra/scripts/run-with-environment.py infra/.env \
  uv run --directory apps/ingestion norvii-ingestion-worker
```

Stop with SIGINT or SIGTERM. Poll, lease, acquisition, redirect, size, timeout, and
pipeline-version settings are documented in `infra/.env.example`.

The default `corpus-ingestion-v3` pipeline embeds chunks with
`text-embedding-3-small` and the fixed 1536-dimensional PostgreSQL vector schema.
Configure the `NORVII_EMBEDDING_*` variables in the ignored `infra/.env`; a blank
embedding API key reuses `NORVII_CHAT_API_KEY`. Reprocess a ready source through the
workspace to create an immutable v3 document version and backfill its chunks. A
provider failure records a safe failed attempt and leaves the preceding ready document
unchanged.

## Build a graph release

Semantic extraction is performed only during offline ingestion. After reprocessing sources and
publishing a snapshot, build its derived Neo4j projection explicitly. This command never changes
the canonical PostgreSQL source documents or activates a snapshot:

```bash
python infra/scripts/run-with-environment.py infra/.env \
  uv run --directory apps/ingestion norvii-build-graph-release \
  --corpus-id <corpus-id> --snapshot-id <snapshot-id>
```

The build uses only supported extraction records for the named published snapshot. It is
idempotent for the same semantic manifest; a graph or hybrid chat request safely remains
unavailable until the release is ready.

## Quality and contracts

```bash
make -C apps/ingestion ci
make -C apps/ingestion coverage-integration
python .github/scripts/validate_contracts.py
```

The module contract enforces the lockfile, Ruff format and lint, strict mypy, unit
tests, and package build. Service-backed tests cover URL safety, HTML/PDF extraction,
leases, crash recovery, idempotent publication, changed revisions, and preservation
of prior ready documents after failure.

The worker never trusts proxy configuration for acquisition and rejects non-public
addresses at every redirect. Structured lifecycle logs contain allowlisted worker,
work, corpus, source, kind, reason, state, pipeline, duration, count, and safe-category
fields, never credentials, origin bytes, complete normalized documents, or URL user
information.
