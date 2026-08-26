# Task 017 report

## Delivered behavior

- Removed obsolete legal starter-question content from both localization resources while retaining the non-content suggestion-region label.
- Extended `ResearchChat` coverage to prove provider-supplied starter questions use the normal runtime submission path with the selected interface language, and that an empty input never creates a local fallback region.
- Added deterministic workspace integration coverage through the HTTP research-provider seam. It verifies received rank order and selected locale, failure without replacement content, and cancellation plus late-response discard on locale, corpus-route, and active-snapshot identity changes.

## Verification

- `npm test -- --run src/features/workspace/ResearchChat.test.tsx src/features/workspace/CorpusWorkspacePage.test.tsx` - passed: 28 tests.
- `npm run format:check` - passed.
- `npm run lint` - passed.
- `npm run typecheck` - passed.
- `npm run build` - passed, including runtime-fixture and bundle-budget checks. Vite emitted its existing informational chunk-size advisory.
- `npm test` - passed: 84 tests.
- `git diff --check` - passed for task files.
- `.github/scripts/validate_repository_language.py` was run from the repository root. It remains blocked only by pre-existing `AGENTS.md` and Task 013/016 report content; this task adds no reported violation.

## Self-review and concerns

- No production workspace code was changed because the existing identity guards and effect cleanup satisfy the required behavior.
- The active-snapshot lifecycle test replaces the real provider instance to cause a refreshed corpus response; this exercises the public component/provider seam without exposing internal hooks for testing.
