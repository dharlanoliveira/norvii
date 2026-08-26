# Task 028 second remediation report

## Delivered behavior

- Migration 018 now checks every case-level metric scorer version against the owning run scoring policy while terminalizing the run.
- Direct PostgreSQL integration coverage rejects terminal runs with incomplete, duplicate, unknown, or scorer-version-mismatched metric ledgers.
- The same coverage rejects failed and cancelled terminal cases that contain a scored metric.
- The scorer-version insert trigger is also exercised directly. The terminal-run checks are exercised with malformed historical ledger records staged before the final run transition.

## Validation

- `go test ./internal/evaluation/...` in `apps/api` passed.
- `go test -tags=integration ./tests/integration -run '^$'` in `apps/api` passed.
- `go test ./...` in `apps/api` passed.
- `go vet ./...` in `apps/api` passed.
- `git diff --check` passed.
- `make persistence-integration` applied migration 018 twice in an isolated PostgreSQL environment and executed the new direct PostgreSQL test without a failure.

## Remaining validation boundary

`make persistence-integration` did not complete because unrelated tests failed in catalog and opening-suggestion modules: `TestInitialRepositoriesAreOrderedStableAndCorpusIsolated`, `TestOpeningSuggestionRepositoryUsesOnlyTheCurrentCorpusRelease`, and `TestOpeningSuggestionRepositoryHidesProjectionAfterLatestPublicationBecomesUnavailable`.

The repository language validator also reports existing non-ASCII violations in `AGENTS.md` and older Task 013 through Task 023 report files. This remediation report is English and ASCII-only.
