# Tasks: Corpus Catalog and Ingestion

**Input**: Design documents from `specs/004-corpus-catalog/`

**Prerequisites**: `plan.md`, `spec.md`, `research.md`, `data-model.md`, `contracts/`, `quickstart.md`

**TDD rule**: Every RED task must be observed failing for the intended missing behavior before its GREEN task begins. Refactor only while the focused and module suites remain green.

## Phase 1: Setup

- [X] T001 Promote the versioned HTTP and ingestion-work designs into machine-readable schemas under `contracts/corpus-ingestion/v1/` and register them in `contracts/README.md` [FR-029, FR-030]
- [X] T002 [P] Pin pypdf and Trafilatura runtime dependencies plus extraction test tooling in `apps/ingestion/pyproject.toml` and `apps/ingestion/uv.lock` [FR-017, FR-023-FR-025]
- [X] T003 [P] Add API server and ingestion worker configuration examples without usable secrets in `infra/.env.example`, `apps/api/internal/platform/config/`, and `apps/ingestion/src/norvii_ingestion/config/` [FR-016, FR-018, FR-021]
- [X] T004 [P] Add contract-schema validation helpers and tests in `.github/scripts/validate_contracts.py` and `.github/scripts/tests/test_validate_contracts.py` [FR-029, FR-030]

## Phase 2: Foundational

**Purpose**: Establish contracts, schema, transaction boundaries, and runnable composition roots required by every story.

- [X] T005 Write failing migration contract and PostgreSQL integration tests for all canonical tables, constraints, indexes, ownership, seeds, and repeatability in `apps/api/tests/integration/catalog_schema_test.go` [FR-001-FR-007, FR-012-FR-016, FR-020-FR-028]
- [X] T006 Implement ordered migration `apps/api/migrations/002_corpus_ingestion.sql` and update `apps/api/migrations/embed.go` until T005 passes [FR-001-FR-007, FR-012-FR-016, FR-020-FR-028]
- [X] T007 [P] Write failing Go domain tests for corpus/source validation, lifecycle, concurrency, limits, and safe errors in `apps/api/internal/catalog/domain/` and `apps/api/internal/source/domain/` [FR-002-FR-020, FR-030]
- [X] T008 [P] Write failing Python domain tests for claims, hashes, artifact hierarchy, publication commands, and safe failures in `apps/ingestion/tests/unit/domain/` [FR-020-FR-028, FR-030]
- [X] T009 Implement cohesive Go corpus/source domain models and typed errors in `apps/api/internal/catalog/domain/` and `apps/api/internal/source/domain/` until T007 passes [FR-002-FR-020, FR-030]
- [X] T010 Implement immutable Python work, origin, revision, document, unit, and failure models in `apps/ingestion/src/norvii_ingestion/domain/` until T008 passes [FR-020-FR-028, FR-030]
- [X] T011 [P] Write failing provider/consumer contract tests in `apps/api/tests/contract/corpus_ingestion_contract_test.go`, `apps/ingestion/tests/contract/test_ingestion_work_contract.py`, and `apps/web/src/api/contract.test.ts` [FR-029, FR-030]
- [X] T012 Implement shared contract fixtures and module adapters under `contracts/corpus-ingestion/v1/fixtures/`, `apps/api/internal/platform/contracts/`, `apps/ingestion/src/norvii_ingestion/contracts/`, and `apps/web/src/api/` until T011 passes [FR-029, FR-030]
- [X] T013 [P] Write failing API composition tests for configuration, middleware, request IDs, body limits, safe errors, shutdown, and health in `apps/api/internal/platform/httpserver/server_test.go` [FR-016, FR-030]
- [X] T014 Implement the standard-library HTTP server and `apps/api/cmd/server/main.go` until T013 passes [FR-016, FR-030]
- [X] T015 [P] Write failing worker composition tests for polling, bounded leases, cancellation, safe logs, and shutdown in `apps/ingestion/tests/unit/orchestration/test_worker.py` [FR-020, FR-021]
- [X] T016 Implement the worker shell and `norvii-ingestion-worker` entry point in `apps/ingestion/src/norvii_ingestion/orchestration/worker.py` and `apps/ingestion/pyproject.toml` until T015 passes [FR-020, FR-021]

