# Research: Bilingual Corpus Snapshots

## Decision 1: Model a snapshot as immutable membership, not as the latest source state

**Decision**: A snapshot records the exact `document_versions` selected for every source in
one corpus. It also stores a deterministic manifest hash. Existing source revisions, document
versions, units, and retrieval chunks remain the immutable artifacts referenced by that
membership.

**Rationale**: `sources.latest_ready_document_id` is an ingestion lifecycle projection. It is
useful to identify a newly ready candidate, but selecting it directly in online retrieval makes
research change silently after reingestion. A snapshot membership is explicit, auditable, and
can be reproduced using the existing immutable document and legal-location identities.

**Alternatives considered**:

- Add a `published` flag to document versions. Rejected because it cannot represent a complete
  multi-source release and makes historical release composition ambiguous.
- Copy text, units, and vectors into every snapshot. Rejected because existing artifact IDs are
  already immutable; copying would increase storage and create divergence risk.

## Decision 2: Keep the active selection in a dedicated release pointer

**Decision**: Keep snapshot manifests immutable. Store the selected active snapshot in a
separate `corpus_snapshot_releases` row keyed by corpus with a monotonically increasing version.
The release row is updated transactionally only after a new manifest has been validated and
stored.

**Rationale**: An `is_active` field on a snapshot makes an otherwise immutable entity mutable
and complicates concurrent publication. A dedicated pointer separates immutable evidence from
mutable release selection and gives the publication operation an optimistic-concurrency value.

**Alternatives considered**:

- Add `active_snapshot_id` to `corpora`. Rejected because it introduces a circular foreign-key
  lifecycle and mixes catalog metadata with release selection.
- Make the newest snapshot active by timestamp. Rejected because it violates explicit
  publication and makes a failed or partial candidate dangerous.

## Decision 3: Publish from one explicit candidate while carrying forward the active set

**Decision**: The maintainer publication command identifies the corpus, source, candidate
document, and expected release version. It starts from the active snapshot membership, replaces
only that source's document, validates the complete resulting set, then inserts and activates a
new snapshot in one transaction.

**Rationale**: This supports the current one-source corpora and future multi-source corpora.
It prevents unrelated sources from drifting into a release merely because their latest document
happened to finish processing.

**Alternatives considered**:

- Publish every latest ready source together. Rejected because it can silently include unrelated
  candidates and weakens maintainer control.
- Publish a partial snapshot containing only the requested source. Rejected because a snapshot
  must define the complete evidence boundary for a corpus.

## Decision 4: Validate readiness from canonical artifacts

**Decision**: A candidate can be published only when it belongs to the requested corpus and
source, has a published document version, has at least one legal unit, and every retrieval chunk
for that document is `ready` with an embedding. Validation performs no model or graph work.

**Rationale**: The online vector retriever requires ready embeddings and legal locations. A
metadata-only validation prevents a broken candidate from reaching research without spending
additional tokens or credits.

**Alternatives considered**:

- Let the agent skip invalid chunks at query time. Rejected because it would publish an
  incomplete research boundary and make result quality non-deterministic.
- Re-run ingestion as part of publication. Rejected because acquisition and publication are
  separate lifecycles and a publish action must be quick and reproducible.

## Decision 5: Enforce the snapshot boundary in the agent's SQL query

**Decision**: The Go API resolves the active release before forwarding chat work. The agent
receives a required `snapshotId`, joins `corpus_snapshot_documents` in its retrieval query, and
checks snapshot and corpus identity at the query boundary. Returned evidence carries the same
snapshot ID.

**Rationale**: UI filters are insufficient for a legal evidence boundary. Enforcement in the
canonical retrieval query prevents a newer candidate, another corpus, or another language from
appearing in an answer even if a source's `latest_ready_document_id` has changed.

**Alternatives considered**:

- Filter only in Go before calling the agent. Rejected because the agent owns vector retrieval.
- Filter by source status only. Rejected because status does not identify the immutable
  document version selected for a release.

## Decision 6: Initialize snapshots explicitly and idempotently

**Decision**: Add an API-owned initialization command that creates releases only for the two
seeded corpora after their initial documents are ready. It derives the same canonical manifest
on repeated execution and returns the existing release rather than duplicating it.

**Rationale**: Database migrations create schema and seed corpus/source identity, but cannot
reliably reference asynchronous ingestion output. A separate deterministic initialization step
is observable, retryable, and keeps future reingestion from accidentally publishing a release.

**Alternatives considered**:

- Snapshot creation in the SQL migration. Rejected because documents and embeddings do not yet
  exist when migrations run.
- Auto-publish from the ingestion worker. Rejected because it breaks the explicit publication
  requirement after any reingestion.

## Decision 7: Keep release visibility compact in the existing workflow

**Decision**: Show the active snapshot identifier and publication time in the catalog and
workspace. The source view exposes a ready candidate and an explicit publish action, while a
small history/manifest view supports evaluator inspection. The standard researcher workflow
continues to use the active snapshot automatically.

**Rationale**: The POC needs to demonstrate reproducibility without turning the research UI
into release-management software. The existing verified catalog and workspace remain visually
primary.

## Decision 8: Version the internal agent contract

**Decision**: Add `snapshotId` to the Go-to-agent request and to evidence projections in the
internal agent contract. The browser-facing SSE projections mirror immutable snapshot identity.

**Rationale**: The Go API and Python agent evolve independently. A named contract change
prevents accidental fallback to corpus-only retrieval and makes evidence inspection verifiable.
