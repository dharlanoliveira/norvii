# Tasks: Production Corpus Research Experience

**Input**: Design documents from `specs/002-production-web-experience/`

**Prerequisites**: `plan.md`, `spec.md`, `research.md`, `data-model.md`, `contracts/client-provider.md`, and completed checklists

**Tests**: Required. Every behavior is implemented as a vertical RED-GREEN-REFACTOR cycle through a public domain or user interface boundary.

**Organization**: Tasks are grouped by independently testable user story. A RED task must fail for the missing behavior before its paired GREEN task starts.

## Phase 1: Setup

**Purpose**: Establish one independently buildable production React module without importing prototype files.

- [x] T001 Create the pinned React, TypeScript, Vite, Vitest, Playwright, ESLint, and Prettier module manifests in `apps/web/package.json`, `apps/web/package-lock.json`, `apps/web/tsconfig.json`, `apps/web/tsconfig.app.json`, `apps/web/tsconfig.node.json`, `apps/web/vite.config.ts`, `apps/web/vitest.config.ts`, and `apps/web/playwright.config.ts` per FR-023 and FR-024
- [x] T002 Create the module CI and repository hygiene configuration in `apps/web/Makefile`, `apps/web/eslint.config.js`, `apps/web/.prettierrc.json`, and `apps/web/.prettierignore` per FR-028
- [x] T003 Create the HTML and React composition entry points in `apps/web/index.html`, `apps/web/src/main.tsx`, and `apps/web/src/app/App.tsx` per FR-021 and FR-023

**Checkpoint**: The empty production module installs, type checks, and exposes a test environment independently from the prototype.

---

## Phase 2: Foundational boundaries

**Purpose**: Define strict domain, demonstration-provider, localization, testing, and design-system foundations shared by every story.

- [x] T004 Write a failing domain-boundary test for unique corpus ownership, HTTPS external links, and cross-corpus citation rejection in `apps/web/src/research/domain/researchCatalog.test.ts` per FR-003, FR-010, FR-014, and FR-025 (RED)
- [x] T005 Implement discriminated research models, catalog validation, citation resolution, and the production demonstration provider in `apps/web/src/research/domain/models.ts`, `apps/web/src/research/domain/researchCatalog.ts`, and `apps/web/src/research/demonstration/createDemonstrationCatalog.ts` per FR-001, FR-003, FR-010, FR-014, and FR-025 (GREEN)
- [x] T006 Write a failing locale parity and English-default test in `apps/web/src/i18n/config.test.ts` per FR-018 and FR-019 (RED)
- [x] T007 Implement typed English and Portuguese localization resources and provider configuration in `apps/web/src/i18n/config.ts`, `apps/web/src/i18n/en/translation.ts`, and `apps/web/src/i18n/pt/translation.ts` per FR-018 and FR-019 (GREEN)
- [x] T008 Create shared semantic presentation primitives and an axe-enabled render harness in `apps/web/src/components/EmptyState.tsx`, `apps/web/src/components/LanguageControl.tsx`, `apps/web/src/test/render.tsx`, and `apps/web/src/test/setup.ts` per FR-018 and FR-020
- [x] T009 Create production-owned editorial design tokens, font entry points, global reset, focus treatment, reduced-motion handling, and application shell styling in `apps/web/src/styles/tokens.css`, `apps/web/src/styles/global.css`, and `apps/web/src/app/AppShell.tsx` per FR-020-FR-023

**Checkpoint**: Domain invariants and localization fail safely through public tests; shared presentation has no prototype dependency.

---

## Phase 3: User Story 1 - Select a legal corpus (Priority: P1) MVP

**Goal**: Present two isolated corpus choices and navigate valid or unknown corpus routes.

**Independent Test**: Open the production catalog, identify both corpus languages, enter either workspace, and recover from an unknown corpus address.

- [x] T010 [US1] Write one failing catalog journey test for two corpus summaries, localized metadata, and valid navigation in `apps/web/src/features/catalog/CorpusCatalogPage.test.tsx` per FR-001, FR-002, FR-003, and FR-018 (RED)
- [x] T011 [US1] Implement corpus cards and catalog composition in `apps/web/src/features/catalog/CorpusCard.tsx`, `apps/web/src/features/catalog/CorpusCatalogPage.tsx`, and `apps/web/src/features/catalog/catalog.css` per FR-001-FR-003, FR-021, and FR-022 (GREEN)
- [x] T012 [US1] Write one failing unknown-route recovery test in `apps/web/src/features/workspace/UnknownCorpusPage.test.tsx` per FR-004 and FR-018 (RED)
- [x] T013 [US1] Implement lazy production routes and unknown-corpus recovery in `apps/web/src/app/routes.tsx` and `apps/web/src/features/workspace/UnknownCorpusPage.tsx` per FR-003, FR-004, and FR-027 (GREEN)
- [x] T014 [US1] Refactor the catalog slice while green in `apps/web/src/features/catalog/CorpusCatalogPage.tsx` and run `apps/web/src/features/catalog/CorpusCatalogPage.test.tsx` with its accessibility scan per FR-020, FR-021, SC-001, SC-002, and SC-003

