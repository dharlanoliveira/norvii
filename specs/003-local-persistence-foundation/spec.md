# Feature Specification: Local Persistence Foundation

**Feature Branch**: `003-local-persistence-foundation`

**Created**: 2026-08-13

**Status**: Implemented

**Input**: User description: "Continue implementation with the next vertical capability: a reproducible local persistence foundation for PostgreSQL with vector support and a standalone Neo4j graph projection, including health, initialization, and Go and Python connectivity evidence."

## User Scenarios & Testing *(mandatory)*

### User Story 1 - Start the persistence environment (Priority: P1)

A contributor with the documented prerequisites starts Norvii's complete local persistence environment through one maintained command and can see when each required service is ready.

**Why this priority**: Later corpus, source, ingestion, retrieval, and GraphRAG capabilities need a reproducible persistence baseline before they can be implemented or reviewed independently.

**Independent Test**: Follow the quickstart from a clean checkout, start the default environment, and verify that both required stores become healthy without manual database administration.

**Acceptance Scenarios**:

1. **Given** a clean checkout and the documented prerequisites, **When** the contributor runs the start command with valid local configuration, **Then** the canonical store and graph projection store start together and report healthy within the documented readiness window.
2. **Given** no local environment file exists, **When** the contributor follows the setup instructions, **Then** a versioned example identifies every required value without containing a usable secret.
3. **Given** one service cannot become healthy, **When** readiness is inspected, **Then** the failing service is identifiable and the contributor receives an actionable diagnostic command.
4. **Given** the environment is healthy, **When** the contributor stops it normally and starts it again, **Then** intentionally persisted local data remains available.

---

### User Story 2 - Initialize and verify canonical storage (Priority: P1)

A contributor applies the maintained database initialization path and verifies that vector-capable canonical storage is ready for later feature-owned schemas.

**Why this priority**: A running container is not sufficient evidence that migrations, required extensions, credentials, and persistence behavior are usable by application modules.

**Independent Test**: Start the environment, apply initialization from a clean data volume, verify the migration state and vector capability, repeat initialization, and confirm that the result remains valid without duplicate changes.

**Acceptance Scenarios**:

1. **Given** a new canonical store, **When** initialization runs, **Then** the required extension and migration ledger are present and their state can be verified through a maintained command.
2. **Given** initialization already completed, **When** the same initialization command runs again, **Then** it succeeds without duplicating or corrupting state.
3. **Given** initialization cannot complete, **When** the command exits, **Then** it returns a failure and identifies the failed migration or prerequisite without exposing credentials.
4. **Given** the canonical store is initialized, **When** a marker record is written and the environment is restarted normally, **Then** the marker remains available.

---

### User Story 3 - Prove module connectivity (Priority: P1)

A contributor runs independent Go API and Python ingestion checks that prove each production runtime can authenticate to its permitted local stores through narrow infrastructure adapters.

**Why this priority**: The foundation must demonstrate that the selected services work with the project's actual runtime boundaries, not only with administrator command-line tools.

**Independent Test**: With healthy initialized services, run the Go and Python module verification commands and observe successful canonical-store and graph-store connectivity without product records or business workflows.

**Acceptance Scenarios**:

1. **Given** healthy initialized stores, **When** the Go module connectivity check runs, **Then** it verifies its required connections and exits successfully without creating product data.
2. **Given** healthy initialized stores, **When** the Python module connectivity check runs, **Then** it verifies its required connections and exits successfully without publishing ingestion artifacts.
3. **Given** invalid credentials or an unavailable store, **When** either module check runs, **Then** it fails within the documented timeout and reports a safe, actionable error.
4. **Given** either module is verified independently, **When** its standard quality command runs, **Then** it does not require the React client or import another module's implementation.

---

### User Story 4 - Operate and intentionally reset local data (Priority: P2)

A contributor can inspect health, stop services without data loss, and intentionally reset only Norvii's named POC data through explicit documented commands.

