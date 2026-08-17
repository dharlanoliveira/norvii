# Feature Specification: Production Corpus Research Experience

**Feature Branch**: `002-production-web-experience`

**Created**: 2026-08-13

**Status**: Approved for implementation

**Input**: User description: "Implement the approved first-stage corpus research experience in the production React client, following the established architecture and quality guidance and using TDD. This stage does not depend on Neo4j."

## User Scenarios & Testing *(mandatory)*

### User Story 1 - Select a legal corpus (Priority: P1)

A researcher opens the production application, compares the Portuguese and English legal collections, and enters the collection relevant to the research question.

**Why this priority**: Corpus selection establishes the evidence boundary for every source and conversation that follows.

**Independent Test**: Open the production application, identify both corpus languages, open each corpus in turn, and observe a workspace containing only the selected corpus.

**Acceptance Scenarios**:

1. **Given** a fresh session, **When** the catalog opens, **Then** one Portuguese corpus and one English corpus are presented with name, language, jurisdiction, purpose, and source count.
2. **Given** both corpora are visible, **When** the researcher opens one, **Then** the application navigates to a dedicated workspace that identifies the active corpus and contains only its sources and conversation.
3. **Given** an unknown corpus address, **When** the workspace cannot resolve it, **Then** a localized recovery state explains the problem and provides a return to the catalog.

---

### User Story 2 - Browse and inspect sources (Priority: P1)

A researcher navigates an accessible source tree, distinguishes PDF documents from external links, and opens a source in the workspace without losing corpus context.

**Why this priority**: Direct source inspection makes the corpus boundary visible and prepares the product for later evidence-backed retrieval.

**Independent Test**: Enter either corpus, operate the source tree with pointer and keyboard, open a PDF and an external link, and observe the selected source and appropriate viewer state.

**Acceptance Scenarios**:

1. **Given** a corpus workspace, **When** the source library appears, **Then** the active corpus is the root of an expandable hierarchy with PDF and external-link groups.
2. **Given** an expanded group, **When** a PDF is selected, **Then** the right panel enters Source mode and presents its title, metadata, current location, readable prepared content, and document navigation controls.
3. **Given** an expanded group, **When** an external link is selected, **Then** Source mode presents its title, destination, available preview or unavailable state, and a safe action to open the original page in a new browser context.
4. **Given** one source is open, **When** another source is selected, **Then** the active selection and viewer update while the current conversation and draft remain intact.
5. **Given** a source cannot be previewed, **When** Source mode opens, **Then** its identity, unavailability reason, and safe alternative remain visible without blocking access to chat.

---

### User Story 3 - Research through chat and citations (Priority: P1)

A researcher alternates between source reading and a corpus-scoped chat, submits prepared questions, reads clearly simulated grounded answers or abstentions, and follows citations back to source passages without losing context.

**Why this priority**: The central experience connects document review to evidence-grounded conversation while later retrieval and model services remain out of scope.

**Independent Test**: Open a source, switch to chat, submit supported and unsupported prepared questions, open a citation, and switch repeatedly between Chat and Source while observing preserved source and conversation state.

**Acceptance Scenarios**:

1. **Given** a corpus workspace, **When** the researcher changes the right-panel mode, **Then** Chat and Source alternate without route changes or lost workspace state.
2. **Given** a source, conversation, and unsent draft exist, **When** the researcher switches modes repeatedly, **Then** the active source, document location, completed messages, draft, and meaningful reading positions remain available.
3. **Given** a supported prepared question, **When** the local demonstration response completes, **Then** it is labeled as simulated and every material claim cites an active-corpus source and stable location.
4. **Given** a citation in a completed response, **When** it is opened, **Then** Source mode selects the cited source and restores the cited location while preserving the conversation.
5. **Given** an unsupported prepared question, **When** the response completes, **Then** the application explicitly abstains and presents no unsupported legal conclusion.
6. **Given** a prepared failure scenario, **When** response generation fails, **Then** an actionable localized error appears and the question draft can be recovered or retried without duplicated completed messages.

---

### User Story 4 - Use a polished bilingual interface (Priority: P2)

A researcher uses the complete experience in English by default or changes the interface to Portuguese while legal content remains in its original language. The production client remains coherent, accessible, and responsive at the approved desktop sizes.

**Why this priority**: Bilingual accessibility and presentation quality are part of the product baseline and portfolio evidence.

**Independent Test**: Complete the catalog, source, chat, citation, and recovery journeys in both interface languages using pointer and keyboard at 1280 by 720 and 1440 by 900 pixels.

**Acceptance Scenarios**:

