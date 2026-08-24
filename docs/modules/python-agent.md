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
- snapshot-scoped vector, graph, and hybrid retrieval composition.

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

Graph and hybrid requests additionally receive a ready graph-release ID from the API. Neo4j
traversal is parameterized by corpus, snapshot, and release identity; every graph path returns
PostgreSQL-backed evidence locations rather than treating inferred relationships as legal
authority. Hybrid results retain vector and graph contributions as separate structured inspection
data. If the graph release is absent, failed, or insufficient, the agent emits a safe
strategy-specific outcome without fallback.

## Verification

Run the module's formatter, Ruff, mypy, unit tests, and package build with:

```bash
make -C apps/agent ci
```

Run the PostgreSQL snapshot-isolation integration test through the configured local environment:

```bash
python infra/scripts/run-with-environment.py infra/.env make -C apps/agent test-integration
```
