# Task 023 report

## Delivered behavior

- Added a dedicated Go evaluation-to-agent adapter under
  `apps/api/internal/evaluation/agent/`. It posts an explicit corpus ID, immutable snapshot ID,
  question, normalized interface language, and frozen retrieval configuration to the Python
  evaluation endpoint.
- The adapter accepts only terminal JSON, rejects streaming responses and bounded unsafe failure
  bodies, and does not import or call the public chat application service.
- Before returning data to a worker or scorer, it validates response model/build identities,
  graph-grounding state, ordered evidence rank, one-based citation marker mapping, complete
  evidence provenance, fixed corpus/snapshot ownership, and nullable non-negative telemetry. It
  then materializes the result through the existing deterministic scoring boundary.
- Added contract tests for exact fixed-snapshot transport, cross-boundary and malformed response
  rejection, non-streaming safe failure behavior, and the absence of a public-chat application
  dependency.

## Verification

- `gofmt -w internal/evaluation/agent/client.go internal/evaluation/agent/client_test.go` — passed.
- `go test ./...` in `apps/api/` — passed.
- `go vet ./...` in `apps/api/` — passed.
- `git diff --check` — passed.
- `python3 ../../.github/scripts/validate_repository_language.py` from `apps/api/` — passed.
- `make ci` in `apps/agent/` — passed: Ruff, mypy, 60 selected pytest tests, and package build.

The same language validator at the repository root remains blocked by 69 pre-existing violations
in `AGENTS.md` and earlier task reports; Task 023 files pass the scoped validation.

## Scope

This task adds no worker persistence, HTTP endpoint, web behavior, migration, or `AGENTS.md`
change. Existing concurrent worker and migration changes remain outside this task.
