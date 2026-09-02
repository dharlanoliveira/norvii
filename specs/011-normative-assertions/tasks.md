# Tasks: Evidence-Backed Normative Assertions

**Input**: Design documents from `specs/011-normative-assertions/`

**Prerequisites**: [plan.md](plan.md), [spec.md](spec.md), [research.md](research.md), [data-model.md](data-model.md), [normative assertion stream contract](contracts/normative-assertion-stream.md), and [quickstart.md](quickstart.md)

**Tests**: Required for every changed behavior, persistence invariant, cross-language contract, and destructive-reset guard.

## Phase 1: Foundation

**Purpose**: Replace the canonical relationship shape before changing extraction, projection, or retrieval.

- [x] T001 Create the destructive migration replacing `semantic_relationships` and graph-release relationship membership with `normative_assertions`, legal-unit membership, and assertion membership in `apps/api/migrations/011_normative_assertions.sql` (FR-001--FR-007, FR-010).
- [x] T002 [P] Extend migration schema coverage for assertion ownership, predicate constraints, unit/document ownership, and graph-release membership in `apps/api/tests/integration/catalog_schema_test.go` (FR-001--FR-007, FR-010).
- [x] T003 [P] Replace direct semantic relationship fixtures with normative assertion fixtures in `apps/ingestion/tests/unit/publication/test_postgres_repository.py` and `apps/ingestion/tests/integration/test_recovery.py` (FR-003, FR-007).
- [x] T004 Update extraction domain values and document-scoped identity rebinding for explicit establishing and evidence unit IDs in `apps/ingestion/src/norvii_ingestion/semantic/extraction.py` and `apps/ingestion/src/norvii_ingestion/publication/postgres/repository.py` (FR-003--FR-007).
- [x] T005 Update canonical assertion persistence, idempotent backfill, and invalid-provenance rejection in `apps/ingestion/src/norvii_ingestion/publication/postgres/repository.py` (FR-003, FR-007, FR-010).
- [x] T006 Add deterministic repository tests for distinct establishing/evidence units, incomplete assertion rejection, and snapshot ownership in `apps/ingestion/tests/unit/publication/test_postgres_repository.py` (FR-003, FR-006, FR-007, FR-010).

**Checkpoint**: PostgreSQL can persist only complete, snapshot-isolated assertions; no direct semantic relationship remains in the canonical write path.

---

## Phase 2: User Story 1 - Inspect an exact legal assertion (Priority: P1) MVP

**Goal**: Project and retrieve a provision's exact assertion, two endpoints, establishing unit, and evidence unit.

**Independent Test**: A controlled document has article- and item-established assertions. Graph retrieval returns their direct source context and rejects incomplete provenance.

- [x] T007 [P] [US1] Add extraction tests for allowed predicate direction, atomic entity decomposition, explicit establishing unit, evidence unit, qualifiers, and incomplete output rejection in `apps/ingestion/tests/unit/semantic/test_extraction.py` (FR-003--FR-007a).
- [x] T008 [US1] Replace direct relationship extraction with typed normative assertion extraction, atomic entity decomposition, and validation in `apps/ingestion/src/norvii_ingestion/semantic/extraction.py` (FR-003--FR-007a).
- [x] T009 [P] [US1] Add graph-projection tests for assertion-node topology and exact citation provenance in `apps/ingestion/tests/unit/graph_projection/test_builder.py` and `apps/ingestion/tests/unit/publication/persistence/test_neo4j.py` (FR-002--FR-004, FR-008).
- [x] T010 [US1] Project legal units, entities, and assertion memberships from PostgreSQL in `apps/ingestion/src/norvii_ingestion/graph_projection/builder.py` (FR-001--FR-004, FR-008, FR-010).
- [x] T011 [US1] Replace direct Neo4j legal-relationship edges with `CONTAINS`, `ESTABLISHES`, `SUBJECT`, and `OBJECT` projection in `apps/ingestion/src/norvii_ingestion/publication/persistence/neo4j.py` (FR-002--FR-004, FR-008).
- [x] T012 [P] [US1] Add assertion-path decoding and graph query tests in `apps/agent/tests/unit/test_graph_retrieval.py` (FR-008, FR-013, SC-001).
- [x] T013 [US1] Replace direct-edge capability and retrieval queries with parameterized assertion-path queries in `apps/agent/src/norvii_agent/retrieval/graph.py` and assertion-path values in `apps/agent/src/norvii_agent/graph.py` (FR-005, FR-008, FR-010, FR-015).
- [x] T014 [P] [US1] Add agent stream-provider contract tests for assertion-path serialization in `apps/agent/tests/unit/transport/` (FR-008, FR-015).
- [x] T015 [US1] Carry the assertion-path inspection projection through `apps/agent/src/norvii_agent/transport/`, `apps/api/internal/chat/`, `apps/web/src/api/contract.ts`, and `apps/web/src/features/workspace/` according to `specs/011-normative-assertions/contracts/normative-assertion-stream.md` (FR-008, FR-015, SC-005).
- [x] T016 [US1] Add Go and TypeScript contract coverage for assertion IDs, predicates, locators, and qualifiers in `apps/api/internal/chat/` tests and `apps/web/src/api/contract.test.ts` (FR-008, FR-015, SC-005).

