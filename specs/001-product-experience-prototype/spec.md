# Feature Specification: Corpus Research Workspace Prototype

**Feature Branch**: `001-product-experience-prototype`

**Created**: 2026-08-12

**Status**: Verified

**Input**: User description: "Create the first product prototype with a home page listing the Portuguese and English legal corpora and their languages. Opening a corpus leads to a detail workspace with an expandable source tree on the left and chat on the right. Selecting a PDF or external link opens it in the right panel, where the user can switch between the source viewer and chat without losing context. Data ingestion is excluded. Deliver a first-class layout and user experience."

## User Scenarios & Testing *(mandatory)*

### User Story 1 - Choose a legal corpus (Priority: P1)

A reviewer arrives at a focused home page, understands that Norvii offers two isolated legal collections, identifies the language of each collection, and opens the collection relevant to the intended research task.

**Why this priority**: Corpus selection establishes the evidence boundary for every source and conversation that follows.

**Independent Test**: Open the home page, compare the two available corpora, identify which is Portuguese and which is English, select each one in turn, and verify that the correct corpus workspace opens.

**Acceptance Scenarios**:

1. **Given** a reviewer opens Norvii, **When** the home page is displayed, **Then** exactly one Portuguese corpus and one English corpus are presented as distinct selectable collections.
2. **Given** the two corpora are visible, **When** the reviewer compares them, **Then** each item clearly communicates its name, corpus language, jurisdiction, short purpose, and source count.
3. **Given** the reviewer selects a corpus, **When** navigation completes, **Then** the detail workspace clearly identifies that corpus as active and contains only its sources and conversation.
4. **Given** a reviewer navigates directly to an unknown corpus, **When** the workspace cannot resolve it, **Then** an informative recovery state offers a return to the corpus catalog.

---

### User Story 2 - Browse and open corpus sources (Priority: P1)

A reviewer explores the active corpus through a source tree on the left, distinguishes PDF documents from external links, expands or collapses groups, and opens a source without leaving the research workspace.

**Why this priority**: Direct access to source material makes the corpus boundary tangible and lets reviewers verify the evidence behind the product experience.

**Independent Test**: Open a corpus, navigate the complete source tree with pointer and keyboard, select a PDF and an external link, and verify that each opens in the source viewing mode while the active source remains apparent.

**Acceptance Scenarios**:

1. **Given** a corpus workspace is open, **When** the source tree appears, **Then** it presents the active corpus as its root and organizes PDF documents and external links into understandable expandable groups.
2. **Given** a source group is expanded, **When** the reviewer selects a PDF, **Then** the right panel switches to source mode and presents the document with its title, source metadata, current location, and document navigation controls.
3. **Given** a source group is expanded, **When** the reviewer selects an external link, **Then** the right panel switches to source mode and presents its title, destination, available preview, and an explicit action to open the original page.
4. **Given** one source is already open, **When** the reviewer selects another source, **Then** the viewer replaces the active source, the tree selection updates, and the chat state remains intact.
5. **Given** the selected source cannot be displayed, **When** source mode opens, **Then** the reviewer sees the source identity, the reason it is unavailable, and any safe alternative action without losing access to chat.

---

### User Story 3 - Alternate between source review and chat (Priority: P1)

A reviewer moves between the active source and corpus-scoped chat in the right panel, asks prepared questions about the corpus, reads grounded responses, and returns to source material without losing either context.

**Why this priority**: The central product experience is the close relationship between source reading and evidence-grounded conversation.

**Independent Test**: Open a source, switch to chat, enter a prepared question, inspect the deterministic response and citation, return to the source, and switch back to chat while verifying that source position, messages, draft text, and relevant scroll positions are preserved.

**Acceptance Scenarios**:

