# Norvii Production Web Client

This React and TypeScript application implements Norvii's approved corpus research experience. It presents one Portuguese and one English legal corpus, an accessible source library, prepared document and external-link viewers, and a structured simulated chat with citations and abstention.

## Prerequisites

- Node.js 24
- npm
- Chromium for Playwright browser tests

Docker, PostgreSQL, Neo4j, the Go API, Python ingestion, model credentials, and network access to corpus sources are not required.

## Start the client

From the repository root:

```bash
npm --prefix apps/web ci
npm --prefix apps/web run dev
```

Open the local address printed by Vite. The client starts in English; use the language control to switch product-authored interface text to Portuguese.

## Validate the module

Run the module CI contract:

```bash
npm --prefix apps/web exec playwright install chromium
make -C apps/web ci
```

The command verifies formatting, lint rules, strict TypeScript checks, component and accessibility tests, the production build, the compressed initial JavaScript entry budget, and Playwright browser journeys.

Run browser journeys independently when iterating on the interface:

```bash
npm --prefix apps/web exec playwright install chromium
npm --prefix apps/web run test:e2e
```

Playwright covers the approved 1280 by 720 and 1440 by 900 viewports, keyboard navigation, citation navigation, locale switching, unknown routes, and visual references.

## Current boundaries

- Corpus and source records are immutable production-owned demonstration data.
- Chat responses are deterministic and simulated; they do not call an LLM.
- Conversations and interface language are limited to the mounted browser session.
- Legal content, corpus titles, citations, and user questions are not automatically translated.
- Fixed external HTTPS links open only after explicit user activation.
- The client imports no code, fixture, style, or asset from `prototypes/web/`.

The [Feature 002 quickstart](../../specs/002-production-web-experience/quickstart.md) defines the acceptance review sequence.
