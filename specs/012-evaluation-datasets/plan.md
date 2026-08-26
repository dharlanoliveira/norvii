# Implementation Plan: Versioned Evaluation Datasets

**Branch**: `012-evaluation-datasets` | **Date**: 2026-08-25 | **Spec**: [spec.md](spec.md)

## Summary

Create three independently searchable legal corpora, internalize their bilingual golden datasets as reviewed immutable revisions, and evaluate a selected historical corpus snapshot without crossing corpus boundaries. The identities are **Brazilian Personal Data Protection (LGPD)**, **Brazilian Anti-Corruption and White-Collar Crime**, and **United States Fair Housing and Disability Accommodations**. They are not information-security corpora and may neither share sources/snapshots nor be combined in evaluation retrieval.

The API owns the immutable catalog, compatibility preflight, opening-suggestion publication,
work ledger, and maintainer API. A dedicated evaluation-to-agent adapter invokes a fixed snapshot;
it never uses the ordinary chat service, because that service replaces a request snapshot with the
active release. The agent returns per-case answer, evidence, citations, and telemetry. A
database-leased worker executes queued cases so one provider failure does not prevent unrelated
cases from reaching a terminal state. A narrow read-only opening-suggestion endpoint supplies the
ordinary workspace with one corpus and active-snapshot-bound question list; it never exposes
evaluation records or changes the chat stream. The web application exposes a compact maintainer
run, inspection, comparison, and researcher discovery journey after prototype approval.

## Technical Context

**Language/Version**: Go 1.26, Python 3.13, TypeScript 5 / React 19

**Primary Dependencies**: Existing Go HTTP/PostgreSQL layers, Python grounded-chat graph and snapshot-scoped retrieval, React client; no judge model or external benchmark service

**Storage**: PostgreSQL canonical store and versioned project-owned JSON/JSONL assets under `data/corpora/`; Neo4j remains a rebuildable retrieval projection

**Testing**: Go unit/integration/contract tests, Python pytest unit/integration tests, TypeScript tests, migration verification, repository-language validator, and prototype verification

**Target Platform**: Local Compose development environment and the existing Linux service model

**Project Type**: Multi-service web application with offline/background evaluation work

**Performance Goals**: Preflight rejects incompatible selections before any model call; import is deterministic; one run creates exactly one terminal ledger record per case; result inspection is available from persisted records after a run completes; opening suggestions return a stable, rank-ordered empty list or corpus/snapshot-matched list without an evaluation model call

**Constraints**: Project-owned assets only; no network/model call during import; a run uses one explicit corpus snapshot; existing answer token/retry limits remain in force; safe logs exclude raw provider payloads, prompts, credentials, and unnecessary personal data

**Scale/Scope**: Three initial datasets (16 Brazilian anti-corruption, 18 LGPD, and 18 U.S. fair-housing cases), each with PT/EN pairs; maintainer-facing POC rather than arbitrary uploaded or large public benchmarks

## Constitution Check

### Before research

| Principle | Plan response | Status |
| --- | --- | --- |
| I. Specification before implementation | Feature 012 has an independently testable specification, checklist, plan, contracts, data model, and quickstart before implementation tasks. | Pass |
| II. Explicit module boundaries | API owns catalog/results and public controls; agent owns grounded execution; web owns presentation; assets stay repository-owned; modules communicate through versioned contracts. | Pass |
| III. Evidence-grounded legal answers | Each run fixes one named corpus and one immutable snapshot; actual and expected evidence retain immutable provenance; normative authority is distinct from guidance. | Pass |
| IV. Versioned contracts | Evaluation work/result and maintainer HTTP contracts are versioned in `contracts/evaluation/v1/`; all providers/consumers receive contract tests. | Pass |
| V. Tested maintainable code | Import, source binding, locator resolution, snapshot isolation, worker lease recovery, scoring, contracts, and views have deterministic tests. | Pass |
| VI. Reproducible and cost-bounded POC | Hashes, manifests, reviewed publications, frozen configuration, bounded case sets, and a local quickstart make results repeatable. | Pass |
| VII. Observable and safe by design | Run/case state, scoring rationale, evidence identities, telemetry, and bounded failures are stored; sensitive prompt/provider detail is excluded. | Pass |
| VIII. English engineering language | New code, contracts, docs, schemas, errors, and tests are English; the existing PT legal questions and evidence remain deterministic corpus content. | Pass |

### After design

