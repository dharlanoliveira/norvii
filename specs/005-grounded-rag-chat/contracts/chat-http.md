# Grounded Chat HTTP Contract

## Ownership and compatibility

- **Owner**: Go API
- **Consumer**: React web client and contract tests
- **Initial version**: `v1`
- **Scope**: One active-corpus research request and its structured stream
- **Compatibility**: Additive event fields are compatible; changing event meaning or required
  fields requires a new version.

## Request

`POST /api/v1/corpora/{corpusId}/chat/stream`

```json
{
  "question": "What conditions apply to this obligation?",
  "interfaceLanguage": "en"
}
```

The API validates UUID, enabled corpus ownership, non-empty question, language (`en` or `pt`),
and configured size limits before retrieval. The question is not logged.

## Response framing

The response uses `text/event-stream`. Each event has one JSON `data` object. The API emits one
`started`, one `evidence`, zero or more `delta`, and exactly one terminal event (`completed`,
`abstained`, `cancelled`, or `error`). Provider payloads never cross this boundary.

```text
event: started
data: {"type":"started","requestId":"...","corpusId":"..."}

event: evidence
data: {"type":"evidence","requestId":"...","references":[...]}

event: delta
data: {"type":"delta","requestId":"...","text":"..."}

event: completed
data: {"type":"completed","requestId":"...","answer":"...","references":[...],"telemetry":{...}}
```

## Evidence reference

Each reference has `id`, `corpusId`, `sourceId`, `documentId`, `unitLocator`, `startOffset`,
`endOffset`, `excerpt`, and `rank`. `excerpt` is bounded and faithful to the preserved document.

## Terminal outcomes

- `completed`: final answer and at least one resolvable reference for each supported factual
  segment.
- `abstained`: insufficient evidence, unavailable ready documents, or failed grounding validation;
  no answer citations are presented as factual support.
- `cancelled`: the client or server cancelled the request; partial text is not marked complete.
- `error`: stable public error code and safe message; internal causes remain in diagnostics.

## Public error codes

`invalid_question`, `corpus_unavailable`, `insufficient_evidence`, `retrieval_failed`,
`generation_failed`, `request_cancelled`, `evidence_unavailable`, and `internal_error`.
