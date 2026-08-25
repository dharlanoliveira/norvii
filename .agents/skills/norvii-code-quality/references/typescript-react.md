# TypeScript and React Guidelines

## TypeScript

- Enable and preserve strict TypeScript checking. Do not use `any` to bypass a design or integration problem.
- Use `unknown` at untrusted boundaries and narrow it through runtime validation.
- Model variants with discriminated unions and exhaustive checks.
- Keep API DTOs, UI view models, and domain concepts distinct when their evolution differs.
- Derive types from shared schemas where practical; do not maintain independent handwritten copies of the same contract.
- Avoid broad type assertions, non-null assertions, enums with unclear serialization, and ambient mutable state.
- Keep modules cohesive and expose narrow named exports.

## React

- Build functional components with hooks and composition.
- Keep rendering declarative. Move orchestration and external effects into focused hooks or service modules.
- Keep state as local as possible and derive values instead of synchronizing duplicate state.
- Use context for stable cross-cutting dependencies or state, not as a default global store.
- Separate server state from local UI state. Use a dedicated query/cache library only when its behavior is justified.
- Keep accessibility in component contracts: semantic elements, keyboard behavior, labels, focus management, and announced async states.
- Render assistant messages from structured parts. Do not parse generated Markdown to reconstruct citations, tool calls, graph paths, or metrics.
- Treat assistant-ui and AI SDK as adapters. Keep Norvii domain models independent of library-specific message types where practical.
- Reference stable localization keys for all product-authored user-facing and assistive text. Do not embed interface copy in React pages, components, hooks, schemas, or adapters.
- Keep English and Portuguese resources structurally complete and type-safe, with English as the default locale. Do not translate legal content, citations, user input, or generated answers through interface resources.
- Avoid large page components, prop drilling across many layers, effect-driven state machines, and premature memoization.

## React orchestration and complexity control

- A page component may compose feature concerns, but it must not own several independent lifecycles at once. Treat server loading and refresh, mutation forms, source or document selection, streaming, and transient display state as separate concerns.
- When a component starts coordinating more than one of those concerns, extract the lifecycle into a focused hook or child component. Give the extracted unit a domain name and a small input/output contract; do not hide the same branching in a generic helper.
- Keep async effects and their cleanup next to the state they own. A hook that polls sources should not also own form visibility; a document-selection hook should not also mutate source records.
- Use a complexity checkpoint before handoff for changed React pages, hooks, and event handlers: review each conditional branch for a distinct responsibility, and split independent decision trees before static analysis reports cognitive complexity.
- If SonarCloud or another configured analyzer reports cognitive complexity, fix the design and add user-visible tests for the extracted behavior. Do not disable the rule, raise the threshold, or replace the finding with superficial wrappers.

## Static-analysis-safe React expression patterns

- Do not use nested ternaries in JSX or view-model construction. Select a named value with a short `if`/`else` decision or extract a focused rendering function when the states have domain meaning.
- Do not reassign function parameters. Use a default parameter when the fallback is part of the contract, otherwise create a clearly named local value.
- Do not nest template literals. Compute dynamic localization keys and formatted values in named locals before composing the final text.
- Prefer native semantic behavior over redundant ARIA. Do not add an explicit `role` when the element already exposes that implicit role; add ARIA only when it supplies information that native semantics cannot.
- Before handoff, review changed TypeScript and React files for configured Sonar-style findings: nested conditional expressions, parameter reassignment, nested interpolation, and redundant accessibility roles. Correct the expression or component structure rather than suppressing the finding.

## Network and streaming

- Validate responses and streaming parts at the client boundary.
- Represent loading, partial, complete, empty, cancelled, and failed states explicitly.
- Abort requests when the user cancels or the owning view unmounts.
- Keep retries safe and visible. Do not retry mutations without idempotency guarantees.
- Never expose backend credentials or provider keys in the browser bundle.

## Testing and tooling

- Test user-visible behavior with Testing Library semantics rather than component internals.
- Add focused tests for reducers, parsers, schema validation, and streaming reconciliation.
- Cover keyboard and accessible-name behavior for interactive components.
- Verify locale-resource parity and fail checks for missing keys or hardcoded product-authored interface copy.
- Run the configured formatter, linter, type checker, unit tests, and production build.
- If tools are not selected yet, propose ESLint, Prettier, Vitest, and Testing Library rather than silently adding a stack.
