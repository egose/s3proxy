#!/usr/bin/env bash
set -euo pipefail

root=$(cd "$(dirname "$0")/.." && pwd)
tmp=$(mktemp -d)
trap 'rm -rf "$tmp"' EXIT

make_archive() {
  local name=$1 archive=$2
  mkdir -p "$tmp/archive"
  printf '#!/bin/sh\nexit 0\n' >"$tmp/archive/$name"
  tar -C "$tmp/archive" -czf "$archive" "$name"
  rm -rf "$tmp/archive"
}

make_archive s3proxy "$tmp/good.tar.gz"
mkdir "$tmp/download" "$tmp/install"
cp "$tmp/good.tar.gz" "$tmp/download/s3proxy-linux-amd64.tar.gz"
env ASDF_INSTALL_TYPE=version ASDF_INSTALL_VERSION=1.2.3 ASDF_DOWNLOAD_PATH="$tmp/download" ASDF_INSTALL_PATH="$tmp/install" "$root/bin/install"
test -x "$tmp/install/bin/s3proxy"

mkdir "$tmp/release" "$tmp/download-checksum" "$tmp/fakecurl"
cp "$tmp/good.tar.gz" "$tmp/release/s3proxy-linux-amd64.tar.gz"
(cd "$tmp/release" && sha256sum s3proxy-linux-amd64.tar.gz >SHA256SUMS)
cat >"$tmp/fakecurl/curl" <<'EOF'
#!/bin/sh
while test "$#" -gt 0; do
  case "$1" in
    -o) output=$2; shift 2 ;;
    http*) url=$1; shift ;;
    *) shift ;;
  esac
done
cp "$FIXTURES/${url##*/}" "$output"
EOF
chmod +x "$tmp/fakecurl/curl"
env PATH="$tmp/fakecurl:$PATH" FIXTURES="$tmp/release" ASDF_INSTALL_VERSION=1.2.3 ASDF_DOWNLOAD_PATH="$tmp/download-checksum" "$root/bin/download"
test -f "$tmp/download-checksum/s3proxy-linux-amd64.tar.gz"
printf '%064d  s3proxy-linux-amd64.tar.gz\n' 0 >"$tmp/release/SHA256SUMS"
if env PATH="$tmp/fakecurl:$PATH" FIXTURES="$tmp/release" ASDF_INSTALL_VERSION=1.2.3 ASDF_DOWNLOAD_PATH="$tmp/download-checksum" "$root/bin/download"; then
  echo "download accepted a checksum mismatch" >&2
  exit 1
fi

rm -rf "$tmp/download" "$tmp/install"
mkdir "$tmp/download" "$tmp/archive"
ln -s /bin/sh "$tmp/archive/s3proxy"
tar -C "$tmp/archive" -czf "$tmp/download/unsafe.tar.gz" s3proxy
if env ASDF_INSTALL_TYPE=version ASDF_INSTALL_VERSION=1.2.3 ASDF_DOWNLOAD_PATH="$tmp/download" ASDF_INSTALL_PATH="$tmp/install" "$root/bin/install"; then
  echo "installer accepted a symlink" >&2
  exit 1
fi

mkdir "$tmp/fakebin"
cat >"$tmp/fakebin/git" <<'EOF'
#!/bin/sh
printf '%s\t%s\n' a refs/tags/v1.2.3 b refs/tags/not-a-version c refs/tags/v2.0.0
EOF
chmod +x "$tmp/fakebin/git"
versions=$(PATH="$tmp/fakebin:$PATH" ASDF_S3PROXY_GITHUB_REPOSITORY=egose/s3proxy "$root/bin/list-all")
test "$versions" = "1.2.3 2.0.0"

echo "asdf plugin tests passed"
