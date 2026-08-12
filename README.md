<p align="center">
  <img src="assets/brand/norvii-logo.png" alt="Norvii logo" width="180">
</p>

# Norvii

Norvii is a proof of concept for evidence-grounded legal research using RAG,
GraphRAG, LLMs, MCP, and reusable skills. The application will let a user select a
small Portuguese or English corpus, browse its sources, and ask questions in a chat
whose answers include traceable citations.

The project is a technical demonstration and does not provide legal advice.

## Project status

The repository is in the specification and product-prototyping phase. Production application modules have not been scaffolded yet. The first feature will create an executable React prototype with deterministic data before production implementation begins. The database, vector index, graph storage, models, and ingestion trigger remain intentionally undecided and will be selected through feature research and architecture decisions.

## Documentation map

- [Product overview](docs/product/overview.md): vision, scope, product behavior, and MVP boundary.
- [Legal corpora](docs/product/corpora.md): source model and proposed Portuguese and English collections.
- [Documentation guide](docs/README.md): ownership and location of every document
  type.
- [Feature map](docs/product/feature-map.md): proposed delivery sequence.
- [Spec-driven workflow](docs/development/spec-driven-development.md): how Codex and
  contributors move from capability to verified code.
- [Development tooling](docs/development/tooling.md): pinned Spec Kit version, project preset, workflow overlay, and upgrade rules.
- [Executable prototyping](docs/development/prototyping.md): how React prototypes are built, reviewed, and approved before production.
- [Prototype workspace](prototypes/README.md): executable product experiments kept separate from production applications.
- [Repository structure](docs/architecture/repository-structure.md): target monorepo
  boundaries.
- [Module models](docs/modules/README.md): client, Go API, and Python ingestion
  responsibilities.
- [Constitution](.specify/memory/constitution.md): non-negotiable engineering and
  product rules.
- [Feature specifications](specs/README.md): numbered Spec Kit workspaces.

## Spec Kit workflow

For each capability, use the repository-local Spec Kit skills in this order:

1. `speckit-specify`
2. `speckit-clarify` when material ambiguity remains
3. `speckit-checklist`
4. `speckit-plan`
5. `speckit-tasks`
6. `speckit-analyze`
7. implementation approval gate
8. `speckit-implement`
9. `speckit-converge`

Do not start application code from the roadmap alone. Create or select the numbered
feature that owns the behavior first.

## License

Copyright (c) 2026 Dharlan Oliveira. Norvii is proprietary and available for personal evaluation and portfolio review only. See [LICENSE](LICENSE) for the full terms and [third-party notices](THIRD_PARTY_NOTICES.md) for separately licensed components.
