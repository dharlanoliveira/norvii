# Implementation Plan: Local Persistence Foundation

**Branch**: `003-local-persistence-foundation` | **Date**: 2026-08-13 | **Spec**: [spec.md](spec.md)

**Input**: Feature specification from `specs/003-local-persistence-foundation/spec.md`

## Summary

Create the smallest reproducible persistence foundation that proves Norvii's accepted
database topology. A root orchestration contract will start digest-pinned PostgreSQL
18 with pgvector 0.8.6 and standalone Neo4j Community 2026.06.0 from
`infra/compose.yaml`. Go owns the PostgreSQL migration path and verifies both stores
through production drivers. Python independently verifies both stores through its
own typed, class-based adapters. Service-backed CI and a deliberately confirmed reset
journey prove initialization, restart persistence, graph-volume isolation, safe
failure, and clean reproduction without introducing product records.

## Technical Context

**Language/Version**: Go 1.26.0 language level with Go 1.26.5 CI toolchain; Python 3.13; Docker Compose specification

**Primary Dependencies**: pgx 5.10.0, Neo4j Go Driver 6.2.0, tern 2.4.2, psycopg 3.3.4, Neo4j Python Driver 6.2.0, pytest 9.1.1, Ruff 0.16.3, mypy 2.3.0, uv

**Storage**: PostgreSQL 18 with pgvector 0.8.6 as canonical storage; Neo4j Community 2026.06.0 as a separately persisted graph projection

**Testing**: Go table-driven unit tests plus `go test -race`, `go vet`, migration integration tests; Python pytest unit and integration tests plus Ruff and mypy; Docker Compose configuration and service-backed lifecycle tests

**Target Platform**: Linux containers on amd64 and arm64; native Go and Python commands on contributor hosts and GitHub-hosted Ubuntu runners

**Project Type**: Monorepo persistence infrastructure with independent Go command and Python package entry points

**Performance Goals**: Both stores ready within 2 minutes after images are present; each runtime connectivity check completes within 10 seconds; clean setup completes within 10 minutes excluding image download

**Constraints**: No product schema or business endpoints; no model, corpus, or source-network dependency; credentials never logged; normal stop preserves data; reset targets only two exact project-owned volumes; PostgreSQL and Neo4j remain independently removable

**Scale/Scope**: One contributor-local POC environment, two database containers, one initialization migration, two module connectivity verifiers, and one service-backed CI journey

## Constitution Check

*GATE: Passed before research and re-checked after design.*

| Principle | Plan evidence | Result |
| --- | --- | --- |
| I. Specification Before Implementation | Feature 003 has testable scenarios, a requirements checklist, this plan, and will receive dependency-ordered tasks before code changes. | Pass |
| II. Vertical Features and Explicit Module Boundaries | `infra/` owns services, `apps/api/` owns migrations and Go adapters, and `apps/ingestion/` owns Python adapters; neither module imports the other. | Pass |
| III. Evidence-Grounded Legal Answers | No retrieval, answer, corpus, or citation behavior is introduced. | N/A |
| IV. Versioned Cross-Language Contracts | Feature-local environment and command contracts define the shared configuration without making either runtime the schema source. | Pass |
| V. Idiomatic, Tested, Maintainable Code | TDD tasks cover configuration, adapters, failures, migrations, and lifecycle behavior; each module owns formatting, analysis, tests, and build checks. | Pass |
| VI. Reproducible and Cost-Bounded POC | Images and dependencies are pinned; only two required stores run; explicit memory expectations and a no-model boundary keep the POC bounded. | Pass |
| VII. Observable and Safe by Design | Authenticated health checks, bounded timeouts, redacted diagnostics, exact-volume reset validation, and CI failure attribution are designed with the feature. | Pass |
| VIII. English as the Engineering Language | All feature artifacts, code, configuration, commands, logs, and tests are English and repository language validation remains mandatory. | Pass |

Post-design re-check: the data model introduces no product ownership conflict; the
environment contract is language-neutral; quickstart reset steps are explicit and
scoped; no constitutional exception is required.

