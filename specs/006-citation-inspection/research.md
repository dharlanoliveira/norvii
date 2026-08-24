# Research: Citation Navigation and Inspection

## Decision 1: Keep inspection session-scoped

- **Decision**: Store an immutable inspection snapshot in the React runtime keyed by Assistant UI
  assistant-message identifier. Do not add a conversation or inspection table.
- **Rationale**: The Feature 006 assumption keeps chat history session-only. The inspection must
  survive changing between Chat and Source in one workspace session, which the existing runtime
  already does for answer text and references.
- **Alternatives considered**:
  - Persist answers and inspections in PostgreSQL: rejected because it expands privacy, retention,
    authorization, and lifecycle scope beyond this POC feature.
  - Recalculate inspection on demand: rejected because it can produce a different document version
    or metric set after reprocessing and would violate answer-specific immutability.

## Decision 2: Add an immutable document-version read route

- **Decision**: Add `GET /api/v1/corpora/{corpusId}/sources/{sourceId}/documents/{documentVersionId}`
  for citation resolution. It returns exactly the published document version named by the
  reference, or the existing safe not-found response.
- **Rationale**: The current source document route intentionally returns the latest ready version.
  A reprocessed source can therefore make a historical answer point at current text with the same
  locator. Retrieval already references `document_versions.id`, so the identity exists.
- **Alternatives considered**:
  - Pass an expected document identifier to the latest route: rejected because a query parameter
    weakens the route's latest semantics and makes accidental fallback easier.
  - Use the latest version and compare locator only: rejected because locator equality does not
    establish immutable evidence identity.

## Decision 3: Preserve legal hierarchy while highlighting an exact range

- **Decision**: Retain the reader's visible document/article hierarchy. A cited paragraph or item
  resolves to its nearest visible ancestor, while `startOffset` and `endOffset` identify the exact
  highlighted range inside the rendered content.
- **Rationale**: Brazilian items and paragraphs remain meaningful only in their article context.
  Existing reading behavior correctly keeps that context and should not be replaced with an
  independent nested-location pane.
- **Alternatives considered**:
  - Display each nested unit as an independent location: rejected because it fragments the legal
    structure and regresses Feature 004's Brazilian normative reading behavior.
  - Highlight the entire article only: rejected because it does not satisfy exact cited-location
    navigation.

## Decision 4: Use nullable, measured inspection data

- **Decision**: Represent unavailable retrieval score, stage timings, and provider token use as
  JSON `null`. Measure retrieval and generation around their Python owner operations and total
  request time in the Go public handler. Surface provider usage only when supplied by the provider.
- **Rationale**: Existing telemetry is hard-coded to zero, which falsely implies a measurement.
  `null` distinguishes unavailable data from a genuinely measured zero.
- **Alternatives considered**:
  - Estimate tokens from characters: rejected by the specification and inaccurate across models.
  - Show zeros for missing telemetry: rejected because it misrepresents unavailable values.
  - Measure all stages in Go: rejected because retrieval and generation occur inside Python and
    would be conflated with network transport.

## Decision 5: Expose vector distance, not a manufactured score

- **Decision**: Return nullable `cosineDistance` from the current vector query and label it as
  distance in inspection. Keep retrieval rank as the primary ordering signal.
- **Rationale**: PostgreSQL currently orders by the pgvector cosine-distance operator. A distance
  is a real available value; converting it into a score would create a new undocumented semantic.
- **Alternatives considered**:
  - Expose rank only: rejected because the feature requires relevance information beyond order.
  - Convert distance to percentage relevance: rejected because vector similarity calibration is
    model- and corpus-dependent.

## Decision 6: Extend the existing SSE contract additively

- **Decision**: Add optional-for-legacy `inspection` data to terminal SSE events and explicit
  immutable document-version fields to references. Feature 006 clients require the fields for
  completed answer inspection; clients that do not recognize additive fields remain compatible.
- **Rationale**: Feature 005 owns the existing public stream and allows additive fields. A separate
  inspection endpoint would require server-side answer persistence contrary to the feature scope.
- **Alternatives considered**:
  - Create a separate inspection endpoint: rejected because no durable answer identifier or
    inspection storage exists.
  - Replace the SSE schema with a breaking version: rejected because the change is additive and
    does not change existing field meaning.

## Decision 7: Keep the inspector inline and delegate chat rendering

- **Decision**: Add an accessible per-answer disclosure beside the existing Assistant UI message
  rendering. Use the current `@assistant-ui/react` and `@assistant-ui/react-markdown` message
  primitives; do not introduce custom Markdown parsing or a modal.
- **Rationale**: An inline disclosure keeps evidence, answer, and inspector visibly associated,
  preserves the thread state, avoids modal focus complexity, and fulfills the earlier decision to
  delegate LLM rendering to Assistant UI.
- **Alternatives considered**:
  - New inspector route: rejected because it breaks the answer/source comparison flow.
  - Modal drawer: rejected because it obscures the answer and adds focus-trap requirements without
    stronger POC value.
