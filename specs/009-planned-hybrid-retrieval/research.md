# Research: Planned Hybrid Retrieval

## Decision 1: Always retrieve vector evidence first

**Decision**: Both Vector and Hybrid requests run snapshot-scoped vector retrieval before any
optional graph work.

**Rationale**: Semantic passages are the reliable baseline for broad, document-level, and
unanticipated questions. A graph miss must not suppress valid semantic evidence.

**Alternatives considered**:

- Select vector or graph through query token matching. Rejected because generic question terms do
  not reliably express whether legal relationships can add value.
- Require graph evidence for Hybrid. Rejected because it makes Hybrid fail on useful broad
  questions and misuses the graph as a replacement for source text.

## Decision 2: Plan graph augmentation from a bounded capability catalog

**Decision**: Hybrid calls a planner with the question and a compact active-snapshot catalog of
graph entity types, relationship types, and canonical entity labels. The planner returns a
validated decision to use or skip the graph plus bounded relationship-type and canonical-entity
filters. A selected canonical label may differ from the language used in the question.

**Rationale**: The model can recognize that a question about an actor, right, obligation, or
relationship may benefit from graph context, while application code retains query authority and
scope boundaries. Canonical labels prevent an English question from missing an otherwise relevant
Portuguese graph entity through a literal text comparison.

**Alternatives considered**:

- Let the model generate graph query text. Rejected because it would create an unsafe,
  unbounded query surface and make provenance difficult to validate.
- Give the model the entire graph or document text. Rejected because it increases cost and prompt
  exposure without improving the small POC's decision boundary.

## Decision 3: Preserve Hybrid when graph does not contribute

**Decision**: A Hybrid answer remains labelled Hybrid even when graph planning skips the graph,
the graph has no evidence, or the graph stage safely fails after vector retrieval. Its research
record states the exact non-contribution state.

**Rationale**: Hybrid identifies the chosen adaptive approach. Calling the result Vector would
hide that planning occurred; calling a graph miss a request failure discards valid evidence.

**Alternatives considered**:

- Relabel every vector-only Hybrid result as Vector. Rejected because it conceals requested
  adaptive behavior and loses auditability.
- Return an error if graph augmentation cannot contribute. Rejected because no graph
  contribution is expected for many valid questions.

## Decision 4: Keep standalone Graph historical only

**Decision**: New public requests and comparisons support Vector and Hybrid. An already rendered
Graph result remains readable during the current browser session.

**Rationale**: The strategy selector communicates a meaningful user decision while preserving
traceability for the POC's in-session earlier results.

**Alternatives considered**:

- Rewrite an already-rendered Graph result. Rejected because a strategy change must not alter the
  provenance shown for an existing in-session answer.

## Decision 5: Report stages instead of one ambiguous retrieval outcome

**Decision**: Each terminal inspection contains explicit vector, planning, graph, and generation
stage records. Every record has state, safe reason when relevant, elapsed time, evidence count,
and provider-reported token use when available.

**Rationale**: A request can complete even if a stage finds no evidence. Separate records prevent
the UI from equating successful transport execution with a graph contribution.

**Alternatives considered**:

- Infer stage state from answer wording. Rejected because it is brittle, inaccessible, and hides
  execution facts in generated text.
