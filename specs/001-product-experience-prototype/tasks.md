# Tasks: Corpus Research Workspace Prototype

**Input**: Design documents from `specs/001-product-experience-prototype/`

**Prerequisites**: `plan.md`, `spec.md`, `research.md`, `data-model.md`, `contracts/ui-state.md`, `quickstart.md`

**Tests**: Required for changed behavior, accessibility, failure paths, localization parity, and primary browser journeys.

**Organization**: Tasks are grouped by user story so each journey reaches an independently testable checkpoint.

## Phase 1: Setup

**Purpose**: Establish a reproducible prototype module and owned CI contract.

- [X] T001 Initialize the pinned React, TypeScript, and Vite module in `prototypes/web/package.json`, `prototypes/web/package-lock.json`, `prototypes/web/index.html`, and `prototypes/web/src/main.tsx` (FR-029)
- [X] T002 Configure strict TypeScript, Vite, Vitest, ESLint, Prettier, and Playwright in `prototypes/web/tsconfig.json`, `prototypes/web/tsconfig.app.json`, `prototypes/web/tsconfig.node.json`, `prototypes/web/vite.config.ts`, `prototypes/web/eslint.config.js`, `prototypes/web/.prettierrc.json`, and `prototypes/web/playwright.config.ts` (FR-026, FR-029, FR-031)
- [X] T003 Create the module-owned CI and developer commands in `prototypes/web/Makefile` and update `prototypes/web/README.md` (FR-029, FR-031)

---

## Phase 2: Foundational

**Purpose**: Provide typed fixtures, localization, application routing, and the visual foundation required by every story.

**CRITICAL**: No user-story implementation begins until this phase is complete.

- [X] T004 [P] Define immutable corpus, source, viewer, conversation, message, and citation types in `prototypes/web/src/fixtures/models.ts` (FR-005, FR-016, FR-022)
- [X] T005 [P] Create isolated Portuguese and English corpus fixtures with PDF, external-link, citation, answer, abstention, and unavailable states in `prototypes/web/src/fixtures/corpora.ts` (FR-001, FR-005, FR-015, FR-016, FR-018, FR-029, FR-030)
- [X] T006 [P] Create parity-tested English and Portuguese localization resources in `prototypes/web/src/i18n/en/translation.ts`, `prototypes/web/src/i18n/pt/translation.ts`, `prototypes/web/src/i18n/config.ts`, and `prototypes/web/src/i18n/config.test.ts` (FR-019, FR-021, FR-022)
- [X] T007 [P] Implement the editorial design tokens, typography, reset, focus, motion, and responsive foundations in `prototypes/web/src/styles/tokens.css` and `prototypes/web/src/styles/global.css` (FR-023, FR-025, FR-026, FR-027, FR-028)
- [X] T008 Implement the application shell, route definitions, language control, and global disclaimer in `prototypes/web/src/app/App.tsx`, `prototypes/web/src/app/AppShell.tsx`, and `prototypes/web/src/app/routes.tsx` (FR-004, FR-019, FR-020, FR-030)
- [X] T009 [P] Add shared accessible icons, badges, segmented control, empty state, and status components in `prototypes/web/src/components/` (FR-008, FR-012, FR-024, FR-025, FR-026)
- [X] T010 Configure deterministic test rendering, accessibility assertions, and fixture helpers in `prototypes/web/src/test/setup.tsx` and `prototypes/web/src/test/render.tsx` (FR-026, FR-031)

**Checkpoint**: The module builds, routes, localizes, and renders a reusable visual foundation without network or backend dependencies.

---

## Phase 3: User Story 1 - Choose a legal corpus (Priority: P1)

**Goal**: Present two clear corpus choices and navigate into an isolated workspace.

**Independent Test**: Identify both corpus languages and jurisdictions, open each corpus, and recover from an unknown corpus route.

### Tests for User Story 1

- [X] T011 [P] [US1] Add catalog content, navigation, and unknown-corpus recovery tests in `prototypes/web/src/features/catalog/CorpusCatalogPage.test.tsx` and `prototypes/web/src/features/workspace/UnknownCorpusPage.test.tsx` (FR-001, FR-002, FR-004)

### Implementation for User Story 1

