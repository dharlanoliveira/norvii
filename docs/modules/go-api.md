# Go API Model

## Mission

`apps/api/` is the public online backend facade. It serves product APIs, validates
corpus ownership, and exposes the public SSE contract while delegating online AI
orchestration to `apps/agent/`.

## Owns

- corpus and source application use cases;
- HTTP and streaming request validation;
- public request validation and active-corpus boundary checks;
- internal agent request forwarding and public SSE translation;
- online state transitions, errors, metrics, and audit-safe traces.
- immutable corpus snapshot publication and active-release resolution.
- graph-release status inspection for a named corpus snapshot.

## Does not own

- browser rendering or client state;
- PDF and HTML extraction;
- legal-aware document normalization and chunk production;
- Python implementation details;
- vendor-specific persistence behavior inside domain rules.

## Boundary model

Public handlers validate and map transport data into application commands or
queries. Application behavior depends on small interfaces defined by the consumer.
Adapters implement databases, internal agent HTTP, and streaming protocols. Retrieval,
model providers, citation verification, abstention, MCP, and skills belong to the
Python agent or later dedicated modules, not to this facade.

The API MUST not expose database rows or provider payloads directly. Errors crossing
HTTP, stream, MCP, and ingestion boundaries use stable codes and safe messages with
internal causes preserved for diagnostics.

The facade resolves the corpus's active immutable snapshot before every chat request
and forwards that identity to the Python agent. A chat request never falls back to a
source's newest candidate document. The snapshot endpoints provide explicit
publication and immutable manifest inspection; an optimistic release version prevents
two maintainers from silently replacing each other's release.

For `graph` and `hybrid` chat strategies, the facade also resolves a ready graph release for
that same corpus and active snapshot. If no ready release exists, the request returns the safe
`graph_unavailable` outcome; it never silently falls back to vector retrieval. The release record
is inspectable at:

```text
GET /api/v1/corpora/{corpusId}/snapshots/{snapshotId}/graph-release
```

## Target organization

```text
apps/api/
|-- cmd/                     # Executable composition roots
|-- internal/
|   |-- catalog/             # Corpus catalog domain and use cases
|   |-- source/
|   |-- chat/
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

## Implemented persistence foundation

Feature 003 initializes the module with Go 1.26 and keeps vendor behavior under
`internal/platform/persistence/`. The module owns:

- the embedded tern migration path in `migrations/`;
- typed validation of the shared local persistence environment;
- narrow pgx and Neo4j adapters that execute bounded read-only verification queries;
- separate `migrate`, `migration-status`, and `verify-persistence` composition roots.

The public corpus/source endpoints and chat SSE facade are now available. Run module
quality and local connectivity independently:

```bash
make -C apps/api ci
python infra/scripts/run-with-environment.py infra/.env make -C apps/api verify-persistence
```

Service-backed tests use the `integration` build tag and remain outside the standard
module CI target because the dedicated persistence CI job owns their database
lifecycle.
