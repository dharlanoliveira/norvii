# Reconciliation: GraphRAG and Hybrid Retrieval

**Date**: 2026-08-25

## Scope

This audit compares Feature 008 requirements and unchecked historical tasks with the current
repository state. Feature 009 is the later product decision: the client exposes Vector and
planned Hybrid retrieval, not a standalone Graph strategy. Feature 011 later replaces the
canonical direct semantic-relationship model with evidence-backed normative assertions for every
new graph release after its controlled local reset.

## Implementation Present

| Capability | Evidence | Historical tasks |
| --- | --- | --- |
| Canonical graph-release model, persistence, and HTTP inspection | `apps/api/internal/graphrelease/` and `apps/api/migrations/008_graphrag.sql` | T004, T005, T016 |
| Rebuildable graph projection and semantic artifact persistence | `apps/ingestion/src/norvii_ingestion/semantic/`, `apps/ingestion/src/norvii_ingestion/graph_projection/`, and `apps/ingestion/src/norvii_ingestion/release/` | T013-T015, T027, T028 |
| Graph-ready snapshot activation | `apps/api/internal/snapshot/` and `apps/ingestion/src/norvii_ingestion/release/coordinator.py` | T027, T028 |
| Client inspection, release status, and strategy comparison | `apps/web/src/features/source-management/SourceStatus.tsx`, `apps/web/src/features/workspace/StrategyComparison.tsx`, and `apps/web/src/research/domain/strategyComparison.ts` | T017-T019, T030 |
| Vector-first planned Hybrid retrieval | `apps/agent/src/norvii_agent/retrieval/{planning,graph,hybrid}.py` and `specs/009-planned-hybrid-retrieval/` | T006-T011, T018-T020 |

## Evidence Gaps

The audit did not find service-backed PostgreSQL/Neo4j isolation coverage, a three-run graph
projection reproducibility journey, or Playwright coverage for the final Vector/Hybrid strategy
model. It also found no recorded configured-provider reingestion and graph-build acceptance for
both seeded corpora.

## Superseded Design

Tasks that require a standalone Graph selector are superseded by Feature 009. The remaining
validation must demonstrate Vector and Hybrid behavior, including a safe unavailable graph stage,
without restoring the removed product option.

Feature 011 supersedes direct `semantic_relationships` persistence and direct relationship-edge
inspection for future releases. It keeps Feature 008's snapshot isolation, graph-release
lifecycle, and evidence-grounding requirements, but represents the supported statement as a
normative assertion with distinct establishing and evidence legal units. This note does not alter
the historical Feature 008 acceptance evidence.

## Reconciled Execution Path

The Phase 8 tasks in [tasks.md](tasks.md#phase-8-reconciliation) consolidate the remaining
verification and integration work.
