# Tasks: Planned Hybrid Retrieval

**Input**: Design documents from `specs/009-planned-hybrid-retrieval/`

**Prerequisites**: [spec.md](spec.md), [plan.md](plan.md), [research.md](research.md),
[data-model.md](data-model.md), [planned-hybrid-stream.md](contracts/planned-hybrid-stream.md),
and [quickstart.md](quickstart.md)

**Tests**: Tests are required for every changed behavior, cross-language boundary, failure path,
and user-visible strategy state.

## Phase 1: Foundation

**Purpose**: Establish the contract and domain types used by all stories.

- [X] T001 Define staged inspection and evidence-contribution values in `apps/agent/src/norvii_agent/graph/grounded_chat.py` per FR-006, FR-009, FR-010, FR-011, and FR-016.
- [X] T002 [P] Add agent inspection serialization tests in `apps/agent/tests/unit/test_transport_server.py` per FR-009, FR-010, and FR-016.
- [X] T003 [P] Add Go agent stream-decoding tests in `apps/api/internal/chat/agent/client_test.go` per FR-009, FR-010, and contract `planned-hybrid-stream.md`.
- [X] T004 [P] Add TypeScript stream-validation tests in `apps/web/src/api/chat.test.ts` per FR-009, FR-010, FR-011, and contract `planned-hybrid-stream.md`.
- [X] T005 Implement staged inspection serialization and Go/TypeScript DTO validation in `apps/agent/src/norvii_agent/transport/server.py`, `apps/api/internal/chat/agent/client.go`, `apps/api/internal/chat/domain/service.go`, `apps/api/internal/chat/http/handler.go`, and `apps/web/src/api/chat.ts` per T002-T004 and FR-009 through FR-016.

**Checkpoint**: The public stream can carry validated staged inspection data without changing an
answer's evidence or citation identity.

---

## Phase 2: User Story 1 - Receive a grounded answer through planned hybrid retrieval (Priority: P1)

**Goal**: Every Hybrid question starts with vector evidence and selectively adds graph evidence
only when a bounded planner recognizes a supported graph capability.

**Independent Test**: Unit tests demonstrate a broad Hybrid question retaining vector evidence
after graph skip/no-evidence/failure and a relationship question adding graph evidence with a path.

### Tests for User Story 1

- [X] T006 [P] [US1] Add planner decision and provider-response validation tests in `apps/agent/tests/unit/test_graph_planner.py` per FR-003, FR-004, and FR-016.
- [X] T007 [P] [US1] Add graph-capability catalog and parameterized-filter tests in `apps/agent/tests/unit/test_graph_retrieval.py` per FR-003, FR-004, FR-005, and FR-012.
- [X] T008 [P] [US1] Extend vector-first Hybrid composition tests in `apps/agent/tests/unit/test_hybrid_retrieval.py` per FR-002, FR-005, FR-006, FR-007, and FR-008.
- [X] T009 [P] [US1] Extend grounded-chat inspection and no-evidence tests in `apps/agent/tests/unit/test_grounded_chat.py` per FR-008, FR-009, FR-010, and FR-011.

### Implementation for User Story 1

- [X] T010 [US1] Create bounded graph capability and Hybrid plan domain values in `apps/agent/src/norvii_agent/retrieval/planning.py` per FR-003, FR-004, and `data-model.md`.
- [X] T011 [US1] Implement an OpenAI-compatible structured Hybrid planner in `apps/agent/src/norvii_agent/providers/planning.py` and export it from `apps/agent/src/norvii_agent/providers/__init__.py` per FR-003, FR-004, FR-015, and FR-016.
- [X] T012 [US1] Extend `apps/agent/src/norvii_agent/retrieval/graph.py` with active-snapshot capability discovery and bounded parameterized graph-plan execution per FR-003, FR-004, FR-005, FR-012, and FR-013.
- [X] T013 [US1] Refactor `apps/agent/src/norvii_agent/retrieval/hybrid.py` into vector-first adaptive composition with explicit stage results and deduplicated evidence contribution per FR-002 and FR-005 through FR-011.
- [X] T014 [US1] Update `apps/agent/src/norvii_agent/graph/grounded_chat.py` and `apps/agent/src/norvii_agent/transport/__main__.py` to generate from merged evidence and compose the planner without hidden global state per FR-006 through FR-012.
- [X] T015 [US1] Run the focused agent tests and `uv run ruff check src tests && uv run mypy src` in `apps/agent/` per SC-001 through SC-004.
- [X] T015a [US1] Select validated canonical graph entity labels rather than question-literal terms, so a question can use a graph whose labels are in another language; cover the planner and Neo4j parameter boundary.

**Checkpoint**: Hybrid always has a vector baseline, safely augments from graph when appropriate,
and preserves evidence/citation boundaries.

---

## Phase 3: User Story 2 - Select an intelligible retrieval approach (Priority: P2)

**Goal**: New questions and comparisons allow only Vector and Hybrid, while Hybrid remains valid
without a ready graph release.

**Independent Test**: API, client-domain, and workspace tests reject Graph for a new request and
submit/compare only Vector and Hybrid.

### Tests for User Story 2

- [X] T016 [P] [US2] Update Go request-validation and application behavior tests in `apps/api/internal/chat/http/handler_test.go` and `apps/api/internal/chat/application/service_test.go` per FR-001, FR-007, and FR-014.
- [X] T017 [P] [US2] Update browser strategy contract and comparison-domain tests in `apps/web/src/api/chat.test.ts` and `apps/web/src/research/domain/strategyComparison.test.ts` per FR-001 and FR-014.
- [X] T018 [P] [US2] Update strategy-selector and comparison journeys in `apps/web/src/features/workspace/ResearchChat.test.tsx` and `apps/web/src/features/workspace/StrategyComparison.test.tsx` per FR-001, FR-002, FR-007, and FR-015.

