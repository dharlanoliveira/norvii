# Implementation Plan: Grounded RAG Chat

**Branch**: `005-grounded-rag-chat` | **Date**: 2026-08-20 | **Spec**: [spec.md](spec.md)

**Input**: Feature specification from `specs/005-grounded-rag-chat/spec.md`

## Summary

Deliver the first real research conversation in Norvii. The React workspace sends a question
scoped to the active corpus, the Go API retrieves bounded evidence from published document
chunks, asks a configured model to synthesize only from that evidence, and returns a structured
stream containing evidence, answer deltas, abstention, cancellation, or safe failure events.
Python owns legal-aware chunking and embedding publication. PostgreSQL remains the canonical
vector store. Neo4j, GraphRAG, MCP, reusable skills, and evaluation dashboards remain deferred.

## Technical Context

**Language/Version**: Go 1.26; Python 3.13; TypeScript 6.0 and React 19 on Node.js 24

**Primary Dependencies**: Existing Go `net/http` and pgx stack; existing Python ingestion and
HTTP transport; existing React Router, i18next, and workspace components; provider-neutral
embedding and chat-model ports with OpenAI-compatible HTTP adapters behind configuration

**Storage**: PostgreSQL with pgvector remains canonical. A migration adds immutable retrieval
chunks and embeddings linked to corpus, source revision, document version, and document unit.
Neo4j is not changed or populated by this feature.

**Testing**: Go unit, handler, contract, PostgreSQL integration, race, and streaming tests;
Python unit and PostgreSQL integration tests for chunking/embedding publication; Vitest,
accessibility, contract, and Playwright tests for the React chat; contract and language
validation; module quality gates and bounded fake-provider journeys

**Target Platform**: Local Linux/WSL development and GitHub Actions with the existing local
PostgreSQL, Neo4j, API, ingestion, and web composition

**Project Type**: Monorepo vertical web application with an online Go API, offline Python
enrichment pipeline, and React client

**Performance Goals**: In deterministic local tests, first visible stream progress within 1
second of the first accepted model event; 95% of bounded requests terminate within 10 seconds;
retrieval returns at most eight evidence chunks and never exceeds the configured context budget

**Constraints**: Single-user POC; English engineering language; English and Portuguese product
locales; active-corpus isolation; no complete document text, full prompts, credentials, or
provider payloads in logs; bounded question, context, response, timeout, and token budgets;
provider unavailability must result in an honest state

**Scale/Scope**: Two initial corpora, up to 20 sources per corpus, immutable document versions,
one in-flight response per workspace, ephemeral conversations, and one retrieval/model provider
configuration at a time

## Constitution Check

*GATE: Passed before Phase 0 research. Re-check after Phase 1 design.*

- **I. Specification before implementation**: PASS. This feature has user journeys,
  acceptance scenarios, measurable outcomes, explicit assumptions, and bounded scope.
- **II. Vertical features and module boundaries**: PASS. React owns presentation; Go owns
  online retrieval, model orchestration, streaming, errors, and telemetry; Python owns chunk and
  embedding publication; PostgreSQL remains the canonical store.
- **III. Evidence-grounded legal answers**: PASS. Answers are generated only from active-corpus
  evidence, citations resolve to immutable document locations, and insufficient support abstains.
- **IV. Versioned cross-language contracts**: PASS. The chat HTTP request, structured stream,
  retrieval evidence, and errors are versioned in this feature before promotion.
- **V. Idiomatic, tested, maintainable code**: PASS. Consumer-owned ports, TDD slices, contract
  tests, and module quality gates are required.
- **VI. Reproducible and cost-bounded POC**: PASS. Retrieval, context, response, timeout, and
  provider budgets are explicit; fake providers make tests deterministic; conversations are not
  persisted.
- **VII. Observable and safe by design**: PASS. Corpus ownership, prompt-injection boundaries,
  cancellation, safe errors, provider timeouts, and content-free telemetry are explicit.
- **VIII. English engineering language**: PASS. Code, contracts, tests, and documentation are
  English; preserved legal excerpts and locale values retain their source language.

## Phase 0: Research Decisions

See [research.md](research.md) for the resolved decisions:

- Use immutable legal-aware chunks anchored to document units and offsets.
- Use a configurable OpenAI-compatible embedding adapter with a fixed 1536-dimensional default
  for the POC and fake deterministic embeddings in automated tests.
- Use a configurable OpenAI-compatible chat-completions adapter with server-side streaming;
  provider details remain behind Go ports.
- Use one provider-neutral NDJSON-over-SSE stream contract with evidence emitted before deltas.
- Enforce grounding with a retrieval score threshold, an evidence-only prompt, citation markers,
  and a completed-event validation step; failure or insufficient support abstains.

## Phase 1: Design and Contracts

### Project Structure

```text
specs/005-grounded-rag-chat/
|-- spec.md
|-- plan.md
|-- research.md
|-- data-model.md
|-- quickstart.md
|-- contracts/
|   |-- chat-http.md
|   `-- chat-stream.schema.json
|-- checklists/requirements.md
`-- tasks.md

apps/api/
|-- internal/chat/{domain,application,http}/
|-- internal/retrieval/{domain,application,postgres}/
|-- internal/platform/{llm,streaming}/
|-- migrations/005_grounded_rag.sql
`-- tests/{contract,integration}/

