# UI State Contract: Corpus Research Workspace Prototype

This contract defines observable prototype behavior. It is not a production API or a cross-language schema.

## Routes

| Route | Observable state |
| --- | --- |
| `/` | Two-corpus catalog with English as the fresh-session interface language |
| `/corpora/:corpusId` | Active-corpus workspace with source tree and right-panel mode selector |
| Unknown corpus identifier | Recovery state with a return-to-catalog action |

## Catalog contract

Each corpus option exposes its title, language, jurisdiction, purpose, source count, and open action. Activating the option navigates to that corpus without transferring another corpus's source or conversation state.

## Source-tree contract

- The tree exposes the active corpus as its root.
- PDF and external-link groups can expand and collapse independently.
- Arrow keys move through visible nodes and expand or collapse groups according to the tree interaction pattern.
- Enter or Space selects a source leaf.
- Selection is communicated semantically and visually without color-only meaning.
- Collapsing the parent of an active source does not clear the viewer.

## Right-panel mode contract

- The mode selector exposes exactly Chat and Source.
- Selecting a source activates Source mode.
- Switching modes preserves the active source, source location, chat messages, draft, and meaningful reading positions.
- Source mode without a selection explains how to open a source.
- The active mode is conveyed semantically and visually.

## Citation contract

Activating a citation selects only a source owned by the active corpus, switches to Source mode, and targets the referenced location. Failure to resolve a location retains conversation state and presents an actionable viewer message.

## Localization contract

- Every fresh session starts in English.
- English and Portuguese resource trees have identical keys.
- Product-authored visible and assistive text uses resource keys.
- Interface language changes do not alter corpus titles, legal source text, user questions, prepared answers, route, source, viewer location, or conversation.

## Prototype boundary contract

The application performs no ingestion, upload, remote content fetch, persistence, backend request, or live model call. All responses, sources, delays, failures, and citations originate from deterministic local fixtures and are visibly identified as simulated when confusion is possible.
