# Norvii Web

The React and TypeScript client presents the authoritative bilingual corpus catalog
and research workspace. It consumes the Go HTTP API, manages corpus lifecycle,
submits public HTTPS pages and PDFs, polls bounded ingestion state, and renders
persisted documents and addressable units. Product-authored text comes exclusively
from the English and Portuguese localization resources.

## Run locally

Use `make bootstrap` at the repository root for the complete product. For focused web
development, first run persistence, the API, and worker, then:

```bash
npm --prefix apps/web ci
npm --prefix apps/web run dev
```

Vite proxies `/api` to `http://127.0.0.1:8080`. Open
`http://127.0.0.1:5173` unless Vite reports another configured port.

## Quality contract

```bash
npm --prefix apps/web exec playwright install chromium
make -C apps/web ci
```

The contract enforces formatting, ESLint, strict TypeScript, unit and accessibility
tests, production build, runtime-fixture exclusion, bundle budget, and Playwright
journeys. Browser tests start controlled API responses; they do not require a live
database.

## Current boundary

- PostgreSQL-backed API responses are authoritative; runtime demonstration fixtures
  are prohibited.
- English is the default interface and Portuguese has complete key parity.
- Legal content remains in its source language and is not translated.
- PDF origin bytes and captured URL origins are opened only after user activation.
- Chat clearly reports that grounded answers are unavailable until Feature 005.

The [Feature 004 quickstart](../../specs/004-corpus-catalog/quickstart.md) defines the
end-to-end acceptance journey.
