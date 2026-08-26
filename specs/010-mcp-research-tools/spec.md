# Feature Specification: MCP Research Tools and Skills

**Feature Branch**: `010-mcp-research-tools`

**Created**: 2026-08-25

**Status**: In Progress

**Input**: User description: "Expose curated corpus research capabilities through a versioned MCP server and reusable evidence-grounded research skills, preserving active corpus and immutable snapshot boundaries, traceable citations, safe failures, and the existing web and HTTP experiences."

## Clarifications

### Session 2026-08-25

- Q: How will the MCP server be operated in the final environment? -> It runs as a
  Docker service through a Streamable HTTP endpoint. Development may also use the
  stdio transport. The service remains private to the Docker network by default, with
  optional local host access limited to `127.0.0.1`.

## User Scenarios & Testing *(mandatory)*

### User Story 1 - Research a Curated Corpus Through MCP (Priority: P1)

An evaluator connects an MCP-capable research client, discovers the available corpus
research tools, selects a corpus, searches its published material, and opens an
identified legal unit with its traceable evidence.

**Why this priority**: This makes the existing corpus, retrieval, and citation
capabilities usable by an external research workflow without recreating their logic
or bypassing their safeguards.

**Independent Test**: Connect a configured MCP client to the containerized endpoint,
list the available corpora, list documents in one corpus, search for a prepared term,
and retrieve one returned article. Verify that every corpus-scoped response identifies
one immutable snapshot and that the article is reachable through the returned evidence
reference.

**Acceptance Scenarios**:

1. **Given** a configured trusted client connected through the containerized MCP
   endpoint and at least one published corpus, **When**
   the evaluator discovers the research capabilities, **Then** the client receives
   clear, versioned descriptions of the available tools and reusable skills.
2. **Given** a corpus with an active published snapshot, **When** the evaluator
   lists documents, searches the corpus, and retrieves an identified article,
   **Then** every result is limited to that corpus and identifies the snapshot used.
3. **Given** a search result containing evidence, **When** the evaluator follows its
   reference, **Then** the returned legal unit and its location match the referenced
   immutable document version.

---

### User Story 2 - Inspect Related Provisions Safely (Priority: P2)

An evaluator uses relationship and comparison capabilities to inspect related legal
provisions in one corpus and receives only published, evidence-backed paths and
comparisons.

**Why this priority**: It demonstrates the project's graph and Hybrid retrieval
value through explicit, inspectable research operations rather than hidden agent
behavior.

**Independent Test**: In a corpus with a ready graph release, find related articles,
traverse a prepared legal relationship, and compare two identified provisions. Verify
that every returned path and comparison includes evidence from the selected snapshot.

**Acceptance Scenarios**:

1. **Given** a corpus with a ready graph release, **When** the evaluator requests
   related articles or a bounded relationship traversal, **Then** the result contains
   only snapshot-scoped relationships with their supporting evidence locations.
2. **Given** two provisions in the same corpus, **When** the evaluator requests a
   comparison, **Then** the result distinguishes source text from any generated
   synthesis and cites the supporting locations.
3. **Given** a corpus without a ready graph release, **When** the evaluator requests
   a graph-dependent operation, **Then** the client receives a safe unavailable
   outcome and no substitute result from another corpus or snapshot.

---

### User Story 3 - Run Reusable Evidence Workflows (Priority: P3)

An evaluator invokes a reusable research skill to conduct evidence-grounded research,
compare provisions, or verify citation support without manually assembling each
underlying research operation.

**Why this priority**: Reusable workflows demonstrate disciplined tool composition:
they make the research method visible, preserve evidence requirements, and provide a
repeatable outcome for future MCP clients.

**Independent Test**: Invoke each initial skill against prepared corpus scenarios and
verify that its outcome states the corpus and snapshot used, separates evidence from
interpretation, contains traceable citations for material claims, and abstains when
the required evidence is unavailable.

**Acceptance Scenarios**:

1. **Given** a selected corpus with sufficient published evidence, **When** the
   evaluator invokes the evidence-grounded research skill, **Then** the outcome
   records its research steps, answers only from available evidence, and cites each
   material legal claim.
2. **Given** two supported provisions, **When** the evaluator invokes the provision
   comparison skill, **Then** the outcome presents the relevant passages and a
   traceable, evidence-bounded comparison.
