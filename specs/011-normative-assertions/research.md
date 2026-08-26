# Research: Evidence-Backed Normative Assertions

## Decision 1: Reuse `document_units` as the legal-unit authority

**Decision**: Do not create a second canonical legal-unit table. Use the existing normalized `document_units` rows and their parent relationship for every law, title, chapter, section, article, paragraph, and item.

**Rationale**: `document_units` already has stable IDs, kind, locator, offsets, and an ownership-preserving parent reference. A duplicate would introduce synchronization risk without adding legal meaning.

**Alternatives considered**:

- Create a `legal_units` mirror: rejected because it duplicates canonical source locations.
- Continue treating locations as semantic entities: rejected because structural navigation and legal concepts have different identity and validation rules.

## Decision 2: Replace direct semantic relationships with normative assertions

**Decision**: Replace `semantic_relationships` with `normative_assertions` after the verified local reset. An assertion stores its extraction run, document ownership, subject entity, object entity, establishing unit, evidence unit, predicate, qualifier, and validation state.

**Rationale**: The user authorized removal of local legacy data. A replacement eliminates ambiguous direct-edge compatibility and makes provenance mandatory rather than optional.

**Alternatives considered**:

- Keep both models indefinitely: rejected because it creates two meanings for graph evidence.
- Upgrade existing direct rows in place: rejected because old rows cannot prove a separate establishing unit or satisfy the new predicate semantics.

## Decision 3: Project assertions as nodes, not dynamic relationship types

**Decision**: Project `LegalUnit`, `LegalEntity`, and `NormativeAssertion` as distinct Neo4j nodes. Keep the normalized predicate as an assertion property and use fixed edges for hierarchy, establishment, subject, and object.

**Rationale**: It makes the assertion itself queryable and traceable, keeps predicates allowlisted, and separates structural hierarchy from legal meaning.

**Alternatives considered**:

- Store all provenance as properties on an entity-to-entity edge: rejected because one edge becomes the owner of several independent identities and cannot naturally preserve an establishment path.
- Generate one Neo4j relationship type per predicate: rejected because dynamic types complicate query validation and omit the assertion's own identity.

## Decision 4: Enforce predicate semantics at extraction and persistence boundaries

**Decision**: Permit only `defines`, `applies_to`, `must_be_observed_by`, `imposes_duty_on`, `grants`, `protects`, `assigns_responsibility_to`, and `conditions`. The extractor prompt, typed validation, database check constraint, and retrieval planner all use this same vocabulary and direction.

**Rationale**: Broad predicates such as `requires` and `governs` hide legal roles and led to unreliable planner choices. Repeated enforcement prevents vocabulary drift across stages.

**Alternatives considered**:

- Permit arbitrary model predicates: rejected because they cannot be planned or queried safely.
- Infer a replacement predicate from legacy rows: rejected because it would create unsupported legal claims.

## Decision 5: Scope graph retrieval through the legal hierarchy

**Decision**: For a requested legal scope, match the scope legal unit, traverse only its descendants within a bounded depth, then select assertions established by those units. Return the direct establishing/evidence locations and a minimal ancestor path.

**Rationale**: A chapter provides context for article-level rules but is not automatically their author. Descendant-first selection avoids returning unrelated chapters or all provisions.

**Alternatives considered**:

- Copy all child assertions to their ancestors: rejected because it loses exact authorship and inflates results.
- Retrieve every unit in the document then filter in the answer model: rejected because it spends context on irrelevant evidence and weakens grounded retrieval.

## Decision 6: Gate destructive reset on completed migration and controlled-data validation

**Decision**: Extend the existing volume-owned local reset operation so it runs only after schema migrations and deterministic controlled-data validation complete successfully. It stops local services and removes the verified PostgreSQL and Neo4j volumes together. The next bootstrap recreates the schema and configured seed sources but not legacy corpus artifacts. It leaves application configuration and external source systems untouched.

**Rationale**: The user's approved reset is irreversible for local corpus data. A preflight gate prevents deleting a working corpus for a migration that cannot produce valid assertions.

**Alternatives considered**:

- Drop Docker volumes before migration: rejected because it obscures migration failures and destroys schema-level diagnostic evidence.
- Delete PostgreSQL rows only: rejected because stale Neo4j paths could remain queryable.
- Preserve old snapshots: rejected because the requested fresh ingestion requires one consistent model.
