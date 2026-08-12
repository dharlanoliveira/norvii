# Spec-Driven Development

Norvii uses GitHub Spec Kit as the path from a product capability to verified code.
The global roadmap identifies candidate features; a numbered feature directory owns
the requirements, design, tasks, and verification for one delivery increment.

The first feature is the executable product prototype. Production features start only after its approval gate passes. See [executable prototyping](prototyping.md).

## Working unit

A feature is a vertical, independently demonstrable behavior. It may touch several
modules. It is not a technical layer such as "create the Go backend" or "build the
React client" unless the user-visible outcome is a runnable development experience.

Use this lifecycle:

```text
product intent
    -> spec.md
    -> clarify when needed
    -> quality checklist
    -> plan.md + research.md + data-model.md + contracts/ + quickstart.md
    -> tasks.md
    -> analyze
    -> implementation approval
    -> implementation
    -> converge and verification
```

## Before specification

1. Select one item from the [feature map](../product/feature-map.md).
2. Confirm that no existing numbered feature already owns the behavior.
3. Describe the user or evaluator outcome, not the framework work.
4. State explicit exclusions to prevent the feature from absorbing later roadmap
   items.
5. For a production UI feature, identify the approved prototype baseline and any intentional difference.

## Specification

Run `speckit-specify`. The resulting `spec.md` owns:

- prioritized, independently testable user stories;
- acceptance scenarios and edge cases;
- functional requirements with stable identifiers;
- scope boundaries and assumptions;
- measurable, technology-independent success criteria.

The specification MUST describe what and why. It MUST NOT select a database,
framework package, table layout, endpoint shape, or code pattern.

## Clarification

Run `speckit-clarify` before planning when an unanswered question changes data
ownership, public contracts, safety, legal evidence rules, cost, or acceptance. Small
implementation details can remain for research.

## Planning

Run `speckit-plan`. The plan MUST identify every affected module and explain why.
Expected artifacts are:

| Artifact | Purpose |
| --- | --- |
| `plan.md` | Technical approach, module impact, quality gates, and structure |
| `research.md` | Evidence for unresolved technical decisions and rejected options |
| `data-model.md` | Entities, invariants, state transitions, and ownership |
| `contracts/` | Feature-owned API, event, stream, or artifact schemas |
| `quickstart.md` | Reproducible setup and acceptance verification |

Production UI plans MUST link to the approved prototype stories, screenshots, or journeys they implement. A deviation is valid only when the plan explains the product or engineering reason.

When research resolves a consequential or hard-to-reverse project choice, add an ADR
under `docs/decisions/` and link it from the plan.

## Quality checklist

Run `speckit-checklist` after clarification and before accepting the specification. The checklist tests whether requirements are complete, unambiguous, measurable, and consistent with the corpus, evidence, prototype, and cost boundaries that apply to the feature.

## Tasks and analysis

Run `speckit-tasks`. Tasks MUST:

- reference their user story and requirement identifiers;
- name exact repository paths;
- include tests and contract checks with the behavior they protect;
- include documentation, migration, security, and observability work where introduced;
- end each user story at an independently runnable checkpoint.

Run `speckit-analyze` before implementation. Resolve critical conflicts between the
specification, plan, contracts, constitution, and tasks before editing application
code.

The implementation approval gate confirms that analysis findings are resolved. A production UI feature cannot pass this gate until its prototype baseline is marked Verified.

## Implementation

Run `speckit-implement` against the selected feature. The active plan supplies the
technical context, while `AGENTS.md` supplies repository-wide code generation rules.
Do not implement a later user story merely because its code is adjacent.

Run `speckit-converge` after implementation. Convergence executes the verification defined by the plan and tasks, diagnoses failures, applies in-scope corrections, and repeats until the required checks pass or a genuine blocker is documented.

The bundled workflow is extended by the project-owned Norvii overlay. Run the composed lifecycle with:

```bash
specify workflow run speckit -i spec="Describe the capability and outcome"
```

See [development tooling](tooling.md) for layer ownership and upgrade rules.

## Completion evidence

A feature can be marked complete only when:

- acceptance scenarios pass;
- relevant unit, integration, and contract tests pass;
- formatters, static analysis, and builds pass for affected modules;
- migrations and schemas match the implementation;
- the quickstart works from a clean environment;
- global documents and ADRs are updated when durable behavior changed;
- remaining risks and deferred requirements are explicit.

## Global versus feature documentation

Use global documents for stable rules shared by many features. Use feature artifacts
for details that exist because of one capability. If a fact appears in both, the
global document states the rule and the feature document links to it while describing
only the local impact.
