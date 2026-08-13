# Feature Map

This map proposes the order in which Norvii capabilities become independently
demonstrable. Numbers are reserved when `speckit-specify` creates a feature; this
document does not create feature branches or implementation commitments by itself.

## Delivery sequence

| Order | Candidate feature | Demonstrable outcome | Likely modules | Depends on |
| --- | --- | --- | --- | --- |
| [001](../../specs/001-product-experience-prototype/spec.md) | Product experience prototype | A reviewer navigates the bilingual core Norvii journeys and states in a deterministic React prototype without backend services | Prototype web | None |
| 002 | Local project foundation | A contributor can start required backing services and run module checks from documented commands | Infra, all production module shells | 001 |
| 003 | Corpus catalog | A user can create, edit, list, select, and disable a corpus | Web, API, contracts | 001, 002 |
| 004 | Corpus source management | A user can add a PDF or official URL and see its processing state | Web, API, contracts | 001, 003 |
| 005 | Source ingestion | A pending source becomes a versioned, validated set of searchable artifacts | Ingestion, API, contracts, infra | 004 |
| 006 | Corpus workspace | A user opens one corpus and browses only its ready sources in the left panel | Web, API | 001, 003, 004 |
| 007 | Grounded RAG chat | A user asks a question and receives a streamed, cited answer or explicit abstention | Web, API, contracts, infra | 001, 005, 006 |
| 008 | Citation navigation and inspection | A user opens cited evidence and an evaluator inspects retrieval, latency, and token use | Web, API, contracts | 001, 007 |
| 009 | Portuguese and English snapshots | A user switches between two isolated, reproducible legal corpora | Ingestion, API, Web | 005, 007 |
| 010 | GraphRAG and hybrid retrieval | An evaluator compares vector, graph, and hybrid evidence paths | Ingestion, API, Web, contracts, infra | 008, 009 |
| 011 | MCP research tools and skills | An evaluator invokes explicit research tools and reusable evidence workflows | API, contracts | 008, 010 |
| 012 | Evaluation showcase | A contributor runs versioned quality and cost evaluations and views comparable results | Ingestion, API, Web | 009, 010, 011 |

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

## First active feature

The active [product experience prototype specification](../../specs/001-product-experience-prototype/spec.md) establishes an approved, executable English and Portuguese baseline for the core journeys and required states defined in the [prototyping workflow](../development/prototyping.md). It uses deterministic fixtures and does not create production modules, backend services, databases, or public contracts.

After prototype approval, `002-production-web-experience` independently implements the approved baseline under `apps/web/` with production-owned code and deterministic data. It has no backend, database, ingestion, or Neo4j dependency. A later local project foundation feature will pin and deliver the PostgreSQL with pgvector plus standalone Neo4j Community environment selected by [ADR 0005](../decisions/0005-postgresql-and-neo4j-persistence.md) before a vertical slice needs those services.
