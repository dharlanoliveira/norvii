# Task 023 remediation report

## Delivered repair

- Added the Python `POST /v1/evaluations/execute` transport consumed by the Go evaluation adapter. It is a dedicated strict JSON handler, never routes through the public chat/SSE handler, rejects malformed or unknown request fields, and returns only bounded JSON errors.
- Wired the endpoint to `EvaluationExecutor` with the caller-supplied corpus ID, snapshot ID, normalized language, and frozen retrieval configuration. The provider adapter requests a terminal non-streaming completion.
- Extended vector and graph retrieval provenance with legal-unit ID, canonical legal locator, and evidence content SHA-256. Graph projection now carries that immutable metadata and advances to graph projection build version `legal-assertion-graph-v2`, forcing rebuilds rather than reusing projections that lack the fields.
- Added Python transport tests and a Go contract test that launches the Python transport fixture. The cross-language test verifies request serialization, fixed snapshot scope, response JSON content type, ordered citation mapping, legal-unit provenance, evidence content hash, and materialized citations.

## Verification

- `make ci` in `apps/agent/` — passed: Ruff formatting/lint, mypy, 62 selected pytest tests, and package build.
- `make ci` in `apps/ingestion/` — passed: lock validation, Ruff formatting/lint, mypy, 94 selected pytest tests, and package build.
- `go test ./...` and `go vet ./...` in `apps/api/` — passed, including the Go-to-Python evaluation transport contract test.
- `git diff --check` — passed.

## Limitation

Existing Neo4j graph releases must be rebuilt with `legal-assertion-graph-v2` before hybrid evaluation can return the new complete provenance. Until rebuilt, the strict evaluation endpoint rejects incomplete graph evidence instead of emitting unverifiable provenance.
