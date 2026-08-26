# Task 019 report

## Delivered behavior

- Added append-only evaluation dataset review publication with bounded review notes and validated
  lifecycle transitions. PostgreSQL locks the dataset revision while it reads the latest review
  state and appends the new immutable record, so concurrent review attempts cannot overwrite or
  bypass a valid transition.
- Added an application-owned maintainer source-binding boundary that validates exact corpus,
  revision, source alias, and source UUID before the existing one-time PostgreSQL binding API.
- Added an all-or-nothing fixed-snapshot compatibility preflight. It requires the dataset's exact
  corpus, its latest approved and available publication, every manifest source binding and snapshot
  membership, and every exact canonical locator resolution. It returns only a complete detached
  resolution plan on success; otherwise it returns a bounded safe requirement list.
- Added PostgreSQL catalog reads for publication state, dataset ownership, source requirements,
  expected evidence, named-snapshot membership, and the existing immutable locator resolver.
  No active snapshot, title, URL, text, or structural-locator fallback is used.

## Verification

- `go test ./internal/evaluation/...` — passed.
- `go test ./...` — passed.
- `go vet ./internal/evaluation/...` — passed.
- `go vet ./...` — passed.
- `go test -tags=integration ./tests/integration -run 'TestEvaluationPreflight'` — compiled and
  reached the integration setup, but could not run because `NORVII_POSTGRES_PORT` is not defined
  in this environment.
- `git diff --check` — passed.
- `.github/scripts/validate_repository_language.py` — blocked by pre-existing non-English text in
  `AGENTS.md` and prior task reports; the Task 019 source and synthetic fixture text use English.

## Self-review and concerns

- The preflight service has no agent, model, run-writer, queue, or HTTP dependency. Unit and
  PostgreSQL integration tests include a model-call sentinel that remains at zero for every
  rejected gate.
- The current imported asset schema stores display locators, not a separate reviewed canonical
  locator mapping. The repository therefore accepts only display locators that are already exact
  canonical aliases; all other locators fail closed as unresolved. A future reviewed mapping must
  be persisted before the existing human-readable legal locators can form successful run plans.
- The task intentionally does not create evaluation runs, start workers, invoke models, or alter
  active snapshots.

## Remediation

- Added a reviewed `canonical_locator` to all 60 expected-evidence selectors across the three
  project-owned JSONL assets (52 cases). The original `locator` remains the human display value.
- Added migration `015_evaluation_expected_evidence_canonical_locators.sql`, with a reversible
  column drop. Older immutable revisions remain readable but fail closed during preflight until
  they carry a canonical locator; no display-locator fallback is introduced.
- Asset validation, domain validation, PostgreSQL import, catalog reads, and preflight now carry
  the exact canonical locator. Human display strings such as `Article 1` are rejected as a
  canonical value, while the successful resolver path retains that display string alongside
  `article:1`.
- Preflight now accumulates bounded source/snapshot and locator failures in one diagnostic list.
  The error remains `ErrSnapshotIncompatible` when a required source is missing, preserving the
  more fundamental compatibility cause.

## Remediation verification

- `GOWORK=off go test ./...` — passed.
- `GOWORK=off go vet ./...` — passed.
- `GOWORK=off go test -tags=integration ./tests/integration -run '^$'` — passed compilation.
- JSONL validation confirmed 52 cases and 60 canonical selectors matching the canonical-locator
  grammar.
- `git diff --check` — passed.
- `make persistence-integration` — blocked before tests: local host port `17687` was already in
  use while starting the isolated Neo4j container. The integration runner cleaned up after the
  failed start.
- `.github/scripts/validate_repository_language.py` — blocked by pre-existing non-ASCII text in
  `AGENTS.md` and prior Task 013, 016, 018, and 019 reports. The changed Go, SQL, and tests use
  English; corpus JSONL remains an approved legal-content exception.
