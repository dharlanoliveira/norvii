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

## Graph-ready snapshot lifecycle

Semantic extraction is performed only during offline ingestion. For normal worker processing,
the worker first publishes the canonical document artifacts, then stages an immutable candidate
snapshot through the Go API, builds that snapshot's derived Neo4j graph release, and finally asks
the API to activate the candidate. The API activation boundary requires a `ready` graph release;
it rejects a non-ready graph with `graph_release_not_ready`, leaving the preceding active snapshot
unchanged.

The standalone graph command is not part of the normal worker flow. Use it only to recover or
rebuild the derived projection for an existing snapshot:

```bash
python infra/scripts/run-with-environment.py infra/.env \
  uv run --directory apps/ingestion norvii-build-graph-release \
  --corpus-id <corpus-id> --snapshot-id <snapshot-id>
```

The build uses only supported extraction records for the named snapshot. It is idempotent for the
same semantic manifest and never activates a snapshot or changes canonical PostgreSQL source
documents. Graph and hybrid chat remain unavailable until a ready release is activated through the
API.

## Evaluation prerequisites

Ingestion supplies the reviewed official-source artifacts and normalized legal locators that
evaluation requires. The Go API owns evaluation source binding, dataset review and availability,
compatibility preflight, and snapshot publication and activation. Ingestion does not import
evaluation datasets, publish opening suggestions, create runs, score cases, or execute the
evaluation worker; those responsibilities belong to `apps/api/`.

Review and publication establish whether an evaluation revision is available; that state
is independent of any particular snapshot. Before a run uses a selected corpus snapshot,
API compatibility preflight verifies that every manifest source is bound within the same
corpus, is a member of that snapshot, and resolves every required legal locator uniquely
there. If required source or legal-locator artifacts are missing, reprocess the affected source
material through ingestion before using the API-owned binding and snapshot publication flow.
Other preflight failures must be corrected through that API-owned flow; do not alter
`data/corpora/*/evaluation/` assets to bypass it. The API importer command and maintainer
boundary are documented in [`apps/api/README.md`](../api/README.md).

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