## Project Structure

### Documentation (this feature)

```text
specs/003-local-persistence-foundation/
|-- spec.md
|-- plan.md
|-- research.md
|-- data-model.md
|-- quickstart.md
|-- contracts/
|   `-- local-persistence-environment.md
|-- checklists/
|   `-- requirements.md
`-- tasks.md
```

### Source Code (repository root)

```text
Makefile
apps/
|-- api/
|   |-- cmd/
|   |   |-- migrate/
|   |   `-- verify-persistence/
|   |-- internal/platform/persistence/
|   |-- migrations/
|   |-- tests/integration/
|   |-- go.mod
|   |-- go.sum
|   `-- Makefile
`-- ingestion/
    |-- src/norvii_ingestion/
    |   `-- publication/persistence/
    |-- tests/
    |   |-- unit/
    |   `-- integration/
    |-- pyproject.toml
    |-- uv.lock
    `-- Makefile
infra/
|-- compose.yaml
|-- .env.example
`-- scripts/
    |-- inspect-health.sh
    |-- reset-local-data.sh
    `-- verify-foundation.sh
.github/workflows/ci.yml
docs/operations/local-environment.md
```

**Structure Decision**: Preserve the accepted three-module monorepo. Root Make targets
orchestrate contributor journeys while delegating language checks to module-owned
Makefiles. Infrastructure scripts operate only on the Compose project and never
contain application domain behavior. The initial migration remains Go-owned under
`apps/api/migrations/`; Python consumes the shared environment contract but owns its
driver construction and verification behavior.

## Complexity Tracking

No constitutional violations require justification.

## Norvii Implementation Requirements

### Module Impact

| Module | Change | Responsibility | Verification |
| --- | --- | --- | --- |
| `prototypes/web/` | No change | Approved prototype remains isolated. | Existing module CI only |
| `apps/web/` | No change | Production client remains independently runnable with fixtures. | Existing module CI only |
| `apps/api/` | Change | Own initial PostgreSQL migration plus Go configuration and connectivity adapters for both stores. | Format, vet, unit tests, race tests, build, migration and connectivity integration tests |
| `apps/ingestion/` | Change | Own typed Python configuration and class-based connectivity adapters for both stores. | Ruff format/lint, mypy, pytest, package build, connectivity integration tests |
| `contracts/` | No change | No durable product payload is exchanged yet. | N/A |
| `infra/` | Change | Own digest-pinned services, health checks, volumes, lifecycle diagnostics, and reset guard. | Compose validation and service-backed lifecycle journey |

### Boundaries and Constraints

- **Cost limits**: No model or token use. Default service limits target at most 512 MiB for PostgreSQL and 1.5 GiB for Neo4j; integration data is disposable and minimal.
- **Prototype baseline**: N/A; no UI is changed.
- **Public contracts**: Feature-local environment variables and command exit behavior are defined in `contracts/local-persistence-environment.md`; no HTTP or streaming contract changes.
- **Persistence**: One PostgreSQL migration enables `vector`; tern maintains the migration ledger. No product table, binary, retention policy, or graph domain schema is introduced. Rollback is a deliberate local reset because extension removal could affect later data.
- **Ingestion artifacts**: None are created or published.
- **Streaming**: N/A.
- **Corpus boundary and citations**: No corpus or evidence operation exists in this feature.
- **Security and privacy**: Actual `.env` values remain ignored; configuration is passed as discrete fields rather than credential-bearing URLs; errors name only service and failure class; health and verification use authentication; no stored content is logged.
- **Observability**: Commands expose service-scoped readiness, migration version, and safe failure context through exit status and concise diagnostics. Product metrics and tracing are deferred.
- **Local environment**: `infra/compose.yaml` starts both default services with digest-pinned multi-architecture images, authenticated health checks, separate named volumes, bounded resources, migration commands, and an explicitly confirmed reset.

### Repository Paths

Only the modules marked as changed above receive implementation changes. Durable
operator documentation is updated only where the feature replaces previously
deferred commands with executable ones.
