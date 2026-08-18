# Norvii Ingestion

The Python worker claims leased work from PostgreSQL, safely acquires public HTTPS
content or stored PDF bytes, extracts complete normalized text and legal units, and
atomically publishes immutable document revisions. The module uses class-based
acquisition, extraction, orchestration, and repository boundaries.

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
