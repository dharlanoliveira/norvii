# Implementation Plan: Evidence-Backed Normative Assertions

**Branch**: `011-normative-assertions` | **Date**: 2026-08-25 | **Spec**: [spec.md](spec.md)

**Input**: Feature specification from `specs/011-normative-assertions/spec.md`

## Summary

Replace the direct semantic-relationship graph with a graph of legal units, legal entities, and evidence-backed normative assertions. `document_units` remains the canonical legal hierarchy; each assertion retains its exact establishing and evidence units and connects its subject and object entities through a controlled predicate. Rebuild the derived graph and online retrieval around that model. After migrations and controlled-data verification succeed, execute a documented local-data reset and reingest the configured sources so no legacy relation remains.

## Technical Context

**Language/Version**: Go 1.26, Python 3.13, TypeScript 5, React 19, SQL migrations

**Primary Dependencies**: pgx v5, psycopg, Neo4j Python driver, LangGraph, OpenAI-compatible chat completions, assistant-ui, Vite, Vitest, Testing Library

**Storage**: PostgreSQL remains canonical for corpus, legal units, semantic entities, normative assertions, and graph-release manifests. Neo4j remains a rebuildable immutable projection for one snapshot. The reset removes local canonical and derived corpus data only; it does not alter external sources.

**Testing**: Go migration and HTTP tests; Python pytest unit, repository, graph-projection, retrieval, and integration tests; TypeScript contract and component tests; module format, type, lint, build, language, and controlled reset verification.

**Target Platform**: Local Linux development environment with Docker Compose PostgreSQL and Neo4j, a configured OpenAI-compatible provider, and modern desktop browsers.

**Project Type**: Multi-service web application with a React client, Go public API, Python online agent, and Python ingestion service.

**Performance Goals**: Graph retrieval returns no more than eight distinct cited locations and one minimal hierarchy path per location. Graph planning receives no more than 32 predicate descriptors and 128 canonical entity labels. A scope traversal follows at most six hierarchy edges from a requested legal unit.

**Constraints**: Every graph assertion must be snapshot-scoped and evidence-backed. A graph failure cannot prevent a vector-backed Hybrid response. No complete source content, provider prompt, credential, or hidden reasoning enters logs or stream events. The destructive reset may run only after migration and deterministic controlled-data checks pass, and it must not contact or delete external sources.

**Scale/Scope**: Two curated legal corpora are reingested from zero after reset. The predicate vocabulary is limited to the eight normative predicates in the specification. The initial extraction budget remains bounded per document and is observable.

## Constitution Check

*GATE: Passed before research and re-checked after design.*

- Principle I: Feature 011 owns the specification, plan, data model, contract, validation guide, tasks, and acceptance evidence before implementation.
- Principle II: Ingestion owns canonical extraction, assertion persistence, and Neo4j projection; the agent owns graph retrieval and inspection; the Go API validates and forwards the public stream; React renders the established contract. No module imports another module's internals.
- Principle III: Assertions require resolvable units and remain subordinate to corpus evidence. Retrieval abstains or falls back when a scope has no supported assertion.
- Principle IV: The inspected graph-path fields are an explicit versioned cross-language contract with provider and consumer verification.
- Principle V: The model is represented by narrow typed values and repository boundaries; migrations, invalid provenance, reset ordering, and query scoping receive deterministic tests.
- Principle VI: The reset is local, versioned, and preflight-gated. Extraction and traversal have explicit size limits; no provider call occurs during controlled-data validation.
- Principle VII: Logs and inspection identify the assertion, source units, predicate, and scope without exposing full content or prompts. The reset has a visible empty-state outcome.
- Principle VIII: All engineering artifacts for this feature are written in English.

## Research Decisions

Detailed alternatives and rationale are recorded in [research.md](research.md). The design uses existing `document_units` as the legal-unit authority, replaces direct semantic relationships with a dedicated canonical assertion record, projects unit hierarchy and assertions as distinct Neo4j nodes and edges, and requires a preflight-gated reset before fresh source ingestion.

## Project Structure

### Documentation (this feature)

```text
specs/011-normative-assertions/
|-- plan.md
|-- research.md
|-- data-model.md
|-- quickstart.md
|-- contracts/
|   `-- normative-assertion-stream.md
`-- tasks.md
```

### Source Code (repository root)

