# Repository Structure

Norvii is a monorepo with three independently testable application modules and an
explicit contract boundary between them. Directories are created by the feature that
first needs them; this tree is the target, not a request to scaffold empty code.

```text
norvii/
|-- .agents/                 # Codex project skills
|-- .specify/                # Spec Kit workflow, templates, and constitution
|-- apps/
|   |-- web/                 # Production React, TypeScript, Vite SPA
|   |-- api/                 # Production Go online backend
|   `-- ingestion/           # Production Python offline pipeline
|-- prototypes/
|   `-- web/                 # Executable React product prototype
|-- contracts/               # Stable cross-language schemas and compatibility policy
|-- docs/                    # Durable product, architecture, and operation documents
|-- infra/
|   `-- compose.yaml         # Local backing services after their selection
|-- specs/                   # Numbered feature workspaces
|-- assets/                  # Brand and documentation assets
|-- AGENTS.md                # Codex repository guidance
`-- README.md                # Project entry point
```

## Dependency direction

```text
apps/web ------ public HTTP and streaming contracts ------> apps/api
                                                             |
                                                             | artifact and job contracts
                                                             v
                                                     apps/ingestion

prototypes/web ------ deterministic fixtures only ------> no production runtime

all modules ------ versioned schemas ------> contracts/
all runtimes ------ local dependencies ----> infra/compose.yaml
```

- The web client depends on public API behavior, never Go packages.
- The Go API consumes published ingestion artifacts, never Python modules.
- The Python service reads source work and publishes artifacts through explicit
  persistence or job contracts, never Go internals.
- Production applications do not import code, fixtures, or styles from prototypes.
- A database schema is owned by the module that writes it. Shared records require a
  documented contract and migration ownership.

## Feature impact

Every feature plan includes a module impact table. A module marked `No change` MUST
not receive opportunistic edits. Cross-cutting work belongs to the feature that first
needs it and must remain narrowly scoped.

| Area | Typical feature impact |
| --- | --- |
| `apps/web/` | Routes, screens, components, chat rendering, client adapters |
| `apps/api/` | Domain behavior, use cases, HTTP or streaming adapters |
| `apps/ingestion/` | Acquisition, extraction, transformation, publication |
| `prototypes/web/` | Product journeys, UI states, fixtures, stories, and visual baselines |
| `contracts/` | Stable schemas shared by two or more modules or features |
| `infra/` | Backing services, health checks, volumes, local configuration |
| `docs/` | Durable rules or decisions only |
| `specs/NNN-feature/` | All reasoning and acceptance evidence local to the feature |

## Dependency management

Each application module owns its dependency files, tests, formatting, static
analysis, and build entry points. Root automation may orchestrate module commands but
MUST NOT hide module-specific failure output or couple their dependency graphs.

Dependencies are introduced by the feature that uses them. The feature plan records
the need, the research artifact compares meaningful options when a choice is not
obvious, and an ADR records difficult-to-reverse selections.
