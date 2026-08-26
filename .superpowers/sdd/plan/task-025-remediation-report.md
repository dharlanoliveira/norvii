# Task 025 Remediation Report

## Delivered

- Published the maintainer dataset catalog, detail, and side-effect-free preflight routes in the durable Evaluation Contract v1.
- Added language-neutral catalog, detail, successful preflight, and bounded incompatible-preflight fixtures.
- Added contract validation for immutable revision identity, review availability, source binding, starter metadata, preflight identity, and safe bounded diagnostics.
- Expanded HTTP boundary coverage for authorization and non-disclosure on all three routes, invalid route and query identifiers, not found detail requests, catalog-only projection, and preflight rejection without a run start.
- Added PostgreSQL-to-HTTP integration coverage for persisted starter-case metadata disclosure, catalog projection, successful preflight, rejected preflight diagnostics, and zero created runs.

## Validation

- Passed: `go test ./...` from `apps/api`.
- Passed: `go vet ./...` from `apps/api`.
- Passed: `go test ./internal/evaluation/http ./tests/contract` from `apps/api`.
- Passed: `git diff --check`.
- Attempted: `go test -tags=integration ./tests/integration -run 'TestEvaluation(DatasetInspectionHTTPPreservesPersistedStarterMetadataAndPreflightSafety|DatasetCatalogReadsImmutableManifestBindingAndUnavailableReview)$' -count=1` from `apps/api`.
- Blocked integration execution: `NORVII_POSTGRES_PORT` is not configured in this environment.
- Attempted repository language validation with `python3 .github/scripts/validate_repository_language.py`.
- Blocked language validation: existing non-ASCII and Portuguese violations are reported in `AGENTS.md` and earlier Task 013, 016, and 018 through 023 reports. This remediation report and its scoped files are English ASCII.

## Scope

The remediation changes only Task 025 contracts, fixtures, tests, and this report. `AGENTS.md`, web code, datasets, and task checklists were not modified.
