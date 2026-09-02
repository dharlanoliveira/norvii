# Feature Specification: Evidence-Backed Normative Assertions

**Feature Branch**: `011-normative-assertions`

**Created**: 2026-08-25

**Status**: Verified

**Input**: User description: "Replace the current direct legal graph relations with a structure in which a law, title, chapter, section, article, paragraph, or item can establish a semantically typed legal assertion with exact evidence. After the migration is verified, remove all existing local corpus data and reingest from zero using the new model."

## User Scenarios & Testing *(mandatory)*

### User Story 1 - Inspect an exact legal assertion (Priority: P1)

An investigator asks a question about the legal relationship established by a provision and receives only the relevant relationship, its two legal concepts, the exact legal unit that establishes it, and a resolvable source location.

**Why this priority**: The graph is useful only if each semantic claim is traceable to the specific normative text that establishes it instead of appearing as an unexplained relation between broad document concepts.

**Independent Test**: A prepared legal document with assertions established by an article and by an item is queried. Each returned assertion identifies its predicate, subject, object, establishing unit, evidence unit, and source location.

**Acceptance Scenarios**:

1. **Given** an article establishes an obligation for a legal actor, **When** an investigator asks who has that obligation, **Then** the result identifies the article as the establishing unit and cites the article or a more precise descendant evidence unit.
2. **Given** an item establishes a definition, **When** an investigator asks for the definition, **Then** the result identifies the item rather than incorrectly attributing the statement directly to its chapter or law.
3. **Given** a relationship lacks an exact establishing unit or evidence location, **When** it is considered for an answer, **Then** it is excluded from grounded graph evidence.

---

### User Story 2 - Explore a chapter without over-retrieving (Priority: P2)

An investigator asks about a chapter and obtains relevant assertions established in its descendants, together with the minimal hierarchy context required to understand the source, rather than every chapter or every provision in the law.

**Why this priority**: A chapter is a navigational scope, not necessarily the author of all normative statements within it. Descending through its hierarchy produces focused, explainable evidence.

**Independent Test**: A document contains multiple chapters with distinct assertions. A chapter-scoped question returns only matching assertions whose establishing units are descendants of the requested chapter, plus their ancestor path.

**Acceptance Scenarios**:

1. **Given** a chapter contains several articles and only one establishes the requested relationship, **When** an investigator asks the chapter-scoped question, **Then** the answer includes that article's assertion and does not include unrelated sibling articles.
2. **Given** a legal unit directly establishes a relationship, **When** the relationship is retrieved through an ancestor scope, **Then** the result preserves the direct unit as the assertion owner and presents the ancestor only as hierarchy context.
3. **Given** no matching descendant assertion exists, **When** an investigator asks the chapter-scoped question, **Then** the graph contributes no unsupported relationship and normal grounded-answer behavior remains in effect.

---

### User Story 3 - Start the corpus from the new model (Priority: P3)

An operator can remove all existing local corpus data only after the new graph structure is migrated and verified, then reingest the chosen sources from zero so every active graph assertion follows the new model.

**Why this priority**: A clean reingestion removes ambiguous legacy graph relationships and ensures that active evidence is generated consistently by one model.

**Independent Test**: With pre-existing local corpus data present, the migration is verified with controlled data, the local corpus data is removed, and a fresh ingestion creates a new snapshot whose graph assertions all satisfy the new provenance rules.

**Acceptance Scenarios**:

1. **Given** pre-existing local corpus data, **When** the migration and controlled-data validation have succeeded, **Then** the operator can remove the old corpus records, document versions, snapshots, graph releases, and derived graph projection in one documented operation.
2. **Given** the local corpus data has been removed, **When** a configured source is ingested, **Then** the new snapshot and graph release contain only legal units, entities, and normative assertions produced by the new model.
3. **Given** migration validation fails, **When** an operator attempts the destructive reset, **Then** the reset is blocked and the existing local data remains intact.

### Edge Cases

