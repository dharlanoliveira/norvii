# Tasks: Local Persistence Foundation

**Input**: Design documents from `specs/003-local-persistence-foundation/`

**Prerequisites**: `plan.md`, `spec.md`, `research.md`, `data-model.md`, `contracts/`, and `quickstart.md`

**Tests**: TDD is mandatory. For each behavior task, write the listed test first,
observe the relevant failure, implement the smallest passing behavior, and refactor
only while the test remains green.

**Organization**: Tasks are grouped by user story and use only modules marked as
changed in the plan.

## Format: `[ID] [P?] [Story] Description`

- **[P]**: Can run in parallel because it changes distinct files and has no unmet dependency.
- **[Story]**: Maps the task to a user story in `spec.md`.
- Every task names requirement IDs and exact repository paths.

## Phase 1: Setup (Shared Module Foundations)

**Purpose**: Establish independently owned Go, Python, and root orchestration entry points.

- [X] T001 Create the Go 1.26 module, pin pgx 5.10.0, Neo4j Go Driver 6.2.0, and tern 2.4.2, and expose format, vet, test-race, build, and CI targets in `apps/api/go.mod`, `apps/api/go.sum`, and `apps/api/Makefile` [FR-014, FR-015]
- [X] T002 [P] Create the Python 3.13 uv project, pin psycopg 3.3.4, Neo4j Python Driver 6.2.0, pytest 9.1.1, Ruff 0.16.3, and mypy 2.3.0, and expose format, lint, type-check, test, build, and CI targets in `apps/ingestion/pyproject.toml`, `apps/ingestion/uv.lock`, and `apps/ingestion/Makefile` [FR-014, FR-015]
- [X] T003 Create root persistence orchestration entry points that delegate without suppressing module output in `Makefile` [FR-001, FR-014, FR-016]
- [X] T004 Add generated Python artifacts, Go build artifacts, and `infra/.env` exclusions without weakening existing rules in `.gitignore` [FR-004, FR-013]

---

## Phase 2: Foundational Configuration (Blocking Prerequisites)

**Purpose**: Implement the shared environment contract independently in each runtime before opening database connections.

**Critical**: No user story implementation begins until configuration validation is green in both modules.

- [X] T005 [P] Write failing table-driven tests for required values, port ranges, timeout bounds, and secret-safe Go validation errors in `apps/api/internal/platform/persistence/config_test.go` [FR-004, FR-008, FR-013]
- [X] T006 [P] Write failing pytest cases for required values, port ranges, timeout bounds, immutable typed configuration, and secret-safe Python validation errors in `apps/ingestion/tests/unit/publication/persistence/test_config.py` [FR-004, FR-008, FR-013]
- [X] T007 Implement typed environment loading and validation with discrete credential fields in `apps/api/internal/platform/persistence/config.go` until T005 passes [FR-004, FR-008, FR-013]
- [X] T008 [P] Implement immutable configuration value classes and an environment loader class in `apps/ingestion/src/norvii_ingestion/publication/persistence/config.py` until T006 passes [FR-004, FR-008, FR-013]
- [X] T009 Add package boundaries and exports without cross-module imports in `apps/ingestion/src/norvii_ingestion/__init__.py`, `apps/ingestion/src/norvii_ingestion/publication/__init__.py`, and `apps/ingestion/src/norvii_ingestion/publication/persistence/__init__.py` [FR-015]

**Checkpoint**: Go and Python reject unsafe or incomplete persistence configuration deterministically and independently.

---

## Phase 3: User Story 1 - Start the Persistence Environment (Priority: P1) MVP

**Goal**: Start exactly two digest-pinned stores with authenticated readiness, separate named data, and one documented command.

**Independent Test**: From valid local configuration, render Compose, start the default environment, and observe PostgreSQL plus Neo4j become healthy within two minutes with no third service.

### Tests for User Story 1

> Write and run these tests first; they must fail before the corresponding implementation.

- [X] T010 [P] [US1] Write failing configuration tests for exact services, image tags and digests, authenticated health checks, resource bounds, ports, and separate volumes in `.github/scripts/tests/test_persistence_configuration.py` [FR-001, FR-002, FR-003, FR-005, FR-009]
- [X] T011 [P] [US1] Write failing shell-contract tests for bounded health inspection, safe diagnostics, and credential redaction in `.github/scripts/tests/test_persistence_scripts.py` [FR-005, FR-008, FR-013, SC-002, SC-005]

