# Tasks: Grounded RAG Chat

**Input**: Design documents from `specs/005-grounded-rag-chat/`

**Prerequisites**: `plan.md`, `spec.md`, `research.md`, `data-model.md`, `contracts/`, `quickstart.md`

**TDD rule**: Every RED task must be observed failing for the intended missing behavior before
its GREEN implementation task begins. Refactor only while focused and module suites remain green.

## Phase 1: Setup

**Purpose**: Establish feature-local contracts and bounded provider configuration without changing
the approved prototype.

- [ ] T001 [P] Add bounded grounded-chat, retrieval, embedding, and model-provider configuration with safe defaults in `apps/api/internal/platform/config/`, `apps/agent/src/norvii_agent/config/`, `apps/ingestion/src/norvii_ingestion/config/`, and `infra/.env.example` per FR-001, FR-016-FR-018.
- [ ] T002 [P] Add feature-local stream fixtures and valid/invalid payload examples under `specs/005-grounded-rag-chat/contracts/fixtures/` per FR-013-FR-018.
- [ ] T003 [P] Add contract validation coverage for `specs/005-grounded-rag-chat/contracts/chat-stream.schema.json` in `.github/scripts/tests/test_validate_contracts.py` per FR-013-FR-018.

## Phase 2: Foundational

**Purpose**: Build the immutable retrieval, provider, stream, and safety boundaries required by
all user stories. No user-story implementation begins until this phase is green.

- [ ] T004 [P] Write a failing PostgreSQL migration test for retrieval chunks, fixed-dimension embeddings, document-version ownership, latest-version filtering, idempotency, and corpus isolation in `apps/api/tests/integration/grounded_rag_schema_test.go` per FR-003, FR-006, FR-018.
- [ ] T005 Implement migration `apps/api/migrations/005_grounded_rag.sql`, embed it through `apps/api/migrations/embed.go`, and make T004 pass without changing Feature 004 tables per FR-003, FR-006, FR-018.
- [x] T006 [P] Write failing Python unit tests for legal-aware chunk boundaries, article context for nested paragraphs/items, contiguous offsets, hash identity, and bounded chunk size in `apps/ingestion/tests/unit/enrichment/test_chunking.py` per FR-003, FR-005-FR-006, FR-018.
- [x] T007 Implement immutable chunking domain objects and legal-aware chunker in `apps/ingestion/src/norvii_ingestion/enrichment/chunking/` until T006 passes per FR-003, FR-005-FR-006.
- [x] T008 [P] Write failing provider contract tests for deterministic embedding dimensions, model version, malformed response, and safe error mapping in `apps/ingestion/tests/unit/enrichment/test_embedding_provider.py` and `apps/agent/tests/unit/test_embedding_provider.py` per FR-013, FR-016-FR-018.
- [x] T009 Implement consumer-owned embedding and chat-model ports plus provider-neutral configuration and fake adapters in `apps/ingestion/src/norvii_ingestion/enrichment/embedding/` and `apps/agent/src/norvii_agent/providers/` until T008 passes per FR-004, FR-012-FR-018.
- [x] T010 [P] Write failing Python agent retrieval tests for active-corpus filtering, latest published document filtering, ranked top-eight results, and immutable evidence projection in `apps/agent/tests/unit/test_postgres_retrieval.py` per FR-002-FR-006, FR-018.
- [x] T011 Implement the agent retrieval port and PostgreSQL adapter in `apps/agent/src/norvii_agent/retrieval/` until T010 passes per FR-002-FR-006, FR-018.
- [ ] T012 [P] Write failing stream contract tests for event ordering, exactly one terminal event, malformed provider events, and safe public errors in `apps/api/tests/contract/grounded_chat_stream_test.go` and `apps/web/src/api/chat.test.ts` per FR-007-FR-018.
- [ ] T013 Implement the feature-local stream event encoder, decoder, and terminal-state validator in `apps/api/internal/platform/streaming/` and `apps/web/src/api/chat.ts` until T012 passes per FR-007-FR-013.

**Checkpoint**: Migration, chunking, retrieval, provider ports, and stream contracts are green;
user-story work can proceed.

## Phase 3: User Story 1 - Ask a Corpus-Grounded Question (Priority: P1) MVP

**Goal**: Ask a question in one active corpus and receive a bounded streamed grounded answer or
an explicit terminal outcome.