Pass. The design keeps the production modules separate and introduces no unapproved technology. The only added durable data boundary is PostgreSQL, already canonical under ADR 0005. Prototype work precedes a new production maintainer journey.

## Design Decisions

1. **Create three legal corpora; never reuse an information-security corpus.** Seed explicit stable corpus identities: `brazil-personal-data-protection`, `brazil-anti-corruption-white-collar-crime`, and `us-fair-housing-disability-accommodations`. Dataset eligibility is keyed to corpus UUID, not a title, URL, jurisdiction label, or security-domain tag.
2. **Internalize local assets, not external URLs.** A deterministic Go import command validates `data/corpora/*/evaluation/`, canonicalizes each manifest/JSONL pair, and creates an immutable revision identified by combined SHA-256. It performs neither acquisition nor model work.
3. **Publish separately from import.** Imported revisions are draft. An append-only review and publication record promotes a revision only after its official source bindings and legal review are complete. Re-importing an identical hash is idempotent.
4. **Bind source aliases once and resolve legal locators snapshot-by-snapshot.** Manifest aliases bind to a corpus `sources.id`. Preflight resolves each legal display locator to immutable source, document version, legal unit/span, and content hash in the selected snapshot. URLs remain provenance, never runtime identity. No fuzzy title/URL/text matching is allowed.
5. **Preserve a canonical legal locator.** Ingestion must retain or derive an auditable legal locator/alias for each legal unit. It must not compare human locators such as `Art. 19, II` directly to generic extractor labels such as `article-19`. Compound locators are expanded to atomic targets and all required targets must resolve.
6. **Execute with an explicitly selected historical snapshot.** The evaluation adapter sends the stored corpus and snapshot IDs directly to the agent's non-streaming per-case port. It does not pass through `chat/application.Service`, which resolves the active release.
7. **Use deterministic, evidence-focused scoring in v1.** Release-gate metrics are import integrity, preflight compatibility, expected-evidence retrieved/cited coverage, citation scope/validity, abstention outcome, execution status, and telemetry. `expected_answer` and `required_propositions` remain review material; prose similarity and LLM-as-judge are not a quality gate. Claim-level semantic support is `needs_human_review` until a reviewed claim-to-citation rubric exists.
8. **Persist actual retrieval and citations separately.** The runner maps `[n]` answer markers to the corresponding evidence item, then stores retrieved evidence and actually cited evidence as different immutable records. An uncited retrieval cannot earn citation coverage.
9. **Make comparison strict and honest.** Direct deltas require the same dataset revision/content hash, corpus snapshot/manifest hash, ordered full case set, and scoring-policy version. Model and retrieval configuration may differ and are shown as the experimental variables. Failed/cancelled cases are never converted to zero; denominators are explicit.
10. **Publish opening suggestions separately from evaluation runs.** A reviewed available dataset
    revision explicitly ranks a bounded paired case subset. The API materializes it as an
    append-only corpus and snapshot-bound projection containing only case IDs, checksums, ranks,
    languages, and question text. The ordinary workspace reads this projection through its own
    versioned contract; it does not read evaluation tables at runtime, invoke evaluation
    preflight/scoring, or alter public chat streaming. A former projection is hidden when a newer
    snapshot becomes active.

## Project Structure

### Documentation

```text
specs/012-evaluation-datasets/
+-- spec.md
+-- plan.md
+-- research.md
+-- data-model.md
+-- quickstart.md
+-- checklists/
+-- contracts/

contracts/evaluation/v1/             # Shared durable wire contracts
contracts/corpus-opening-suggestions/v1/ # Public researcher-facing suggestion read contract
docs/product/corpora.md              # Named corpus scope and source manifests
docs/product/evaluation.md           # Evaluation strategy and metric semantics
data/corpora/*/evaluation/           # Immutable curated input assets
```

### Source code

```text
apps/api/
+-- cmd/import-evaluation-datasets/  # Deterministic local asset importer
+-- internal/evaluation/
|   +-- domain/                      # Dataset, binding, run, scoring vocabulary
|   +-- application/                 # Import, preflight, run, compare use cases
|   +-- agent/                       # Explicit-snapshot agent adapter
|   +-- postgres/                    # Canonical catalog/run/result persistence
|   +-- http/                        # Maintainer dataset/run/result API
|   +-- suggestions/                 # Opening-suggestion publication and read API
+-- migrations/                      # Corpus/source seeds and evaluation tables
+-- tests/

apps/agent/
+-- src/norvii_agent/                # Non-streaming explicit-snapshot case execution port
+-- tests/

apps/web/
+-- src/api/                         # Versioned evaluation client
+-- src/features/evaluation/         # Maintainer run, inspection, comparison views
+-- src/features/workspace/          # Corpus-specific empty-chat suggestions
+-- src/app/routes.tsx

prototypes/web/                      # Evaluation journey baseline/approval
```

