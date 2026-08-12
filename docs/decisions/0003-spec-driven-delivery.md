# ADR 0003: Spec-Driven Feature Delivery

- Status: Accepted
- Date: 2026-08-12
- Feature: project foundation
- Deciders: project maintainers

## Context

Codex will generate a substantial part of Norvii. The repository must preserve requirements, technical reasoning, contracts, tasks, and verification so generated code remains reviewable and the project can serve as a reference.

## Decision drivers

- Traceability from user value to code and tests.
- Consistent planning across React, Go, and Python.
- Prevention of silent architectural decisions during code generation.
- Incremental delivery of demonstrable capabilities.

## Options considered

### GitHub Spec Kit per vertical feature

Each capability receives specification, clarification when needed, plan, research, contracts, tasks, analysis, implementation, and verification artifacts.

### One evolving global plan

A single file is easy to start but mixes vision, architecture, backlog, decisions, and task state. It duplicates details once feature artifacts exist.

### Code-first generation with README updates

This produces implementation quickly but does not retain sufficient reasoning or acceptance traceability for a reference project.

## Decision

Use GitHub Spec Kit for every product capability. Keep durable sources of truth under `docs/`, governance in the constitution, and one numbered workspace under `specs/` for each vertical feature. Do not maintain a separate global implementation plan.

## Consequences

### Positive

- Requirements, decisions, contracts, and tasks remain reviewable.
- Each feature can stop at an independently testable checkpoint.
- Global documentation contains only durable information.

### Negative

- Small changes require judgment about whether they belong to an existing feature.
- Documents must be updated with the implementation to avoid stale specifications.
- Spec Kit templates and project guidance require maintenance when governance changes.

## Verification

`AGENTS.md` requires the Spec Kit workflow before product implementation. Feature analysis checks constitution, specification, plan, contracts, and tasks before code changes.

## References

- [Spec-driven development](../development/spec-driven-development.md)
- [Feature specifications](../../specs/README.md)
- [Project constitution](../../.specify/memory/constitution.md)
