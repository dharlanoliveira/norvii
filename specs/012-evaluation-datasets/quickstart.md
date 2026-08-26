# Quickstart: Versioned Evaluation Datasets

This guide defines the expected verification flow after implementation. It does not authorize external source acquisition; use the existing source-ingestion procedure for official documents.

## 1. Prepare the local environment

1. Start the standard Norvii local environment and apply API migrations.
2. Create exactly these three legal corpora, never an information-security corpus:
   - Brazilian Personal Data Protection (LGPD): `brazil-personal-data-protection`;
   - Brazilian Anti-Corruption and White-Collar Crime:
     `brazil-anti-corruption-white-collar-crime`; and
   - United States Fair Housing and Disability Accommodations:
     `us-fair-housing-disability-accommodations`.
3. Process the reviewed official sources for each corpus and publish an immutable snapshot. Before
   an evaluation revision or opening-suggestion projection is eligible, verify all of the
   following:
   - every manifest source has an explicit source binding in the same corpus and is a member of
     the snapshot;
   - every required legal locator resolves uniquely in the snapshot; and
   - the dataset revision is reviewed and available.

A dataset remains draft when a source is missing, a locator is unresolved, or its review state is
not available. Never substitute a source, snapshot, corpus, or approximate locator to make a
dataset eligible.

## 2. Import and publish assets

Run the implemented local importer against the three project-owned directories:

```text
data/corpora/brazil-lgpd/evaluation/
data/corpora/brazil-anti-corruption/evaluation/
data/corpora/us-fair-housing-disability-accommodations/evaluation/
```

Verify that the importer reports all 52 cases, source requirements, original `pt-BR`/`en` language values, and reciprocal pairs. Re-run without changes and verify the same immutable revision is returned. Review source bindings and legal-locator aliases, then create an `available` publication record only after every expected target resolves in the intended snapshot.

For each reviewed available revision, verify that starter-case selection has at most five complete
reciprocal ranks. Publish an opening-suggestion projection only for its compatible corpus snapshot.
The projection must contain only rank, case ID, checksum, language, and the original question
text; it must not contain answers, expected evidence, review notes, scores, run outcomes, or other
evaluation internals.

Open an empty chat in each corpus in both interface languages. Where the active snapshot has a
matching projection, it must show the exact rank-ordered question for that corpus and interface
language. Where no matching projection exists, it must show no suggestion list. It must never
translate, synthesize, randomize, or fall back to LGPD questions in another corpus.

## 3. Verify preflight safety

Attempt to start each dataset with:

- an information-security corpus;
- another named legal corpus;
- a snapshot missing a manifest source;
- a snapshot containing the source but lacking a required legal locator; and
- an unpublished/draft dataset revision.

Each request must return bounded missing requirements and create no run/case work or model call.

## 4. Run and inspect

Start a run for one available dataset and its selected historical snapshot. Confirm that:

1. A ledger row is created for every case before execution.
2. Every result retains dataset hash, corpus/snapshot IDs, configuration fingerprint, agent/model identity, safe answer/telemetry fields, and separate retrieved/cited evidence.
3. A forced provider failure leaves that case `failed` and unscored while other cases reach a terminal state.
4. A later active snapshot cannot enter the in-progress historical run.

## 5. Verify scoring and comparison

Use test fixtures to cover right source/right location, right source/wrong location, retrieved but not cited evidence, invalid marker, cross-snapshot evidence, correct/incorrect abstention, and missing telemetry. Verify honest metric denominators. Compare two runs with the same strict key and different configurations; compare a different snapshot or revision and verify the UI/API returns `non_comparable` with no quality delta.

## 6. Required automated checks

Run the relevant Go, Python, and TypeScript test targets; migration/persistence verification; the prototype approval checks; and:

```bash
python3 .github/scripts/validate_repository_language.py
git diff --check
```

Add the exact implemented commands to this guide during the implementation task. Keep the technical-measurement and not-legal-advice notice visible in the maintainer experience.

Evaluation results and opening suggestions are technical information about corpus-grounded system
behavior. They do not provide legal advice or a legal conclusion about a real person or entity.
