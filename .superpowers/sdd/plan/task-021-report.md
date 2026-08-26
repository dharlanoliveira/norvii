# Task 021 report

## Delivered behavior

- Added a transport-independent evaluation result materializer in the API evaluation application.
  It validates the fixed corpus and snapshot identity plus source, source-revision, document,
  unit, locator, span, and content-hash provenance before producing detached actual-evidence
  records.
- Retrieved evidence keeps its deterministic retrieval position. Numeric answer markers are parsed
  left-to-right against that order; valid markers produce separate cited-evidence records while
  invalid markers remain visible to scoring and cannot fabricate a citation.
- Added deterministic scorer v1 components for retrieval and citation coverage, citation marker
  validity, citation scope, expected abstention, terminal execution state, nullable latency and
  token measurements, and semantic claim support. Every numeric component declares a numerator
  and denominator. Semantic support remains `needs_human_review`.
- Retrieval and citation coverage now deduplicate identical expected-evidence identities for both
  numerator and denominator, so a repeated target cannot produce an inflated denominator.
- Failed and cancelled cases produce only `not_scored` components with no value, numerator, or
  denominator. They never become fabricated zero scores.
- Added table-driven unit coverage for marker order and invalid markers, provenance and terminal
  result rejection, every score component, duplicate expected-evidence coverage, abstention
  success/failure, nullable telemetry, failed cases, and cancelled cases.

## Verification

- `gofmt -w internal/evaluation/application/scoring.go internal/evaluation/application/scoring_test.go` — passed.
- `go test ./internal/evaluation/...` — passed.
- `go vet ./internal/evaluation/...` — passed.
- `go test ./...` — passed.
- `go vet ./...` — passed.
- `git diff --check` — passed.
- `python3 .github/scripts/validate_repository_language.py` — blocked by pre-existing non-English
  content in `AGENTS.md` and Task 013, 016, 018, and 019 reports. This task's Go source, tests,
  and report use English.

## Scope

This task intentionally adds no worker, persistence adapter, transport, HTTP, web, migration, or
AGENTS change. The later execution layer can map its fixed-snapshot agent result into the pure
materialization and scoring boundary before writing immutable run records.
