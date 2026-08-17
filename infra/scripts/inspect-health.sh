#!/usr/bin/env bash

set -euo pipefail

readonly environment_file="${1:-}"
readonly script_directory="$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd)"
readonly compose_file="${script_directory%/scripts}/compose.yaml"

if [[ -z "${environment_file}" || ! -f "${environment_file}" ]]; then
  echo "A readable persistence environment file is required." >&2
  exit 1
fi

readonly -a compose_command=(
  docker compose
  --env-file "${environment_file}"
  -f "${compose_file}"
)

inspect_service() {
  local service_name="$1"
  local display_name="$2"
  local health

  if ! health="$("${compose_command[@]}" ps --format '{{.Health}}' "${service_name}")"; then
    echo "${display_name} health could not be inspected." >&2
    echo "Inspect with: docker compose --env-file ${environment_file} -f ${compose_file} logs --tail 100 ${service_name}" >&2
    return 1
  fi

  if [[ "${health}" != "healthy" ]]; then
    echo "${display_name} is ${health:-not running}." >&2
    echo "Inspect with: docker compose --env-file ${environment_file} -f ${compose_file} logs --tail 100 ${service_name}" >&2
    return 1
  fi

  echo "${display_name} is healthy."
}

status=0
inspect_service postgres PostgreSQL || status=1
inspect_service neo4j Neo4j || status=1
exit "${status}"
