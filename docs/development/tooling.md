# Development Tooling

This document records repository-wide tools that shape the development workflow. Feature-specific dependencies belong in the owning feature plan.

## Spec Kit

Norvii is initialized with GitHub Spec Kit `0.16.2` and the Codex integration. Install the pinned version with `uv`:

```bash
uv tool install specify-cli --force --from git+https://github.com/github/spec-kit.git@v0.16.2
specify --version
```

The repository keeps upstream-managed files separate from Norvii policy:

| Layer | Location | Ownership |
| --- | --- | --- |
| Spec Kit core templates and workflow | `.specify/templates/`, `.specify/workflows/speckit/workflow.yml` | Upstream-managed; do not edit directly |
| Norvii template policy | `.specify/presets/norvii/` | Project-owned compositional preset |
| Norvii workflow gates | `.specify/workflows/overlays/speckit/` | Project-owned workflow overlay |
| Project constitution | `.specify/memory/constitution.md` | Project-owned non-negotiable guidance |
| Agent code quality rules | `.agents/skills/norvii-code-quality/` | Project-owned implementation guidance |

The Norvii preset appends architecture, prototype, evidence, cost, and task-quality requirements to the upstream templates. The workflow overlay adds clarification, checklist, analysis, implementation approval, and convergence to the bundled workflow.

Validate the effective layers after installation or an upgrade:

```bash
specify integration status
specify preset info norvii
specify preset resolve spec-template
specify preset resolve plan-template
specify preset resolve tasks-template
specify workflow resolve speckit
```

Upgrade only after reviewing the upstream release and backing up repository state:

```bash
specify integration upgrade codex
specify extension update
```

Do not use `--force` when the integration status reports modified managed files until those changes have been migrated to a project preset, overlay, or another supported project-owned layer.

## Git integration

The Spec Kit Git extension creates feature branches and validates their naming. Automatic commits are disabled in `.specify/extensions/git/git-config.yml`; maintainers inspect and create complete logical commits themselves.

## Module toolchains

Go, Python, React, database, and container tool versions remain intentionally unset until the feature that scaffolds each module researches and records them. Once selected, pin versions in executable toolchain files and record only repository-wide consequences here.
