# Tasks: Versioned Evaluation Datasets

**Input**: Design documents from `specs/012-evaluation-datasets/`

**Prerequisites**: `spec.md`, `plan.md`, `research.md`, `data-model.md`,
`contracts/`, and `quickstart.md`

**Tests**: Tests are required for every changed behavior, public boundary, failure path, and
cross-language contract.

**Organization**: Tasks are grouped by independently testable user story. `US0` is the
corpus-specific empty-chat discovery flow; `US1` through `US3` are the evaluation journeys.

## Phase 1: Setup and owned assets

**Purpose**: Make the reviewed input and public contracts explicit before adding runtime behavior.

- [x] T001 [P] Add the ranked reciprocal starter-case metadata required by FR-014 to `data/corpora/brazil-lgpd/evaluation/brazil-lgpd-v1.jsonl`, `data/corpora/brazil-anti-corruption/evaluation/brazil-anti-corruption-v1.jsonl`, and `data/corpora/us-fair-housing-disability-accommodations/evaluation/us-fair-housing-v1.jsonl`.
- [x] T002 [P] Define machine-readable evaluation and opening-suggestion fixtures from `contracts/evaluation/v1/README.md` and `contracts/corpus-opening-suggestions/v1/README.md` for Go, Python, and TypeScript contract tests.
- [x] T003 [P] Update `specs/012-evaluation-datasets/quickstart.md`, `docs/product/corpora.md`, and `docs/product/evaluation.md` with the three corpus identities, source-readiness prerequisites, starter-question publication, and no-legal-advice boundary.

---

## Phase 2: Foundational corpus, locator, and persistence boundaries

**Purpose**: Establish the isolated corpus, canonical locator, immutable catalog, and publication
boundaries that block every story.

- [x] T004 Add migration `apps/api/migrations/013_evaluation_datasets.sql` with the three isolated corpus/source seeds, evaluation revision/source/case/evidence/publication tables, opening-suggestion projection tables, ownership constraints, indexes, and dependency-safe rollback.
- [x] T005 Add migration and schema assertions in `apps/api/tests/integration/migration_test.go`, `apps/api/tests/integration/catalog_schema_test.go`, and a new `apps/api/tests/integration/evaluation_schema_test.go` for corpus isolation, append-only storage, starter rank/language uniqueness, and absence of information-security reuse.
- [x] T006 Add canonical legal-locator alias extraction and immutable provenance support in `apps/ingestion/src/norvii_ingestion/extraction/`, `apps/ingestion/src/norvii_ingestion/domain/artifacts.py`, and `apps/ingestion/src/norvii_ingestion/publication/postgres/repository.py`, with tests in `apps/ingestion/tests/unit/extraction/` and `apps/ingestion/tests/unit/publication/`.
- [x] T007 Add explicit corpus-source binding and snapshot locator-resolution repositories in `apps/api/internal/evaluation/postgres/`, with integration tests proving all required targets resolve only in the intended corpus snapshot.
- [x] T008 Add shared evaluation domain vocabulary and immutable records in `apps/api/internal/evaluation/domain/`, with unit tests for lifecycle, corpus/snapshot ownership, paired language cases, and stable checksums.
- [x] T009 Add deterministic local asset validation and import application ports in `apps/api/internal/evaluation/application/`, including bounded paths/sizes, unknown-field rejection, hashes, reciprocal pairs, expected evidence, expected outcomes, and ranked starter-case validation, with unit tests.
- [x] T010 Add `apps/api/cmd/import-evaluation-datasets/main.go`, command wiring in `apps/api/Makefile`, and importer integration tests proving three assets and 52 cases import idempotently without network or model calls.

**Checkpoint**: The three corpus identities, immutable dataset catalog, legal locators, and import
boundary are ready. No public chat or model behavior changes in this phase.

---

## Phase 3: User Story 0 - Corpus-grounded chat opening questions (Priority: P1)

**Goal**: An empty chat presents only deterministic, language-matched questions from the active
corpus and active snapshot.

**Independent Test**: Publish a selected set for each fixture corpus/snapshot, load the workspace
in both interface languages, submit one suggestion, and verify no question or chat request crosses
the corpus boundary.

