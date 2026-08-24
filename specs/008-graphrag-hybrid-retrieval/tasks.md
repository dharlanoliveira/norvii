# Tasks: GraphRAG and Hybrid Retrieval

**Input**: Design documents from `/specs/008-graphrag-hybrid-retrieval/`

**Prerequisites**: [plan.md](plan.md), [spec.md](spec.md), [research.md](research.md),
[data-model.md](data-model.md), [strategy contract](contracts/graphrag-stream.md), and
[quickstart.md](quickstart.md)

**Tests**: Tests are required for changed behavior, safe failure paths, migrations, and public or
cross-language contracts. Each behavior change follows red-green-refactor.

## Phase 1: Graph foundation

**Purpose**: Add immutable canonical extraction and graph-release records before online strategy
selection can use them.

- [X] T001 Add canonical semantic extraction and graph-release migration with schema integration coverage in `apps/api/migrations/008_graphrag.sql`, `apps/api/tests/integration/migration_test.go`, and `apps/api/tests/integration/catalog_schema_test.go` per FR-003, FR-005, and FR-013.
- [X] T002 [P] Add graph-release domain values, validation, and unit tests in `apps/api/internal/graphrelease/domain/graph_release.go` and `apps/api/internal/graphrelease/domain/graph_release_test.go` per FR-002 through FR-006.
- [X] T003 [P] Version the strategy, graph-path, contribution, and safe-outcome stream contract in `specs/005-grounded-rag-chat/contracts/chat-stream.schema.json`, `specs/008-graphrag-hybrid-retrieval/contracts/graphrag-stream.md`, and consumer/provider contract tests per FR-001, FR-007 through FR-009.
- [ ] T004 Implement canonical graph-release persistence, matching-snapshot validation, deterministic manifest identity, and repository tests in `apps/api/internal/graphrelease/postgres/repository.go` and `apps/api/internal/graphrelease/postgres/repository_test.go` per FR-002 through FR-006 and FR-013.
- [ ] T005 Implement the explicit graph-release status and inspection HTTP capability with handler tests in `apps/api/internal/graphrelease/application/service.go`, `apps/api/internal/graphrelease/http/handler.go`, and related tests per FR-003, FR-009, and FR-015.

**Checkpoint**: Canonical extraction artifacts and immutable graph-release records can safely
describe one published snapshot before any graph is queried.

---

## Phase 2: User Story 1 - Answer a connected legal question with grounded graph evidence (Priority: P1)

**Goal**: A researcher can select vector, graph, or hybrid retrieval and receive only
snapshot-scoped cited evidence or a safe strategy-specific abstention.

**Independent Test**: Seed a ready graph release for one snapshot, ask a connected question with
each strategy, and prove graph/hybrid evidence cannot cross corpus or snapshot boundaries.

- [X] T006 [P] [US1] Add strategy and graph-path domain values plus behavior tests in `apps/agent/src/norvii_agent/graph/grounded_chat.py` and `apps/agent/tests/unit/test_grounded_chat.py` per FR-001, FR-007 through FR-009.
- [X] T007 [P] [US1] Add snapshot-scoped Neo4j graph retrieval and deterministic hybrid evidence composition with unit tests in `apps/agent/src/norvii_agent/retrieval/graph.py`, `apps/agent/src/norvii_agent/retrieval/hybrid.py`, and corresponding unit tests per FR-002, FR-007, and FR-008.
- [X] T008 [US1] Adapt agent request decoding, graph composition, evidence serialization, and contract tests in `apps/agent/src/norvii_agent/transport/server.py`, `apps/agent/src/norvii_agent/transport/__main__.py`, and `apps/agent/tests/unit/test_transport_server.py` per FR-001, FR-002, and FR-009.
- [X] T009 [US1] Validate public strategy requests, resolve active graph-release status, forward typed strategy/snapshot data, and map safe strategy failures in `apps/api/internal/chat/application/service.go`, `apps/api/internal/chat/agent/client.go`, `apps/api/internal/chat/http/handler.go`, and related tests per FR-001, FR-002, FR-009, and FR-015.
- [X] T010 [US1] Add browser strategy request/stream parsing adapters and behavior tests in `apps/web/src/api/chat.ts`, `apps/web/src/api/contract.ts`, `apps/web/src/api/researchProvider.ts`, and related tests per FR-001, FR-007 through FR-010.
- [X] T011 [US1] Add accessible strategy selection and structured path/evidence inspection to `apps/web/src/features/workspace/ResearchChat.tsx`, focused child components under `apps/web/src/features/workspace/`, `apps/web/src/features/workspace/workspace.css`, localization resources, and component tests per FR-010, FR-012, and FR-014.
- [ ] T012 [US1] Add PostgreSQL/Neo4j snapshot-isolation integration coverage for vector, graph, and hybrid retrieval in `apps/agent/tests/integration/test_graph_retrieval.py` per FR-002, FR-007 through FR-009 and SC-001 through SC-004.

**Checkpoint**: The normal workspace can demonstrate a safe, strategy-labelled graph or hybrid
answer without weakening the active snapshot boundary.

---

## Phase 3: User Story 2 - Publish and inspect an evidence-backed legal graph (Priority: P2)

**Goal**: A maintainer can build, inspect, and safely diagnose a graph release for a published
snapshot; candidates remain excluded.

**Independent Test**: Build a graph release from a seeded snapshot, inspect its entity and
relationship provenance, reingest a candidate, and prove it is absent until publication plus a
new graph build complete.

