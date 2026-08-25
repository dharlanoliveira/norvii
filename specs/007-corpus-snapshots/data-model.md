# Data Model: Bilingual Corpus Snapshots

## Existing immutable artifacts

| Entity | Ownership | Snapshot role |
| --- | --- | --- |
| `corpora` | Go API | Defines the isolated legal collection and preserved language. |
| `sources` | Go API / ingestion lifecycle | Identifies an official URL or uploaded PDF. `latest_ready_document_id` identifies a candidate, not active research evidence. |
| `source_revisions` | Ingestion | Captured immutable official content and origin metadata. |
| `document_versions` | Ingestion | Immutable normalized text for one source revision. |
| `document_units` | Ingestion | Addressable legal locations inside a document version. |
| `retrieval_chunks` | Ingestion | Immutable chunk and embedding artifacts used by online retrieval. |

## New entities

### Corpus snapshot

An immutable representation of the full evidence set for one corpus.

| Field | Type | Rules |
| --- | --- | --- |
| `id` | UUID | Stable primary identity. |
| `corpus_id` | UUID | Must own every member. |
| `manifest_sha256` | text | SHA-256 of canonical ordered membership and provenance fields; unique per corpus. |
| `created_at` | timestamptz | UTC creation time. |
| `created_by` | text | Bounded operational actor, initially `local-maintainer`. |

`corpus_snapshots` is append-only. It contains no active flag and no mutable membership.

### Corpus snapshot document

One selected source document inside a snapshot.

| Field | Type | Rules |
| --- | --- | --- |
| `snapshot_id` | UUID | References one immutable snapshot. |
| `corpus_id` | UUID | Matches the snapshot and document corpus. |
| `source_id` | UUID | Exactly one member per source within a snapshot. |
| `source_revision_id` | UUID | Must be the revision that produced `document_id`. |
| `document_id` | UUID | Must be a published document version owned by source and corpus. |
| `official_origin` | text | Normalized official URL or PDF SHA-256 captured in the manifest. |
| `captured_at` | timestamptz | Capture time copied from the source revision. |
| `content_sha256` | text | Content identity copied from the source revision. |

The composite primary key is `(snapshot_id, source_id)`. The model deliberately records the
provenance fields that are otherwise several joins away so a manifest can be inspected and
reproduced after operational projections change.

### Corpus snapshot release

The one mutable active-selection pointer per corpus.

| Field | Type | Rules |
| --- | --- | --- |
| `corpus_id` | UUID | Primary key; references a corpus. |
| `snapshot_id` | UUID | References an immutable snapshot belonging to the same corpus. |
| `version` | integer | Starts at one and increases for every successful activation. |
| `activated_at` | timestamptz | UTC activation time. |

Publication locks this row, validates the expected version, inserts the new immutable snapshot
and membership, then updates this pointer in the same transaction.

## Relationships

```text
corpus 1 -- * source 1 -- * source revision 1 -- 1 document version
                                                    |-- * document unit
                                                    `-- * retrieval chunk

corpus 1 -- * corpus snapshot 1 -- * corpus snapshot document -- 1 document version
corpus 1 -- 1 corpus snapshot release -- 1 active corpus snapshot
```

## State and invariants

1. A source revision created by ingestion is a **candidate** when its document is not selected
   by the active release.
2. Reingestion may update `sources.latest_ready_document_id`; it must not update
   `corpus_snapshot_releases`.
3. Every snapshot contains exactly one selected document for every source in the release set.
4. Every selected document must belong to the same `(corpus_id, source_id, source_revision_id)`.
5. Every selected document must have at least one legal unit and only ready embedded retrieval
   chunks; incomplete candidates fail publication.
6. The canonical membership order is source UUID ascending. The manifest hash is deterministic
   over snapshot member identity and provenance fields, so identical sets cannot be duplicated.
7. Online retrieval accepts only `snapshot_id` resolved from the active release and joins
   `corpus_snapshot_documents`; it does not use `latest_ready_document_id` as an evidence
   selector.
8. Historical snapshot membership is never overwritten. A later activation changes only the
   release pointer.
