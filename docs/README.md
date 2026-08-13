# Norvii Documentation

This directory contains durable project documentation. Feature-specific decisions
belong in the numbered Spec Kit directory that introduced them.

## Source of truth by subject

| Subject | Source of truth | Update trigger |
| --- | --- | --- |
| Product vision, scope, and experience | [Product overview](product/overview.md) | Product direction changes |
| Interface language behavior | [Product overview](product/overview.md#interface-languages) | Supported locale or default-language behavior changes |
| Corpus model and proposed sources | [Legal corpora](product/corpora.md) | Corpus scope or source rules change |
| AI capability scope | [AI capabilities](product/ai-capabilities.md) | RAG, GraphRAG, MCP, or skill scope changes |
| Evaluation strategy | [Evaluation](product/evaluation.md) | Evaluation categories or metrics change |
| Non-negotiable rules | [Constitution](../.specify/memory/constitution.md) | Governance changes |
| Feature order and dependencies | [Feature map](product/feature-map.md) | Roadmap changes |
| Prototype workflow and approval | [Executable prototyping](development/prototyping.md) | Prototype boundary or approval changes |
| Development tools and Spec Kit layers | [Development tooling](development/tooling.md) | Tool version or workflow customization changes |
| Continuous integration and quality gates | [Continuous integration](development/continuous-integration.md) | CI gate, Sonar, build, or notification behavior changes |
| Repository and module ownership | [Repository structure](architecture/repository-structure.md) | Durable boundary changes |
| System flows and retrieval model | [System architecture](architecture/overview.md) | Cross-module flow changes |
| Cross-feature risks | [Architecture risks](architecture/risks.md) | Risk or mitigation direction changes |
| Client model | [Web client](modules/web-client.md) | Client responsibility changes |
| Online backend model | [Go API](modules/go-api.md) | API responsibility changes |
| Offline processing model | [Python ingestion](modules/python-ingestion.md) | Pipeline responsibility changes |
| Public data exchange | [`contracts/`](../contracts/README.md) | Cross-language contract changes |
| Local backing services | [Local environment](operations/local-environment.md) | Runtime dependency changes |
| Architectural choices | [Decision records](decisions/README.md) | A consequential choice is accepted |
| Unresolved technical choices | [Decision backlog](decisions/backlog.md) | Research opens or resolves a choice |
| One capability | [`specs/`](../specs/README.md) | Feature creation or implementation |
| Executable product experiments | [`prototypes/`](../prototypes/README.md) | Prototype creation or boundary changes |
| Brand identity assets | [`assets/brand/`](../assets/brand/README.md) | Canonical visual asset changes |

## Documentation rules

- Link to a source of truth instead of copying it into another document.
- Keep product specifications technology-agnostic; record implementation choices in
  the feature plan and research artifacts.
- Keep temporary reasoning inside the feature directory. Promote only durable rules
  to global documentation.
- Record a decision as an ADR when it changes module boundaries, persistence,
  protocols, deployment, security posture, or a difficult-to-reverse dependency.
- Update documentation in the same feature that changes the described behavior.
- Do not maintain a second global plan. The product overview states direction, the feature map states sequence, and numbered feature workspaces own implementation detail.

## Languages

English is the mandatory language for all project-owned source code and technical documentation, including specifications, plans, tasks, ADRs, runbooks, configuration, schemas, migrations, tests, comments, docstrings, logs, errors, identifiers, file names, directory names, and commit messages. This gives TypeScript, Go, and Python one shared engineering vocabulary.

Portuguese is permitted only in Portuguese localization values, preserved legal or corpus content, quotations and citations, and deterministic fixtures or test literals that verify those cases. Localization keys, surrounding code, test descriptions, and technical explanations remain in English. The product interface supports English and Portuguese independently from corpus and engineering language, with English as the default.

The repository validation job runs `.github/scripts/validate_repository_language.py`. It detects non-ASCII text and a conservative set of Portuguese technical terms outside approved content paths. This deterministic check supplements code review; it does not attempt unreliable natural-language classification.
