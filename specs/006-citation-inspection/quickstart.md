# Feature 006 Verification Quickstart

## Prerequisites

1. Configure the local environment as described in the repository root README.
2. Start the complete environment with `make bootstrap`.
3. Ensure both sample corpus sources are in the `ready` state and have existing retrieval chunks.
4. Use a configured model provider only when validating real token-use behavior. Deterministic
   tests must not require external provider access.

## Automated Verification

Run the relevant module checks while implementing:

```bash
make -C apps/api test
make -C apps/agent ci
npm --prefix apps/web run lint
npm --prefix apps/web run typecheck
npm --prefix apps/web test
python .github/scripts/validate_contracts.py
python .github/scripts/validate_repository_language.py
```

Run the affected browser journey after the web and API are available:

```bash
npm --prefix apps/web run test:e2e
```

Use the repository CI command or pull request checks for the complete cross-module and Sonar
validation.

## Manual Acceptance Journey

1. Open a corpus workspace and ask a supported question that returns at least two citations.
2. Select an inline citation. Verify that the source view opens the exact preserved document
   version, scrolls to the visible legal context, and highlights the cited span.
3. Return to Chat. Verify the completed answer is unchanged.
4. Open **Inspect answer**. Verify retrieval order, vector distance when available, retrieval,
   generation, and total timing, plus input/output token values or localized unavailable labels.
5. Select each inspection evidence item and repeat step 2.
6. Reprocess a source, then select a citation from the old answer. Verify it opens the cited
   immutable version when still published, or shows a localized unavailable state without loading
   the new current source text.
7. Repeat in Portuguese UI and verify only product-authored labels change language; the legal
   excerpt and answer retain their original content.
