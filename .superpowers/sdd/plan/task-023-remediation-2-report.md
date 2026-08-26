# Task 023 second remediation report

## Delivered repair

- Graph projection replacement now atomically removes every prior Neo4j release and its derived nodes for the target corpus and snapshot before materializing the replacement. Other corpus and snapshot releases are outside that Cypher match.
- Neo4j graph releases now persist their graph build version. Retrieval, capabilities, and planned retrieval select one deterministic `legal-assertion-graph-v2` release only after confirming every assertion has the required evidence unit, canonical locator, and content hash.
- Added deterministic coverage with coexisting incomplete v1 and provenance-complete v2 projection fixtures. The regression test confirms graph retrieval returns only the v2 evidence. Existing vector fallback behavior is unchanged.

## Verification

- Focused agent graph retrieval tests, Ruff, and mypy passed.
- Focused ingestion graph builder and Neo4j persistence tests, Ruff, and mypy passed.

## Limitation

No live Neo4j instance is available in this validation environment, so the atomic Cypher replacement and release-selection query were verified through deterministic adapter fixtures and static query assertions rather than an integration database run.
