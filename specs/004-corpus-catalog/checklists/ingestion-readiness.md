# Ingestion Readiness Checklist: Corpus Catalog and Ingestion

**Purpose**: Validate that requirements and design are complete enough for a security-sensitive, cross-module ingestion implementation
**Created**: 2026-08-17
**Feature**: [Specification](../spec.md)

**Note**: This checklist tests the requirements and design artifacts, not the implementation.

## Scope and Ownership

- [x] CHK001 Are the real end-to-end outcome and the removal of runtime fixtures explicit? [Completeness, Spec User Story 1, FR-031]
- [x] CHK002 Are module mutation responsibilities assigned once without internal cross-language imports? [Consistency, Plan Module Impact, Contract ingestion-work Ownership]
- [x] CHK003 Are model, embedding, graph, OCR, chat, and citation exclusions explicit and consistent? [Scope, Spec FR-037-FR-038]
- [x] CHK004 Is the trusted single-user assumption distinguished from input-safety requirements? [Clarity, Spec Assumptions, Plan Security and privacy]

## Data and Lifecycle Requirements

- [x] CHK005 Are identities, uniqueness, corpus ownership, concurrency, and seed idempotency specified? [Completeness, Spec FR-002-FR-011, Data Model]
- [x] CHK006 Are source, work, attempt, revision, document, and unit lifecycle transitions unambiguous? [Clarity, Spec FR-020-FR-028, Contract ingestion-work]
- [x] CHK007 Is last-ready preservation during retry and failed reprocessing defined consistently? [Consistency, Spec User Story 5, Data Model Source]
- [x] CHK008 Are atomic publication and incomplete-artifact visibility requirements explicit? [Recovery, Spec FR-026-FR-028]
- [x] CHK009 Are immutable history and unchanged-content idempotency objectively defined by hashes and pipeline version? [Measurability, Research Idempotent publication]

## Acquisition and Security Requirements

- [x] CHK010 Are HTTPS protocol, DNS/address class, address pinning, TLS hostname, redirect, timeout, proxy, type, and size requirements all documented? [Security, Spec FR-016-FR-018, Research Safe URL acquisition]
- [x] CHK011 Are PDF validity, encryption, extractable-text, binary retention, filename, media, and size boundaries complete? [Security, Spec FR-014, FR-016-FR-017]
- [x] CHK012 Are duplicate checks scoped so they cannot disclose foreign-corpus records? [Security, Spec FR-019]
- [x] CHK013 Are logs and public failures bounded to safe categories and prohibited from copying content or secrets? [Privacy, Spec FR-021, FR-030]
- [x] CHK014 Are crash, expired lease, partial extraction, partial publication, and offline initial-source cases addressed? [Recovery, Spec Edge Cases, Contract ingestion-work]

## Document Integrity Requirements

- [x] CHK015 Is complete normalized text retained independently from addressable units? [Completeness, Spec FR-023-FR-025]
- [x] CHK016 Are hierarchy, parent, order, offset, page, locator, hash, cycle, and sibling-overlap invariants specified? [Clarity, Spec FR-024-FR-026, Data Model Document Unit]
- [x] CHK017 Is deterministic fallback behavior defined when legal structure cannot be recognized? [Edge Case, Spec FR-025]
- [x] CHK018 Is preservation of legal language separated from interface localization? [Consistency, Spec FR-034-FR-035]

## Contracts and Experience Requirements

- [x] CHK019 Are every required online operation, discriminated origin, state, error, and compatibility rule covered by the HTTP design contract? [Completeness, Contract http-api]
- [x] CHK020 Are provider/consumer ownership, leases, publication, recovery, and breaking-change rules covered by the ingestion contract? [Completeness, Contract ingestion-work]
- [x] CHK021 Are loading, empty, processing, ready, failed, unavailable, retry, stale, keyboard, focus, and confirmation states specified? [UX Coverage, Spec FR-032-FR-036]
- [x] CHK022 Is unavailable chat behavior explicit and free of simulated answer fallback? [Clarity, Spec FR-037]

## Acceptance and Operations

- [x] CHK023 Are latency, ingestion time, isolation, idempotency, accessibility, and clean-bootstrap outcomes measurable? [Acceptance Criteria, Spec SC-001-SC-013]
- [x] CHK024 Are live-source acceptance and deterministic automated-test dependencies clearly separated? [Reliability, Spec Assumptions, Quickstart]
- [x] CHK025 Are one-command startup, component logs, migrations, initial ingestion, inspection, stop, and quality-gate expectations documented? [Operations, Quickstart]
- [x] CHK026 Does the design avoid an unnecessary service while preserving recoverable work semantics? [Complexity, Research Durable work dispatch]

## Notes

- All 26 requirement-quality gates pass before task generation.
- Implementation evidence is evaluated later by feature tasks and convergence; these checks do not claim that code exists.
