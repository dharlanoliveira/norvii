# Evaluation Strategy

Norvii must demonstrate reproducible answer quality, not only selected manual examples. The evaluation feature will version questions, expected evidence, corpus snapshots, configuration, results, and cost measurements.

## Question categories

- direct answer from one provision;
- answer distributed across two documents;
- exception, condition, or deadline;
- multi-hop relation;
- question with no answer in the corpus;
- ambiguous question;
- question written in a language different from the source;
- attempt to mix jurisdictions or corpora.

## Candidate Portuguese questions

- Uma startup e obrigada a indicar encarregado?
- Quando um incidente deve ser comunicado a ANPD?
- Quais direitos podem ser exercidos pelo titular?
- Qual e a relacao entre a LGPD e as regras para agentes de pequeno porte?

## Candidate English questions

- When must a controller report a personal data breach?
- What is the difference between a controller and a processor?
- Which GDPR provisions support the answer?
- What obligations are connected to a data subject access request?

## Metrics

- relevant-document retrieval accuracy;
- relevant-provision retrieval accuracy;
- citation coverage;
- claim support by cited evidence;
- correct abstention rate;
- quality on multi-hop questions;
- end-to-end latency;
- tokens consumed per answer;
- comparison across vector RAG, GraphRAG, and hybrid retrieval.

Approval thresholds remain an [open decision](../decisions/backlog.md) until the first versioned evaluation set exists.
