# GraphRAG Strategy and Stream Contract

## Public research request

`POST /api/v1/corpora/{corpusId}/chat/stream` accepts an optional retrieval strategy:

```json
{
  "question": "Which obligations are connected to a data subject access request?",
  "interfaceLanguage": "en",
  "strategy": "hybrid"
}
```

`strategy` is one of `vector`, `graph`, or `hybrid`. Omission preserves `vector` as the default
for backward compatibility. The browser never supplies a snapshot or graph-release identity.

## Internal agent request

Go resolves the active immutable snapshot and sends the agent a versioned request containing the
corpus ID, required snapshot ID, strategy, question, and interface language. The agent rejects a
missing or invalid snapshot ID or strategy and resolves a ready graph release internally only for
`graph` and `hybrid`.

## Evidence and inspection additions

Every evidence item retains `snapshotId` and may contain a `contribution` of `vector`, `graph`, or
`both`. The inspection event adds the selected strategy, graph-release identity when applicable,
vector and graph evidence counts, and structured graph paths. Every material path relationship
contains a source document and immutable legal location.

## Safe outcomes

| Code | Meaning |
| --- | --- |
| `graph_unavailable` | The active snapshot has no ready graph release. |
| `graph_insufficient_evidence` | The requested graph strategy found no adequately supported path. |
| `graph_release_mismatch` | A release does not belong to the requested corpus snapshot. |
| `graph_projection_failed` | The graph release failed and cannot be queried. |

Safe graph outcomes never include a substitute vector result or foreign evidence.