- A provision may contain structural text only. It remains in the legal-unit hierarchy but establishes no normative assertion.
- A statement may be evidenced by a descendant item while being established by its parent article. Both locations must be retained and distinguishable.
- An extraction may identify a relationship but cannot resolve one endpoint to a legal entity. The candidate must not be published as a complete assertion.
- A source may list independently addressable legal actors in one grammatical phrase. The extraction must emit one entity and assertion per actor rather than a comma-separated aggregate entity.
- A collective legal body may remain one entity only when the source treats that body as an indivisible legal subject.
- An assertion may use a valid predicate but fall outside the requested hierarchy scope. It must not be returned merely because it mentions the same entity.
- A reset may be interrupted after derived graph data is removed but before fresh ingestion completes. The system must present an empty local corpus state rather than presenting stale graph evidence.

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: The system MUST represent every normalized legal location as a legal unit with a stable identity, unit kind, source locator, and optional parent legal unit.
- **FR-002**: The system MUST represent the directed hierarchy between legal units separately from semantic legal assertions.
- **FR-003**: The system MUST represent each published legal assertion as a distinct, evidence-backed record with one predicate, one subject legal entity, one object legal entity, one establishing legal unit, and one evidence legal unit.
- **FR-004**: The system MUST allow any legal-unit kind in the supported hierarchy to establish an assertion; an assertion MUST NOT be duplicated onto its ancestors solely for retrieval convenience.
- **FR-005**: The system MUST support the following normative predicates with these directions and meanings: `defines` from a legal term to its definition; `applies_to` from a norm to a covered person, activity, or situation; `must_be_observed_by` from a norm to an obligated public body; `imposes_duty_on` from a norm to an obligated actor; `grants` from a norm to a right or beneficiary; `protects` from a norm to a protected right or interest; `assigns_responsibility_to` from a norm to a responsible authority or actor; and `conditions` from a legal effect or duty to a condition of application.
- **FR-006**: The system MUST retain any qualifier or condition needed to avoid turning a conditional legal statement into an unconditional relationship.
- **FR-007**: The system MUST reject incomplete assertions and assertions whose establishing or evidence unit cannot be resolved within the same corpus snapshot.
- **FR-007a**: Each legal entity MUST represent exactly one legally addressable referent. A coordinated list of independently addressable entities MUST be decomposed into one entity and one assertion per member; it MUST NOT be stored as a comma-separated aggregate entity.
- **FR-008**: The system MUST make the establishing unit, evidence unit, predicate, subject, object, and hierarchy context available to grounded retrieval and inspection behavior.
- **FR-009**: For a hierarchy-scoped request that identifies a published legal location, the system MUST select that location only from the active snapshot's legal-unit catalog, return only assertions established by matching descendant legal units, and return only the minimal hierarchy context needed to explain each result.
- **FR-010**: The system MUST preserve snapshot isolation: assertions, legal units, entities, and retrieval results MUST never cross corpus-snapshot boundaries.
- **FR-011**: The system MUST provide a documented destructive-reset operation that removes all local corpus records, document versions, snapshots, graph releases, and derived graph projection only after schema migration and controlled-data validation have succeeded.
- **FR-012**: The destructive-reset operation MUST fail safely before changing data if migration readiness or controlled-data validation is unsuccessful.
- **FR-013**: After a successful reset, the system MUST prevent retrieval from using stale graph evidence and MUST expose the empty corpus state until a new snapshot is published and activated.
- **FR-014**: The first corpus reingestion after the reset MUST generate legal units, entities, and normative assertions exclusively through the new model; direct legacy semantic relationships MUST NOT be generated.
- **FR-015**: The system MUST expose enough retrieval-decision information to determine whether a graph result came from a normative assertion, which legal units supplied its evidence, and why the selected hierarchy scope was used.

### Key Entities

- **Legal Unit**: A document location that can participate in the legal hierarchy and can establish or evidence a normative assertion. Its kind may be a law, title, chapter, section, article, paragraph, or item.
- **Legal Entity**: A canonical legal concept, actor, right, obligation, condition, activity, or other concept that can be the subject or object of an assertion.
- **Normative Assertion**: An evidence-backed semantic statement established by one legal unit and relating a subject entity to an object entity through a defined predicate.
- **Corpus Snapshot**: The isolated, versioned corpus boundary that owns legal units, entities, assertions, and their retrieval results.
- **Hierarchy Context**: The minimum ancestor path that explains how an assertion's establishing unit belongs to a requested legal scope.

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: In controlled document fixtures, 100% of published normative assertions resolve both endpoints, an establishing legal unit, an evidence legal unit, and a source locator within one corpus snapshot.
- **SC-002**: In a document containing at least three sibling legal scopes, a scope-specific relationship query returns no assertion established outside the requested scope.
- **SC-003**: In controlled chapter queries, every returned assertion preserves its direct establishing legal unit and contains no copied assertion attributed to an ancestor.
- **SC-004**: Once migration and controlled-data validation succeed, a documented reset removes every local corpus record, document version, snapshot, graph release, and derived graph-projection record; no stale graph result remains queryable.
- **SC-005**: The first completed post-reset ingestion produces a fresh active snapshot whose graph assertions all meet SC-001 through SC-003.
- **SC-006**: An investigator can determine the predicate, two endpoints, exact evidence location, and ancestor context for every graph assertion used in an answer without consulting operational logs.

## Assumptions

- The existing legal-unit normalization provides stable document locations that can be used as the basis for the required hierarchy.
- The user has authorized deletion of all local corpus data after migration and controlled-data validation; this authorization does not include external source deletion.
- The initial controlled-data validation may use deterministic fixtures and must complete before deleting any local corpus data.
- Existing local direct graph relations are removed by the reset and are not upgraded in place into normative assertions.
- The existing evidence-grounded answer rules, active-corpus boundary, and abstention behavior remain authoritative.

## Out of Scope

- Automatically inferring new assertions from legacy direct graph relations.
- Expanding the predicate vocabulary beyond the predicates listed in FR-005.
- Treating the graph as an independent source of legal authority without resolvable corpus evidence.
