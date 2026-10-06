#!/usr/bin/env bash
set -euo pipefail

root=$(cd "$(dirname "$0")/.." && pwd)
tmp=$(mktemp -d)
trap 'rm -rf "$tmp"' EXIT
mkdir "$tmp/tools" "$tmp/source"
printf '#!/bin/sh\nprintf "complete binary\\n"\n' >"$tmp/source/s3proxy"
tar -C "$tmp/source" -czf "$tmp/good.tar.gz" s3proxy
cp "$tmp/good.tar.gz" "$tmp/good before.tar.gz"

cat >"$tmp/tools/tool" <<'EOF'
#!/usr/bin/env bash
set -euo pipefail
tool=${0##*/}
export PATH="$FAULT_REAL_PATH"
last=${!#}
for argument in "$@"; do
  case "$argument" in
    -*|0755|"$FAULT_ROOT"/*) ;;
    *) echo "fault tool outside fixture: $tool $*" >&2; exit 90 ;;
  esac
done
printf '%s %s\n' "$tool" "$*" >>"$FAULT_ROOT/trace"
hit() {
  printf '%s\n' "$FAULT" >"$FAULT_ROOT/injected"
  echo "injected $FAULT" >&2
}
case "$tool:$FAULT" in
  cut:cut)
    hit
    cut "$@"
    exit 71 ;;
  mv:cross-device-copy)
    hit
    case "$1" in
      "$FAULT_ROOT/parent space/install space/"*) ;;
      *)
        mkdir "$last"
        printf 'partial copy\n' >"$last/s3proxy"
        exit 76 ;;
    esac ;;
  tar:list|tar:list-empty)
    if [ "$1" = -tzf ]; then
      hit
      if [ "$FAULT" = list ]; then printf 's3proxy\n'; fi
      exit 71
    fi ;;
  tar:verbose-list)
    if [ "$1" = -tvzf ]; then
      hit
      printf '%s\n' '-rwxr-xr-x user/group 1 date s3proxy'
      exit 71
    fi ;;
  tar:extract-partial|tar:missing|tar:directory|tar:symlink|tar:fifo)
    if [ "$1" = -xzf ]; then
      case "$last" in
        "$FAULT_ROOT"/parent\ space/.s3proxy.install.*/bin|"$FAULT_ROOT"/parent\ space/install\ space/.s3proxy.install.*/bin) ;;
        *) exit 91 ;;
      esac
      hit
      case "$FAULT" in
        extract-partial) printf 'partial bytes\n' >"$last/s3proxy"; exit 72 ;;
        missing) ;;
        directory) mkdir "$last/s3proxy" ;;
        symlink) ln -s "$FAULT_ROOT/target/sentinel" "$last/s3proxy" ;;
        fifo) mkfifo "$last/s3proxy" ;;
      esac
      exit 0
    fi ;;
  chmod:chmod|dirname:dirname|mktemp:mktemp|mv:mv|rmdir:rmdir)
    hit
    exit 73 ;;
  mkdir:mkdir-parent)
    if [ "$last" = "$FAULT_ROOT/parent space" ]; then hit; exit 74; fi ;;
  mkdir:mkdir-stage)
    case "$last" in */.s3proxy.install.*/bin) hit; exit 74 ;; esac ;;
  mkdir:mkdir-destination)
    if [ "$last" = "$FAULT_ROOT/parent space/install space" ]; then hit; exit 74; fi ;;
esac
exec "$tool" "$@"
EOF
chmod +x "$tmp/tools/tool"
for tool in tar cut chmod dirname mktemp mkdir mv rmdir; do
  ln -s tool "$tmp/tools/$tool"
done

failures=0
cases=0
fail() {
  echo "FAIL $name: $*" >&2
  failures=$((failures + 1))
}