## Phase 3: User Story 1 - Explore Two Initial Real Corpora (P1) MVP

**Independent Test**: From clean storage, initialize once and repeatedly, process the two official sources, and browse isolated persisted documents or honest retryable failures.

- [X] T017 [P] [US1] Write failing Go seed/read repository tests for fixed initial identities, non-overwrite, ordering, and corpus isolation in `apps/api/internal/catalog/postgres/repository_test.go` [FR-004-FR-008, FR-031]
- [X] T018 [P] [US1] Write failing Python integration tests for deterministic queue claim, lease ownership, initial URL extraction, and atomic publication in `apps/ingestion/tests/integration/test_initial_ingestion.py` [FR-005-FR-007, FR-020-FR-028]
- [X] T019 [P] [US1] Write failing React journey tests for authoritative initial catalog/workspace data, loading/failure states, and zero runtime fixture fallback in `apps/web/src/features/catalog/CorpusCatalogPage.test.tsx` and `apps/web/src/features/workspace/CorpusWorkspacePage.test.tsx` [FR-031-FR-037]
- [X] T020 [US1] Implement corpus/source/document PostgreSQL read repositories in `apps/api/internal/catalog/postgres/`, `apps/api/internal/source/postgres/`, and `apps/api/internal/document/postgres/` until T017 passes [FR-004-FR-008, FR-031-FR-033]
- [X] T021 [US1] Implement leased claim and atomic artifact publication repositories in `apps/ingestion/src/norvii_ingestion/publication/postgres/` until T018 reaches extraction boundaries [FR-020-FR-028]
- [X] T022 [US1] Implement the minimal pinned-address HTTPS acquirer, structured HTML extraction, normalized block assembly, fallback units, legal-marker detection, hierarchy validation, and hashing in `apps/ingestion/src/norvii_ingestion/acquisition/` and `apps/ingestion/src/norvii_ingestion/extraction/` until T018 passes with the controlled initial URLs [FR-015, FR-016, FR-018, FR-023-FR-028]
- [X] T023 [US1] Implement read-only catalog/source/document HTTP handlers in `apps/api/internal/catalog/http/`, `apps/api/internal/source/http/`, and `apps/api/internal/document/http/` [FR-008, FR-029-FR-033]
- [X] T024 [US1] Implement React HTTP catalog/workspace providers and route loading in `apps/web/src/api/`, `apps/web/src/app/routes.tsx`, and `apps/web/src/research/domain/` until T019 passes [FR-031-FR-035]
- [X] T025 [US1] Remove production runtime construction of demonstration catalog data from `apps/web/src/app/App.tsx` and add a static/runtime-fixture absence check in `apps/web/scripts/check-runtime-fixtures.mjs` [FR-031, FR-037]
- [X] T026 [US1] Run the clean and repeated initial-data checkpoint and record measured results in `specs/004-corpus-catalog/quickstart.md` [SC-001, SC-002, SC-009]

## Phase 4: User Story 2 - Create and Manage a Corpus (P2)

**Independent Test**: Create, edit, disable, direct-open, and re-enable one corpus while preserving identity and rejecting stale/invalid writes.

- [X] T027 [P] [US2] Write failing Go use-case, repository, and handler tests for corpus create/edit/disable/enable, validation, atomicity, and optimistic concurrency in `apps/api/internal/catalog/` [FR-008-FR-011, FR-029, FR-030]
- [X] T028 [P] [US2] Write failing React component/accessibility tests for corpus forms, confirmation, stale state, validation, and enabled/disabled views in `apps/web/src/features/catalog/` [FR-008-FR-011, FR-034-FR-036]
- [X] T029 [US2] Implement corpus application services, PostgreSQL mutations, and HTTP handlers in `apps/api/internal/catalog/application/`, `apps/api/internal/catalog/postgres/`, and `apps/api/internal/catalog/http/` until T027 passes [FR-008-FR-011, FR-029, FR-030]
- [X] T030 [US2] Implement corpus management UI and HTTP mutations in `apps/web/src/features/catalog/` and `apps/web/src/api/` until T028 passes [FR-008-FR-011, FR-034-FR-036]
- [X] T031 [US2] Add English and Portuguese corpus-management resources with parity tests in `apps/web/src/i18n/en/translation.ts`, `apps/web/src/i18n/pt/translation.ts`, and `apps/web/src/i18n/config.test.ts` [FR-034, FR-035]
- [X] T032 [US2] Refactor the corpus slice while focused Go and React suites remain green and record its checkpoint in `specs/004-corpus-catalog/quickstart.md` [SC-003, SC-010, SC-011]

