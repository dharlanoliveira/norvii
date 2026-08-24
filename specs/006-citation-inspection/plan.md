# Implementation Plan: Citation Navigation and Inspection

**Branch**: `006-citation-inspection` | **Date**: 2026-08-24 | **Spec**: [spec.md](spec.md)

## Summary

Make every grounded-answer citation resolve to the immutable document version and exact legal
span that produced it. Add a session-scoped, answer-level inspection disclosure that exposes
retrieval order, safe relevance metadata, execution measurements, and evidence navigation.

The implementation will extend the existing versioned chat SSE contract additively, collect
measurements at their owning boundaries, and retain inspection records only in the browser's
current Assistant UI runtime. A new corpus- and source-scoped document-version read route will
serve an old published version only when it matches the cited immutable identity. The existing
latest-document route remains the normal source browsing path.

## Technical Context

**Language/Version**: TypeScript 6 / React 19 / Node 24; Go 1.26; Python 3.13

**Primary Dependencies**: Vite, Assistant UI, react-i18next, Vitest, Playwright; Go standard
library, pgx; LangGraph, psycopg, OpenAI-compatible HTTP client, pytest

**Storage**: Existing PostgreSQL `document_versions`, `document_units`, `retrieval_chunks`,
`sources`, and source revisions; no new persistence and no Neo4j change

**Testing**: Vitest and Testing Library with axe-core; Playwright; Go unit and HTTP-handler tests;
pytest; repository language and contract validators

**Target Platform**: Local Docker-backed POC and GitHub Actions on Linux; modern desktop browser

**Project Type**: Three-module web application: React client, Go HTTP facade, Python LangGraph
agent

**Performance Goals**: Citation selection needs no more than two interactions; deterministic
inspection is available within one second of terminal answer state; scrolling visibly reveals the
selected legal span

**Constraints**: Active-corpus isolation; immutable document-version identity; fail closed on
missing, foreign, or no-longer-published evidence; no token estimates; no provider payload,
prompt, credential, or cross-corpus disclosure; English and Portuguese product copy; no persisted
chat history

**Scale/Scope**: Single-user POC, two initial corpora, up to eight retrieved references per
answer, one session-local inspection record per terminal Assistant UI message

## Constitution Check

| Principle | Status | Plan response |
|---|---|---|
| I. Spec-first delivery | Pass | This plan derives from Feature 006 specification, research, contract, and data model artifacts before task generation. |
| II. Vertical module boundaries | Pass | React owns presentation/session state, Go owns public HTTP and document ownership checks, and Python owns retrieval/generation measurements. |
| III. Evidence-grounded answers | Pass | Navigation validates corpus, source, document version, location, and offsets; unresolved evidence never falls back to current text. |
| IV. Versioned cross-language contracts | Pass | The current SSE contract receives additive inspection fields and a documented immutable document route. |
| V. Quality gates | Pass | Unit, integration, accessibility, browser, contract, language, and module quality checks are planned. |
| VI. Reproducible, cost-aware delivery | Pass | Existing local services are reused; token use is supplied only by the provider and is never estimated. |
| VII. Safe observability | Pass | Inspection contains structured measurements and cited excerpts only; it excludes prompts, credentials, raw provider payloads, and user-question logs. |
| VIII. English engineering language | Pass | Plan and implementation artifacts remain English; Portuguese is limited to localization values and preserved legal text. |

The check was repeated after Phase 1 design. No constitution exception or complexity justification
is required.

## Project Structure

### Documentation

```text
specs/006-citation-inspection/
|-- spec.md
|-- plan.md
|-- research.md
|-- data-model.md
|-- quickstart.md
`-- contracts/
    `-- citation-inspection.md
```

### Source Code

```text
apps/
|-- web/
|   `-- src/
|       |-- api/
|       |   |-- chat.ts
|       |   |-- contract.ts
|       |   `-- researchProvider.ts
|       `-- features/workspace/
|           |-- CorpusWorkspacePage.tsx
|           |-- LegalDocumentReader.tsx
|           |-- ResearchChat.tsx
|           `-- useNorviiChatRuntime.ts
|-- api/
|   `-- internal/
|       |-- chat/{agent,domain,http}/
|       `-- document/{http,postgres}/
`-- agent/
    `-- src/norvii_agent/
        |-- graph/grounded_chat.py
        |-- providers/chat.py
        |-- retrieval/postgres.py
        `-- transport/server.py
```

**Structure Decision**: The feature extends the existing vertical modules. It introduces no shared
library, new runtime, database table, migration, or Neo4j dependency. Feature-local contracts
describe the change; implementation updates the established Feature 005 SSE contract as its
public source of truth.

## Delivery Design

1. Extend the Python graph result with typed retrieval facts and timing. The retriever will return
   the source title, immutable document version identifier, and nullable cosine distance together
   with the current ordered evidence. The graph measures retrieval and generation independently.
2. Extend the OpenAI-compatible provider to read usage only when the provider actually supplies
   it. It returns nullable input and output token values; character counts and guessed values are
   prohibited.
3. Relay the typed inspection payload from Python through Go without rebuilding or defaulting its
   component values. The Go public handler measures the complete facade request and adds that
   measured total to the terminal SSE payload.
4. Add a document-version route that reads a requested published `documentVersionId` only when it
   belongs to the requested corpus and source. The latest-document route remains unchanged. A
   missing, foreign, unpublished, or mismatched version returns the existing safe not-found
   response.
5. Extend the React parser and Assistant UI runtime to retain an immutable answer inspection per
   assistant message. Render one accessible inline disclosure per completed answer; never render
   evidence inspection for abstained, cancelled, or failed outcomes.
6. Route inline citations and inspection evidence through one resolver. It verifies the active
   corpus and returned document version, maps nested locations to their visible parent legal unit,
   validates the cited offset range, scrolls it into view, and highlights the exact span. Failure
   is explicit and localized, never a closest-match fallback.

## Verification Strategy

- Go repository and handler tests cover latest versus explicit immutable document reads, corpus
  ownership, published-state filtering, and no-substitution errors.
- Python unit tests cover vector retrieval ordering and cosine distance propagation, timing
  ownership, nullable provider usage, and safe terminal payloads.
- Go agent-client and SSE tests prove inspection metadata survives Python-to-Go-to-browser
  transport and that total request duration is measured rather than hard-coded.
- React parser/runtime/component tests cover typed validation, message-scoped inspection, localized
  unavailable values, keyboard operation, foreign/unresolved citation failures, nested-unit offset
  highlighting, and answer preservation.
- Playwright verifies source switching and return to the same session answer for both direct and
  inspection citations.
- Run the module quality commands, repository contract and language validators, and the existing
  Sonar-backed CI workflow before merge.

## Complexity Tracking

No constitution violations require justification.