run_case() {
  local name=$1 fault=$2 expected=$3 collision=${4:-none}
  local fixture="$tmp/$name" status=0 before=$failures link='' archive=s3proxy
  cases=$((cases + 1))
  mkdir -p "$fixture/download space" "$fixture/parent space/install space" "$fixture/target"
  local destination="$fixture/parent space/install space"
  printf 'caller sentinel\n' >"$destination/sentinel"
  if [ "$name" = install-path-link ]; then
    mv "$destination" "$fixture/linked install"
    ln -s "$fixture/linked install" "$destination"
  fi
  printf 'target sentinel\n' >"$fixture/target/sentinel"
  chmod 0644 "$fixture/target/sentinel"
  cp "$tmp/good.tar.gz" "$fixture/download space/release.tar.gz"
  printf 'caller checksum contents\n' >"$fixture/download space/SHA256SUMS"
  case "$collision" in
    directory) mkdir "$destination/bin"; cp "$destination/sentinel" "$destination/bin/sentinel" ;;
    file) cp "$destination/sentinel" "$destination/bin" ;;
    live-directory) link="$fixture/target" ;;
    live-file) link="$fixture/target/sentinel" ;;
    dangling) link="$fixture/target/missing" ;;
  esac
  if [ -n "$link" ]; then ln -s "$link" "$destination/bin"; fi
  case "$name" in
    no-archive) rm "$fixture/download space/release.tar.gz" ;;
    two-archives) cp "$tmp/good.tar.gz" "$fixture/download space/second.tar.gz" ;;
    archive-link)
      rm "$fixture/download space/release.tar.gz"
      ln -s "$tmp/good.tar.gz" "$fixture/download space/release.tar.gz" ;;
    exe)
      archive=s3proxy.exe
      mkdir "$fixture/source"
      cp "$tmp/source/s3proxy" "$fixture/source/$archive"
      tar -C "$fixture/source" -czf "$fixture/download space/release.tar.gz" "$archive" ;;
    archive-directory|archive-symlink|archive-multiple|archive-nested|archive-duplicate)
      mkdir "$fixture/source"
      case "$name" in
        archive-directory) mkdir "$fixture/source/s3proxy" ;;
        archive-symlink) ln -s "$fixture/target/sentinel" "$fixture/source/s3proxy" ;;
        archive-multiple) cp "$tmp/source/s3proxy" "$fixture/source/s3proxy"; touch "$fixture/source/extra" ;;
        archive-nested) mkdir "$fixture/source/nested"; cp "$tmp/source/s3proxy" "$fixture/source/nested/s3proxy" ;;
        archive-duplicate) cp "$tmp/source/s3proxy" "$fixture/source/s3proxy" ;;
      esac
      local members=(s3proxy)
      case "$name" in
        archive-multiple) members+=(extra) ;;
        archive-nested) members=(nested/s3proxy) ;;
        archive-duplicate) members+=(s3proxy) ;;
      esac
      tar -C "$fixture/source" -czf "$fixture/download space/release.tar.gz" "${members[@]}" ;;
  esac
  cp -R "$fixture/download space" "$fixture/download before"
  cp -R "$fixture/target" "$fixture/target before"
  env PATH="$tmp/tools:$PATH" FAULT_REAL_PATH="$PATH" FAULT_ROOT="$fixture" FAULT="$fault" \
    ASDF_INSTALL_TYPE=version ASDF_INSTALL_VERSION=1.2.3 \
    ASDF_DOWNLOAD_PATH="$fixture/download space" ASDF_INSTALL_PATH="$destination" \
    "$root/bin/install" >"$fixture/output" 2>&1 || status=$?
  if [ "$fault" != none ] && [ ! -f "$fixture/injected" ]; then fail "fault was not reached"; fi
  if [ "$expected" = success ]; then
    if [ "$status" -ne 0 ]; then fail "exit $status on valid install"; fi
    if ! grep -q 'installation was successful!' "$fixture/output"; then fail "missing success output"; fi
  else
    if [ "$status" -eq 0 ]; then fail "exit 0 after $fault / $collision"; fi
    if grep -q 'installation was successful!' "$fixture/output"; then fail "false success output"; fi
    if ! grep -q 'Error: Installation failed.' "$fixture/output"; then fail "missing failure diagnostic"; fi
  fi
  if [ "$expected" = success ] || [ "$expected" = published ]; then
    if ! cmp -s "$tmp/source/s3proxy" "$destination/bin/$archive"; then fail "installed bytes differ"; fi
    if [ ! -f "$destination/bin/$archive" ] || [ -L "$destination/bin/$archive" ]; then fail "installed binary not regular"; fi
    if [ "$(find "$destination/bin/$archive" -prune -type f -perm 0755)" != "$destination/bin/$archive" ]; then fail "installed mode is not 0755"; fi
  elif [ "$collision" = none ]; then
    if [ -e "$destination/bin" ] || [ -L "$destination/bin" ]; then fail "published bin on failure"; fi
  fi
  case "$collision" in
    directory)
      if ! cmp -s "$destination/sentinel" "$destination/bin/sentinel"; then fail "existing bin sentinel changed"; fi
      local children=("$destination/bin/"*)
      if [ "${#children[@]}" -ne 1 ]; then fail "existing bin gained content"; fi ;;
    file) if ! cmp -s "$destination/sentinel" "$destination/bin"; then fail "existing bin file changed"; fi ;;
    live-directory|live-file|dangling)
      if [ ! -L "$destination/bin" ] || [ "$(readlink "$destination/bin")" != "$link" ]; then fail "existing bin symlink changed"; fi ;;
  esac
  if [ "$(cat "$destination/sentinel")" != 'caller sentinel' ]; then fail "caller content changed"; fi
  if [ "$name" = install-path-link ]; then
    if [ ! -L "$destination" ] || [ "$(readlink "$destination")" != "$fixture/linked install" ]; then fail "install path symlink changed"; fi
  fi
  if ! diff -r "$fixture/target before" "$fixture/target" >/dev/null; then fail "symlink target changed"; fi
  if [ -x "$fixture/target/sentinel" ]; then fail "symlink target permissions changed"; fi
  if ! diff -r "$fixture/download before" "$fixture/download space" >/dev/null; then fail "downloads/checksums changed"; fi
  if [ "$name" = archive-link ]; then
    if [ ! -L "$fixture/download space/release.tar.gz" ] || [ "$(readlink "$fixture/download space/release.tar.gz")" != "$tmp/good.tar.gz" ]; then fail "download symlink changed"; fi
  fi
  if ! cmp -s "$tmp/good before.tar.gz" "$tmp/good.tar.gz"; then fail "caller archive target changed"; fi
  local stages=("$fixture/parent space/".s3proxy.install.* "$destination/".s3proxy.install.*) stage
  for stage in "${stages[@]}"; do
    if [ -e "$stage" ]; then fail "stage was not cleaned"; fi
  done
  if [ "$before" -ne "$failures" ]; then
    cat "$fixture/output" >&2
    if [ -f "$fixture/trace" ]; then cat "$fixture/trace" >&2; fi
  else
    echo "PASS $name (installer exit $status)"
  fi
}

for fault in list list-empty verbose-list cut extract-partial missing directory symlink fifo chmod dirname mktemp mkdir-parent mkdir-stage mkdir-destination mv; do
  run_case "$fault" "$fault" failure
done
run_case rmdir rmdir published
run_case cross-device-copy cross-device-copy success
for collision in directory file live-directory live-file dangling; do
  run_case "collision-$collision" none failure "$collision"
done
for archive in no-archive two-archives archive-directory archive-symlink archive-multiple archive-nested archive-duplicate; do
  run_case "$archive" none failure
done
run_case success none success
run_case exe none success
run_case archive-link none success
run_case install-path-link none success

if [ "$failures" -ne 0 ]; then
  echo "asdf installer faults: $failures failed assertions across $cases cases" >&2
  exit 1
fi
echo "asdf installer faults: $cases cases passed"
