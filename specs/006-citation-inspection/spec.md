# Feature Specification: Citation Navigation and Inspection

**Feature Branch**: `006-citation-inspection`

**Created**: 2026-08-24

**Status**: Draft

**Input**: User description: "Create the next feature: a user opens cited evidence and an evaluator inspects retrieval, latency, and token use."

## User Scenarios & Testing *(mandatory)*

### User Story 1 - Open cited legal evidence (Priority: P1)

A researcher reading a grounded answer selects an inline citation and is taken to the exact
preserved legal location that supports the cited claim, without losing the current answer or
active corpus.

**Why this priority**: Traceability is the essential promise of evidence-led legal research. A
useful answer cannot be independently reviewed if its cited passage cannot be opened reliably.

**Independent Test**: Complete an answer with references to two locations, select each
citation, and verify that the correct active-corpus source opens at the matching highlighted
location while the conversation remains available.

**Acceptance Scenarios**:

1. **Given** a completed answer with valid citations, **When** the researcher reviews its source
   locations, **Then** the workspace groups repeated passages that resolve to the same source and
   legal location and opens that cited source location on selection.
2. **Given** an answer with citations to two different sources, **When** the researcher selects
   either citation, **Then** the selected source and location match that citation and the answer
   remains available when the researcher returns to chat.
3. **Given** a citation whose preserved evidence is unavailable or no longer eligible,
   **When** the researcher selects it, **Then** the workspace explains the unavailable evidence
   without silently substituting another location.

---

### User Story 2 - Inspect answer evidence and execution (Priority: P2)

An evaluator opens technical inspection for a completed answer and reviews the evidence used,
its retrieval order, the retrieval approach, timing, and token-use measurements needed to assess
the demonstration.

**Why this priority**: Norvii is a proof of capability. The evaluator must be able to understand
what evidence shaped an answer and what it cost, rather than treating the answer as opaque.

**Independent Test**: Complete an answer with deterministic evidence and execution measurements,
open its inspection, and verify that every displayed evidence item, timing value, and token value
belongs to that answer and its active corpus.

**Acceptance Scenarios**:

1. **Given** a completed grounded answer, **When** the evaluator opens its inspection,
   **Then** the inspection lists the evidence in retrieval order with source, stable location,
   excerpt, and relevance information.
2. **Given** measured execution details for a completed answer, **When** the evaluator opens its
   inspection, **Then** it shows the retrieval approach, outcome, relevant elapsed times, and
   input and output token-use measurements.
3. **Given** a provider does not supply a measurement, **When** the evaluator opens inspection,
   **Then** the corresponding value is clearly shown as unavailable rather than estimated or
   represented as zero.

---

### User Story 3 - Compare citations from one answer (Priority: P3)

A researcher uses the inspection evidence list to move between the evidence items for one answer
and compare their passages in the source view.

**Why this priority**: Legal answers often synthesize more than one provision. Inspecting each
supporting passage makes the answer's reasoning reviewable without a new query.

**Independent Test**: Open inspection for an answer with multiple references, select each
evidence item, and verify that it opens its exact cited location without changing the answer's
inspection record.

**Acceptance Scenarios**:

1. **Given** an inspected answer with multiple evidence items, **When** the researcher selects
   an evidence item, **Then** the corresponding location opens using the same navigation behavior
   as its inline citation.
2. **Given** the researcher returns from a source location to the answer, **When** they reopen
   inspection, **Then** it remains associated with the same completed answer and its immutable
   evidence set.

### Edge Cases

- A citation targets a nested legal unit, such as a paragraph or item within an article.
- Multiple citations resolve to the same location but use distinct supporting excerpts.
- An answer is abstained, cancelled, or failed before completion.
- An active corpus changes while an inspection or source location is open.
- A source is reprocessed after an answer completes.
- A displayed excerpt contains preserved Portuguese or English legal text that differs from the
  current interface language.

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: The system MUST make every unique source and legal location represented by valid
  evidence in a completed answer selectable through an accessible citation control. Repeated
  passages at that location MUST be represented as one location with its supporting-passage count.
- **FR-002**: Selecting a citation MUST open the referenced source within the active corpus and
  bring the immutable cited location into view with a clear visual indication.
- **FR-003**: Citation navigation MUST preserve the completed answer and allow the researcher to
  return to it without issuing another question.
- **FR-004**: The system MUST fail closed when a citation is foreign to the active corpus,
  unresolved, superseded for inspection, or otherwise unavailable; it MUST explain that state and
  MUST NOT substitute nearby or current source text.
- **FR-005**: Each completed grounded answer MUST provide an answer-specific technical inspection
  view that identifies its outcome and evidence set. The control MUST be collapsed by default and
  visually subordinate to the answer and its inline citations.
- **FR-006**: The inspection view MUST show every evidence item in retrieval order with its
  source identity, stable legal location, and relevance information. The source view remains the
  authoritative place for each preserved excerpt.
