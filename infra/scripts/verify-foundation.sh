#!/usr/bin/env bash

set -euo pipefail

readonly repository_root="$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")/../.." && pwd)"
readonly compose_file="${repository_root}/infra/compose.yaml"
readonly environment_file="$(mktemp)"
readonly runner="${repository_root}/infra/scripts/run-with-environment.py"
readonly postgres_volume="norvii_integration_postgres_data"
readonly neo4j_volume="norvii_integration_neo4j_data"

cleanup() {
  docker compose --env-file "${environment_file}" -f "${compose_file}" \
    down --volumes --remove-orphans >/dev/null 2>&1 || true
  rm -f -- "${environment_file}"
}
trap cleanup EXIT

chmod 600 "${environment_file}"
printf '%s\n' \
  'NORVII_COMPOSE_PROJECT_NAME=norvii-integration' \
  'NORVII_POSTGRES_VOLUME_NAME=norvii_integration_postgres_data' \
  'NORVII_NEO4J_VOLUME_NAME=norvii_integration_neo4j_data' \
  'NORVII_POSTGRES_HOST=localhost' \
  'NORVII_POSTGRES_PORT=15432' \
  'NORVII_POSTGRES_DATABASE=norvii_integration' \
  'NORVII_POSTGRES_USER=norvii_integration' \
  'NORVII_POSTGRES_PASSWORD=integration-postgres-value' \
  'NORVII_NEO4J_HTTP_PORT=17474' \
  'NORVII_NEO4J_BOLT_PORT=17687' \
  'NORVII_NEO4J_URI=neo4j://localhost:17687' \
  'NORVII_NEO4J_USER=neo4j' \
  'NORVII_NEO4J_PASSWORD=integration-neo4j-value' \
  'NORVII_NEO4J_DATABASE=neo4j' \
  'NORVII_PERSISTENCE_TIMEOUT_SECONDS=5' >"${environment_file}"

readonly -a compose_command=(
  docker compose
  --env-file "${environment_file}"
  -f "${compose_file}"
)
readonly -a environment_command=(python "${runner}" "${environment_file}")

echo "Starting isolated persistence integration environment."
"${compose_command[@]}" up --detach --wait --wait-timeout 120
bash "${repository_root}/infra/scripts/inspect-health.sh" "${environment_file}"

echo "Applying initialization twice."
"${environment_command[@]}" go -C "${repository_root}/apps/api" run ./cmd/migrate
"${environment_command[@]}" go -C "${repository_root}/apps/api" run ./cmd/migrate
"${environment_command[@]}" go -C "${repository_root}/apps/api" run ./cmd/migration-status

echo "Running service-backed runtime checks."
"${environment_command[@]}" go -C "${repository_root}/apps/api" test \
  -tags=integration ./tests/integration -count=1
"${environment_command[@]}" make -C "${repository_root}/apps/ingestion" test-integration
"${environment_command[@]}" go -C "${repository_root}/apps/api" run ./cmd/verify-persistence
"${environment_command[@]}" make -C "${repository_root}/apps/ingestion" verify-persistence

echo "Verifying normal restart persistence."
"${compose_command[@]}" exec -T postgres sh -ec \
  'PGPASSWORD="$POSTGRES_PASSWORD" psql --username "$POSTGRES_USER" --dbname "$POSTGRES_DB" --no-psqlrc --command "CREATE TABLE IF NOT EXISTS foundation_marker (id text PRIMARY KEY); INSERT INTO foundation_marker (id) VALUES ('"'"'restart-marker'"'"') ON CONFLICT DO NOTHING;" >/dev/null'
"${compose_command[@]}" exec -T neo4j sh -ec \
  'cypher-shell --username "$NORVII_NEO4J_USER" --password "$NORVII_NEO4J_PASSWORD" --database "$NORVII_NEO4J_DATABASE" '"'"'MERGE (:FoundationMarker {id: "graph-marker"})'"'"'' >/dev/null
"${compose_command[@]}" restart >/dev/null
"${compose_command[@]}" up --detach --wait --wait-timeout 120 >/dev/null

postgres_marker_count="$(
  "${compose_command[@]}" exec -T postgres sh -ec \
    'PGPASSWORD="$POSTGRES_PASSWORD" psql --username "$POSTGRES_USER" --dbname "$POSTGRES_DB" --no-psqlrc --tuples-only --no-align --command "SELECT count(*) FROM foundation_marker WHERE id = '"'"'restart-marker'"'"';"'
)"
neo4j_marker_count="$(
  "${compose_command[@]}" exec -T neo4j sh -ec \
    'cypher-shell --username "$NORVII_NEO4J_USER" --password "$NORVII_NEO4J_PASSWORD" --database "$NORVII_NEO4J_DATABASE" --format plain '"'"'MATCH (marker:FoundationMarker {id: "graph-marker"}) RETURN count(marker) AS count'"'"'' |
    tail -n 1
)"
[[ "${postgres_marker_count}" == "1" ]] || { echo "PostgreSQL marker did not survive restart." >&2; exit 1; }
[[ "${neo4j_marker_count}" == "1" ]] || { echo "Neo4j marker did not survive restart." >&2; exit 1; }

