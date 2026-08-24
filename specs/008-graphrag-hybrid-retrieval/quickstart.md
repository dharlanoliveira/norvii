# Quickstart: GraphRAG and Hybrid Retrieval

## Prerequisites

- Complete Feature 007 initialization with two ready active corpus snapshots.
- Configure `NORVII_SEMANTIC_*` settings for bounded semantic extraction.
- Start Norvii with `make bootstrap`.

## Build graph releases explicitly

1. Reprocess each source that must participate in the graph. This explicit ingestion step runs
   bounded semantic extraction and may call the configured provider.
2. Publish the ready candidate snapshot in the workspace and copy its corpus and snapshot IDs
   from Snapshot history.
3. Build the graph release from the repository root:

   ```bash
   python infra/scripts/run-with-environment.py infra/.env \
     uv run --directory apps/ingestion norvii-build-graph-release \
     --corpus-id <corpus-id> --snapshot-id <snapshot-id>
   ```

4. Inspect readiness:

   ```bash
   curl http://127.0.0.1:8080/api/v1/corpora/<corpus-id>/snapshots/<snapshot-id>/graph-release
   ```

5. Repeat the build command. It must report the same ready release identity and not create a
   duplicate projection.

The graph-build command is explicit and reads only persisted canonical artifacts; it does not call
the semantic provider. Neither semantic extraction nor graph construction runs as a side effect of
application startup, source browsing, or a chat request.

## Verify strategy isolation

1. Open one corpus and select `graph` or `hybrid` retrieval.
2. Ask a seeded connected legal question.
3. Inspect the result. Confirm the strategy, graph-release identity, path, and every cited
   location belong to that corpus's active snapshot.
4. Open each path citation and confirm the exact preserved source location opens.
5. Repeat with the other corpus. No entity, relationship, path, or citation may cross corpora.

## Verify candidate safety

1. Reingest one official source without publishing its candidate.
2. Run graph and hybrid retrieval for the current active snapshot.
3. Confirm the candidate's entities, relationships, and evidence remain absent.
4. Publish the candidate snapshot, build its graph release explicitly, and confirm the preceding
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

The graph-release build and end-to-end cross-store isolation journey require semantic artifacts
for a freshly reprocessed and explicitly published snapshot. Do not trigger that reingestion as a
verification side effect when it would incur provider cost; run it deliberately as the first part
of this quickstart.
