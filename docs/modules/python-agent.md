# Python LangGraph Agent Model

## Mission

`apps/agent/` is the online AI orchestration service. It receives a corpus-scoped
question from the Go facade and runs a bounded LangGraph workflow for retrieval,
evidence sufficiency, model generation, citation validation, abstention, and internal
SSE events.

## Owns

- retrieval from canonical PostgreSQL and pgvector artifacts;
- graph state transitions and bounded context assembly;
- OpenAI-compatible model adapters and provider-neutral configuration;
- evidence-only prompting, citation-marker validation, and fail-closed abstention;
- internal streaming events and redacted provider diagnostics.

## Does not own

- public browser-facing HTTP contracts;
- corpus and source mutations;
- PDF or HTML acquisition and normalization;
- direct React dependencies or Go package imports;
- Neo4j projection writes until GraphRAG is explicitly specified.

## Boundary model

The service exposes a private loopback HTTP/SSE contract used only by the Go API. The
graph receives typed request state and emits bounded evidence and answer events. The
public API never exposes provider payloads or internal graph state.

## Verification

Run the module's formatter, Ruff, mypy, unit tests, and package build with:

```bash
make -C apps/agent ci
```
