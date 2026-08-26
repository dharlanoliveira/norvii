# Data Model: Evidence-Backed Normative Assertions

## Canonical entities

### Legal Unit

`document_units` is the canonical legal-unit record. Its existing immutable document identity, kind, locator, offsets, and `parent_id` form the hierarchy.

| Invariant | Rule |
| --- | --- |
| Identity | `(document_id, id)` identifies one legal unit. |
| Hierarchy | `parent_id` is null only at the document root; a parent and child belong to the same document. |
| Direction | The projection emits `CONTAINS` from parent to child only. |
| Scope | A requested unit includes itself and bounded descendants, never sibling or ancestor assertions. |

### Legal Entity

`semantic_entities` remains the canonical extracted entity record, but its allowed types exclude structural locations. An entity has an immutable evidence unit and belongs to one extraction run and one document.

| Field | Rule |
| --- | --- |
| `id` | Stable UUID scoped to the immutable document version. |
| `entity_type` | `concept`, `actor`, `right`, `obligation`, or `condition`. |
| `label`, `normalized_label` | Non-empty canonical display and matching values. |
| `evidence_unit_id` | Resolves to a legal unit in the same document. |
| `validation_status` | Only `supported` entities may be endpoints of a published assertion. |

Entities are atomic. For example, `Union, States, Federal District, and Municipalities` produces
four actor entities and four assertions with the same predicate and provenance; it is never one
comma-separated entity. A collective stays one entity only when it is itself the legal subject.

### Normative Assertion

`normative_assertions` is the canonical legal semantic statement. It replaces direct `semantic_relationships`.

| Field | Rule |
| --- | --- |
| `id` | Stable UUID scoped to the immutable document version and extraction result. |
| `extraction_run_id` | Identifies the reproducible extraction that produced the assertion. |
| `corpus_id`, `source_id`, `document_id` | Match the extraction run and both referenced entities and units. |
| `subject_entity_id`, `object_entity_id` | Distinct supported legal-entity endpoints. |
| `establishing_unit_id` | The exact legal unit that establishes the statement. |
| `evidence_unit_id` | The exact legal unit containing evidence; it may equal or descend from the establishing unit. |
| `predicate` | One allowed directed normative predicate. |
| `qualifier` | Optional non-empty condition, exception, or limitation. |
| `validation_status` | Only `supported` assertions are projected or retrieved. |

### Predicate vocabulary

| Predicate | Subject | Object |
| --- | --- | --- |
| `defines` | Legal term | Definition |
| `applies_to` | Norm | Covered person, activity, or situation |
| `must_be_observed_by` | Norm | Obligated public body |
| `imposes_duty_on` | Norm | Obligated actor |
| `grants` | Norm | Right or beneficiary |
| `protects` | Norm | Protected right or interest |
| `assigns_responsibility_to` | Norm | Responsible authority or actor |
| `conditions` | Legal effect or duty | Condition of application |

### Graph release

One immutable graph release belongs to one published corpus snapshot. Its memberships include the selected legal units, legal entities, and normative assertions. A ready release is the only online graph source for that snapshot.

### Hierarchy scope selection

The online graph capability catalog includes a bounded list of published legal-unit locators. A retrieval plan may include one optional `scope_locator`, selected only from that catalog. When absent, retrieval matches assertions by predicate and endpoints without a hierarchy filter. When present, it includes the selected legal unit and its descendants up to the configured depth bound.

## Derived graph topology

```text
(LegalUnit)-[:CONTAINS]->(LegalUnit)
(LegalUnit)-[:ESTABLISHES]->(NormativeAssertion {predicate, qualifier})
(NormativeAssertion)-[:SUBJECT]->(LegalEntity)
(NormativeAssertion)-[:OBJECT]->(LegalEntity)
```

Each node is namespaced by graph release, corpus, and snapshot. `NormativeAssertion` contains the source, document, and evidence locator properties required to produce a citation without asking Neo4j to become a source of text.

## State transitions

```text
normalized document
  -> validated legal units and entities
  -> supported normative assertions
  -> published snapshot
  -> ready assertion graph release
  -> snapshot activation

migration + controlled-data validation
  -> local reset permitted
  -> empty local corpus state
  -> source registration and fresh ingestion
```

No assertion or graph release crosses a corpus snapshot boundary. A failed preflight cannot enter the reset-permitted state.
