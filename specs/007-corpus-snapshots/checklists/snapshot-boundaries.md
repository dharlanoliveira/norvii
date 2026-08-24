# Snapshot Boundary Checklist: Bilingual Corpus Snapshots

**Purpose**: Review the written snapshot, publication, and evidence-boundary requirements before implementation.
**Created**: 2026-08-24
**Feature**: [spec.md](../spec.md)

## Requirement Completeness

- [x] CHK001 Are immutable snapshot identity, membership, provenance, and active-release selection specified separately? [Completeness, Spec FR-002, FR-008]
- [x] CHK002 Are candidate creation, validation, explicit publication, duplicate handling, and activation requirements defined? [Completeness, Spec FR-005 through FR-010]
- [x] CHK003 Are catalog, workspace, history, and chat boundary requirements identified without expanding the researcher workflow beyond scope? [Completeness, Spec FR-003, FR-004, FR-011]

## Requirement Clarity and Consistency

- [x] CHK004 Is the distinction between an ingested candidate and an active published snapshot unambiguous? [Clarity, Spec User Story 2, FR-005, FR-007]
- [x] CHK005 Are corpus, language, snapshot, source revision, document, and legal-location boundaries consistent across user stories and requirements? [Consistency, Spec User Stories, FR-002 through FR-004]
- [x] CHK006 Is the duplicate-publication condition defined using the complete source-revision and content-identity set rather than a single source update? [Clarity, Spec FR-009]

## Scenario and Failure Coverage

- [x] CHK007 Are failed acquisition, incomplete retrieval artifacts, stale releases, wrong-corpus candidates, and unavailable history addressed? [Coverage, Spec Edge Cases, FR-006, FR-010]
- [x] CHK008 Are historical inspection and clean-environment reproduction outcomes specified independently from activation? [Coverage, Spec User Story 3, FR-008, FR-012]
- [x] CHK009 Are measurable initialization, isolation, reingestion, publication, and reproduction outcomes stated? [Measurability, Spec SC-001 through SC-005]

## Scope and Assumptions

- [x] CHK010 Are the two-corpus POC scope, maintainer assumption, and excluded graph, scheduler, and evaluation work explicit? [Completeness, Spec Assumptions, Scope and Boundaries]
- [x] CHK011 Does the specification preserve the no-silent-substitution legal-answer rule after reingestion? [Consistency, Spec Evidence and Corpus Boundaries, FR-003]

## Notes

- Review result: all requirements are sufficiently specified for the approved implementation scope.
