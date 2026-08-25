# Implementation Plan: Bilingual Corpus Snapshots

**Branch**: `007-corpus-snapshots` | **Date**: 2026-08-24 | **Spec**: [spec.md](spec.md)

**Input**: Feature specification from `specs/007-corpus-snapshots/spec.md`

## Summary

Create reproducible, immutable evidence snapshots for the two curated legal corpora.
Ingestion continues to produce immutable source revisions, document versions, units, and
retrieval chunks. A snapshot records the exact set of those documents that may be used for
research. The active-snapshot release pointer is the only mutable selection used by chat.
Maintainers explicitly publish a validated candidate into a new snapshot; ordinary
reingestion never changes the active release.

## Technical Context

<!--
  ACTION REQUIRED: Replace the content in this section with the technical details
  for the project. The structure here is presented in advisory capacity to guide
  the iteration process.
-->

**Language/Version**: Go 1.26, Python 3.13, TypeScript 5 with React 19

**Primary Dependencies**: pgx v5, PostgreSQL 17 with pgvector, LangGraph, psycopg,
Vite, assistant-ui, Vitest, Testing Library

**Storage**: PostgreSQL is the canonical snapshot, source, document, and retrieval store.
Neo4j remains available for later GraphRAG work and is not part of snapshot publication.

**Testing**: Go unit, handler, repository, and integration tests; pytest unit and
integration tests; Vitest and Testing Library client tests; module CI targets and repository
language validation

**Target Platform**: Linux containers and local Docker Compose development environment;
modern desktop browsers

**Project Type**: Multi-service web application with a React SPA, Go public API, and Python
agent and ingestion services

**Performance Goals**: Snapshot publication and catalog reads are bounded by the small POC
corpora; chat retrieval remains limited to eight vector candidates within one active snapshot.

**Constraints**: Exactly two initial curated corpora; one active snapshot per corpus;
publication must be atomic; reingestion must not activate a candidate; snapshot creation must
not call an LLM or create new embeddings; all generated legal answers remain corpus- and
snapshot-grounded.

**Scale/Scope**: Two initial corpora and one official source per corpus, while the membership
model supports multiple sources in future snapshots.

## Constitution Check

*GATE: Must pass before Phase 0 research. Re-check after Phase 1 design.*

**Pre-design result: PASS**

- Principle I: this numbered spec, plan, data model, API contract, quickstart, and task list
  make the vertical feature traceable before application code changes.
- Principle II: PostgreSQL snapshot release ownership remains in the Go API; the Python agent
  only consumes the explicit snapshot boundary; ingestion remains responsible for immutable
  artifacts; the React client consumes public API projections.
- Principle III: the active snapshot is added to the enforced retrieval boundary and evidence
  references carry its stable identity.
- Principle IV: PostgreSQL remains canonical; no new graph projection, model call, or provider
  dependency is introduced.
- Principles V--VIII: existing safety, observable lifecycle, quality, and English-language
  checks apply to the new endpoint, migration, commands, tests, and UI.

**Post-design result: PASS.** The design keeps membership immutable, separates the active
release pointer from the snapshot manifest, and does not introduce an unjustified shared
implementation or cross-module import.

## Project Structure

### Documentation (this feature)

```text
specs/007-corpus-snapshots/
|- plan.md               # This file
|- research.md           # Phase 0 output
|- data-model.md         # Phase 1 output
|- quickstart.md         # Phase 1 output
|- contracts/            # Phase 1 output
`- tasks.md              # Phase 2 output
```

### Source Code (repository root)
<!--
  ACTION REQUIRED: Replace the placeholder tree below with the concrete layout
  for this feature. Delete unused options and expand the chosen structure with
  real paths (e.g., apps/admin, packages/something). The delivered plan must
  not include Option labels.
-->

```text
apps/
|- api/
|  |- cmd/                            # API and deterministic snapshot-initialization roots
|  |- internal/catalog/               # Corpus projection with active release summary
|  |- internal/snapshot/              # Snapshot publication, manifest, and persistence
|  |- internal/chat/                  # Resolves active snapshot before agent forwarding
|  `- migrations/                     # Canonical snapshot schema
|- agent/
|  `- src/norvii_agent/               # Snapshot-scoped retrieval and internal stream contract
|- ingestion/
|  `- src/norvii_ingestion/           # Produces candidate document and chunk artifacts only
`- web/
   `- src/
      |- api/                         # Snapshot API and stream contract adapters
      `- features/                    # Catalog, source, and workspace snapshot presentation

