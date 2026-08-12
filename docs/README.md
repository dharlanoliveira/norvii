# Norvii Documentation

This directory contains durable project documentation. Feature-specific decisions
belong in the numbered Spec Kit directory that introduced them.

## Source of truth by subject

| Subject | Source of truth | Update trigger |
| --- | --- | --- |
| Product vision, scope, and experience | [Product overview](product/overview.md) | Product direction changes |
| Corpus model and proposed sources | [Legal corpora](product/corpora.md) | Corpus scope or source rules change |
| AI capability scope | [AI capabilities](product/ai-capabilities.md) | RAG, GraphRAG, MCP, or skill scope changes |
| Evaluation strategy | [Evaluation](product/evaluation.md) | Evaluation categories or metrics change |
| Non-negotiable rules | [Constitution](../.specify/memory/constitution.md) | Governance changes |
| Feature order and dependencies | [Feature map](product/feature-map.md) | Roadmap changes |
| Prototype workflow and approval | [Executable prototyping](development/prototyping.md) | Prototype boundary or approval changes |
| Development tools and Spec Kit layers | [Development tooling](development/tooling.md) | Tool version or workflow customization changes |
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

Project documentation currently uses technical English. Public schemas and code identifiers also use English so TypeScript, Go, and Python share one unambiguous vocabulary. User-facing product content may be Portuguese or English according to the active corpus.
