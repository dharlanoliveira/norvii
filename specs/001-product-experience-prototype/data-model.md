# Data Model: Corpus Research Workspace Prototype

All records are deterministic prototype data. They define client behavior and do not establish production persistence or cross-language schemas.

## InterfaceLanguage

| Field | Type | Rules |
| --- | --- | --- |
| `code` | `en` or `pt` | Defaults to `en` for every fresh session |
| `labelKey` | Localization key | Exists in both resource sets |

Changing interface language preserves workspace state and never mutates legal content.

## Corpus

| Field | Type | Rules |
| --- | --- | --- |
| `id` | Stable string | Unique across fixtures and route-safe |
| `title` | Localized or proper title | Displayed without automatic legal-content translation |
| `description` | Localized interface description | Available in both interface languages |
| `language` | `en` or `pt` | Describes corpus content, not interface state |
| `jurisdiction` | String | Visible in catalog and workspace |
| `sourceIds` | Ordered source identifiers | Every source must belong to this corpus |

A corpus owns its sources and conversation fixtures. Foreign source identifiers are invalid.

## SourceGroup

| Field | Type | Rules |
| --- | --- | --- |
| `id` | Stable string | Unique within a corpus |
| `labelKey` | Localization key | Describes PDF documents or external links |
| `kind` | `pdf` or `external` | Matches every child source kind |
| `sourceIds` | Ordered source identifiers | Contains only active-corpus sources |

Groups can be expanded or collapsed. Collapsing a group does not close its active source.

## Source

| Field | Type | Rules |
| --- | --- | --- |
| `id` | Stable string | Unique across fixtures |
| `corpusId` | Corpus identifier | Immutable ownership boundary |
| `kind` | `pdf` or `external` | Determines viewer presentation |
| `title` | Preserved source title | Never routed through interface translation |
| `publisher` | String | Preserved source metadata |
| `location` | Page, article, section, or URL | Stable target for citations |
| `status` | `available` or `unavailable` | Unavailable sources retain identity and recovery guidance |
| `content` | Fixture sections or preview | Small, deterministic, and non-authoritative |
| `externalUrl` | Optional HTTPS URL | Present only for external sources |

## ViewerState

| Field | Type | Rules |
| --- | --- | --- |
| `sourceId` | Source identifier or none | Must belong to the active corpus |
| `location` | Stable location or none | Must exist in the selected fixture when present |
| `scrollAnchor` | Local presentation state | Preserved while switching to chat |

Selecting a source enters Source mode. Opening a citation selects its source and target location atomically.

## Conversation

| Field | Type | Rules |
| --- | --- | --- |
| `corpusId` | Corpus identifier | One isolated conversation per active corpus |
| `messages` | Ordered messages | Deterministic and preserved during mode switching |
| `draft` | String | Preserved until submitted or cleared |
| `responseState` | `idle`, `responding`, `complete`, `abstained`, or `failed` | Drives accessible status feedback |

## Message

| Field | Type | Rules |
| --- | --- | --- |
| `id` | Stable string | Unique in the conversation |
| `role` | `user` or `assistant` | Determines presentation and accessible label |
| `content` | String | User input or deterministic fixture answer |
| `citations` | Ordered citation identifiers | Assistant messages only |

## Citation

| Field | Type | Rules |
| --- | --- | --- |
| `id` | Stable string | Unique in fixture data |
| `sourceId` | Source identifier | Must belong to the conversation corpus |
| `location` | Stable source location | Resolves in the source fixture |
| `label` | Preserved legal reference | Identifies page, article, or section |

## WorkspaceState transitions

| Event | Preconditions | Result |
| --- | --- | --- |
| Open corpus | Corpus exists | Corpus becomes active; its isolated workspace state loads |
| Select source | Source belongs to active corpus | Source and location become active; mode becomes Source |
| Select unavailable source | Source belongs to active corpus | Source mode shows identity and recovery state |
| Switch mode | Workspace is open | Mode changes; viewer and conversation state remain unchanged |
| Submit prepared question | Draft matches a supported fixture | User message and deterministic cited answer are appended |
| Submit unsupported question | No supported fixture matches | User message and explicit abstention are appended |
| Open citation | Citation and source belong to active corpus | Source and location become active; mode becomes Source |
| Change interface language | Supported language selected | Interface resources change; workspace and legal content remain unchanged |
