# Contributing to Norvii

Norvii is currently a proprietary portfolio project. External code or documentation contributions are accepted only by prior written agreement with the project owner. Issues and review feedback are welcome, but opening an issue does not grant a license to the repository contents.

## Maintainer workflow

All product capabilities use the repository-local Spec Kit workflow. Before changing application behavior:

1. Select or create the numbered feature under `specs/`.
2. Run `speckit-specify` and resolve material ambiguity with `speckit-clarify`.
3. Generate the quality checklist with `speckit-checklist`.
4. Run `speckit-plan`, then `speckit-tasks`.
5. Run `speckit-analyze` and resolve critical findings.
6. Obtain the implementation gate approval. Production UI features require a Verified prototype baseline.
7. Run `speckit-implement`, then `speckit-converge` until required verification passes.

The composed workflow can orchestrate these steps:

```bash
specify workflow run speckit -i spec="Describe the capability and outcome"
```

Repository foundation, documentation-only maintenance, and tool upgrades do not require an artificial product feature. They must still preserve the constitution and documentation ownership rules.

## Code and documentation standards

- Follow `AGENTS.md` and `.specify/memory/constitution.md`.
- Apply the `norvii-code-quality` skill to every code or configuration change.
- Keep module ownership aligned with `docs/architecture/repository-structure.md`.
- Update the durable source of truth in the same change that alters it.
- Never commit secrets, local environment files, generated reports, or dependency directories.
- Write all project-owned source code, configuration, tests, technical documentation, specifications, plans, tasks, comments, identifiers, logs, errors, and commit messages in English.
- Keep Portuguese inside Portuguese localization values, preserved legal or corpus content, quotations and citations, and deterministic fixtures that verify those cases. Localization keys and the surrounding code remain in English.
- Every scaffolded module provides a `Makefile` `ci` target that owns its formatting, analysis, tests, coverage, and build checks.

## Commit history

Each commit must be a complete, reviewable checkpoint. A commit should explain one coherent change and include the tests, documentation, or migration needed to understand and verify it.

Use a short imperative subject with one of these prefixes when useful:

- `docs:` for durable documentation or specifications;
- `feat:` for user-visible capability;
- `fix:` for defect correction;
- `test:` for test-only changes;
- `refactor:` for behavior-preserving restructuring;
- `chore:` for tooling, dependencies, or repository maintenance.

Do not commit broken intermediate generation, unrelated edits, secrets, or build outputs. Spec Kit automatic commits remain disabled so a maintainer can inspect and group each checkpoint intentionally.

## Before committing

Run the verification relevant to every affected module and the feature quickstart. For repository-level documentation and configuration changes, at minimum run:

```bash
specify check
specify integration status
specify workflow resolve speckit
specify preset resolve spec-template
python -m unittest discover -s .github/scripts/tests -v
python .github/scripts/validate_repository_language.py
git diff --check
```

Review `git status --short` and the complete staged diff before creating the commit.

See [continuous integration](docs/development/continuous-integration.md) for the module build contract, SonarQube Cloud setup, and required GitHub checks.
