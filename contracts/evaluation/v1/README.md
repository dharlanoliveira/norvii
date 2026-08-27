# Evaluation Contract v1

This durable contract governs the maintainer evaluation API and the API-to-agent fixed-snapshot
execution boundary. The approved feature design remains in
[`specs/012-evaluation-datasets/contracts/evaluation-run-v1.md`](../../../specs/012-evaluation-datasets/contracts/evaluation-run-v1.md).

Every route in this document requires maintainer authorization. Unauthorized callers receive only
the standard authorization problem; they cannot determine whether a dataset revision, source,
starter case, corpus, or snapshot exists.

## Run inspection and comparison

- `GET /api/v1/evaluations/{runId}` returns one immutable run summary: dataset and snapshot
  identities, frozen retrieval and execution identities, lifecycle timestamps, aggregate case
  totals, metric numerators and denominators, metric rationales, and safe case status summaries.
- `GET /api/v1/evaluations/{runId}/cases/{caseId}` returns one immutable case result. Its
  `expectedEvidence` and `actualEvidence` arrays are separate. Actual items identify whether they
  were `retrieved` or `cited`; all evidence is immutable provenance for the run corpus and
  snapshot, never source text. Case metrics retain their rationale and scorer version. Safe
  failures expose only `failureCode`.
- `GET /api/v1/evaluations/compare?left={runId}&right={runId}` compares two terminal historical
  runs. It emits quality deltas only with the same immutable dataset, corpus/snapshot, ordered
  case-set, and scoring-policy identities. Otherwise it returns `comparisonState:
  "non_comparable"`, the differing identity fields, and no totals or metric deltas. A comparable
  response includes the permitted experimental variables, paired/unpaired and failure counts,
  and each metric's explicit paired numerator and denominator arithmetic.

Run and comparison responses never include provider prompts, provider payloads, credentials, raw
document text, or hidden scoring inputs. The data is a technical measurement of corpus-grounded
system behavior, not legal advice.

## Dataset inspection

The API publishes immutable dataset inspection under these routes:

- `GET /api/v1/evaluation-datasets` returns a catalog projection. Each entry contains the immutable
  revision identity, language and hash identity, availability, and latest review state. It never
  includes source bindings or starter cases.
- `GET /api/v1/evaluation-datasets/{datasetRevisionId}` returns that catalog identity plus immutable
  manifest source authority and corpus bindings, and starter-case metadata. Starter metadata is
  limited to selection ID, case ID, rank, query language, and review eligibility; it excludes case
  questions, reference answers, expected evidence, review notes, outcomes, and provider data.
- `GET /api/v1/evaluation-datasets/{datasetRevisionId}/preflight?corpusId={corpusId}&snapshotId={snapshotId}`
  verifies compatibility for exactly those immutable identities. It does not create a run, queue
  work, call a model, or replace the selected snapshot.

`datasetRevisionId`, `corpusId`, and `snapshotId` must be non-nil UUIDs. An invalid identifier
returns `400 invalid_input`; an absent revision returns `404 not_found` only to an authorized
caller. A successful preflight echoes all three selected identities and returns `compatible: true`.

An incompatible preflight returns `422` with one of the existing compatibility codes:
`dataset_not_available`, `corpus_mismatch`, `snapshot_incompatible`, or `locator_unresolved`.
Its optional `missingRequirements` array contains at most 32 safe entries with a source alias,
optional display locator, and safe reason. It never contains corpus text, case content, reference
answers, required propositions, prompts, credentials, or provider payloads.

The catalog and detail fixtures preserve the revision's corpus identity, hashes, supported query
languages, and authoritative evidence language. `available` is true only when the latest immutable
review is `approved` and its publication state is `available`.

## Fixtures

The language-neutral JSON examples in [`fixtures/`](fixtures/) are shared input for Go, Python,
and TypeScript contract tests. Dataset inspection fixtures are:

- `dataset-catalog-response.json`
- `dataset-detail-response.json`
- `dataset-preflight-success.json`
- `dataset-preflight-error-snapshot-incompatible.json`
- `dataset-preflight-error-not-found.json`
- `evaluation-run-summary-response.json`
- `evaluation-run-case-response.json`
- `evaluation-comparison-response.json`
- `evaluation-comparison-non-comparable-response.json`

The existing start and preflight-error fixtures represent queued starts and rejected starts. All
preflight-error fixtures use the public error envelope with bounded `missingRequirements`.
Invalid request fixtures must be rejected by strict request decoders.

Fixtures use synthetic identifiers, questions, reference answers, locators, and hashes. They do
not contain legal corpus text, provider payloads, prompts, or credentials.
