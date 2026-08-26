# Task 018 report

## Delivered behavior

- Added a browser-level, local-fixture journey for the three Feature 012 corpus selections in
  both supported interface languages.
- Every journey uses catalog navigation, checks the exact five rank-ordered synthetic questions
  returned for the active corpus and language, and submits the first question through the normal
  corpus-bound chat stream.
- Added active-snapshot drift coverage: an initial suggestion response renders for the original
  snapshot; after the workspace reloads with a new active snapshot, that stale response is hidden
  and no other corpus's questions are substituted.
- The test server intercepts only local browser requests. It does not call a database, model,
  provider, or production endpoint.

## Verification

- `npm run typecheck` — passed.
- `npm run lint` — passed.
- `npm exec prettier -- --check tests/e2e/corpus-opening-suggestions.spec.ts
../../.superpowers/sdd/plan/task-018-report.md` — passed.
- `npm run test:e2e -- corpus-opening-suggestions.spec.ts` — 7 Playwright journeys passed.
- `.github/scripts/validate_repository_language.py` — passed.
- `git diff --check -- apps/web/tests/e2e/corpus-opening-suggestions.spec.ts
.superpowers/sdd/plan/task-018-report.md` — passed.

## Self-review and concerns

- The E2E fixtures deliberately contain synthetic question text rather than legal content or
  evaluation-only data.
- The suite asserts the actual HTTP path and exact request body, so it protects the ordinary chat
  submission boundary without coupling to assistant-ui internals.
