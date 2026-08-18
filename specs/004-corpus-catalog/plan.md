# Implementation Plan: Corpus Catalog and Ingestion

**Branch**: `004-corpus-catalog` | **Date**: 2026-08-17 | **Spec**: [spec.md](spec.md)

**Input**: Feature specification from `specs/004-corpus-catalog/spec.md`

## Summary

Deliver one real vertical corpus-to-document workflow across the production client, Go API, Python ingestion worker, and canonical PostgreSQL store. The API owns online corpus/source commands and reads; PostgreSQL supplies a leased queue; Python safely acquires PDF or HTTPS content and atomically publishes immutable revisions, complete normalized documents, and hierarchical units; React replaces runtime demonstration data with versioned HTTP adapters. Two idempotent seeds enqueue the official LGPD and GDPR pages. Chat becomes an honest unavailable state; embeddings, model extraction, Neo4j publication, RAG, and citations remain deferred.

## Technical Context

**Language/Version**: Go 1.26; Python 3.13; TypeScript 6.0 and React 19 on Node.js 24

**Primary Dependencies**: Go standard `net/http` plus pgx v5; Python psycopg 3.3.4, pypdf 6.16.1, and Trafilatura 2.2.0; existing React Router, i18next, and Vite stack

**Storage**: PostgreSQL with pgvector remains canonical; PDF binaries use `bytea`; Neo4j remains running but receives no feature data

**Testing**: Go table, handler, repository, and PostgreSQL integration tests; pytest unit/contract/integration tests; Vitest component/adapter tests; Playwright end-to-end journeys; repository language and Sonar gates

**Target Platform**: Local Linux/WSL development and GitHub Actions; evergreen desktop browsers at the existing notebook and desktop viewports

**Project Type**: Monorepo vertical web application with an online API and offline worker

**Performance Goals**: Catalog/workspace outcome within 2 seconds for 95% of local reads; supported sources up to 2 MB ingest within 90 seconds in 95% of clean runs

**Constraints**: 10 MB per origin; 20 sources per corpus; one bounded worker by default; no OCR, model, embedding, graph, or chat cost; English engineering artifacts; Portuguese only in localization and preserved legal content

**Scale/Scope**: Trusted single-user POC, two seeded corpora and two seeded URLs, no more than 20 sources per corpus, one latest ready document plus immutable history per source

## Constitution Check

*GATE: Passed before research and re-checked after design.*

- **I. Specification before implementation**: PASS. The numbered spec defines six independently testable stories, failure behavior, boundaries, and measurable outcomes. No product code changes precede approval.
- **II. Vertical features and module boundaries**: PASS. Web owns presentation, Go owns online behavior, Python owns ingestion, and PostgreSQL interactions cross modules only through the feature contract.
- **III. Evidence-grounded legal answers**: PASS. Corpus ownership is enforced for every record. The feature generates no answers or citations and disables simulated chat.
- **IV. Versioned cross-language contracts**: PASS. HTTP v1 and ingestion-work v1 contracts cover all exchanged data, errors, state transitions, leases, and compatibility.
- **V. Idiomatic, tested, maintainable code**: PASS. TDD is required at every behavior boundary; the project code-quality skill governs all touched languages.
- **VI. Reproducible and cost-bounded POC**: PASS. Dependencies are pinned, seeds are stable, limits are explicit, source hashes and pipeline versions are retained, and no model cost exists.
- **VII. Observable and safe by design**: PASS. URL address pinning, redirect revalidation, size/time bounds, PDF validation, safe failure categories, leases, and content-free logs are designed with the feature.
- **VIII. English engineering language**: PASS. Engineering artifacts remain English; preserved Portuguese legal content and locale values use the documented exceptions.

Post-design re-check: PASS. No exception or unjustified complexity was introduced; the PostgreSQL queue avoids a fourth runtime service.

## Project Structure

### Documentation (this feature)

```text
specs/004-corpus-catalog/
|-- spec.md
|-- plan.md
|-- research.md
|-- data-model.md
|-- quickstart.md
|-- contracts/
|   |-- http-api.md
|   `-- ingestion-work.md
|-- checklists/
`-- tasks.md
```