**Independent Test**: Use a deterministic published document and fake providers, submit a
supported question, and verify corpus-scoped evidence, answer deltas, citations, and completion.

### Tests for User Story 1

- [ ] T014 [P] [US1] Write failing Python enrichment tests for embedding publication, unchanged-content idempotency, pipeline-versioned backfill, changed-document versioning, and failed enrichment preservation in `apps/ingestion/tests/integration/test_grounded_rag_enrichment.py` per FR-003, FR-006, FR-018, FR-021, SC-010.
- [ ] T015 [P] [US1] Write failing Python agent graph tests for valid questions, active-corpus ownership, evidence threshold, citation-marker validation, and provider cancellation in `apps/agent/tests/unit/test_grounded_chat.py` per FR-001-FR-008, FR-011-FR-018.
- [ ] T016 [P] [US1] Write failing Go HTTP tests for request validation, SSE headers, event order, terminal outcomes, disconnect cancellation, and safe errors in `apps/api/internal/chat/http/handler_test.go` per FR-001, FR-007-FR-008, FR-013, FR-017.
- [ ] T017 [P] [US1] Write failing React tests for question submission, pending/streaming/completed states, cancellation, terminal failure, and no fixture fallback in `apps/web/src/features/workspace/ResearchChat.test.tsx` and `apps/web/src/features/workspace/CorpusWorkspacePage.test.tsx` per FR-001, FR-007-FR-008, FR-013-FR-014.
- [ ] T018 [P] [US1] Write failing end-to-end coverage for supported grounded chat using deterministic provider fixtures in `apps/web/tests/e2e/grounded-rag-chat.spec.ts` per SC-001, SC-003, SC-006.

### Implementation for User Story 1

- [ ] T019 [US1] Implement Python enrichment orchestration and atomic retrieval-chunk publication in `apps/ingestion/src/norvii_ingestion/enrichment/` and `apps/ingestion/src/norvii_ingestion/publication/postgres/` until T014 passes per FR-003, FR-005-FR-006, FR-016-FR-018, FR-021.
- [ ] T020 [US1] Implement LangGraph retrieval orchestration, active-corpus checks, latest-version filters, bounded context assembly, and cosine-ranked evidence from ready embeddings in `apps/agent/src/norvii_agent/{graph,retrieval}/` until T015 passes per FR-002-FR-006, FR-018, FR-021.
- [ ] T021 [US1] Implement grounded-answer validation, evidence-only prompt construction, citation-marker checks, abstention boundary, and cancellation propagation in `apps/agent/src/norvii_agent/graph/` until T015 passes per FR-004, FR-011-FR-012, FR-018.
- [x] T022 [US1] Implement the versioned chat stream HTTP handler and route wiring in `apps/api/internal/chat/http/`, `apps/api/internal/platform/httpserver/`, and `apps/api/cmd/server/` until T016 passes per FR-001-FR-003, FR-007-FR-008, FR-013, FR-017.
- [ ] T023 [US1] Implement the configured OpenAI-compatible embedding and chat adapters with timeout, cancellation, response-shape validation, and redacted diagnostics in `apps/ingestion/src/norvii_ingestion/enrichment/embedding/` and `apps/agent/src/norvii_agent/providers/` until T008 and T016 pass per FR-012-FR-018.
- [x] T024 [US1] Implement the React chat stream adapter, assistant-ui conversation runtime, request cancellation, localized composer and starter questions, semantic Markdown answer rendering, and technical disclaimer in `apps/web/src/api/chat.ts`, `apps/web/src/features/workspace/ResearchChat.tsx`, `apps/web/src/features/workspace/useNorviiChatRuntime.ts`, `apps/web/src/features/workspace/AssistantMarkdown.tsx`, and `apps/web/src/features/workspace/workspace.css` until T017 passes per FR-001, FR-007-FR-008, FR-013-FR-015, FR-022-FR-023.
- [ ] T025 [US1] Add English and Portuguese grounded-chat resources with parity tests in `apps/web/src/i18n/en/translation.ts`, `apps/web/src/i18n/pt/translation.ts`, and `apps/web/src/i18n/config.test.ts` per FR-014-FR-015, SC-009.
- [ ] T026 [US1] Make T018 pass with deterministic provider fixtures and document the supported local journey in `specs/005-grounded-rag-chat/quickstart.md` per SC-001, SC-003, SC-006.

