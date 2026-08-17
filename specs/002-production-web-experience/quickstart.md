# Quickstart: Production Corpus Research Experience

## Prerequisites

- Node.js 24
- npm
- a current Chromium browser for end-to-end verification

No Docker service, database, Go API, Python ingestion process, remote corpus source, or model credential is required.

## Install and validate

From the repository root:

```bash
npm --prefix apps/web ci
npm --prefix apps/web exec playwright install chromium
make -C apps/web ci
```

The module CI target runs formatting verification, linting, strict type checking, component tests, locale parity and accessibility tests, the production build, the compressed initial-entry budget, and the Playwright browser journeys.

Run browser journeys independently when iterating on the interface:

```bash
npm --prefix apps/web exec playwright install chromium
npm --prefix apps/web run test:e2e
```

## Local review

```bash
npm --prefix apps/web run dev
```

Vite prints the local address after startup. No Docker Compose command is needed for this feature.

Open the printed local address and validate:

1. The catalog starts in English and identifies one Portuguese and one English corpus.
2. Each corpus opens an isolated workspace.
3. PDF and external-link sources open from the keyboard-accessible source tree.
4. Chat and Source mode changes preserve the current source, conversation, and draft.
5. A supported prepared question produces a simulated answer with a citation.
6. The citation opens the correct source location.
7. An unsupported question abstains and a prepared failure remains recoverable.
8. Portuguese interface selection changes product copy without translating legal content or clearing state.
9. Catalog, workspace, and unknown-route recovery remain usable at 1280 by 720 and 1440 by 900 pixels.

## Expected isolation

Browser network inspection shows no corpus, model, database, or backend request. Fixed official-source links are activated only when the reviewer explicitly opens them in a new browser context.
