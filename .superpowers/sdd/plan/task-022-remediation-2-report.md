# Task 022 second remediation report

## Delivered repair

- Failed and cancelled `Complete` calls now discard supplied metrics and persist
  exactly the required scorer-v1 components as `not_scored`, with no numeric
  fields. Retry exhaustion uses the same canonical terminal component set.
- Completed and abstained results must provide every required scorer-v1
  component exactly once before their terminal state is persisted. This prevents
  partial terminal aggregates.
- Repository integration coverage directly completes failed and cancelled cases
  with supplied scored metrics, verifies their aggregates cannot become numeric,
  and verifies that an abstained partial result is rejected before a complete
  scorer-v1 result can terminalize.

## Aggregate guard limitation

Migration 017's aggregate trigger is a timing guard only: it permits a
run-level aggregate row only after every case is terminal. It cannot prove that
an aggregate was derived from the case-metric rows or that every component is
present. The repository derives the rows, and the complete component-set checks
plus integration coverage enforce that application contract.

## Validation

- `GOWORK=off go test ./internal/evaluation/...` - passed.
- `GOWORK=off go vet ./internal/evaluation/...` - passed.
- `GOWORK=off go test ./...` - passed.
- `GOWORK=off go vet ./...` - passed.
- `GOWORK=off go test -tags=integration ./tests/integration -run '^$' -count=0`
  - compiled successfully.
- `git diff --check` - passed.
- `python3 .github/scripts/validate_repository_language.py` - blocked by
  pre-existing language-policy violations in ignored task reports and concurrent
  `AGENTS.md` edits; this report contains only ASCII English.

`GOWORK=off go test -tags=integration ./tests/integration -run
'^TestEvaluationRunRepositoryRecoversLeasesAndPreservesTerminalResults$' -count=1`
could not run because `NORVII_POSTGRES_PORT` is not configured in this session.
