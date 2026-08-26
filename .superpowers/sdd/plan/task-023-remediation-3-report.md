# Task 023 third remediation report

## Delivered repair

- Updated the provenance-complete release subquery to the current Neo4j `CALL () { ... }` syntax.
- Added both a deterministic semantic graph fixture and a service-backed Neo4j replacement test. They seed v1 and v2 releases for one corpus and snapshot plus unrelated corpus and snapshot releases, then prove that replacement retires only the target releases and derived nodes.
- Extended release-candidate coverage with an incomplete v2 candidate and a separate provenance-complete v2 candidate. Retrieval returns only the complete release.

## Verification

- Focused agent retrieval tests, Ruff, and mypy passed.
- Focused ingestion persistence unit tests, integration replacement test against the running Neo4j service, Ruff, and mypy passed.
- `git diff --check` passed.
