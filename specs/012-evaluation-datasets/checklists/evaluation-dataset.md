# Evaluation Dataset Requirements Checklist: Versioned Evaluation Datasets

**Purpose**: Assess whether the evaluation-dataset requirements are complete, clear, and safe

**Created**: 2026-08-25

**Feature**: [spec.md](../spec.md)

## Dataset Provenance and Integrity

- [x] CHK001 Are immutable dataset revision and source-manifest requirements specified for every curated dataset? [Completeness, SpecFR-001, ?FR-002]
- [x] CHK002 Are corpus, jurisdiction, snapshot-date, and source requirements defined independently enough to prevent cross-corpus substitution? [Completeness, SpecFR-004]
- [x] CHK003 Are malformed assets, duplicate case identifiers, missing source references, and broken language pairs explicitly rejected? [Coverage, SpecFR-003, Edge Cases]
- [x] CHK004 Is the condition for a draft dataset to become available distinguished from the condition for a run to complete? [Clarity, SpecAssumptions, ?FR-003]

## Evidence and Authority

- [x] CHK005 Are expected evidence and actual generated citations explicitly distinguished so that the reference answer cannot substitute for answer grounding? [Clarity, SpecFR-006, Evidence Behavior]
- [x] CHK006 Are requirements defined for preserving legal locators when a source revision changes its structure? [Coverage, Edge Cases, SpecFR-004]
- [x] CHK007 Are statutory, regulatory, case, and guidance authority roles consistently specified for scoring and inspection? [Consistency, SpecFR-007]
- [x] CHK008 Is the response when a snapshot lacks required evidence unambiguous and measured before model generation? [Completeness, SpecFR-004, SC-002]

## Bilingual Evaluation

- [x] CHK009 Are query language, expected answer language, and authoritative source language separately required for every case? [Completeness, SpecFR-002]
- [x] CHK010 Are reciprocal cross-language case links defined and validated without requiring the two answers to use the same prose? [Clarity, SpecFR-003, SC-005]

## Results and Comparison

- [x] CHK011 Are all run identities necessary for reproduction specified at case and aggregate level? [Completeness, SpecFR-005, ?FR-011]
- [x] CHK012 Are success measures defined for evidence support, expected-evidence retrieval, citation validity, abstention, latency, and token use? [Completeness, SpecFR-008]
- [x] CHK013 Is the treatment of individual execution failures separated from quality scoring and aggregate summaries? [Consistency, SpecFR-010]
- [x] CHK014 Are incompatible run comparisons visibly bounded rather than presented as quality deltas? [Clarity, SpecFR-009, SC-006]

## Safety and Scope

- [x] CHK015 Is the feature's boundary between technical evaluation and legal advice defined for both results and user-facing presentation? [Completeness, SpecFR-012]
- [x] CHK016 Are personal-data, raw-prompt, and provider-diagnostic exclusions specified for results and logs? [Coverage, SpecCost and Operational Limits]
