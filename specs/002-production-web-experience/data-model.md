# Data Model: Production Corpus Research Experience

## Corpus

- `id`: stable route-safe identifier.
- `language`: `en` or `pt`; describes legal content, not interface language.
- `name`: preserved domain content.
- `jurisdiction`: preserved domain content.
- `summary`: preserved domain content.
- `sources`: ordered source collection owned exclusively by this corpus.
- `suggestedQuestions`: preserved domain content for prepared demonstration paths.

Invariants:

- exactly two demonstration corpora exist in this feature;
- corpus identifiers are unique;
- every source identifier is unique within its corpus;
- all conversation, response, and citation lookup starts with an active corpus identifier.

## Source

Shared fields:

- `id`, `corpusId`, `title`, and `kind`;
- authority, official reference, and availability;
- stable ordered locations.

PDF-specific fields:

- page count, current page, and prepared page content.

External-link-specific fields:

- fixed HTTPS destination, captured preview text when available, and preview status.

Invariants:

- `kind` is a discriminant and only matching fields are present;
- a source belongs to exactly one corpus;
- external destinations use HTTPS;
- unavailable previews retain source identity and safe original-link access.

## Source location

- stable identifier unique within a source;
- human-readable label such as page, article, or section;
- prepared source text or position required by the viewer.

## Prepared response

- prompt matchers scoped to one corpus;
- outcome: `answered`, `abstained`, or `failed`;
- ordered structured parts containing text and citations;
- simulation label and no-legal-advice status supplied by interface localization.

Invariants:

- every cited source belongs to the response corpus;
- every citation location exists in its source;
- abstained responses contain no material legal conclusion;
- failed responses do not become completed assistant messages.

## Citation

- stable identifier;
- source identifier;
- source-location identifier;
- preserved domain label.

## Workspace state

- active corpus identifier from the route;
- selected source identifier or no selection;
- mode: `chat` or `source`;
- current location per visited source;
- meaningful scroll positions per mode;
- assistant thread and unsent draft owned by the mounted chat runtime.

Transitions:

1. Opening a valid corpus initializes Chat mode with no selected source.
2. Selecting a source validates corpus ownership and enters Source mode.
3. Selecting Chat or Source changes only mode; Source without a selection exposes guidance.
4. Opening a citation validates source and location, selects both, and enters Source mode.
5. Changing interface language changes only localized presentation.
6. Leaving the mounted workspace may discard session-only state.
