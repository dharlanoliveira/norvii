<p align="center">
  <img src="assets/brand/norvii-logo.png" alt="Norvii logo" width="180">
</p>

# Norvii

[![CI](https://github.com/dharlanoliveira/norvii/actions/workflows/ci.yml/badge.svg)](https://github.com/dharlanoliveira/norvii/actions/workflows/ci.yml)
[![Web Quality Gate](https://sonarcloud.io/api/project_badges/measure?project=dharlanoliveira_norvii-web&metric=alert_status)](https://sonarcloud.io/dashboard?id=dharlanoliveira_norvii-web)
[![API Quality Gate](https://sonarcloud.io/api/project_badges/measure?project=dharlanoliveira_norvii-api&metric=alert_status)](https://sonarcloud.io/dashboard?id=dharlanoliveira_norvii-api)
[![Ingestion Quality Gate](https://sonarcloud.io/api/project_badges/measure?project=dharlanoliveira_norvii-ingestion&metric=alert_status)](https://sonarcloud.io/dashboard?id=dharlanoliveira_norvii-ingestion)

Norvii is a technical proof of concept for corpus-isolated, evidence-grounded legal
research. A researcher selects a legal corpus, explores its versioned sources, and
asks questions whose answers are constrained to retrieved evidence and traceable
citations. The project demonstrates RAG, GraphRAG, LLM orchestration, MCP research
tools, reusable workflows, and reproducible evaluation assets.

Norvii is a technical demonstration, not legal advice or a substitute for
professional legal research.

## Current repository state

The repository contains an executable local stack: a React client, Go HTTP API,
Python LangGraph agent, Python ingestion worker, PostgreSQL with pgvector, Neo4j,
and a local-only MCP service. The runtime supports corpus and source management,
safe URL and PDF ingestion, immutable document versions and snapshots, graph-ready
releases, corpus-bound retrieval, citations, abstention, technical inspection, and
maintainer evaluation operations.

Model-backed chat and embeddings are optional local configuration. Without their
provider settings, the catalog and source reader remain available, while chat fails
closed rather than producing an ungrounded response. Graph and hybrid retrieval need
an active graph-ready snapshot.

The numbered [feature specifications](specs/README.md) are the authoritative record
of each capability's lifecycle and acceptance evidence; this README summarizes the
currently versioned repository and local environment.

## Corpora and evaluation assets

The local database migrations seed four independently searchable corpora. Source
counts below are the seeded official URL sources, not a claim about their current
ingestion status in any particular local database.

| Seeded corpus | Stable key | Source language | Jurisdiction | Sources |
| --- | --- | --- | --- | ---: |
| Brazilian Personal Data Protection (LGPD) | `brazil-personal-data-protection` | Portuguese | Brazil, federal | 1 |
| European Union General Data Protection Regulation | `initial-gdpr-en` | English | European Union | 1 |
| Brazilian Anti-Corruption and White-Collar Crime | `brazil-anti-corruption-white-collar-crime` | Portuguese | Brazil, federal | 5 |
| United States Fair Housing and Disability Accommodations | `us-fair-housing-disability-accommodations` | English | United States, federal | 5 |

The project-owned evaluation scope is deliberately narrower: it contains three
separate, draft datasets for LGPD, Brazilian anti-corruption, and U.S. fair housing.
Their manifests and cases are repository assets, not automatically published
evaluations or permission to fetch or alter sources.

| Corpus | Dataset | Cases | Dataset assets |
| --- | --- | ---: | --- |
| Brazilian Personal Data Protection (LGPD) | `brazil-lgpd-v1` | 18 | [LGPD dataset](data/corpora/brazil-lgpd/evaluation/README.md) |
| Brazilian Anti-Corruption and White-Collar Crime | `brazil-anti-corruption-v1` | 16 | [Anti-corruption dataset](data/corpora/brazil-anti-corruption/evaluation/README.md) |
| United States Fair Housing and Disability Accommodations | `us-fair-housing-v1` | 18 | [Fair-housing dataset](data/corpora/us-fair-housing-disability-accommodations/evaluation/README.md) |

Every query, source, snapshot, dataset revision, opening suggestion, and evaluation
run is corpus-bound. A draft dataset becomes usable only after review, source binding,
locator resolution, and compatibility with an immutable published snapshot. See
[Legal corpora](docs/product/corpora.md) and the [evaluation strategy](docs/product/evaluation.md).

## Quick start

Prerequisites: Docker Engine with Docker Compose, Node.js 24 and npm, Go 1.26.5 (or a
compatible Go 1.26 patch release), Python 3.13, uv 0.11, GNU Make, and Bash.

Create the ignored local configuration, replace the password markers, and start the
complete local environment:

```bash
cp infra/.env.example infra/.env
# Replace the local PostgreSQL and Neo4j password markers in infra/.env.
make bootstrap
```

`make bootstrap` starts PostgreSQL, Neo4j, the MCP profile, API, agent, ingestion
worker, evaluation worker, and web client. It validates service and process health,
then waits for the two stable initial sources to reach `ready` or `failed` within the
configured bound. Runtime logs are retained in `.log/`; use these commands to inspect
or stop the managed environment without deleting database volumes:

```bash
make local-status
make local-stop
```

The web client normally runs at `http://127.0.0.1:5173`; the local MCP endpoint is
`http://127.0.0.1:8091/mcp`. The detailed [local environment guide](docs/operations/local-environment.md)
covers configuration, health checks, isolated integration, troubleshooting, and the
guarded local reset.

## Work on one module

Use the complete bootstrap for an end-to-end journey. Each production module also
has a focused guide and a module-owned `ci` target:

| Module | Responsibility | Guide |
| --- | --- | --- |
| Web | Bilingual corpus catalog, workspace, citations, and evaluation presentation | [apps/web](apps/web/README.md) |
| API | Public HTTP contracts, PostgreSQL transactions, snapshots, and evaluation ledger | [apps/api](apps/api/README.md) |
| Agent | Corpus-bound retrieval, grounded generation, citation validation, and MCP server | [apps/agent](apps/agent/README.md) |
| Ingestion | Safe acquisition, document processing, embeddings, and graph releases | [apps/ingestion](apps/ingestion/README.md) |

For example, validate the web module with:

```bash
npm --prefix apps/web exec playwright install chromium
make -C apps/web ci
```

The MCP server exposes eight bounded, read-only research tools and three reusable
prompts. Use Docker Streamable HTTP for the managed local service or stdio for local
client development; see the [MCP quickstart](specs/010-mcp-research-tools/quickstart.md).

## Documentation and verification

[Documentation guide](docs/README.md) maps every durable subject to its source of
truth. Useful starting points include:

- [Product overview](docs/product/overview.md) and [AI capabilities](docs/product/ai-capabilities.md)
- [System architecture](docs/architecture/overview.md) and [module models](docs/modules/README.md)
- [Legal corpora](docs/product/corpora.md) and [evaluation strategy](docs/product/evaluation.md)
- [Local environment](docs/operations/local-environment.md) and [continuous integration](docs/development/continuous-integration.md)
- [Spec-driven development](docs/development/spec-driven-development.md) and [feature workspaces](specs/README.md)
- [Project constitution](.specify/memory/constitution.md)

CI checks repository language and contracts, runs each scaffolded module's `ci`
target, exercises persistence integration, and analyzes the web, API, and ingestion
modules with SonarQube Cloud. Run the verification relevant to the files you change;
for a repository documentation change, start with:

```bash
python -m unittest discover -s .github/scripts/tests -v
python .github/scripts/validate_repository_language.py
python .github/scripts/validate_contracts.py
git diff --check
```

## Contributing

Norvii is a proprietary portfolio project. External code or documentation
contributions require prior written agreement with the project owner. See
[CONTRIBUTING.md](CONTRIBUTING.md) for the Spec Kit workflow, engineering standards,
and commit expectations.

## License

Copyright (c) 2026 Dharlan Oliveira. Norvii is proprietary and available for
personal evaluation and portfolio review only. See [LICENSE](LICENSE) for the full
terms and [third-party notices](THIRD_PARTY_NOTICES.md) for separately licensed
components.