## Phase 5: User Story 3 - Add and Ingest an Official URL (P3)

**Independent Test**: Add a controlled public HTTPS page, safely traverse redirects, publish it, and reject unsafe, duplicate, unsupported, timed-out, and oversized cases.

- [X] T033 [P] [US3] Write failing Go source-creation tests for URL normalization, corpus/source limits, duplicates, work creation, contract errors, and isolation in `apps/api/internal/source/` [FR-012-FR-016, FR-018-FR-020, FR-029, FR-030]
- [X] T034 [P] [US3] Write failing Python URL safety tests with injected DNS/socket/TLS responses for public address pinning, every private/reserved class, redirects, proxy bypass, timeouts, media types, and streaming size in `apps/ingestion/tests/unit/acquisition/test_https_acquirer.py` [FR-015, FR-016, FR-018]
- [X] T035 [P] [US3] Write failing Python HTML extraction and hierarchy tests in English and Portuguese in `apps/ingestion/tests/unit/extraction/test_html_extractor.py` [FR-023-FR-026, FR-035]
- [X] T036 [P] [US3] Write failing React source-add and lifecycle tests in `apps/web/src/features/source-management/UrlSourceForm.test.tsx` [FR-012-FR-020, FR-034-FR-036]
- [X] T037 [US3] Implement URL source use cases, repository mutation, and HTTP handler in `apps/api/internal/source/` until T033 passes [FR-012-FR-020, FR-029, FR-030]
- [X] T038 [US3] Harden the MVP pinned-address HTTPS acquisition in `apps/ingestion/src/norvii_ingestion/acquisition/https.py` for the exhaustive redirect, address-class, timeout, proxy, media, and size cases until T034 passes [FR-015, FR-016, FR-018]
- [X] T039 [US3] Extend the MVP Trafilatura extraction in `apps/ingestion/src/norvii_ingestion/extraction/html.py` for the complete English and Portuguese hierarchy cases until T035 passes [FR-023-FR-026]
- [X] T040 [US3] Implement URL source form, status polling, and localized outcomes in `apps/web/src/features/source-management/` until T036 passes [FR-012-FR-020, FR-034-FR-036]
- [X] T041 [US3] Complete controlled URL integration and corpus-isolation tests in `apps/ingestion/tests/integration/test_url_ingestion.py` and `apps/api/tests/integration/source_api_test.go` [FR-018-FR-030, SC-004-SC-009]

## Phase 6: User Story 4 - Upload and Ingest a PDF (P4)

**Independent Test**: Upload one valid text PDF, preserve and deliver its bytes, publish pages/sections, and reject duplicate, malformed, encrypted, image-only, and oversized files.

- [X] T042 [P] [US4] Write failing Go multipart streaming, PDF signature/media, filename, size, hash, duplicate, binary-delivery, and isolation tests in `apps/api/internal/source/http/pdf_test.go` and `apps/api/internal/source/postgres/pdf_repository_test.go` [FR-013-FR-017, FR-019, FR-029-FR-030, FR-033]
- [X] T043 [P] [US4] Write failing pypdf extraction, page fallback, legal-unit, empty/encrypted/image-only, and normalization tests in `apps/ingestion/tests/unit/extraction/test_pdf_extractor.py` [FR-017, FR-023-FR-026]
- [X] T044 [P] [US4] Write failing React upload progress, validation, lifecycle, and origin-action tests in `apps/web/src/features/source-management/PdfSourceForm.test.tsx` [FR-013-FR-017, FR-033-FR-036]
- [X] T045 [US4] Implement streamed PDF origin persistence and delivery in `apps/api/internal/source/application/`, `apps/api/internal/source/postgres/`, and `apps/api/internal/source/http/` until T042 passes [FR-013-FR-017, FR-019, FR-029-FR-030, FR-033]
- [X] T046 [US4] Implement pypdf page extraction and legal-unit enrichment in `apps/ingestion/src/norvii_ingestion/extraction/pdf.py` until T043 passes [FR-017, FR-023-FR-026]
- [X] T047 [US4] Implement PDF upload UI and safe origin access in `apps/web/src/features/source-management/` and `apps/web/src/features/workspace/PdfSourceViewer.tsx` until T044 passes [FR-013-FR-017, FR-033-FR-036]
- [X] T048 [US4] Complete generated-fixture PDF integration tests in `apps/ingestion/tests/integration/test_pdf_ingestion.py` and `apps/api/tests/integration/pdf_api_test.go` [SC-004-SC-009]

