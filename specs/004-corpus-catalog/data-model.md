# Data Model: Corpus Catalog and Ingestion

PostgreSQL is canonical. UUID primary keys, UTC timestamps, lowercase checked enums, monotonic optimistic versions, and SHA-256 hashes are used throughout. All child access is corpus-scoped even when identifiers are globally unique.

## Corpus

| Field | Rules |
| --- | --- |
| `id` | UUID; stable and immutable |
| `seed_key` | Nullable stable unique key; only the two initial corpora use it |
| `name`, `description`, `jurisdiction` | Trimmed nonblank domain content |
| `language` | `en` or `pt` |
| `status` | `enabled` or `disabled` |
| `version` | Positive integer incremented on every mutation |
| `created_at`, `updated_at` | UTC timestamps |

A corpus owns at most 20 sources in this POC. Disabling is reversible and never cascades deletion.

## Source

| Field | Rules |
| --- | --- |
| `id`, `corpus_id` | UUID; required ownership FK |
| `seed_key` | Nullable stable unique key for the initial sources |
| `title` | Trimmed nonblank domain content |
| `kind` | `pdf` or `url`; immutable |
| `processing_status` | `pending`, `processing`, `ready`, or `failed` |
| `latest_failure_category` | Nullable safe enum/code |
| `latest_ready_document_id` | Nullable immutable published document FK |
| `version` | Optimistic concurrency integer |
| `created_at`, `updated_at` | UTC timestamps |

Unique `(corpus_id, normalized_url)` and `(corpus_id, pdf_sha256)` constraints are enforced through origin tables. A partial unique index permits only one active work item/attempt per source.

### State transitions

```text
pending -> processing -> ready
                      -> failed
failed  -> pending (explicit retry)
ready   -> pending (explicit reprocess; latest ready document remains)
```

## PDF Origin

One-to-one with a PDF source. Stores original filename, sanitized delivery filename, declared/detected media types, size up to 10 MB, SHA-256, and binary `bytea`. The binary is immutable.

## URL Origin

One-to-one with a URL source. Stores submitted HTTPS URL and its normalized duplicate key. Captured final URL, response media type/size, capture time, and extracted-content hash belong to immutable source revisions rather than overwriting the origin.

## Ingestion Work

| Field | Rules |
| --- | --- |
| `id`, `source_id`, `corpus_id` | UUID ownership and queue identity |
| `reason` | `initial`, `retry`, or `reprocess` |
| `status` | `pending`, `leased`, `succeeded`, or `failed` |
| `requested_at` | Deterministic queue order with `id` |
| `lease_token`, `worker_id`, `lease_expires_at` | Nullable; present only while leased |
| `created_at`, `updated_at` | UTC timestamps |

One non-terminal work row per source. Claim and source transition occur in one transaction.

## Processing Attempt

Belongs to work and source. Records attempt number, pipeline version, status, start/finish times, lease identity, safe failure category/detail, acquired byte count, normalized character count, unit count, and duration. It stores no complete source content.

## Source Revision

Immutable capture associated with one source and successful attempt. Common fields are content SHA-256, capture time, media type, byte size, pipeline version, and created time. URL revisions also record final URL and extracted-content hash; PDF revisions reference the immutable PDF origin. Unique source/content hash prevents duplicate revisions.

## Document Version

Immutable derived artifact associated one-to-one with a source revision and pipeline/text hash combination. Stores complete normalized Unicode text, SHA-256, pipeline version, publication state/time, and created time. Only `published` documents may become latest ready.

## Document Unit

| Field | Rules |
| --- | --- |
| `id`, `document_id` | UUID and owning document |
| `parent_id` | Nullable unit in same document; no cycles |
| `kind` | `document`, `title`, `chapter`, `section`, `article`, `paragraph`, `item`, `recital`, `page`, or `block` |
| `ordinal` | Unique deterministic sibling order |
| `marker`, `label` | Preserved visible identifiers when present |
| `start_offset`, `end_offset` | Half-open range inside complete text |
| `start_page`, `end_page` | Nullable positive inclusive page range |
| `locator` | Stable unique locator within document |
| `content_sha256` | Hash of the referenced text span |

Children fall within their parent span. Peer content spans do not overlap. A root document unit covers the complete text. PDF fallback units cover pages; URL fallback units cover ordered extracted blocks.

## Initial identities

The migration uses fixed UUIDs and seed keys for:

- Brazilian Portuguese data-protection corpus and official LGPD URL source;
- European Union English data-protection corpus and official English GDPR URL source.

Seed upserts insert only when the seed key is absent. They never update mutable corpus text or replace existing origin/revision data.

## Indexes and read projections

- enabled corpus order by language/name/id;
- corpus source order by created time/id;
- pending work order by requested time/id with partial index;
- source attempts by newest first;
- latest ready document lookup;
- document units by document/parent/ordinal and document/locator;
- corpus ownership included in online joins to prevent foreign-child disclosure.

No vector index or Neo4j identifier is added by this feature.