**Checkpoint**: User Story 1 is independently runnable and testable as the production MVP.

---

## Phase 4: User Story 2 - Browse and inspect sources (Priority: P1)

**Goal**: Provide keyboard-accessible source hierarchy and PDF, external-link, and unavailable viewers.

**Independent Test**: Enter a corpus, navigate the tree with pointer and keyboard, open each source kind, and retain safe access when preview is unavailable.

- [x] T015 [US2] Write one failing keyboard and selection behavior test for the ARIA source tree in `apps/web/src/features/workspace/SourceTree.test.tsx` per FR-005, FR-006, FR-007, FR-008, and FR-020 (RED)
- [x] T016 [US2] Implement the accessible source tree and source grouping in `apps/web/src/features/workspace/SourceTree.tsx` per FR-005-FR-008 and FR-020 (GREEN)
- [x] T017 [US2] Write one failing viewer behavior test covering PDF location, external preview, and unavailable preview in `apps/web/src/features/workspace/SourceViewer.test.tsx` per FR-009-FR-011 (RED)
- [x] T018 [US2] Implement source mode, PDF navigation, and safe external-link presentation in `apps/web/src/features/workspace/SourceViewer.tsx`, `apps/web/src/features/workspace/PdfSourceViewer.tsx`, and `apps/web/src/features/workspace/ExternalSourceViewer.tsx` per FR-008-FR-011 and FR-020 (GREEN)
- [x] T019 [US2] Write one failing workspace-controller preservation test for source selection, locations, and mode transitions in `apps/web/src/features/workspace/useWorkspaceController.test.tsx` per FR-008 and FR-011-FR-012 (RED)
- [x] T020 [US2] Implement the focused workspace controller and page composition in `apps/web/src/features/workspace/useWorkspaceController.ts` and `apps/web/src/features/workspace/CorpusWorkspacePage.tsx` per FR-003, FR-005, FR-008, and FR-011-FR-012 (GREEN)
- [x] T021 [US2] Refactor the source slice while green in `apps/web/src/features/workspace/SourceTree.tsx` and `apps/web/src/features/workspace/SourceViewer.tsx`, then run focused keyboard and axe verification in `apps/web/src/features/workspace/SourceTree.test.tsx` and `apps/web/src/features/workspace/SourceViewer.test.tsx` per FR-020, SC-001, and SC-002

**Checkpoint**: User Story 2 is independently testable with every source state and no chat dependency.

---

## Phase 5: User Story 3 - Research through chat and citations (Priority: P1)

**Goal**: Provide structured simulated chat, citation navigation, abstention, failure recovery, and preserved workspace context.

**Independent Test**: Submit supported, unsupported, and failure questions; open a citation; and alternate between Chat and Source without state loss.

- [x] T022 [US3] Write one failing prepared-response engine test for answered, abstained, failed, and active-corpus citation outcomes in `apps/web/src/research/demonstration/preparedResponseEngine.test.ts` per FR-013, FR-014, FR-015, FR-016, FR-017, FR-025, and FR-026 (RED)
- [x] T023 [US3] Implement the deterministic prepared-response engine in `apps/web/src/research/demonstration/preparedResponseEngine.ts` per FR-013-FR-017 and FR-025-FR-026 (GREEN)
- [x] T024 [US3] Write one failing user-visible chat test for structured citations, simulation labels, abstention, failure, and retry in `apps/web/src/features/workspace/ResearchChat.test.tsx` per FR-013-FR-017, FR-020, and FR-026 (RED)
- [x] T025 [US3] Implement the assistant-ui adapter and research chat in `apps/web/src/research/adapters/createAssistantAdapter.ts` and `apps/web/src/features/workspace/ResearchChat.tsx` per FR-013-FR-017, FR-025, and FR-026 (GREEN)
- [x] T026 [US3] Write one failing integrated citation and mode-preservation test in `apps/web/src/features/workspace/CorpusWorkspacePage.test.tsx` per FR-012 and FR-014-FR-015 (RED)
- [x] T027 [US3] Integrate the persistent Chat and Source surfaces and citation resolution in `apps/web/src/features/workspace/CorpusWorkspacePage.tsx` and `apps/web/src/features/workspace/WorkspaceModeSelector.tsx` per FR-011-FR-017 (GREEN)
- [x] T028 [US3] Refactor the chat slice while green in `apps/web/src/features/workspace/ResearchChat.tsx` and `apps/web/src/research/adapters/createAssistantAdapter.ts`, then run focused response, citation, failure, and accessibility tests in `apps/web/src/features/workspace/ResearchChat.test.tsx` per FR-020, FR-026, SC-001, SC-002, SC-003, SC-004, and SC-005

**Checkpoint**: All P1 research journeys are runnable without backend, model, database, or graph services.

---

## Phase 6: User Story 4 - Polished bilingual interface (Priority: P2)

**Goal**: Complete bilingual state preservation, responsive styling, keyboard journeys, and visual evidence.

**Independent Test**: Complete every journey in both interface languages and approved viewports using keyboard and pointer.

