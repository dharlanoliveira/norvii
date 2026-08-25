# Snapshot HTTP and Stream Contract

## Public API

All paths are rooted at `/api/v1`. UUIDs use canonical string encoding and timestamps use UTC
RFC 3339 encoding.

### Corpus summary additions

`GET /corpora` and `GET /corpora/{corpusId}` add an optional `activeSnapshot` object:

```json
{
  "id": "50000000-0000-4000-8000-000000000001",
  "manifestSha256": "f5d1...",
  "createdAt": "2026-08-24T12:00:00Z",
  "activatedAt": "2026-08-24T12:00:00Z",
  "releaseVersion": 1
}
```

An enabled corpus without an active snapshot is unavailable for chat and has `activeSnapshot:
null` until initialization succeeds.

### Inspect snapshot history

`GET /corpora/{corpusId}/snapshots`

Returns snapshots newest first. Each immutable manifest member includes its official origin,
source revision ID, document ID, capture time, and content hash.

`GET /corpora/{corpusId}/snapshots/{snapshotId}`

Returns one historical or active immutable manifest. A snapshot from another corpus returns
`404 not_found`.

### Publish a candidate

`POST /corpora/{corpusId}/snapshots`

Request:

```json
{
  "sourceId": "20000000-0000-4000-8000-000000000001",
  "documentId": "40000000-0000-4000-8000-000000000001",
  "expectedReleaseVersion": 1
}
```

Success `201 Created` returns the newly active snapshot and its manifest. If the complete
resulting membership has the active manifest hash, return `200 OK` with the existing active
snapshot and `published: false`. No duplicate snapshot is written.

Failures use the standard problem contract:

| Status | Code | Meaning |
| --- | --- | --- |
| `400` | `invalid_input` | Malformed IDs or release version. |
| `404` | `not_found` | The corpus, snapshot, or candidate document is unavailable. |
| `409` | `stale_state` | The active release changed before publication. |
| `422` | `publication_failed` | Document, legal units, or embeddings are incomplete. |

### Chat stream additions

The public browser request stays corpus-scoped; clients do not choose a snapshot in the normal
research flow. The Go API resolves the active release and includes `snapshotId` in all
`evidence`, `completed`, and inspection evidence projections.

```json
{
  "snapshotId": "50000000-0000-4000-8000-000000000001",
  "documentId": "40000000-0000-4000-8000-000000000001",
  "unitLocator": "article-3"
}
```

If an enabled corpus has no active snapshot, chat emits the standard terminal `error` event with
the safe code `snapshot_unavailable`; it must not query a latest candidate as a fallback.

## Internal Go-to-agent contract

The internal JSON request gains a required `snapshotId`:

```json
{
  "corpusId": "10000000-0000-4000-8000-000000000001",
  "snapshotId": "50000000-0000-4000-8000-000000000001",
  "question": "Which rights are established?",
  "interfaceLanguage": "en"
}
```

The agent rejects missing or invalid snapshot IDs. Each internal evidence object includes the
same snapshot ID, which the Go API preserves unchanged in public stream projections.
