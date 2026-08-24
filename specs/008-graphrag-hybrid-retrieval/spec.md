# Feature Specification: GraphRAG and Hybrid Retrieval

**Feature Branch**: `008-graphrag-hybrid-retrieval`

**Created**: 2026-08-24

**Status**: Draft

**Input**: User description: "GraphRAG and hybrid retrieval: enable evaluators to compare snapshot-scoped vector, graph, and hybrid evidence paths for the curated Portuguese and English legal corpora."

## User Scenarios & Testing *(mandatory)*

### User Story 1 - Answer a connected legal question with grounded graph evidence (Priority: P1)

A researcher asks a question whose answer depends on connected legal concepts, provisions, rights,
obligations, actors, or references. Norvii uses the selected evidence strategy and returns either
a cited, corpus- and snapshot-scoped answer or an honest abstention.

**Why this priority**: The POC must demonstrate that GraphRAG adds an evidence-led capability
that a vector-only search cannot explain: following explicit, inspectable legal relationships.

**Independent Test**: In each curated corpus, ask one seeded connected-question scenario and
verify that a graph or hybrid answer cites only locations in the selected published snapshot and
identifies the strategy used.

**Acceptance Scenarios**:

1. **Given** a selected corpus has an active snapshot with a ready graph release, **When** a
   researcher asks a connected legal question using graph retrieval, **Then** Norvii returns only
   cited evidence from that snapshot and exposes the graph path that led to the evidence.
2. **Given** a selected corpus has an active snapshot with a ready graph release, **When** a
   researcher asks the same question using hybrid retrieval, **Then** Norvii combines relevant
   semantic passages and graph-connected evidence without including another corpus or snapshot.
3. **Given** the selected strategy lacks sufficient grounded evidence, **When** Norvii completes
   the request, **Then** it abstains or explains that the strategy is unavailable; it does not
   represent a vector-only answer as graph or hybrid retrieval.

---

### User Story 2 - Publish and inspect an evidence-backed legal graph (Priority: P2)

A corpus maintainer can see whether the active snapshot has a graph release and inspect the
evidence-backed legal concepts and relationships that make graph retrieval possible. A newly
ingested candidate cannot change the graph available to researchers until it is explicitly
published and its graph release is ready.

**Why this priority**: Graph-derived facts can be misleading unless their source location,
snapshot, and publication state are clear. The POC must show that the graph is a traceable view
of official material, not an independent source of legal truth.

**Independent Test**: Process a changed candidate, confirm that it remains unavailable to graph
retrieval, publish it, and verify that its graph release becomes inspectable only when its
evidence-backed projection is ready.

**Acceptance Scenarios**:

1. **Given** an active snapshot with a ready graph release, **When** a maintainer inspects its
   graph summary, **Then** the summary identifies the snapshot and shows the supported legal
   entities and relationships with links to their evidence locations.
2. **Given** a candidate source revision has completed ingestion but is not published, **When** a
   researcher uses graph or hybrid retrieval, **Then** no candidate entity, relationship, or
   evidence location is returned.
3. **Given** a graph release cannot be built or validated, **When** a maintainer inspects the
   snapshot, **Then** Norvii reports the safe failure state and preserves the preceding ready
   graph release for its preceding snapshot.

---

### User Story 3 - Compare retrieval strategies for one research question (Priority: P3)

An evaluator runs the same question against vector, graph, and hybrid retrieval for one selected
snapshot and compares the resulting evidence, legal paths, outcome, elapsed time, and token use.

**Why this priority**: The portfolio value is not merely that a graph exists; it is that its
effect on retrieval can be inspected fairly against the existing vector baseline.

**Independent Test**: Run one fixed question against each available strategy for the same corpus
and snapshot, and verify that the comparison keeps the strategy, evidence set, path, outcome,
and measurements distinct.

**Acceptance Scenarios**:

