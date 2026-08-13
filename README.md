<p align="center">
  <img src="assets/brand/norvii-logo.png" alt="Norvii logo" width="180">
</p>

# Norvii

[![CI](https://github.com/dharlanoliveira/norvii/actions/workflows/ci.yml/badge.svg)](https://github.com/dharlanoliveira/norvii/actions/workflows/ci.yml)

Norvii is a proof of concept for evidence-grounded legal research using RAG,
GraphRAG, LLMs, MCP, and reusable skills. The application will let a user select a
small Portuguese or English corpus, browse its sources, and ask questions in a chat
whose answers include traceable citations.

The project is a technical demonstration and does not provide legal advice.

## Project status

The first production React client is available under `apps/web/`. It implements the approved bilingual corpus catalog and research workspace with production-owned deterministic data, source browsing, simulated structured chat, citations, abstention, and failure states.

The local persistence foundation is also available. It runs PostgreSQL with pgvector
as canonical storage and standalone Neo4j Community as a rebuildable graph
projection, applies the initial vector migration, and verifies both stores through
independent Go and Python production drivers. The current React client remains
independently runnable and does not yet call these modules.

## Run the production client

Prerequisites: Node.js 24 and npm.

```bash
npm --prefix apps/web ci
npm --prefix apps/web run dev
```

Open the local address printed by Vite. Validate the complete module with:

```bash
npm --prefix apps/web exec playwright install chromium
make -C apps/web ci
```

See the [production web client guide](apps/web/README.md) for browser tests, supported behavior, and current limitations.

## Run the persistence foundation

Prerequisites: Docker Compose, Go 1.26, Python 3.13, uv, Make, and Bash.

```bash
cp infra/.env.example infra/.env
# Replace both password markers in infra/.env.
make persistence-up
make persistence-migrate
make persistence-verify
```

See the [local environment guide](docs/operations/local-environment.md) for health,
troubleshooting, isolated integration, non-destructive stop, and guarded reset
commands.

## Code quality analysis

GitHub Actions is prepared to analyze Norvii with SonarQube Cloud after the public project is imported and its repository variables and analysis token are configured. Each same-repository pull request and push to `main` waits for the Sonar quality gate, then applies the stricter Norvii policy that fails the build when any unresolved Sonar issue remains.

View the [Norvii analysis dashboard on SonarQube Cloud](https://sonarcloud.io/project/overview?id=dharlanoliveira_norvii). The dashboard will become available after the initial SonarQube Cloud setup and first successful analysis. See [continuous integration](docs/development/continuous-integration.md#sonarqube-cloud-free-setup) for the setup procedure, required GitHub configuration, and gate behavior.

## Documentation map

- [Product overview](docs/product/overview.md): vision, scope, product behavior, and MVP boundary.
- [Legal corpora](docs/product/corpora.md): source model and proposed Portuguese and English collections.
- [Documentation guide](docs/README.md): ownership and location of every document
  type.
- [Feature map](docs/product/feature-map.md): proposed delivery sequence.
- [Spec-driven workflow](docs/development/spec-driven-development.md): how Codex and
  contributors move from capability to verified code.
- [Development tooling](docs/development/tooling.md): pinned Spec Kit version, project preset, workflow overlay, and upgrade rules.
- [Continuous integration](docs/development/continuous-integration.md): GitHub Actions builds, SonarQube Cloud gates, and failure notifications.
- [Local environment](docs/operations/local-environment.md): PostgreSQL, Neo4j, migration, verification, and guarded reset commands.
- [Executable prototyping](docs/development/prototyping.md): how React prototypes are built, reviewed, and approved before production.
- [Prototype workspace](prototypes/README.md): executable product experiments kept separate from production applications.
- [Production web client](apps/web/README.md): local startup, validation commands, and current client boundaries.
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
