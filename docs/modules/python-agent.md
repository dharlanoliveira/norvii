# Python LangGraph Agent Model

## Mission

`apps/agent/` is the online AI orchestration service. It receives a corpus-scoped
question from the Go facade and runs a bounded LangGraph workflow for retrieval,
evidence sufficiency, model generation, citation validation, abstention, and internal
SSE events.

## Owns

- retrieval from canonical PostgreSQL and pgvector artifacts constrained to a snapshot;
- graph state transitions and bounded context assembly;
- OpenAI-compatible model adapters and provider-neutral configuration;
- evidence-only prompting, citation-marker validation, and fail-closed abstention;
- internal streaming events and redacted provider diagnostics.
- snapshot-scoped Vector and planned Hybrid retrieval composition.

## Does not own

- public browser-facing HTTP contracts;
- corpus and source mutations;
- PDF or HTML acquisition and normalization;
- direct React dependencies or Go package imports;
- Neo4j projection writes and graph-release lifecycle management.

## Boundary model

The service exposes a private loopback HTTP/SSE contract used only by the Go API. The
graph receives typed request state and emits bounded evidence and answer events. The
public API never exposes provider payloads or internal graph state.

Every request includes the active `snapshotId`. PostgreSQL retrieval joins immutable
snapshot membership before ranking chunks, so a reingested candidate cannot affect an
answer until the API explicitly publishes a new release.

Every request begins with vector retrieval. For Hybrid, the agent obtains a compact,
snapshot-scoped graph capability catalog and asks the configured model for a bounded structured
decision. The decision can select only published relationship types and canonical entity labels; it never
contains Cypher or authorizes unbounded graph access. It selects canonical entity labels from the
active graph catalog, so the question language does not need to match the graph-label language.
Neo4j traversal remains parameterized by corpus and snapshot, and every graph path returns
PostgreSQL-backed evidence locations rather than treating inferred relationships as legal
authority.

Vector evidence remains the answer baseline. A graph miss, planner failure, unavailable graph
release, or unavailable Neo4j connection is recorded as a safe stage result and does not discard
valid vector evidence. Hybrid inspection data therefore identifies the Vector, planning, and
graph stages separately, including their content-free measurements and evidence contributions.

## Verification

Run the module's formatter, Ruff, mypy, unit tests, and package build with:

```bash
make -C apps/agent ci
```

Run the PostgreSQL snapshot-isolation integration test through the configured local environment:

```bash
python infra/scripts/run-with-environment.py infra/.env make -C apps/agent test-integration
```