1. **Given** vector, graph, and hybrid retrieval are available for an active snapshot, **When**
   an evaluator runs the same question with each strategy, **Then** Norvii presents one distinct,
   attributable result per strategy without combining their evidence or measurements.
2. **Given** one strategy is unavailable for the selected snapshot, **When** an evaluator opens
   the comparison, **Then** Norvii labels that strategy unavailable and preserves the results of
   the other completed strategies.
3. **Given** an evaluator opens a graph path from a comparison result, **When** they select an
   associated evidence location, **Then** the source view opens the exact immutable legal location
   while the comparison result remains available.

### Edge Cases

- The question mentions no entity or relationship represented in the selected snapshot.
- A model-derived relationship has insufficient or conflicting source evidence.
- A graph release is pending, failed, stale, or belongs to a different snapshot.
- A document is reingested while a researcher has an earlier graph or hybrid answer open.
- Multiple legal units express the same relationship with different qualifiers or exceptions.
- The interface language differs from the language of the corpus or preserved legal evidence.
- The same concept name appears in both curated corpora.

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: The system MUST support vector, graph, and hybrid retrieval as distinct,
  inspectable evidence strategies for research requests.
- **FR-002**: Every graph and hybrid research request MUST be constrained to the selected corpus
  and its active immutable snapshot. It MUST NOT use candidate, superseded, foreign-corpus, or
  foreign-snapshot graph data or evidence.
- **FR-003**: The system MUST create a graph release only from published snapshot evidence and
  keep the release identity, readiness state, and source snapshot identity inspectable.
- **FR-004**: The system MUST preserve the current graph release for a historical snapshot and
  keep a new candidate revision unavailable to graph and hybrid retrieval until explicit
  publication and graph-release validation succeed.
- **FR-005**: Graph entities and model-derived relationships MUST identify their type, source
  snapshot, supporting immutable evidence location, extraction provenance, and validation state.
- **FR-006**: The initial graph vocabulary MUST remain limited to document structure plus the
  legal concepts, actors, rights, obligations, and evidence-backed relationships required by the
  seeded evaluation questions. It MUST not treat an extracted relationship as an official fact.
- **FR-007**: Graph retrieval MUST return the selected graph path and the evidence locations that
  support each material relationship used in an answer.
- **FR-008**: Hybrid retrieval MUST make its vector and graph contributions separately
  inspectable, deduplicate shared evidence locations, and ground every material answer claim in
  cited official text.
- **FR-009**: When graph or hybrid retrieval is unavailable, incomplete, conflicting, or lacks
  sufficient evidence, the system MUST return a safe, localized outcome. It MUST NOT silently
  substitute another strategy or fabricate a path.
- **FR-010**: The workspace MUST let an evaluator select an available strategy for a question and
  show the strategy, graph-release identity, graph-path summary, evidence locations, outcome,
  elapsed time, and available token-use measurements for each completed result.
- **FR-011**: The comparison workflow MUST run each strategy against the same selected corpus,
  active snapshot, question, and interface language, and keep results, citations, paths, and
  measurements attributable to that strategy.
- **FR-012**: Citation and graph-path navigation MUST open the exact immutable evidence location
  in the source workspace and preserve the originating answer or comparison result.
- **FR-013**: The system MUST support reproducible rebuilding and validation of a graph release
  from its recorded snapshot, extraction provenance, and evidence locations without changing
  canonical legal content or the active snapshot.
- **FR-014**: Product-authored graph, strategy, availability, failure, and inspection text MUST
  follow the selected interface language. Preserved legal text, source titles, user questions,
  and generated answers MUST retain their original language.
- **FR-015**: Graph extraction, projection, and retrieval MUST record content-free operational
  measurements and safe failure categories without exposing prompts, credentials, complete model
  payloads, or unrelated corpus content.

### Key Entities

- **Graph release**: A versioned, rebuildable evidence graph associated with exactly one published
  corpus snapshot and a readiness state.
