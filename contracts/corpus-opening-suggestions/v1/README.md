# Corpus Opening Suggestions Contract v1

This contract exposes a read-only, researcher-facing selection of questions for an empty corpus
chat. It is separate from evaluation execution and public chat streaming. A response contains no
reference answer, expected evidence, dataset revision, scoring state, provider data, or prompt.

## Read request

`GET /api/v1/corpora/{corpusId}/opening-suggestions?interfaceLanguage=en|pt`

`interfaceLanguage` accepts only `en` or `pt`. The response is scoped to the requested corpus's
current active snapshot at request time.

## Read response

```json
{
  "corpusId": "uuid",
  "activeSnapshotId": "uuid-or-null",
  "activeSnapshotManifestSha256": "sha256-or-null",
  "interfaceLanguage": "en",
  "suggestions": [
    {
      "caseId": "lgpd-002-en",
      "rank": 1,
      "question": "What is the difference between a controller and an operator under the LGPD?"
    }
  ]
}
```

`suggestions` is an empty array when the corpus is disabled, has no active snapshot, has no
published projection, or the published projection does not match the active snapshot and manifest.
The response is rank-ordered and contains at most five items.

## Invariants

- Every item is an original selected dataset question from the requested corpus.
- Every item has one reciprocal counterpart in the other supported interface language at the same
  rank; the endpoint returns only the requested language member.
- The client discards a response when its active snapshot identity no longer matches the workspace
  corpus response, then reloads it if appropriate.
- Selecting a question sends the exact `question` through the existing corpus-bound chat flow.
- The endpoint neither invokes evaluation preflight/scoring nor changes the public chat stream.

## Fixtures

The language-neutral JSON examples in [`fixtures/`](fixtures/) are shared by Go, Python, and
TypeScript contract tests. Valid fixtures cover reciprocal English and Portuguese ranks plus empty
outcomes for unavailable and stale projections. Invalid fixtures must be rejected by strict
consumers because they either violate rank ordering or expose evaluation/provider-only fields.

Fixture questions and identifiers are synthetic. They are not legal corpus content and do not
contain reference answers, expected evidence, provider payloads, prompts, or credentials.
