# Contract: Corpus Ingestion HTTP API v1

The implementation promotes this design to `contracts/corpus-ingestion/v1/openapi.yaml`. All JSON uses UTF-8, UUID strings, RFC 3339 UTC timestamps, lowercase enum values, and SHA-256 lowercase hexadecimal hashes. Routes are rooted at `/api/v1`.

## Error envelope

```json
{"error":{"code":"invalid_input","message":"Localized-safe key or English fallback","fields":{"name":"required"},"requestId":"uuid"}}
```

Codes: `invalid_input`, `payload_too_large`, `unsafe_url`, `unsupported_content`, `duplicate_source`, `stale_state`, `not_found`, `unavailable`, `acquisition_failed`, `extraction_failed`, `publication_failed`, and `internal_error`.

## Corpus operations

| Method and path | Input | Success | Errors |
| --- | --- | --- | --- |
| `GET /corpora?includeDisabled=false` | Query flag | `200` corpus summaries | `internal_error` |
| `POST /corpora` | `CorpusWrite` | `201` corpus | validation, conflict |
| `GET /corpora/{corpusId}` | Corpus UUID | `200` corpus | not found, unavailable |
| `PATCH /corpora/{corpusId}` | Mutable fields plus `version` | `200` corpus | validation, stale, not found |
| `POST /corpora/{corpusId}/disable` | `version` | `200` corpus | stale, not found |
| `POST /corpora/{corpusId}/enable` | `version` | `200` corpus | stale, not found |

`CorpusWrite` requires `name`, `description`, `language` (`en` or `pt`), and `jurisdiction`. A corpus response adds `id`, `status` (`enabled` or `disabled`), `sourceCount`, `version`, `createdAt`, and `updatedAt`.

## Source operations

| Method and path | Input | Success | Errors |
| --- | --- | --- | --- |
| `GET /corpora/{corpusId}/sources` | Corpus UUID | `200` ordered source summaries | not found, unavailable |
| `POST /corpora/{corpusId}/sources/url` | Title and HTTPS URL | `202` source | validation, unsafe, duplicate, limit |
| `POST /corpora/{corpusId}/sources/pdf` | Multipart title and PDF, max 10 MB | `202` source | validation, unsupported, duplicate, limit |
| `GET /corpora/{corpusId}/sources/{sourceId}` | Scoped UUIDs | `200` source detail | not found |
| `POST /corpora/{corpusId}/sources/{sourceId}/retry` | Source `version` | `202` source | stale, invalid state, not found |
| `POST /corpora/{corpusId}/sources/{sourceId}/reprocess` | Source `version` | `202` source | stale, invalid state, not found |

A source summary contains identity, title, `kind` (`url` or `pdf`), `processingStatus` (`pending`, `processing`, `ready`, or `failed`), safe failure category, version, timestamps, latest attempt, complete newest-first safe attempt history, and optional latest ready document identity. Origin detail is discriminated by kind and never includes PDF bytes.

## Document and origin operations

| Method and path | Success | Notes |
| --- | --- | --- |
| `GET /corpora/{corpusId}/sources/{sourceId}/document` | `200` latest ready document and units | `404` when none is ready; may coexist with failed latest attempt |
| `GET /corpora/{corpusId}/sources/{sourceId}/origin/pdf` | `200 application/pdf` | Content-Disposition uses sanitized recorded filename |
| `GET /corpora/{corpusId}/sources/{sourceId}/origin/url` | `200` submitted/final URL metadata | Client opens only returned HTTPS final URL in a safe new context |

The document response includes source revision provenance, document identity, pipeline version, normalized text, text hash, created time, and ordered units. Each unit includes `id`, optional `parentId`, kind, order, marker, label, start/end offsets, optional page range, stable locator, and content hash.

## Compatibility and safety

- All record lookups require both corpus and child identifiers; a foreign child is indistinguishable from not found.
- Mutations require optimistic `version`; stale writes return `409 stale_state` and current version metadata.
- Request bodies other than PDF are bounded at 1 MB. PDF streaming stops at 10 MB before persistence.
- Additive optional fields are compatible in v1. Removing/renaming fields, changing enum meaning, or changing route semantics requires v2.
- Error messages are presentation-safe but clients localize by stable error and field codes.
