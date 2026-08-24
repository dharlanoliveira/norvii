# Tasks: Citation Navigation and Inspection

**Input**: Design documents from `/specs/006-citation-inspection/`

**Prerequisites**: [spec.md](spec.md), [plan.md](plan.md), [research.md](research.md),
[data-model.md](data-model.md), [citation-inspection.md](contracts/citation-inspection.md)

**Tests**: Tests are required for changed behavior, failure paths, cross-language contracts, and
accessible interactions.

## Phase 1: Contract and Test Foundation

**Purpose**: Establish the additive cross-language contract before behavior changes.

- [X] T001 [P] Update `specs/005-grounded-rag-chat/contracts/chat-stream.schema.json` and `specs/005-grounded-rag-chat/contracts/chat-http.md` with Feature 006 reference and terminal-inspection fields per FR-005 to FR-008 and the Feature 006 contract.
- [X] T002 [P] Add failing stream-parser tests for immutable reference identity, nullable measurements, malformed values, and terminal inspection payloads in `apps/web/src/api/chat.test.ts` per FR-005 to FR-008 and FR-012.
- [X] T003 [P] Add failing Python graph/provider/transport tests for retrieval distance, measured timings, nullable provider usage, and inspection event serialization in `apps/agent/tests/unit/test_grounded_chat.py`, `apps/agent/tests/unit/test_chat_provider.py`, and `apps/agent/tests/unit/test_transport_server.py` per FR-006 to FR-008 and FR-012.
- [X] T004 [P] Add failing Go agent-client and chat-handler tests for inspection relay, total request timing, and safe non-completed terminals in `apps/api/internal/chat/agent/client_test.go` and `apps/api/internal/chat/http/handler_test.go` per FR-005 to FR-008 and FR-013.

**Checkpoint**: The future payload is defined and protected at every public boundary.

---

## Phase 2: Foundational Immutable Evidence and Execution Data

**Purpose**: Provide correct immutable evidence and answer-specific inspection data for all user stories.

- [X] T005 Add `GetByID` corpus/source/document-version read behavior and repository tests in `apps/api/internal/document/postgres/repository.go` and `apps/api/internal/document/postgres/repository_test.go` per FR-002, FR-004, and FR-014.
- [X] T006 Add and test the immutable document-version HTTP route in `apps/api/internal/document/http/handler.go` and `apps/api/internal/document/http/handler_test.go` per FR-002, FR-004, and FR-014.
- [X] T007 Extend the Python evidence and graph result types, measure retrieval and generation at their owner operations, and add vector distance/source identity in `apps/agent/src/norvii_agent/graph/grounded_chat.py` and `apps/agent/src/norvii_agent/retrieval/postgres.py` per FR-005 to FR-008 and FR-014.
- [X] T008 Extend `apps/agent/src/norvii_agent/providers/chat.py` and `apps/agent/src/norvii_agent/transport/server.py` to propagate provider-reported usage, nullable measurements, and safe inspection terminal payloads per FR-007, FR-008, FR-012, and FR-013.
- [X] T009 Extend Go chat domain, agent adapter, and SSE handler in `apps/api/internal/chat/domain/service.go`, `apps/api/internal/chat/agent/client.go`, and `apps/api/internal/chat/http/handler.go` to relay inspection data and measure public total duration per FR-005 to FR-008 and FR-013.

**Checkpoint**: The API can return answer-specific immutable evidence, exact document versions, and honest execution measurements.

---

## Phase 3: User Story 1 - Open Cited Legal Evidence (Priority: P1)

**Goal**: A citation opens the exact immutable legal passage within the active corpus while the answer remains available.

**Independent Test**: Complete an answer with direct and nested citations; select each and verify the matching immutable document, visible legal context, and highlighted range. Verify foreign, malformed, and unavailable references do not load substitute content.

- [X] T010 [P] [US1] Add failing provider and workspace tests for explicit document-version reads, active-corpus validation, unavailable citation feedback, and preserved chat state in `apps/web/src/api/researchProvider.test.ts` and `apps/web/src/features/workspace/CorpusWorkspacePage.test.tsx` per FR-001 to FR-004 and FR-011.
- [X] T011 [P] [US1] Add failing reader tests for nested-unit parent resolution, offset-bound validation, exact highlight, and scrolling the cited span into view in `apps/web/src/features/workspace/LegalDocumentReader.test.tsx` per FR-002, FR-004, and FR-010.
- [X] T012 [US1] Add a typed immutable-document request to `apps/web/src/api/researchProvider.ts`, `apps/web/src/api/contract.ts`, and `apps/web/src/research/domain/authoritative.ts` per FR-002, FR-004, and FR-014.
- [X] T013 [US1] Centralize direct and inspection citation resolution with corpus, source, document-version, and offset validation plus localized unavailable feedback in `apps/web/src/features/workspace/CorpusWorkspacePage.tsx` per FR-001 to FR-004, FR-009 to FR-011, and FR-014.
- [X] T014 [US1] Render and focus the exact cited span while preserving the existing visible article context in `apps/web/src/features/workspace/LegalDocumentReader.tsx` and related workspace styles per FR-002, FR-004, and FR-010.

