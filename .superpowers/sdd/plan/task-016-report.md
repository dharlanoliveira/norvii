# Task 016 report

## Delivered behavior

- `CorpusWorkspacePage` now owns the opening-suggestion request and passes only verified,
  rank-ordered contract items to `ResearchChat`.
- A request starts only when the selected corpus has an active snapshot. Its identity includes the
  corpus ID, interface language, snapshot ID, manifest hash, and release version.
- Requests are aborted when the corpus route, interface language, active snapshot, manifest hash,
  release version, or provider changes. The UI returns an empty list unless the response identity
  matches the current workspace identity, so old requests cannot render after a change.
- `ResearchChat` renders the exact provided question text as accessible native buttons. Empty,
  missing, stale, or mismatched sets render no suggestion region. Selecting a button still appends
  the exact question to the existing assistant-ui thread and therefore uses the normal
  `useNorviiChatRuntime` submission path.
- Removed the static LGPD-specific questions from the production chat component. Interface text
  remains localized; question content is supplied by the versioned contract.

## Verification

- `npm run format:check` — passed.
- `npm run lint` — passed.
- `npm run typecheck` — passed.
- `npm test -- --run src/features/workspace/ResearchChat.test.tsx src/features/workspace/CorpusWorkspacePage.test.tsx` — 23 tests passed.
- `npm run build` — passed, including the runtime-fixture and bundle-budget checks. Vite emitted
  its existing informational chunk-size advisory.
- `npm test` — 79 tests passed.
- `git diff --check` — passed for task files.
- `.github/scripts/validate_repository_language.py` was run from the repository root. It remains
  blocked by the user-owned Portuguese `AGENTS.md` changes and concurrent Task 013 report content;
  this task adds only English technical text.

## Self-review and concerns

- Reviewed all accepted response fields against the public opening-suggestions contract and the
  Task 015 provider seam. The release version is part of the request lifecycle identity even though
  the read response exposes snapshot ID and manifest hash as its comparable identity.
- The component does not sort, translate, supplement, or otherwise transform suggestion text; the
  strict provider parser guarantees the received order and maximum size.
- Task 017 remains responsible for removing obsolete localization entries and adding broader
  route-change, locale-change, and no-set regression coverage.
