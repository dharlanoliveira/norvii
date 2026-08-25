# Feature Map

This map proposes the order in which Norvii capabilities become independently
demonstrable. Numbers are reserved when `speckit-specify` creates a feature; this
document does not create feature branches or implementation commitments by itself.

## Delivery sequence

| Order | Candidate feature | Demonstrable outcome | Likely modules | Depends on |
| --- | --- | --- | --- | --- |
| [001](../../specs/001-product-experience-prototype/spec.md) | Product experience prototype | A reviewer navigates the bilingual core Norvii journeys and states in a deterministic React prototype without backend services | Prototype web | None |
| [002](../../specs/002-production-web-experience/spec.md) | Production web experience | A reviewer uses the approved bilingual corpus research experience from the independently owned production React client | Web | 001 |
| [003](../../specs/003-local-persistence-foundation/spec.md) | Local persistence foundation | A contributor starts both required stores and verifies initialization plus Go and Python connectivity from documented commands | Infra, API shell, ingestion shell | 002 |
| [004](../../specs/004-corpus-catalog/spec.md) | Corpus catalog and ingestion | A user manages corpora and PDF or URL sources, observes ingestion, and browses real versioned documents and sections | Web, API, ingestion, contracts, infra | 002, 003 |
| 005 | Grounded RAG chat | A user asks a question and receives a streamed, cited answer or explicit abstention | Web, API, contracts, infra | 004 |
| 006 | Citation navigation and inspection | A user opens cited evidence and an evaluator inspects retrieval, latency, and token use | Web, API, contracts | 002, 005 |
| 007 | Portuguese and English snapshots | A user switches between two isolated, reproducible curated legal corpora | Ingestion, API, Web | 004, 005 |
| 008 | GraphRAG and hybrid retrieval | An evaluator compares vector, graph, and hybrid evidence paths | Ingestion, API, Web, contracts, infra | 006, 007 |
| [009](../../specs/009-planned-hybrid-retrieval/spec.md) | Planned hybrid retrieval | A researcher uses vector-first retrieval with graph augmentation planned only when it can add snapshot-scoped evidence | Agent, API, Web, contracts | 008 |
| 010 | MCP research tools and skills | An evaluator invokes explicit research tools and reusable evidence workflows | API, contracts | 006, 009 |
| 011 | Evaluation showcase | A contributor runs versioned quality and cost evaluations and views comparable results | Ingestion, API, Web | 007, 009, 010 |

## Slicing rules

- Keep P1 of each feature small enough to demonstrate without later features.
- Do not create one feature per technical module. The module documents define stable
  ownership; feature specifications define behavior.
- A dependency indicates required behavior, not permission to copy requirements from
  another feature.
- Split a candidate when it cannot be independently accepted or when its plan grows
  unrelated contracts and data models.
- Merge candidates only when neither produces meaningful standalone value.
- Production UI features MUST link to the approved prototype baseline and record intentional differences.

Feature 004 intentionally consolidates the earlier catalog, source-management,
ingestion, and workspace-integration candidates. The combined slice avoids a mixed
runtime where an authoritative catalog opens simulated documents, while keeping
model-backed retrieval and graph behavior in later independently measurable features.

## Active foundation sequence

The verified [product experience prototype specification](../../specs/001-product-experience-prototype/spec.md) establishes the approved, executable English and Portuguese baseline for the core journeys and required states defined in the [prototyping workflow](../development/prototyping.md). It uses deterministic fixtures and does not create production modules, backend services, databases, or public contracts.

After prototype approval, [Feature 002](../../specs/002-production-web-experience/spec.md) independently implements the approved baseline under `apps/web/` with production-owned code and deterministic data. It has no backend, database, ingestion, or Neo4j dependency.

[Feature 003](../../specs/003-local-persistence-foundation/spec.md) now owns the PostgreSQL with pgvector plus standalone Neo4j Community environment selected by [ADR 0005](../decisions/0005-postgresql-and-neo4j-persistence.md). It introduces only the service, initialization, and module-connectivity foundation needed before later vertical data features.
