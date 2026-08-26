# Task 017 remediation report

## Delivered repair

- Replaced the active-snapshot regression test that swapped provider instances.
- The new deterministic workspace test retains one HTTP research provider, keeps the original suggestion request pending, then publishes a snapshot whose ID and manifest change.
- It verifies that publication aborts the original request, loads suggestions for the refreshed identity, and discards a late response for the prior snapshot.

## Verification

- `npm test -- --run src/features/workspace/CorpusWorkspacePage.test.tsx` - passed: 12 tests.
- `npm test` - passed: 84 tests.
- `npm run format:check` - passed.
- `npm run lint` - passed.
- `npm run typecheck` - passed.
- `npm run build` - passed, including runtime-fixture and bundle-budget checks. Vite emitted its existing chunk-size advisory.
- `git diff --check` - passed.
- `python3 .github/scripts/validate_repository_language.py` - blocked by 53 pre-existing violations in `AGENTS.md` and earlier Task 013, 016, 018, and 019 reports; this repair adds no violation.
