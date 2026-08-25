# Feature Specification: Planned Hybrid Retrieval

**Feature Branch**: `009-planned-hybrid-retrieval`

**Created**: 2026-08-25

**Status**: Implemented - manual configured-provider verification pending

**Input**: User description: "For every research question, retrieve vector evidence first. In
Hybrid mode, plan whether the selected snapshot's graph schema can add relevant evidence, use
the graph only when it can, and expose the vector, planning, and graph contributions in the
research record. Replace the standalone Graph option with Vector and Hybrid options."

## User Scenarios & Testing *(mandatory)*

### User Story 1 - Receive a grounded answer through planned hybrid retrieval (Priority: P1)

A researcher asks any question about a selected corpus and chooses Hybrid retrieval. Norvii
always retrieves semantic evidence from the active snapshot. It evaluates whether the graph can
add useful, schema-supported context, then uses graph evidence only when that evaluation finds a
relevant path. The final response distinguishes its evidence sources and remains grounded.

**Why this priority**: A researcher should not need to predict whether a broad question is
better suited to semantic retrieval or legal relationships. Hybrid retrieval must adapt to the
question rather than failing because one evidence source has no match.

**Independent Test**: Ask one broad document question and one relationship-focused question in
each seeded corpus. Verify that both start with snapshot-scoped vector retrieval; only the
relationship-focused question receives a graph contribution when its graph has a relevant path.

**Acceptance Scenarios**:

1. **Given** a corpus has an active snapshot, **When** a researcher asks a broad question with
   Hybrid retrieval, **Then** Norvii retrieves vector evidence and returns a cited answer or a
   safe evidence-limited outcome even when the graph contributes nothing.
2. **Given** a corpus has an active snapshot and a relevant ready graph path, **When** a
   researcher asks a relationship-focused question with Hybrid retrieval, **Then** Norvii
   combines cited vector and graph evidence without representing graph-derived context as
   official text.
3. **Given** a question is in a different language from the graph's canonical entity labels,
   **When** a relevant ready graph path exists, **Then** Hybrid maps the question to only
   published canonical graph entities and returns its cited graph evidence.
4. **Given** the question does not map to a useful graph concept or relationship, **When** a
   researcher uses Hybrid retrieval, **Then** Norvii skips graph retrieval with an inspectable
   reason and continues with vector evidence.
5. **Given** neither retrieval path finds sufficient evidence, **When** Norvii responds,
   **Then** it does not make unsupported legal claims and offers a localized, safe next step.

---

### User Story 2 - Select an intelligible retrieval approach (Priority: P2)

A researcher selects either Vector or Hybrid before asking a question. They do not select a
standalone Graph mode because graph use is an evidence-planning decision within Hybrid.

**Why this priority**: The strategy control should represent meaningful research choices rather
than require the researcher to understand the graph vocabulary before asking a question.

**Independent Test**: Open the workspace in both interface languages and verify that it offers
Vector and Hybrid only, explains the chosen approach, and retains the choice for the submitted
question.

**Acceptance Scenarios**:

1. **Given** a researcher opens a corpus workspace, **When** they inspect retrieval choices,
   **Then** they can choose Vector or Hybrid and cannot submit a standalone Graph request.
2. **Given** a researcher submits a question with Vector retrieval, **When** the answer is
   produced, **Then** Norvii uses only vector evidence and records that no graph planning was
   requested.
3. **Given** a researcher submits a question with Hybrid retrieval, **When** the answer is
   produced, **Then** the visible strategy remains Hybrid whether the graph was used, skipped,
   unavailable, or produced no evidence.

---

### User Story 3 - Inspect how Hybrid made its retrieval decision (Priority: P3)

An evaluator opens the research record for a Hybrid answer and understands what vector evidence
was retrieved, whether graph use was considered appropriate, whether a graph query ran, and what
each evidence path contributed to the final answer.

**Why this priority**: Hybrid retrieval is credible only when its adaptive behavior can be
inspected without exposing prompts, credentials, full document contents, or opaque internal
model reasoning.

**Independent Test**: Run one Hybrid question for each of these outcomes: graph used, graph
skipped as irrelevant, graph unavailable, and no evidence. Verify that each research record
identifies the outcome, safe reason, timing, and cited contribution of every retrieval stage.

**Acceptance Scenarios**:

1. **Given** Hybrid retrieval uses graph evidence, **When** an evaluator opens the research
   record, **Then** it shows the graph decision, a concise schema-relevant rationale, the graph
   path, and the immutable evidence locations that support it.
