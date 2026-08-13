---
name: norvii-code-quality
description: Generate, modify, refactor, debug, or review code in the Norvii repository with idiomatic language practices, high cohesion, low coupling, explicit boundaries, maintainable design, and proportionate tests. Use for every task that creates or changes Go, Python, TypeScript, React, SQL, configuration, tests, or build code in this project.
---

# Norvii Code Quality

Apply this workflow to every code change. Favor the simplest design that preserves clear boundaries and can evolve safely.

## Load the relevant references

Read every reference for a language touched by the task before editing:

- Go: [go.md](references/go.md)
- Python: [python.md](references/python.md)
- TypeScript or React: [typescript-react.md](references/typescript-react.md)

For mixed-language work, read all applicable references and treat data exchanged between modules as a public contract.

## Engineering language

Write all project-owned code, configuration, schemas, migrations, tests, comments, docstrings, identifiers, logs, error messages, and technical documentation in English. English is the default product and engineering language.

Portuguese is permitted only in Portuguese localization values, preserved legal or corpus content, quotations and citations, and deterministic test data for those cases. Keep localization keys, surrounding code, test descriptions, and technical explanations in English. Run `.github/scripts/validate_repository_language.py` for every change that can affect scanned text.

## Workflow

### 1. Inspect before designing

- Read the surrounding implementation, tests, configuration, and public contracts.
- Preserve established project conventions unless the task explicitly changes them.
- Identify the module that owns the behavior and keep the change there.
- Check for user changes in the worktree and avoid unrelated edits.

### 2. Define the contract

- State inputs, outputs, errors, side effects, and invariants before choosing abstractions.
- Keep domain rules independent from transport, storage, frameworks, and vendors.
- Validate data at system boundaries and pass valid domain values internally.
- Version contracts shared by Go, Python, and TypeScript when compatibility matters.

### 3. Choose the smallest maintainable design

- Prefer high cohesion and low coupling over layer count or pattern count.
- Keep dependencies directed toward stable domain behavior.
- Introduce an interface, class, generic, factory, or abstraction only when it creates a real seam, enforces an invariant, or removes demonstrated duplication.
- Prefer composition over inheritance and dependency injection over hidden global state.
- Keep functions and methods focused. Split branching code when naming a distinct rule makes the behavior clearer.
- Avoid speculative extension points, pass-through wrappers, generic utility buckets, and premature frameworks.
- Make invalid states difficult to represent where the language supports it.

### 4. Implement explicitly

- Use domain vocabulary in names. Avoid vague names such as `data`, `manager`, `helper`, `processor`, or `utils` without a narrow qualifier.
- Handle errors at the layer that can add context or decide recovery.
- Keep I/O, time, randomness, environment access, and external services injectable at useful boundaries.
- Do not log secrets, document contents, prompts, credentials, or personal data.
- Delete obsolete code created by the change. Do not leave commented-out alternatives.
- Comment decisions, constraints, and non-obvious invariants, not syntax.

### 5. Verify behavior

- Add or update tests for changed behavior, important failure paths, and boundary validation.
- Prefer deterministic tests through public behavior.
- Use integration or contract tests at database, HTTP, queue, filesystem, LLM, and cross-language boundaries.
- Run the narrowest relevant formatter, static analysis, tests, and build commands available in the repository.
- Do not claim a check passed unless it was executed successfully.

### 6. Review the diff

Before handing off, verify:

- the implementation satisfies the requested behavior;
- responsibilities are owned by the correct module;
- no new cyclic dependency or hidden global state was introduced;
- abstractions have more value than indirection cost;
- errors preserve actionable context;
- public contracts and migrations are compatible or explicitly versioned;
- tests would fail for the defect or missing behavior they protect;
- documentation changed when architecture, operation, or public behavior changed.

## Decision rules

- Do not optimize for the fewest lines. Optimize for clarity of behavior and change isolation.
- Do not apply Clean Architecture mechanically. Add boundaries where independent change, testing, or replacement justifies them.
- Do not refactor unrelated code during a scoped feature or fix.
- Do not hide complexity inside generic helpers. Name domain concepts and policies.
- If two modules must share a concept, prefer a small explicit contract over importing internal implementation details.
- If quality and delivery conflict, reduce scope rather than bypass correctness, safety, or verification.

## Handoff

Report the behavior delivered, important design choices, checks executed, and remaining risks. Mention language-specific tradeoffs only when they affect future work.
