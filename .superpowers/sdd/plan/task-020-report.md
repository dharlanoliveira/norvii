# Task 020 report

## Delivered behavior

- Added an evaluation-specific, non-streaming Python execution contract under
  `apps/agent/src/norvii_agent/evaluation/`.
- The executor accepts an explicit corpus, immutable snapshot, question, normalized interface
  language, and frozen retrieval configuration. It calls only the supplied snapshot-scoped
  retrieval port and never resolves an active release or calls the public chat graph/service.
- Results carry ordered retrieved evidence, one-based citation-marker mapping inputs, graph
  grounding state, model and agent-build identities, and nullable duration/token telemetry.
- Cross-corpus and cross-snapshot evidence is rejected before model generation. The contract has
  no provider payload or prompt fields and does not log either.
- Generation validation rejects negative, boolean, and non-integer input/output token counts
  before they can enter result telemetry.
- Unit coverage changes the fake active release after fixed-snapshot retrieval and before model
  generation, proving that model evidence remains bound to the requested historical snapshot. It
  also verifies exact multi-evidence retrieval order and one-based citation marker mapping.

## Verification

- `make ci` in `apps/agent/` — passed: Ruff formatting and lint, strict mypy, 60 selected
  pytest tests (3 integration tests deselected), and package build.
- `git diff --check` — passed.
- `.github/scripts/validate_repository_language.py` — blocked by pre-existing non-English text
  in `AGENTS.md` and earlier ignored task reports. Task 020 source and tests use English.
