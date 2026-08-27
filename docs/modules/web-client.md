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
- interface language selection and locale-sensitive presentation;
- active snapshot identity, candidate-release state, publication controls/status, and
  on-demand snapshot-manifest inspection;
- retrieval-strategy selection, graph-path inspection, and independent strategy comparison;
- opening-suggestion presentation and maintainer evaluation readiness, run inspection,
  and comparison presentation;
- client-side validation that improves feedback without replacing server validation.

## Does not own

- retrieval, citation truth, or legal-answer policy;
- corpus authorization or isolation enforcement;
- prompt construction or model-provider calls;
- document extraction or chunking;
- persistence rules.
- maintainer authorization, evaluation execution, scoring, or evaluation-record storage.

## Boundary model

The client consumes versioned HTTP and streaming contracts. Structured citations,
retrieval traces, graph paths, tool events, and metrics remain typed message parts;
the client MUST NOT reconstruct them by parsing Markdown.

Transport clients map external schemas into feature-owned client models. UI
components receive those models and do not call `fetch` directly.

## Evaluation presentation boundary

The workspace obtains opening suggestions from the separate public
`/api/v1/corpora/{corpusId}/opening-suggestions?interfaceLanguage=en|pt` contract.
It renders only the current corpus, current active snapshot, requested language, and
rank-ordered original questions. An absent or stale projection is an empty state.
Selecting a question uses the ordinary corpus-bound chat path; the suggestion response
never contains evaluation answers, evidence, scoring, review data, or provider data.

The maintainer experience uses `/evaluations`, `/evaluations/:runId`, and
`/evaluations/compare` to show preflight state, frozen identity, safe case failures,
provenance, metrics, and non-comparable outcomes. The browser client uses same-origin
credentials and must not contain a bearer token. The API is the sole enforcement point
for maintainer authorization, source binding, compatibility preflight, and snapshot
publication/activation, and returns safe diagnostics. Client parsing rejects malformed
or evaluation-leaking opening-suggestion responses before they reach UI state.

## Corpus snapshot presentation

The catalog shows the active immutable snapshot for each corpus without displacing its language
or jurisdiction. The workspace keeps research bound to that release: chat citations include the
snapshot identity returned by the stream, and the source workspace distinguishes an active
document from a ready candidate. Publication and activation are API-owned operations;
the client exposes the maintainer controls and resulting state. A maintainer must
explicitly publish a ready candidate.

Snapshot history and manifests load only when requested from the source workspace. The view
exposes the snapshot identity, manifest hash, source revisions, official origins, capture times,
document identities, and content hashes so an evaluator can inspect a release without treating
raw retrieval internals as normal reading UI.

The same on-demand inspection surface shows the active snapshot's graph-release readiness and
entity/relationship counts. Chat lets an evaluator choose vector, graph, or hybrid retrieval;
the comparison tool reruns the latest question independently for all three strategies. A graph
or hybrid result remains explicitly unavailable when no ready graph release exists, and citation
or graph-path evidence opens the immutable source location without losing the answer.

## Internationalization boundary

English and Portuguese product copy is owned by versioned client localization resources, with English as the default locale. React pages, components, hooks, schemas, and adapters MUST reference stable localization keys instead of embedding user-facing product text directly.

The rule covers visible copy and assistive text, including headings, navigation, actions, labels, placeholders, validation, loading, empty, error, retry, status, inspection, `aria-label`, `aria-description`, live-region, and image alternative text. Tests may use literal text to assert rendered behavior. Legal documents, citations, corpus data, user input, and generated answer content are domain content and MUST NOT be routed through interface translation resources.

Each localization key MUST exist in both the English and Portuguese resources. Missing entries MUST be detected by automated verification. Runtime fallback to English may prevent an unusable screen, but a fallback is a defect and cannot satisfy the feature acceptance gate.

## Target organization

```text
apps/web/src/
|-- app/                     # Composition, providers, routing
|-- features/                # Vertical client behavior by product feature
|-- components/              # Reusable presentation with proven reuse
|-- contracts/               # Generated public contract types
|-- i18n/                    # Locale setup, stable keys, and per-language resources
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
