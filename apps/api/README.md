# Norvii API

The Go API owns synchronous corpus and source commands, authoritative reads, public
HTTP contracts, PostgreSQL transactions, migrations, safe error envelopes, and the
evaluation catalog, compatibility preflight, immutable run ledger, and
opening-suggestion projection. It does not acquire remote content, parse PDFs, call
models directly, or write the Neo4j projection.

## Run locally

Use `make bootstrap` from the repository root for the complete application. For a
focused API process with `infra/.env` already configured and PostgreSQL running:

```bash
make persistence-migrate
python infra/scripts/run-with-environment.py infra/.env \
  go -C apps/api run ./cmd/server
```

The default listener is `127.0.0.1:8080`; verify it with
`curl http://127.0.0.1:8080/healthz`. Configuration keys and bounded defaults are
documented in `infra/.env.example`.

## Evaluation operations

Evaluation requires PostgreSQL migrations, the three reviewed legal corpora with
their published snapshots, and a non-empty `NORVII_EVALUATION_MAINTAINER_TOKEN` in
the ignored `infra/.env`. The API and agent must agree on
`NORVII_EVALUATION_RETRIEVAL_STRATEGY`,
`NORVII_EVALUATION_RETRIEVAL_FINGERPRINT`, `NORVII_EVALUATION_AGENT_BUILD`,
`NORVII_CHAT_MODEL`, and `NORVII_EMBEDDING_MODEL`; those values are frozen into each
run.

Import the project-owned assets only after the local stores and migrations are ready:

```bash
python infra/scripts/run-with-environment.py infra/.env \
  make -C apps/api import-evaluation-datasets
```

The importer reads only `data/corpora/*/evaluation/`, creates or returns immutable
draft revisions, and never fetches sources, calls a model, binds sources, publishes
a dataset, or changes an active snapshot. `make local-start` manages the fixed-snapshot
worker with the rest of the local environment. For focused troubleshooting, run it
under the same environment:

```bash
python infra/scripts/run-with-environment.py infra/.env \
  make -C apps/api evaluation-worker
```

The worker polls the persisted ledger, leases bounded batches, and allows three total
attempts per case, including the initial claim. After the third failed attempt it
records a safe terminal failure. It writes `.log/evaluation-worker.log` when managed
locally. Use `make local-status` to verify its readiness. It never executes normal
chat streaming.

Review and publication establish whether a dataset revision is available; that state
does not certify compatibility with any particular corpus snapshot. Before a run uses
a selected snapshot,
compatibility preflight verifies the revision's source bindings, snapshot membership,
and exact legal-locator resolution for that corpus and snapshot.

At the private agent boundary, malformed or incomplete request data is reported as
`invalid_request`; a well-formed request whose runtime cannot provide the persisted
frozen execution or retrieval identity is reported as `frozen_identity_unavailable`.
These diagnostics are distinct and content-free.

All `/api/v1/evaluation-datasets` and `/api/v1/evaluations` routes require
`Authorization: Bearer <NORVII_EVALUATION_MAINTAINER_TOKEN>` and fail closed when no
token is configured. Use dataset preflight before creating a run; incompatible
selections return bounded `dataset_not_available`, `corpus_mismatch`,
`snapshot_incompatible`, or `locator_unresolved` diagnostics and create no work.
Run inspection and comparison return immutable, safe records only. See
[`contracts/evaluation/v1/README.md`](../../contracts/evaluation/v1/README.md) for
the exact routes and response boundary.

Opening suggestions are intentionally separate and researcher-facing:

```text
GET /api/v1/corpora/{corpusId}/opening-suggestions?interfaceLanguage=en|pt
```

They return at most five original, rank-ordered questions only when a compatible
projection matches the active snapshot. They never disclose evaluation answers,
evidence, review data, scores, provider data, or prompts. The exact public contract
is [`contracts/corpus-opening-suggestions/v1/README.md`](../../contracts/corpus-opening-suggestions/v1/README.md).

## Quality and contracts

```bash
make -C apps/api ci
make persistence-integration
python .github/scripts/validate_contracts.py
```

The module contract runs dependency integrity, Go formatting, vet, race tests, and
build. Service-backed integration tests exercise migrations, transactions,
optimistic concurrency, duplicates, isolation, PDF delivery, and ingestion
publication. The public OpenAPI contract is
`contracts/corpus-ingestion/v1/openapi.json`.

HTTP failures use stable English codes and safe messages with a request identifier.
Structured access logs record request ID, method, path, status, duration, and response
size without query parameters. Logs and error responses must not expose database
credentials, PDF bytes, complete documents, or URL user information.
