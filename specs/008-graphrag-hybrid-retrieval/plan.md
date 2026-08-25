# Implementation Plan: GraphRAG and Hybrid Retrieval

**Branch**: `008-graphrag-hybrid-retrieval` | **Date**: 2026-08-24 | **Spec**: [spec.md](spec.md)

**Input**: Feature specification from `specs/008-graphrag-hybrid-retrieval/spec.md`

## Summary

Add a small, evidence-backed legal graph to the existing snapshot-scoped RAG system. Ingestion
records bounded semantic extraction artifacts against immutable document versions, stages a
candidate snapshot, builds its graph projection, and activates the new snapshot only after the
derived projection is ready. The graph is never authoritative.
The agent offers vector, graph, and hybrid retrieval as distinct strategies; every path returns
the same snapshot-bound evidence references used by the answer and inspection UI.

## Technical Context

**Language/Version**: Go 1.26, Python 3.13, TypeScript 5 with React 19

**Primary Dependencies**: pgx v5, psycopg, Neo4j Python driver, LangGraph, Vite,
assistant-ui, Vitest, Testing Library

**Storage**: PostgreSQL remains canonical for document versions, semantic extraction artifacts,
graph-release manifests, and evidence locations. Neo4j Community is a rebuildable graph
projection for one ready snapshot release.

**Testing**: Go domain, repository, handler, and PostgreSQL integration tests; Python pytest
unit and Neo4j/PostgreSQL integration tests; TypeScript contract and component tests; module CI,
language validation, and reproducible graph rebuild checks.

**Target Platform**: Local Linux development environment with PostgreSQL, Neo4j, and modern
desktop browsers

**Project Type**: Multi-service web application: React client, Go public API, Python agent, and
Python ingestion service

**Performance Goals**: The POC builds one graph release for each small curated snapshot within
the documented local budget. Each online strategy retrieves no more than eight cited evidence
locations; graph and hybrid requests remain bounded to a maximum of three graph hops.

**Constraints**: Exactly two curated corpora; one active corpus snapshot per corpus; graph
releases are immutable and snapshot-scoped; no fallback may be represented as another strategy;
model enrichment is bounded and never runs in the chat request; candidate snapshots are never
active before graph validation; Neo4j is derived and a recorded release may be rebuilt without
changing canonical legal content or the active snapshot.

**Scale/Scope**: Two initial single-source legal corpora. The initial graph contains document
structure, legal concepts, actors, rights, obligations, and only evidence-backed relationships
needed by a compact seeded question set.

## Constitution Check

*GATE: Must pass before Phase 0 research. Re-check after Phase 1 design.*

**Pre-design result: PASS**

- Principle I: this numbered feature owns its specification, design, contracts, validation guide,
  and task list before application changes.
- Principle II: ingestion owns semantic extraction and derived graph publication; the Go API owns
  the public strategy contract and snapshot resolution; the Python agent owns online retrieval;
  the React client only consumes public projections.
- Principle III: all graph relationships and paths resolve to immutable source locations in one
  corpus snapshot, and insufficient evidence remains an abstention.
- Principle IV: the public and cross-language strategy/evidence contract is versioned under this
  feature; PostgreSQL is authoritative and Neo4j is reconstructible.
- Principles V--VIII: bounded provider use, explicit failures, automated contract/integration
  tests, safe logs, and English engineering artifacts are required.

**Post-design result: PASS.** The design keeps snapshot activation separate from graph readiness,
stores model-derived artifacts separately from official legal content, and confines each lifecycle
to one owning module.

## Project Structure

### Documentation (this feature)

```text
specs/008-graphrag-hybrid-retrieval/
|- plan.md
|- research.md
|- data-model.md
|- quickstart.md
|- contracts/
|  `- graphrag-stream.md
`- tasks.md
```

### Source Code (repository root)

```text
apps/
|- api/
|  |- internal/graphrelease/         # Public graph-release status and strategy projection
|  |- internal/chat/                 # Strategy request validation and agent forwarding
|  `- migrations/                    # Canonical semantic and graph-release schema
|- agent/
|  `- src/norvii_agent/
|     |- graph/                      # Retrieval strategy state and inspection values
|     `- retrieval/                  # Vector, graph, and hybrid strategy adapters
|- ingestion/
|  `- src/norvii_ingestion/
|     |- semantic/                   # Bounded extraction values and provider adapter
|     `- graph_projection/           # Explicit, idempotent Neo4j projection builder
`- web/
   `- src/features/workspace/        # Strategy selection and graph-path inspection