### Implementation for User Story 1

- [X] T012 [US1] Define the `norvii` Compose project with digest-pinned pgvector PostgreSQL and Neo4j Community services, authenticated health checks, ports, resource bounds, and separate named volumes in `infra/compose.yaml` until T010 passes [FR-001, FR-002, FR-003, FR-005, FR-009]
- [X] T013 [P] [US1] Provide every required non-secret value and conspicuous password replacement markers in `infra/.env.example` [FR-004]
- [X] T014 [US1] Implement bounded service-status inspection with service-scoped next-step diagnostics in `infra/scripts/inspect-health.sh` until T011 passes [FR-005, FR-008, FR-013, FR-016]
- [X] T015 [US1] Complete `persistence-config`, `persistence-up`, and `persistence-health` delegation in `Makefile` with `infra/.env` preflight validation [FR-001, FR-003, FR-004, FR-016, SC-001, SC-002]
- [X] T016 [US1] Run the independent start journey and record the implementation checkpoint in `specs/003-local-persistence-foundation/quickstart.md` [FR-001, FR-005, SC-001, SC-002]

**Checkpoint**: User Story 1 is independently runnable; normal startup yields exactly two authenticated healthy stores.

---

## Phase 4: User Story 2 - Initialize and Verify Canonical Storage (Priority: P1)

**Goal**: Enable vector storage through an inspectable, repeatable Go-owned migration path and prove canonical data survives restart.

**Independent Test**: On a clean PostgreSQL volume, apply initialization twice, inspect migration version and vector capability, then restart and verify disposable marker persistence.

### Tests for User Story 2

> Write and run these tests first; they must fail before the corresponding implementation.

- [X] T017 [US2] Write failing Go tests for embedded migration loading, current-version reporting, repeat execution, failed-migration attribution, and credential-safe errors in `apps/api/internal/platform/persistence/migrator_test.go` [FR-006, FR-007, FR-008, SC-003, SC-005]
- [X] T018 [P] [US2] Write a failing service-backed initialization test for vector availability and idempotent migration state in `apps/api/tests/integration/migration_test.go` [FR-006, FR-007, SC-003]

### Implementation for User Story 2

- [X] T019 [US2] Add the ordered `vector` extension migration with a reversible tern section in `apps/api/migrations/001_enable_vector.sql` [FR-006, FR-007]
- [X] T020 [US2] Implement the embedded tern migrator and safe status model in `apps/api/internal/platform/persistence/migrator.go` until T017 passes [FR-006, FR-007, FR-008]
- [X] T021 [US2] Implement migration and migration-status composition roots in `apps/api/cmd/migrate/main.go` and `apps/api/cmd/migration-status/main.go` [FR-006, FR-007, FR-008, FR-016]
- [X] T022 [US2] Expose `persistence-migrate` and `persistence-migration-status` through `apps/api/Makefile` and the root `Makefile` [FR-007, FR-016]
- [X] T023 [US2] Implement the disposable vector, repeated-initialization, and normal-restart marker journey in `infra/scripts/verify-foundation.sh` until T018 passes [FR-006, FR-007, FR-009, SC-003, SC-006]

**Checkpoint**: User Story 2 is independently testable against the User Story 1 environment and leaves migration version 1 with vector capability enabled.

---

## Phase 5: User Story 3 - Prove Module Connectivity (Priority: P1)

**Goal**: Prove Go and Python can independently authenticate to both stores through narrow production-driver adapters without creating product artifacts.

**Independent Test**: Run each module verifier with valid, invalid, and unavailable-store configuration; valid checks finish within ten seconds and failures remain service-scoped and secret-safe.

### Tests for User Story 3

> Write and run these tests first; they must fail before the corresponding implementation.

