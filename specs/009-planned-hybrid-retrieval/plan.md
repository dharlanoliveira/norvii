# Implementation Plan: Planned Hybrid Retrieval

**Branch**: `009-planned-hybrid-retrieval` | **Date**: 2026-08-25 | **Spec**: [spec.md](spec.md)

**Input**: Feature specification from `specs/009-planned-hybrid-retrieval/spec.md`

## Summary

Replace the three-way researcher strategy choice with Vector and Hybrid. Both approaches always
retrieve snapshot-scoped vector evidence. Hybrid then uses a bounded model-planning step to decide
whether the active graph release can add relevant, evidence-backed context. The planner sees a
small graph capability catalog, returns a validated structured decision, and never receives direct
graph-query authority. The agent executes an allowlisted, parameterized graph query only when
appropriate, merges evidence without losing contribution attribution, and exposes all stages in
the research record.

## Technical Context

**Language/Version**: Go 1.26, Python 3.13, TypeScript 5, React 19

**Primary Dependencies**: pgx v5, psycopg, Neo4j Python driver, LangGraph,
OpenAI-compatible chat completions, assistant-ui, Vite, Vitest, Testing Library

**Storage**: PostgreSQL is canonical for legal content, retrieval chunks, source revisions, and
snapshot identity. Neo4j is an immutable, rebuildable graph projection associated with one ready
snapshot release. No schema migration is expected.

**Testing**: Go application and HTTP boundary tests; Python pytest unit and transport tests;
TypeScript contract, domain, and component tests; module format, type, lint, build, language, and
integration checks.

**Target Platform**: Local Linux development environment with PostgreSQL, Neo4j, a configured
OpenAI-compatible provider, and modern desktop browsers.

**Project Type**: Multi-service web application with a React client, Go public API, Python online
agent, and Python ingestion service.

**Performance Goals**: Retrieval returns no more than eight distinct cited locations. Hybrid
planning receives no more than 32 type descriptors and 128 canonical entity labels, and adds one
bounded provider request to a Hybrid answer. Graph traversal remains at most three hops and eight
path-supported locations.

**Constraints**: Vector retrieval is mandatory for each new question. A graph-release absence or
planner/query failure must not prevent a vector-backed Hybrid response. The system must not reveal
prompts, graph credentials, complete source content, or hidden reasoning. Only active-snapshot
evidence can reach answer generation.

**Scale/Scope**: Two seeded, single-source legal corpora. Graph capability data is deliberately
small and supports existing legal concepts, actors, rights, obligations, and relationships.

## Constitution Check

*GATE: Must pass before Phase 0 research. Re-check after Phase 1 design.*

**Pre-design and post-design result: PASS**

- Principle I: this numbered feature owns its requirements, plan, cross-language contract,
  verification guide, and tasks before implementation.
- Principle II: the Python agent owns adaptive online retrieval; Go validates the public approach
  and resolves the immutable snapshot; React renders public inspection projections; ingestion and
  Neo4j graph publication remain unchanged.
- Principle III: planning does not create evidence. Every final material claim remains tied to an
  active-snapshot citation.
- Principle IV: the changed request and inspection event shapes are versioned in this feature and
  verified at Go, Python, and TypeScript boundaries.
- Principles V--VIII: planning is isolated behind a small typed port, provider calls are bounded,
  failures are safe and inspectable, and changed behavior is tested in all affected modules.

## Research Decisions

Detailed alternatives and rationale are recorded in [research.md](research.md). The decisive
choices are vector-first retrieval, a conservative planner supplied with a snapshot-scoped
capability catalog, graph augmentation rather than a required graph contribution, and explicit
per-stage inspection rather than an ambiguous overall "completed" result.

## Project Structure

### Documentation (this feature)

```text
specs/009-planned-hybrid-retrieval/
|- plan.md
|- research.md
|- data-model.md
|- quickstart.md
|- contracts/
|  `- planned-hybrid-stream.md
`- tasks.md
```

### Source Code (repository root)

