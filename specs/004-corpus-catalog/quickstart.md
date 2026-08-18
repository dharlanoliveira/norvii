# Quickstart: Corpus Catalog and Ingestion

This guide is the acceptance target for the implemented feature. The one-command bootstrap entry point remains `make bootstrap` through the project skill.

## Prerequisites

- Docker with Compose v2
- Go 1.26
- Python 3.13 and uv 0.11
- Node.js 24 and npm
- Source-network access to the official LGPD and GDPR HTTPS pages for the live seed journey
- Local secrets configured from `infra/.env.example`

## Start the complete environment

```bash
make bootstrap
```

Expected outcomes:

- PostgreSQL and Neo4j are healthy;
- migrations and idempotent initial records are applied;
- Go API, Python ingestion worker, and React client are running;
- component logs are written separately under `.log/`;
- the two initial sources reach `ready` or a safe retryable `failed` state within the bounded acquisition window.

Open the URL printed by bootstrap. The catalog must show one English EU corpus and one Portuguese Brazilian corpus. When source access succeeds, each workspace must show one real official source and complete document content. Chat must show its unavailable state.

## Validate corpus management

From the product UI:

1. Create an enabled English or Portuguese corpus.
2. Edit its metadata and observe the unchanged identity.
3. Disable it and verify that direct workspace access is unavailable.
4. Re-enable it and verify that the same source ownership is retained.
5. Attempt a stale update in the automated contract scenario and verify no overwrite.

## Validate URL ingestion

Add a controlled public HTTPS document in a non-seeded corpus. Observe `pending`, `processing`, and `ready`; inspect provenance, complete normalized text, and units. Repeat the same normalized URL and expect a duplicate error. Run the controlled unsafe-address, redirect, timeout, content-type, and size cases through integration tests; never target arbitrary private infrastructure.

## Validate PDF ingestion

Upload a valid text PDF under 2 MB. Confirm its binary metadata, ready document, page fallback, and any detected legal units. Repeat the same PDF and expect a duplicate error. Use committed generated test fixtures for encrypted, image-only, malformed, and oversize cases; do not commit copyrighted legal PDFs.

## Validate retry and reprocess

Cause a controlled acquisition failure, retry after correcting it, and confirm the same source identity becomes ready. Reprocess unchanged content and confirm no duplicate active document. Cause a failed reprocess and confirm the prior ready document remains browsable.

## Automated quality gates

Run the module and service-backed quality contracts:

```bash
make -C apps/api ci
make -C apps/ingestion ci
make -C apps/web ci
make persistence-integration
python .github/scripts/validate_repository_language.py
git diff --check
```

These commands include:

- Go format, vet, unit, contract, race where supported, and PostgreSQL integration checks;
- Python Ruff, strict mypy, unit, contract, and integration checks;
- React formatting, lint, strict types, component/accessibility, build budget, and browser journeys;
- contract compatibility, repository-language validation, migration repeatability, corpus isolation, lease recovery, atomic publication, and runtime-fixture absence;
- Sonar quality gates for web, API, and ingestion.

## Inspect failures

Use the files under `.log/` named for each component. Logs may contain identifiers, states, safe categories, counts, and durations. They must not contain credentials, PDF bytes, complete normalized documents, or URL user information.

## US1 measured checkpoint

The first vertical checkpoint was executed on 2026-08-17:

- the ordered migrations were applied three additional times and retained exactly two seed
  corpora and two seed sources without changing the schema version;
- the controlled claim/acquisition/extraction/publication integration journey passed twice in
  succession after its transaction cleanup was verified;
- the official English GDPR source reached `ready` with one immutable revision, one complete
  353,129-character document, and 101 addressable units;
- the official LGPD endpoint returned a bounded `acquisition_failed` outcome during the corrected
  live run, leaving its corpus and source available for explicit retry with no partial document;
- the worker handled SIGINT cooperatively and emitted only safe structured identifiers and event
  names.

This is an accepted US1 outcome because remote source failure is explicit and retryable. It is not
evidence that every future run will reach the same terminal state; source-network availability is
an external prerequisite.

## US2 measured checkpoint

The corpus-management slice was completed on 2026-08-17:

- focused application, HTTP, and live PostgreSQL tests covered create, edit, disable, re-enable,
  validation, stable identity, monotonic versions, and rejection of stale writes;
- React component and accessibility tests covered authoritative creation, editing, disable
  confirmation, lifecycle visibility, and a localized stale-state outcome;
- the complete Go quality gate and all 24 React component tests passed, followed by the strict
  TypeScript production build and bundle-budget check;
- corpus deletion remains intentionally unsupported, so owned sources, documents, and ingestion
  history survive lifecycle changes.

## Final convergence checkpoint

The complete isolated acceptance journey was executed twice on 2026-08-17 after all module gates:

- run one completed in 79.27 seconds and run two completed in 78.68 seconds;
- both runs applied migration version 2 three times without drift and recreated the isolated
  volumes from an empty state;
- both runs passed the Go service-backed suite and all 54 Python tests, including controlled HTTPS,
  generated PDF, corpus isolation, expired-lease recovery, idempotent publication, changed-content
  revision, and failed reprocess preservation;
- controlled source assertions reached `ready` after atomic publication and `failed` after the
  intentional reprocess failure while retaining the preceding ready document;
- every isolated container, network, and volume was removed by the guarded cleanup trap.

The default local bootstrap was then run against retained developer data. PostgreSQL, Neo4j, web,
API, and ingestion reported healthy or running. The API returned both enabled seed corpora. Its
English source was `ready`; the Portuguese source was safely `failed` after the remote acquisition
outcome and remained eligible for retry. Repeating `make bootstrap` reuses verified long-lived
process identities. The final bootstrap implementation also waits up to the configured bound for
both stable source identities to reach `ready` or `failed` and prints their terminal states before
reporting readiness.

## Stop without deleting data

Use the existing bootstrap/local-environment stop command documented in `docs/operations/local-environment.md`. Normal stop preserves PostgreSQL and Neo4j volumes. Destructive reset remains explicit and guarded.
