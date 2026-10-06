#!/usr/bin/env bash
set -euo pipefail

root=$(cd "$(dirname "$0")/.." && pwd)
tmp=$(mktemp -d)
trap 'rm -rf "$tmp"' EXIT
plugin="$tmp/plugin with spaces"
mkdir -p "$plugin" "$tmp/fakes" "$tmp/outside cwd" "$tmp/release" "$tmp/payload"
cp -R "$root/bin" "$root/scripts" "$plugin/"
original_path=$PATH
failures=0
checks=0
check() {
  checks=$((checks + 1))
  if ! "$@"; then
    printf 'FAIL: %s\n' "$*" >&2
    failures=$((failures + 1))
  fi
}

valid=(
  0.0.0
  1.0.0-alpha 1.0.0-alpha.1 1.0.0-alpha.beta
  1.0.0-beta 1.0.0-beta.2 1.0.0-beta.11 1.0.0-rc.1 1.0.0
  1.2.3-0 1.2.3-0.0 1.2.3-0.1 1.2.3-1 1.2.3-2 1.2.3-10
  1.2.3-999999999999999999999999999999 1.2.3-1000000000000000000000000000000
  1.2.3-- 1.2.3-01a 1.2.3-A 1.2.3-Z 1.2.3-a
  1.2.3-alpha 1.2.3-alpha+Build.02 1.2.3-alpha+build.1
  1.2.3-alpha.0 1.2.3-alpha.0.1 1.2.3-alpha.1 1.2.3-alpha.a
  1.2.3-rc.1 1.2.3-rc.1+build.4 1.2.3-rc.2 1.2.3-rc.10
  1.2.3-rc.9007199254740992 1.2.3-rc.9007199254740993
  1.2.3 1.2.3+0 1.2.3+01 1.2.3+A 1.2.3+a 1.2.3+build.10 1.2.3+build.2
  2.0.0 3.0.0-rc.1+build.4 3.0.0
  9.0.0 10.0.0 10.0.9007199254740992 10.0.9007199254740993
  10.9007199254740992.0 10.9007199254740993.0
  9007199254740992.0.0 9007199254740993.0.0
  999999999999999999999999999999.0.0 1000000000000000000000000000000.0.0
)
invalid=(
  '' 1 1.2 1.2.3.4 01.2.3 1.02.3 1.2.03 +1.2.3 -1.2.3
  1.2.3- 1.2.3+ 1.2.3-.a 1.2.3-a. 1.2.3-a..b 1.2.3-00
  1.2.3-01 1.2.3-a.01 1.2.3-000000000000000000000000000001
  1.2.3+.a 1.2.3+a. 1.2.3+a..b 1.2.3-a+ 1.2.3-a++b
  1.2.3_a 1.2.3-a_b 1.2.3+a_b '1.2.3 a' ' 1.2.3' '1.2.3 '
  '1.2.3-*' '1.2.3-é' '1.2.3/evil' v1.2.3 vv1.2.3
  $'1.2.3\n2.0.0' $'1.2.3\r' $'1.2.3\t'
)
expected=$(IFS=' '; printf '%s' "${valid[*]}")
refs="$tmp/refs"
: >"$refs"
for ((i=${#valid[@]} - 1; i >= 0; i--)); do
  printf 'a\trefs/tags/v%s\nb\trefs/tags/%s\nc\trefs/tags/v%s\n' "${valid[i]}" "${valid[i]}" "${valid[i]}" >>"$refs"
done
for version in "${invalid[@]}"; do
  printf 'd\trefs/tags/v%s\n' "$version" >>"$refs"
done

cat >"$tmp/fakes/git" <<'EOF'
#!/usr/bin/env bash
set -euo pipefail
[[ "$*" == "ls-remote --tags --refs https://github.com/${EXPECTED_REPOSITORY}.git" ]] || exit 91
printf 'called\n' >>"$FIXTURE/git.calls"
cat "$REFS"
if [[ ${GIT_FAIL:-0} == 1 ]]; then
  printf 'injected git failure\n' >&2
  exit 73
fi
EOF
cat >"$tmp/fakes/curl" <<'EOF'
#!/usr/bin/env bash
set -euo pipefail
url= output=
while (( $# )); do
  case "$1" in
    -o) output=$2; shift 2 ;;
    https:*) url=$1; shift ;;
    *) shift ;;
  esac
done
printf '%s\n' "$url" >>"$FIXTURE/curl.calls"
[[ $output == "$FIXTURE/"* ]] || exit 92
base="https://github.com/${EXPECTED_REPOSITORY}/releases/download/v${ASDF_INSTALL_VERSION}"
case "$url" in
  "$base/s3proxy-linux-amd64.tar.gz"|"$base/SHA256SUMS") cp "$FIXTURE/release/${url##*/}" "$output" ;;
  *) exit 93 ;;
esac
EOF
cat >"$tmp/fakes/uname" <<'EOF'
#!/bin/sh
case "$1" in
  -s) printf 'Linux\n' ;;
  -m) printf 'x86_64\n' ;;
  *) exit 94 ;;
