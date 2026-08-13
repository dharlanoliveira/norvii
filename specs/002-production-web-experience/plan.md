# Implementation Plan: Production Corpus Research Experience

**Branch**: `002-production-web-experience` | **Date**: 2026-08-13 | **Spec**: [spec.md](spec.md)

**Input**: Feature specification from `specs/002-production-web-experience/spec.md`

## Summary

Create the first production React module under `apps/web/` by independently implementing the catalog and research workspace approved in Feature 001. The client uses production-owned deterministic legal demonstration data behind narrow domain boundaries, assistant-ui as a structured chat renderer, lazy route loading, typed English and Portuguese resources, and behavior-first automated tests. No production file imports from the prototype, and no backend, persistence, ingestion, PostgreSQL, or Neo4j service participates in this increment.

## Technical Context

**Language/Version**: TypeScript 6.0.3, React 19.2.8, HTML, and CSS

**Primary Dependencies**: Vite 8.2.1, React Router 7.18.2, i18next 26.3.6, react-i18next 17.0.11, assistant-ui 0.15.14, Lucide React 1.31.0, Fraunces variable font 5.3.0, and Manrope variable font 5.3.0

**Storage**: Production-owned immutable demonstration records and session-scoped React state; no durable storage

**Testing**: Vitest 4.1.10, Testing Library 16.3.2, user-event 14.6.4, jest-dom 7.0.1, axe-core 4.13.0, and Playwright 1.62.1

**Target Platform**: Current evergreen desktop browsers at 1280 by 720 and 1440 by 900 reference viewports

**Project Type**: Production client-side web application

**Performance Goals**: Catalog interactive within 2 seconds on the reference development environment; interaction feedback within 100 milliseconds; compressed initial JavaScript entry below 350 kB

**Constraints**: English default, complete Portuguese localization, WCAG 2.2 AA baseline, no backend or network corpus dependency, no prototype imports, no persistence, and no live model or database cost

**Scale/Scope**: Two corpora, representative PDF and external-link sources, four primary journeys, two interface languages, two reference viewports, and one production module

## Constitution Check

*GATE: Passed before research and passed again after Phase 1 design.*

- **I. Specification Before Implementation**: PASS. Feature 001 is Verified and approved. Feature 002 has its own approved specification, requirement checklist, plan, design artifacts, and traceable TDD tasks before application code.
- **II. Vertical Features and Explicit Module Boundaries**: PASS. Only `apps/web/` changes. Domain records and prepared response behavior stay independent of React and assistant-ui adapters. Other production modules remain untouched.
- **III. Evidence-Grounded Legal Answers**: PASS. All demonstration responses are corpus-scoped, contain structured citations, abstain when unsupported, and display the technical-demonstration disclaimer.
- **IV. Versioned Cross-Language Contracts**: PASS. No cross-language exchange is introduced. The feature-local client provider contract is explicitly non-public and will be replaced by a versioned contract when the Go API is introduced.
- **V. Idiomatic, Tested, Maintainable Code**: PASS. Strict TypeScript, functional React composition, narrow dependency seams, public-behavior tests, and red-green-refactor cycles are required.
- **VI. Reproducible and Cost-Bounded POC**: PASS. Dependencies are pinned in a lockfile; all records and responses are local; service, storage, network, model, and token costs are zero.
- **VII. Observable and Safe by Design**: PASS. Demonstration states are visible; no arbitrary URL, upload, secret, prompt, personal data, or remote document enters the client. External actions use fixed HTTPS destinations with `noopener noreferrer`.
- **VIII. English as the Engineering Language**: PASS. Code, configuration, tests, keys, and documents use English. Portuguese occurs only in the Portuguese resource and preserved legal demonstration content.

## Project Structure

### Documentation (this feature)

