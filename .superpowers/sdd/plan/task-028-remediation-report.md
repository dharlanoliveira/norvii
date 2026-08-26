# Task 028 remediation report

## Delivered behavior

- Comparison rejects persisted terminal ledgers with a scored failed or cancelled case, a mismatched scorer version, incomplete metrics, duplicate metrics, or unknown metrics.
- Every loaded metric retains its scorer version and must match the immutable run scoring policy.
- Comparable results always include every required scorer-v1 metric. Metrics without jointly scored observations have the explicit `not_scored` state and nil arithmetic values.
- Migration 018 enforces scorer-version consistency and complete known terminal metric ledgers before terminal case and run transitions.

## Validation

- `go test ./internal/evaluation/application ./internal/evaluation/postgres`
- `go test -tags integration ./tests/integration -run '^TestEvaluationComparisonMetricLedgerMigrationIsEmbedded$' -count=1`
- `go test ./...`
- `go vet ./...`
- `git diff --check`
- `make persistence-integration` applied migration 018 twice in an isolated PostgreSQL environment.

## Integration result

The isolated integration journey stopped on unrelated catalog and opening-suggestion tests. The reported failures were `TestInitialRepositoriesAreOrderedStableAndCorpusIsolated` and two opening-suggestion repository tests. No evaluation comparison failure was reported before the suite stopped.