1. **Given** no stored interface preference, **When** the application opens, **Then** all product-authored interface content uses English.
2. **Given** active workspace state, **When** the interface changes to Portuguese, **Then** product-authored content changes language without resetting route, corpus, source, mode, location, conversation, or draft.
3. **Given** interface and corpus languages differ, **When** legal source content, a user question, or a prepared answer appears, **Then** that domain content is not automatically translated.
4. **Given** keyboard-only operation, **When** the researcher completes each primary journey, **Then** focus order, visible focus, tree behavior, mode selection, document controls, chat controls, and recovery actions remain usable.
5. **Given** either approved viewport, **When** the catalog or workspace appears, **Then** primary content and actions remain reachable without overlap or horizontal page scrolling.

### Edge Cases

- A corpus has no sources, only PDF sources, or only external links.
- A source group is collapsed while its selected descendant remains open.
- A corpus or source title becomes substantially longer after localization or fixture changes.
- An external destination rejects embedded preview, is unavailable, or requires leaving Norvii.
- A prepared PDF has many pages or no searchable text.
- A citation refers to an unavailable source or a location that cannot be restored.
- The interface language changes while focus is inside the tree, viewer, or composer.
- A valid or unknown workspace is opened directly or refreshed.
- Source mode is selected before a source is active.
- Reduced-motion and increased-contrast preferences are active.

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: The production home page MUST present exactly two production-owned demonstration corpora: one Portuguese corpus and one English corpus.
- **FR-002**: Every corpus item MUST expose name, language, jurisdiction, purpose, source count, and an accessible action to open it.
- **FR-003**: Opening a corpus MUST navigate to a dedicated workspace whose sources, conversation, and citations remain isolated to that corpus.
- **FR-004**: An unknown corpus route MUST present a localized recovery state with a catalog action.
- **FR-005**: The workspace MUST keep a source library on the left and one primary Chat or Source panel on the right at approved viewports.
- **FR-006**: The source library MUST expose an accessible hierarchical tree with the active corpus as root, expandable source-type groups, and selectable PDF or external-link leaves.
- **FR-007**: Source nodes MUST communicate type, title, selection, and availability without color as the only signal.
- **FR-008**: Selecting a source MUST switch the primary panel to Source mode and visibly associate the viewer with the selected tree node.
- **FR-009**: PDF Source mode MUST present production-owned prepared content, metadata, current location, and controls required to traverse that content.
- **FR-010**: External-link Source mode MUST present metadata, a safe preview or preview-unavailable state, and an explicit action that opens the fixed HTTPS destination in a new browser context.
- **FR-011**: The primary panel MUST provide an accessible Chat and Source mode selector and explain Source mode when no source is selected.
- **FR-012**: Mode changes and source changes MUST preserve active source, source location, conversation history, unsent draft, and meaningful reading positions.
- **FR-013**: Chat MUST expose empty, composing, responding, completed, abstained, and failed demonstration states through structured message parts.
- **FR-014**: Every material prepared legal claim MUST cite a source belonging to the active corpus at a stable passage, article, section, or page location.
- **FR-015**: Opening a citation MUST select its source and restore its stable location when available without discarding conversation state.
- **FR-016**: Unsupported prepared questions MUST produce explicit abstention instead of an unsupported legal conclusion.
- **FR-017**: Prepared response failures MUST preserve recoverable user input and MUST NOT duplicate completed messages when retried.
- **FR-018**: All product-authored visible and assistive text MUST reference structurally complete English and Portuguese localization resources, with English as the default.
- **FR-019**: Changing interface language MUST preserve current route and workspace state and MUST NOT translate corpus titles, legal content, citations, user questions, or prepared answers.
- **FR-020**: The interface MUST meet WCAG 2.2 AA expectations for semantic structure, contrast, keyboard access, visible focus, accessible names, tree behavior, and status announcements across primary journeys.
- **FR-021**: The accepted editorial research-desk identity, hierarchy, and Chat and Source relationship from Feature 001 MUST remain recognizable, with intentional production differences documented in this feature.
- **FR-022**: The catalog and workspace MUST remain usable at 1280 by 720 and 1440 by 900 pixels without overlap, clipped primary actions, or horizontal page scrolling.
- **FR-023**: The production client MUST use production-owned code, styles, fixtures, and assets and MUST NOT import any module or file from `prototypes/web/`.
- **FR-024**: This feature MUST run without the Go API, Python ingestion, PostgreSQL, Neo4j, live model calls, remote corpus fetches, or persistence.
- **FR-025**: The client MUST expose its demonstration corpus data and response behavior through narrow replaceable boundaries so later versioned API adapters do not require presentation components to change their domain model.
- **FR-026**: The application MUST identify simulated answers and state that Norvii is a technical demonstration, not legal advice.
- **FR-027**: Initial catalog content MUST become interactive within 2 seconds on the reference development environment, and the initial production JavaScript entry chunk MUST remain below 350 kB compressed.
- **FR-028**: Automated tests MUST demonstrate every primary acceptance scenario through public user-visible behavior and include locale parity, keyboard interaction, accessibility scanning, and production build verification.