1. **Given** a workspace is open, **When** the reviewer uses the right-panel mode selector, **Then** the reviewer can switch between Chat and Source without leaving the workspace.
2. **Given** a source is open and chat already contains messages, **When** the reviewer switches between modes repeatedly, **Then** the active source, document location, conversation history, unsent draft, and meaningful reading positions remain available.
3. **Given** the reviewer submits a prepared question, **When** deterministic evidence is available in the active corpus, **Then** chat presents a clearly staged response with citations that identify the supporting source and stable passage location.
4. **Given** the reviewer opens a citation from chat, **When** its source is available, **Then** source mode opens the cited source at the relevant passage and preserves the conversation.
5. **Given** the prepared corpus does not support a question, **When** the response completes, **Then** chat explicitly abstains and does not present an unsupported legal conclusion.
6. **Given** a response is being presented, **When** the reviewer changes the active source or right-panel mode, **Then** the response state remains understandable and no input or completed content is lost.

---

### User Story 4 - Use a polished bilingual experience (Priority: P2)

A reviewer uses the complete experience in English by default or changes the product interface to Portuguese while corpus content remains in its original language. The interface feels coherent, calm, and purpose-built at desktop and notebook sizes.

**Why this priority**: Bilingual operation and presentation quality are portfolio evidence and must apply consistently across the primary journeys rather than being added after screen design.

**Independent Test**: Complete corpus selection, source navigation, source viewing, chat, citation navigation, and recovery flows in both interface languages using keyboard and pointer at the approved viewport sizes.

**Acceptance Scenarios**:

1. **Given** no interface preference exists, **When** Norvii opens, **Then** all product-authored interface content is in English.
2. **Given** a workspace has an active corpus, source, and conversation, **When** the reviewer changes the interface to Portuguese, **Then** product-authored interface content changes language without resetting navigation, source, viewer, or chat state.
3. **Given** the interface language differs from the corpus language, **When** a source or answer is displayed, **Then** legal content remains in its source or answer language while controls and guidance use the selected interface language.
4. **Given** a reviewer uses only a keyboard, **When** the reviewer completes each primary journey, **Then** focus order, visible focus, tree navigation, mode switching, viewer controls, and chat operation are predictable and complete.
5. **Given** the workspace is viewed at an approved notebook or wide-desktop size, **When** the reviewer opens sources and chat, **Then** hierarchy, controls, reading area, and input remain usable without overlap, clipped primary actions, or horizontal page scrolling.

### Edge Cases