contracts/
`- agent/                             # Versioned Go-to-agent request and stream schemas

docs/
`- operations/                        # Local initialization and reproducibility instructions
```

**Structure Decision**: Keep snapshot behavior as a cohesive Go capability under
`apps/api/internal/snapshot/`. The API owns public publication and active-release decisions;
the agent receives a stable `snapshotId` in its internal request and uses it only to constrain
retrieval. The web application exposes compact release information and maintainer actions via
feature-owned adapters and components. The ingestion module does not gain publication policy:
its already immutable output is the candidate input to the Go-owned publication use case.

## Complexity Tracking

> **Fill ONLY if Constitution Check has violations that must be justified**

| Violation | Why Needed | Simpler Alternative Rejected Because |
|-----------|------------|-------------------------------------|
| [e.g., 4th project] | [current need] | [why 3 projects insufficient] |
| [e.g., Repository pattern] | [specific problem] | [why direct DB access insufficient] |



## Norvii Implementation Requirements *(mandatory)*

### Module Impact

| Module | Change | Responsibility | Verification |
| --- | --- | --- | --- |
| `prototypes/web/` | No change | Feature 001 remains the approved experience baseline. | N/A |
| `apps/web/` | Change | Display active snapshot identity, candidate status, publication feedback, and history without disrupting research. | Vitest, locale parity, build |
| `apps/api/` | Change | Own snapshot manifest, release activation, validation, public endpoints, and agent forwarding. | Go unit, handler, repository, integration tests |
| `apps/agent/` | Change | Receive a snapshot identity and query only its immutable document membership. | pytest unit and contract tests |
| `apps/ingestion/` | Change | Assert reingestion produces candidate artifacts without activating a snapshot. | pytest integration regression tests |
| `contracts/` | Change | Version internal agent request and evidence records with snapshot identity. | Contract tests |
| `infra/` | No change | Existing PostgreSQL and pgvector Compose service is sufficient. | Existing persistence verification |

### Boundaries and Constraints

- **Cost limits**: Snapshot validation is metadata-only and reuses existing documents and
  embeddings. It performs no LLM, embedding, or graph call. The two corpus sources remain
  small, official URLs.
- **Prototype baseline**: Feature 001's verified corpus catalog and workspace remain the
  baseline. Snapshot information is compact operational metadata, not a new page redesign.
- **Public contracts**: The public REST/SSE contract adds snapshot identity where an answer,
  corpus, or publication result is represented. The Go-to-agent request becomes a versioned
  internal contract with mandatory `snapshotId`.
- **Persistence**: A Go-owned migration adds immutable snapshot manifests, immutable membership,
  and a mutable, versioned release pointer per corpus. Migration down paths remove only the new
  tables after dependent indexes and constraints.
- **Ingestion artifacts**: Existing source revision, document version, legal unit, and retrieval
  chunk identities remain immutable. The newest ready document is a candidate, not online
  evidence, until explicitly published.
- **Streaming**: The chat request forwarded to the agent includes `snapshotId`; every returned
  evidence item and inspection projection includes it. Existing stream event ordering remains
  unchanged.
- **Corpus boundary and citations**: Retrieval joins snapshot membership by corpus, source, and
  document identity. The API rejects absent active releases and mismatched publication inputs.
  Citation navigation continues to use immutable document and location identities.
- **Security and privacy**: Publication accepts UUIDs and an optimistic release version only;
  it does not re-fetch URLs, accept document contents, log secrets, or expose provider payloads.
- **Observability**: Publication reports safe `validation_failed`, `stale_release`, and
  `candidate_not_ready` categories. Inspection shows snapshot identity but never raw prompts,
  credentials, or full provider diagnostics.
- **Local environment**: Existing Compose topology is sufficient. A deterministic API command
  creates initial snapshots after initial ingestion; repeated execution is idempotent. The
  quickstart documents migration, ingestion, initialization, and verification.

### Repository Paths

Use only the modules marked as changed above:

```text
apps/web/                    # React and TypeScript production client
apps/api/                    # Go online backend
apps/ingestion/              # Python offline pipeline
prototypes/web/              # Executable React product prototype
contracts/                   # Stable cross-language schemas
infra/                       # Local backing services
docs/                        # Durable global documentation
```