- **FR-007**: The inspection view MUST show the retrieval approach and the available retrieval,
  generation, and total elapsed-time measurements for the selected answer.
- **FR-008**: The inspection view MUST show available input and output token-use measurements and
  MUST label any unavailable measurement explicitly without inventing a value.
- **FR-009**: Selecting an evidence item from inspection MUST use the same corpus-scoped,
  immutable navigation behavior as selecting its inline citation.
- **FR-010**: Inspection and citation controls MUST be usable by keyboard and expose meaningful
  accessible names, selected states, and unavailable states.
- **FR-011**: Product-authored inspection text, labels, errors, and empty states MUST follow the
  selected interface language. Preserved legal text, source titles, user questions, and generated
  answers MUST retain their original language.
- **FR-012**: Inspection data MUST remain scoped to the answer and its active corpus. It MUST NOT
  reveal credentials, complete prompts, hidden provider payloads, or evidence from another corpus.
- **FR-013**: An abstained, cancelled, or failed request MUST expose its honest outcome and any
  safe, available execution details, but MUST NOT present an evidence inspection as if a grounded
  answer completed.
- **FR-014**: Reprocessing a source after an answer completes MUST NOT change the evidence identity
  presented by that answer's inspection.
- **FR-015**: A completed answer with more than three unique cited locations MUST initially show
  three locations and provide an accessible control to reveal the remaining locations. The
  inspection MUST continue to expose every individual evidence item in retrieval order.
- **FR-016**: When no source is selected in the Source tab, the selection state MUST provide an
  accessible action that opens an available source. When the corpus has no sources, it MUST provide
  an accessible action that opens the official-URL source form.

### Key Entities

- **Citation target**: The immutable corpus-scoped source location and evidence span referenced by
  an answer claim.
- **Answer inspection**: The answer-specific record shown to an evaluator, containing outcome,
  evidence order, retrieval approach, and available execution measurements.
- **Execution measurement**: A content-free value describing elapsed time or token use for one
  completed, abstained, cancelled, or failed request.

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: In automated supported-answer journeys, 100% of inline citations and inspection
  evidence selections open their expected active-corpus legal location in no more than two user
  interactions.
- **SC-002**: In automated nested-location and multiple-citation journeys, 100% of selected
  evidence locations are visibly brought into view and retain their matching source identity.
- **SC-003**: In deterministic local journeys, technical inspection becomes available within one
  second after its answer reaches a terminal state.
- **SC-004**: In automated corpus-isolation and unavailable-evidence journeys, 100% of foreign,
  unresolved, or unavailable citations fail closed without exposing substitute or cross-corpus
  content.
- **SC-005**: In automated measurement journeys, 100% of available timing and token-use values
  are attributed to the selected answer, while all unavailable values are labelled unavailable.
- **SC-006**: Keyboard-only automated journeys complete citation navigation and inspection for
  every supported state without critical or serious accessibility findings.
- **SC-007**: English and Portuguese interface journeys have complete product-authored inspection
  copy, while preserved legal content remains unchanged.

## Assumptions

- Norvii remains a single-user POC without persisted conversation history; inspection is available
  only for answers retained in the current workspace session.
- Feature 005 supplies immutable evidence references and content-free execution telemetry for
  completed answers.
- A technical inspection control is available to the same POC user; no separate evaluator account
  or authorization model is introduced by this feature.
- Exact scores and provider measurements are displayed only when already available from the
  retrieval or generation request; the feature does not estimate missing values.

## Norvii Feature Requirements *(mandatory)*

### Scope and Boundaries

- **In scope**: Citation selection, immutable evidence navigation, answer-specific inspection,
  retrieval evidence order and relevance, execution timing and token-use presentation,
  localized accessible states, and deterministic verification.
- **Out of scope**: Graph paths and graph retrieval, cross-corpus comparison, persisted
  conversations, evaluator accounts, prompt disclosure, reranking changes, new model providers,
  and new corpus ingestion behavior.

### Prototype Baseline *(mandatory for production UI features)*

- **Approved baseline**: The production corpus workspace delivered by
  [Feature 002](../002-production-web-experience/spec.md), the authoritative source and document
  viewer delivered by [Feature 004](../004-corpus-catalog/spec.md), and the grounded answer
  experience delivered by [Feature 005](../005-grounded-rag-chat/spec.md).
- **Intentional differences**: The workspace gains answer-level technical inspection and stronger
  citation-location feedback. It retains the existing source library, active-corpus boundary,
  chat/source relationship, editorial visual language, and technical-demonstration disclaimer.

### Evidence and Corpus Boundaries *(mandatory for retrieval, chat, or citations)*

- **Active corpus constraint**: Every citation target, evidence item, inspection value, and source
  navigation action is limited to the corpus that produced the selected answer.
- **Evidence behavior**: Completed answers expose only their immutable evidence references.
  Missing, foreign, superseded, or unresolved evidence is shown as unavailable; the system does
  not silently replace it. Abstained, cancelled, and failed requests retain an honest status.
