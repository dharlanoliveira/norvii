# Task 028 Report

## Delivered

- Added a read-only evaluation comparison application service with explicit comparable and non-comparable outcomes.
- Enforced the immutable comparison key: dataset revision and content hash, corpus, snapshot and manifest hash, ordered case-set hash, and scoring-policy version.
- Preserved retrieval configuration and model/build identities as visible experimental variables rather than compatibility requirements.
- Calculated each direct metric delta from summed numerator and denominator values for jointly scored dataset-case pairs only.
- Kept failed, cancelled, not-scored, and not-applicable observations out of numeric deltas while reporting paired and unpaired case counts plus failure and cancellation counts.
- Added a PostgreSQL reader that uses immutable run, run-case, and metric ledger records only; it does not resolve active snapshots or mutable catalog data.

## Verification

- `gofmt` on Task 028 Go files.
- `go test ./internal/evaluation/application ./internal/evaluation/postgres`.
- `python infra/scripts/run-with-environment.py infra/.env go test -C apps/api -tags=integration ./tests/integration -run '^TestEvaluationComparisonRepositoryReadsImmutablePairedHistoricalMetrics$' -count=1`.
- `go vet ./...`.
- `go test ./...`.
- `python .github/scripts/validate_repository_language.py`.
- `git diff --check`.

All checks passed.

## Scope and blockers

No blockers remain. The implementation is limited to Task 028 comparison application logic, PostgreSQL repository support, focused unit coverage, and PostgreSQL integration coverage.
