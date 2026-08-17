# Executable Prototyping

Norvii validates the product experience in an executable React prototype before production implementation begins. The approved prototype is the interaction and visual baseline for later feature specifications, but it is not the production client.

## Purpose

The prototype answers product questions while change is inexpensive:

- whether users understand corpus selection and isolation;
- whether sources and chat can share the workspace without competing for attention;
- how deterministic response staging, citations, abstention, and recoverable errors appear;
- which information belongs in the default research experience;
- which client data is required before public API contracts are designed;
- how the experience behaves on desktop and notebook-sized viewports.

The prototype MUST not be used to validate retrieval quality, legal correctness, persistence, security controls, or production performance.

## Location and boundary

All executable prototypes live under `prototypes/`. The first prototype lives in `prototypes/web/` and uses React, TypeScript, and Vite.

Production applications live under `apps/`. Production code MUST NOT import prototype modules, fixtures, or styling. Prototype code may be promoted only through an explicit production feature that re-evaluates behavior, accessibility, architecture, contracts, and tests. Copying the prototype directory into `apps/web/` is not an implementation strategy.

## Prototype stack

The web prototype uses the intended client technologies where they help validate behavior:

- React and TypeScript for executable interactions;
- Vite for local development;
- assistant-ui for chat primitives and streaming presentation;
- deterministic fixtures and in-memory adapters instead of backend services;
- Vitest and Testing Library for isolated interaction and accessibility states;
- Playwright for approved journeys and visual baselines.

The prototype supports English and Portuguese interface resources from its first executable slice, with English as the default. Product-authored React copy is referenced through localization keys and maintained once per supported language. Legal fixture content remains separate from interface localization.

Package versions and supporting libraries belong to the prototype feature plan. The prototype does not require the Go API, Python ingestion, databases, external URLs, model providers, embeddings, or Docker Compose.

## Required journeys

The initial prototype covers:

1. Browse and select a Portuguese or English corpus.
2. Open a corpus workspace with sources on the left and chat on the right.
3. Open prepared PDF and external-link sources without leaving the workspace.
4. Ask a question and observe a deterministic staged response.
5. Open a citation and locate its source passage.
6. Receive explicit abstention when simulated evidence is insufficient.
7. Review a recoverable deterministic response failure.
8. Complete the primary journeys in English and Portuguese without losing state when the interface language changes.

Source registration, ingestion states, retrieval inspection, GraphRAG visualization, MCP execution, and skills execution belong to later numbered features and are not part of the initial prototype baseline.

## Required state coverage

| Area | Required states |
| --- | --- |
| Corpus catalog | Populated and unknown-route recovery |
| Corpus | Ready and isolated by active selection |
| Source | Available and preview unavailable |
| Chat | Empty, composing, responding, complete, abstained, and failed |
| Answer | Cited and insufficient evidence |
| Citation | Collapsed, open, passage located, and source unavailable |
| Layout | Wide desktop and notebook |

## Fixtures and simulated behavior

Fixtures use realistic but non-authoritative legal examples. They MUST be deterministic, contain no private data, and be visibly identified as prototype data when confusion is possible.

Fixture types may inform later contract design, but they are not public API contracts. The production feature derives versioned contracts from approved user needs and may replace the prototype shape.

Simulated streaming uses a deterministic scheduler or controllable clock. Tests MUST not depend on a live model, network access, or nondeterministic timing.

## Review loop

1. Define journeys, state matrix, and acceptance criteria in the prototype feature specification.
2. Generate two or three materially different workspace directions when the layout is still open.
3. Review the directions in the browser and record the selected direction and rejected tradeoffs.
4. Build the selected journey with fixtures and isolated component tests.
5. Review keyboard behavior, responsive layout, content, and failure states.
6. Capture approved screenshots and journey tests.
7. Mark the prototype baseline approved or return to an earlier step.

Review feedback changes the prototype feature artifacts before production specifications inherit the result.

## Approval gate

Production feature implementation starts only after the prototype baseline is approved. Approval requires:

- all required journeys are navigable without backend services;
- required states are represented and reviewable;
- corpus isolation is understandable in the interface;
- citations and abstention are understandable without technical explanation;
- keyboard navigation and visible focus have been reviewed;
- English and Portuguese interface coverage is complete, with English verified as the default and no hardcoded product-authored React copy;
- notebook and wide desktop layouts have been reviewed;
- approved screenshots, interaction evidence, and outstanding product questions are recorded;
- the prototype feature is marked `Verified`.

Approval freezes a baseline, not the product forever. Later production features may change it through explicit specification and updated visual evidence.

## Relationship to Spec Kit

The prototype is the first numbered Spec Kit feature. Its application code is limited to `prototypes/web/`. Later production features link to the approved prototype baseline from their `spec.md` and explain intentional differences in their `plan.md`.

The prototype specification owns user behavior. The prototype implementation provides executable evidence. Production contracts and architecture remain owned by the later feature that introduces them.