- A corpus has no sources, only PDF sources, or only external links.
- A source group is collapsed while its selected descendant is open in the viewer.
- A source title or corpus name is substantially longer in one supported language.
- An external destination rejects embedded preview, is temporarily unavailable, or requires leaving Norvii.
- A PDF cannot render, has many pages, or does not expose searchable text.
- A user opens a citation whose source is unavailable or whose saved passage location cannot be restored.
- A user switches interface language while focus is inside the source tree, document viewer, or chat composer.
- A user refreshes or directly opens a valid corpus workspace without visiting the home page first.
- The right panel has no active source and the reviewer selects Source mode.
- The viewport becomes narrower while a source is open or while keyboard focus is inside a panel that must collapse.
- Reduced-motion or high-contrast preferences are active.

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: The home page MUST present exactly two deterministic corpus examples for this prototype: one Portuguese corpus and one English corpus.
- **FR-002**: Every corpus item MUST present its name, language, jurisdiction, short purpose, source count, and a clear action to open it.
- **FR-003**: The product MUST use `Corpus` or an equivalent unambiguous legal-collection label and MUST NOT describe the collection as a legal or technical process.
- **FR-004**: Opening a corpus MUST navigate to a dedicated workspace that keeps the active corpus identity and language visible.
- **FR-005**: Sources and conversations MUST remain isolated to the active corpus throughout every prototype journey.
- **FR-006**: The workspace MUST provide a persistent source area on the left and a primary content area on the right at approved desktop and notebook sizes.
- **FR-007**: The source area MUST use an accessible hierarchical tree with the active corpus as root, expandable source-type groups, and selectable PDF or external-link leaves.
- **FR-008**: Every source node MUST communicate its source type, title, selection state, and availability without relying on color alone.
- **FR-009**: Selecting any available source MUST open it in the right panel and MUST visibly associate the viewer with the selected tree node.
- **FR-010**: PDF source mode MUST provide readable document content, source metadata, current location, and the navigation controls required by the prepared document.
- **FR-011**: External-link source mode MUST provide source metadata, the available preview or a clear preview-unavailable state, and an explicit action to open the original destination.
- **FR-012**: The right panel MUST provide an accessible mode selector for Chat and Source, indicate the current mode, and explain Source mode when no source is selected.
- **FR-013**: Switching right-panel mode MUST preserve the active source, source location, conversation history, unsent chat draft, and meaningful scroll positions.
- **FR-014**: Selecting a different source MUST update source mode without clearing or restarting the active corpus conversation.
- **FR-015**: Chat MUST demonstrate empty, composing, responding, completed, abstained, and failed states using deterministic prototype behavior.
- **FR-016**: Every material legal claim in a prepared answer MUST cite a source from the active corpus at a stable passage, page, article, or section location.
- **FR-017**: Opening a citation MUST select the corresponding source and navigate the viewer to the cited location when available.
- **FR-018**: Prepared unsupported questions MUST produce explicit abstention rather than an unsupported legal conclusion.
- **FR-019**: The complete product interface MUST support English and Portuguese, with English active by default.
- **FR-020**: Changing interface language MUST preserve the current route, active corpus, selected source, viewer mode and location, conversation, draft, and open citation.
- **FR-021**: Product-authored navigation, labels, actions, states, errors, viewer controls, chat content labels, and accessibility text MUST be complete in both interface languages.
- **FR-022**: Corpus titles, source content, citations, user questions, and prepared answers MUST remain separate from interface localization and MUST NOT be automatically translated when interface language changes.
- **FR-023**: The prototype MUST define and consistently apply a coherent visual system for typography, spacing, color, iconography, elevation, focus, selection, and state feedback across the catalog and workspace.
- **FR-024**: Visual hierarchy MUST keep corpus identity, active source, current right-panel mode, primary action, and recovery action distinguishable at a glance.
- **FR-025**: Interactions MUST provide clear hover, focus, active, selected, disabled, loading, empty, and error feedback where those states apply.
- **FR-026**: The experience MUST meet WCAG 2.2 AA expectations for contrast, keyboard access, focus visibility, semantic structure, accessible names, and status announcements across the primary journeys.
- **FR-027**: Motion MUST support comprehension, respect reduced-motion preferences, and MUST NOT be required to understand state or navigation.
- **FR-028**: The primary journeys MUST remain usable at minimum viewport sizes of 1280 by 720 pixels and at a wide-desktop reference size of 1440 by 900 pixels.
- **FR-029**: The prototype MUST operate without backend services, live model calls, network-dependent corpus data, persistence, or data ingestion.
- **FR-030**: The product MUST clearly identify simulated answers and state that Norvii is a technical demonstration, not legal advice.
- **FR-031**: The approved prototype baseline MUST include reviewable evidence for every primary journey, supported interface language, required interaction state, and approved viewport size.

### Key Entities