**Checkpoint**: A graph answer can expose one exact evidence-backed assertion through the browser inspection boundary.

---

## Phase 3: User Story 2 - Explore a chapter without over-retrieving (Priority: P2)

**Goal**: Retrieve only assertion evidence established in the relevant legal scope's descendants.

**Independent Test**: A document with three sibling chapters returns an assertion from only the requested chapter and preserves its direct provision owner.

- [x] T017 [P] [US2] Add a controlled multi-chapter assertion fixture and scope-isolation integration coverage in `apps/agent/tests/integration/test_hybrid_isolation.py` (FR-004, FR-009, FR-010, SC-002, SC-003).
- [x] T018 [P] [US2] Add unit coverage for descendant-only scope matching, six-edge bounds, and minimal hierarchy context in `apps/agent/tests/unit/test_graph_retrieval.py` (FR-009, FR-013, SC-002, SC-003).
- [x] T019 [US2] Extend planned graph retrieval and capability validation for assertion predicates, matching endpoint labels, and an optional scope locator selected only from the active snapshot legal-unit catalog in `apps/agent/src/norvii_agent/retrieval/planning.py`, `apps/agent/src/norvii_agent/providers/planning.py`, and `apps/agent/src/norvii_agent/retrieval/graph.py` (FR-005, FR-009, FR-013).
- [x] T020 [US2] Render the establishing location, evidence location, and minimal hierarchy context without exposing query or prompt content in `apps/web/src/features/workspace/ResearchChat.tsx`, `apps/web/src/features/workspace/ResearchChat.test.tsx`, and `apps/web/src/i18n/en/translation.ts` (FR-008, FR-009, FR-015).
- [x] T021 [US2] Add Portuguese localization values for the assertion inspection labels in `apps/web/src/i18n/pt/translation.ts` (FR-008, FR-009, FR-015).

**Checkpoint**: Chapter-scoped graph inspection returns only matching descendant assertions and makes its narrow scope visible.

---

## Phase 4: User Story 3 - Start the corpus from the new model (Priority: P3)

**Goal**: Prevent premature data deletion, then reset local corpus data and reingest only the assertion model.

**Independent Test**: Preflight failure preserves data; passing preflight permits a reset that leaves no PostgreSQL or Neo4j evidence; a subsequent source ingestion produces a graph-ready assertion snapshot.

- [x] T022 [P] [US3] Add reset-preflight, safe-failure, and volume-ownership tests in `.github/scripts/tests/test_persistence_reset.py` and extend `.github/scripts/tests/test_local_environment_manager.py` (FR-011, FR-012, FR-013, FR-014, SC-004).
- [x] T023 [US3] Extend the preflight-gated local corpus reset command in `infra/scripts/reset-local-data.sh` and `Makefile`, preserving explicit confirmation and volume-ownership checks before managed-service shutdown, PostgreSQL and Neo4j volume removal, and empty-state reporting (FR-011--FR-013, SC-004).
- [x] T024 [P] [US3] Add catalog, snapshot, and graph-empty-state tests in `apps/api/internal/catalog/`, `apps/api/internal/snapshot/`, and `apps/agent/tests/integration/test_hybrid_isolation.py` (FR-013, SC-004).
- [x] T025 [US3] Implement empty-corpus handling across `apps/api/internal/catalog/`, `apps/api/internal/snapshot/`, `apps/agent/src/norvii_agent/retrieval/`, and `apps/web/src/features/workspace/` (FR-013, SC-004).
- [x] T026 [US3] Update source registration and automatic graph-release build assertions in `apps/ingestion/tests/integration/test_recovery.py` and `apps/ingestion/src/norvii_ingestion/orchestration/processor.py` so fresh ingestion cannot publish a direct legacy relationship (FR-014, SC-005).
- [x] T027 [US3] Document the exact preflight, reset, source registration, reingestion, and post-ingestion validation sequence in `docs/operations/local-environment.md`, `docs/modules/python-ingestion.md`, and `specs/011-normative-assertions/quickstart.md` (FR-011--FR-014).