**Structure Decision**: The API is the canonical PostgreSQL writer because datasets, snapshots, and maintainer-facing results share one transactional provenance boundary. The agent only executes grounded cases against an already-approved fixed snapshot. The web client never reads persistence directly. New corpus source acquisition and extraction follow the existing controlled ingestion protocol but are not part of dataset import.

## Implementation Phases

### Phase 0 -- Establish corpus identity and source readiness

1. Amend the product corpus registry with the three named legal corpora, their jurisdiction, language, scope, official-source manifest, and an explicit statement that none is an information-security corpus.
2. Add deliberate corpus/source seed definitions and snapshot initialization support. Reconcile the LGPD golden source with the authoritative source selected for the LGPD corpus; seed the anti-corruption and U.S. fair-housing sources rather than silently attaching them elsewhere.
3. Run ordinary source acquisition/extraction through the existing controlled process, preserving origin, capture date, hash, authority role, and document versions. Publish snapshots only when the intended source set is ready.
4. Extend legal-unit extraction/provenance with canonical legal locator aliases. Keep positional locators for structure, but bind evaluation targets to canonical legal locations and immutable unit/span identities.

**Exit criteria**: All three corpus IDs exist independently; no source belongs to a security corpus; each target source has official provenance; fixtures prove every dataset locator can be resolved in its intended published snapshot or the dataset remains draft.

### Phase 1 -- Dataset catalog, binding, and compatibility preflight

1. Define the contracts in `contracts/evaluation/v1/` and new immutable PostgreSQL tables from [data-model.md](data-model.md). Create migrations after the current latest migration, with repository-standard rollback and indexes.
2. Add the local importer. It validates JSON/JSONL schema, sizes, IDs, manifest references,
   language vocabulary, reciprocal PT/EN pairs, explicitly ranked opening-suggestion pairs,
   non-empty evidence, compound locator expansion, and idempotent content hashes.
3. Store immutable input records, source requirements, cases, and evidence expectations; create a separate review/publication record. Source bindings map manifest aliases to corpus source UUIDs.
4. Implement preflight: dataset publication, selected corpus identity/jurisdiction, snapshot membership for every manifest source, full legal-locator resolution, and frozen retrieval configuration. Return all bounded missing requirements before any model call.
5. Add explicit `expected_outcome` (`answer` or `abstain`) to the schema. Revise the initial
   datasets with legally reviewed abstention cases before claiming abstention accuracy; until then
   report that metric as `not_applicable`. Store the required ranked opening-suggestion pair
   selection separately from JSONL order.

**Exit criteria**: All assets import reproducibly as drafts; only reviewed revisions with complete bindings and resolved evidence can be selected; incompatible corpus/snapshot selection makes zero agent calls.

### Phase 2 -- Fixed-snapshot execution and durable results

1. Add an evaluation-specific non-streaming agent contract that accepts only a fixed corpus, snapshot, case question, normalized interface language, and frozen retrieval configuration. It returns answer, evidence, citation-marker mapping inputs, outcome, graph validation state, model/build metadata, and nullable telemetry.
2. Add `evaluation_run` plus one leased case-work record per dataset case at creation. A worker claims bounded batches, retries safely, records terminal execution state, and continues after independent case failures. It snapshots all identities/configuration before work starts.
3. Parse answer markers against evidence order; materialize expected targets and actual retrieved and cited evidence separately. Reject or flag cross-snapshot/corpus evidence even if a provider response attempts to include it.
4. Apply scorer version 1. Store component values, verdict, denominators, and rationale rather than only a pass/fail. Keep provider failure, cancellation, scope-limited completion, and abstention separate.

**Exit criteria**: A compatible run yields one terminal record per case, preserves historical snapshot provenance, stores safe inspection data, and cannot be changed by a later active release.

### Phase 3 -- Maintainer experience and comparison