esac
EOF
chmod +x "$tmp/fakes/"*
export FIXTURE="$tmp" REFS="$refs" EXPECTED_REPOSITORY=fixture-owner/fixture.repo
export ASDF_S3PROXY_GITHUB_REPOSITORY="$EXPECTED_REPOSITORY"
export PATH="$tmp/fakes:$original_path"
cd "$tmp/outside cwd"

for locale in C C.utf8; do
  if ! locale -a | LC_ALL=C grep -ixq "$locale"; then
    continue
  fi
  status=0
  LC_ALL="$locale" "$plugin/bin/list-all" >"$tmp/list" 2>"$tmp/error" || status=$?
  check test "$status" = 0
  check test "$(cat "$tmp/list")" = "$expected"
done
LC_ALL=C sort "$refs" >"$tmp/reordered"
status=0
REFS="$tmp/reordered" "$plugin/bin/list-all" >"$tmp/list" 2>"$tmp/error" || status=$?
check test "$status" = 0
check test "$(cat "$tmp/list")" = "$expected"
for version in "${invalid[@]}"; do
  if [[ $version == *$'\n'* ]]; then
    continue
  fi
  printf 'a\trefs/tags/v%s\n' "$version" >"$tmp/invalid ref"
  status=0
  REFS="$tmp/invalid ref" "$plugin/bin/list-all" >"$tmp/list" 2>"$tmp/error" || status=$?
  check test "$status" = 0
  check test -z "$(cat "$tmp/list")"
done
status=0
GIT_FAIL=1 "$plugin/bin/list-all" >"$tmp/list" 2>"$tmp/error" || status=$?
check test "$status" = 73
check test ! -s "$tmp/list"
check grep -q 'injected git failure' "$tmp/error"
printf 'a\trefs/tags/not-a-version\n' >"$tmp/empty"
status=0
REFS="$tmp/empty" "$plugin/bin/list-all" >"$tmp/list" 2>"$tmp/error" || status=$?
check test "$status" = 0
check test -z "$(cat "$tmp/list")"
: >"$tmp/empty"
status=0
REFS="$tmp/empty" "$plugin/bin/list-all" >"$tmp/list" 2>"$tmp/error" || status=$?
check test "$status" = 0
check test -z "$(cat "$tmp/list")"
status=0
GIT_FAIL=1 REFS="$tmp/empty" "$plugin/bin/list-all" >"$tmp/list" 2>"$tmp/error" || status=$?
check test "$status" = 73
check test ! -s "$tmp/list"
check grep -q 'injected git failure' "$tmp/error"
: >"$tmp/git.calls"
status=0
ASDF_S3PROXY_GITHUB_REPOSITORY='bad repo/name' "$plugin/bin/list-all" >"$tmp/list" 2>"$tmp/error" || status=$?
check test "$status" -ne 0
check test ! -s "$tmp/list"
check test ! -s "$tmp/git.calls"
status=0
env -u ASDF_S3PROXY_GITHUB_REPOSITORY EXPECTED_REPOSITORY=egose/s3proxy "$plugin/bin/list-all" >"$tmp/list" 2>"$tmp/error" || status=$?
check test "$status" = 0
check test "$(cat "$tmp/list")" = "$expected"

printf '#!/bin/sh\nprintf "fixture binary works\\n"\n' >"$tmp/payload/s3proxy"
tar -C "$tmp/payload" -czf "$tmp/release/s3proxy-linux-amd64.tar.gz" s3proxy
if command -v sha256sum >/dev/null 2>&1; then
  digest=$(sha256sum "$tmp/release/s3proxy-linux-amd64.tar.gz")
else
  digest=$(shasum -a 256 "$tmp/release/s3proxy-linux-amd64.tar.gz")
fi
printf '%s  s3proxy-linux-amd64.tar.gz\n' "${digest%% *}" >"$tmp/release/SHA256SUMS"
for version in "${valid[@]}"; do
  export ASDF_INSTALL_VERSION="$version" ASDF_INSTALL_TYPE=version
  export ASDF_DOWNLOAD_PATH="$tmp/download $version" ASDF_INSTALL_PATH="$tmp/install $version"
  : >"$tmp/curl.calls"
  status=0
  "$plugin/bin/download" >"$tmp/output" 2>"$tmp/error" || status=$?
  check test "$status" = 0
  base="https://github.com/$EXPECTED_REPOSITORY/releases/download/v$version"
  printf '%s\n' "$base/s3proxy-linux-amd64.tar.gz" "$base/SHA256SUMS" >"$tmp/expected.urls"
  check cmp "$tmp/expected.urls" "$tmp/curl.calls"
  if [[ $status == 0 ]]; then
    check cmp "$tmp/release/SHA256SUMS" "$ASDF_DOWNLOAD_PATH/SHA256SUMS"
    check cmp "$tmp/release/s3proxy-linux-amd64.tar.gz" "$ASDF_DOWNLOAD_PATH/s3proxy-linux-amd64.tar.gz"
    status=0
    "$plugin/bin/install" >"$tmp/output" 2>"$tmp/error" || status=$?
    check test "$status" = 0
    check test -x "$ASDF_INSTALL_PATH/bin/s3proxy"
    check cmp "$tmp/payload/s3proxy" "$ASDF_INSTALL_PATH/bin/s3proxy"
    if [[ -x "$ASDF_INSTALL_PATH/bin/s3proxy" ]]; then
      check test "$("$ASDF_INSTALL_PATH/bin/s3proxy")" = 'fixture binary works'
    fi
  fi
