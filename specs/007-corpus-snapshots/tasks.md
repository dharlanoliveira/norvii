# Tasks: Bilingual Corpus Snapshots

**Input**: Design documents from `/specs/007-corpus-snapshots/`

**Prerequisites**: [plan.md](plan.md), [spec.md](spec.md), [research.md](research.md),
[data-model.md](data-model.md), [snapshot contract](contracts/snapshot-http.md), and
[quickstart.md](quickstart.md)

**Tests**: Tests are required for changed behavior, failure paths, migrations, and public or
cross-language contracts.

## Phase 1: Foundational snapshot boundary

**Purpose**: Add immutable snapshot persistence and the typed boundaries all stories require.

- [x] T001 [P] Add snapshot migration and migration-schema coverage in `apps/api/migrations/007_corpus_snapshots.sql` and `apps/api/tests/integration/catalog_schema_test.go` per FR-002, FR-008, FR-009, and FR-012.
- [x] T002 [P] Add snapshot domain values, validation, and unit tests in `apps/api/internal/snapshot/domain/snapshot.go` and `apps/api/internal/snapshot/domain/snapshot_test.go` per FR-002, FR-006, FR-009, and FR-010.
- [x] T003 [P] Add the versioned `snapshotId` request/evidence contract and contract tests in `specs/005-grounded-rag-chat/contracts/chat-stream.schema.json`, `apps/api/internal/chat/agent/client.go`, `apps/api/internal/chat/agent/client_test.go`, `apps/agent/src/norvii_agent/transport/server.py`, and `apps/agent/tests/unit/test_transport_server.py` per FR-003 and FR-004.
- [x] T004 Implement snapshot persistence queries, manifest hashing, candidate validation, and transactional release activation with repository tests in `apps/api/internal/snapshot/postgres/repository.go` and `apps/api/internal/snapshot/postgres/repository_test.go` per FR-002, FR-005 through FR-010.
- [x] T005 Implement the snapshot publication and inspection application service with behavior tests in `apps/api/internal/snapshot/application/service.go` and `apps/api/internal/snapshot/application/service_test.go` per FR-006 through FR-010.

**Checkpoint**: Immutable manifests and a concurrency-safe release pointer exist before any
research request can depend on them.

---

## Phase 2: User Story 1 - Research the curated bilingual corpora (Priority: P1)

**Goal**: Researchers see the release they are using and every grounded answer is retrieved only
from that corpus's active snapshot.

**Independent Test**: Seed one active snapshot per corpus, request each corpus through the
catalog and chat contracts, and assert all evidence and citations retain the matching snapshot
identity.

- [x] T006 [US1] Add active-snapshot catalog projection and API response tests in `apps/api/internal/catalog/postgres/repository.go`, `apps/api/internal/catalog/postgres/repository_test.go`, `apps/api/internal/catalog/http/handler.go`, and `apps/api/internal/catalog/http/handler_test.go` per FR-001, FR-011, and SC-001.
- [x] T007 [US1] Resolve the active snapshot before chat forwarding, preserve it in SSE evidence and inspection responses, and test unavailable-release failures in `apps/api/internal/chat/domain/service.go`, `apps/api/internal/chat/http/handler.go`, and `apps/api/internal/chat/http/handler_test.go` per FR-003, FR-004, and FR-010.
- [x] T008 [US1] Make agent graph requests and evidence snapshot-aware, then constrain vector SQL with snapshot membership in `apps/agent/src/norvii_agent/graph/grounded_chat.py`, `apps/agent/src/norvii_agent/retrieval/postgres.py`, `apps/agent/tests/unit/test_grounded_chat.py`, and `apps/agent/tests/unit/test_postgres_retrieval.py` per FR-003, FR-004, and SC-002.
- [x] T009 [US1] Adapt snapshot-bearing catalog and chat contracts in `apps/web/src/api/contract.ts`, `apps/web/src/api/contract.test.ts`, `apps/web/src/api/researchProvider.ts`, `apps/web/src/api/chat.ts`, and their tests per FR-003, FR-004, and FR-011.
- [x] T010 [US1] Display compact active-snapshot identity in the catalog, workspace, citations, and inspection UI with localized copy and interaction tests in `apps/web/src/features/catalog/CorpusCard.tsx`, `apps/web/src/features/workspace/CorpusWorkspacePage.tsx`, `apps/web/src/features/workspace/ResearchChat.tsx`, `apps/web/src/features/workspace/workspace.css`, `apps/web/src/i18n/en/translation.ts`, `apps/web/src/i18n/pt/translation.ts`, and related tests per FR-004 and FR-011.

