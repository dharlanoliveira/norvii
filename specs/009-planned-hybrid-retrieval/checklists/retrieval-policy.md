# Retrieval Policy Checklist: Planned Hybrid Retrieval

**Purpose**: Verify that the feature requirements define an auditable vector-first Hybrid policy.

**Created**: 2026-08-25

**Feature**: [spec.md](../spec.md)

## Vector-first behavior

- [x] Is vector retrieval required for both selectable approaches?
- [x] Does the specification define behavior when vector retrieval returns no evidence?
- [x] Does the specification prevent a graph miss from discarding vector evidence?

## Planning and graph constraints

- [x] Does the specification limit the planner to a snapshot-scoped capability catalog?
- [x] Does the specification prohibit arbitrary model-authored graph queries?
- [x] Are graph skipped, no-evidence, unavailable, and failed outcomes distinct?
- [x] Is graph release availability optional for Hybrid request completion?

## Evidence and inspection

- [x] Are vector and graph contributions attributable after evidence deduplication?
- [x] Are graph paths required to retain immutable supporting locations?
- [x] Are planning and graph diagnostics safe to display without hidden reasoning?
- [x] Is a completed request explicitly distinguished from a contributing graph stage?

## Experience and compatibility

- [x] Are the only new-request choices Vector and Hybrid?
- [x] Is the chosen approach preserved in the research record?
- [x] Is localized product copy required for all new stages?
- [x] Are historical standalone Graph records preserved without creating new ones?

## Notes

- The policy is complete enough for implementation planning. No unresolved requirement blocks a
  plan or task list.