1. Build/verify the evaluation journey in `prototypes/web/`, referencing Feature 001's approved research workspace baseline, then implement the approved compact web view.
2. Expose dataset readiness, fixed corpus/snapshot selection, start status, aggregate results, case-level expected-versus-actual evidence, failure rationale, and the technical-not-legal-advice notice.
3. Expose comparison only after validating the strict comparison key. Show configuration deltas, counts, eligible denominators, paired-case asymmetry, and a non-comparable state instead of a quality delta when keys differ.
4. Publish the selected starter-question pairs as a snapshot-compatible corpus projection and
   expose a narrow versioned read endpoint. Render only rank-ordered questions for the active
   corpus and interface language in an empty ordinary chat; hide the region for an absent or stale
   projection, and preserve the existing chat request and stream.

**Exit criteria**: Maintainers can start, inspect, and correctly compare compatible runs without exposing prompts/secrets or suggesting legal conclusions.

### Phase 4 -- Verification and documentation

1. Run all contract, unit, integration, migration, persistence, prototype, and client checks in [quickstart.md](quickstart.md).
2. Verify direct selection failures for a security corpus, wrong legal corpus, missing source, unresolved locator, stale/non-member snapshot, and cross-snapshot agent evidence.
3. Update durable product/module documentation with actual commands, source readiness, scorer limits, the human-review boundary, and operational ownership.

## Norvii Implementation Requirements

### Module Impact

| Module | Change | Responsibility | Verification |
| --- | --- | --- | --- |
| `prototypes/web/` | Change | Validate maintainer evaluation entry, inspection, and non-comparable state before production UI. | Prototype approval evidence/screenshots. |
| `apps/web/` | Change | Render dataset availability, run status, results, paired cases, comparison caveats, and corpus-specific opening suggestions. | Component/API parsing tests. |
| `apps/api/` | Change | Own catalog import, source bindings, opening-suggestion projection, preflight, run/result persistence, worker, and maintainer API. | Go unit/integration/migration/contract tests. |
| `apps/agent/` | Change | Execute one case against the explicitly supplied immutable snapshot and return safe evaluation output. | Python snapshot-isolation and contract tests. |
| `apps/ingestion/` | Change | Preserve canonical legal locator aliases and source authority metadata during normal source extraction. | Python extraction/provenance tests. |
| `contracts/` | Change | Define versioned evaluation work/result, public inspection, and corpus opening-suggestion read contracts. | Go/Python/TypeScript contract compatibility tests. |
| `infra/` | Change | Register the managed evaluation worker with the existing local lifecycle, persistent logs, and readiness checks. | Bootstrap/status tests and worker lifecycle verification. |

### Boundaries and constraints

- **Cost limits**: One queued run covers one published dataset revision only. Use existing per-answer model/retrieval limits; record nullable token and latency telemetry; no LLM judge.
- **Prototype baseline**: Feature 001 research workspace; new maintainer evaluation UI is intentionally separate from researcher chat and requires prototype approval.
- **Public contracts**: New `evaluation/v1` contracts require explicit provider/consumer tests. Existing chat streaming compatibility is unchanged.
- **Opening suggestions**: A separate read-only `corpus-opening-suggestions/v1` contract returns
  only rank, case ID, and original question text for the active corpus and matching snapshot. It
  never returns evaluation answers/evidence and never makes the workspace depend on the evaluator.
- **Persistence**: Append-only dataset revisions/publications; run identity/configuration and materialized evidence are immutable. Migrations include rollback in dependency-safe order.
- **Ingestion artifacts**: Canonical legal locator aliases are versioned with document units and source revisions. Dataset import does not fetch URLs or run ingestion.
- **Streaming**: Evaluation uses an internal non-streaming case path. It must not change public chat stream events.
- **Corpus boundary and citations**: Every expected and actual evidence item must be a member of the stored run snapshot and named legal corpus. Information-security corpora are invalid inputs.
- **Security and privacy**: Assets are maintainer-owned and bounded. Do not log answers' raw prompts, provider payloads, credentials, or nonessential identifiers; validate local paths and reject unrecognized fields/oversized assets.
- **Observability**: Persist dataset revision/hash, snapshot/manifest hash, source binding, locator resolution, agent build/model/configuration, case lifecycle, scorer version, counts, and safe failure code.
- **Local environment**: Document the seed/source/snapshot prerequisite and run the evaluation
  worker through the existing managed local lifecycle; no new external service is required.

## Complexity Tracking

No constitution violation requires an exception. The worker, separate agent adapter, and legal locator layer are required to preserve existing explicit module and evidence boundaries.
