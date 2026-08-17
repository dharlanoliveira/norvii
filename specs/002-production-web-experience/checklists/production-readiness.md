# Production Readiness Requirements Checklist: Production Corpus Research Experience

**Purpose**: Validate that the production promotion, TDD, accessibility, localization, isolation, and performance requirements are complete and reviewable
**Created**: 2026-08-13
**Feature**: [Specification](../spec.md)

**Note**: This checklist evaluates requirement quality, not implementation behavior.

## Requirement completeness

- [x] CHK001 Are catalog, source, chat, citation, recovery, localization, and responsive journeys all represented by independently testable stories? [Completeness, Spec User Stories 1-4]
- [x] CHK002 Are empty, unavailable, abstained, failed, direct-route, and state-preservation scenarios explicitly bounded? [Coverage, Spec Edge Cases and FR-004, FR-011-FR-017]
- [x] CHK003 Are production ownership and the prohibition on prototype imports stated for code, styles, fixtures, and assets? [Completeness, Spec FR-023]
- [x] CHK004 Are all excluded backend, persistence, ingestion, model, and graph capabilities explicitly listed? [Completeness, Spec FR-024 and Scope and Boundaries]

## Requirement clarity

- [x] CHK005 Is the active corpus boundary defined for sources, conversations, responses, and citations? [Clarity, Spec FR-003, FR-014, and Evidence and Corpus Boundaries]
- [x] CHK006 Is the distinction between interface localization and preserved domain content unambiguous? [Clarity, Spec FR-018-FR-019]
- [x] CHK007 Are prepared answered, abstained, and failed outcomes distinguishable and recoverable according to explicit rules? [Clarity, Spec FR-013-FR-017]
- [x] CHK008 Are the approved viewport sizes and prohibited layout failures quantified? [Clarity, Spec FR-022]

## Requirement consistency

- [x] CHK009 Does session-only state consistently align with mode and language preservation without promising refresh persistence? [Consistency, Spec FR-012, FR-019, and Assumptions]
- [x] CHK010 Does the production baseline preserve the approved visual identity while requiring independent production ownership? [Consistency, Spec FR-021, FR-023, and Prototype Baseline]
- [x] CHK011 Do the structured citation requirements align with the prohibition on unsupported legal conclusions? [Consistency, Spec FR-013-FR-016]

## Acceptance criteria quality

- [x] CHK012 Can every primary scenario be observed through public user behavior without inspecting component internals? [Measurability, Spec SC-001]
- [x] CHK013 Are accessibility outcomes tied to keyboard completion and automated scan results? [Measurability, Spec SC-002]
- [x] CHK014 Are localization completeness and the default locale objectively measurable? [Measurability, Spec SC-003]
- [x] CHK015 Are bundle size and initial interactivity expressed with numeric thresholds? [Measurability, Spec FR-027 and SC-006]

## Non-functional and dependency boundaries

- [x] CHK016 Are WCAG expectations specified for semantic structure, contrast, keyboard behavior, focus, names, tree behavior, and announcements? [Coverage, Spec FR-020]
- [x] CHK017 Is the external-link safety boundary limited to fixed HTTPS destinations opened in a safe new context? [Security, Spec FR-010 and Assumptions]
- [x] CHK018 Is the zero-service, zero-network-corpus, and zero-model-cost expectation objectively stated? [Dependency, Spec FR-024 and SC-008]
- [x] CHK019 Is the future API replacement seam required without implying a premature public transport contract? [Dependency, Spec FR-025 and Assumptions]
- [x] CHK020 Is TDD evidence required for acceptance behavior, locale parity, keyboard interaction, accessibility, and build verification? [Coverage, Spec FR-028]

## Notes

- All requirements-quality questions are resolved by the specification and approved prototype baseline.
- Implementation evidence is produced by `tasks.md`, automated tests, and the feature quickstart.