- [x] T011 [US0] Add opening-suggestion publication domain/application rules in `apps/api/internal/suggestions/domain/` and `apps/api/internal/suggestions/application/`, requiring an available reviewed dataset, compatible snapshot, resolved expected evidence, five-or-fewer ranks, and no stale active-release match.
- [x] T012 [US0] Add append-only projection persistence and active-snapshot matching reads in `apps/api/internal/suggestions/postgres/`, with repository tests for stale releases, cross-corpus rejection, language pairing, rank order, and empty outcomes.
- [x] T013 [US0] Implement the `contracts/corpus-opening-suggestions/v1/README.md` HTTP endpoint in `apps/api/internal/suggestions/http/` and server wiring in `apps/api/cmd/server/main.go`, with handler and contract tests for `en`, `pt`, disabled/missing corpus, stale responses, and no evaluation-data leakage.
- [x] T014 [US0] Add prototype corpus-specific suggestion fixtures and empty-chat journeys in `prototypes/web/src/fixtures/` and `prototypes/web/src/`, including locale switching, no-set, and release-drift states without importing production code.
- [x] T015 [US0] Add strict opening-suggestion response parsing and abortable provider support in `apps/web/src/api/contract.ts`, `apps/web/src/api/researchProvider.ts`, and their tests.
- [x] T016 [US0] Replace the static LGPD-only question list in `apps/web/src/features/workspace/ResearchChat.tsx` with the corpus/snapshot-bound contract data passed from `apps/web/src/features/workspace/CorpusWorkspacePage.tsx`; preserve accessible buttons and normal `useNorviiChatRuntime` submission.
- [x] T017 [US0] Update `apps/web/src/features/workspace/ResearchChat.test.tsx`, `apps/web/src/features/workspace/CorpusWorkspacePage.test.tsx`, `apps/web/src/i18n/en/translation.ts`, and `apps/web/src/i18n/pt/translation.ts` to verify no hard-coded legal question remains in localization, exact rank/language display, abort/route-change safety, normal submission, and no fallback.
- [x] T018 [US0] Add API-to-web integration coverage for all three corpus starter selections and active-snapshot changes in `apps/web/tests/e2e/corpus-ingestion.spec.ts` or a new `apps/web/tests/e2e/corpus-opening-suggestions.spec.ts`.

**Checkpoint**: User Story 0 is independently demonstrable without invoking the evaluation runner
or changing the public chat stream.

---

## Phase 4: User Story 1 - Run a corpus-grounded evaluation (Priority: P1)

**Goal**: An evaluator starts a fixed-snapshot run and receives a terminal result for every
compatible dataset case.

**Independent Test**: Select a compatible published revision and snapshot, execute a fake-agent
run, and inspect one persisted terminal record per case with corpus/snapshot provenance.

- [x] T019 [US1] Implement reviewed dataset publication, source binding, and all-or-nothing compatibility preflight in `apps/api/internal/evaluation/application/` and `apps/api/internal/evaluation/postgres/`, with tests for draft, wrong corpus, missing source, unresolved locator, and zero model calls.
- [x] T020 [US1] Implement the fixed-snapshot, non-streaming evaluation request/result contract in `apps/agent/src/norvii_agent/evaluation/`, reusing only explicit snapshot retrieval ports and adding Python unit and snapshot-isolation tests.
- [x] T021 [US1] Implement safe retrieved-versus-cited evidence materialization, marker parsing, cross-boundary rejection, abstention handling, and deterministic scorer v1 in `apps/api/internal/evaluation/`, with unit tests for each metric and failure outcome.
- [x] T022 [US1] Implement immutable `evaluation_run` and leased `evaluation_run_case` persistence, worker claim/retry/terminal transitions, aggregate denominators, and managed local lifecycle integration in `apps/api/internal/evaluation/postgres/`, `apps/api/cmd/evaluation-worker/`, `Makefile`, and `infra/scripts/manage-local-environment.py`, with recovery and lifecycle integration tests.
- [x] T023 [US1] Implement the evaluation-to-agent adapter in `apps/api/internal/evaluation/agent/` and contract tests against the Python evaluation port, ensuring the public chat application service is never used.
- [x] T024 [US1] Implement maintainer start/status/case-result HTTP endpoints in `apps/api/internal/evaluation/http/` and server wiring in `apps/api/cmd/server/main.go`, with tests for immutable results, safe diagnostics, and historical snapshot retention.

**Checkpoint**: A compatible run produces one immutable terminal case ledger record per case,
continues after independent provider failure, and never follows a later active release.

---

## Phase 5: User Story 2 - Trust dataset provenance and compatibility (Priority: P2)

**Goal**: A maintainer can inspect dataset identity, source requirements, review state, and all
preflight failures before a model call.

