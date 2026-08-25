# Planned Hybrid Stream Contract

## Version and compatibility

This contract refines the GraphRAG stream contract from Feature 008 for new requests. An already
rendered legacy `graph` result remains readable within its browser session; public new-request
validation rejects `graph`.

## Public research request

`POST /api/v1/corpora/{corpusId}/chat/stream` accepts an optional approach:

```json
{
  "question": "Which rights are connected to an access request?",
  "interfaceLanguage": "en",
  "strategy": "hybrid"
}
```

`strategy` is `vector` or `hybrid`. Omission remains `vector` for compatibility. The browser does
not send a snapshot or graph-release identifier.

## Internal agent request

The API resolves the active immutable snapshot and forwards corpus ID, snapshot ID, question,
interface language, and strategy. Hybrid requests are valid even when there is no ready graph
release; the agent reports the graph stage as unavailable while preserving vector retrieval.

## Terminal inspection projection

The `completed` event adds `inspection.stages`:

```json
{
  "strategy": "hybrid",
  "stages": [
    {
      "name": "vector",
      "state": "completed",
      "reasonCode": null,
      "evidenceCount": 4,
      "durationMilliseconds": 231,
      "inputTokens": null,
      "outputTokens": null
    },
    {
      "name": "planning",
      "state": "completed",
      "reasonCode": null,
      "evidenceCount": 0,
      "durationMilliseconds": 355,
      "inputTokens": 176,
      "outputTokens": 22
    },
    {
      "name": "graph",
      "state": "no_evidence",
      "reasonCode": null,
      "evidenceCount": 0,
      "durationMilliseconds": 15,
      "inputTokens": null,
      "outputTokens": null
    }
  ]
}
```

Each evidence reference may include `contribution` with `vector`, `graph`, or `vector_and_graph`. Existing
inspection fields remain available during the transition. `graphPath` is non-empty only when the
graph stage contributes evidence.

## Stage states and safe reason codes

| Stage | Allowed states | Example safe reason codes |
| --- | --- | --- |
| `vector` | `completed`, `no_evidence` | none |
| `planning` | `completed`, `skipped`, `unavailable`, `failed` | `not_relevant`, `graph_release_unavailable`, `graph_unavailable`, `planner_unavailable` |
| `graph` | `completed`, `skipped`, `unavailable`, `no_evidence` | `not_relevant`, `graph_release_unavailable`, `graph_unavailable`, `planner_unavailable` |

No event contains prompts, hidden reasoning, graph query text, credentials, or full corpus
content.