### Source Code (repository root)

```text
apps/api/
|-- cmd/server/
|-- internal/catalog/{domain,application,postgres,http}/
|-- internal/source/{domain,application,postgres,http}/
|-- internal/document/{application,postgres,http}/
|-- internal/platform/httpserver/
|-- migrations/
`-- tests/{contract,integration}/

apps/ingestion/
|-- src/norvii_ingestion/
|   |-- acquisition/
|   |-- extraction/
|   |-- domain/
|   |-- orchestration/
|   `-- publication/
`-- tests/{unit,contract,integration}/

apps/web/src/
|-- api/
|-- features/catalog/
|-- features/source-management/
|-- features/workspace/
|-- i18n/{en,pt}/
`-- research/domain/

contracts/
`-- corpus-ingestion/v1/

infra/scripts/
docs/operations/
```

**Structure Decision**: Extend the three existing production modules in place. Domain and application packages remain independent of transports. PostgreSQL adapters implement module-owned ports. Feature design contracts are promoted to `contracts/corpus-ingestion/v1/` during implementation and verified by every provider and consumer. The prototype is unchanged.

## Complexity Tracking

No constitution violations require justification.

## Norvii Implementation Requirements

### Module Impact

| Module | Change | Responsibility | Verification |
| --- | --- | --- | --- |
| `prototypes/web/` | No change | Preserve the approved UX baseline | Existing CI only |
| `apps/web/` | Change | HTTP adapters, management UI, lifecycle polling, real tree/viewer, unavailable chat | Vitest, accessibility, Playwright, build budget |
| `apps/api/` | Change | Corpus/source commands, reads, binary delivery, schema migration, server | Unit, contract, repository, integration tests |
| `apps/ingestion/` | Change | Lease worker, safe acquisition, extraction, hierarchy, atomic publication | Unit, contract, integration tests |
| `contracts/` | Change | Versioned HTTP and ingestion-work schemas | Provider/consumer contract validation |
| `infra/` | Change | Bootstrap API and worker; verification of initial ingestion | Clean-state integration journey |

### Boundaries and Constraints

- **Cost limits**: 10 MB origin, 20 sources per corpus, one worker by default, three-minute end-to-end target, no model/token cost.
- **Prototype baseline**: Features 001 and 002 remain the layout/accessibility baseline. Runtime fixtures and prepared chat are intentionally removed.
- **Public contracts**: Add `corpus-ingestion/v1`; additive compatible changes remain v1, breaking changes require v2. Contract tests cover Go provider, Python work consumer/publisher, and TypeScript consumer.
- **Persistence**: One ordered migration adds canonical tables, constraints, indexes, seeds, and rollback sections. PDF bytes are separate from hot metadata. Immutable revisions are retained.
- **Ingestion artifacts**: Pipeline version plus origin, extraction, document, and unit hashes identify idempotent output. Publication is one PostgreSQL transaction.
- **Streaming**: N/A. Poll source state with bounded interval; chat streaming remains deferred.
- **Corpus boundary and citations**: Every API path and repository query includes corpus ownership. No citation or legal answer exists in this feature.
- **Security and privacy**: Trusted local actor does not remove validation. HTTPS only, public-address pinning per redirect, TLS hostname verification, no environment proxies, size/time/redirect bounds, PDF signature/media validation, safe filenames, and content-free diagnostics are mandatory.
- **Observability**: Structured English logs include operation, corpus/source/attempt identifiers, state, duration, byte counts, unit counts, pipeline version, and safe category; never URLs with credentials, binaries, or complete text.
- **Local environment**: Existing Compose stores remain. Bootstrap runs migrations, API, worker, and web; logs stay under `.log/`; verification waits for initial terminal states and accepts explicit remote-source failure only in the offline scenario.

### Repository Paths

```text
apps/web/
apps/api/
apps/ingestion/
contracts/
infra/
docs/
specs/004-corpus-catalog/
```
