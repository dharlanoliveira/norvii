# U.S. Fair Housing Golden Dataset v1

This draft dataset evaluates a federal U.S. Fair Housing corpus focused on disability
accommodations and accessible housing. It combines statutory text, agency regulation, official
guidance, operational complaint information, and a guidance PDF. This is a legal-research
evaluation set, not individualized legal advice.

## Scope

- Jurisdiction: United States, federal law only.
- Snapshot date: 2026-08-25.
- Cases: 18, arranged as nine English/Portuguese semantic pairs.
- Primary demonstration language: English.
- Authoritative source language: English.
- Exclusions: state and local law, individual eligibility determinations, and factual findings
  about any named landlord, tenant, or housing provider.

## Dataset contract

Each JSON Lines record contains a stable case identifier, expected answer language, question,
concise reference answer, source-location requirements, scenario category, and an equivalent
case in the other supported query language. A passing answer must cite the indicated statutory
or official location and must not treat HUD or DOJ guidance as a replacement for the Fair
Housing Act or applicable regulation.

Portuguese records are cross-language retrieval tests. Their expected answers translate the
supported English proposition for evaluation purposes; English legal text and its captured
revision remain authoritative.

## Review and maintenance

Before the dataset becomes a release gate, capture every listed source in one corpus snapshot
and have a qualified reviewer verify all expected answers against those exact revisions. Update
the manifest and re-review the cases when the U.S. Code, regulation, or agency guidance changes.
The `eCFR` is a continuously updated online source and must therefore record its displayed
version date and capture hash in the corpus manifest.