- **Interface Language**: The selected language for product-authored interface content and locale-sensitive presentation; English is the default and Portuguese is the alternative.
- **Corpus**: An isolated legal collection with a stable identity, name, language, jurisdiction, purpose, source count, and deterministic prototype sources.
- **Source Group**: A hierarchical tree node that organizes sources by type within one corpus and owns no source content itself.
- **Source**: A PDF document or external URL that belongs to exactly one corpus and exposes its type, title, metadata, availability, and viewing location.
- **Viewer State**: The selected source, viewing mode, stable passage or page location, and meaningful reading position preserved while the reviewer uses chat.
- **Conversation**: The corpus-scoped sequence of questions, deterministic response states, answers, citations, and unsent draft.
- **Citation**: A reference from a material prepared claim to one source in the active corpus and a stable passage, page, article, or section location.

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: At least 90% of first-time reviewers can identify the Portuguese and English corpora and open the requested one in under 20 seconds without instruction.
- **SC-002**: At least 90% of reviewers can locate and open a requested PDF or external link from the source tree in under 30 seconds without instruction.
- **SC-003**: At least 90% of reviewers can switch from an open source to chat and back, then correctly identify the active source and conversation state without assistance.
- **SC-004**: In 100% of required mode-switching tests, the active source, source location, completed messages, and unsent draft are preserved.
- **SC-005**: Every prepared material legal claim exposes a citation that opens the correct active-corpus source and location, while every unsupported prepared question produces explicit abstention.
- **SC-006**: All primary journeys can be completed using keyboard only, with no critical or serious accessibility findings in the approved baseline review.
- **SC-007**: All primary pages, controls, states, and accessibility labels are complete in English and Portuguese, with English active in 100% of fresh sessions.
- **SC-008**: The catalog and workspace pass the project visual-review rubric with no unresolved high-severity findings in hierarchy, readability, consistency, interaction feedback, or responsive behavior.
- **SC-009**: All primary content and actions remain reachable without overlap or horizontal page scrolling at 1280 by 720 and 1440 by 900 pixels.
- **SC-010**: Every required journey remains fully demonstrable without ingestion, backend services, live model calls, network-dependent corpus data, or model usage cost.

## Assumptions

- `Corpus` is the product and domain term for each selectable legal collection. It replaces the ambiguous term `Process`, which can imply a court proceeding or background operation.
- The prototype contains exactly two preloaded deterministic corpora and a small set of representative PDF and external-link fixtures.
- The source tree uses the active corpus as root and source type as the minimum grouping hierarchy; deeper legal structure may be explored during prototype design when supported by fixture content.
- Selecting a source switches the right panel to Source mode. Reviewers can explicitly return to Chat through the same mode selector.
- Opening an external destination may leave the prototype or open a new browser context; the interface communicates that behavior before activation.
- Chat responses are deterministic product examples and do not validate retrieval quality, model quality, or legal correctness.
- Interface language is independent from corpus, source, question, and answer language.
- Engineering artifacts remain in English. Portuguese is limited to localized interface values and preserved legal fixture content; localization keys and surrounding code remain in English.
- Authentication, user accounts, durable preferences, and user-specific conversation history are not required for the prototype.
- The visual-review rubric and final responsive layout direction will be defined during planning and approved through the prototype review loop.

## Norvii Feature Requirements *(mandatory)*

### Scope and Boundaries

- **In scope**: An executable bilingual prototype covering the two-corpus catalog, corpus navigation, corpus detail workspace, hierarchical source tree, PDF and external-link viewing, Chat and Source mode switching, deterministic grounded chat, citation navigation, abstention, accessibility, responsive behavior, polished visual presentation, and review evidence.
- **Out of scope**: Corpus registration or editing, source upload or registration, data ingestion, source processing states, persistence, production application modules, backend services, databases, authentication, real retrieval, live model calls, legal correctness validation, technical retrieval inspection, GraphRAG visualization, MCP execution, skills execution, additional interface languages, and automatic translation of legal content.

### Prototype Baseline *(mandatory for production UI features)*

- **Approved baseline**: This feature establishes the initial executable product baseline; no earlier UI baseline exists. Approval requires the feature to reach `Verified` through the documented prototype review gate.
- **Intentional differences**: None at specification time. Approved journeys, visual evidence, interaction rules, and documented product decisions become the baseline for later production features.

### Evidence and Corpus Boundaries *(mandatory for retrieval, chat, or citations)*

- **Active corpus constraint**: Every prepared source, question, answer, and citation belongs to exactly one active deterministic corpus. Changing or directly opening a corpus never carries sources, citations, or conversation content from another corpus into the workspace.
- **Evidence behavior**: Prepared answers cite stable fixture locations for material claims, citations open the corresponding source location, and unsupported questions visibly abstain. All simulated answers remain labeled as prototype behavior and non-authoritative legal content.
