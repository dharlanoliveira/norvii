# Data Model: Planned Hybrid Retrieval

## Retrieval Approach

The new-request enum contains `vector` and `hybrid` only. It is immutable for a submitted
question and is retained in the terminal inspection. Legacy Graph inspection values are displayed
unchanged only when they already exist in the current browser session.

| Field | Rules |
| --- | --- |
| `strategy` | `vector` or `hybrid` for new requests. |
| `corpus_id` | Required selected corpus identifier. |
| `snapshot_id` | Required active immutable snapshot resolved by the API. |

## Graph Capability Catalog

A transient, active-snapshot-scoped read model used by the planner. It is not persisted as a new
authority and contains no complete document content.

| Field | Rules |
| --- | --- |
| `snapshot_id` | Must equal the current active snapshot. |
| `release_id` | Must identify one ready graph release. |
| `entity_types` | Allowlisted graph entity types, bounded to 32 descriptors total. |
| `relationship_types` | Allowlisted relationship types that a graph query may request. |
| `entity_labels` | Published canonical entity labels, bounded to 128 values and used only for validated graph filters. |
| `relationship_capabilities` | Published relationship-type to canonical-entity-label combinations. A plan may use only a combination in this catalog. |

## Hybrid Retrieval Plan

An ephemeral, validated planner result. It does not contain query text.

| Field | Rules |
| --- | --- |
| `state` | `used`, `skipped`, `unavailable`, `no_evidence`, or `failed`. |
| `reason_code` | Safe bounded reason; required outside `used`. |
| `relationship_types` | Subset of the capability catalog with at least one selected compatible entity label; empty when skipped. |
| `entity_labels` | Canonical labels selected only from a matching published relationship capability; no more than 8. They may differ from the question language. |
| `planning_milliseconds` | Non-negative duration. |
| `input_tokens` / `output_tokens` | Provider-reported counts or null. |

## Retrieval Stage Contribution

An inspectable stage result for one answer.

| Field | Rules |
| --- | --- |
| `stage` | `vector`, `planning`, `graph`, or `generation`. |
| `state` | `completed`, `skipped`, `unavailable`, `no_evidence`, or `failed`. |
| `reason_code` | Safe bounded diagnostic or null. |
| `evidence_count` | Non-negative count; zero for planning and skipped stages. |
| `elapsed_milliseconds` | Non-negative duration or null when not started. |
| `input_tokens` / `output_tokens` | Provider-reported counts or null. |

## Evidence Contribution

Every answer evidence location retains the existing immutable location fields plus a contribution
value:

| Value | Meaning |
| --- | --- |
| `vector` | Returned only by vector retrieval. |
| `graph` | Returned only by a graph path. |
| `both` | The same immutable location was returned by both stages. |

Deduplication identity is document ID, unit locator, start offset, and end offset. The earliest
retrieval rank becomes the displayed rank.

## State Transitions

```text
request
  -> vector completed | vector no_evidence | vector failed
  -> (Hybrid only) capability available -> planning completed -> graph completed | graph no_evidence
  -> (Hybrid only) capability unavailable | planning skipped | planning failed
  -> generation completed | grounded abstention
```

No graph state changes the active graph release or corpus snapshot. A generation request receives
only the deduplicated evidence emitted by completed retrieval stages.
