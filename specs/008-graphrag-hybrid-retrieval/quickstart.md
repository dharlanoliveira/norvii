# Quickstart: GraphRAG and Hybrid Retrieval

## Prerequisites

- Complete Feature 007 initialization with two ready active corpus snapshots.
- Configure `NORVII_SEMANTIC_*` settings for bounded semantic extraction.
- Start Norvii with `make bootstrap`.

## Reingest a graph-ready corpus

1. Reprocess a source from the workspace. The ingestion pipeline runs bounded semantic
   extraction, stages an immutable snapshot, builds its derived graph release, and activates the
   snapshot only after that release is ready.
2. Inspect the active snapshot and its graph-release readiness:

   ```bash
   curl http://127.0.0.1:8080/api/v1/corpora/<corpus-id>/snapshots/<snapshot-id>/graph-release
   ```

3. Confirm that Vector and planned Hybrid can answer against the new active snapshot without a
   separate publication or build action.

The optional graph-build command reads only persisted canonical artifacts and exists to reproduce
an already recorded release. Neither semantic extraction nor graph construction runs as a side
effect of application startup, source browsing, or a chat request.

## Verify strategy isolation

1. Open one corpus and select `vector` or `hybrid` retrieval.
2. Ask a seeded connected legal question.
3. Inspect the result. Confirm the strategy, graph-release identity, path, and every cited
   location belong to that corpus's active snapshot.
4. Open each path citation and confirm the exact preserved source location opens.
5. Repeat with the other corpus. No entity, relationship, path, or citation may cross corpora.

## Verify candidate safety

1. Reingest one official source and observe its release-stage state.
2. While the candidate is staging or graph validation is in progress, run Vector and Hybrid
   against the active snapshot.
3. Confirm the candidate's entities, relationships, and evidence remain absent.
4. Confirm that activation occurs only after the new graph release is ready, while the preceding
   graph release remains inspectable for its preceding snapshot.

## Verify failure behavior

Request Hybrid retrieval for an active snapshot without a ready release. Norvii must return the
safe graph-unavailable stage, retain the selected strategy in inspection, and not silently show a
vector result.

## Validation record

Run the module and contract checks from the repository root:

```bash
make -C apps/web test
make -C apps/web build
make -C apps/api ci
make -C apps/agent ci
make -C apps/ingestion ci
python .github/scripts/validate_contracts.py
python .github/scripts/validate_repository_language.py
```

The graph-release and end-to-end cross-store isolation journey require semantic artifacts for a
freshly reprocessed source. Do not trigger reingestion as a verification side effect when it would
incur provider cost; run it deliberately as the first part of this quickstart.

## Completion Evidence

- 2026-08-25: Hybrid retrieval completed against the configured provider for both seeded corpora.
- 2026-08-25: Rebuilding snapshot `0ff03bec-ea38-4636-87ec-639770a6290c` three times reused graph release `a6ecdff6-41fb-586f-bc1d-432a5a7af9a0` each time, with 101 entities and 48 relationships.
- 2026-08-25: A deliberate configured-provider reingestion completed for the Brazilian corpus in
  148,602 ms. Its active snapshot `876346d4-ed46-4067-9422-53bc46a25e86` has ready graph
  release `90149ff5-3513-55f3-ab56-b2d7658f35b2` with 76 entities and 41 relationships.
- 2026-08-25: A deliberate configured-provider retry completed for the European Union corpus in
  142,641 ms after semantic-artifact backfill. Its active snapshot
  `0ff03bec-ea38-4636-87ec-639770a6290c` has ready graph release
  `96c8b85a-24ae-519c-8971-0eb4cfcc2667` with 145 entities and 85 relationships.
