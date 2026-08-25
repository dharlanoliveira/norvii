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

3. Confirm that Vector, Graph, and Hybrid can answer against the new active snapshot without a
   separate publication or build action.

The optional graph-build command reads only persisted canonical artifacts and exists to reproduce
an already recorded release. Neither semantic extraction nor graph construction runs as a side
effect of application startup, source browsing, or a chat request.

## Verify strategy isolation

1. Open one corpus and select `graph` or `hybrid` retrieval.
2. Ask a seeded connected legal question.
3. Inspect the result. Confirm the strategy, graph-release identity, path, and every cited
   location belong to that corpus's active snapshot.
4. Open each path citation and confirm the exact preserved source location opens.
5. Repeat with the other corpus. No entity, relationship, path, or citation may cross corpora.

## Verify candidate safety

1. Reingest one official source and observe its release-stage state.
2. While the candidate is staging or graph validation is in progress, run all strategies against
   the active snapshot.
3. Confirm the candidate's entities, relationships, and evidence remain absent.
4. Confirm that activation occurs only after the new graph release is ready, while the preceding
   graph release remains inspectable for its preceding snapshot.

## Verify failure behavior

Request graph or hybrid retrieval for an active snapshot without a ready release. Norvii must
return the safe graph-unavailable state, retain the selected strategy in inspection, and not
silently show a vector result.

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
