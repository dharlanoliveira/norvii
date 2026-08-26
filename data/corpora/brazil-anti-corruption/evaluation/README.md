# Brazil Anti-Corruption Golden Dataset v1

This draft dataset evaluates grounded retrieval over a future Brazilian anti-corruption
corpus. It contains paired Portuguese and English questions. The authoritative evidence is
always the Portuguese official source; English expected answers are informational renderings
of the same supported proposition and do not translate or replace the cited legal text.

## Scope

- Jurisdiction: Brazil, federal law and federal administrative guidance.
- Snapshot date: 2026-08-25.
- Cases: 16, arranged as eight Portuguese/English semantic pairs.
- Evidence: official Planalto, Coaf, and CGU pages only.
- Answer type: concise, corpus-grounded legal information; not legal advice.

## Dataset contract

Each JSON Lines record contains:

- `id`: stable test-case identifier;
- `language`: language expected for the generated answer;
- `question`: researcher question;
- `expected_answer`: the reference proposition, intentionally limited to the cited evidence;
- `expected_evidence`: source identifiers and required legal locators;
- `category`: evaluation scenario class;
- `paired_item_id`: equivalent case in the other query language;
- `source_language`: language of the authoritative evidence.

A generated answer passes only when it states no material proposition beyond the supported
reference answer and cites the required source location. Matching prose is not required.

## Review and maintenance

Before this dataset is used as a release gate, a legal-domain reviewer must validate each
reference answer against the exact captured document revision. Re-run that review whenever a
source is reingested, amended, revoked, or superseded. The dataset deliberately excludes
case-specific allegations, news reporting, and unverified enforcement facts.