**Checkpoint**: A passing preflight authorizes a complete local reset, and the first new graph-ready snapshot is assertion-only.

---

## Phase 5: Cross-Cutting Verification

- [x] T028 [P] Update durable architecture ownership and graph model documentation in `docs/modules/python-agent.md`, `docs/modules/python-ingestion.md`, and `docs/architecture/overview.md` (FR-001--FR-015).
- [x] T029 [P] Update Feature 008 and 009 supersession notes in `specs/008-graphrag-hybrid-retrieval/` and `specs/009-planned-hybrid-retrieval/` without rewriting their historical acceptance evidence (FR-005, FR-015).
- [x] T030 Run `make -C apps/ingestion ci`, `make -C apps/agent ci`, `make -C apps/api test`, `make -C apps/web ci`, `.github/scripts/validate_repository_language.py`, and `git diff --check`; record results in `specs/011-normative-assertions/tasks.md` (SC-001--SC-006).
- [x] T031 Run the preflight, execute the authorized local corpus reset, re-register official sources, and complete fresh ingestion and assertion-path acceptance checks using `specs/011-normative-assertions/quickstart.md` (FR-011--FR-015, SC-004--SC-006).

## Dependencies & Execution Order

- T001--T006 establish the only permitted canonical model and block all story work.
- US1 depends on the foundation and supplies the assertion projection consumed by US2 and US3.
- US2 depends on US1's projection and inspection values.
- US3 depends on a passing controlled-data implementation from US1 and US2. T031 is the sole task permitted to delete local corpus data.
- Cross-cutting documentation and verification follow completed story work.

## Parallel Opportunities

- T002 and T003 can proceed independently after the migration design is fixed.
- T007, T009, T012, and T014 can prepare failing tests against the foundation in parallel.
- T017 and T018 can proceed in parallel after US1 assertions are queryable.
- T022 and T024 can proceed in parallel after the assertion model is verified.
- T028 and T029 can proceed in parallel after public contracts stabilize.

## Implementation Strategy

1. Deliver the schema and controlled assertion model through US1; do not reset data.
2. Deliver and verify hierarchy-scoped retrieval through US2; do not reset data.
3. Deliver reset preflight and empty-state behavior through US3; run all automated checks.
4. Only then execute T031, the user-authorized destructive reset and fresh ingestion.

## Reconciliation and Verification Record

- 2026-09-01: An implementation-presence audit reconciled T001-T028 with the Feature 011 merge
  commits (`7a41c03`, `9c845cc`, `a0f6af2`, and `136ff57`). The migration, extraction and
  canonical persistence, PostgreSQL-to-Neo4j assertion projection, scoped agent retrieval,
  cross-language stream contract, browser inspection, reset preflight, and empty-corpus behavior
  are present with their task-specified tests. No behavior was recreated during reconciliation.
- 2026-09-01: T029 added Feature 011 supersession notes to the Feature 008 reconciliation and
  Feature 009 specification. The notes preserve those features' historical acceptance evidence.
- 2026-09-01: T030 passed from the repository root: `make -C apps/ingestion ci` (104 selected
  tests), `make -C apps/agent ci` (70 selected tests), `make -C apps/api test`, and
  `make -C apps/web ci` (119 unit and 16 end-to-end tests). The agent suite emitted one
  non-failing third-party LangChain deprecation warning; the web build emitted its existing
  non-failing large-chunk advisory.
- 2026-09-01: `make persistence-assertion-preflight` passed without provider calls. PostgreSQL
  and Neo4j were healthy, schema migration version 20 was current, and the controlled ingestion
  and agent suites passed (104 and 70 selected tests respectively).
- 2026-09-01: T031 passed with `make persistence-reset CONFIRM=reset-norvii-data`. The preflight
  passed (migration 20, ingestion 104, agent 70), and the command removed the managed local
  volumes `norvii_postgres_data` and `norvii_neo4j_data`. `make bootstrap` recreated the environment;
  fresh ingestion produced LGPD snapshot `756d90a8-1df3-4c54-8f59-204a6e79f3af` with 11 normative
  assertions and GDPR snapshot `9a327d12-7aee-4984-8a66-2cf82a9a975b` with 13. A live LGPD hybrid
  request completed with an assertion path containing an assertion ID, predicate, endpoint labels,
  establishing and evidence locators, and hierarchy context.
