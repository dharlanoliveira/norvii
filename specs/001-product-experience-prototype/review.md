# Prototype Review: Corpus Research Workspace

**Status**: Approved - verified prototype baseline

**Reviewed**: 2026-08-13

**Approved**: 2026-08-13

## Selected direction

The implemented direction is an editorial research desk: warm paper surfaces, high-contrast ink typography, restrained burgundy and teal accents, and a persistent evidence boundary. It deliberately presents Norvii as a legal research product rather than a generic administration dashboard.

The catalog establishes corpus isolation before interaction. The workspace keeps the source library on the left and uses one primary panel for Chat and Source, preserving context while avoiding competing reading surfaces.

## Review evidence

| Evidence | Result |
| --- | --- |
| Corpus catalog and unknown-route recovery | Passed component and browser tests |
| PDF, external-link, and unavailable-preview states | Passed component and browser tests |
| Mode, draft, source, conversation, and citation preservation | Passed component and browser tests |
| Cited answer, abstention, and deterministic failure | Passed component tests and primary browser journey |
| English default and Portuguese interface switch | Passed localization parity, component, and browser tests |
| Keyboard activation and semantic source tree | Passed component and browser tests |
| Automated accessibility scan | No detected violations; rendered contrast reviewed visually |
| 1280 by 720 and 1440 by 900 layouts | Passed deterministic visual comparison |

Reference screenshots:

- [Notebook catalog](../../prototypes/web/tests/e2e/research-workspace.spec.ts-snapshots/notebook-catalog-chromium-linux.png)
- [Notebook workspace](../../prototypes/web/tests/e2e/research-workspace.spec.ts-snapshots/notebook-workspace-chromium-linux.png)
- [Wide-desktop catalog](../../prototypes/web/tests/e2e/research-workspace.spec.ts-snapshots/wide-desktop-catalog-chromium-linux.png)
- [Wide-desktop workspace](../../prototypes/web/tests/e2e/research-workspace.spec.ts-snapshots/wide-desktop-workspace-chromium-linux.png)

## Approval decision

The stakeholder approved the prototype on 2026-08-13. The approval establishes the following decisions:

1. The editorial research-desk direction is the accepted visual identity for the initial Norvii baseline.
2. The corpus cards provide an appropriate amount of context before workspace entry.
3. The source-library density and Chat/Source relationship are accepted for production planning.
4. Feature 001 is `Verified` and becomes the required baseline for the first production-client feature.

No unresolved high-severity usability or responsive-layout finding remains at approval.

## Deferred by specification

Corpus registration, source upload, ingestion and processing states, persistence, real retrieval, GraphRAG inspection, MCP execution, and skills execution remain intentionally outside Feature 001. They require later numbered specifications and must not be inferred from this prototype.

## Technical note

The prototype intentionally favors review fidelity over initial bundle optimization. The assistant-ui dependency produces a minified JavaScript bundle above Vite's advisory 500 kB threshold. This does not block the local prototype gate; production planning must introduce route-level loading and an explicit performance budget before promotion into `apps/web/`.
