# Task 018 remediation report

## Delivered behavior

- Replaced the invented E2E corpus UUIDs with the three Feature 012 seed corpus UUIDs for LGPD, Brazilian anti-corruption, and United States fair-housing disability accommodations.
- Replaced every synthetic opening question with the exact five rank-ordered curated PT and EN starter cases from each project-owned evaluation dataset. Each local response now declares its expected case ID, rank, and question explicitly.
- Kept local request interception, normal corpus-bound chat submission assertions, and the stale active-snapshot response scenario. The E2E suite still has no database, model, provider, or production-network dependency.

## Verification

- `npm run typecheck`: passed.
- `npm run lint`: passed.
- `npm exec prettier -- --check tests/e2e/corpus-opening-suggestions.spec.ts`: passed.
- `npm run test:e2e -- corpus-opening-suggestions.spec.ts`: 7 Playwright journeys passed.
- A read-only Node validation compared the three seed UUIDs and all 30 curated case ID/question pairs in the E2E fixture with the project-owned JSONL datasets: passed.
- `python3 .github/scripts/validate_repository_language.py`: blocked by 53 pre-existing violations (28 in `AGENTS.md`, 5 in the Task 013 report, 7 in the Task 016 report, 6 in the original Task 018 report, and 7 in the Task 019 report); this remediation report adds no violation.
- `git diff --check -- apps/web/tests/e2e/corpus-opening-suggestions.spec.ts .superpowers/sdd/plan/task-018-remediation-report.md`: passed.

## Self-review

- The fixture remains intentionally local and stale-snapshot deterministic. Snapshot UUIDs are test-only identities because Feature 012 creates active snapshots after source ingestion; only the corpus UUIDs are stable Feature 012 seeds.
- No task tracking was marked complete.
