# Quickstart: Planned Hybrid Retrieval

## Prerequisites

- A configured local Norvii environment with an active source in each seeded corpus.
- A configured OpenAI-compatible chat provider for generated answers and Hybrid planning.
- A ready graph release is required only to observe graph contribution; Vector and Hybrid remain usable without one.

Start the environment from the repository root:

```sh
make bootstrap
```

Open `http://127.0.0.1:5173`.

## Validate Vector

1. Open either corpus.
2. Select **Vector**.
3. Ask a broad question, such as `What is the purpose of this document?`.
4. Open the research record.

Expected result: the response is cited when semantic evidence is sufficient. The record identifies the active snapshot and a completed Vector stage. It contains no planning or graph contribution.

## Validate Hybrid with no graph contribution

1. Select **Hybrid**.
2. Ask the same broad question.
3. Open the research record.

Expected result: Vector evidence remains available. The record shows whether graph planning skips the graph or a graph query finds no supported path; neither state is reported as a request failure.

## Validate Hybrid with graph contribution

1. Select **Hybrid** in a corpus with a ready graph release.
2. Ask a seeded relationship-focused question, such as `Which rights are connected to an access request?`.
3. Open the research record and select a graph-path citation.

Expected result: the record lists Vector, planning, and Graph stages; graph evidence has immutable citation locations; selecting a location opens the exact preserved legal unit.

## Validate graph unavailability

Use a historical snapshot without a graph release, or temporarily run the approved failure scenario. Submit a Hybrid question with vector evidence.

Expected result: the answer remains vector-backed. The record labels the graph stage unavailable with a safe diagnostic. It does not show Graph as a selectable strategy.

## Automated verification

Run the relevant module checks from the repository root:

```sh
make agent-check
make api-check
make web-check
python3 .github/scripts/validate_repository_language.py
```