### Key Entities

- **Interface Language**: The selected language for product-authored interface content; English is the default and Portuguese is the alternative.
- **Corpus**: An isolated legal collection with a stable identifier, language, jurisdiction, purpose, and production-owned demonstration sources.
- **Source Group**: A hierarchical source-tree node that groups sources by type without owning legal content.
- **Source**: A PDF document or fixed external URL belonging to exactly one corpus, with type, title, metadata, availability, and a stable viewing location.
- **Workspace State**: The active corpus, selected source, panel mode, source location, meaningful reading positions, conversation, and draft for one browser session.
- **Conversation**: A corpus-scoped sequence of user and simulated assistant messages represented as structured parts.
- **Citation**: A structured reference from one prepared claim to an active-corpus source and stable location.

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: All catalog, source navigation, mode switching, citation, abstention, failure recovery, and unknown-route acceptance scenarios pass automated user-visible tests.
- **SC-002**: Every primary journey can be completed with keyboard only and automated accessibility scans report no detected violations in catalog, workspace, and recovery states.
- **SC-003**: English and Portuguese resources contain identical key structures, and English is active in every fresh-session test.
- **SC-004**: Every prepared material claim opens the correct source and stable location, and every unsupported prepared question abstains in automated tests.
- **SC-005**: Workspace state remains unchanged in 100% of automated mode-switch and language-switch preservation tests except for the explicitly changed mode or language.
- **SC-006**: The production build completes without warnings treated as quality failures, and its initial compressed JavaScript entry chunk remains below 350 kB.
- **SC-007**: Visual regression evidence at 1280 by 720 and 1440 by 900 shows no unresolved high-severity difference from the approved hierarchy and interaction baseline.
- **SC-008**: The complete feature remains demonstrable locally with zero database services, backend services, remote corpus requests, or model usage cost.

## Assumptions

- The stakeholder approval recorded in Feature 001 authorizes production implementation of the accepted journeys and visual direction.
- The production client starts with production-owned deterministic data behind a replaceable data boundary; live HTTP and streaming contracts are deferred to the feature that introduces the Go API.
- The feature is session-only and does not promise refresh persistence for conversation or interface preference.
- Fixed external-source destinations are HTTPS links to official sources. Tests do not make network requests.
- Current evergreen desktop browsers are the supported runtime for this increment. Narrow mobile layouts are deferred.
- Performance measurements use a clean production build and the reference local validation commands defined during planning.

## Norvii Feature Requirements *(mandatory)*

### Scope and Boundaries

- **In scope**: A production React client under `apps/web/` that independently reimplements the approved corpus catalog and research workspace, production-owned deterministic corpus data, source viewing, structured simulated chat, citation navigation, abstention and failure states, bilingual resources, accessibility, responsive desktop behavior, automated tests, and a module CI contract.
- **Out of scope**: Go API implementation, Python ingestion, corpus or source registration, file upload, persistence, database containers, PostgreSQL, Neo4j, real retrieval, live model calls, GraphRAG inspection, MCP execution, skills execution, authentication, mobile layouts, and automatic translation of legal content.

### Prototype Baseline *(mandatory for production UI features)*

- **Approved baseline**: [Feature 001 verified prototype](../001-product-experience-prototype/review.md), including its recorded catalog and workspace screenshots at 1280 by 720 and 1440 by 900 pixels.
- **Intentional differences**: Production code, fixtures, styles, and tests are independently owned under `apps/web/`. Route-level loading and a 350 kB compressed initial-entry budget address the prototype bundle advisory. Architectural seams anticipate later HTTP and streaming adapters without adding them now.

### Evidence and Corpus Boundaries *(mandatory for retrieval, chat, or citations)*

- **Active corpus constraint**: The production-owned demonstration provider requires one corpus identifier for every source, conversation, response, and citation lookup. Cross-corpus citations are rejected before reaching presentation components.
- **Evidence behavior**: Supported prepared responses contain structured citations to stable demonstration locations, unsupported questions abstain, failure scenarios remain recoverable, and every response is visibly labeled as simulated and non-authoritative.
