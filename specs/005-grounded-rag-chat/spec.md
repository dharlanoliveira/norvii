# Feature Specification: Grounded RAG Chat

**Feature Branch**: `005-grounded-rag-chat`

**Created**: 2026-08-20

**Status**: In Progress

**Input**: User description: "Implement grounded RAG chat over the active legal corpus with streamed answers, evidence-linked citations, explicit abstention, bilingual UI, and bounded retrieval observability. Use the existing corpus documents as the evidence boundary; defer Neo4j GraphRAG, MCP tools, skills, and evaluation dashboards to later features."

## User Scenarios & Testing _(mandatory)_

### User Story 1 - Ask a Corpus-Grounded Question (Priority: P1)

A researcher opens a ready corpus, asks a question in the chat, and receives an answer based only on the published documents belonging to that corpus. The answer appears progressively and clearly communicates that it is an evidence-grounded research result rather than legal advice.

**Why this priority**: This is Norvii's first useful AI journey and proves the product can connect the real corpus and document viewer to an answer without returning unsupported model output.

**Independent Test**: Seed a corpus with a deterministic published document, ask a question whose answer is present in that document, and verify that the response completes with supporting evidence references and no content from another corpus.

**Acceptance Scenarios**:

1. **Given** a researcher is inside an enabled corpus with at least one ready document, **when** they submit a valid question, **then** the chat shows a pending state followed by progressively rendered, semantic Markdown answer content and a completed state.
2. **Given** the answer is supported by one or more published document passages, **when** generation completes, **then** the answer includes one or more stable evidence references that identify the source and document location used.
3. **Given** the researcher changes the active corpus before submitting a question, **when** the request is processed, **then** retrieval and answer generation use only the newly active corpus.
4. **Given** the researcher cancels an in-progress response or navigates away, **when** cancellation completes, **then** the client stops rendering that response and keeps the prior conversation state intact.
5. **Given** a ready source was published before vector enrichment was enabled, **when** it is reprocessed with a new enrichment pipeline version, **then** the system publishes a new immutable document version with one ready, model-versioned embedding per retrieval chunk before that version is eligible for chat retrieval.

### User Story 2 - Inspect and Navigate Supporting Evidence (Priority: P2)

A researcher reviews the evidence attached to an answer, sees which source and legal document location support it, and opens the corresponding document view without losing the conversation.

**Why this priority**: Traceability is essential for legal research. A fluent answer without a path back to the preserved text would not satisfy Norvii's evidence-first purpose.

**Independent Test**: Ask a question with a known answer, select each returned evidence reference, and verify that the workspace opens the matching corpus-owned source and scrolls to the referenced document unit or text span.

**Acceptance Scenarios**:

1. **Given** a completed answer contains evidence references, **when** the researcher opens one reference, **then** the workspace displays the source title, document location, and preserved text span identified by that reference.
2. **Given** an answer contains multiple evidence references, **when** the researcher switches between them, **then** the conversation remains visible and the selected evidence is clearly distinguished.
3. **Given** an evidence reference no longer resolves to a published document, **when** the researcher opens it, **then** the product shows a localized unavailable-evidence state and does not substitute unrelated content.

### User Story 3 - Receive a Scope-Limited Response (Priority: P3)

A researcher asks a greeting or a question that is not sufficiently supported by the active corpus and receives a helpful, scope-limited response. The response explains the evidence boundary without presenting unsupported legal or factual claims as findings.

**Why this priority**: A responsive conversational experience makes the assistant approachable while preserving the legal-research boundary that prevents plausible but unsupported statements from becoming findings.

**Independent Test**: Ask a deterministic greeting and a question absent from the corpus. Verify that both receive a response in the question language, contain no unsupported factual claim or citation, and offer a useful corpus-related next action when appropriate.

**Acceptance Scenarios**:

1. **Given** retrieval does not provide sufficient evidence for the question, **when** generation completes, **then** the chat returns a scope-limited response that explains the missing support and does not make legal or factual claims.
2. **Given** all documents in the active corpus are pending or failed, **when** the researcher submits a question, **then** the chat explains that no ready evidence is available and invites a corpus-related question without making unsupported claims.
3. **Given** an upstream retrieval, model, or streaming failure occurs, **when** the request ends, **then** the chat displays a safe localized failure with retry guidance and does not expose internal details or partial unsupported content.

### Edge Cases