done
for version in "${invalid[@]}"; do
  : >"$tmp/curl.calls"
  export ASDF_INSTALL_VERSION="$version" ASDF_DOWNLOAD_PATH="$tmp/invalid download"
  rm -rf "$ASDF_DOWNLOAD_PATH"
  status=0
  "$plugin/bin/download" >"$tmp/output" 2>"$tmp/error" || status=$?
  check test "$status" -ne 0
  check test ! -e "$ASDF_DOWNLOAD_PATH"
  check test ! -s "$tmp/curl.calls"
  check grep -q 'Invalid version' "$tmp/error"
done
printf '%064d  s3proxy-linux-amd64.tar.gz\n' 0 >"$tmp/release/SHA256SUMS"
status=0
ASDF_INSTALL_VERSION=3.0.0-rc.1+build.4 ASDF_DOWNLOAD_PATH="$tmp/bad checksum" "$plugin/bin/download" >"$tmp/output" 2>"$tmp/error" || status=$?
check test "$status" -ne 0
check grep -q 'Checksum mismatch' "$tmp/error"
check test ! -e "$tmp/bad checksum/s3proxy-linux-amd64.tar.gz"
check test ! -e "$tmp/bad checksum/s3proxy-linux-amd64.tar.gz.tmp"
check test ! -e "$tmp/bad checksum/SHA256SUMS.tmp"

export PATH="$original_path"
mkdir "$tmp/release repo"
cd "$tmp/release repo"
git init -q
fixture_git() {
  git -c user.name=Fixture -c user.email=fixture@example.invalid -c core.hooksPath=/dev/null -c commit.gpgsign=false -c tag.gpgsign=false "$@"
}
printf 'fixture\n' >VERSION
fixture_git add VERSION
fixture_git commit -qm initial
initial=$(git rev-parse HEAD)
printf 'second\n' >VERSION
fixture_git add VERSION
fixture_git commit -qm second
export GITHUB_SHA
GITHUB_SHA=$(git rev-parse HEAD)
for version in "${valid[@]}"; do
  fixture_git tag "v$version"
  printf '%s\n' "$version" >VERSION
  status=0
  bash "$plugin/scripts/validate-release-tag.sh" "v$version" >"$tmp/output" 2>"$tmp/error" || status=$?
  check test "$status" = 0
done
fixture_git tag -a v4.0.0 -m annotated
printf '4.0.0\n' >VERSION
status=0
bash "$plugin/scripts/validate-release-tag.sh" v4.0.0 >"$tmp/output" 2>"$tmp/error" || status=$?
check test "$status" = 0
for version in "${invalid[@]}"; do
  status=0
  bash "$plugin/scripts/validate-release-tag.sh" "v$version" >"$tmp/output" 2>"$tmp/error" || status=$?
  check test "$status" -ne 0
  check grep -q 'release tag must be' "$tmp/error"
done
status=0
bash "$plugin/scripts/validate-release-tag.sh" 4.0.0 >"$tmp/output" 2>"$tmp/error" || status=$?
check test "$status" -ne 0
check grep -q 'release tag must be' "$tmp/error"
status=0
bash "$plugin/scripts/validate-release-tag.sh" v3.0.0 >"$tmp/output" 2>"$tmp/error" || status=$?
check test "$status" -ne 0
check grep -q 'VERSION does not match' "$tmp/error"
fixture_git tag v5.0.0 "$initial"
printf '5.0.0\n' >VERSION
status=0
bash "$plugin/scripts/validate-release-tag.sh" v5.0.0 >"$tmp/output" 2>"$tmp/error" || status=$?
check test "$status" -ne 0
check grep -q 'tag, event source commit, and checkout must match' "$tmp/error"
status=0
GITHUB_SHA="$initial" bash "$plugin/scripts/validate-release-tag.sh" v5.0.0 >"$tmp/output" 2>"$tmp/error" || status=$?
check test "$status" -ne 0
check grep -q 'tag, event source commit, and checkout must match' "$tmp/error"
status=0
GITHUB_SHA="$initial" bash "$plugin/scripts/validate-release-tag.sh" v4.0.0 >"$tmp/output" 2>"$tmp/error" || status=$?
check test "$status" -ne 0
check grep -q 'tag, event source commit, and checkout must match' "$tmp/error"

printf 'version regressions: %s checks, %s failures; %s valid / %s invalid versions\n' "$checks" "$failures" "${#valid[@]}" "${#invalid[@]}"
test "$failures" = 0
