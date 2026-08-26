#!/usr/bin/env bash

set -euo pipefail

readonly environment_file="${1:-}"
readonly confirmation="${2:-}"
readonly expected_confirmation="reset-norvii-data"
readonly expected_preflight="passed"
readonly expected_project="norvii"
readonly postgres_volume="norvii_postgres_data"
readonly neo4j_volume="norvii_neo4j_data"
readonly script_directory="$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd)"
readonly compose_file="${script_directory%/scripts}/compose.yaml"

if [[ "${confirmation}" != "${expected_confirmation}" ]]; then
  echo "Reset requires explicit confirmation: ${expected_confirmation}." >&2
  exit 1
fi
if [[ "${NORVII_ASSERTION_RESET_PREFLIGHT:-}" != "${expected_preflight}" ]]; then
  echo "Reset requires the normative assertion preflight to pass." >&2
  exit 1
fi
if [[ -z "${environment_file}" || ! -f "${environment_file}" ]]; then
  echo "A readable persistence environment file is required." >&2
  exit 1
fi

readonly -a compose_command=(
  docker compose
  --env-file "${environment_file}"
  -f "${compose_file}"
)

project_name="$(
  "${compose_command[@]}" config --format json |
    python -c 'import json, sys; print(json.load(sys.stdin).get("name", ""))'
)"
if [[ "${project_name}" != "${expected_project}" ]]; then
  echo "Reset refused because the Compose project identity is not ${expected_project}." >&2
  exit 1
fi

"${compose_command[@]}" down --remove-orphans

owned_volume_output="$(
  docker volume ls \
    --filter "label=com.docker.compose.project=${expected_project}" \
    --format '{{.Name}}'
)"
mapfile -t owned_volumes <<<"${owned_volume_output}"
if [[ "${owned_volume_output}" == "" || ${#owned_volumes[@]} -ne 2 ]]; then
  echo "Reset refused because the project has missing or unexpected persistence volumes." >&2
  exit 1
fi

for expected_volume in "${postgres_volume}" "${neo4j_volume}"; do
  if [[ " ${owned_volumes[*]} " != *" ${expected_volume} "* ]]; then
    echo "Reset refused because the project has missing or unexpected persistence volumes." >&2
    exit 1
  fi
  ownership="$(
    docker volume inspect \
      --format '{{ index .Labels "com.docker.compose.project" }} {{ index .Labels "com.docker.compose.volume" }}' \
      "${expected_volume}"
  )"
  if [[ "${ownership}" != "${expected_project} ${expected_volume}" ]]; then
    echo "Reset refused because ${expected_volume} ownership is invalid." >&2
    exit 1
  fi
done

docker volume rm norvii_postgres_data norvii_neo4j_data >/dev/null
echo "Norvii local PostgreSQL and Neo4j data was removed and cannot be recovered."