- [X] T024 [P] [US3] Write failing Go unit tests with injected clients for ordered store checks, context deadlines, cleanup, partial failure, and secret-safe diagnostics in `apps/api/internal/platform/persistence/verifier_test.go` [FR-011, FR-012, FR-013, SC-004, SC-005]
- [X] T025 [P] [US3] Write failing Python unit tests with injected adapters for ordered store checks, timeouts, cleanup, partial failure, and secret-safe diagnostics in `apps/ingestion/tests/unit/publication/persistence/test_verifier.py` [FR-011, FR-012, FR-013, SC-004, SC-005]
- [X] T026 [P] [US3] Write failing Go service-backed checks for authenticated PostgreSQL and Neo4j operations with no product writes in `apps/api/tests/integration/connectivity_test.go` [FR-011, FR-012, FR-019]
- [X] T027 [P] [US3] Write failing Python service-backed checks for authenticated PostgreSQL and Neo4j operations with no product writes in `apps/ingestion/tests/integration/publication/persistence/test_connectivity.py` [FR-011, FR-012, FR-019]

### Implementation for User Story 3

- [X] T028 [US3] Implement narrow pgx and Neo4j driver adapters plus a deadline-bound verifier in `apps/api/internal/platform/persistence/postgres.go`, `apps/api/internal/platform/persistence/neo4j.go`, and `apps/api/internal/platform/persistence/verifier.go` until T024 and T026 pass [FR-011, FR-012, FR-013]
- [X] T029 [US3] Implement the Go verifier composition root with safe English output in `apps/api/cmd/verify-persistence/main.go` [FR-008, FR-011, FR-012, FR-013]
- [X] T030 [P] [US3] Implement class-based psycopg and Neo4j adapters plus a typed verifier in `apps/ingestion/src/norvii_ingestion/publication/persistence/postgres.py`, `apps/ingestion/src/norvii_ingestion/publication/persistence/neo4j.py`, and `apps/ingestion/src/norvii_ingestion/publication/persistence/verifier.py` until T025 and T027 pass [FR-011, FR-012, FR-013]
- [X] T031 [US3] Implement the Python module entry point and package script with safe English output in `apps/ingestion/src/norvii_ingestion/publication/persistence/__main__.py` and `apps/ingestion/pyproject.toml` [FR-008, FR-011, FR-012, FR-013]
- [X] T032 [US3] Expose independent module and aggregate `verify-persistence` commands in `apps/api/Makefile`, `apps/ingestion/Makefile`, and `Makefile` [FR-014, FR-015, FR-016]
- [X] T033 [US3] Extend the service-backed journey with both module verifiers, invalid-credential failures, unavailable-store timeouts, and no-product-artifact assertions in `infra/scripts/verify-foundation.sh` [FR-011, FR-012, FR-013, FR-019, SC-004, SC-005]

**Checkpoint**: User Story 3 proves each production runtime and store boundary independently, including bounded and redacted failures.

---

## Phase 6: User Story 4 - Operate and Intentionally Reset Local Data (Priority: P2)

**Goal**: Preserve both stores on normal stop and remove only exact Norvii volumes after explicit destructive confirmation.

**Independent Test**: Preserve marker data across normal stop/start, refuse unsafe reset requests, confirm the exact reset, then recreate a healthy initialized environment while PostgreSQL and Neo4j volume isolation remains proven.

### Tests for User Story 4

> Write and run these tests first; they must fail before the corresponding implementation.

- [X] T034 [US4] Write failing reset-contract tests for confirmation, exact project and volume ownership, absent targets, unexpected targets, and zero arbitrary path input in `.github/scripts/tests/test_persistence_reset.py` [FR-010, FR-017, SC-006, SC-007]

### Implementation for User Story 4

- [X] T035 [US4] Implement exact Compose project and named-volume validation plus confirmed deletion in `infra/scripts/reset-local-data.sh` until T034 passes [FR-010, FR-017]
- [X] T036 [US4] Complete non-destructive `persistence-stop` and guarded `persistence-reset` targets in `Makefile` [FR-009, FR-016, FR-017]
- [X] T037 [US4] Extend the service-backed journey to prove normal-stop retention, graph-volume-only rebuild, canonical marker isolation, confirmed full reset, and clean recreation in `infra/scripts/verify-foundation.sh` [FR-009, FR-010, FR-018, SC-006, SC-007, SC-008]

**Checkpoint**: User Story 4 is independently runnable and all destructive behavior is exact, explicit, and reproducible.

---

## Phase 7: Cross-Cutting Verification and Documentation

**Purpose**: Integrate all story evidence into CI and replace deferred global instructions with executable commands.

