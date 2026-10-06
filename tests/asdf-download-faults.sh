#!/usr/bin/env bash
set -euo pipefail

root=$(cd "$(dirname "$0")/.." && pwd)
tmp=$(mktemp -d)
trap 'rm -rf "$tmp"' EXIT
mkdir "$tmp/tools" "$tmp/release"
printf 'verified fixture bytes\n' >"$tmp/release/s3proxy-linux-amd64.tar.gz"
digest=$(shasum -a 256 "$tmp/release/s3proxy-linux-amd64.tar.gz")
digest=${digest%% *}
printf '%s  s3proxy-linux-amd64.tar.gz\n' "$digest" >"$tmp/release/SHA256SUMS"
real_path=$PATH
for tool in bash dirname mkdir awk mv rm; do
  ln -s "$(command -v "$tool")" "$tmp/tools/$tool"
done
cat >"$tmp/tools/uname" <<'EOF'
#!/bin/bash
case "$1" in -s) printf 'Linux\n' ;; -m) printf 'x86_64\n' ;; *) exit 90 ;; esac
EOF
cat >"$tmp/tools/curl" <<'EOF'
#!/bin/bash
set -euo pipefail
url= output=
while (( $# )); do
  case "$1" in
    -o) output=$2; shift 2 ;;
    https:*) url=$1; shift ;;
    *) shift ;;
  esac
done
[[ $output == "$CASE/"* ]] || exit 90
base=https://github.com/fixture/repo/releases/download/v3.0.0-rc.1+build.4
case "$url" in
  "$base/s3proxy-linux-amd64.tar.gz"|"$base/SHA256SUMS") ;;
  *) exit 91 ;;
esac
printf '%s\n' "$url" >>"$CASE/curl.calls"
PATH=$REAL_PATH cp "$FIXTURE/release/${url##*/}" "$output"
if [[ $FAULT == "curl-${url##*/}" ]]; then
  printf 'injected curl failure\n' >&2
  exit 74
fi
EOF
cat >"$tmp/tools/hash" <<'EOF'
#!/bin/bash
set -euo pipefail
[[ ${!#} == "$CASE/"* ]] || exit 92
printf '%s\n' "${0##*/}" >>"$CASE/hash.calls"
PATH=$REAL_PATH "${0##*/}" "$@"
if [[ $FAULT == hash ]]; then
  printf 'injected hash failure\n' >&2
  exit 75
fi
EOF
chmod +x "$tmp/tools/uname" "$tmp/tools/curl" "$tmp/tools/hash"
ln -s hash "$tmp/tools/shasum"

cases=0
for hash in sha256sum shasum; do
  if [[ $hash == sha256sum ]]; then
    ln -s hash "$tmp/tools/sha256sum"
  else
    rm "$tmp/tools/sha256sum"
  fi
  for fault in none star curl-s3proxy-linux-amd64.tar.gz curl-SHA256SUMS hash missing duplicate malformed mismatch; do
    fixture="$tmp/$hash $fault"
    mkdir "$fixture"
    printf 'caller archive\n' >"$fixture/s3proxy-linux-amd64.tar.gz"
    printf 'caller manifest\n' >"$fixture/SHA256SUMS"
    cp "$fixture/s3proxy-linux-amd64.tar.gz" "$fixture/archive.before"
    cp "$fixture/SHA256SUMS" "$fixture/manifest.before"
    printf '%s  s3proxy-linux-amd64.tar.gz\n' "$digest" >"$tmp/release/SHA256SUMS"
    case "$fault" in
      star) printf '%s *s3proxy-linux-amd64.tar.gz\n' "$digest" >"$tmp/release/SHA256SUMS" ;;
      missing) printf '%s  other.tar.gz\n' "$digest" >"$tmp/release/SHA256SUMS" ;;
      duplicate) printf '%s  s3proxy-linux-amd64.tar.gz\n' "$digest" >>"$tmp/release/SHA256SUMS" ;;
      malformed) printf 'not-a-hash  s3proxy-linux-amd64.tar.gz\n' >"$tmp/release/SHA256SUMS" ;;
      mismatch) printf '%064d  s3proxy-linux-amd64.tar.gz\n' 0 >"$tmp/release/SHA256SUMS" ;;
    esac
    status=0
    env PATH="$tmp/tools" REAL_PATH="$real_path" FIXTURE="$tmp" CASE="$fixture" FAULT="$fault" \
      ASDF_S3PROXY_GITHUB_REPOSITORY=fixture/repo ASDF_INSTALL_VERSION=3.0.0-rc.1+build.4 \
      ASDF_DOWNLOAD_PATH="$fixture" "$root/bin/download" >"$fixture/output" 2>"$fixture/error" || status=$?
    case "$fault" in
      none|star)
        test "$status" = 0
        grep -q 'Download and checksum verification successful' "$fixture/output"
        cmp "$tmp/release/s3proxy-linux-amd64.tar.gz" "$fixture/s3proxy-linux-amd64.tar.gz"
        cmp "$tmp/release/SHA256SUMS" "$fixture/SHA256SUMS"
        test "$(cat "$fixture/hash.calls")" = "$hash" ;;
      *)
        test "$status" -ne 0
        if grep -q 'verification successful' "$fixture/output"; then exit 1; fi
        cmp "$fixture/archive.before" "$fixture/s3proxy-linux-amd64.tar.gz"
        cmp "$fixture/manifest.before" "$fixture/SHA256SUMS"
        case "$fault" in
          curl-*)
            test "$status" = 74
            grep -q 'injected curl failure' "$fixture/error"
            test ! -e "$fixture/hash.calls" ;;
          hash)
            test "$status" = 75
            grep -q 'injected hash failure' "$fixture/error"
            test "$(cat "$fixture/hash.calls")" = "$hash" ;;
          missing|duplicate|malformed)
            grep -q 'Missing checksum' "$fixture/error"
            test ! -e "$fixture/hash.calls" ;;
          mismatch) grep -q 'Checksum mismatch' "$fixture/error" ;;
        esac ;;
    esac
    test ! -e "$fixture/s3proxy-linux-amd64.tar.gz.tmp"
    test ! -e "$fixture/SHA256SUMS.tmp"
    calls=2
    if [[ $fault == curl-s3proxy-linux-amd64.tar.gz ]]; then calls=1; fi
    test "$(wc -l <"$fixture/curl.calls" | tr -d ' ')" = "$calls"
    cases=$((cases + 1))
    printf 'PASS download %s / %s (exit %s)\n' "$hash" "$fault" "$status"
  done
done
printf 'asdf download faults: %s cases passed\n' "$cases"