apps/ingestion/
|-- src/norvii_ingestion/enrichment/{chunking,embedding}/
|-- src/norvii_ingestion/publication/postgres/
`-- tests/{unit,integration}/enrichment/

apps/web/src/
|-- api/{chat,contract}.ts
|-- features/workspace/{ResearchChat,ResearchChatMessage,EvidenceReference}.tsx
|-- i18n/{en,pt}/
`-- features/workspace/*.test.tsx

contracts/corpus-ingestion/v1/  # promoted only after the feature contract stabilizes
```

**Structure Decision**: Extend the existing production modules in their capability-owned
packages. Chat and retrieval ports remain independent of HTTP, PostgreSQL, and model providers.
The client consumes the feature stream through a narrow adapter and never calls a provider or
database directly. The prototype is unchanged.

### Data and Persistence Design

The migration adds a `retrieval_chunks` table with corpus/source-revision/document-version
ownership, optional document-unit identity, contiguous text offsets, immutable normalized chunk
text and hash, ordinal, embedding model/version, vector, and timestamps. A unique identity over
document version, ordinal, and chunk hash makes publication idempotent. Indexes support active
corpus filtering and vector nearest-neighbor search. Chunks from superseded versions remain
immutable but are excluded from retrieval unless they belong to the source's latest published
document.

Python creates chunks at legal-unit boundaries, preserving article context for nested paragraphs
and items, then publishes embeddings in one transaction after the document artifact is ready.
Reprocessing unchanged content is idempotent; a changed document gets a new chunk set linked to
the new immutable version.

### Retrieval and Answer Design

The Go API validates the active corpus and question, creates a request identity, embeds the
question through a provider port, retrieves at most eight chunks with corpus and latest-version
filters, and applies the configured support threshold. If support is insufficient, it emits an
abstention without invoking the answer model. Otherwise it builds a bounded evidence-only prompt
with numbered references and calls the chat-model port. The stream emits request metadata,
evidence references, answer deltas, and exactly one terminal event. The terminal answer is checked
for citation markers and evidence IDs before it can be marked completed; failed validation emits
abstention or safe failure.

The model prompt treats document content as quoted evidence, not instructions. Provider adapters
redact credentials and provider payloads from logs. Disconnects and client cancellation cancel
the model request through context propagation.

### Compatibility and Promotion

The feature contract begins under `specs/005-grounded-rag-chat/contracts/`. Once the stream and
HTTP semantics are stable, promote the compatible schema to `contracts/corpus-ingestion/v2/` or a
separately named chat contract family; do not silently alter corpus-ingestion v1. Provider and
consumer tests must validate both valid and malformed terminal events.

## Complexity Tracking

No constitution violations require justification. The additional retrieval-chunk table and
provider ports are required to keep ingestion, online orchestration, and persistence concerns
decoupled; storing embeddings in the canonical PostgreSQL boundary avoids a new runtime service.

## Norvii Implementation Requirements

### Module Impact

| Module | Change | Responsibility | Verification |
| --- | --- | --- | --- |
| `prototypes/web/` | No change | Preserve approved prototype baseline | Existing prototype CI |
| `apps/web/` | Change | Question composer, streaming state machine, answer rendering, citation navigation, bilingual states | Vitest, accessibility, Playwright, build budget |
| `apps/api/` | Change | Question validation, corpus boundary, retrieval, model orchestration, stream, abstention, telemetry | Go unit, handler, contract, integration, race |
| `apps/ingestion/` | Change | Legal-aware chunking, embedding generation, immutable chunk publication | pytest unit, provider contract, PostgreSQL integration |
| `contracts/` | No change initially | Promote only after feature contract stabilizes | Existing contract validator |
| `infra/` | Change | Configure bounded provider settings and optional local enrichment verification | Bootstrap and service-backed verification |

### Boundaries and Constraints

- **Cost limits**: Configurable question/context/response token budgets, at most eight retrieved
  chunks, one in-flight response, bounded provider timeout, and no persisted conversations.
- **Prototype baseline**: Preserve Feature 004 source tree, document viewer, corpus boundary,
  keyboard behavior, bilingual layout, and technical disclaimer; replace only unavailable chat.
- **Public contracts**: Add feature-local HTTP and stream schemas first; additive promotion is
  allowed only with compatibility tests; breaking changes require a new version.
- **Persistence**: One migration adds immutable retrieval chunks and vectors with corpus and
  document-version ownership, idempotency keys, latest-version filtering, and rollback support.
- **Ingestion artifacts**: Chunk and embedding model versions participate in artifact identity;
  failed enrichment never changes a source's latest ready document.
- **Streaming**: Use SSE framing with JSON event payloads and explicit `started`, `evidence`,
  `delta`, `completed`, `abstained`, `cancelled`, and `error` terminal semantics. The client must
  never display a non-terminal response as completed.
- **Corpus boundary and citations**: Every retrieval query, evidence row, stream event, and
  document navigation target carries corpus and immutable document identity.
- **Security and privacy**: Validate question length, treat retrieved text as untrusted quoted
  evidence, enforce provider timeouts, propagate cancellation, redact secrets, and keep full
  prompts/document text out of logs.
- **Observability**: Record request identity, corpus identity, outcome, evidence count, latency,
  model/embedding versions, and token or character counts when available; omit content.
- **Local environment**: Provider configuration is optional; without valid credentials the UI
  shows a localized unavailable state. Enrichment and chat use deterministic fakes in automated
  tests and never require live providers in CI.

### Repository Paths

```text
apps/web/
apps/api/
apps/ingestion/
contracts/
infra/
specs/005-grounded-rag-chat/
```

## Constitution Re-check After Design

PASS. The design preserves the three-module boundary, adds no graph or MCP behavior, keeps
evidence immutable and corpus-scoped, defines versioned stream semantics before implementation,
and bounds model/provider cost. All remaining provider choices are configuration or adapter
details, not unresolved product behavior.