- [X] T012 [P] [US1] Implement the corpus card with language, jurisdiction, purpose, source count, and interaction states in `prototypes/web/src/features/catalog/CorpusCard.tsx` and `prototypes/web/src/features/catalog/catalog.css` (FR-002, FR-023, FR-024, FR-025)
- [X] T013 [US1] Implement the catalog page and route navigation in `prototypes/web/src/features/catalog/CorpusCatalogPage.tsx` (FR-001, FR-002, FR-004)
- [X] T014 [US1] Implement the corpus route boundary and recovery page in `prototypes/web/src/features/workspace/CorpusWorkspacePage.tsx` and `prototypes/web/src/features/workspace/UnknownCorpusPage.tsx` (FR-004, FR-005)

**Checkpoint**: User Story 1 is independently navigable and testable.

---

## Phase 4: User Story 2 - Browse and open corpus sources (Priority: P1)

**Goal**: Navigate an accessible source tree and view PDF and external-link sources without leaving the workspace.

**Independent Test**: Use pointer and keyboard to expand groups, open each source type, change the active source, and review unavailable-source recovery.

### Tests for User Story 2

- [X] T015 [P] [US2] Add tree keyboard, selection, PDF viewer, external viewer, and unavailable-source tests in `prototypes/web/src/features/workspace/SourceTree.test.tsx` and `prototypes/web/src/features/workspace/SourceViewer.test.tsx` (FR-006, FR-007, FR-008, FR-009, FR-010, FR-011)

### Implementation for User Story 2

- [X] T016 [P] [US2] Implement the accessible corpus-rooted source tree in `prototypes/web/src/features/workspace/SourceTree.tsx` (FR-007, FR-008, FR-009, FR-026)
- [X] T017 [P] [US2] Implement PDF document presentation and stable location navigation in `prototypes/web/src/features/workspace/PdfSourceViewer.tsx` (FR-010)
- [X] T018 [P] [US2] Implement external-link preview and safe original-destination action in `prototypes/web/src/features/workspace/ExternalSourceViewer.tsx` (FR-011)
- [X] T019 [US2] Compose source selection and viewer recovery states in `prototypes/web/src/features/workspace/SourceViewer.tsx` and `prototypes/web/src/features/workspace/workspace.css` (FR-006, FR-009, FR-024, FR-025, FR-028)

**Checkpoint**: User Story 2 independently demonstrates source discovery and reading for both source types.

---

## Phase 5: User Story 3 - Alternate between source review and chat (Priority: P1)

**Goal**: Preserve source and conversation context while switching modes and opening citations.

**Independent Test**: Open a source, create a draft, switch modes, submit cited and unsupported questions, open a citation, and verify all state is preserved.

### Tests for User Story 3

- [X] T020 [P] [US3] Add controller tests for corpus isolation, mode preservation, draft retention, deterministic responses, abstention, and citation navigation in `prototypes/web/src/features/workspace/useWorkspaceController.test.tsx` (FR-005, FR-013, FR-014, FR-015, FR-016, FR-017, FR-018)
- [X] T021 [P] [US3] Add chat and mode-selector interaction tests in `prototypes/web/src/features/workspace/ResearchChat.test.tsx` and `prototypes/web/src/features/workspace/WorkspaceModeSelector.test.tsx` (FR-012, FR-013, FR-015, FR-017, FR-018)

### Implementation for User Story 3

- [X] T022 [US3] Implement the cohesive workspace state controller and corpus-bound selectors in `prototypes/web/src/features/workspace/useWorkspaceController.ts` (FR-005, FR-013, FR-014, FR-015, FR-016, FR-017, FR-018)
- [X] T023 [P] [US3] Implement the accessible Chat and Source mode selector in `prototypes/web/src/features/workspace/WorkspaceModeSelector.tsx` (FR-012, FR-013)
- [X] T024 [US3] Implement deterministic assistant-ui chat presentation, composer, response states, citations, and abstention in `prototypes/web/src/features/workspace/ResearchChat.tsx` (FR-015, FR-016, FR-017, FR-018, FR-030)
- [X] T025 [US3] Integrate preserved viewer and chat state into `prototypes/web/src/features/workspace/CorpusWorkspacePage.tsx` (FR-006, FR-012, FR-013, FR-014, FR-017)