**Independent Test**: Inspect an imported revision, bind it to its intended source/snapshot, and
prove incompatible selections return bounded missing requirements with no queued case work.

- [x] T025 [US2] Add dataset catalog/read models and maintainer inspection endpoints in `apps/api/internal/evaluation/http/` and `apps/api/internal/evaluation/postgres/`, with Go API tests for manifest, language, source authority, review state, and starter-case disclosure only in maintainer scope.
- [x] T026 [US2] Add strict TypeScript evaluation catalog contracts and API client methods in `apps/web/src/api/`, with parser tests for availability, missing requirements, and unknown fields.
- [x] T027 [US2] Build the approved maintainer dataset readiness and snapshot-selection view in `apps/web/src/features/evaluation/`, with component tests proving incompatible choices cannot start a run and the technical-not-legal-advice notice remains visible.

**Checkpoint**: Dataset provenance and compatibility are independently inspectable before run
execution, with no cross-corpus substitution.

---

## Phase 6: User Story 3 - Compare repeatable quality results (Priority: P3)

**Goal**: A maintainer can inspect results and compare only compatible runs with honest metrics.

**Independent Test**: Complete two fake-agent runs on one snapshot/dataset with distinct recorded
configurations, compare them, then prove another snapshot/revision is non-comparable.

- [x] T028 [US3] Implement strict comparison application/repository logic in `apps/api/internal/evaluation/application/` and `apps/api/internal/evaluation/postgres/`, with tests for equality key, paired-case denominators, failed/cancelled cases, and non-comparable results.
- [x] T029 [US3] Add run summary, case inspection, and comparison HTTP endpoints in `apps/api/internal/evaluation/http/`, with contract tests for evidence separation, metric rationale, safe failure fields, and no prompt/provider-payload exposure.
- [x] T030 [US3] Build maintainer run summary, case expected-versus-actual evidence inspection, and comparison/non-comparable UI in `apps/web/src/features/evaluation/`, with TypeScript tests for accessibility, localized product copy, and immutable identity display.

**Checkpoint**: Comparable runs show explicit deltas and denominators; incompatible identities show
only the reason they cannot be compared.

---

## Phase 7: Polish and cross-cutting verification

**Purpose**: Prove the complete feature follows the approved prototype, contracts, safety rules,
and operational documentation.

- [x] T031 Update `apps/api/README.md`, `apps/agent/README.md`, `apps/ingestion/README.md`, `apps/web/README.md`, `docs/modules/go-api.md`, `docs/modules/python-agent.md`, `docs/modules/python-ingestion.md`, and `docs/modules/web-client.md` with implemented evaluation and opening-suggestion commands, ownership, and diagnostics.
- [x] T032 Add end-to-end fixture coverage across `apps/api/tests/integration/`, `apps/agent/tests/integration/`, and `apps/web/tests/e2e/` for the three isolated corpora, language pairs, stale suggestion sets, source updates, importer idempotency, and run retention.
- [ ] T033 Run the exact validation sequence recorded in `specs/012-evaluation-datasets/quickstart.md`, including Go, Python, TypeScript, migration, persistence, prototype, contract, repository-language, and diff checks; record any bounded operational prerequisite in the quickstart.

---

## Dependencies and execution order

- Phase 1 establishes owned inputs and contracts.
- Phase 2 blocks all user stories because every story depends on corpus isolation, locators, and
  immutable data.
- US0 depends on T001-T013 and is independently demonstrable before the evaluation runner.
- US1 depends on T004-T010 and T019; US2 depends on the same catalog/preflight work; US3 depends
  on a completed US1 run ledger.
- Phase 7 follows all desired user stories.

## Parallel opportunities

- T001-T003 can proceed in parallel because they edit separate assets, contracts, and documents.
- T006 and the contract fixture portion of T002 can proceed in parallel with T004-T005 after the
  migration ownership is agreed.
- T014 can proceed in parallel with T011-T013; T015 can proceed after the contract is stable and
  before the workspace integration T016.
- T020 and the fake-agent portions of T022 can proceed in parallel after the evaluation contract
  is stable; merge them only after T019 and T021 establish persistence and scoring ownership.

## Implementation strategy

1. Deliver the data/catalog foundation and starter suggestions first, because they correct the
   current cross-corpus chat behavior without exposing evaluation answers or changing chat.
2. Add fixed-snapshot evaluation execution and durable results next.
3. Add provenance, maintainer inspection, and comparison after the runner is observable.
4. Finish with full cross-service verification and documentation.
