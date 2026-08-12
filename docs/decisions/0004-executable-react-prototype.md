# ADR 0004: Executable React Prototype Before Production

- Status: Accepted
- Date: 2026-08-12
- Feature: product experience prototype
- Deciders: project maintainers

## Context

Norvii has interaction-heavy behavior such as streaming chat, navigable citations, source processing, responsive panels, and technical inspection. Static mockups alone do not expose enough behavior for approval, while building the production client and backend before validating the experience creates avoidable rework.

The free Figma plan can support manual exploration, but its limited MCP read quota does not support an intensive Codex feedback loop. The repository already targets React, TypeScript, Vite, assistant-ui, Storybook, and Playwright-compatible testing.

## Decision drivers

- Review product behavior in a real browser before production architecture is implemented.
- Give Codex structured, versioned, and executable design context.
- Validate complete UI states, not only ideal screenshots.
- Avoid requiring backend services, model providers, databases, or paid design tooling.
- Preserve a clean boundary between exploratory and production code.

## Options considered

### Executable React prototype in the repository

The prototype uses the intended client stack with deterministic fixtures. It supports browser review, Storybook states, interaction tests, and visual baselines.

### Figma as the primary prototype

Figma is effective for visual exploration but provides a constrained agent feedback loop on the free plan and does not fully exercise chat streaming or application state.

### Implement the production client immediately

This produces reusable code sooner but couples product discovery to contracts, architecture, and backend decisions that have not been validated.

## Decision

Create executable prototypes under `prototypes/`, beginning with `prototypes/web/`. The first Spec Kit feature validates the core product journeys with React, TypeScript, Vite, assistant-ui, deterministic fixtures, Storybook, and browser-based tests selected in its plan.

Keep production applications under `apps/`. Production modules MUST NOT depend on prototype code. Production implementation starts only after the prototype feature is verified and its baseline is approved.

Figma remains optional for sketches, reference boards, or later handoff. It is not a required source of truth.

## Consequences

### Positive

- Product decisions are reviewed before backend and contract investment.
- Approved states become executable context for coding assistants.
- The prototype works without external services or usage costs.
- Storybook and browser tests make feedback reproducible.

### Negative

- Some UI code will be intentionally reimplemented for production.
- The team must resist accidental imports from `prototypes/` into `apps/`.
- Prototype fidelity can create false confidence about unimplemented legal and retrieval behavior.

## Verification

The prototype feature must satisfy the approval gate in the [prototyping workflow](../development/prototyping.md). Production feature plans must link to the approved baseline and identify intentional differences.

## References

- [Executable prototyping workflow](../development/prototyping.md)
- [Feature map](../product/feature-map.md)
- [Repository structure](../architecture/repository-structure.md)