**Checkpoint**: Direct citation navigation is immutable, accessible, visibly precise, and fails closed.

---

## Phase 4: User Story 2 - Inspect Answer Evidence and Execution (Priority: P2)

**Goal**: A completed answer offers an accessible, localized inspector with its immutable evidence and available measurements.

**Independent Test**: Complete deterministic answers with available and unavailable measurements; open the inspector and verify ordered evidence, strategy, metrics, language, and no sensitive data.

- [X] T015 [P] [US2] Add failing client parser and runtime tests for per-message inspection snapshots, nullable values, and terminal outcome distinctions in `apps/web/src/api/chat.test.ts` and `apps/web/src/features/workspace/useNorviiChatRuntime.test.tsx` per FR-005 to FR-008, FR-012, and FR-013.
- [X] T016 [P] [US2] Add failing accessible component tests for the answer inspector, localized unavailable labels, keyboard disclosure, ordered evidence, and axe checks in `apps/web/src/features/workspace/ResearchChat.test.tsx` per FR-005 to FR-008 and FR-010 to FR-013.
- [X] T017 [US2] Extend the validated TypeScript chat contract and retain terminal inspection snapshots by assistant message in `apps/web/src/api/chat.ts` and `apps/web/src/features/workspace/useNorviiChatRuntime.ts` per FR-005 to FR-008, FR-012, and FR-013.
- [X] T018 [US2] Render the Assistant UI-backed answer inspector and complete English/Portuguese localization in `apps/web/src/features/workspace/ResearchChat.tsx`, `apps/web/src/features/workspace/workspace.css`, and `apps/web/src/locales/{en,pt}.json` per FR-005 to FR-008 and FR-010 to FR-013.

**Checkpoint**: Each completed answer has a safe, readable, localized, keyboard-accessible inspection disclosure.

---

## Phase 5: User Story 3 - Compare Citations from One Answer (Priority: P3)

**Goal**: Inspection evidence actions use exactly the same immutable navigation behavior as inline citations.

**Independent Test**: Open one completed answer's inspector, select several evidence items, return to Chat after each one, and verify the original inspection remains tied to that answer.

- [X] T019 [US3] Connect inspector evidence actions to the shared citation resolver and add multi-answer persistence tests in `apps/web/src/features/workspace/ResearchChat.tsx`, `apps/web/src/features/workspace/CorpusWorkspacePage.tsx`, and their tests per FR-003, FR-009, and FR-014.

**Checkpoint**: Evidence comparison is possible without a new query or a changing citation identity.

---

## Phase 6: Cross-Cutting Verification

**Purpose**: Verify the feature end-to-end and preserve repository quality gates.

- [X] T020 Add Playwright citation and inspection journeys, including immutable reprocessing and keyboard paths, in `apps/web/tests/e2e/citation-inspection.spec.ts` per SC-001 to SC-007.
- [X] T021 Run the commands in `specs/006-citation-inspection/quickstart.md`, update that guide only if execution reveals an incorrect command, and document any deliberate test-environment limitation per SC-001 to SC-007.
- [X] T022 Run `.github/scripts/validate_contracts.py`, `.github/scripts/validate_repository_language.py`, and `git diff --check`; resolve all output before handoff per Constitution IV, V, and VIII.

## Dependencies and Execution Order

- Phase 1 defines tests and contract shape before code.
- Phase 2 is required before either UI story because it produces immutable version reads and inspection payloads.
- US1 (Phase 3) is the MVP and must complete before US3 because both direct and inspector evidence use the same resolver.
- US2 (Phase 4) consumes the Phase 2 stream and can be developed after the foundational phase; it must complete before US3.
- US3 (Phase 5) joins the completed P1 resolver and P2 inspector.
- Phase 6 runs after all three stories.

## Parallel Opportunities

- T001 to T004 can be developed in parallel because they define tests and documents in distinct modules.
- T005 and T007 can proceed in parallel after their relevant tests exist; T006 depends on T005, T008 depends on T007, and T009 depends on T008.
- Within each user story, tasks marked `[P]` can run in parallel. Implementation tasks touching the same component or contract remain sequential.

## Implementation Strategy

1. Complete the contract/test and foundational phases until a terminal answer contains correct immutable evidence and truthful inspection metadata.
2. Deliver and test direct immutable citation navigation as the independently useful P1 slice.
3. Add the per-answer inspector and its localization/accessibility as P2.
4. Reuse the same resolver for P3 evidence comparison, then run browser and full quality checks.