3. **Given** an answer or research result with incomplete support, **When** the
   evaluator invokes the citation-support verification skill, **Then** the outcome
   identifies the unsupported or unavailable claim without inventing evidence.

### Edge Cases

- A client requests a corpus, document, article, or evidence reference that is
  malformed, unavailable, disabled, outside the selected corpus, or absent from the
  selected snapshot.
- A published corpus has no ready documents, no retrieval evidence, or no ready graph
  release.
- A tool request exceeds its documented result, traversal, or comparison bounds.
- A reusable skill receives instruction-like corpus text, conflicting evidence, or an
  unsupported request for legal advice.
- A provider, retrieval dependency, or graph dependency is unavailable during a
  request.

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: The system MUST let a trusted MCP client discover the research tools,
  reusable skills, their purpose, accepted inputs, result shape, access conditions,
  and stable error categories.
- **FR-002**: The system MUST provide the following initial research tools:
  `list_corpora`, `list_documents`, `search_documents`, `get_article`,
  `get_document_metadata`, `find_related_articles`, `traverse_legal_graph`, and
  `compare_provisions`.
- **FR-003**: The system MUST require an explicit corpus selection for every
  corpus-scoped tool and MUST restrict its result, evidence, and errors to that
  corpus.
- **FR-004**: The system MUST bind every corpus-scoped tool invocation to one
  published immutable snapshot, return that snapshot identity in the result, and
  prevent unpublished or superseded candidate content from affecting the result.
- **FR-005**: The system MUST return stable source, document-version, legal-unit, and
  location references whenever a tool result contains retrieved text, a relationship,
  a comparison, or another material research claim.
- **FR-006**: The system MUST make an evidence reference resolvable to the exact
  immutable legal location within the selected corpus or return a safe unavailable
  outcome; it MUST NOT substitute newer, foreign, or approximate content.
- **FR-007**: The system MUST ensure that graph-dependent tools return only
  published, snapshot-scoped graph paths supported by canonical evidence locations.
- **FR-008**: The system MUST return a distinct safe unavailable outcome for a
  graph-dependent request when a matching graph release or graph dependency is not
  ready; it MUST NOT silently broaden, switch, or substitute the requested strategy.
- **FR-009**: The system MUST limit each tool invocation to its documented scope and
  bounds and MUST NOT expose arbitrary data-store queries, unbounded graph traversal,
  provider payloads, private prompts, secrets, or internal reasoning.
- **FR-010**: The system MUST provide safe, actionable failures for invalid input,
  missing evidence, unavailable dependencies, unsupported operations, and access
  denial without exposing protected corpus content or internal diagnostics.
- **FR-011**: The system MUST provide three reusable initial skills: evidence-grounded
  legal research, provision comparison, and citation-support verification.
- **FR-012**: Each reusable skill MUST state the corpus and snapshot used, declare the
  evidence it relied on, distinguish source text from interpretation, cite every
  material legal claim, and abstain rather than invent evidence.
- **FR-013**: The system MUST preserve the existing browser and public application
  HTTP experiences; MCP research access is an additional capability and does not
  replace or weaken their contracts.
- **FR-014**: The system MUST define versioned, language-neutral contracts for MCP
  tool inputs, outputs, errors, limits, compatibility expectations, and the evidence
  reference format, with provider and consumer verification.
- **FR-015**: The system MUST record content-safe measurements for each tool and
  skill invocation, including outcome, corpus and snapshot identifiers, selected
  strategy when applicable, latency, and bounded usage measurements.
- **FR-016**: The system MUST present a clear technical-demonstration and
  non-legal-advice boundary wherever a skill produces interpretative guidance.
- **FR-017**: The system MUST operate the production MCP server as a Docker service
  using Streamable HTTP at one documented MCP endpoint. A local stdio transport MAY
  be available for development and MUST provide equivalent tool and skill contracts.
- **FR-018**: The containerized MCP service MUST be reachable by other authorized
  services on the Norvii Docker network and MUST expose a health signal suitable for
  container orchestration.
- **FR-019**: The containerized MCP endpoint MUST remain private to the Docker network
  by default. Any host publication MUST bind only to `127.0.0.1` in the local POC;
  public or remote exposure requires authenticated access, encrypted transport, and
  Origin validation.
