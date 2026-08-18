# Norvii API

The Go API owns synchronous corpus and source commands, authoritative reads, public
HTTP contracts, PostgreSQL transactions, migrations, and safe error envelopes. It
does not acquire remote content, parse PDFs, call models, or write the Neo4j
projection.

## Run locally

Use `make bootstrap` from the repository root for the complete application. For a
focused API process with `infra/.env` already configured and PostgreSQL running:

```bash
make persistence-migrate
python infra/scripts/run-with-environment.py infra/.env \
  go -C apps/api run ./cmd/server
```

The default listener is `127.0.0.1:8080`; verify it with
`curl http://127.0.0.1:8080/healthz`. Configuration keys and bounded defaults are
documented in `infra/.env.example`.

## Quality and contracts

```bash
make -C apps/api ci
make persistence-integration
python .github/scripts/validate_contracts.py
```

The module contract runs dependency integrity, Go formatting, vet, race tests, and
build. Service-backed integration tests exercise migrations, transactions,
optimistic concurrency, duplicates, isolation, PDF delivery, and ingestion
publication. The public OpenAPI contract is
`contracts/corpus-ingestion/v1/openapi.json`.

HTTP failures use stable English codes and safe messages with a request identifier.
Structured access logs record request ID, method, path, status, duration, and response
size without query parameters. Logs and error responses must not expose database
credentials, PDF bytes, complete documents, or URL user information.
