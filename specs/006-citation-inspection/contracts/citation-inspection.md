# Citation Navigation and Inspection Contract

## Ownership and Compatibility

- **Public owner**: Go API
- **Internal producer**: Python agent
- **Consumer**: React web client
- **Base contract**: Feature 005 grounded chat stream
- **Compatibility**: This is an additive extension to the existing `v1` stream. Existing fields
  retain their meaning. Feature 006 clients require the new completed-answer inspection fields.

## Immutable Document Read

```text
GET /api/v1/corpora/{corpusId}/sources/{sourceId}/documents/{documentVersionId}
```

The response has the existing `DocumentResponse` representation. It is returned only when the
named published document version belongs to both route identifiers. The route does not fall back
to the latest ready document.

| Condition | Response |
|---|---|
| Invalid route identifier | `400 invalid_input` |
| Foreign, absent, unpublished, or unavailable document version | `404 not_found` |
| Database failure | existing safe `5xx` error envelope |

The existing `GET .../sources/{sourceId}/document` endpoint remains the latest-document browser
route and is not suitable for resolving answer citations.

## Additive Reference Fields

Every completed-answer reference retains Feature 005 fields and adds:

```json
{
  "documentVersionId": "a9c6e9c7-93d1-4bc2-aa4d-11cff3b9f6b4",
  "sourceRevisionId": "ee5457c0-9e85-4db8-a169-e6b09e4e3d98",
  "pipelineVersion": "corpus-ingestion-v1",
  "sourceTitle": "Official English GDPR text",
  "cosineDistance": 0.1834
}
```

`documentId` remains present for compatibility and has the same value as `documentVersionId`.
`cosineDistance` is either a finite non-negative number or `null`; it is not a percentage or an
application-defined score.

## Terminal Inspection Payload

Terminal events gain an `inspection` object. Its values are structured, content-free apart from
the already public citation excerpt, and must never contain prompts, user questions, credentials,
or raw provider payloads.

```json
{
  "type": "completed",
  "requestId": "c27fe7c5-1cca-4209-a33c-53c0116db16b",
  "answer": "...",
  "references": [],
  "telemetry": {
    "outcome": "completed",
    "evidenceCount": 2,
    "durationMilliseconds": 412
  },
  "inspection": {
    "outcome": "completed",
    "retrieval": {
      "strategy": "vector",
      "topK": 8,
      "returnedCount": 2,
      "embeddingModel": null
    },
    "measurements": {
      "retrievalMilliseconds": 37,
      "generationMilliseconds": 298,
      "totalMilliseconds": 412,
      "inputTokens": null,
      "outputTokens": null
    },
    "evidence": []
  }
}
```

For a `completed` event, `inspection.evidence` has the same ordered immutable records as
`references`. For an `abstained`, `cancelled`, or `error` event, the payload may contain outcome
and available measurements but MUST omit `evidence` and retrieval information that would imply a
completed grounded answer.

All duration fields are milliseconds. Missing provider or measurement support is encoded as JSON
`null`, never `0` or an estimate. `telemetry.durationMilliseconds` remains a compatibility value
and is the same measured value as `inspection.measurements.totalMilliseconds` when both are
available.

## Validation Rules

1. A reference's `corpusId`, `sourceId`, `documentVersionId`, offsets, and rank are validated by
   the producing agent and the consuming API/client.
2. The Go API relays agent inspection stage values unchanged and only adds its owned total request
   duration.
3. The React client rejects malformed values, foreign corpus references, non-finite distances, or
   negative durations/tokens. It shows an explicit localized unavailable state instead of
   navigating to a substitute.
4. Product copy is localized through the existing English and Portuguese translation resources;
   preserved excerpts and generated answers are not translated by the inspection renderer.
