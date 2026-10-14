#!/usr/bin/env bash
set -euo pipefail

root=$(cd "$(dirname "$0")/.." && pwd)
tmp=$(mktemp -d)
trap 'rm -rf "$tmp"' EXIT
mkdir -p "$tmp/repo/scripts" "$tmp/repo/sandbox" "$tmp/bin" "$tmp/state"
cp "$root/Makefile" "$tmp/repo/"
cp "$root/scripts/run-integration.sh" "$root/scripts/sandbox-compose.sh" "$tmp/repo/scripts/"
cp "$root/sandbox/docker-compose.yml" "$tmp/repo/sandbox/"
printf 'COMPOSE_PROJECT_NAME=sandbox\nSANDBOX_PROJECT_NAME=s3proxy-dotenv\nSANDBOX_COMPOSE=/must-not-run\nTEST_CREDENTIAL=loaded\n' >"$tmp/repo/.env"

cat >"$tmp/bin/fake-compose" <<'EOF'
#!/usr/bin/env bash
set -euo pipefail
project=sandbox
while test "$#" -gt 0; do
  case "$1" in
    --project-name|-p) project=$2; shift 2 ;;
    --env-file) test "$2" = "$FAKE_ROOT/.env"; shift 2 ;;
    -f) test "$2" = "$FAKE_ROOT/sandbox/docker-compose.yml"; shift 2 ;;
    *) break ;;
  esac
done
printf '%s %s\n' "$project" "$*" >>"$FAKE_STATE/compose.log"
case "$1" in
  up)
    touch "$FAKE_STATE/$project-stack"
    if [[ " $* " == *' --remove-orphans '* ]]; then
      rm -f "$FAKE_STATE/$project-unrelated"
    fi
    exit "${FAKE_UP_RC:-0}"
    ;;
  down)
    rm -f "$FAKE_STATE/$project-stack"
    if [[ " $* " == *' --remove-orphans '* ]]; then
      rm -f "$FAKE_STATE/$project-unrelated"
    fi
    exit "${FAKE_DOWN_RC:-0}"
    ;;
  ps)
    case "$*" in
      'ps -a -q minio-init seaweedfs-init') printf '%s\n' "$project-minio-init" "$project-seaweedfs-init" ;;
      'ps -q s3-error') printf '%s\n' "$project-s3-error" ;;
      ps) ;;
      *) exit 92 ;;
    esac
    ;;
  logs|config) ;;
  *) exit 93 ;;
esac
EOF

cat >"$tmp/bin/docker" <<'EOF'
#!/usr/bin/env bash
set -euo pipefail
case "$1" in
  compose)
    shift
    if test "$1" = version; then exit "$FAKE_PLUGIN_RC"; fi
    test "$FAKE_PLUGIN_RC" = 0
    echo plugin >>"$FAKE_STATE/frontend.log"
    exec fake-compose "$@"
    ;;
  wait)
    [[ "$2" == "$EXPECTED_PROJECT-"* ]]
    printf '%s\n' "${FAKE_INIT_RC:-0}"
    ;;
  inspect)
    test "${*: -1}" = "$EXPECTED_PROJECT-s3-error"
    echo healthy
    ;;
  logs) ;;
  *) echo "unexpected docker call: $*" >&2; exit 94 ;;
esac
EOF

cat >"$tmp/bin/docker-compose" <<'EOF'
#!/usr/bin/env bash
set -euo pipefail
echo standalone >>"$FAKE_STATE/frontend.log"
exec fake-compose "$@"
EOF

cat >"$tmp/bin/go" <<'EOF'
#!/usr/bin/env bash
set -euo pipefail
case "$1" in
  build)
    test "${FAKE_BUILD_RC:-0}" = 0 || exit "$FAKE_BUILD_RC"
    cp "$FAKE_PROXY" dist/s3proxy
    ;;
  test)
    test "$TEST_CREDENTIAL" = loaded
    test -f dist/s3proxy-integration.pid
    cp dist/s3proxy-integration.pid "$FAKE_STATE/proxy.pid"
    exit "${FAKE_TEST_RC:-0}"
    ;;
  *) exit 95 ;;
esac
EOF

cat >"$tmp/bin/proxy" <<'EOF'
#!/usr/bin/env bash
set -euo pipefail
test "$TEST_CREDENTIAL" = loaded
exec sleep 300
EOF

cat >"$tmp/bin/curl" <<'EOF'
#!/usr/bin/env bash
echo 500
EOF
chmod +x "$tmp/bin/"*

export PATH="$tmp/bin:$PATH" FAKE_ROOT="$tmp/repo" FAKE_STATE="$tmp/state" FAKE_PROXY="$tmp/bin/proxy"
export COMPOSE_PROJECT_NAME=sandbox
unset SANDBOX_PROJECT_NAME SANDBOX_COMPOSE
unset MAKEFLAGS MFLAGS MAKELEVEL