**Checkpoint**: The P1 MVP answers supported questions from one corpus with bounded streaming,
resolvable evidence, and no simulated fallback.

## Phase 4: User Story 2 - Inspect and Navigate Supporting Evidence (Priority: P2)

**Goal**: Open every answer reference in the existing source and document viewer while keeping the
conversation visible.

**Independent Test**: Select each citation from a deterministic completed answer and verify that
the matching corpus-owned source and immutable document location are selected and visible.

- [ ] T027 [P] [US2] Write failing evidence-reference and workspace navigation tests in `apps/web/src/features/workspace/EvidenceReference.test.tsx`, `apps/web/src/features/workspace/LegalDocumentReader.test.tsx`, and `apps/web/src/features/workspace/CorpusWorkspacePage.test.tsx` per FR-005-FR-010, SC-002, SC-005.
- [ ] T028 [US2] Implement typed evidence-reference rendering, citation marker interaction, and accessible focus behavior in `apps/web/src/features/workspace/EvidenceReference.tsx` and `apps/web/src/features/workspace/ResearchChatMessage.tsx` until T027 passes per FR-005, FR-009, FR-014.
- [ ] T029 [US2] Implement immutable document/unit resolution and scroll targeting for evidence references in `apps/web/src/features/workspace/CorpusWorkspacePage.tsx`, `apps/web/src/features/workspace/LegalDocumentReader.tsx`, and `apps/web/src/research/domain/` until T027 passes per FR-005-FR-010.
- [ ] T030 [P] [US2] Write failing Go and React tests for unresolved, superseded, foreign-corpus, and nested paragraph/item references in `apps/api/internal/document/http/handler_test.go` and `apps/web/src/features/workspace/EvidenceReference.test.tsx` per FR-006, FR-009-FR-010, SC-002-SC-003.
- [ ] T031 [US2] Implement fail-closed evidence resolution and localized unavailable-evidence behavior in `apps/api/internal/document/` and `apps/web/src/features/workspace/` until T030 passes per FR-006, FR-009-FR-010.
- [ ] T032 [US2] Extend deterministic Playwright coverage for multiple citations, keyboard navigation, and language parity in `apps/web/tests/e2e/grounded-rag-chat.spec.ts` per SC-002, SC-005, SC-009.

**Checkpoint**: Every citation is inspectable and navigates to the exact preserved legal context.

## Phase 5: User Story 3 - Receive an Honest Abstention (Priority: P3)

**Goal**: Refuse unsupported questions safely and distinguish insufficient evidence from provider
or transport failures.

**Independent Test**: Submit unsupported, unavailable-corpus, injection-like, timeout, malformed
stream, and cancellation scenarios using deterministic fakes and verify the correct localized
terminal state.

- [ ] T033 [P] [US3] Write failing Python agent abstention and prompt-boundary tests for low evidence, no ready sources, instruction-like retrieved text, missing citations, and provider timeout in `apps/agent/tests/unit/test_abstention.py` per FR-004, FR-011-FR-013, SC-004, SC-007.
- [ ] T034 [P] [US3] Write failing React tests for localized abstention, unavailable evidence, provider failure, incomplete stream, and retry behavior in `apps/web/src/features/workspace/ResearchChat.test.tsx` per FR-007, FR-011, FR-013-FR-015, SC-007.
- [ ] T035 [US3] Implement scope-limited assistant generation for missing evidence, prompt injection boundary, and terminal validation in `apps/agent/src/norvii_agent/graph/` until T033 passes per FR-004, FR-011-FR-013.
- [ ] T036 [US3] Implement localized abstention, retry, unavailable-provider, incomplete-stream, and technical-disclaimer states in `apps/web/src/features/workspace/ResearchChat.tsx`, `apps/web/src/features/workspace/ResearchChatMessage.tsx`, and `apps/web/src/i18n/` until T034 passes per FR-007, FR-011, FR-013-FR-015.
- [ ] T037 [US3] Extend Playwright coverage for unsupported questions, provider absence, cancellation, and no-ready-source states in `apps/web/tests/e2e/grounded-rag-chat.spec.ts` per SC-004, SC-006-SC-007.

**Checkpoint**: Unsupported or unsafe questions never become completed answers or fabricated
citations.

## Phase 6: Polish and Cross-Cutting Concerns

