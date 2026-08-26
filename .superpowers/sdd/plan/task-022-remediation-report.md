# Task 022 remediation report

## Delivered behavior

- Added reversible migration `017_evaluation_terminal_result_ledger.sql`. PostgreSQL now rejects
  expected-evidence, actual-evidence, and case-metric inserts after the parent case is terminal.
  Run-level metric inserts require every run case to be terminal.
- Reordered case completion so the worker locks its active lease, persists actual evidence and
  case metrics, and only then writes the terminal case state in the same transaction.
- Terminal retry exhaustion writes explicit `not_scored` component metrics before marking the
  case failed. These rows have no numeric value, numerator, or denominator, preventing synthetic
  failure or cancellation zeros.
- Materialized immutable run-level component metrics from terminal case metric rows. Each read
  returns the component state, value, numerator, denominator, rationale, and scorer version;
  telemetry numerators are totals and denominators are reported-measurement counts.
- Added integration coverage for the valid atomic completion path, PostgreSQL rejection of all
  later child insert types, metric-specific aggregate values, and failed-case unscored aggregates.

## Verification

- `GOWORK=off go test ./internal/evaluation/...` in `apps/api` — passed.
- `GOWORK=off go test -tags=integration ./tests/integration -run 'TestEvaluation(TerminalResultLedgerMigrationIsEmbedded|RunLedgerMigrationIsEmbedded)$'` in `apps/api` — passed.
- `GOWORK=off go vet ./...` in `apps/api` — passed.
- `GOWORK=off go test ./...` in `apps/api` — passed.
- `GOWORK=off go test -tags=integration ./tests/integration -run '^TestEvaluationRunRepositoryRecoversLeasesAndPreservesTerminalResults$' -count=1` in `apps/api` — blocked because `NORVII_POSTGRES_PORT` is not set. The integration package and the migration tests compile successfully with the integration tag.
- `git diff --check` — passed.

## Remaining integration boundary

The required local PostgreSQL connection variables are unavailable in this session, so the
database-backed repository test could not be executed against a migrated PostgreSQL instance.
Run the normal local persistence startup and migration workflow, then rerun the targeted
integration test above to exercise the terminal-child triggers and aggregate persistence.