assert_isolated() {
  local project args
  test -s "$FAKE_STATE/compose.log"
  while read -r project args; do
    test "$project" = "$EXPECTED_PROJECT"
    [[ " $args " != *' --remove-orphans '* ]]
  done <"$FAKE_STATE/compose.log"
  test -f "$FAKE_STATE/sandbox-unrelated"
  test -f "$FAKE_STATE/other-unrelated"
  test -f "$FAKE_STATE/$EXPECTED_PROJECT-unrelated"
}

reset_state() {
  rm -f "$FAKE_STATE/"*
  touch "$FAKE_STATE/sandbox-unrelated" "$FAKE_STATE/other-unrelated" "$FAKE_STATE/$EXPECTED_PROJECT-unrelated"
}

for frontend in plugin standalone override; do
  export FAKE_PLUGIN_RC=0
  unset SANDBOX_COMPOSE
  case "$frontend" in
    standalone) export FAKE_PLUGIN_RC=1 ;;
    override) export SANDBOX_COMPOSE="$tmp/bin/fake-compose" ;;
  esac
  for project in s3proxy-sandbox s3proxy-regression; do
    export EXPECTED_PROJECT="$project"
    unset SANDBOX_PROJECT_NAME
    make_args=()
    if test "$project" != s3proxy-sandbox; then
      make_args+=("SANDBOX_PROJECT_NAME=$project")
    fi
    reset_state
    for target in sandbox-up sandbox-ps sandbox-logs sandbox-logs-follow sandbox-down sandbox-destroy sandbox-reset sandbox-integration-down; do
      make -s -C "$FAKE_ROOT" VERSION=test "${make_args[@]}" DAEMON=true "$target"
    done
    assert_isolated
    test ! -e "$FAKE_STATE/$project-stack"
    if test "$frontend" != override; then
      test "$(sort -u "$FAKE_STATE/frontend.log")" = "$frontend"
    fi

    if test "$project" != s3proxy-sandbox; then export SANDBOX_PROJECT_NAME="$project"; fi
    for scenario in success tests-fail startup-fail init-fail build-fail cleanup-fail; do
      reset_state
      export FAKE_TEST_RC=0 FAKE_UP_RC=0 FAKE_INIT_RC=0 FAKE_BUILD_RC=0 FAKE_DOWN_RC=0
      expected_rc=0
      case "$scenario" in
        tests-fail) export FAKE_TEST_RC=37; expected_rc=37 ;;
        startup-fail) export FAKE_UP_RC=23; expected_rc=23 ;;
        init-fail) export FAKE_INIT_RC=19; expected_rc=1 ;;
        build-fail) export FAKE_BUILD_RC=17; expected_rc=2 ;;
        cleanup-fail) export FAKE_TEST_RC=37 FAKE_DOWN_RC=29; expected_rc=37 ;;
      esac
      rc=0
      bash "$FAKE_ROOT/scripts/run-integration.sh" >"$tmp/run.log" 2>&1 || rc=$?
      if test "$rc" != "$expected_rc"; then
        cat "$tmp/run.log" >&2
        echo "$frontend/$project/$scenario: expected $expected_rc, got $rc" >&2
        exit 1
      fi
      assert_isolated
      test "$(tail -n 1 "$FAKE_STATE/compose.log")" = "$project down"
      test ! -e "$FAKE_STATE/$project-stack"
      test ! -e "$FAKE_ROOT/dist/s3proxy-integration.pid"
      if test -f "$FAKE_STATE/proxy.pid"; then
        if kill -0 "$(cat "$FAKE_STATE/proxy.pid")" 2>/dev/null; then
          echo 'integration proxy survived cleanup' >&2
          exit 1
        fi
      fi
    done
    export FAKE_TEST_RC=37 FAKE_UP_RC=0 FAKE_INIT_RC=0 FAKE_BUILD_RC=0 FAKE_DOWN_RC=0
    reset_state
    rc=0
    make -s -C "$FAKE_ROOT" VERSION=test "${make_args[@]}" sandbox-integration-up >"$tmp/run.log" 2>&1 || rc=$?
    test "$rc" = 2
    grep -q 'Error 37' "$tmp/run.log"
    assert_isolated
    test ! -e "$FAKE_STATE/$project-stack"
  done
done

for project in sandbox unrelated s3proxy- 's3proxy-bad name'; do
  reset_state
  if SANDBOX_PROJECT_NAME="$project" bash "$FAKE_ROOT/scripts/sandbox-compose.sh" up >"$tmp/run.log" 2>&1; then
    echo "accepted invalid project: $project" >&2
    exit 1
  fi
  test ! -e "$FAKE_STATE/compose.log"
done

echo 'sandbox lifecycle tests passed (3 Compose frontends, 2 projects, 6 integration outcomes; Make failure propagation and invalid projects)'