- [X] T038 Extend module setup by language and add an isolated service-backed persistence job with cleanup in `.github/workflows/ci.yml` [FR-014, FR-020]
- [X] T039 Make SonarQube Cloud and failure email notification depend on persistence integration results in `.github/workflows/ci.yml` [FR-020]
- [X] T040 Update executable lifecycle commands, resource expectations, security notes, and troubleshooting in `docs/operations/local-environment.md` [FR-002, FR-004, FR-013, FR-016]
- [X] T041 [P] Update Go migration and connectivity verification guidance in `docs/modules/go-api.md` [FR-011, FR-014, FR-016]
- [X] T042 [P] Update Python package and connectivity verification guidance in `docs/modules/python-ingestion.md` [FR-011, FR-014, FR-016]
- [X] T043 Execute every clean-state scenario in `specs/003-local-persistence-foundation/quickstart.md`, record measured readiness and verifier timings there, and resolve any documentation drift [FR-018, SC-001, SC-002, SC-004, SC-008]
- [X] T044 Run `make -C apps/api ci`, `make -C apps/ingestion ci`, all `.github/scripts/tests`, Compose validation, the service-backed foundation journey, repository language validation, and `git diff --check`, then record evidence in `specs/003-local-persistence-foundation/quickstart.md` [FR-014, FR-018, FR-020]
- [X] T045 Mark the feature implemented only after all requirement, task, quickstart, and convergence evidence agrees in `specs/003-local-persistence-foundation/spec.md` and `specs/003-local-persistence-foundation/tasks.md` [FR-018, FR-020]

---

## Dependencies and Execution Order

### Phase dependencies

- **Setup (Phase 1)** starts immediately.
- **Foundational Configuration (Phase 2)** depends on Phase 1 and blocks all stories.
- **User Story 1 (Phase 3)** depends on Phase 2 and provides the running stores.
- **User Story 2 (Phase 4)** depends on User Story 1 only for its external store fixture; migration behavior remains Go-owned and independently testable.
- **User Story 3 (Phase 5)** depends on User Story 1 for service-backed tests and User Story 2 for initialized canonical capability; unit tests remain independent.
- **User Story 4 (Phase 6)** depends on User Stories 1 and 2 for lifecycle and migration fixtures.
- **Cross-Cutting Verification (Phase 7)** depends on all selected user stories.

### Requirement traceability by story

- **US1**: FR-001 through FR-005, FR-008, FR-009, FR-013, FR-016; SC-001, SC-002, SC-005.
- **US2**: FR-006 through FR-009, FR-016; SC-003, SC-005, SC-006.
- **US3**: FR-008, FR-011 through FR-016, FR-019; SC-004, SC-005.
- **US4**: FR-009, FR-010, FR-016 through FR-018; SC-006 through SC-008.
- **Cross-cutting**: FR-014, FR-018, FR-020 and all affected acceptance evidence.

### Parallel opportunities

- T001 and T002 can proceed in parallel; T004 is independent of language scaffolding.
- T005 and T006 are independent red tests; T007 and T008 can then proceed in parallel.
- T010 and T011 can be written in parallel; T013 is independent once variable names are fixed.
- T017 and T018 cover distinct unit and service-backed migration boundaries.
- T024 through T027 are four independent red-test streams; T028 and T030 can proceed in parallel after their respective tests fail.
- T041 and T042 update distinct module documentation after behavior stabilizes.

## Parallel Example: User Story 3

```text
Task T024: Go verifier unit tests
Task T025: Python verifier unit tests
Task T026: Go service-backed connectivity tests
Task T027: Python service-backed connectivity tests
```

After those tests demonstrate the missing behavior:

```text
Task T028: Go persistence adapters and verifier
Task T030: Python persistence adapters and verifier
```

## Implementation Strategy

### MVP first

1. Complete Setup and Foundational Configuration.
2. Complete User Story 1.
3. Stop and validate the two-service authenticated environment independently.

### Incremental delivery

1. Add repeatable vector initialization and restart persistence through User Story 2.
2. Add independent production-driver evidence through User Story 3.
3. Add guarded lifecycle and reset behavior through User Story 4.
4. Integrate the complete journey into CI and durable documentation.

## Notes

- Tests must visibly fail for the intended missing behavior before implementation.
- Integration tests may create only disposable foundation markers and must clean them.
- Do not add product tables, HTTP endpoints, ingestion artifacts, embeddings, graph
  domain data, or client dependencies in this feature.
- Commit complete logical groups; do not create partial scaffolding-only history.
