# Norvii Agent Instructions

## Code quality skill

For every task that creates, modifies, refactors, debugs, or reviews code, load and follow the project skill at `.agents/skills/norvii-code-quality/SKILL.md` before editing.

Read the language reference for every language touched by the task:

- Go: `.agents/skills/norvii-code-quality/references/go.md`
- Python: `.agents/skills/norvii-code-quality/references/python.md`
- TypeScript or React: `.agents/skills/norvii-code-quality/references/typescript-react.md`

Apply the skill to tests, migrations, build files, configuration, and generated code as well as application code.

## Spec-driven development

Before implementing a product capability, create or select its numbered workspace under `specs/` and follow the repository-local Spec Kit flow:

1. `speckit-specify`
2. `speckit-clarify` when material ambiguity remains
3. `speckit-checklist`
4. `speckit-plan`
5. `speckit-tasks`
6. `speckit-analyze`
7. Obtain the implementation approval gate
8. `speckit-implement`
9. `speckit-converge`

Treat `.specify/memory/constitution.md` as non-negotiable project guidance. Use `docs/README.md` to find the global source of truth for each subject and do not implement application code directly from a roadmap entry.

## Prototype gate

Build the initial executable product prototype only under `prototypes/web/` and follow `docs/development/prototyping.md`. Do not create production application code under `apps/` until the prototype feature is `Verified` and its approval gate passes. Production modules must not import prototype code, fixtures, or styles.


<!-- SPECKIT START -->
For additional context about technologies to be used, project structure,
shell commands, and other important information, read the current plan
<!-- SPECKIT END -->