```text
apps/
|- agent/
|  |- src/norvii_agent/
|  |  |- graph/                 # Grounded response and inspection models
|  |  |- providers/             # OpenAI-compatible answer and planning adapters
|  |  |- retrieval/             # Vector, capability, graph, and hybrid composition
|  |  `- transport/             # Internal versioned stream boundary
|  `- tests/                    # Unit and transport behavior tests
|- api/
|  `- internal/chat/            # Public validation and agent SSE forwarding
`- web/
   |- src/api/                  # Validated client stream DTOs
   |- src/research/domain/      # Vector/Hybrid comparison models
   `- src/features/workspace/   # Selector and research record UI

specs/009-planned-hybrid-retrieval/contracts/
docs/modules/
```

**Structure Decision**: Keep adaptive retrieval entirely in `apps/agent`; Go remains a thin
boundary and does not know planner internals. The browser consumes a stable inspection projection
and does not infer stage state from answer text. The graph capability catalog is read through the
existing graph adapter, never exposed as a browser or public graph query API.

## Complexity Tracking

| Violation | Why Needed | Simpler Alternative Rejected Because |
| --- | --- | --- |
| One additional bounded model request for Hybrid | The user requires question-aware graph augmentation. | Token matching cannot determine whether the graph schema can help a broad legal question. |

## Norvii Implementation Requirements

### Module Impact

| Module | Change | Responsibility | Verification |
| --- | --- | --- | --- |
| `prototypes/web/` | No change | The approved prototype remains a visual baseline. | N/A |
| `apps/web/` | Change | Select Vector/Hybrid, compare those approaches, and render stage inspection. | Vitest, typecheck, build |
| `apps/api/` | Change | Validate the two public approaches, resolve snapshots, forward inspection events. | Go unit and HTTP tests |
| `apps/agent/` | Change | Perform vector-first retrieval, bounded planning, optional graph augmentation, and safe stage inspection. | pytest unit and transport tests |
| `apps/ingestion/` | No change | Continues to build active graph releases and evidence-backed projections. | Existing ingestion checks |
| `specs/009.../contracts/` | Change | Own the versioned request and inspection event contract. | Producer/consumer contract tests |
| `infra/` | No change | Existing provider, PostgreSQL, and Neo4j configuration remains sufficient. | Existing bootstrap verification |

### Boundaries and Constraints

- **Cost limits**: Planning runs only for Hybrid, with one provider call and at most 32 compact
  capability descriptors. It uses the configured chat model and time limit; it never runs during
  ingestion or vector-only questions.
- **Prototype baseline**: Features 002, 005, 006, 007, and 008 remain the approved workspace and
  evidence baseline. The strategy selector is an intentional production refinement.
- **Public contracts**: The browser may send only `vector` or `hybrid`. Go forwards one
  active-snapshot request. Agent terminal inspection includes vector, planning, graph, and
  generation stage data as validated structured values.
- **Persistence**: No migration. Existing snapshot and graph-release identities remain canonical.
- **Ingestion artifacts**: No change. Ingestion continues to publish a ready graph release with a
  successful snapshot so Hybrid can discover capabilities automatically.
- **Streaming**: Existing message event ordering remains unchanged. The terminal inspection gains
  typed stage contributions; safe graph outcomes do not end the request if vector evidence exists.
- **Corpus boundary and citations**: Vector and graph calls both receive the selected active
  corpus and snapshot. Merged evidence is deduplicated by immutable location and is the only
  evidence supplied to answer generation.
- **Security and privacy**: Planner outputs are strict, bounded data; graph queries are
  parameterized; prompts, credentials, hidden reasoning, and full legal content are never logged.
- **Observability**: Stage status, bounded reason, evidence count, duration, and provider token
  counts are inspectable. Logs remain content-free.
- **Local environment**: No new service. `make bootstrap` uses existing agent credentials and
  graph-release readiness.

### Repository Paths

```text
apps/web/                    # React and TypeScript production client
apps/api/                    # Go online backend
apps/agent/                  # Python LangGraph retrieval and generation service
apps/ingestion/              # Python offline ingestion worker; no change
specs/009-planned-hybrid-retrieval/ # Feature plan and public contract
docs/modules/                # Durable agent responsibility documentation
```