- [X] T013 [P] [US2] Add bounded semantic extraction values, provider adapter, and deterministic tests in `apps/ingestion/src/norvii_ingestion/semantic/`, `apps/ingestion/tests/unit/semantic/`, and `apps/ingestion/src/norvii_ingestion/config/__init__.py` per FR-005, FR-006, and FR-015.
- [X] T014 [US2] Persist extraction runs, entities, and relationships atomically with immutable document provenance in `apps/ingestion/src/norvii_ingestion/publication/postgres/repository.py`, `apps/ingestion/src/norvii_ingestion/orchestration/processor.py`, and regression tests per FR-004 through FR-006 and FR-015.
- [ ] T015 [US2] Implement idempotent published-snapshot graph projection and explicit build command in `apps/ingestion/src/norvii_ingestion/graph_projection/`, `apps/ingestion/src/norvii_ingestion/graph_projection/__main__.py`, and integration tests per FR-003, FR-004, FR-013, and SC-002 and SC-005.
- [ ] T016 [US2] Expose graph-release summary and safe build/readiness state through `apps/api/internal/graphrelease/`, `apps/api/cmd/server/main.go`, and API tests per FR-003, FR-009, and FR-015.
- [ ] T017 [US2] Add graph-release status, rebuild guidance, and evidence-attributed graph inspection to `apps/web/src/features/source-management/`, `apps/web/src/features/workspace/CorpusWorkspacePage.tsx`, localized resources, and component tests per FR-003 through FR-006 and FR-014.

**Checkpoint**: Graph data is an inspectable, rebuildable projection of published evidence and
cannot expose an unpublished candidate.

---

## Phase 4: User Story 3 - Compare retrieval strategies for one research question (Priority: P3)

**Goal**: An evaluator compares independent vector, graph, and hybrid results for one immutable
snapshot and can navigate the supporting locations.

**Independent Test**: Run the same seeded question through every available strategy; ensure each
result retains its own evidence, path, outcome, and measurements, including an unavailable state.

- [ ] T018 [P] [US3] Add comparison result domain parsing and tests in `apps/web/src/research/domain/`, `apps/web/src/api/contract.ts`, and corresponding tests per FR-010 and FR-011.
- [ ] T019 [US3] Implement an accessible strategy-comparison view and source-navigation retention in focused components under `apps/web/src/features/workspace/`, `apps/web/src/features/workspace/workspace.css`, localization resources, and behavior tests per FR-010 through FR-014 and SC-003 through SC-004.
- [ ] T020 [US3] Add end-to-end comparison and unavailable-strategy journeys in `apps/web/tests/e2e/graphrag.spec.ts` and cross-service contract coverage per FR-009 through FR-012 and SC-001 through SC-004.

**Checkpoint**: The POC visibly compares the three strategies without conflating their evidence
or hiding a failed graph release.

---

## Phase 5: Documentation and verification

**Purpose**: Make graph construction reproducible and validate the feature end to end.

- [ ] T021 [P] Update graph-release operation and module documentation in `docs/operations/local-environment.md`, `docs/modules/go-api.md`, `docs/modules/python-agent.md`, `docs/modules/python-ingestion.md`, and `docs/modules/web-client.md` per FR-013 through FR-015.
- [ ] T022 Run the quickstart journey, language validation, module CI targets, graph rebuild/isolation integration tests, and record results in `specs/008-graphrag-hybrid-retrieval/quickstart.md` per all FRs and SCs.

## Dependencies and execution order

1. T001--T005 establish canonical releases and shared strategy contracts.
2. T006--T012 deliver the P1 online graph and hybrid research path.
3. T013--T017 create and expose the offline extraction and release lifecycle required for live P1
   graph data.
4. T018--T020 add strategy comparison after individual strategy results are reliable.
5. T021--T022 run only after all behavior and contracts stabilize.

## Parallel opportunities

- T002 and T003 can proceed in parallel after T001 starts.
- T006 and T007 can proceed in parallel after the strategy contract exists.
- T013 can proceed alongside the agent-only P1 unit work.
- T018 can start after stream contract fields stabilize.
- T021 can begin once public commands and UI copy are stable.

## Implementation strategy

1. Establish immutable canonical graph-release boundaries and safe contract variants.
2. Deliver and test the online strategy selection with seeded graph data, preserving all existing
   vector behavior.
3. Add bounded semantic extraction and an explicit idempotent graph builder that supplies real
   graph data for published snapshots.
4. Add the evaluator comparison view only after each strategy's independent inspection is clear.
5. Run the complete reproducibility and quality gate, then converge remaining work.

## Phase 6: Convergence

- [ ] T023 Add PostgreSQL and Neo4j service-backed isolation coverage for vector, graph, and hybrid retrieval, including foreign-corpus and foreign-snapshot exclusions, per T012, FR-002, FR-007 through FR-009, and SC-001 through SC-004 (partial).
- [ ] T024 Add a service-backed graph-projection integration journey that builds the same published snapshot three times and verifies identical graph membership, evidence locations, readiness, and candidate exclusion per T015, FR-003, FR-004, FR-013, SC-002, and SC-005 (partial).
- [ ] T025 Add Playwright strategy-comparison journeys for ready graph/hybrid results, an unavailable strategy, and graph-path source navigation retention per T020, FR-009 through FR-012, and SC-001 through SC-004 (missing).
- [ ] T026 Execute the documented paid reingestion, publish, and graph-build journey for both curated corpora, then record the real graph-release measurements and reproducibility evidence in quickstart per T022, SC-005, and SC-006 (partial).
