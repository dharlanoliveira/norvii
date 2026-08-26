## Delivered route behavior

Added `GET /api/v1/corpora/{corpusId}/opening-suggestions?interfaceLanguage=en|pt`.
The handler accepts exactly one supported interface-language query value and a non-zero corpus UUID.
It returns the current active snapshot identity and a rank-ordered, at-most-five public question list.
Disabled, missing, unpublished, and stale projection states remain normal `200` responses with an
empty `suggestions` array, as returned by the projection repository.

The public DTO contains only corpus and active-snapshot identity, interface language, case ID,
rank, and question. It has no evaluation answer, evidence, dataset, score, configuration,
provider, or prompt fields. Malformed requests return `400 invalid_input`; repository failures
return a safe `503 unavailable` envelope without their cause.

## Dependency wiring

The API server now constructs the Task 012 PostgreSQL opening-suggestion repository from the
shared PostgreSQL pool and registers its read-only HTTP handler on the existing application mux.
The route does not call evaluation preflight, scoring, agents, writes, or the chat stream.

## Tests and results

- Handler tests compare English, Portuguese, no-active-snapshot, and stale responses with the
  shared contract fixtures; cover malformed UUID/language input, reader failures, rank ordering,
  and evaluation-field absence.
- Contract tests strictly accept valid shared fixtures and reject fixtures with unknown
  evaluation-only fields or out-of-order ranks.
- `go test ./internal/suggestions/http ./tests/contract` — passed.
- `go vet ./internal/suggestions/http ./tests/contract` — passed.
- `go test ./cmd/server` — passed (the command has no test files).
- `git diff --check` — passed.
- `python3 .github/scripts/validate_repository_language.py` — blocked by 28 pre-existing
  violations in the user-owned Portuguese multi-agent section of `AGENTS.md`; this task did not
  modify that file.

## Self-review and concerns

The handler is intentionally a thin read boundary over the Task 012 repository, which owns
active-release, disabled/missing, publication, and stale-projection semantics. No schema,
repository, chat, evaluation, web, prototype, or `AGENTS.md` files were changed. There are no
known implementation concerns beyond the normal requirement that the deployment has applied the
existing projection migration.