```text
apps/
|-- api/
|   |-- migrations/                         # Canonical assertion schema and reset preflight
|   `-- internal/                           # Snapshot/catalog status and stream validation
|-- ingestion/
|   |-- src/norvii_ingestion/semantic/      # Typed assertion extraction and validation
|   |-- src/norvii_ingestion/publication/   # Canonical PostgreSQL persistence and Neo4j adapter
|   |-- src/norvii_ingestion/graph_projection/ # Snapshot assertion projection
|   `-- tests/                              # Unit, repository, projection, and reset fixtures
|-- agent/
|   |-- src/norvii_agent/retrieval/         # Assertion capability, scoped query, and paths
|   `-- tests/                              # Retrieval and stream contract behavior
`-- web/
    |-- src/api/                            # Validated inspection DTOs
    `-- src/features/workspace/             # Assertion evidence and hierarchy inspection

infra/
|-- scripts/                                # Preflight-gated local corpus reset command
`-- compose.yaml                            # Existing local services only

docs/
|-- modules/                                # Durable ingestion and agent ownership
`-- operations/                             # Reset and reingestion runbook
```

**Structure Decision**: Keep `document_units` as the single legal-unit representation rather than duplicating hierarchy into semantic entities. Ingestion owns the canonical assertion write model and all derived Neo4j writes. The agent consumes only the ready snapshot projection. The browser receives an inspection projection rather than a graph query surface.

## Module Impact

| Module | Change | Responsibility | Verification |
| --- | --- | --- | --- |
| `apps/api/migrations/` | Change | Replace legacy relationship persistence with assertion persistence and preserve referential/snapshot invariants. | Migration integration tests and schema test |
| `apps/ingestion/` | Change | Extract, validate, persist, reset-check, and project legal units, entities, and assertions. | pytest unit, repository, projection, and integration tests |
| `apps/agent/` | Change | Plan and retrieve bounded assertion paths with source-unit context. | pytest unit, transport, and snapshot-isolation tests |
| `apps/api/` | Change | Validate and forward the updated inspection contract; expose safe empty-corpus state. | Go unit and HTTP tests |
| `apps/web/` | Change | Validate and render assertion provenance and hierarchy inspection. | Vitest, typecheck, build |
| `infra/` | Change | Execute a preflight-gated local data reset and document the reingestion sequence. | Script tests and local dry run |
| `docs/` | Change | Describe the durable assertion model and safe reset operation. | Documentation review and quickstart |
| `specs/011.../contracts/` | Change | Own the graph inspection boundary. | Producer and consumer contract tests |

## Boundaries and Constraints

- **Legal units**: `document_units` is canonical. Parent-child links are directed from parent to child. Structural units do not become semantic entities.
- **Assertions**: An assertion has two distinct endpoint entities, one establishing unit, one evidence unit, a directed allowed predicate, optional qualifier, extraction provenance, and supported validation state. Both unit references must belong to the assertion document and snapshot.
- **Graph release**: A release records its selected entities, legal units, and assertions. Neo4j is rebuilt from these records and cannot become a source of canonical legal text.
- **Graph topology**: `LegalUnit-[:CONTAINS]->LegalUnit`, `LegalUnit-[:ESTABLISHES]->NormativeAssertion`, and `NormativeAssertion-[:SUBJECT|OBJECT]->LegalEntity`. Predicate is a validated assertion property, not a dynamic Neo4j relationship type.
- **Online retrieval**: Planning sees only available assertion predicates and compatible entity labels. A hierarchy scope matches a legal unit and its descendants, then returns the exact assertion and smallest ancestor path. All query values are parameterized and bounded.
- **Reset**: Extend the existing volume-owned `persistence-reset` command with a preflight that runs assertion-schema migration and deterministic controlled-data checks before it stops services and removes the two verified local persistence volumes. A failure before volume deletion changes no data; an interruption after deletion leaves an explicit empty state. The next bootstrap recreates schema, configured seed sources, and fresh corpus artifacts. External source systems are never deleted.
- **Reingestion**: The reset workflow re-registers and ingests the configured sources. The first graph-ready snapshot must use the assertion model; no legacy relationships may be generated.
- **Observability**: Inspection includes assertion ID, predicate, subject/object labels, establishing/evidence locators, and hierarchy scope. Structured logs record safe IDs and counts, not full content or prompts.

## Complexity Tracking

| Violation | Why Needed | Simpler Alternative Rejected Because |
| --- | --- | --- |
| Canonical assertion node plus separate graph node | Legal meaning must retain two endpoint roles, direct authoring unit, precise evidence, qualifiers, and hierarchy without duplicating an assertion onto its ancestors. | A direct typed edge cannot represent all provenance and scope semantics without overloaded properties and ambiguous ownership. |