```text
specs/002-production-web-experience/
|-- spec.md
|-- plan.md
|-- research.md
|-- data-model.md
|-- quickstart.md
|-- contracts/
|   `-- client-provider.md
|-- checklists/
|   |-- requirements.md
|   `-- production-readiness.md
`-- tasks.md
```

### Source Code (repository root)

```text
apps/web/
|-- public/
|-- scripts/
|   `-- check-bundle-budget.mjs
|-- src/
|   |-- app/
|   |   |-- App.tsx
|   |   |-- AppShell.tsx
|   |   `-- routes.tsx
|   |-- components/
|   |-- features/
|   |   |-- catalog/
|   |   `-- workspace/
|   |-- research/
|   |   |-- domain/
|   |   |-- demonstration/
|   |   `-- adapters/
|   |-- i18n/
|   |   |-- en/
|   |   `-- pt/
|   |-- styles/
|   |-- test/
|   `-- main.tsx
|-- tests/e2e/
|-- Makefile
|-- package.json
|-- package-lock.json
|-- tsconfig.json
|-- vite.config.ts
|-- vitest.config.ts
`-- playwright.config.ts
```

**Structure Decision**: Create one independently buildable production web module. Vertical UI behavior stays in `features/catalog` and `features/workspace`; stable research vocabulary and invariants stay in `research/domain`; production demonstration records and response matching stay in `research/demonstration`; framework translation stays in `research/adapters`. Shared components are introduced only for demonstrated reuse. No file imports from `prototypes/web/`.

## Complexity Tracking

No constitution violation or exceptional complexity is required.

## Norvii Implementation Requirements *(mandatory)*

### Module Impact

| Module | Change | Responsibility | Verification |
| --- | --- | --- | --- |
| `prototypes/web/` | No change | Approved visual and interaction reference only | Existing review evidence |
| `apps/web/` | Change | Production catalog, workspace, source viewer, structured demonstration chat, localization, accessibility, and module CI | Format, lint, type check, component tests, accessibility tests, browser journeys, bundle budget, and production build |
| `apps/api/` | No change | No online backend behavior in this feature | N/A |
| `apps/ingestion/` | No change | No offline processing behavior in this feature | N/A |
| `contracts/` | No change | No cross-language exchange exists yet | N/A |
| `infra/` | No change | No backing service is needed | N/A |

### Boundaries and Constraints

- **Cost limits**: Exactly two small production-owned demonstration corpora; zero model tokens, remote storage, external document requests, and backing services.
- **Prototype baseline**: Feature 001 review and four reference screenshots define the accepted editorial research-desk hierarchy. Production code is independently authored. Route-level loading and the initial-entry budget are intentional improvements.
- **Public contracts**: None. The feature-local `client-provider.md` describes a client seam and cannot be consumed as a Go or Python contract.
- **Persistence**: None. All state is limited to the mounted browser session; no schema, migration, retention, or rollback behavior is introduced.
- **Ingestion artifacts**: None. Demonstration sources are immutable production-client records and are not ingestion output.
- **Streaming**: assistant-ui receives structured local message parts from a deterministic adapter. The shape does not claim compatibility with the future Go stream.
- **Corpus boundary and citations**: Domain lookup validates that selected sources and citations belong to the active corpus before presentation. Invalid cross-corpus references become controlled failures.
- **Security and privacy**: No arbitrary input is interpreted as HTML or a URL. Fixed external links use HTTPS and safe new-context attributes. Logs contain no legal content or user questions.
- **Observability**: Loading, empty, responding, completed, abstained, and failed states are user-visible; no production telemetry service is added.
- **Local environment**: Node.js 24 and npm only. Docker Compose, migrations, health checks, PostgreSQL, and Neo4j are unchanged.

### TDD Execution Model

Implementation proceeds in vertical red-green-refactor cycles. Each behavior task begins with one test through a public route, rendered component, or domain boundary. The test must fail for the expected missing behavior before the smallest implementation is added. Internal collaborators are not mocked; only browser or future external boundaries may be replaced. Each user story finishes at a runnable checkpoint before the next story begins.

### Repository Paths

```text
apps/web/                              # Only production application module changed
specs/002-production-web-experience/  # Feature reasoning and verification
docs/product/feature-map.md            # Durable feature sequence correction
```