### Implementation for User Story 2

- [X] T019 [US2] Restrict new API strategy validation to Vector and Hybrid and remove Hybrid graph-release preflight blocking in `apps/api/internal/chat/http/handler.go` and `apps/api/internal/chat/application/service.go` per FR-001, FR-005, FR-007, and FR-014.
- [X] T020 [US2] Restrict new agent transport strategy validation to Vector and Hybrid in `apps/agent/src/norvii_agent/transport/server.py` per FR-001 and FR-014.
- [X] T021 [US2] Restrict client strategy values and comparison candidates in `apps/web/src/api/chat.ts` and `apps/web/src/research/domain/strategyComparison.ts` per FR-001 and FR-014.
- [X] T022 [US2] Update strategy selection, comparison presentation, and localized strategy copy in `apps/web/src/features/workspace/ResearchChat.tsx`, `apps/web/src/features/workspace/StrategyComparison.tsx`, `apps/web/src/i18n/en/translation.ts`, and `apps/web/src/i18n/pt/translation.ts` per FR-001, FR-014, and FR-015.
- [X] T023 [US2] Run Go and web validation in `apps/api/` and `apps/web/` per SC-001, SC-004, and SC-005.

**Checkpoint**: The workspace presents two meaningful approaches and a missing graph release no
longer makes Hybrid unusable.

---

## Phase 4: User Story 3 - Inspect how Hybrid made its retrieval decision (Priority: P3)

**Goal**: The research record distinguishes vector evidence, planning, graph contribution, and
safe non-contribution outcomes in the selected interface language.

**Independent Test**: A completed Hybrid event renders vector, planning, and graph stage states;
each non-contribution state is understandable without being shown as an answer failure.

### Tests for User Story 3

- [X] T024 [P] [US3] Add research-record stage rendering tests in `apps/web/src/features/workspace/ResearchChat.test.tsx` per FR-009, FR-010, FR-011, and FR-015.
- [X] T025 [P] [US3] Add localized stage-label and safe-reason coverage tests in `apps/web/src/i18n/config.test.ts` per FR-011 and FR-015.

### Implementation for User Story 3

- [X] T026 [US3] Render structured stage contributions and graph-path evidence in `apps/web/src/features/workspace/ResearchChat.tsx` and `apps/web/src/features/workspace/workspace.css` per FR-009 through FR-013.
- [X] T027 [US3] Add complete English and Portuguese stage-state, reason, and contribution resources in `apps/web/src/i18n/en/translation.ts` and `apps/web/src/i18n/pt/translation.ts` per FR-011 and FR-015.
- [X] T028 [US3] Update the agent module responsibility document in `docs/modules/python-agent.md` per FR-003 through FR-016.

**Checkpoint**: The research record explains whether the graph helped, was skipped, had no path,
or was unavailable without exposing private reasoning.

---

## Phase 5: Polish and Cross-Cutting Validation

- [ ] T029 Validate the Feature 009 quickstart scenarios in `specs/009-planned-hybrid-retrieval/quickstart.md` per SC-001 through SC-007.
- [X] T030 Run repository format, static analysis, tests, build, language validation, and `git diff --check`; record results in `specs/009-planned-hybrid-retrieval/tasks.md` per Principles V and VIII.

## Verification Record

- 2026-08-25: `make ci` passed in `apps/agent/` (format, lint, mypy, 24 selected pytest tests, build).
- 2026-08-25: `make ci` passed in `apps/api/` (dependency checks, formatting, vet, race tests, build).
- 2026-08-25: `make ci` passed in `apps/web/` (format, lint, typecheck, 67 Vitest tests, production build, and 3 Playwright tests).
- 2026-08-25: `python3 .github/scripts/validate_repository_language.py` and `git diff --check` passed from the repository root.
- 2026-08-25: Canonical-entity planning verification passed: an English question about the data
  protection authority selected the Portuguese `autoridade nacional` graph entity and retrieved
  five snapshot-scoped graph evidence locations in an isolated configured-provider run.
- T029 remains open for configured-provider, manual quickstart scenarios. Automated unit, contract, and browser verification is complete; the live planner and ready-graph-release paths need validation in a local environment with an active model provider.

## Dependencies and Execution Order

- Phase 1 blocks all implementation because the stream inspection contract is shared.
- User Story 1 depends on Phase 1 and is the functional MVP.
- User Story 2 depends on User Story 1 because public selection invokes the planned Hybrid path.
- User Story 3 depends on Phase 1 and User Story 1 inspection values; it can proceed after their
  contract is stable.
- Phase 5 depends on all preceding phases.

## Parallel Opportunities

- T002-T004 can run in parallel because each is an independent consumer-boundary test.
- T006-T009 can run in parallel before implementation because they cover distinct agent seams.
- T016-T018 can run in parallel because they cover Go API, client domain, and workspace behavior.
- T024-T025 can run in parallel after the inspection contract is stable.

## Implementation Strategy

1. Establish the stream data model and tests.
2. Build and validate the vector-first Hybrid composition before changing user controls.
3. Restrict public strategy selection to Vector/Hybrid.
4. Present stage-level evidence in the research record.
5. Complete the quickstart and repository quality gates.
