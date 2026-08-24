#!/bin/sh
set -eu
root=$(cd -- "$(dirname "$0")/.." && pwd)
cd "$root"
set -a
# shellcheck source=/dev/null
. ./.env
set +a
pkill -f 'dist/s3proxy serve' 2>/dev/null || true
sleep 1
# Use setsid to fully detach the proxy from this script's process group so
# that when the script exits, the proxy isn't terminated by SIGHUP.
setsid sh -c './dist/s3proxy serve --config sandbox/integration-config.hcl >dist/s3proxy-integration.log 2>&1 < /dev/null' &
pid=$!
printf '%s\n' "$pid" >dist/s3proxy-integration.pid
echo "started s3proxy (pgid $pid)"
sleep 2
tail -n 3 dist/s3proxy-integration.log