**Checkpoint**: User Story 3 independently demonstrates Norvii's source-to-chat research loop.

---

## Phase 6: User Story 4 - Use a polished bilingual experience (Priority: P2)

**Goal**: Complete every primary journey in English and Portuguese with first-class visual, responsive, and accessible behavior.

**Independent Test**: Switch language in an active workspace, complete all journeys by keyboard, and review both reference viewports with normal and reduced motion.

### Tests for User Story 4

- [X] T026 [P] [US4] Add language-state preservation and full-page accessibility tests in `prototypes/web/src/app/App.test.tsx` (FR-019, FR-020, FR-021, FR-022, FR-026)
- [X] T027 [P] [US4] Add Playwright catalog, source, chat, citation, keyboard, and responsive journeys in `prototypes/web/tests/e2e/research-workspace.spec.ts` (FR-020, FR-026, FR-028, FR-031)

### Implementation for User Story 4

- [X] T028 [US4] Refine catalog and workspace hierarchy, typography, responsive behavior, focus, selection, feedback, and reduced motion in `prototypes/web/src/styles/global.css`, `prototypes/web/src/features/catalog/catalog.css`, and `prototypes/web/src/features/workspace/workspace.css` (FR-023, FR-024, FR-025, FR-026, FR-027, FR-028)
- [X] T029 [US4] Complete all English and Portuguese copy, accessible names, statuses, and recovery guidance in `prototypes/web/src/i18n/en/translation.ts` and `prototypes/web/src/i18n/pt/translation.ts` (FR-019, FR-021, FR-022, FR-030)
- [X] T030 [US4] Capture deterministic visual baselines for both reference viewports in `prototypes/web/tests/e2e/research-workspace.spec.ts-snapshots/` and document selected design evidence in `specs/001-product-experience-prototype/quickstart.md` (FR-028, FR-031)

**Checkpoint**: User Story 4 completes the reviewable bilingual prototype baseline.

---

## Phase 7: Polish and Cross-Cutting Verification

**Purpose**: Ensure the full prototype is maintainable, reproducible, and ready for stakeholder layout validation.

- [X] T031 [P] Audit dependency, semantic, keyboard, contrast, reduced-motion, and external-link behavior in `prototypes/web/package.json` and `prototypes/web/src/` (FR-011, FR-026, FR-027, FR-029)
- [X] T032 Run and document `make -C prototypes/web ci` and the Feature 001 quickstart in `specs/001-product-experience-prototype/quickstart.md` (FR-031)
- [X] T033 Record prototype review status, known layout questions, and approval evidence in `specs/001-product-experience-prototype/review.md` (FR-023, FR-028, FR-031)

---

## Dependencies and Execution Order

### Phase dependencies

- Setup has no dependencies.
- Foundational depends on Setup and blocks all user stories.
- US1 depends on Foundational and establishes route navigation.
- US2 depends on Foundational and can be built independently using fixture-selected state.
- US3 depends on the source viewer from US2 for citation navigation and on the workspace route from US1.
- US4 depends on all primary journeys because it verifies complete localization, accessibility, and visual behavior.
- Polish depends on every included story.

### Parallel opportunities

- T004 through T007 and T009 can proceed in parallel after configuration exists.
- Test tasks marked `[P]` can be authored before their associated implementation and run concurrently.
- Source tree, PDF viewer, and external viewer can be implemented in parallel.
- Localization, visual foundations, and fixture models use separate files and can be developed concurrently before integration.

## Implementation Strategy

1. Establish reproducible tooling and deterministic foundations.
2. Deliver the catalog route as the first reviewable slice.
3. Add source navigation and viewing as the second slice.
4. Add preserved chat and citation navigation as the core POC slice.
5. Complete bilingual, accessibility, responsive, and visual validation.
6. Run the module CI contract and capture evidence before requesting prototype approval.

## Notes

- `[P]` tasks touch independent files and can run concurrently when dependencies are satisfied.
- Every implementation task identifies the requirements it owns.
- Production modules under `apps/` and shared contracts under `contracts/` remain untouched.
- Completed tasks are marked `[X]` only after their relevant verification passes.
