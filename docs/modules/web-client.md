# Web Client Model

## Mission

`apps/web/` presents corpus selection, source browsing, grounded chat, citations, and
technical inspection as a React and TypeScript SPA built with Vite.

This document describes the production client. Product exploration lives in `prototypes/web/` and follows the [executable prototyping workflow](../development/prototyping.md). The production client does not import prototype code.

## Owns

- navigation and page composition;
- accessible UI state and interaction behavior;
- assistant-ui message, composer, source, tool, and artifact presentation;
- adaptation between API stream parts and client view models;
- loading, empty, error, retry, and abstention states;
- client-side validation that improves feedback without replacing server validation.

## Does not own

- retrieval, citation truth, or legal-answer policy;
- corpus authorization or isolation enforcement;
- prompt construction or model-provider calls;
- document extraction or chunking;
- persistence rules.

## Boundary model

The client consumes versioned HTTP and streaming contracts. Structured citations,
retrieval traces, graph paths, tool events, and metrics remain typed message parts;
the client MUST NOT reconstruct them by parsing Markdown.

Transport clients map external schemas into feature-owned client models. UI
components receive those models and do not call `fetch` directly.

## Target organization

```text
apps/web/src/
|-- app/                     # Composition, providers, routing
|-- features/                # Vertical client behavior by product feature
|-- components/              # Reusable presentation with proven reuse
|-- contracts/               # Generated public contract types
|-- lib/                     # Narrow framework adapters
`-- test/                    # Shared test setup only
```

Feature code keeps components, hooks, state, adapters, and tests together until
reuse across features is demonstrated. Avoid generic `utils`, global stores without
need, and components that combine network, domain, and presentation decisions.

## Verification model

- unit tests for transformations and interaction rules;
- component tests for visible behavior and accessibility;
- contract tests for generated schema compatibility;
- end-to-end tests for critical user journeys;
- type checking, linting, formatting, and production build for affected changes.

Mock at the network or provider boundary. Prefer rendering real feature components
over mocking their internals.