docs/
|- modules/                           # Module responsibilities and local commands
`- operations/                        # Graph release lifecycle and diagnosis
```

**Structure Decision**: Keep semantic artifacts and graph-release manifests in Python-owned
ingestion packages, but persist them in PostgreSQL as canonical evidence records. Keep online
strategy composition in the Python agent. Go remains the public boundary that resolves an active
snapshot and validates a requested strategy. No application module imports another module's
implementation.

## Complexity Tracking

| Violation | Why Needed | Simpler Alternative Rejected Because |
| --- | --- | --- |
| Derived graph store | The POC must demonstrate explicit graph traversal and hybrid retrieval. | PostgreSQL-only joins would not demonstrate a separately rebuildable GraphRAG projection. |

## Norvii Implementation Requirements

### Module Impact

| Module | Change | Responsibility | Verification |
| --- | --- | --- | --- |
| `prototypes/web/` | No change | Existing approved workspace remains the UI baseline. | N/A |
| `apps/web/` | Change | Select strategies and inspect graph paths without obscuring answers or citations. | Vitest, build, accessibility journeys |
| `apps/api/` | Change | Validate public strategy requests, resolve active snapshots, and expose release status. | Go tests, HTTP contract tests |
| `apps/agent/` | Change | Retrieve vector, graph, or hybrid evidence inside one snapshot boundary. | pytest unit and integration tests |
| `apps/ingestion/` | Change | Produce canonical semantic artifacts and coordinate staged snapshot, graph release, and activation. | pytest unit and integration tests |
| `contracts/` | Change | Version strategy, graph path, readiness, and safe failure fields. | Producer and consumer contract tests |
| `infra/` | No change | Existing PostgreSQL and Neo4j services already satisfy the topology. | Existing persistence checks |

### Boundaries and Constraints

- **Cost limits**: Semantic extraction is offline and limited to a configured maximum
  of 12 legal units per provider request, 32 requests per document, and one extraction attempt per
  document-version/prompt-version pair. Provider calls occur during ingestion extraction; the
  release coordinator only reads persisted canonical artifacts. Chat requests never
  trigger extraction.
- **Prototype baseline**: Features 002, 004, 005, 006, and 007 remain the approved workspace and
  evidence baseline. Strategy and graph-path controls are compact inspection affordances.
- **Public contracts**: The browser requests an optional retrieval strategy. Go resolves the
  active snapshot and forwards a required strategy plus snapshot identity to the agent. SSE
  evidence and inspection records carry strategy, graph-release identity when applicable, and
  paths as structured data.
- **Persistence**: A Go-owned migration creates immutable extraction runs, extracted entities,
  extracted relationships, and immutable graph-release records. The mutable readiness pointer is
  separate from snapshot membership. PostgreSQL remains canonical on all failures.
- **Ingestion artifacts**: Extraction runs are bound to immutable document versions and evidence
  spans. The coordinator stages a named immutable snapshot before graph projection, and activates
  it only after the matching graph release is ready. A rebuild is idempotent for the same
  extraction manifest.
- **Streaming**: Existing message event ordering remains unchanged. The terminal inspection gains
  strategy, graph-release, vector contribution, graph contribution, and path fields. Safe
  `graph_unavailable` and `graph_insufficient_evidence` outcomes do not emit grounded evidence.
- **Corpus boundary and citations**: Every graph query requires corpus ID, snapshot ID, and ready
  graph-release ID. The graph stores canonical IDs and returns no text not already preserved in
  PostgreSQL. Every path relationship has an immutable evidence location.
- **Security and privacy**: Extraction prompts contain only bounded selected legal units. Provider
  payloads, prompts, credentials, and full documents are never persisted or logged. Graph queries
  are parameterized and bounded by three hops and eight cited locations.
- **Observability**: Extraction and projection record model, prompt version, status, counts,
  duration, and provider-reported token usage. API logs include safe strategy and graph-release
  identifiers for request correlation.
- **Local environment**: Reingestion runs the idempotent release coordinator after canonical
  artifact publication. A separate graph-build command remains for reproducibility only and never
  makes provider calls. Quickstart documents readiness, rebuild, and failure diagnosis.

### Repository Paths

```text
apps/web/                    # React and TypeScript production client
apps/api/                    # Go online backend
apps/agent/                  # Python online LangGraph agent
apps/ingestion/              # Python offline extraction and graph projection
specs/008-graphrag-hybrid-retrieval/ # Feature-owned contracts
docs/                        # Durable module and operational documentation
```
