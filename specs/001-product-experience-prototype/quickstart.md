# Quickstart: Corpus Research Workspace Prototype

## Prerequisites

- Node.js 24
- npm distributed with Node.js 24
- A current Chromium-based browser, Firefox, or WebKit browser

## Install and run

From the repository root:

```bash
npm --prefix prototypes/web ci
npm --prefix prototypes/web run dev
```

Open the local URL printed by Vite. The prototype must not require environment variables, credentials, backend services, Docker, or network access after dependency installation.

## Validate locally

Run the complete module contract:

```bash
make -C prototypes/web ci
```

Run browser journeys after installing the pinned browser once:

```bash
npm --prefix prototypes/web exec playwright install chromium
npm --prefix prototypes/web run test:e2e
```

## Required review journeys

### 1. Select each corpus

1. Open a fresh session at `/`.
2. Confirm the interface starts in English.
3. Identify the Portuguese and English corpora by language and jurisdiction.
4. Open each corpus and verify the workspace contains only its sources.

### 2. Browse sources

1. Expand and collapse PDF and external-link groups using pointer and keyboard.
2. Open one PDF and one external link.
3. Confirm selection, metadata, location, viewer state, and unavailable-preview recovery behavior.

### 3. Preserve source and chat context

1. Open a source and record its location.
2. Switch to Chat, enter a draft, and switch back to Source.
3. Confirm the source and location remain active.
4. Submit a prepared question, open its citation, and confirm the viewer targets the cited location.
5. Return to Chat and confirm the conversation remains intact.
6. Submit `Simulate a failed response` and confirm the recoverable failure preserves the corpus context.

### 4. Review localization and layout

1. Change the interface to Portuguese in the active workspace.
2. Confirm controls change language while legal content and interaction state remain unchanged.
3. Complete the journeys with keyboard only.
4. Review at 1280 by 720 and 1440 by 900 with reduced motion enabled and disabled.

## Expected result

The catalog and workspace are coherent and polished, all required interactions are accessible, state survives mode and language changes, citations connect chat to the correct source, unsupported questions abstain, and no backend or ingestion capability is invoked.

## Recorded verification

Verification completed on 2026-08-13 with Node.js 24:

- formatting, ESLint, strict TypeScript, 14 Vitest tests, and the production build passed;
- five Playwright journeys passed in Chromium, including keyboard-only activation and language-state preservation;
- the catalog accessibility scan reported no automatically detectable violations, with color contrast reviewed visually because JSDOM cannot compute rendered contrast;
- the repository language validator passed with Portuguese limited to localization and legal fixtures;
- deterministic visual baselines were captured for both reference viewports.

Visual evidence:

- [Notebook catalog baseline](../../prototypes/web/tests/e2e/research-workspace.spec.ts-snapshots/notebook-catalog-chromium-linux.png)
- [Notebook workspace baseline](../../prototypes/web/tests/e2e/research-workspace.spec.ts-snapshots/notebook-workspace-chromium-linux.png)
- [Wide-desktop catalog baseline](../../prototypes/web/tests/e2e/research-workspace.spec.ts-snapshots/wide-desktop-catalog-chromium-linux.png)
- [Wide-desktop workspace baseline](../../prototypes/web/tests/e2e/research-workspace.spec.ts-snapshots/wide-desktop-workspace-chromium-linux.png)
