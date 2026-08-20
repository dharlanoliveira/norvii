# Research: Grounded RAG Chat

## Retrieval artifact boundary

- **Decision**: Create immutable retrieval chunks during Python enrichment and anchor every chunk
  to a corpus, source revision, document version, optional legal unit, and normalized text span.
- **Rationale**: Legal answers need stable evidence locations, while chunking in the ingestion
  boundary keeps extraction and normalization out of the online request path. Version ownership
  prevents a later reprocessing run from changing an existing answer's evidence.
- **Alternatives considered**: Chunking on every chat request increases latency and makes offsets
  difficult to reproduce. Storing only opaque vector records loses article and paragraph context.

## Embeddings

- **Decision**: Define a provider-neutral embedding port and an OpenAI-compatible HTTP adapter,
  with a configured 1536-dimensional model as the POC default. Tests use a deterministic fake
  embedding provider. The configured model and embedding version are persisted with each chunk.
- **Rationale**: A multilingual hosted provider keeps the POC small and supports both corpus
  languages without adding a large local model runtime. The port keeps the product independent of
  one vendor and allows a local provider later. A fixed vector dimension permits a predictable
  PostgreSQL vector index and deterministic configuration validation.
- **Alternatives considered**: A local multilingual transformer avoids provider credentials but
  adds a large model download and runtime cost. Provider-specific SDKs couple Python and Go to a
  vendor and complicate tests. Dimensionless vectors weaken index and configuration guarantees.

## Chat model and streaming

- **Decision**: Define a Go chat-model port and use an OpenAI-compatible streaming chat adapter
  selected by configuration. Expose a Norvii-owned SSE contract with JSON events rather than
  leaking provider payloads.
- **Rationale**: The API owns online orchestration and can enforce corpus boundaries, prompt
  policy, cancellation, citation validation, and safe errors. A product-owned stream lets the
  React client remain stable if the provider changes.
- **Alternatives considered**: Calling the provider from React would expose credentials and move
  policy outside the API. Returning provider-native events couples the public contract to a
  vendor. Buffering the complete answer weakens perceived responsiveness and cancellation.

## Grounding and abstention

- **Decision**: Retrieve a bounded top-k set, require a configured minimum support score, build an
  evidence-only prompt with numbered references, and validate the completed answer's reference
  markers before publishing the completed event. Insufficient evidence, missing markers, or
  provider failure produces abstention or a safe failure.
- **Rationale**: Retrieval alone cannot guarantee that a model used the evidence. A threshold,
  constrained prompt, and terminal validation create independent safety checks appropriate to a
  POC. The product can later compare this baseline with reranking and GraphRAG.
- **Alternatives considered**: Free-form generation invites unsupported claims. Requiring an
  external factuality judge adds a second model cost and is deferred to evaluation work.

## Citation identity

- **Decision**: Cite immutable evidence by corpus ID, source ID, document version ID, stable unit
  locator, and start/end offsets; include a short faithful excerpt for inspection. Resolve the
  reference through the existing workspace source and document routes.
- **Rationale**: Stable IDs and offsets survive UI changes and distinguish a superseded revision
  from the current source. Existing document units already model articles, paragraphs, items, and
  source order.
- **Alternatives considered**: Page-only citations do not work reliably for HTML and are too
  coarse for nested legal units. Repeating full chunks in answer text increases response size and
  makes the answer harder to inspect.

## Conversation lifecycle and telemetry

- **Decision**: Keep messages in client memory for this feature, permit one active response per
  workspace, and emit content-safe terminal telemetry for every request outcome.
- **Rationale**: Persistence and multi-user conversation ownership are not needed to demonstrate
  grounded retrieval. Request-level counts, durations, model versions, and outcome categories
  support later evaluation without logging protected content.
- **Alternatives considered**: Durable conversation storage increases schema, retention, and
  privacy scope. Full prompt/response logging is useful for debugging but violates the project's
  content-free diagnostic boundary.

## Deferred capabilities

- **Decision**: Do not add Neo4j writes, graph traversal, MCP tools, reusable skills, reranking,
  or evaluation dashboards in this feature.
- **Rationale**: Feature 005 must establish a measurable vector-RAG baseline before Feature 008
  compares vector, graph, and hybrid retrieval. Keeping those capabilities deferred preserves an
  independently testable P1 slice.
