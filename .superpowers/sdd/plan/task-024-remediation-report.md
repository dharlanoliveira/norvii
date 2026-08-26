# Task 024 Remediation Report

## Delivered scope

- Connected the managed evaluation worker to the fixed-snapshot agent adapter and bounded polling loop.
- Required a run request to exactly match the server runnable retrieval configuration before preflight or persistence.
- Frozen the agent build, chat model, and embedding model identities when creating the immutable run ledger.
- Verified worker agent responses against the frozen build and chat-model identity.
- Protected every evaluation write and read endpoint with a fail-closed Bearer maintainer token boundary.
- Preserved historical snapshot selection and safe, bounded public diagnostics.

## Verification

- `go test ./internal/evaluation/agent -run TestProcessor -count=1`
- `go test ./internal/evaluation/application -run 'TestRunService|TestWorker' -count=1`
- `go test ./internal/evaluation/http -count=1`
- `go test ./internal/platform/config -count=1`
- `go test ./internal/evaluation/postgres -run 'Test(CreateEvaluationRun|Claim|Complete|Release)'`
- `go vet ./cmd/evaluation-worker ./cmd/server ./internal/evaluation/agent ./internal/evaluation/http ./internal/platform/config`
- `python3 .github/scripts/validate_repository_language.py`
- `git diff --check`

## Known verification blocker

The broad `go test ./internal/evaluation/...` command currently fails in the concurrent comparison work: `TestComparisonServiceRejectsEveryMismatchedStableKey/scoring_policy` and `TestComparisonServiceReturnsNonScoredResultsForEveryMetricWithoutJointScores`. This remediation neither changes comparison behavior nor includes those concurrent files in its commit.