2. **Given** Hybrid retrieval skips the graph, **When** an evaluator opens the research record,
   **Then** it shows that the graph was not relevant to the question and shows the vector
   contribution without treating the result as a failure.
3. **Given** the graph is unavailable or errors after vector retrieval succeeds, **When** an
   evaluator opens the research record, **Then** it shows an actionable safe diagnostic and the
   answer remains accurately identified as Hybrid with a vector-only contribution.

### Edge Cases

- A broad question contains no graph entity or relationship but vector retrieval finds relevant
  legal passages.
- A relationship-focused question matches a graph concept but the graph path has no supporting
  immutable evidence location.
- A graph release is unavailable, stale, pending, failed, or belongs to another snapshot.
- The graph planner returns an invalid, unsupported, or unsafe graph request.
- Vector retrieval returns no evidence while graph retrieval identifies a relevant path.
- Both retrieval paths return no evidence for a greeting, general question, or corpus-external
  question.
- The interface language differs from the corpus language or the question language.
- A question refers to a graph entity in a language different from its canonical graph label.
- A researcher compares historical answers created before the standalone Graph option was
  removed.

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: The system MUST expose Vector and Hybrid as the only selectable retrieval
  approaches for new research questions. It MUST remove standalone Graph selection from the
  workspace and comparison workflow.
- **FR-002**: Vector retrieval MUST run for every new research question against the selected
  corpus and active immutable snapshot, regardless of the selected approach.
- **FR-003**: For every Hybrid question, the system MUST evaluate the question against a
  versioned description of the active snapshot's available graph entities, relationship types,
  and supported graph capabilities before deciding whether to run graph retrieval.
- **FR-004**: The Hybrid planning outcome MUST be a bounded structured decision that identifies
  whether graph retrieval is relevant and, when relevant, the allowed graph concepts or
  relationships to investigate. It MUST NOT authorize unbounded graph access or introduce
  external evidence.
- **FR-004a**: A Hybrid plan MUST select graph entities only from the active snapshot's published
  canonical entity catalog. It MAY select such an entity when its label differs from the question
  language; direct lexical overlap with the question is not required.
- **FR-005**: The system MUST run graph retrieval for a Hybrid question only when the planning
  outcome identifies a relevant, ready, snapshot-scoped graph capability. A graph miss or skip
  MUST NOT discard otherwise valid vector evidence.
- **FR-006**: Hybrid retrieval MUST combine vector and graph evidence only when each
  contribution has immutable source locations in the selected corpus and active snapshot. It
  MUST deduplicate shared evidence locations and retain the contribution of each path.
- **FR-007**: When graph retrieval is skipped, unavailable, fails safely, or returns no evidence,
  Hybrid retrieval MUST still complete from available vector evidence. The response and research
  record MUST accurately state that the graph did not contribute.
- **FR-008**: When no retrieval path returns sufficient evidence, the final response MAY provide
  a localized non-legal conversational, scope, or clarification response, but it MUST NOT make a
  material legal claim without a citation to the selected corpus and snapshot.
- **FR-009**: The research record for each Vector result MUST show its selected evidence,
  retrieval outcome, timing, available token measurements, and snapshot identity.
- **FR-010**: The research record for each Hybrid result MUST separately show vector retrieval,
  graph planning, graph retrieval, graph-path evidence when used, each stage's outcome and
  timing, available token measurements, and the active snapshot identity.
- **FR-011**: Research records MUST use explicit, localized states that distinguish graph
  contribution, graph skipped as not relevant, graph no evidence, graph unavailable, graph
  failure, and vector no evidence. A completed request MUST NOT imply that every retrieval stage
  found evidence.
- **FR-012**: All graph planning, retrieval, evidence, citations, and final answer generation
  MUST remain constrained to the selected corpus and active immutable snapshot. Candidate,
  stale, foreign-corpus, and foreign-snapshot information MUST be excluded.
- **FR-013**: Graph paths used by Hybrid answers MUST identify their relationship types and
  exact immutable supporting locations. They MUST be inspectable without exposing private model
  prompts, credentials, complete documents, or hidden reasoning.
- **FR-014**: The system MUST preserve an already rendered standalone Graph result for inspection
  during the current browser session, but new requests and new comparisons MUST use only Vector
  and Hybrid approaches.
- **FR-015**: Product-authored approach, planning, graph-stage, availability, and inspection
  text MUST follow the selected interface language. User questions, legal evidence, source
  titles, and generated answers MUST retain their original language.
- **FR-016**: The system MUST record content-free operational measurements and safe failure
  categories for vector retrieval, graph planning, graph retrieval, and response generation.
  It MUST not log credentials, prompts, hidden model reasoning, or complete legal content.