**Checkpoint**: The normal researcher flow cannot retrieve from a newer candidate, another
snapshot, or the other corpus.

---

## Phase 3: User Story 2 - Reingest without changing current research (Priority: P2)

**Goal**: A ready reingestion is visible as a candidate while active research continues using the
preceding release.

**Independent Test**: Reprocess a source, create a different ready document, then prove that the
active release and chat evidence remain unchanged until publish is requested.

- [x] T011 [US2] Expose active and latest candidate document identity in source projections and tests in `apps/api/internal/source/domain/source.go`, `apps/api/internal/source/postgres/repository.go`, `apps/api/internal/source/application/service.go`, and corresponding tests per FR-005 and FR-011.
- [x] T012 [US2] Add source-view candidate status and maintain the active release context in `apps/web/src/features/source-management/SourceStatus.tsx`, `apps/web/src/features/workspace/CorpusWorkspacePage.tsx`, localization resources, and component tests per FR-005 and FR-011.
- [x] T013 [US2] Add a deterministic regression test proving ingestion reprocessing creates artifacts without release mutation in `apps/ingestion/tests/integration/test_recovery.py` per FR-005, FR-006, and SC-003.

**Checkpoint**: Candidate data is separately visible and cannot become online evidence by
accident.

---

## Phase 4: User Story 3 - Publish and reproduce a validated snapshot (Priority: P3)

**Goal**: A maintainer explicitly publishes valid candidate evidence, retains history, and can
reproduce the curated releases.

**Independent Test**: Publish a ready candidate with expected release version, inspect the new
and previous manifests, retry the unchanged candidate, and run initialization twice without
duplicating releases.

- [x] T014 [US3] Add snapshot list, manifest, and explicit publication HTTP handlers plus contract tests in `apps/api/internal/snapshot/http/handler.go`, `apps/api/internal/snapshot/http/handler_test.go`, and `apps/api/cmd/api/main.go` per FR-006 through FR-010.
- [x] T015 [US3] Add an idempotent initial-snapshot command and integration coverage in `apps/api/cmd/initialize-snapshots/main.go`, `apps/api/cmd/initialize-snapshots/main_test.go`, and `apps/api/Makefile` per FR-001, FR-009, and FR-012.
- [x] T016 [US3] Add explicit publish, safe failure feedback, and snapshot-history/manifest inspection to the workspace in `apps/web/src/api/researchProvider.ts`, `apps/web/src/features/source-management/SourceStatus.tsx`, `apps/web/src/features/workspace/CorpusWorkspacePage.tsx`, localization resources, and related tests per FR-007, FR-008, FR-010, and FR-011.
- [x] T017 [US3] Add snapshot reproduction and active-boundary integration tests in `apps/api/tests/integration/corpus_snapshot_test.go` and `apps/agent/tests/integration/test_snapshot_retrieval.py` per SC-001 through SC-005.

**Checkpoint**: A maintainer can publish only complete candidates; historical evidence remains
inspectable and each initial corpus release is reproducible.

---

## Phase 5: Documentation and verification

**Purpose**: Make initialization reproducible and verify the feature end to end.

- [x] T018 [P] Update durable operation and module documentation in `docs/operations/local-environment.md`, `docs/modules/go-api.md`, `docs/modules/python-agent.md`, `docs/modules/python-ingestion.md`, and `docs/modules/web-client.md` per FR-012.
- [x] T019 Run the quickstart journey, language validation, module CI targets, migration verification, and snapshot isolation tests; record results in `specs/007-corpus-snapshots/quickstart.md` and mark completed tasks per all FRs and SCs.

## Dependencies and execution order

1. T001--T005 establish schema, contracts, validation, and release behavior.
2. T006--T010 depend on T001--T005 and deliver the P1 active research boundary.
3. T011--T013 depend on T006--T010 because candidate state is meaningful only relative to an
   active release.
4. T014--T017 depend on all preceding tasks because publication changes the active release.
5. T018--T019 run after all feature behavior is complete.

## Parallel opportunities

- T001, T002, and T003 touch separate modules and can start in parallel.
- After foundational work, T006 and T008 can proceed in parallel until their contract integration
  is needed.
- T011 and T013 can proceed in parallel after the snapshot schema exists.
- Documentation T018 can start after public commands and behavior stabilize.

## Implementation strategy

1. Write the repository, service, and agent tests first for an active snapshot boundary.
2. Implement the smallest vertical P1 route: initialize releases, resolve one in Go, filter in
   agent SQL, and display it in the web application.
3. Prove reingestion does not move the pointer.
4. Add explicit publication, manifest inspection, and the reproducible initialization command.
5. Run the full quality gate and use convergence to identify any remaining work.