- **FR-020**: The MCP service MUST shut down safely when its container stops, cancel
  in-flight work according to the selected transport, and release database and graph
  resources without emitting protocol-invalid output.

### Key Entities

- **MCP Tool**: A discoverable, bounded research operation with a versioned contract,
  declared access conditions, stable inputs, outcomes, errors, and limits.
- **MCP Skill**: A discoverable reusable research workflow that composes one or more
  tools and defines its evidence, citation, abstention, and output rules.
- **Tool Invocation**: One auditable request and outcome, bound to the requesting
  client, selected corpus, immutable snapshot, and documented result limits.
- **Evidence Reference**: A stable reference to an immutable document version, legal
  unit, and exact location that supports a tool or skill result.
- **Research Outcome**: The evidence, source text, interpretation when applicable,
  citations, abstention or failure state, and safe measurements returned by a tool or
  skill.

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: A configured trusted client can discover and invoke all eight initial
  research tools and all three initial skills in 100% of deterministic acceptance
  scenarios.
- **SC-002**: In 100% of acceptance scenarios containing evidence, every evidence
  reference resolves to the selected corpus and immutable snapshot, or the operation
  returns its documented unavailable outcome.
- **SC-003**: In 100% of graph-unavailable, foreign-corpus, foreign-snapshot, and
  malformed-reference scenarios, no result exposes substituted or out-of-scope corpus
  content.
- **SC-004**: In prepared corpus scenarios, an evaluator can locate an evidence-backed
  article from a discovered corpus using no more than three tool invocations.
- **SC-005**: In 100% of prepared skill scenarios, every material legal claim is
  supported by a traceable evidence reference or the skill explicitly abstains.
- **SC-006**: At least 95% of bounded, non-generative tool invocations against the
  seeded local corpora complete within five seconds, measured across 100 consecutive
  successful requests under the documented local environment.
- **SC-007**: A configured MCP client running outside the MCP container can discover
  and invoke each initial tool through the documented container endpoint in 100% of
  deterministic Docker acceptance scenarios.
- **SC-008**: In 100% of Docker acceptance scenarios, the MCP service is not reachable
  from a non-Norvii network unless an explicit local host publication is configured.

## Assumptions

- The initial scope remains a local, trusted-user proof of concept. The final local
  runtime is Docker, with Streamable HTTP as its primary transport; multi-user
  tenancy, public exposure, and delegated authorization are outside this feature.
- Only the two curated corpora and their published snapshots are available to initial
  MCP clients.
- Existing corpus, document, retrieval, Hybrid planning, graph-release, and citation
  capabilities provide the evidence-facing behavior that this feature exposes; this
  feature does not introduce a new retrieval strategy or new corpus content.
- The initial skills are limited to research, provision comparison, and citation
  support verification. Security-incident analysis, structured summarization, and
  evaluation question generation remain candidates for later features.
- MCP clients can present structured outcomes and retain the returned corpus,
  snapshot, and evidence identities; no new browser user interface is required for
  this feature.

## Norvii Feature Requirements *(mandatory)*

### Scope and Boundaries

- **In scope**: Discoverable and versioned MCP access to the eight defined corpus
  research tools; the three initial reusable evidence workflows; immutable evidence
  references; safe outcomes; stdio development access; a containerized Streamable
  HTTP service; network-default isolation; contract compatibility; and content-safe
  invocation inspection.
- **Out of scope**: Corpus or source mutation; arbitrary data-store or graph queries;
  new retrieval or graph algorithms; changes to browser or public application HTTP
  behavior; persistent conversations; public or remote MCP deployment; multi-user tenancy;
  specialized legal advice; automated evaluation dashboards; and new corpus ingestion.

### Prototype Baseline *(mandatory for production UI features)*

- **Approved baseline**: N/A. This feature introduces no production browser user
  interface and must preserve the existing approved web experience.
- **Intentional differences**: N/A.

### Evidence and Corpus Boundaries *(mandatory for retrieval, chat, or citations)*

- **Active corpus constraint**: Every corpus-scoped invocation explicitly selects one
  corpus and is bound to its active immutable snapshot for the duration of that
  invocation. No result may cross that boundary.
- **Evidence behavior**: Retrieved, graph-derived, compared, and interpreted results
  expose immutable evidence references. Skills cite material claims and abstain for
  missing, weak, conflicting, or unavailable evidence; they do not provide legal
  advice.
