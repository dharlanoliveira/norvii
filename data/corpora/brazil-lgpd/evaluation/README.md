# Brazil LGPD Golden Dataset v1

This draft dataset evaluates corpus-grounded answers over the Brazilian LGPD source already
registered by the project. The source language is Brazilian Portuguese. Each Portuguese case
has an equivalent English case so that retrieval and answer generation can be evaluated across
languages without treating translated text as authoritative legal evidence.

## Scope

- Jurisdiction: Brazil, federal law.
- Corpus source: current consolidated text of Law 13,709/2018 hosted by the Chamber of Deputies.
- Snapshot date: 2026-08-25.
- Cases: 18, arranged as nine Portuguese/English semantic pairs.
- Evidence: one official source, cited by article and paragraph or item where applicable.

## Dataset contract

Each JSON Lines record provides a stable identifier, answer language, question, concise
reference answer, evidence requirements, scenario category, and a paired case in the other
query language. A passing generated answer must not add a material legal proposition not
supported by the required evidence and must cite the stated legal location.

The English expected answers are explanatory translations only. The official Portuguese text,
its captured revision, and the cited legal location remain the authoritative evidence.

## Review and maintenance

Before using this dataset as an automated quality gate, review every expected answer against the
document revision that belongs to the evaluated corpus snapshot. Re-review it when the official
law changes or when a different LGPD source set, such as ANPD resolutions or guides, is
published. Questions that depend on sources outside this dataset's manifest must result in
abstention rather than unsupported extrapolation.
