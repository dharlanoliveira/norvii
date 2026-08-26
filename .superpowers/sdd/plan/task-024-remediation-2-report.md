# Task 024 remediation 2 report

## Delivered

- Evaluation case claims now include the persisted retrieval configuration and complete execution identity: agent build, chat model, and embedding model.
- The worker sends the claimed identity to the Python executor and validates the returned agent, chat-model, and embedding-model identities against that claim rather than startup settings.
- The Python executor accepts only an exact runnable retrieval fingerprint and execution identity before retrieval. A configuration it cannot serve returns the bounded `frozen_identity_unavailable` result and no retrieval occurs.
- The Go worker records that unavailable frozen identity as a safe terminal case failure instead of retrying an arbitrary current configuration.
- Added unit, cross-language transport, repository, and persistence-backed HTTP coverage for frozen identities, denied routes, rejected preflight without a run, historical identity storage, and safe error boundaries.

## Verification

- Python evaluation and transport tests passed.
- Python Ruff checks and format checks passed.
- Focused Go evaluation packages passed.
- The persistence-backed HTTP integration test was added but could not run in this environment because `NORVII_POSTGRES_PORT` is not configured for the integration test harness.
- Repository language validation was run but is blocked by existing non-ASCII content in earlier SDD reports and concurrent `AGENTS.md` changes. This report is ASCII-only.

## Scope

Only evaluation execution, worker and agent contracts, HTTP persistence tests, and this report were changed. Existing unrelated worktree changes were preserved.
