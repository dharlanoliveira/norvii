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

Ingestion publishes candidate revisions, units, chunks, and embeddings only. It never
changes `corpus_snapshot_releases`; a maintainer publishes a validated candidate into
an immutable active snapshot through the Go API.

## Semantic extraction and graph releases

Semantic extraction runs during explicit ingestion and records bounded, evidence-backed legal
entities and normative assertions against one immutable document version. A published assertion
has exactly one allowed predicate, one subject entity, one object entity, an establishing legal
unit, and an evidence legal unit. The supported predicates are `defines`, `applies_to`,
`must_be_observed_by`, `imposes_duty_on`, `grants`, `protects`,
`assigns_responsibility_to`, and `conditions`.

Each legal entity is atomic: independently addressable people, authorities, activities, rights,
obligations, concepts, or conditions in a coordinated list are emitted as separate entities and
separate assertions. A collective is kept as one entity only when the source treats it as one
indivisible legal subject. Incomplete assertions and assertions with unresolved provenance are
not published. Provider requests are bounded by the configured unit and request limits; the
extraction provider is never invoked by chat or graph retrieval.

The document hierarchy is different: every normalized location and directed parent-child
`CONTAINS` link is projected deterministically, even when semantic extraction is bounded to a
smaller set of legal units. Hierarchy links are structural rather than semantic assertions. This
lets graph retrieval navigate from a relevant provision to its chapter without treating every
chapter as answer evidence.

If a semantic provider response cannot be parsed, ingestion writes a structured diagnostic to
`.log/ingestion.log`. It records the provider request identifier when available, content type,
byte count, SHA-256 fingerprints, parsing location, response attempt, and safe failure subtype.
It never stores the provider response body, legal text, prompts, credentials, or personal data.

The canonical write model is PostgreSQL `normative_assertions`, plus graph-release memberships
for legal units and assertions. Neo4j is an immutable derived projection with this topology:

```text
(LegalUnit)-[:CONTAINS]->(LegalUnit)
(LegalUnit)-[:ESTABLISHES]->(NormativeAssertion {predicate, qualifier})
(NormativeAssertion)-[:SUBJECT]->(LegalEntity)
(NormativeAssertion)-[:OBJECT]->(LegalEntity)
```

The predicate is an allowlisted assertion property; it is not a dynamic Neo4j relationship type.
Every projected node is release-, corpus-, and snapshot-scoped. The projection is rebuildable
from canonical PostgreSQL records and never becomes a source of canonical legal text.

After a maintainer publishes a snapshot, build its Neo4j projection explicitly:

```bash
python infra/scripts/run-with-environment.py infra/.env \
  uv run --directory apps/ingestion norvii-build-graph-release \
  --corpus-id <corpus-id> --snapshot-id <snapshot-id>
```

The builder reads only canonical PostgreSQL semantic records that belong to the named published
snapshot. It writes an idempotent derived Neo4j release and records its manifest, status, counts,
and safe failure category in PostgreSQL. Repeating the command for unchanged inputs reuses the
same release identity. The builder never changes source content, snapshot activation, or
canonical extraction artifacts.

## Assertion-model reset and fresh ingestion

Feature 011 replaces the legacy direct semantic-relationship model. Do not delete local data by
hand and do not run a fresh ingestion against a partially migrated database. The only supported
destructive operation is `make persistence-reset CONFIRM=reset-norvii-data`; it first starts the
stores, applies pending migrations, and runs the ingestion and agent test targets. The reset
script proceeds only when that preflight succeeds and verifies the two Compose-owned persistence
volumes before removing them.

After a successful reset, run `make bootstrap` to recreate the local services, apply migrations,
register the configured seed sources, and perform the normal fresh-ingestion workflow. Until a
new snapshot and graph release are ready, an empty corpus has no graph evidence. The reset never
contacts or deletes external sources.

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