## Phase 7: User Story 5 - Inspect, Retry, and Reprocess Sources (P5)

**Independent Test**: Recover an expired/failing attempt, retry successfully, reprocess unchanged and changed content, and preserve a prior ready document after failure.

- [X] T049 [P] [US5] Write failing Go lifecycle command tests for retry/reprocess state, concurrency, and prior-ready visibility in `apps/api/internal/source/application/lifecycle_test.go` [FR-020-FR-030]
- [X] T050 [P] [US5] Write failing Python lease renewal/expiry, crash recovery, idempotent publication, changed version, and failed-reprocess tests in `apps/ingestion/tests/integration/test_recovery.py` [FR-020-FR-028]
- [X] T051 [P] [US5] Write failing React lifecycle inspection/retry/reprocess tests in `apps/web/src/features/source-management/SourceStatus.test.tsx` [FR-020-FR-021, FR-027-FR-028, FR-034-FR-036]
- [X] T052 [US5] Implement Go retry/reprocess application services and HTTP handlers in `apps/api/internal/source/` until T049 passes [FR-020-FR-030]
- [X] T053 [US5] Implement Python lease renewal, expiry recovery, and idempotent/changed publication in `apps/ingestion/src/norvii_ingestion/orchestration/` and `apps/ingestion/src/norvii_ingestion/publication/postgres/` until T050 passes [FR-020-FR-028]
- [X] T054 [US5] Implement React source status, attempt detail, retry, and reprocess interactions in `apps/web/src/features/source-management/` until T051 passes [FR-020-FR-021, FR-027-FR-028, FR-034-FR-036]

## Phase 8: User Story 6 - Browse Real Documents Without Simulated Chat (P6)

**Independent Test**: Keyboard-browse persisted source states and real PDF/URL documents in both interface languages while Chat stays unavailable and no runtime fixture appears.

- [X] T055 [P] [US6] Rewrite failing source-tree/controller/viewer tests around HTTP source states, real units, origin actions, and corpus isolation in `apps/web/src/features/workspace/` [FR-031-FR-036]
- [X] T056 [P] [US6] Write failing unavailable-chat tests and remove prepared-response expectations in `apps/web/src/features/workspace/ResearchChat.test.tsx` [FR-037, FR-038]
- [X] T057 [US6] Refactor source tree, controller, and URL/PDF viewers for authoritative models in `apps/web/src/features/workspace/` until T055 passes [FR-031-FR-036]
- [X] T058 [US6] Replace assistant runtime composition with localized unavailable chat in `apps/web/src/features/workspace/ResearchChat.tsx` and remove production demonstration engine files under `apps/web/src/research/demonstration/` until T056 and T025 pass [FR-031, FR-037, FR-038]
- [X] T059 [US6] Add bilingual lifecycle/viewer/chat-unavailable resources and parity coverage in `apps/web/src/i18n/` [FR-034-FR-037]
- [X] T060 [US6] Add Playwright catalog-to-real-document, empty, failed, keyboard, locale, and unavailable-chat journeys in `apps/web/tests/e2e/corpus-ingestion.spec.ts` [SC-003, SC-004, SC-009-SC-012]

## Phase 9: Operations, Quality, and Convergence Preparation