- The question is empty, exceeds the documented POC length, or contains only whitespace: the client rejects it locally with an accessible localized explanation.
- The active corpus is disabled or changes while a request is in flight: the request is rejected or cancelled at the corpus boundary and cannot disclose data from another corpus.
- A retrieved document contains instructions directed at an AI system: the content is treated only as legal evidence and never as an instruction that changes system behavior.
- A source or document is superseded while a response is being generated: the response remains tied to the immutable evidence version captured at retrieval time.
- A streamed response ends unexpectedly: the client marks it incomplete and offers retry without presenting it as a completed grounded answer.
- A question asks for legal advice or a conclusion beyond the available text: the response states the research limitation and preserves the technical-demonstration disclaimer.
- Multiple passages support the same statement: the answer may group their references without duplicating identical visible evidence.
- A citation points to a nested item or paragraph: the surrounding article context remains visible when the evidence is opened.
- A vector provider rejects a batch, times out, or returns a malformed or incorrectly sized vector: the attempt fails safely, the prior ready document remains available, and no partially embedded document becomes eligible for retrieval.

## Requirements _(mandatory)_

### Functional Requirements

- **FR-001**: The system MUST accept a non-empty natural-language question from the researcher within a documented POC length limit.
- **FR-002**: Every research request MUST be bound to exactly one active enabled corpus before retrieval begins.
- **FR-003**: Retrieval MUST consider only published document versions and addressable document units owned by the active corpus.
- **FR-004**: The system MUST generate a scope-limited assistant response when retrieved evidence does not meet the configured minimum support threshold. That response MUST NOT make legal or factual claims and MUST explain the active corpus boundary when the user seeks information.
- **FR-005**: A completed answer containing factual support MUST expose evidence references for the supporting passages, including corpus, source, document version, stable unit or text location, and the visible excerpt needed to inspect the support. A scope-limited response with no supporting evidence MUST expose no evidence references.
- **FR-006**: The answer MUST remain associated with the immutable evidence version used for generation even if the source is later reprocessed.
- **FR-007**: The chat MUST render answer output progressively while generation is active and MUST expose pending, streaming, completed, cancelled, abstained, and failed states. Before the first answer text is available, the pending state MUST show a localized, accessible processing indicator with motion reduced when the operating system requests it.
- **FR-008**: The researcher MUST be able to cancel an active response, and cancellation MUST stop further visible output for that response.
- **FR-009**: Selecting an evidence reference MUST open the corresponding corpus-owned source and document location while preserving the current conversation view.
- **FR-010**: If an evidence reference cannot be resolved, the system MUST show a localized unavailable-evidence outcome and MUST NOT substitute another source or document.
- **FR-011**: Scope-limited responses MUST state that the active corpus does not provide sufficient support when relevant and MUST NOT contain fabricated factual claims or unsupported citations.
- **FR-012**: The system MUST treat retrieved legal text as untrusted evidence, never as executable instructions or higher-priority system guidance.
- **FR-013**: Public errors MUST distinguish invalid questions, unavailable corpora, insufficient evidence, retrieval failure, generation failure, cancellation, and unexpected service failure without exposing internals.
- **FR-014**: The client MUST provide localized question, pending, streaming, completed, abstention, cancellation, failure, citation, and disclaimer experiences in English and Portuguese, with English as the default interface language.
- **FR-015**: The answer language MUST follow the language of the question when it can be determined; otherwise it MUST follow the active interface language.
- **FR-016**: The system MUST record bounded, content-safe retrieval and generation telemetry for each request, including corpus identity, evidence count, outcome, latency, and token or character counts when available.
- **FR-017**: Telemetry and errors MUST NOT contain credentials, complete document text, full prompts, or private request data.
- **FR-018**: The system MUST enforce a bounded retrieval result count, context size, generation duration, and streamed response size suitable for a single-user POC.
- **FR-019**: Conversation messages MAY remain ephemeral for this feature; persistent conversation history is not required.
- **FR-020**: This feature MUST NOT publish Neo4j graph projections, perform GraphRAG traversal, expose MCP research tools, invoke reusable research skills, or provide an evaluation dashboard.
- **FR-021**: Retrieval chunks MUST be enriched with a fixed-dimension, model-versioned embedding before publication. Reprocessing a source with a changed enrichment pipeline version MUST create a new immutable document version so sources published before vector enrichment can be backfilled without mutating historical evidence. Only ready chunks with a valid embedding from the latest ready document version are eligible for online retrieval.
- **FR-022**: The client MUST use an approved chat-rendering library for the conversation runtime, thread viewport, message rendering, streaming lifecycle, and composer. Generated assistant Markdown MUST render into accessible semantic elements. Evidence references MUST remain derived from the typed stream contract and MUST NOT be reconstructed by parsing generated Markdown.
- **FR-023**: When a corpus chat has no messages, the client MUST provide at least two localized, corpus-safe starter questions. Selecting one MUST submit it through the same corpus-scoped chat request path as typed input and MUST be disabled while a request is running.

### Key Entities

- **Research question**: The researcher-submitted question, active corpus boundary, interface language, and request identity.
- **Retrieved evidence**: A ranked, immutable reference to a published source document, document version, unit or text span, excerpt, and retrieval score or rationale.
- **Grounded answer**: A streamed or completed response whose factual statements are linked to retrieved evidence, or a scope-limited conversational response that contains no factual claims when support is insufficient.
- **Evidence reference**: A user-selectable citation target that resolves to a corpus-owned source and stable document location.
- **Research request telemetry**: Bounded operational measurements and safe outcome categories for one question lifecycle.

