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
- MCP research tools and reusable prompts through a separate local or containerized
  transport entry point.
- fixed-snapshot, non-streaming execution of one evaluation case for the managed API
  worker.

## Does not own

- public browser-facing HTTP contracts;
- corpus and source mutations;
- PDF or HTML acquisition and normalization;
- direct React dependencies or Go package imports;
- Neo4j projection writes and graph-release lifecycle management.
- evaluation dataset import, review, publication, preflight, scoring, run persistence,
  or maintainer authorization.

## Boundary model

The service exposes a private loopback HTTP/SSE contract used only by the Go API. The
graph receives typed request state and emits bounded evidence and answer events. The
public API never exposes provider payloads or internal graph state.

The MCP entry point is an additional read-only transport. In a container it uses
Streamable HTTP; for development it can use stdio. It resolves the same active
immutable snapshot for every corpus-scoped invocation and does not expose arbitrary
database or graph queries.

Every request includes the active `snapshotId`. PostgreSQL retrieval joins immutable
snapshot membership before ranking chunks, so a reingested candidate cannot affect an
answer until the API explicitly publishes a new release.

Every request begins with vector retrieval. For Hybrid, the agent obtains a compact,
snapshot-scoped normative-assertion capability catalog and asks the configured model for a bounded
structured decision. The model treats vector evidence as the default: it selects graph retrieval
only when a relationship between legal entities is necessary to answer well, and declines it for a
direct provision lookup or explanation. The decision can select only published assertion predicates,
canonical entity labels, and an optional legal-unit scope locator from the active snapshot catalog;
it never contains Cypher or authorizes unbounded graph access. An unsupported predicate, label, or
scope is treated as uncertain and does not execute a graph query.

The graph stores a separate `NormativeAssertion` node for each evidence-backed legal statement.
An assertion is connected to its exact establishing legal unit and to atomic subject and object
legal entities. Legal-unit hierarchy uses `CONTAINS`; semantic predicate values are allowlisted
assertion properties rather than Neo4j relationship types. When a plan selects a scope locator,
the read-only query follows at most six `CONTAINS` edges, returns only assertions established by
matching descendants, and includes the minimal hierarchy path to each direct establishing unit.
Neo4j traversal remains parameterized by corpus and snapshot, and every graph result returns a
PostgreSQL-backed evidence location rather than treating the projection as legal authority.

When graph evidence contributes to an answer, the terminal inspection carries a bounded assertion
path: assertion ID, predicate, subject and object labels, establishing locator, evidence locator,
optional qualifier, and hierarchy context. An empty corpus or a snapshot without a ready assertion
release is an unavailable graph state, never evidence. The MCP transport follows the same
read-only, active-snapshot boundary.

Vector evidence remains the answer baseline. A graph miss, planner failure, unavailable graph
release, or unavailable Neo4j connection is recorded as a safe stage result and does not discard
valid vector evidence. Hybrid inspection data therefore identifies the Vector, planning, and
graph stages separately, including their content-free measurements and evidence contributions.

## Fixed-snapshot evaluation transport

`POST /v1/evaluations/execute` is a private JSON transport used only by the managed
Go evaluation worker. It accepts one explicit corpus and snapshot, question,
interface language, frozen retrieval strategy/fingerprint, and execution identity.
The agent executes only when `NORVII_EVALUATION_AGENT_BUILD`,
`NORVII_CHAT_MODEL`, `NORVII_EMBEDDING_MODEL`,
`NORVII_EVALUATION_RETRIEVAL_STRATEGY`, and
`NORVII_EVALUATION_RETRIEVAL_FINGERPRINT` match that frozen request.

This path does not resolve an active release, emit public SSE events, create a run,
or score output. Its terminal response carries bounded answer/evidence provenance,
outcome, and nullable telemetry for API persistence. Malformed or incomplete request
data produces `invalid_request`; a well-formed request whose supplied execution or
retrieval identity does not match the frozen runtime configuration, or whose generated
model identity no longer matches it, produces `frozen_identity_unavailable`. Other
execution failures produce
`evaluation_unavailable`. All three errors are content-free and never return provider
payloads, prompts, credentials, or source text.

## Verification

Run the module's formatter, Ruff, mypy, unit tests, and package build with:

```bash
make -C apps/agent ci
```

Run the PostgreSQL snapshot-isolation integration test through the configured local environment:

```bash
python infra/scripts/run-with-environment.py infra/.env make -C apps/agent test-integration
```
