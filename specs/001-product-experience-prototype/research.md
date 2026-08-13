# Research: Corpus Research Workspace Prototype

## Visual direction

**Decision**: Use an editorial research-desk direction: warm paper surfaces, deep ink text, restrained burgundy and teal accents, serif display typography paired with a highly legible sans-serif interface face, generous whitespace, and precise document-oriented details.

**Rationale**: The direction feels credible for legal research without copying institutional court software or generic AI dashboards. It supports dense source navigation and long-form reading while keeping chat approachable. The visual hierarchy can distinguish corpus context, navigation, reading, and conversation without excessive cards or decorative gradients.

**Alternatives considered**:

- A conventional SaaS dashboard was rejected because it would make the portfolio prototype indistinguishable from common admin templates.
- A dark technical console was rejected because it prioritizes engineering inspection over legal reading and reduces the calm editorial tone.
- A library-archive metaphor with heavy ornament was rejected because it risks nostalgia, weak contrast, and visual interference with primary actions.

## Workspace composition

**Decision**: Use a stable two-column shell at reference desktop sizes. The left rail owns corpus identity and the accessible source tree. The right work surface owns a compact Chat/Source segmented control and renders only one primary mode at a time.

**Rationale**: One primary content mode prevents document reading and conversation from competing for width. The persistent tree keeps corpus boundaries and source selection visible. Explicit mode switching also gives a clear state model for preserving chat and viewer context.

**Alternatives considered**:

- Three simultaneous columns were rejected because source tree, viewer, and chat become too narrow at 1280 pixels.
- A modal document viewer was rejected because it obscures workspace context and complicates citation navigation.
- Independent browser tabs were rejected because they make conversation and source state harder to relate and validate.

## Component strategy

**Decision**: Use feature-owned React components and CSS custom properties, with Lucide icons and assistant-ui primitives at the chat boundary. Avoid a broad component framework in the prototype.

**Rationale**: Custom composition preserves a distinctive design and keeps the prototype small. Feature ownership prevents an exploratory design system from becoming a premature production abstraction. assistant-ui provides a realistic chat interaction surface without defining backend behavior.

**Alternatives considered**:

- A comprehensive UI kit was rejected because its visual defaults would dominate the prototype and add unused dependencies.
- Fully custom chat behavior was rejected because established accessible chat primitives reduce interaction risk.

## State and fixture architecture

**Decision**: Use typed, immutable fixture records and one workspace controller hook that owns active corpus, source, mode, viewer location, chat draft, messages, and interface language. Derived selectors enforce corpus isolation.

**Rationale**: The state is interdependent and must survive mode switching. A cohesive controller makes invariants explicit while keeping rendering components focused. Local fixtures provide repeatable demonstrations and tests with zero network cost.

**Alternatives considered**:

- A global state library was rejected because the prototype has one bounded workspace and no demonstrated cross-feature synchronization need.
- Component-local state scattered across the tree was rejected because mode and citation transitions must update multiple views atomically.

## Internationalization

**Decision**: Use i18next with separate English and Portuguese resource modules, English as the fallback and default, stable English keys, and automated resource-parity tests.

**Rationale**: The library supports React integration, interpolation, pluralization, and accessible text while keeping localization data separate from legal fixtures. Resource parity directly enforces the feature acceptance criteria.

**Alternatives considered**:

- A custom translation context was rejected because it would recreate fallback, interpolation, and missing-key behavior.
- Browser-language auto-selection was rejected because the specification requires English for every fresh session.

## Verification approach

**Decision**: Combine typed unit and component tests with accessibility assertions, Playwright journey tests, and reference screenshots at 1280 by 720 and 1440 by 900.

**Rationale**: Component tests efficiently protect state and semantics, while browser tests validate routing, focus, responsive layout, mode preservation, and the actual visual baseline. Deterministic fixtures make screenshots and timing repeatable.

**Alternatives considered**:

- Screenshot-only validation was rejected because it cannot protect keyboard behavior or preserved state.
- Unit-only validation was rejected because layout and browser interaction are the primary purpose of this prototype.

## Dependency versions

**Decision**: Pin the mutually compatible versions recorded in `plan.md` and commit the npm lockfile. Use Node.js 24 for local and CI execution. TypeScript 6 is selected because the current type-aware lint toolchain does not yet support TypeScript 7.

**Rationale**: The versions were resolved from the npm registry on 2026-08-13. Exact dependency resolution and a current LTS runtime keep the prototype reproducible.

**Alternatives considered**:

- TypeScript 7 was rejected for this feature because the current type-aware lint parser declares support only through TypeScript 6.
- Floating ranges without a lockfile were rejected because they make visual and test behavior irreproducible.
- Adding backend or monorepo orchestration dependencies was rejected because only one prototype module is being built.
