# Research: Versioned Evaluation Datasets

## Decision 1: Corpus separation and names

**Decision**: Create three new, isolated legal corpora:

| Stable key | Product name | Jurisdiction | Dataset asset |
| --- | --- | --- | --- |
| `brazil-personal-data-protection` | Brazilian Personal Data Protection (LGPD) | Brazil, federal | `data/corpora/brazil-lgpd/evaluation/` |
| `brazil-anti-corruption-white-collar-crime` | Brazilian Anti-Corruption and White-Collar Crime | Brazil, federal | `data/corpora/brazil-anti-corruption/evaluation/` |
| `us-fair-housing-disability-accommodations` | United States Fair Housing and Disability Accommodations | United States, federal | `data/corpora/us-fair-housing-disability-accommodations/evaluation/` |

**Rationale**: A corpus is a retrieval and citation boundary, not a topical label. The sources, authorities, jurisdictions, snapshots, and legal questions differ. This makes each showcase meaningful and prevents accidental evidence leakage.

**Rejected alternative**: Add legal sources or datasets to an information-security corpus. It would produce misleading retrieval, erase a useful legal boundary, and violates the feature's explicit scope.

## Decision 2: Treat local JSON/JSONL as immutable owned inputs

**Decision**: Keep the initial manifests and JSONL files under `data/corpora/*/evaluation/` and import them with a deterministic local command. Store canonical asset hashes and input paths.

**Rationale**: Dataset import is neither source acquisition nor legal-document extraction. It must work without network/model access and be reproducible from a clean checkout.

**Rejected alternative**: Fetch the manifest URLs or generate cases during import. Both would make an evaluation revision mutable and non-reproducible.

## Decision 3: Bind dataset source aliases to source UUIDs

**Decision**: Bind each manifest alias to an intended `sources.id` in its named corpus at publication. Retain title, official URL, document type, issuing authority, and authority role as provenance, but never match a runtime source by title or URL.

**Rationale**: The initial LGPD seed uses Planalto while the golden manifest identifies Camara; anti-corruption and U.S. fair-housing sources are not seeded in the current baseline. URL/title matching could silently attach the wrong version or corpus.

**Rejected alternative**: Use `source_id` aliases as `sources.seed_key` or compare official URLs at run time. Aliases and seed keys have different lifecycles, and origins can legitimately change without changing source identity.

## Decision 4: Resolve legal locators before execution

**Decision**: Add canonical legal locator aliases to document-unit provenance and resolve every expected `(source alias, legal locator)` to immutable snapshot members before a run is queued. Persist original display locator plus resolved source, document version, legal unit/span, and content hash in the run.

**Rationale**: Golden cases contain legally meaningful citations such as Article 5, items VI and VII, and 42 U.S.C. section 3604(f)(3)(B). Current extraction can expose structural labels such as `article-1` or `block-1`, which are not comparable strings. Compound citation selectors expand to atomic targets and require all targets unless a future reviewed schema explicitly declares an alternative.

**Rejected alternative**: Fuzzy text matching or direct locator-string comparison. Both can create false-positive compatibility and lose reproducibility after a source is re-extracted.

## Decision 5: Evaluate a fixed snapshot through a dedicated agent port

**Decision**: The API's evaluation capability invokes an agent execution port with the stored corpus ID and historical snapshot ID. The normal chat application service is not reused.

**Rationale**: The chat application service resolves and overwrites the caller's snapshot with the active release, while the Python agent retrieval layer already accepts explicit snapshots. Evaluation must remain runnable after a later snapshot becomes active.

**Rejected alternative**: Use an active-snapshot chat request for every case. It would make old run results unreproducible and permit release drift mid-evaluation.

## Decision 6: Database-backed leased case work

**Decision**: Create all case ledger rows when a run is started and process them through leased background work. The API records the run; the worker processes case attempts and persists results.

**Rationale**: A dataset contains multiple remote model calls. Request-scoped execution risks HTTP timeouts and cannot safely recover from one case failure. Leasing supports retry/recovery without duplicate terminal results.

**Rejected alternative**: Synchronous API loop or a new queue service. The former is brittle; the latter adds infrastructure without POC value.

## Decision 7: Deterministic v1 scoring, with semantic review explicit

**Decision**: Score deterministic evidence behavior: expected-evidence retrieval/citation coverage, citation scope validity, citation marker validity, expected abstention, execution state, and telemetry. Store `needs_human_review` for semantic claim support rather than infer it from reference-answer similarity.

**Rationale**: The current answer text contains citation markers but has no structured mapping from each material claim to an atomic evidence span. The asset's `required_propositions` are review annotations, not machine-verifiable generated-answer claims. A judge model would introduce uncontrolled cost and a second unverifiable model.

**Rejected alternative**: LLM-as-judge or lexical overlap as a release gate. Neither establishes legal entailment, particularly across Portuguese and English.

## Decision 8: Preserve retrieved evidence separately from actual citations

**Decision**: Save complete retrieved evidence and a separately parsed set of evidence actually cited through `[n]` markers. Evaluate expected-evidence retrieval and citation coverage separately.

**Rationale**: An item may be retrieved but not cited. Calling it cited would overstate grounding. Citation validity requires marker resolution and membership in the stored run snapshot.

## Decision 9: Strict comparison key and honest denominators

**Decision**: A direct comparison requires equal immutable dataset revision/hash, snapshot ID and manifest hash, ordered full case set, and scoring-policy version. Configuration/model changes are permitted and displayed. Failed/cancelled cases are unscored, not zero.

**Rationale**: This keeps a model/retrieval experiment useful without pretending that different law versions, source sets, or benchmarks measure the same thing.

## Open implementation input to capture during tasks

The data assets currently lack reviewed `expected_outcome: abstain` cases. The schema supports it, but initial reports must display abstention correctness as `not_applicable` until each legal corpus receives a legally reviewed abstention fixture. This is a data-quality prerequisite, not a reason to fabricate a metric.