- [X] T061 [P] Extend `infra/scripts/manage-local-environment.py`, root `Makefile`, and `.agents/skills/bootstrap-norvii/` to migrate and run API, worker, and web with separate `.log/` files [FR-006, SC-013]
- [X] T062 [P] Extend persistence CI with migration, lease, atomicity, URL safety, PDF, isolation, and contract jobs in `.github/workflows/ci.yml` and `.github/scripts/` [SC-005-SC-009, SC-012]
- [X] T063 [P] Update `docs/operations/local-environment.md`, module READMEs, root `README.md`, and contract registry for the real workflow and troubleshooting [SC-013]
- [X] T064 Run every module format/lint/type/test/build gate, contract/language validation, `git diff --check`, and Sonar-compatible coverage; resolve all failures [SC-005-SC-012]
- [X] T065 Execute the clean `specs/004-corpus-catalog/quickstart.md` twice, record timings and terminal source states, and correct documentation drift [SC-001-SC-013]

## Dependencies and Execution Order

- Phase 1 precedes Phase 2; Phase 2 blocks all stories.
- US1 is the MVP and proves seed/read/publication seams.
- US2 depends only on Phase 2 and may run beside US1 after shared repository interfaces stabilize.
- US3 depends on the publication seam from US1; US4 may proceed beside US3 after that seam is green.
- US5 depends on at least one ready source path from US3 or US4.
- US6 depends on authoritative read contracts from US1 and source states from US3/US4.
- Operations and convergence preparation follow all selected stories.

## Parallel Opportunities

- Contract validation, dependency/configuration, and language-specific foundational tests use separate files.
- Within each story, Go, Python, and React RED tasks can be written in parallel.
- US2 and US3 API work may proceed independently after Phase 2.
- PDF and URL extractors are independent once shared normalization/publication is stable.
- Documentation and CI tasks T061-T063 are parallel after behavior paths stabilize.

## Implementation Strategy

Complete Setup and Foundation, then deliver US1 as the first demonstrable vertical checkpoint. Continue in priority order, keeping each RED-GREEN-refactor cycle focused. Do not begin a later story to hide a failing earlier checkpoint. Commit complete logical groups only after their focused and module gates pass.

## Format Validation

All tasks use checkbox, sequential ID, optional parallel marker, required story label in story phases, exact repository paths, and requirement identifiers.

## Phase 10: Convergence

- [X] T066 Extend the v1 source detail contract, PostgreSQL projection, Go handlers, TypeScript parser, and bilingual lifecycle UI with origin metadata and the latest safe processing attempt per FR-015, FR-021, FR-029, and US5 (partial)
- [X] T067 Expose complete latest source-revision provenance and contract-compliant URL origin metadata through the document API and render it with safe PDF/URL actions in the workspace per FR-033 and US6/AC3-4 (partial)
- [X] T068 Add bounded initial-source terminal-state verification to the managed local bootstrap with deterministic lifecycle tests per plan: local environment and SC-013 (partial)

## Phase 11: Convergence

- [X] T069 Expose the complete newest-first safe processing-attempt history through the PostgreSQL source projection, v1 contract, Go API, TypeScript parser, and bilingual lifecycle UI per US5/AC2 and FR-021 (partial)
- [X] T070 Add safe structured API access logging and expand worker lifecycle logging with the allowlisted operation, identity, state, duration, count, pipeline, and category fields required by plan: observability (partial)

## Phase 12: Convergence

- [X] T071 Require localized confirmation before source reprocessing and distinguish duplicate PDF responses from client-side PDF validation failures in the React source-management UI per FR-019, FR-034, and FR-036 (partial)

## Phase 13: Convergence

- [X] T072 Enforce the v1 one-megabyte non-PDF request-body limit independently from multipart PDF allowance in Go HTTP middleware and tests per plan: public contracts (partial)

## Phase 14: Convergence

- [X] T073 Render and scroll to the selected stable document-unit span instead of changing selection state alone, with component and browser coverage, per US6/AC2 and FR-033 (partial)
- [X] T074 Add an explicit bilingual source-library empty state while preserving both source-add actions per edge case: corpus has no sources and FR-034 (partial)

## Phase 15: Catalog Management UX

- [X] T075 Move corpus creation and editing to dedicated localized routes, preserve the catalog as a selection surface, and replace unstyled card controls with a polished contextual management menu using accessible React behavior tests per US2/AC6, FR-009, FR-034, and FR-036
