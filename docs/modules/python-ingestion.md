# Python Ingestion Model

## Mission

`apps/ingestion/` transforms a registered PDF or URL source into reproducible,
validated artifacts that online retrieval can consume. It runs outside the chat
request path.

## Owns

- claiming or receiving pending ingestion work;
- safe acquisition of PDF binaries and external URLs;
- source hashing, capture metadata, and manifest updates;
- text extraction and legal-structure normalization;
- legal-aware chunking and stable source locations;
- embeddings and evidence-backed semantic extraction when their features are enabled;
- idempotent publication of the rebuildable Neo4j graph projection;
- artifact validation, publication checkpoints, retry classification, and processing outcomes;
- offline corpus evaluation inputs and measurements.

## Does not own

- corpus or source user interfaces;
- public application APIs;
- chat orchestration and answer generation;
- retrieval decisions made for an online question;
- Go domain types or database adapter internals.

## Object model

Stateful stages and domain behavior use cohesive classes with explicit constructor
dependencies. Examples include a source acquirer, document extractor, legal
normalizer, chunking policy, artifact publisher, and ingestion orchestrator. Small,
stateless, deterministic transformations may remain functions when a class would add
indirection without protecting state or an invariant.

Stages exchange typed domain objects rather than unstructured dictionaries. Vendor
SDK objects remain inside adapters. The orchestrator coordinates stages but does not
contain extraction, chunking, or persistence algorithms itself.

## Target organization

```text
apps/ingestion/
|-- src/norvii_ingestion/
|   |-- domain/              # Sources, documents, artifacts, invariants
|   |-- application/         # Use cases and stage orchestration
|   |-- acquisition/         # PDF and URL adapters
|   |-- extraction/          # Format-specific adapters
|   |-- enrichment/          # Chunking, embeddings, graph extraction
|   `-- publication/         # Contract and persistence adapters
`-- tests/
    |-- unit/
    |-- integration/
    `-- contract/
```

## Failure and idempotency model

Every run has a source identity, source hash, pipeline version, model version, prompt
version, and attempt identity. Repeating the same source and pipeline version MUST
not publish duplicate active artifacts. A failure identifies its stage and whether
retry is safe. Partial canonical output MUST not become visible to online retrieval.
Graph publication happens after canonical PostgreSQL publication, records a
checkpoint, and MUST be safe to retry or rebuild.

## Verification model

- unit tests for each stage and legal-structure invariant;
- golden or fixture tests for representative official documents;
- contract tests for work input and artifact output;
- integration tests for network, database, embedding, and graph adapters;
- deterministic replacements for network and model calls in unit tests;
- formatting, type checking, linting, and tests for affected packages.

## Implemented persistence foundation

Feature 003 initializes the Python 3.13 uv package and adds class-based persistence
adapters under `publication/persistence/`. Immutable configuration objects validate
the shared environment contract. `PostgresStore`, `Neo4jStore`, and
`PersistenceVerifier` keep driver lifecycles, bounded read-only checks, and safe
service attribution outside future ingestion domain behavior.

The package publishes no ingestion artifact and creates no product or graph data.
Run module quality and local connectivity independently:

```bash
make -C apps/ingestion ci
python infra/scripts/run-with-environment.py infra/.env make -C apps/ingestion verify-persistence
```

Tests marked `integration` require the local stores and are executed by the dedicated
persistence journey rather than the standard module test target.
