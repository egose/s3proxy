#!/usr/bin/env bash
set -euo pipefail

root=$(cd "$(dirname "$0")/.." && pwd)
cd "$root"

test -f .env || { echo "missing .env (see .env.example)" >&2; exit 1; }
mkdir -p dist
compose=(env "SANDBOX_PROJECT_NAME=${SANDBOX_PROJECT_NAME:-}" "SANDBOX_COMPOSE=${SANDBOX_COMPOSE:-}" bash "$root/scripts/sandbox-compose.sh")
proxy_pid=

cleanup() {
  rc=$?
  trap - EXIT INT TERM
  if test -n "$proxy_pid"; then
    kill "$proxy_pid" 2>/dev/null || true
    wait "$proxy_pid" 2>/dev/null || true
    rm -f dist/s3proxy-integration.pid
  fi
  "${compose[@]}" down || true
  exit "$rc"
}
trap cleanup EXIT
trap 'exit 130' INT TERM

"${compose[@]}" up --build -d

ids=$("${compose[@]}" ps -a -q minio-init seaweedfs-init)
test -n "$ids" || { echo "sandbox init containers not found" >&2; exit 1; }
for id in $ids; do
  code=$(docker wait "$id")
  if test "$code" != 0; then
    docker logs "$id"
    exit 1
  fi
done

error_id=$("${compose[@]}" ps -q s3-error)
test -n "$error_id" || { echo "s3-error container not found" >&2; exit 1; }
for _ in $(seq 1 30); do
  test "$(docker inspect -f '{{.State.Health.Status}}' "$error_id")" != healthy || break
  sleep 1
done
test "$(docker inspect -f '{{.State.Health.Status}}' "$error_id")" = healthy || { docker logs "$error_id"; exit 1; }
test "$(curl -sS -o /dev/null -w '%{http_code}' http://127.0.0.1:18081/)" = 500 || {
  echo "s3-error did not return HTTP 500" >&2
  exit 1
}

make build
set -a
# shellcheck source=/dev/null
. ./.env
set +a
./dist/s3proxy serve --config "$root/sandbox/integration-config.hcl" >dist/s3proxy-integration.log 2>&1 &
proxy_pid=$!
printf '%s\n' "$proxy_pid" >dist/s3proxy-integration.pid
sleep 1
kill -0 "$proxy_pid" 2>/dev/null || { cat dist/s3proxy-integration.log; exit 1; }

go test -tags integration -count=1 -v ./internal/integration/...
