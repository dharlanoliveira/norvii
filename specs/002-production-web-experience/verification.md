# Verification: Production Corpus Research Experience

**Status**: Passed

**Verified**: 2026-08-13

## Acceptance evidence

| Area | Evidence | Result |
| --- | --- | --- |
| Catalog and corpus isolation | Component and browser journeys cover exactly two corpora, valid and direct cross-corpus navigation, state isolation, and unknown-route recovery | Passed |
| Source inspection | Component and browser journeys cover the complete roving-focus ARIA tree keyboard contract, source availability, PDF locations, external sources, and unavailable previews | Passed |
| Research chat | Component and browser journeys cover prepared answers, structured and unavailable citations, abstention, actionable retry without message duplication, and citation navigation | Passed |
| State preservation | Component and browser journeys preserve source, location, conversation, and draft across mode and locale changes | Passed |
| Localization | Locale parity tests pass; fresh sessions start in English and Portuguese changes only product-authored interface text | Passed |
| Accessibility | Axe scans cover catalog, workspace, and recovery states; keyboard journeys cover tree, mode, composer, and citation controls | No detected violations |
| Responsive presentation | Playwright snapshots pass at 1280 by 720 and 1440 by 900 without horizontal page scrolling | Passed |
| Initial interaction | Browser checks confirm the catalog becomes available in less than 2 seconds on the reference local environment | Passed |
| Bundle budget | Initial entry is 94,941 gzip bytes against the strict limit of less than 358,400 bytes | Passed |
| Service isolation | The complete quickstart uses only Node.js, npm, and Chromium; no backend, database, remote corpus request, or model is required | Passed |

## Automated verification

The following commands passed from a clean lockfile installation:

```bash
npm --prefix apps/web ci
make -C apps/web ci
python .github/scripts/validate_repository_language.py
python -m unittest discover -s .github/scripts/tests -v
git diff --check
```

The module CI contract completed with:

- 11 passing Vitest files and 18 passing tests;
- 4 passing Playwright journeys in Chromium;
- successful formatting, lint, strict TypeScript, and production build checks;
- successful compressed-entry enforcement; and
- 0 dependency vulnerabilities reported by `npm ci`.

The repository policy checks completed with 12 passing Python tests and a successful English engineering-language validation.

## Architecture review

- Production source contains no imports from `prototypes/web/`.
- Production code owns its domain models, demonstration records, styles, routes, and visual assets.
- Corpus and response behavior enter the interface through replaceable catalog and assistant adapter boundaries.
- External destinations are fixed HTTPS URLs and require explicit user activation.
- No production source uses a backend address, `fetch`, WebSocket, or event stream.
- No symbolic link connects production code to the prototype or another project-owned source tree.

## Visual evidence

- [Notebook catalog](../../apps/web/tests/e2e/research-workspace.spec.ts-snapshots/catalog-notebook-chromium-linux.png)
- [Notebook workspace](../../apps/web/tests/e2e/research-workspace.spec.ts-snapshots/workspace-notebook-chromium-linux.png)
- [Desktop catalog](../../apps/web/tests/e2e/research-workspace.spec.ts-snapshots/catalog-desktop-chromium-linux.png)
- [Desktop workspace](../../apps/web/tests/e2e/research-workspace.spec.ts-snapshots/workspace-desktop-chromium-linux.png)

The snapshots preserve the approved editorial research-desk hierarchy and show no unresolved high-severity layout difference or clipped primary action.

## Scope confirmation

Feature 002 delivers only the production client baseline. Persistence, PostgreSQL, Neo4j, the Go API, Python ingestion, live retrieval, LLM calls, GraphRAG, MCP, authentication, and registration workflows remain deferred to later numbered features.