## Success Criteria _(mandatory)_

### Measurable Outcomes

- **SC-001**: In deterministic local scenarios with available evidence, at least 95% of valid questions produce a completed grounded answer or explicit abstention within 10 seconds, excluding unavailable upstream services.
- **SC-002**: At least 100% of completed grounded answers in automated scenarios include a resolvable evidence reference for every answer segment designated as factual support.
- **SC-003**: In automated corpus-isolation scenarios, 100% of retrieved evidence and answer references belong to the active corpus, with zero cross-corpus disclosure.
- **SC-004**: In deterministic unsupported-question and greeting scenarios, 100% of scope-limited responses contain zero fabricated citations, unsupported factual claims, or unsupported evidence references.
- **SC-005**: A researcher can open every evidence reference from a completed answer and reach the corresponding preserved document location in under two interactions.
- **SC-006**: At least 95% of accepted requests show a localized processing indicator within 1 second, streamed responses show visible answer progress within 1 second of the first accepted generation event, and failed or cancelled responses are never displayed as completed.
- **SC-007**: Invalid questions, unavailable evidence, upstream failures, cancellation, and safety-boundary violations produce the expected localized outcome in 100% of automated contract and component scenarios.
- **SC-008**: Retrieval and generation telemetry is present for 100% of terminal request outcomes and contains no complete document text, credentials, full prompts, or private request data in automated log inspections.
- **SC-009**: English and Portuguese interface resources provide parity for all grounded-chat states, and the default interface remains English.
- **SC-010**: In deterministic enrichment scenarios, 100% of a successfully reprocessed source's retrieval chunks have the configured vector dimension and embedding model metadata; malformed or failed embeddings publish zero chunks for the new version and leave the preceding ready version unchanged.

## Assumptions

- Feature 004 remains the authoritative source and document boundary; only ready published documents and immutable units are eligible evidence.
- An existing vector-capable persistence boundary and normalized document text are available for the implementation plan to evaluate as the first retrieval path. The specification does not mandate a particular provider or model.
- The POC is operated by one trusted local user. Authentication, authorization roles, conversation persistence, multi-user quotas, and public deployment are out of scope.
- The initial retrieval path is intended to be small and explainable. Graph relationships, Neo4j projections, MCP, reusable skills, reranking experiments, and evaluation dashboards belong to later features.
- The model provider may be unavailable, rate-limited, or configured without credentials in local development; the product must preserve an honest unavailable or failed state.
- Legal content remains in its source language and is not translated as part of retrieval. The answer may follow the question language, but evidence excerpts remain faithful to the preserved source text.
- The technical-demonstration disclaimer remains visible wherever a grounded answer or abstention is shown; Norvii does not provide legal advice.

## Norvii Feature Requirements _(mandatory)_

### Scope and Boundaries

- **In scope**: One active-corpus question flow; bounded retrieval over published document content; streamed grounded answer or scope-limited conversational response; evidence references linked to stable document locations; bilingual chat states; safe failure and cancellation behavior; content-safe request telemetry; vector enrichment during ingestion; versioned backfill of sources that predate embedding publication; and cosine-ranked retrieval of only ready embeddings from a source's latest ready document version.
- **Out of scope**: GraphRAG, Neo4j graph publication or traversal, MCP research tools, reusable skills, persistent conversations, authentication, multi-user access, model fine-tuning, OCR, automatic translation of evidence, and evaluation dashboards.

### Prototype Baseline _(mandatory for production UI features)_

- **Approved baseline**: The production workspace and unavailable-chat surface delivered by [Feature 004](../004-corpus-catalog/spec.md), which itself follows the verified prototype baseline from [Feature 001](../001-product-experience-prototype/spec.md).
- **Intentional differences**: The unavailable-chat state is replaced by a real question composer, streamed answer states, abstention states, and evidence navigation. Existing source tree, document viewer, corpus boundary, bilingual behavior, and technical disclaimer remain authoritative.

### Evidence and Corpus Boundaries _(mandatory for retrieval, chat, or citations)_

- **Active corpus constraint**: Every question, retrieval operation, evidence reference, document location, telemetry record, and answer is scoped to one active corpus. A request MUST fail closed when that boundary cannot be established.
- **Evidence behavior**: Factual answers are generated only from retrieved passages belonging to immutable published document versions. Each factual answer segment has inspectable supporting references. When evidence is absent, the assistant may provide only a scope-limited conversational response with no factual claims or citations.
- **Version behavior**: Evidence references identify the immutable document version used by the request so later source reprocessing cannot silently change the meaning of an existing answer.