echo "Verifying graph-volume isolation."
"${compose_command[@]}" stop neo4j >/dev/null
"${compose_command[@]}" rm --force neo4j >/dev/null
neo4j_ownership="$(
  docker volume inspect \
    --format '{{ index .Labels "com.docker.compose.project" }} {{ index .Labels "com.docker.compose.volume" }}' \
    "${neo4j_volume}"
)"
[[ "${neo4j_ownership}" == "norvii-integration norvii_neo4j_data" ]] || {
  echo "Integration Neo4j volume ownership is invalid." >&2
  exit 1
}
docker volume rm "${neo4j_volume}" >/dev/null
"${compose_command[@]}" up --detach --wait --wait-timeout 120 neo4j >/dev/null

postgres_marker_count="$(
  "${compose_command[@]}" exec -T postgres sh -ec \
    'PGPASSWORD="$POSTGRES_PASSWORD" psql --username "$POSTGRES_USER" --dbname "$POSTGRES_DB" --no-psqlrc --tuples-only --no-align --command "SELECT count(*) FROM foundation_marker WHERE id = '"'"'restart-marker'"'"';"'
)"
neo4j_marker_count="$(
  "${compose_command[@]}" exec -T neo4j sh -ec \
    'cypher-shell --username "$NORVII_NEO4J_USER" --password "$NORVII_NEO4J_PASSWORD" --database "$NORVII_NEO4J_DATABASE" --format plain '"'"'MATCH (marker:FoundationMarker {id: "graph-marker"}) RETURN count(marker) AS count'"'"'' |
    tail -n 1
)"
[[ "${postgres_marker_count}" == "1" ]] || { echo "Graph reset altered canonical data." >&2; exit 1; }
[[ "${neo4j_marker_count}" == "0" ]] || { echo "Graph reset retained disposable graph data." >&2; exit 1; }

expect_safe_failure() {
  local label="$1"
  shift
  local output_file
  output_file="$(mktemp)"
  if "$@" >"${output_file}" 2>&1; then
    rm -f -- "${output_file}"
    echo "${label} unexpectedly succeeded." >&2
    return 1
  fi
  if grep -F -e 'integration-postgres-value' -e 'integration-neo4j-value' "${output_file}" >/dev/null; then
    rm -f -- "${output_file}"
    echo "${label} disclosed a credential." >&2
    return 1
  fi
  rm -f -- "${output_file}"
}

echo "Verifying bounded and secret-safe failures."
expect_safe_failure "Go invalid credential check" \
  "${environment_command[@]}" env NORVII_POSTGRES_PASSWORD=invalid-credential \
  go -C "${repository_root}/apps/api" run ./cmd/verify-persistence
expect_safe_failure "Python invalid credential check" \
  "${environment_command[@]}" env NORVII_NEO4J_PASSWORD=invalid-credential \
  make -C "${repository_root}/apps/ingestion" verify-persistence
expect_safe_failure "Go unavailable store check" \
  "${environment_command[@]}" env NORVII_POSTGRES_PORT=1 NORVII_PERSISTENCE_TIMEOUT_SECONDS=1 \
  go -C "${repository_root}/apps/api" run ./cmd/verify-persistence
expect_safe_failure "Python unavailable store check" \
  "${environment_command[@]}" env NORVII_POSTGRES_PORT=1 NORVII_PERSISTENCE_TIMEOUT_SECONDS=1 \
  make -C "${repository_root}/apps/ingestion" verify-persistence

echo "Verifying clean-volume reproduction."
"${compose_command[@]}" down --volumes --remove-orphans >/dev/null
if docker volume inspect "${postgres_volume}" "${neo4j_volume}" >/dev/null 2>&1; then
  echo "Integration volumes remained after the isolated reset." >&2
  exit 1
fi
"${compose_command[@]}" up --detach --wait --wait-timeout 120 >/dev/null
"${environment_command[@]}" go -C "${repository_root}/apps/api" run ./cmd/migrate >/dev/null
"${environment_command[@]}" go -C "${repository_root}/apps/api" run ./cmd/verify-persistence >/dev/null
"${environment_command[@]}" make -C "${repository_root}/apps/ingestion" verify-persistence >/dev/null

echo "Persistence foundation integration verification passed."