**Why this priority**: Repeatable reset and recovery are necessary for demonstrations and integration tests, but destructive cleanup must remain deliberate and tightly scoped.

**Independent Test**: Create marker data, stop and restart without data removal, then invoke the documented reset flow and confirm that only the named Norvii persistence data is recreated cleanly.

**Acceptance Scenarios**:

1. **Given** both services contain marker data, **When** the normal stop command runs, **Then** services stop without deleting either persisted data set.
2. **Given** persisted marker data exists, **When** the contributor explicitly confirms the reset procedure, **Then** only the documented Norvii local data is removed and the environment can be initialized again.
3. **Given** a reset target is absent, ambiguous, or outside the expected project scope, **When** reset validation runs, **Then** deletion does not proceed and the command fails safely.
4. **Given** the environment has been reset, **When** the quickstart is repeated, **Then** all health, initialization, and connectivity checks pass from clean state.

### Edge Cases

- A required local port is already in use before startup.
- One store becomes healthy while the other remains unhealthy or repeatedly restarts.
- Existing named data was created by an incompatible local configuration.
- Initialization is interrupted after some migrations complete.
- Credentials contain characters that require environment-file quoting.
- A contributor runs connectivity checks before readiness or initialization completes.
- The graph data volume is removed while canonical data remains intact.
- The canonical data volume is unavailable while a stale graph projection still exists.
- The host architecture differs from the primary development architecture.
- A normal stop is confused with the explicitly destructive reset operation.

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: The project MUST provide one documented default command that starts exactly the canonical persistence store and the standalone graph projection store required by the accepted persistence decision.
- **FR-002**: Every required service MUST use a deliberate immutable version, a stable service identity, a readiness check, and an explicit local resource expectation.
- **FR-003**: Both required services MUST start in the default environment without an optional profile or an unrelated supporting service.
- **FR-004**: Required local configuration MUST be documented in a versioned example that contains no usable secret, while actual local values remain outside version control.
- **FR-005**: Readiness MUST verify that each store accepts authenticated operations, not only that its process or port exists.
- **FR-006**: The canonical store MUST expose the vector capability selected by the accepted persistence decision after maintained initialization completes.
- **FR-007**: Initialization MUST maintain an inspectable migration history and MUST be safe to repeat after partial or complete success.
- **FR-008**: Initialization and verification failures MUST return nonzero results with safe messages that identify the affected service, migration, or prerequisite.
- **FR-009**: Normal stop and restart operations MUST preserve canonical and graph data in separately owned named persistence areas.
- **FR-010**: Removing graph projection data MUST NOT remove or alter canonical data.
- **FR-011**: The project MUST provide independent Go and Python connectivity checks using the configuration paths and database drivers intended for their production modules.
- **FR-012**: Connectivity checks MUST authenticate, perform bounded read-only or disposable verification operations, enforce a timeout, and create no corpus, source, document, ingestion, or graph artifact.
- **FR-013**: Connectivity checks and service diagnostics MUST NOT log credentials, connection strings containing credentials, or stored content.
- **FR-014**: Each production module introduced by this feature MUST own its dependency manifest, formatting, static analysis, tests, build or packaging check, and documented verification entry point.
- **FR-015**: Production modules MUST remain independently testable and MUST NOT import another production module's internal implementation.
- **FR-016**: The project MUST document commands to start services, inspect readiness, apply or verify initialization, stop without deleting data, run all affected module checks, and intentionally reset local POC data.
- **FR-017**: The reset flow MUST require an explicit destructive action, identify the exact project-owned data targets, and refuse ambiguous or out-of-scope targets.
- **FR-018**: A clean reset followed by the maintained quickstart MUST reproduce the same healthy, initialized, connectivity-verified state without manual store administration.
- **FR-019**: The foundation MUST remain usable without model credentials, corpus files, network source capture, embeddings, product records, or application business endpoints.
- **FR-020**: Continuous integration MUST validate configuration structure, initialization behavior, module quality contracts, and service-backed connectivity in a manner that exposes the failing module or service.