- [x] T029 [US4] Write one failing application test for English default, Portuguese switching, preserved workspace state, and untranslated legal content in `apps/web/src/app/App.test.tsx` per FR-018-FR-019 (RED)
- [x] T030 [US4] Complete bilingual application composition and state preservation in `apps/web/src/app/App.tsx`, `apps/web/src/app/AppShell.tsx`, `apps/web/src/features/catalog/CorpusCatalogPage.tsx`, and `apps/web/src/features/workspace/CorpusWorkspacePage.tsx` per FR-018-FR-019 (GREEN)
- [x] T031 [US4] Write failing browser journeys for keyboard use, locale switching, citations, unknown routes, and both viewports in `apps/web/tests/e2e/research-workspace.spec.ts` per FR-020-FR-022 and SC-001-SC-005 (RED)
- [x] T032 [US4] Complete production workspace and responsive styling in `apps/web/src/features/workspace/workspace.css`, `apps/web/src/features/catalog/catalog.css`, and `apps/web/src/styles/global.css` per FR-020-FR-022 (GREEN)
- [x] T033 [US4] Capture and review deterministic catalog and workspace snapshots in `apps/web/tests/e2e/research-workspace.spec.ts-snapshots/` per FR-021-FR-022 and SC-007

**Checkpoint**: User Story 4 passes in both languages and both approved desktop viewports.

---

## Phase 7: Production quality and handoff

**Purpose**: Enforce performance, complete module verification, and align documentation with delivered behavior.

- [x] T034 Write a failing bundle-budget integration check and implement compressed initial-entry enforcement in `apps/web/scripts/check-bundle-budget.mjs` and `apps/web/package.json` per FR-027 and SC-006 (RED-GREEN)
- [x] T035 Update `apps/web/README.md`, `README.md`, and `specs/002-production-web-experience/quickstart.md` with production-client commands, deterministic-data limitations, and zero-service prerequisites per FR-024 and SC-008
- [x] T036 Run format, lint, strict type checking, unit and component tests, production build, bundle budget, repository-language validation, and `git diff --check`; record evidence in `specs/002-production-web-experience/verification.md` per FR-028 and SC-001-SC-008
- [x] T037 Run the feature quickstart and Playwright journeys from the production module, review the complete diff for prototype imports and boundary violations, and record final evidence in `specs/002-production-web-experience/verification.md` per FR-020-FR-028

---

## Dependencies and execution order

- Phase 1 establishes the module and blocks every later phase.
- Phase 2 establishes domain and localization contracts and blocks user stories.
- User Story 1 establishes routes used by all later browser journeys.
- User Story 2 establishes workspace and source behavior used by citations.
- User Story 3 adds chat without changing source ownership.
- User Story 4 validates the complete bilingual presentation.
- Production quality runs only after all user stories are green.

## TDD cycle rule

For each RED task:

1. Add only the named observable behavior test.
2. Execute the narrow test and record that it fails for missing behavior, not setup failure.
3. Start the paired GREEN task and implement only enough behavior to pass.
4. Execute the focused test again before adding the next behavior.
5. Refactor only while all focused tests are green.

## Parallel opportunities

- T002 can proceed independently after T001 determines the package scripts.
- T006-T007 can proceed independently from the domain cycle T004-T005.
- Visual tokens in T009 do not alter domain behavior, but must exist before catalog polish.
- Documentation in T035 can start after behavior stabilizes while T034 owns a separate script.

## Implementation strategy

The MVP checkpoint is User Story 1 after T014. Continue sequentially because source behavior feeds citation behavior and the same workspace files are involved. Stop at each checkpoint if any focused test, accessibility scan, or type check is red.

## Phase 8: Convergence

- [x] T038 Add a failing route-isolation test and remount workspace state when the corpus identifier changes in `apps/web/src/features/workspace/CorpusWorkspacePage.test.tsx` and `apps/web/src/features/workspace/CorpusWorkspacePage.tsx` per FR-003 (partial)
- [x] T039 Add failing source-tree availability assertions and expose localized available or unavailable external-source state without relying on color in `apps/web/src/features/workspace/SourceTree.test.tsx`, `apps/web/src/features/workspace/SourceTree.tsx`, and locale resources per FR-007 (partial)
- [x] T040 Add a failing recovery journey and implement an actionable retry for failed prepared responses without duplicating completed messages in `apps/web/src/features/workspace/ResearchChat.test.tsx` and `apps/web/src/features/workspace/ResearchChat.tsx` per FR-017 and US3/AC6 (partial)
- [x] T041 Add failing ARIA tree keyboard tests and implement roving focus plus Up, Down, Left, Right, Home, and End behavior in `apps/web/src/features/workspace/SourceTree.test.tsx` and `apps/web/src/features/workspace/SourceTree.tsx` per FR-020, SC-002, and US4/AC4 (partial)
- [x] T042 Add a failing unavailable-citation test and present localized recoverable feedback while preserving conversation state in `apps/web/src/features/workspace/CorpusWorkspacePage.test.tsx`, `apps/web/src/features/workspace/CorpusWorkspacePage.tsx`, and locale resources per the unavailable-citation edge case (partial)
