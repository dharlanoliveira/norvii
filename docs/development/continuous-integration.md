# Continuous Integration

Norvii uses one GitHub Actions workflow for repository checks, module-owned builds, and SonarQube Cloud analysis. The workflow runs on pull requests targeting `main` and on pushes to `main`.

GitHub Actions cannot run on a commit that exists only on a developer machine. A commit is checked after it is pushed directly to `main` or included in a pull request targeting `main`.

## Workflow gates

The [`CI` workflow](../../.github/workflows/ci.yml) executes these gates in order:

1. Repository validation tests the CI support scripts, enforces English in project-owned source and documentation, and validates maintained shell scripts.
2. Module builds invoke the `ci` target owned by every scaffolded application module.
3. Persistence integration starts digest-pinned PostgreSQL and Neo4j, then proves
   initialization, runtime connectivity, restart retention, volume isolation, and
   clean reproduction in an isolated Compose project.
4. SonarQube Cloud independently scans the web, API, and ingestion production modules and waits for every built-in quality gate.
5. A project policy queries the SonarQube Cloud API for each module and fails when any open issue remains in the analyzed main branch or pull request.

The workflow uses read-only repository permissions. External actions are pinned to immutable commit SHAs with their release versions recorded in comments.

## Engineering language gate

The repository language validator scans tracked and unignored project-owned text files. It fails on non-ASCII text or a conservative set of common Portuguese technical terms outside approved content paths. The path-based exceptions are limited to Portuguese resources under `i18n/` or `locales/`, preserved content under `corpora/` or `data/corpora/`, and legal-content fixtures.

The exception applies to content, not engineering vocabulary. Localization keys, source structure, code, comments, test descriptions, and documentation remain in English and require review. Upstream-managed Spec Kit scripts, templates, extensions, and bundled skills are excluded because the project does not own their text. Principle VIII in the constitution is authoritative when the deterministic validator cannot classify a phrase.

## Module CI contract

The workflow monitors these module roots:

| Module | Scaffold marker |
| --- | --- |
| `prototypes/web/` | `package.json` |
| `apps/web/` | `package.json` |
| `apps/api/` | `go.mod` |
| `apps/ingestion/` | `pyproject.toml` |

An unscaffolded module is reported and skipped. Once a scaffold marker exists, the module MUST provide a `Makefile` with a `ci` target. A missing target or non-zero target result fails the build.

The module owns its exact commands and tool versions. Its `ci` target MUST perform all applicable dependency-lock validation, formatting checks, linting, static analysis, tests, coverage generation, and production build or package verification. This keeps the workflow stable while each feature plan selects idiomatic Go, Python, or React tooling.

## SonarQube Cloud Free setup

Use the SonarQube Cloud Free plan for the public `dharlanoliveira/norvii` repository. Sign in with the personal GitHub account, select the `dharlanoliveira` organization, and configure `norvii` as a monorepo using CI-based analysis. Disable automatic analysis so automatic and CI-based scans do not conflict.

The production modules are separate SonarQube Cloud projects bound to the same GitHub repository:

| Module | Project name | Project key variable |
| --- | --- | --- |
| `apps/web/` | `norvii-web` | `SONAR_WEB_PROJECT_KEY` |
| `apps/api/` | `norvii-api` | `SONAR_API_PROJECT_KEY` |
| `apps/ingestion/` | `norvii-ingestion` | `SONAR_INGESTION_PROJECT_KEY` |

The approved prototype is not an independently maintained production module and is therefore excluded from SonarQube Cloud analysis.

The web scanner generates LCOV coverage from the client unit tests. API and ingestion coverage comes from the service-backed persistence job, so PostgreSQL and Neo4j adapters remain part of the measured code rather than being excluded as external infrastructure. The workflow retains those temporary coverage artifacts for one day and downloads each report only into its owning module before analysis. Composition roots and deterministic localization or fixture data may be excluded from coverage or duplication metrics when the metric cannot provide meaningful engineering feedback; production behavior remains in scope.

Create a SonarQube Cloud analysis token and store it as a GitHub Actions secret. Store organization and project identifiers as repository variables:

```bash
gh variable set SONAR_ORGANIZATION --body '<sonar-organization-key>'
gh variable set SONAR_WEB_PROJECT_KEY --body '<web-project-key>'
gh variable set SONAR_API_PROJECT_KEY --body '<api-project-key>'
gh variable set SONAR_INGESTION_PROJECT_KEY --body '<ingestion-project-key>'
gh secret set SONAR_TOKEN
```

`gh secret set` prompts for the token without placing it in shell history. Do not add the token to `sonar-project.properties`, workflow YAML, local environment examples, or documentation.

The Free plan does not allow a project-defined custom quality gate. Norvii therefore applies both controls:

- `sonar.qualitygate.wait=true` makes the scanner fail when the built-in Sonar quality gate fails.
- `.github/scripts/enforce_sonar_issues.py` fails when the Sonar API reports any unresolved issue for the analyzed main branch or pull request.

The second control is intentionally stricter than the built-in quality gate. Resolve the issue in code or update the project policy through a reviewed repository change; do not silently bypass the check.

GitHub does not expose repository secrets to workflows opened from untrusted forks. Sonar analysis is skipped for those fork pull requests; repository and module build checks still run. Same-repository pull requests and pushes to `main` can use the configured Sonar token.

## Required status checks

After the first workflow run creates its GitHub check names, protect `main` with a repository ruleset and require these checks before merge:

- `Repository validation`
- every `Build <module>` matrix check
- `Persistence integration`
- `SonarQube Cloud (web)`
- `SonarQube Cloud (api)`
- `SonarQube Cloud (ingestion)`

## Local verification

Run the support-script tests and maintained shell validation before publishing workflow changes:

```bash
python -m unittest discover -s .github/scripts/tests -v
python .github/scripts/validate_repository_language.py
bash -n .specify/scripts/bash/*.sh .specify/extensions/git/scripts/bash/*.sh
make persistence-integration
```

For every scaffolded module, also run its CI contract:

```bash
make -C <module-path> ci
```

The Sonar scan requires external credentials and is verified by the GitHub-hosted workflow after repository setup.
