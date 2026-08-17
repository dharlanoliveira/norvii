# Research: Production Corpus Research Experience

## Production implementation strategy

- **Decision**: Reimplement the approved experience independently under `apps/web/`; use Feature 001 only as a visual and behavioral reference.
- **Rationale**: The constitution prohibits production dependencies on prototype code, fixtures, and styles. Independent ownership keeps the prototype disposable and the production module maintainable.
- **Alternatives considered**: Importing prototype modules was rejected as a direct boundary violation. Moving prototype files into production was rejected because it would erase the experimental history and bypass production design decisions.

## Client boundary before the Go API

- **Decision**: Define a narrow production-client research provider in domain vocabulary and implement it with immutable demonstration records for this feature.
- **Rationale**: Presentation can exercise real production state and accessibility behavior now while a later API feature replaces only the adapter. The boundary returns domain records rather than assistant-ui or transport objects.
- **Alternatives considered**: Calling a placeholder HTTP server adds an unrequested service and false contract. Importing raw fixtures directly into pages couples presentation to data shape.

## Structured chat rendering

- **Decision**: Use assistant-ui through one adapter that translates prepared domain responses into structured text and citation parts.
- **Rationale**: This validates the chosen rendering surface while keeping citations separate from Markdown parsing. The adapter can be replaced when the Go streaming contract exists.
- **Alternatives considered**: A custom chat renderer duplicates library behavior. Parsing citation syntax from Markdown weakens type safety and violates the client architecture guidance.

## Route and bundle strategy

- **Decision**: Keep the catalog in the initial route and lazy-load the workspace, including assistant-ui, as a separate route chunk. Enforce a 350 kB compressed initial-entry budget after build.
- **Rationale**: The prototype documented a bundle advisory caused by eager assistant-ui loading. Route separation keeps the first page focused and gives the budget a deterministic gate.
- **Alternatives considered**: Accepting the prototype warning fails the production performance requirement. Removing assistant-ui avoids the warning but abandons an already selected client adapter without evidence.

## State ownership

- **Decision**: Keep locale in the application provider, corpus-workspace behavior in a focused controller hook, and assistant thread state mounted while Chat and Source visibility changes.
- **Rationale**: State stays near its owner, derived values are not duplicated, and mode changes do not destroy the conversation or draft.
- **Alternatives considered**: A global store is unnecessary at this scale. Encoding all ephemeral state in the URL exposes drafts and produces excessive routing complexity.

## Localization

- **Decision**: Use typed English and Portuguese resource objects with English as the default and an automated recursive parity test.
- **Rationale**: Product copy never appears directly in components, and missing translation keys fail deterministically.
- **Alternatives considered**: Runtime-only fallback hides missing Portuguese content. Separate untyped JSON resources permit key drift.

## Styling and accessibility

- **Decision**: Preserve the approved editorial research-desk direction with production-owned design tokens, semantic HTML, an ARIA tree keyboard model, visible focus, reduced-motion handling, and axe-assisted component tests.
- **Rationale**: The result remains recognizable while meeting production ownership and accessibility gates.
- **Alternatives considered**: A generic component kit would dilute the approved identity. Pixel-copying compiled prototype CSS would violate production ownership and make maintenance opaque.

## Dependency versions

- **Decision**: Pin the production module to the exact Node and frontend dependency versions already validated by Feature 001, with an independent lockfile.
- **Rationale**: This is reproducible, avoids unrelated upgrade research, and isolates production dependency management.
- **Alternatives considered**: Floating or newly selected versions add risk unrelated to this feature. Sharing the prototype lockfile couples independent modules.
