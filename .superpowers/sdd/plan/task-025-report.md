# Task 025 Report

## Delivered

- Added immutable evaluation dataset catalog models for revision identity, review state, source authority and binding, and starter-case metadata.
- Added PostgreSQL catalog reads that retain corpus ownership and select the latest append-only review record.
- Added maintainer-only endpoints for dataset listing, dataset inspection, and side-effect-free snapshot preflight.
- Reused the existing fail-closed bearer authorization boundary and existing bounded compatibility diagnostics.
- Wired the catalog inspection service into the API server.

## Routes

- `GET /api/v1/evaluation-datasets`
- `GET /api/v1/evaluation-datasets/{datasetRevisionId}`
- `GET /api/v1/evaluation-datasets/{datasetRevisionId}/preflight?corpusId={corpusId}&snapshotId={snapshotId}`

The detailed route is the only route that discloses starter-case metadata. All three routes require maintainer authorization. Preflight does not create a run or queue work.

## Verification

- Passed: `go test ./...` from `apps/api`.
- Passed: `go vet ./...` from `apps/api`.
- Passed: `git diff --check`.
- Integration test command was attempted: `go test -tags=integration ./tests/integration -run TestEvaluationDatasetCatalogReadsImmutableManifestBindingAndUnavailableReview -count=1`.
- Integration execution is blocked because `NORVII_POSTGRES_PORT` is not configured in this environment.
- Repository language validation was attempted and is blocked by pre-existing non-ASCII and Portuguese violations in `AGENTS.md` and earlier task reports. Task 025 files are English ASCII.
