## Delivered behavior

- Added additive PostgreSQL migration `016_evaluation_runs.sql` with immutable experiment
  identity, one ledger record per selected dataset case, frozen expected evidence, separately
  stored retrieved/cited evidence, immutable scorer metrics, explicit aggregate denominators, and
  database-enforced corpus/snapshot membership.
- Added safe PostgreSQL claim, expired-lease recovery, retry exhaustion, terminal persistence, and
  run finalization. A terminal lease token cannot overwrite a persisted terminal case result.
- Added a bounded worker application seam. Each claimed case is processed independently; a
  retryable or invalid result releases only that case, allowing unrelated cases in the same batch
  to continue.
- Added the managed `evaluation-worker` process, persistent log and PostgreSQL-readiness marker,
  local restart command, status output, and safe stop cleanup. The worker intentionally remains
  idle until Task 023 wires the explicit-snapshot agent adapter, so it cannot use the active
  snapshot or public chat stream.

## Verification

- `GOWORK=off go test ./...` in `apps/api` — passed.
- `GOWORK=off go vet ./...` in `apps/api` — passed.
- `GOWORK=off go test -tags=integration ./tests/integration -run '^$'` in `apps/api` — passed
  compilation of integration coverage.
- `GOWORK=off go test -tags=integration ./tests/integration -run '^TestEvaluationRunRepositoryRecoversLeasesAndPreservesTerminalResults$'`
  — blocked because direct execution does not receive `NORVII_POSTGRES_PORT`; the test compiled.
- `python3 -m unittest infra/scripts/test_manage_local_environment.py` — passed.
- `python3 -m py_compile infra/scripts/manage-local-environment.py infra/scripts/test_manage_local_environment.py` — passed.
- `git diff --check` — passed.
- `python3 .github/scripts/validate_repository_language.py` — blocked by pre-existing non-English
  content in `AGENTS.md` and earlier task reports; all Task 022 code, tests, migration, and this
  report are English.

## Remaining integration boundary

The local PostgreSQL containers were already running for concurrent workspace activity, so the
additive migration was not applied to that shared database during this task. The repository
integration test is ready to execute after the normal `make persistence-migrate` workflow. Task
023 should provide the explicit-snapshot execution adapter to activate the managed worker's lease
loop; this task deliberately leaves it idle rather than fabricating failures or routing through
chat.
