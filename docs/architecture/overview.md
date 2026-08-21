# System Architecture

## Overview

Norvii has four application modules: a React client, a public Go API facade, an online Python LangGraph agent, and an offline Python ingestion service. They communicate through versioned contracts and share only selected backing services.

These are production modules under `apps/`. The product experience is validated first in the isolated React prototype under `prototypes/web/`; production modules never depend on it.

```mermaid
flowchart LR
    U[User] --> W[React SPA]
    W --> A[assistant-ui]
    A --> K[AI SDK message model]
    K --> B[Go API facade]
    B --> X[Python LangGraph agent]
    X --> R[Retrieval and grounding graph]
    R --> V[PostgreSQL pgvector index]
    R --> G[Neo4j graph projection]
    X --> L[LLM provider]
    L --> C[Citation verification]
    C --> X
    B -->|structured stream| K

    F[Official sources] --> I[Python ingestion]
    D[Canonical PostgreSQL storage] --> I
    I --> N[Versioned artifacts]
    N --> V
    N --> P[Graph projector]
    P --> G
    I --> D

    X --> D
    X --> M[MCP tools and skills later]
```

Detailed ownership lives in the module models:

- [Web client](../modules/web-client.md)
- [Go API](../modules/go-api.md)
- [Python LangGraph agent](../modules/python-agent.md)
- [Python ingestion](../modules/python-ingestion.md)

The [repository structure](repository-structure.md) defines source roots and dependency direction. The [contract registry](../../contracts/README.md) defines how data crosses language boundaries.

## Persistence topology

[ADR 0005](../decisions/0005-postgresql-and-neo4j-persistence.md) selects PostgreSQL with pgvector as the canonical system of record and one standalone Neo4j Community instance as a derived GraphRAG projection. The default local Docker Compose profile starts both services. Exact image versions and initialization contracts belong to the persistence foundation feature and must be pinned before implementation.

The React client has no direct database access. The Go API validates public requests and proxies an internal SSE contract to the Python agent. The agent owns online retrieval, model calls, grounding validation, and abstention. Python ingestion publishes canonical artifacts to PostgreSQL and later updates the graph projection through an idempotent, checkpointed operation.

PostgreSQL owns corpora, sources, source revisions, PDF binaries, URL origins, complete normalized document versions, hierarchical document units, retrieval fragments, embeddings, semantic extractions, evidence spans, and ingestion state. Neo4j stores versioned nodes and relationships that reference those canonical identifiers and evidence locations.

```mermaid
flowchart LR
    S[Source] --> R[Source revision]
    R --> D[Complete document version]
    D --> U[Hierarchical document units]
    U --> F[Retrieval fragments]
    F --> E[Embeddings]

    R --> X[Extraction run]
    X --> K[Statements, entities, and events]
    K --> P[Evidence spans]
    P --> U
```

Graph publication does not use a distributed transaction. PostgreSQL becomes authoritative first; the graph projection may lag temporarily and can be retried or rebuilt from canonical records.

## Ingestion flow

The ingestion pipeline runs outside the chat request path. The online agent is a separate Python process so LangGraph can model retrieval, generation, validation, and abstention as explicit state transitions without coupling the Go facade to a graph framework.

1. Claim a `pending` source and mark it `processing`.
2. Read the stored binary for a PDF or safely capture content for a URL.
3. Record official metadata, hash, and capture date.
4. Create a complete normalized document version while preserving legal structure.
5. Normalize headings, notes, articles, and references.
6. Split text along legal units instead of fixed size alone.
7. Generate embeddings when vector retrieval is enabled.
8. Extract evidence-backed statements, allegations, timeline events, entities, and relationships when GraphRAG is enabled.
9. Validate document hierarchy, evidence spans, counts, references, and representative text samples.
10. Atomically publish canonical PostgreSQL artifacts and mark the source `ready`, or record a classified error and mark it `failed`.
11. Project eligible artifacts into Neo4j and checkpoint publication so retries do not create duplicate active nodes or relationships.

The dispatch mechanism remains an [open decision](../decisions/backlog.md).

## Retrieval strategies

### Vector RAG

- Search semantically across chunks.
- Filter by corpus and optionally by language, document, and type.
- Rerank only when evaluation justifies the added dependency and cost.
- Limit answer context to the strongest evidence.

### GraphRAG

- Identify question entities.
- Traverse relations between provisions, concepts, actors, and obligations.
- Retrieve source passages attached to the relevant graph path.
- Synthesize from the evidence-backed subgraph.

### Hybrid retrieval

- Use vector search to locate initial evidence.
- Expand through the graph for references, exceptions, and related documents.
- Rank combined evidence before constructing model context.

## Initial legal graph

Start with the smallest schema justified by evaluation questions.

Node candidates:

- `Corpus`, `Document`, `Article`, `Section`, and `Recital`;
- `LegalConcept`, `Actor`, `Right`, `Obligation`, `Authority`, `Sanction`, and `Deadline`.

Relation candidates:

- `CONTAINS`, `DEFINES`, `REFERS_TO`, `AMENDS`, and `REGULATED_BY`;
- `GRANTS_RIGHT_TO`, `IMPOSES_OBLIGATION_ON`, `EXCEPTION_TO`, `ENFORCED_BY`, `HAS_DEADLINE`, and `INTERPRETS`.

Add a node or relation type only when an evaluation question requires it.

## Durable architecture decisions

- [Application module boundaries](../decisions/0001-three-module-architecture.md)
- [Corpus and source model](../decisions/0002-corpus-and-source-model.md)
- [Spec-driven delivery](../decisions/0003-spec-driven-delivery.md)
- [Executable React prototype](../decisions/0004-executable-react-prototype.md)
- [PostgreSQL and Neo4j persistence](../decisions/0005-postgresql-and-neo4j-persistence.md)

Technology selections that remain unresolved are tracked in the [decision backlog](../decisions/backlog.md).