- **Graph entity**: A typed legal subject, concept, right, obligation, actor, document, or legal
  location represented in a graph release and linked to immutable evidence.
- **Graph relationship**: A typed, evidence-backed connection between graph entities with
  extraction provenance and validation state.
- **Graph path**: The ordered, inspectable relationships and evidence locations used by a graph
  retrieval result.
- **Retrieval strategy result**: The answer or safe outcome, evidence, graph path, and operational
  measurements produced by one strategy for one corpus snapshot and question.

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: In automated seeded connected-question journeys for both curated corpora, 100% of
  graph and hybrid citations, graph paths, and evidence locations resolve only to the selected
  corpus and active snapshot.
- **SC-002**: In deterministic candidate and failed-projection journeys, 100% of candidate,
  stale, failed, or foreign graph releases are excluded from graph and hybrid retrieval.
- **SC-003**: In automated evaluation journeys, each available vector, graph, and hybrid result
  for the same question exposes a distinct strategy label, outcome, evidence set, and available
  execution measurements in no more than three user interactions after submitting the question.
- **SC-004**: In automated graph-path navigation journeys, 100% of selected path evidence opens
  the exact immutable legal location and retains the originating answer or comparison result.
- **SC-005**: A clean local rebuild reproduces the entity and relationship membership and evidence
  locations of each seeded graph release from its recorded snapshot in 100% of three consecutive
  runs.
- **SC-006**: The initial graph extraction and release process completes within the documented POC
  cost and time budget for each curated corpus, with recorded measurements available for review.
- **SC-007**: English and Portuguese interface journeys have complete strategy and graph-inspection
  copy, while preserved legal evidence remains unchanged.

## Assumptions

- Feature 007 provides the two initial curated corpora, immutable source revisions, and active
  snapshot boundary required by this feature.
- The POC starts with a deliberately small legal graph vocabulary and a small, curated question
  set; new entity or relationship types require an evaluation question that justifies them.
- A graph release may be unavailable while a selected snapshot remains available for vector
  retrieval; unavailable graph or hybrid modes are shown honestly rather than downgraded.
- The POC has no separate maintainer or evaluator authentication model. The workspace exposes
  operational controls and inspection to the local demonstration user.
- The feature does not introduce cross-corpus research, scheduled refresh, autonomous
  publication, external legal knowledge, MCP tools, or the formal evaluation dashboard planned
  for later features.

## Norvii Feature Requirements *(mandatory)*

### Scope and Boundaries

- **In scope**: Evidence-backed legal graph extraction, release and rebuild lifecycle,
  snapshot-scoped graph and hybrid retrieval, strategy selection and comparison, graph-path
  inspection, immutable citation navigation, localized operational states, and deterministic
  seeded verification.
- **Out of scope**: MCP tools and reusable skills, end-user graph authoring, cross-corpus graph
  traversal, automated graph publication, free-form ontology expansion, a broad evaluation
  dashboard, legal advice, and use of external knowledge to complete an answer.

### Prototype Baseline *(mandatory for production UI features)*

- **Approved baseline**: The workspace and source navigation from Features 002 and 004, the
  grounded chat from Feature 005, the citation and inspection patterns from Feature 006, and the
  immutable corpus-snapshot boundary from Feature 007.
- **Intentional differences**: The workspace gains a restrained strategy selector and graph-path
  inspection that remain subordinate to the grounded answer, citations, source library, and
  active corpus boundary. The existing editorial research-desk visual language remains intact.

### Evidence and Corpus Boundaries *(mandatory for retrieval, chat, or citations)*

- **Active corpus constraint**: Every entity, relationship, graph path, retrieval artifact,
  citation, answer, and comparison result is bound to exactly one selected corpus and immutable
  snapshot.
- **Evidence behavior**: Graph-derived relationships are attributed interpretations supported by
  exact official evidence locations. Norvii answers only from those cited locations, abstains on
  inadequate or conflicting evidence, and never uses a graph path as a substitute for source
  text.
