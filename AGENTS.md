# Norvii Agent Instructions

## Code quality skill

For every task that creates, modifies, refactors, debugs, or reviews code, load and follow the project skill at `.agents/skills/norvii-code-quality/SKILL.md` before editing.

Read the language reference for every language touched by the task:

- Go: `.agents/skills/norvii-code-quality/references/go.md`
- Python: `.agents/skills/norvii-code-quality/references/python.md`
- TypeScript or React: `.agents/skills/norvii-code-quality/references/typescript-react.md`

Apply the skill to tests, migrations, build files, configuration, and generated code as well as application code.

## Engineering language

Write all project-owned source code, configuration, tests, technical documentation, specifications, plans, tasks, comments, identifiers, logs, and error messages in English. English is the default product and engineering language.

Portuguese is limited to Portuguese localization values, preserved legal or corpus content, quotations and citations, and deterministic test data for those cases. Keep localization keys, surrounding code, test descriptions, and technical explanations in English. Follow Principle VIII of `.specify/memory/constitution.md` and run `.github/scripts/validate_repository_language.py` before committing.

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


<codex_multiagents_v2>

Apply this section only when the runtime provides subagent creation with
`gpt-5.6-sol`, `gpt-5.6-terra`, and `gpt-5.6-luna`. In any other agent or
runtime, ignore this section entirely.

## Model roles

The primary agent uses `gpt-5.6-sol` with `high` effort exclusively for
orchestration: understand the request, divide work, delegate, validate results,
maintain safety confirmations, and synthesize the final response. It must not
directly perform delegable implementation or routine operational tasks.

For an independent, well-bounded, low-complexity task, delegate one worker by
default with:

- model: `gpt-5.6-luna`;
- reasoning effort: `high`.

For an independent medium- or high-complexity task that can still be solved by
one worker, for example multi-source technical investigation, implementation
across multiple files, or evidence-based diagnosis, delegate one worker with:

- model: `gpt-5.6-terra`;
- reasoning effort: `high`.

The worker receives only the user request, project directory, applicable skill,
and expected result. Do not send the entire conversation history, secrets,
credentials, tokens, or configuration-file contents.

Do not delegate a simple response, a clarification question, a single reading
without analysis, an architecture decision, a destructive operation, or a task
whose next step immediately depends on a primary-agent decision. Do not use
workers in parallel unless the user explicitly requests it or subtasks are
genuinely independent and have no shared state.

The primary agent remains responsible for validation, user confirmation,
result communication, and decisions the worker cannot resolve. If Luna finds
material ambiguity, semantic conflict, production risk, or a need for broad
judgment, do not persist with the same delegation: the primary agent routes it
to a Terra High worker or decides the orchestration issue without turning Sol
into an executor.

</codex_multiagents_v2>
