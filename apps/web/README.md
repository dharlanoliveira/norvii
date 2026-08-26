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
- Snapshot publication, activation, evaluation source binding, and compatibility preflight are
  API-owned; the client presents their state and controls without enforcing them locally.
- Chat clearly reports that grounded answers are unavailable until Feature 005.

The [Feature 004 quickstart](../../specs/004-corpus-catalog/quickstart.md) defines the
end-to-end acceptance journey.

## Evaluation and opening suggestions

The workspace reads corpus opening suggestions through the public, read-only
`GET /api/v1/corpora/{corpusId}/opening-suggestions?interfaceLanguage=en|pt` contract.
It accepts only a response for the selected corpus, interface language, and current
active snapshot; stale, absent, or incompatible projections render no suggestions.
Selecting a suggestion sends its original question through the normal corpus-bound
chat flow. The client never starts evaluation work or receives answers, expected
evidence, scores, provider data, or prompts through this endpoint.

Maintainer evaluation routes are available at `/evaluations`,
`/evaluations/:runId`, and `/evaluations/compare`. The client presents dataset
availability and preflight state, immutable run identity, case status, safe failure
codes, expected-versus-actual provenance, metric denominators, and comparability
results. It does not bind evaluation sources, publish or activate snapshots, or run
preflight itself. Browser code uses same-origin credentials and must not receive or
embed the maintainer bearer token. The API enforces the maintainer boundary and
documents the required local token and frozen evaluation environment in
[`apps/api/README.md`](../api/README.md).