### Key Entities

- **Retrieval approach**: The researcher-visible choice for a new question: Vector or Hybrid.
- **Hybrid retrieval plan**: A bounded, inspectable decision about whether graph evidence can
  add value to a question and which published graph capabilities are eligible to contribute.
- **Graph capability description**: The versioned, snapshot-scoped description of canonical legal
  entity labels, entity types, relationship types, and supported paths available to Hybrid
  planning.
- **Retrieval stage contribution**: The attributable evidence, outcome, timing, and available
  measurements produced by vector retrieval, graph planning, or graph retrieval for one answer.
- **Hybrid research record**: The inspectable account of the stages, decisions, evidence, paths,
  citations, and outcomes that produced a Hybrid answer.

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: In automated seeded journeys for both curated corpora, 100% of new Vector and
  Hybrid requests run vector retrieval against only the selected active snapshot.
- **SC-002**: In automated broad-question journeys where the graph has no relevant path, 100% of
  Hybrid requests retain available vector evidence and clearly report a graph-skipped or
  graph-no-evidence outcome.
- **SC-003**: In automated relationship-focused journeys with a supported graph path, 100% of
  Hybrid answers expose separate vector and graph contributions, and every graph contribution
  resolves to an immutable source location in the selected snapshot.
- **SC-004**: In automated graph-unavailable and graph-failure journeys, 100% of Hybrid requests
  preserve available vector results and expose a localized, actionable graph-stage diagnostic.
- **SC-005**: In usability verification for English and Portuguese, a researcher can select an
  approach and identify whether the graph contributed to a Hybrid answer in no more than two
  interactions after the answer appears.
- **SC-006**: In automated no-evidence journeys, 100% of final responses avoid unsupported legal
  claims while offering a localized safe next step.
- **SC-007**: For the seeded POC corpora, Hybrid planning and retrieval complete within the
  documented cost and latency budget, with stage-level measurements available for review.

## Assumptions

- Feature 008 supplies snapshot-scoped graph releases, evidence-backed graph paths, and the
  graph-release lifecycle that this feature refines.
- Vector retrieval remains the primary baseline for all questions; graph evidence is an optional
  augmentation rather than a replacement for semantic passages.
- The graph capability description is limited to published, evidence-backed graph vocabulary and
  paths from the active snapshot.
- The initial planner is conservative: it skips graph retrieval when it cannot identify a
  supported useful contribution.
- An already rendered standalone Graph result remains readable during its current browser session
  but is not recreated or compared by new requests.
- This feature does not introduce cross-corpus research, arbitrary graph authoring, external
  legal knowledge, MCP tools, or a general-purpose agent planner.

## Norvii Feature Requirements *(mandatory)*

### Scope and Boundaries

- **In scope**: Vector-first retrieval, adaptive Hybrid planning, optional graph augmentation,
  removal of new standalone Graph selection, stage-level research records, safe fallback,
  localized status, snapshot constraints, and deterministic evaluation scenarios.
- **Out of scope**: Cross-corpus evidence, direct user-authored graph queries, arbitrary graph
  traversal, hidden chain-of-thought display, external legal knowledge, MCP tools, broad agentic
  planning, and a general evaluation dashboard.

### Prototype Baseline *(mandatory for production UI features)*

- **Approved baseline**: The production workspace from Feature 002; grounded chat from Feature
  005; citation and inspection patterns from Feature 006; snapshots from Feature 007; and the
  GraphRAG visual language introduced in Feature 008.
- **Intentional differences**: The strategy selector presents Vector and Hybrid rather than
  Vector, Graph, and Hybrid. The research record makes the adaptive graph decision visible while
  keeping it secondary to the grounded answer and citations.

### Evidence and Corpus Boundaries *(mandatory for retrieval, chat, or citations)*

- **Active corpus constraint**: Every vector result, plan, graph capability, graph path,
  evidence location, citation, answer, and research record is bound to exactly one selected
  corpus and active immutable snapshot.
- **Evidence behavior**: Vector and graph contributions are traceable to official text. A graph
  path supplies navigable context but never substitutes for a cited legal source. Where evidence
  is inadequate, conflicting, or absent, Norvii abstains from material legal claims.

### Superseded Feature 008 Behavior

For new requests, this feature supersedes Feature 008 requirements and scenarios that expose or
compare standalone Graph retrieval. Feature 008 continues to own graph-release construction,
snapshot activation, graph provenance, and graph-path evidence; this feature owns the researcher
choice and adaptive retrieval policy built on that foundation.
