# Evaluation Run Contract v1

This feature-local contract is the design source for the durable public contract that will be published at `contracts/evaluation/v1/`. It is maintainer-facing and does not expose provider prompts, credentials, or raw provider payloads.

## Start request

`POST /api/v1/evaluations`

```json
{
  "datasetRevisionId": "uuid",
  "corpusId": "uuid",
  "snapshotId": "uuid",
  "configuration": {
    "strategy": "vector",
    "fingerprint": "sha256"
  }
}
```

The API accepts only an `available` revision. Before it creates a run, it verifies the requested corpus/snapshot, every source binding, every required snapshot membership, every legal-locator resolution, and the retrieval configuration. `snapshotId` is an immutable input and is never replaced with the active release.

## Start response

```json
{
  "runId": "uuid",
  "state": "queued",
  "datasetRevision": {
    "id": "uuid",
    "contentSha256": "sha256"
  },
  "corpusId": "uuid",
  "snapshotId": "uuid",
  "scoringPolicyVersion": "v1"
}
```

## Preflight errors

| Code | Meaning | Side effect |
| --- | --- | --- |
| `dataset_not_available` | Revision has no available publication. | No run or model call. |
| `corpus_mismatch` | Dataset and requested corpus differ, including information-security corpora. | No run or model call. |
| `snapshot_incompatible` | Snapshot is not a member of the corpus or misses source bindings. | No run or model call. |
| `locator_unresolved` | An expected legal location cannot resolve uniquely in the snapshot. | No run or model call. |
| `invalid_configuration` | Strategy/configuration cannot run on the selected snapshot. | No run or model call. |

Errors contain a bounded `missingRequirements` list with source alias, original locator, and safe reason. They do not include raw document text.

## Run inspection

`GET /api/v1/evaluations/{runId}` returns immutable run identity, state, configuration/model identity, aggregate metrics with denominators, and case summaries. `GET /api/v1/evaluations/{runId}/cases/{caseId}` returns the original question/reference answer, expected and actual evidence identities, generated answer, execution state, scoring components, and safe failure details.

Actual evidence differentiates `retrieved` from `cited`; every item includes the run's corpus and snapshot identity. A citation is valid only when its marker resolves and its evidence belongs to the run snapshot.

## Comparison

`GET /api/v1/evaluations/compare?left={runId}&right={runId}` produces direct deltas only if both runs share dataset content hash, snapshot manifest hash, ordered case-set hash, and scoring-policy version. Otherwise it returns `comparisonState: "non_comparable"` and the identities that differ.

## Provider contract

The API-to-agent evaluation port receives `corpusId`, `snapshotId`, question, normalized interface language, and frozen retrieval configuration. Its request must also include `executionIdentity` with exact non-empty `agentBuild`, `chatModelIdentity`, and `embeddingModelIdentity` fields. The agent executes only when that complete frozen identity matches its normalized deployed configuration.

The response returns answer, outcome, ordered retrieved evidence, citation-marker-compatible evidence order, graph-grounding state, `modelIdentity`, `agentBuildIdentity`, required `embeddingModelIdentity`, and nullable latency/token measurements. The worker verifies the returned identities against the persisted `executionIdentity`. The port must reject evidence outside the supplied corpus and snapshot. It is non-streaming and does not alter public chat stream contracts.
