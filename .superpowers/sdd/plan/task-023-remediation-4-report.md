# Task 023 fourth remediation report

## Delivered repair

- Updated all three `_GRAPH_CAPABILITIES` subqueries to Neo4j's current
  `CALL (release) { ... }` variable-import syntax while preserving their
  snapshot-scoped capability projections.
- Added a focused query-contract assertion covering all three subqueries and
  preventing the deprecated `CALL { WITH release ... }` form from returning.
- Removed the redundant `Driver` cast from the service-backed replacement
  integration test; the Neo4j factory already returns the required type.

## Verification

- Agent graph retrieval tests: passed (7 tests).
- Ingestion Neo4j persistence unit tests: passed (2 tests).
- Agent configured `make type-check`: passed (22 source files).
- Ingestion configured `make type-check`: passed (39 source files).
- Explicit mypy on the replacement integration test: passed (1 source file).
- Agent and ingestion focused Ruff format/lint checks: passed.
- Live Neo4j replacement integration test with `infra/.env`: passed (1 test).
- Live read-only Neo4j `EXPLAIN` of `_GRAPH_CAPABILITIES`: passed.
- `git diff --check`: passed.

The complete ingestion integration suite still has an unrelated pre-existing
failure in `test_initial_ingestion`: its reset fixture attempts to delete
`corpus_snapshots` while a `graph_releases_snapshot_ownership_fk` reference
remains. The scoped replacement integration test passes independently.
