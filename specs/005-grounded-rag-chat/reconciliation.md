# Reconciliation: Grounded RAG Chat

**Date**: 2026-08-25

## Scope

This audit compares Feature 005 requirements and unchecked historical tasks with the current
repository state. It records implementation presence, not a substitute for the acceptance
evidence required to mark a task complete.

## Implementation Present

The following capabilities exist in the current codebase and are mapped to the historical task
list:

| Capability | Evidence | Historical tasks |
| --- | --- | --- |
| Bounded provider configuration and OpenAI-compatible adapters | `apps/api/internal/platform/config/`, `apps/agent/src/norvii_agent/config/`, `apps/agent/src/norvii_agent/providers/`, `apps/ingestion/src/norvii_ingestion/config/`, and `apps/ingestion/src/norvii_ingestion/enrichment/embedding/` | T001, T023, T039 |
| Immutable vector storage, chunking, and embedding publication | `apps/api/migrations/005_grounded_rag.sql`, `apps/ingestion/src/norvii_ingestion/enrichment/`, and `apps/ingestion/src/norvii_ingestion/orchestration/processor.py` | T004-T011, T014, T019 |
| Grounded retrieval, answer validation, and safe terminal outcomes | `apps/agent/src/norvii_agent/graph/grounded_chat.py`, `apps/agent/src/norvii_agent/retrieval/postgres.py`, and their unit tests | T015, T020, T021, T033, T035 |
| Public SSE handling and browser stream validation | `apps/api/internal/chat/`, `apps/agent/src/norvii_agent/transport/`, `apps/web/src/api/chat.ts`, and their tests | T012, T013, T016, T017, T024, T034, T036 |
| Immutable citation navigation and inspection | `apps/api/internal/document/`, `apps/web/src/features/workspace/ResearchChat.tsx`, `apps/web/src/features/workspace/LegalDocumentReader.tsx`, and Feature 006 coverage | T027-T032 |
| Localized user-facing states and deterministic browser coverage | `apps/web/src/i18n/`, `apps/web/src/features/workspace/ResearchChat.test.tsx`, and `apps/web/tests/e2e/corpus-ingestion.spec.ts` | T025, T026, T032, T034, T036, T037 |

## Evidence Gaps

The audit did not find feature-owned stream fixtures, a feature-specific contract-validator test,
or the original dedicated `grounded-rag-chat.spec.ts` journey. The quickstart does not yet record
the requested two-run deterministic acceptance evidence. The intended contract-promotion decision
also remains unrecorded.

## Reconciled Execution Path

The Phase 7 tasks in [tasks.md](tasks.md#phase-7-reconciliation) consolidate the remaining work.
They replace duplicated implementation work with contract, service-backed, browser, operational,
and contract-governance verification.