- [ ] T038 [P] Add content-safe retrieval and generation telemetry, bounded counters, and redaction tests in `apps/api/internal/chat/`, `apps/api/internal/platform/`, `apps/ingestion/src/norvii_ingestion/`, and `.github/scripts/tests/` per FR-016-FR-017, SC-008.
- [ ] T039 [P] Add provider configuration, optional enrichment verification, agent health diagnostics, and separate `.log/` output guidance in `infra/scripts/manage-local-environment.py`, `Makefile`, `.agents/skills/bootstrap-norvii/SKILL.md`, and `docs/operations/local-environment.md` per FR-016-FR-018, SC-001, SC-008.
- [ ] T040 [P] Validate the feature-local contract, repository language, migrations, and stream fixtures through `.github/scripts/validate_contracts.py`, `.github/scripts/validate_repository_language.py`, and `git diff --check` per FR-013-FR-018.
- [ ] T041 Run every affected Go, Python, and React quality gate plus the deterministic quickstart twice and record measured results in `specs/005-grounded-rag-chat/quickstart.md` per SC-001-SC-009.
- [ ] T042 Promote the stabilized chat contract to `contracts/` only after provider/consumer compatibility tests are green, update `contracts/README.md`, and record the promotion decision in the feature docs per FR-013-FR-018.

## Phase 7: Reconciliation

The implementation-presence audit recorded in [reconciliation.md](reconciliation.md) found that
the current codebase already contains substantial delivery for the originally unchecked tasks.
The historical checklist remains unchanged because several items do not record their required
acceptance evidence under the originally named test path. Do not reimplement a mapped capability;
complete the consolidated tasks below instead.

- [X] T043 Add feature-owned valid and invalid chat-stream fixtures, then extend contract validation to load the schema and fixtures per T002, T003, and FR-013-FR-018 (partial).
- [X] T044 Add deterministic service-backed grounded-chat coverage for corpus and snapshot isolation, terminal-event ordering, citation navigation, cancellation, and abstention; record the two-run quickstart evidence per T004, T012, T018, T26, T32, T37, T40, T41, and SC-001-SC-009 (partial).
- [X] T045 Verify content-safe chat telemetry, provider diagnostics, and local health/configuration guidance against the implementation; add missing redaction or bounded-counter coverage and update operations documentation per T038 and T039 (partial).
- [X] T046 Decide and document whether the stabilized chat contract is promoted to `contracts/`; if promoted, add producer/consumer compatibility coverage and update the contracts index per T042 and FR-013-FR-018 (missing).

## Dependencies and Execution Order

- **Phase 1** has no feature dependencies and establishes local contracts/configuration.
- **Phase 2** blocks all user stories because migration, chunking, providers, retrieval, and
  stream semantics are shared boundaries.
- **US1 (P1)** depends on Phase 2 and is the MVP. It must be demonstrated before expanding UI
  navigation and failure-state breadth.
- **US2 (P2)** depends on US1's evidence contract and existing workspace navigation.
- **US3 (P3)** depends on US1's terminal state machine and US2's evidence identity.
- **Phase 6** follows the selected user stories; T042 is intentionally last because promotion is
  a compatibility decision, not a prerequisite for the feature-local implementation.

### Parallel Opportunities

- T001-T003 can run in parallel.
- T004, T006, T008, T010, and T012 can run in parallel before their implementations.
- Within US1, Python enrichment, Python agent retrieval, Go facade stream contract, and React behavior tests can be
  written in parallel; implementation remains red-green per boundary.
- Within US2, citation rendering and backend resolution tests can run in parallel.
- Within US3, backend abstention and client terminal-state tests can run in parallel.

## Implementation Strategy

### MVP First (User Story 1 Only)

1. Complete Setup and Foundational phases with deterministic fake providers.
2. Complete US1 and stop at its checkpoint.
3. Run the supported-question journey with a seeded published document and verify evidence IDs.
4. Obtain the implementation approval gate before production coding if the feature plan changes.

### Incremental Delivery

1. Add US2 citation navigation without changing retrieval semantics.
2. Add US3 abstention and safety breadth without weakening the P1 completion invariant.
3. Complete telemetry, local configuration, quality gates, and contract promotion.

## Notes

- All test tasks must be observed failing before their corresponding implementation task.
- All provider calls in CI use deterministic fakes; live credentials are optional for local demos.
- Do not add Neo4j, GraphRAG, MCP, skills, persistent conversations, or evaluation dashboards to
  this feature.
