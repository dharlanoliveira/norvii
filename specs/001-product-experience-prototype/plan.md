# Implementation Plan: Corpus Research Workspace Prototype

**Branch**: `001-product-experience-prototype` | **Date**: 2026-08-13 | **Spec**: [spec.md](spec.md)

**Input**: Feature specification from `specs/001-product-experience-prototype/spec.md`

## Summary

Build an executable React prototype that validates Norvii's initial product experience without a backend. The prototype presents two deterministic corpora, opens a corpus-scoped research workspace, supports keyboard-accessible source-tree navigation, switches the primary panel between document viewing and chat while preserving state, and demonstrates grounded answers with citations and abstention. A custom editorial research-desk visual direction will be implemented with responsive layouts, bilingual resources, deterministic fixtures, and automated interaction and accessibility checks.

## Technical Context

**Language/Version**: TypeScript 6.0.3, React 19.2.8, HTML, and CSS

**Primary Dependencies**: Vite 8.2.1, React Router 7.18.2, i18next 26.3.6, react-i18next 17.0.11, assistant-ui 0.15.14, and Lucide React 1.31.0

**Storage**: In-memory deterministic fixtures; session-scoped browser state only

**Testing**: Vitest 4.1.10, Testing Library 16.3.2, jest-dom, user-event, axe-core integration, and Playwright 1.62.1

**Target Platform**: Current evergreen desktop browsers at 1280 by 720 and 1440 by 900 reference viewports

**Project Type**: Executable client-side web prototype

**Performance Goals**: Initial catalog becomes interactive within 2 seconds on a typical development machine; mode switches and tree interactions provide visual response within 100 milliseconds

**Constraints**: No backend, network-dependent content, persistence, ingestion, live model, or production-module imports; English default; complete Portuguese localization; WCAG 2.2 AA interaction baseline

**Scale/Scope**: Two corpora, representative PDF and external-link fixtures, four primary journeys, two interface languages, and two reference viewports

## Constitution Check

*GATE: Passed before research and passed again after design.*

- **I. Specification Before Implementation**: PASS. The numbered spec, completed requirements checklist, this plan, and subsequent tasks own the prototype. The user explicitly approved prototype implementation on 2026-08-13.
- **II. Vertical Features and Explicit Module Boundaries**: PASS. All executable work stays in `prototypes/web/`; production modules and public contracts remain unchanged.
- **III. Evidence-Grounded Legal Answers**: PASS. Deterministic answers remain corpus-scoped, cite fixture locations, abstain when unsupported, and carry the technical-demonstration disclaimer.
- **IV. Versioned Cross-Language Contracts**: PASS. No cross-language or production contract is introduced. The feature-local UI state contract documents prototype behavior only.
- **V. Idiomatic, Tested, Maintainable Code**: PASS. Feature-oriented React composition, typed models, deterministic adapters, tests, and a module-owned `Makefile ci` target are planned.
- **VI. Reproducible and Cost-Bounded POC**: PASS. Dependencies and lockfile are versioned; fixtures are local; model, token, storage, and service costs are zero.
- **VII. Observable and Safe by Design**: PASS. Simulated states are visibly labeled; no untrusted upload, URL fetch, secret, prompt, or personal data enters the prototype.
- **VIII. English as the Engineering Language**: PASS. Source, tests, configuration, keys, and documentation use English. Portuguese appears only in `pt` localization values and preserved fixture content.

## Project Structure

### Documentation (this feature)

```text
specs/001-product-experience-prototype/
|-- plan.md
|-- research.md
|-- data-model.md
|-- quickstart.md
|-- contracts/
|   `-- ui-state.md
|-- checklists/
|   `-- requirements.md
`-- tasks.md
```

### Source Code (repository root)

```text
prototypes/web/
|-- public/
|   `-- fixtures/
|-- src/
|   |-- app/
|   |-- components/
|   |-- features/
|   |   |-- catalog/
|   |   `-- workspace/
|   |-- fixtures/
|   |-- i18n/
|   |   |-- en/
|   |   `-- pt/
|   |-- styles/
|   `-- test/
|-- tests/
|   `-- e2e/
|-- Makefile
|-- package.json
|-- package-lock.json
|-- tsconfig.json
|-- vite.config.ts
`-- playwright.config.ts
```

**Structure Decision**: Use one self-contained prototype module under `prototypes/web/`. Feature folders own their models, state, components, and tests. Shared components are limited to demonstrated cross-feature presentation patterns. Deterministic fixtures are adapters for product exploration, not public API contracts.

## Complexity Tracking

No constitution violation or exceptional complexity is required.

## Norvii Implementation Requirements

### Module Impact

| Module | Change | Responsibility | Verification |
| --- | --- | --- | --- |
| `prototypes/web/` | Change | Executable catalog and research-workspace prototype | Format, lint, type check, unit/component tests, accessibility checks, end-to-end journeys, production build |
| `apps/web/` | No change | Production client remains gated | N/A |
| `apps/api/` | No change | No backend behavior | N/A |
| `apps/ingestion/` | No change | No ingestion behavior | N/A |
| `contracts/` | No change | No public cross-language contract | N/A |
| `infra/` | No change | No backing service | N/A |

### Boundaries and Constraints

- **Cost limits**: Two small fixture corpora; zero model tokens, remote storage, external requests, and backing services.
- **Prototype baseline**: This feature creates the initial baseline. Approval evidence includes reference screenshots, interaction tests, and documented review findings.
- **Public contracts**: None. `contracts/ui-state.md` is a feature-local prototype behavior contract and cannot be consumed by production modules.
- **Persistence**: None. Navigation and interaction state live only for the running browser session.
- **Ingestion artifacts**: None. Sources are immutable deterministic fixtures.
- **Streaming**: Responses use a deterministic local sequence; no transport or production message contract is implied.
- **Corpus boundary and citations**: Fixture lookup requires an active corpus identifier; citations can resolve only to a source owned by that corpus.
- **Security and privacy**: No upload, remote fetch, arbitrary HTML, secrets, prompts, or personal data. External-source actions use fixed safe fixture URLs and communicate navigation behavior.
- **Observability**: User-visible states and development diagnostics identify deterministic prototype behavior without exposing document bodies in logs.
- **Local environment**: Node.js 24 and npm; no Docker Compose, migration, or health check impact.
