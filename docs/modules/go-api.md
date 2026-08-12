# Go API Model

## Mission

`apps/api/` is the online backend. It serves product APIs and streaming chat while
orchestrating retrieval, LLM calls, MCP tools, skills, citation verification, and
online observability.

## Owns

- corpus and source application use cases;
- HTTP and streaming request validation;
- active-corpus enforcement for every retrieval operation;
- RAG, GraphRAG, and hybrid orchestration;
- LLM provider and retrieval port definitions;
- MCP tool execution and skill orchestration;
- citation verification and abstention policy;
- online state transitions, errors, metrics, and audit-safe traces.

## Does not own

- browser rendering or client state;
- PDF and HTML extraction;
- legal-aware document normalization and chunk production;
- Python implementation details;
- vendor-specific persistence behavior inside domain rules.

## Boundary model

Public handlers validate and map transport data into application commands or
queries. Application behavior depends on small interfaces defined by the consumer.
Adapters implement databases, queues, model providers, vector search, graph search,
and streaming protocols.

The API MUST not expose database rows or provider payloads directly. Errors crossing
HTTP, stream, MCP, and ingestion boundaries use stable codes and safe messages with
internal causes preserved for diagnostics.

## Target organization

```text
apps/api/
|-- cmd/                     # Executable composition roots
|-- internal/
|   |-- corpus/              # Domain and use cases by capability
|   |-- source/
|   |-- chat/
|   |-- retrieval/
|   `-- platform/            # Concrete external adapters
|-- migrations/             # Go-owned persistence migrations
`-- tests/                   # Cross-package integration and contract tests
```

Packages are named for domain capabilities. Interfaces live near their consumers.
Do not create framework-shaped `controllers`, `services`, and `repositories` trees
that scatter one feature across unrelated layers.

## Verification model

- table-driven unit tests for domain rules and transformations;
- handler tests for validation, status, errors, and streaming behavior;
- contract tests against versioned schemas;
- integration tests for database, retrieval, queue, and model adapters;
- race detector where concurrency is introduced;
- formatting, static analysis, tests, and build for affected packages.
