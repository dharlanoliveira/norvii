# Evaluation Strategy

Norvii demonstrates reproducible corpus-grounded behavior rather than selected manual examples.
An evaluation preserves the dataset revision, compatible corpus snapshot, expected evidence,
retrieval configuration, safe execution outcome, and measurement inputs that produced a result.
It is a technical measurement and does not provide legal advice or a legal conclusion about a real
person or entity.

## Versioned corpus datasets

The initial project-owned datasets remain isolated by corpus. They do not authorize retrieval,
evaluation, or comparison across corpus boundaries.

| Corpus | Stable corpus identifier | Dataset revision input |
| --- | --- | --- |
| Brazilian Personal Data Protection (LGPD) | `brazil-personal-data-protection` | `brazil-lgpd-v1` |
| Brazilian Anti-Corruption and White-Collar Crime | `brazil-anti-corruption-white-collar-crime` | `brazil-anti-corruption-v1` |
| United States Fair Housing and Disability Accommodations | `us-fair-housing-disability-accommodations` | `us-fair-housing-v1` |

Every dataset contains paired Portuguese and English questions, source requirements, expected
legal evidence, and immutable case identities. The original corpus content and expected evidence
are maintainer-facing evaluation inputs; they are not exposed as ordinary chat suggestions.

## Publication and compatibility

An imported revision remains draft until reviewed and marked available. Before a run or a
starter-question projection can use it, the intended corpus must have an immutable published
snapshot in which every manifest source is explicitly bound and every required legal locator
resolves uniquely. A missing source, unresolved location, changed snapshot manifest, wrong corpus,
or unavailable revision prevents use before a model call.

Runs retain their selected historical snapshot and do not follow a later active release. Direct
quality deltas are valid only when the compared runs have the same dataset content, snapshot
manifest, ordered case set, and scoring policy. Other comparisons remain visible but are labeled
non-comparable.

## Opening suggestions

A reviewed available revision can designate no more than five reciprocal Portuguese and English
case pairs as starter questions. Publication creates an append-only projection for one compatible
corpus snapshot. The workspace can show only the active corpus's exact language-matched questions
in rank order; it hides the list when the projection is absent, stale, disabled, or incompatible.

The projection contains only the selected question text and its safe identity fields. It never
contains reference answers, expected evidence, scoring, evaluation outcomes, provider data, or
other evaluation internals. Selecting a question uses the ordinary corpus-bound chat request and
does not start an evaluation or change chat streaming.

## Question categories

- direct answer from one provision;
- answer distributed across two documents;
- exception, condition, or deadline;
- multi-hop relation;
- question with no answer in the corpus;
- ambiguous question;
- question written in a language different from the source; and
- attempt to mix jurisdictions or corpora.

## Metrics

- expected-evidence retrieval coverage;
- citation coverage and validity;
- correct abstention rate when reviewed abstention cases exist;
- quality on multi-hop questions;
- end-to-end latency;
- tokens consumed per answer; and
- comparison across vector RAG, GraphRAG, and hybrid retrieval.

Claim support that requires legal or factual judgment remains subject to human review. Approval
thresholds remain an [open decision](../decisions/backlog.md) until the first versioned evaluation
set exists.
