#!/usr/bin/env bash
set -euo pipefail

root=$(cd "$(dirname "$0")/.." && pwd)
project=${SANDBOX_PROJECT_NAME:-s3proxy-sandbox}
if [[ ! "$project" =~ ^s3proxy-[a-z0-9][a-z0-9_-]*$ ]]; then
  echo "SANDBOX_PROJECT_NAME must start with s3proxy- and use lowercase letters, digits, underscores or hyphens" >&2
  exit 1
fi

if test -n "${SANDBOX_COMPOSE:-}"; then
  compose=("$SANDBOX_COMPOSE")
elif docker compose version >/dev/null 2>&1; then
  compose=(docker compose)
elif command -v docker-compose >/dev/null 2>&1; then
  compose=(docker-compose)
else
  echo "Docker Compose is required (docker compose or docker-compose)" >&2
  exit 1
fi

exec "${compose[@]}" --project-name "$project" --env-file "$root/.env" -f "$root/sandbox/docker-compose.yml" "$@"
