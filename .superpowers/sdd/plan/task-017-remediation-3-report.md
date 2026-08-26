# Task 017 third remediation report

## Delivered repair

- Updated the in-place active-snapshot refresh regression test to wait for the
  stale request to complete the real HTTP provider and response-parser path.
- The late response is resolved inside asynchronous `act`, then its hook
  continuation is flushed before asserting that the old question remains absent
  and the refreshed question remains visible.

## Verification

- `npm test -- --run src/features/workspace/CorpusWorkspacePage.test.tsx` - passed: 12 tests.
- `npm test` - passed: 84 tests.
- `npm run format:check` - passed.
- `npm run lint` - passed.
- `npm run typecheck` - passed.
- `npm run build` - passed, including runtime-fixture and bundle-budget checks.
- `git diff --check` - passed.
- `python3 .github/scripts/validate_repository_language.py` - remains blocked by
  59 pre-existing violations in `AGENTS.md` and prior task reports; this repair
  adds no violation.