### Key Entities

- **Local Persistence Environment**: The complete default set of required stores, configuration, health state, and operator commands for one contributor checkout.
- **Canonical Store**: The authoritative local persistence boundary that will later own transactional records, binaries, document versions, vectors, and derived artifacts.
- **Graph Projection Store**: A separately persisted, rebuildable local projection that never becomes the authority for canonical records.
- **Migration**: An ordered, uniquely identified, repeatable change with inspectable application state and failure reporting.
- **Connectivity Check**: A bounded module-owned operation that proves configuration, authentication, driver compatibility, and store readiness without implementing product behavior.
- **Named Persistence Area**: A project-owned local data target whose preservation and intentional deletion behavior are explicitly documented.

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: A contributor with prerequisites installed can move from a clean checkout to two healthy initialized stores in no more than 10 minutes, excluding the first image download.
- **SC-002**: Both required stores report ready within 2 minutes on the reference development environment after their artifacts are locally available.
- **SC-003**: Initialization, normal restart, and repeated initialization pass in 100% of automated foundation journeys without duplicate migration state.
- **SC-004**: Go and Python connectivity checks each complete within 10 seconds after readiness and pass in 100% of supported local and continuous-integration runs.
- **SC-005**: Invalid credentials, unavailable stores, and failed initialization produce a nonzero result and an actionable service-scoped message in 100% of automated failure tests, with no detected credential disclosure.
- **SC-006**: Marker data survives 100% of normal stop and restart tests, while an explicitly confirmed reset removes 100% of project-owned marker data and no target outside the documented environment.
- **SC-007**: Deleting and rebuilding graph projection data leaves canonical marker data unchanged in 100% of isolation tests.
- **SC-008**: Another contributor can complete the quickstart and all module verification commands without manual database administration, product data, model usage, or source-network access.

## Assumptions

- Contributors have a current Docker Engine with Compose support plus the pinned Go and Python toolchains selected during planning.
- The first image download depends on the contributor's network and is excluded from readiness timing.
- Local credentials protect accidental cross-project access but are not production secret-management credentials.
- The feature creates only the smallest module shells and infrastructure adapters required to prove connectivity; product repositories, corpus tables, source workflows, and public endpoints belong to later features.
- The canonical and graph stores use separate persistence areas because graph data must be independently rebuildable.
- Destructive reset remains an explicit operator action and is never part of normal startup, shutdown, tests, or migration commands.
- Current Linux container hosts are the reference environment; additional host notes may be documented when planning research identifies a material difference.

## Norvii Feature Requirements *(mandatory)*

### Scope and Boundaries

- **In scope**: Reproducible local PostgreSQL with pgvector and standalone Neo4j Community services, pinned versions, authenticated health checks, named persistence, environment example, initialization and migration verification, minimal Go and Python module shells, database-driver connectivity checks, module quality commands, CI integration, safe operator commands, and quickstart evidence.
- **Out of scope**: Corpus and source tables, PDF storage, CRUD APIs, ingestion workflows, document extraction, embeddings, vector queries, graph projection publication, GraphRAG traversal, web-client integration, model calls, authentication, production deployment, backup automation, clustering, replication, and monitoring services.

### Prototype Baseline *(mandatory for production UI features)*

- **Approved baseline**: N/A. This feature adds no user interface and does not change the approved production web experience.
- **Intentional differences**: N/A. The production client remains independently runnable without these services until a later vertical feature introduces an API dependency.

### Evidence and Corpus Boundaries *(mandatory for retrieval, chat, or citations)*

- **Active corpus constraint**: No retrieval or corpus data is introduced. Connectivity checks create no product entity and therefore cannot mix corpus data.
- **Evidence behavior**: No answer, citation, model, or evidence behavior is introduced; those constitutional rules remain enforced by the existing client demonstration and later retrieval features.
