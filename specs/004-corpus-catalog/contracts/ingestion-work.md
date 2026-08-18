# Contract: Ingestion Work v1

This contract governs shared PostgreSQL records between the Go online API and Python ingestion worker. Table layout may evolve through migrations, but these ownership and transaction semantics are stable for v1.

## Ownership

- Go creates corpora, sources, origins, and pending work; it performs lifecycle commands and online reads.
- Python claims work, owns attempt/lease transitions, acquires content, and publishes source revisions, documents, and units.
- Neither module updates fields owned by the other except through the transitions below.

## Claim

Within one transaction, Python selects the oldest eligible pending work by `(requested_at, id)` using `FOR UPDATE SKIP LOCKED`, creates an attempt, sets source status to `processing`, and records `worker_id`, `lease_token`, and `lease_expires_at`. A claim returns source identity, corpus identity/language, origin discriminant and metadata, reason (`initial`, `retry`, or `reprocess`), and prior ready artifact identifiers.

Only one non-terminal attempt may exist per source. The default lease is 120 seconds and is renewed before half its duration. A worker may mutate a claim only with its unexpired opaque lease token.

## Completion

Successful publication is one transaction that:

1. locks and validates the lease;
2. inserts or reuses the immutable source revision by source/content hash;
3. inserts or reuses the document by revision/pipeline/text hash;
4. validates and inserts ordered units;
5. marks the document published;
6. updates the source's latest ready document and status to `ready`;
7. marks the attempt succeeded and clears the lease.

If content, pipeline, document, and unit hashes equal the current publication, completion references the existing document and creates no duplicate active artifact.

## Failure and recovery

A categorized failure transaction marks the attempt failed, stores the safe category and bounded technical detail, sets source status to `failed`, preserves any latest ready document, and clears the lease. Categories match the HTTP error taxonomy where applicable.

Expired leases are recovered by a worker transaction that marks the abandoned attempt failed with `lease_expired`; no automatic requeue occurs. The user explicitly retries. Incomplete unpublished artifacts are never returned by online queries.

## Artifact payload

The publisher supplies SHA-256 hashes, capture metadata, pipeline version, complete normalized text, and a canonical ordered unit list. Each unit must have a stable locator, valid offsets, sibling order, parent reference within the same document, and optional page range. Publication rejects cycles, missing parents, invalid spans, inconsistent hashes, and incomplete text coverage.

## Compatibility

New nullable metadata and new failure categories are additive. Changing ownership, transition preconditions, lease meaning, hash algorithm, or artifact invariants requires a new contract version and coordinated provider/consumer migration.
