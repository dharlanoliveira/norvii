# Normative Graph Requirements Checklist: Evidence-Backed Normative Assertions

**Purpose**: Assess whether the requirements fully and unambiguously describe legal-statement provenance, hierarchy scoping, snapshot isolation, and legacy compatibility.

**Created**: 2026-08-25

**Feature**: [spec.md](../spec.md)

**Note**: This checklist assesses the quality of the written requirements; it does not verify an implementation.

## Requirement Completeness

- [x] CHK001 Are identity, kind, locator, and parent requirements defined for every legal unit? [Completeness, Spec FR-001]
- [x] CHK002 Are hierarchy relationships explicitly separated from semantic assertions? [Completeness, Spec FR-002]
- [x] CHK003 Are both the establishing unit and the evidence unit required for every published assertion? [Completeness, Spec FR-003]
- [x] CHK004 Are the complete assertion endpoints and predicate specified without relying on an implicit direct graph relation? [Completeness, Spec FR-003]
- [x] CHK005 Are all supported predicate names, directions, and meanings explicitly defined? [Completeness, Spec FR-005]
- [x] CHK006 Are conditional qualifications required so that a statement cannot become broader during extraction? [Completeness, Spec FR-006]

## Requirement Clarity and Consistency

- [x] CHK007 Is the rule against copying descendant assertions onto ancestors unambiguous? [Clarity, Spec FR-004]
- [x] CHK008 Are establishing-unit ownership and evidence-unit precision compatible when an item evidences an article-level assertion? [Consistency, Spec FR-003, Edge Cases]
- [x] CHK009 Is the hierarchy-scoping rule limited to matching descendants and minimal explanatory context? [Clarity, Spec FR-009]
- [x] CHK010 Are incomplete endpoint or provenance cases explicitly excluded from grounded graph evidence? [Clarity, Spec FR-007, Edge Cases]
- [x] CHK011 Do the predicate vocabulary and the stated out-of-scope boundary avoid contradictory legacy-predicate behavior? [Consistency, Spec FR-005, FR-012, Out of Scope]

## Snapshot and Compatibility Coverage

- [x] CHK012 Is the corpus-snapshot ownership boundary stated for every graph entity and result? [Completeness, Spec FR-010]
- [x] CHK013 Is the post-validation destructive-reset boundary explicit, ordered, and measurable? [Clarity, Spec FR-011, FR-012, SC-004]
- [x] CHK014 Is the reset scope explicit for corpus records, document versions, snapshots, graph releases, and derived graph data? [Completeness, Spec FR-011]
- [x] CHK015 Are interrupted reset and stale-evidence cases covered as exceptions? [Coverage, Spec FR-013, Edge Cases]

## Acceptance and Observability Coverage

- [x] CHK016 Do the user scenarios cover exact provision retrieval, chapter-scoped retrieval, and model introduction without corpus mutation? [Coverage, User Stories 1-3]
- [x] CHK017 Can each success criterion be objectively evaluated with controlled data or snapshot state? [Measurability, Spec Success Criteria]
- [x] CHK018 Are the origin, hierarchy scope, and retrieval decision visible without using operational logs as the sole explanation? [Completeness, Spec FR-008, FR-013, SC-005]

## Notes

- The prerequisite script for this Spec Kit checklist requires a `plan.md`, which conflicts with the repository workflow ordering that places checklist generation before planning. The checklist was generated from `spec.md` using the canonical template and reviewed directly.
