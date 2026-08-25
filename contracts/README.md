# Cross-Language Contracts

This directory is the registry for stable schemas shared across features or by more
than one application module. A feature starts its contract design in
`specs/NNN-feature/contracts/`. Promote a contract here only when its ownership and
compatibility rules are stable enough for reuse.

## Expected contract families

| Family | Likely format | Producer | Consumers |
| --- | --- | --- | --- |
| Public application API | OpenAPI | Go API | Web client, MCP adapters, tests |
| Chat stream parts | JSON Schema or protocol-specific schema | Go API | Web client |
| Ingestion work | JSON Schema | Go API or job dispatcher | Python ingestion |
| Ingestion artifacts | JSON Schema | Python ingestion | Go API and retrieval adapters |
| Evaluation records | JSON Schema | Python evaluation | Go API, web inspection, reports |

Formats remain proposals until feature research confirms them. Do not add an empty
schema or choose a generator outside a feature plan.

## Active contracts

| Family | Version | Producer | Consumers | Location |
| --- | --- | --- | --- | --- |
| Corpus ingestion | v1 | Go API and Python ingestion | React client, Go API, Python ingestion | [`corpus-ingestion/v1`](corpus-ingestion/v1/) |
| Grounded chat stream | Feature-local v1 | Go API | Web client, Python agent | [`specs/005-grounded-rag-chat/contracts`](../specs/005-grounded-rag-chat/contracts/) |

The grounded chat stream remains feature-local. Its schema is not promoted until the Vector and
planned Hybrid compatibility surface has a stable cross-feature owner.

## Contract requirements

Every promoted contract documents:

- owner, producers, and consumers;
- semantic version or explicit compatibility policy;
- required fields, invariants, and stable identifiers;
- errors and partial failure behavior;
- example valid and invalid payloads;
- generated-code commands, if any;
- provider and consumer contract tests;
- deprecation and migration procedure.

Validate all promoted contracts and committed examples with:

```bash
python .github/scripts/validate_contracts.py
```

CI runs this validator before module builds. Go provider tests, Python consumer
tests, and TypeScript response parsers additionally prove the v1 contract at their
runtime boundaries.

Contract names and payload fields use English. Corpus text and user-visible content
retain their original language.
